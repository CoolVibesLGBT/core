package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"core/models/notifications"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type referralNotificationSpy struct {
	mu                     sync.Mutex
	db                     *gorm.DB
	referrerID             uuid.UUID
	expectedBalance        decimal.Decimal
	calls                  int
	committedStateObserved bool
	err                    error
}

func (s *referralNotificationSpy) SendNotificationToUser(sender models.User, receiver models.User, _ string, _ string, _ string, _ notifications.NotificationPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++

	var referrer models.User
	var detailCount int64
	balanceVisible := s.db.Select("id", "balance").First(&referrer, "id = ?", s.referrerID).Error == nil && referrer.Balance.Equal(s.expectedBalance)
	detailVisible := s.db.Model(&models.EngagementDetail{}).
		Where("engager_id = ? AND engagee_id = ? AND kind = ?", sender.ID, receiver.ID, models.EngagementKindReferral).
		Count(&detailCount).Error == nil && detailCount == 1
	s.committedStateObserved = s.committedStateObserved || (balanceVisible && detailVisible)
	return s.err
}

func (s *referralNotificationSpy) snapshot() (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls, s.committedStateObserved
}

func TestAddReferralRejectsInvalidReferralBeforeDatabaseAccess(t *testing.T) {
	repo := &UserRepository{}
	userID := uuid.New()

	if _, err := repo.AddReferral(context.Background(), userID, userID, decimal.NewFromInt(1)); !errors.Is(err, domainuser.ErrSelfReferral) {
		t.Fatalf("AddReferral(self) error = %v, want ErrSelfReferral", err)
	}
	if _, err := repo.AddReferral(context.Background(), uuid.New(), uuid.New(), decimal.Zero); !errors.Is(err, domainuser.ErrInvalidReferralReward) {
		t.Fatalf("AddReferral(zero reward) error = %v, want ErrInvalidReferralReward", err)
	}
}

func TestAmountEngagementDetailKeepsStringAmountAndOptionalDedupe(t *testing.T) {
	aggregateID := uuid.New()
	engagerID := uuid.New()
	engageeID := uuid.New()
	amount := decimal.RequireFromString("2.2500")

	withoutDedupe, err := newAmountEngagementDetail(aggregateID, engagerID, engageeID, amount, models.EngagementKindTip)
	if err != nil {
		t.Fatalf("newAmountEngagementDetail() error = %v", err)
	}
	if withoutDedupe.DedupeKey != nil {
		t.Fatalf("default amount engagement dedupe key = %q, want nil", *withoutDedupe.DedupeKey)
	}
	var payload map[string]string
	if err := json.Unmarshal(withoutDedupe.Details, &payload); err != nil {
		t.Fatalf("decode amount detail: %v", err)
	}
	if payload["amount"] != amount.String() {
		t.Fatalf("amount detail = %q, want %q", payload["amount"], amount.String())
	}

	dedupeKey := "referral:" + uuid.NewString()
	withDedupe, err := newAmountEngagementDetail(aggregateID, engagerID, engageeID, amount, models.EngagementKindReferral, withAmountEngagementDedupeKey(dedupeKey))
	if err != nil {
		t.Fatalf("newAmountEngagementDetail(dedupe) error = %v", err)
	}
	if withDedupe.DedupeKey == nil || *withDedupe.DedupeKey != dedupeKey {
		t.Fatalf("amount engagement dedupe key = %v, want %q", withDedupe.DedupeKey, dedupeKey)
	}
}

func TestReferralBalanceLockUsesNoKeyUpdateInDeterministicOrder(t *testing.T) {
	referral, err := domainuser.NewReferral(uuid.New(), uuid.New(), decimal.NewFromInt(1))
	if err != nil {
		t.Fatalf("NewReferral() error = %v", err)
	}
	db := newDryRunTaxonomyDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	lockKey := "referral:" + referral.DedupeKey()
	lock := lockViewAggregate(db, lockKey)
	if lock.Error != nil {
		t.Fatalf("referral advisory lock query: %v", lock.Error)
	}
	lockSQL := strings.ToLower(lock.Statement.SQL.String())
	if !strings.Contains(lockSQL, "pg_advisory_xact_lock") || !strings.Contains(lockSQL, "hashtextextended") {
		t.Fatalf("referral advisory lock SQL is incomplete: %s", lock.Statement.SQL.String())
	}
	if len(lock.Statement.Vars) != 1 || lock.Statement.Vars[0] != lockKey {
		t.Fatalf("referral advisory lock vars = %#v, want %q", lock.Statement.Vars, lockKey)
	}

	var users []models.User
	query := referralUsersForBalanceUpdateQuery(db, referral).Find(&users)
	if query.Error != nil {
		t.Fatalf("referral balance lock query: %v", query.Error)
	}
	sql := strings.ToUpper(query.Statement.SQL.String())
	for _, fragment := range []string{"ORDER BY ID ASC", "FOR NO KEY UPDATE"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("referral balance lock SQL missing %q: %s", fragment, query.Statement.SQL.String())
		}
	}
}

func TestAddReferralIsAtomicIdempotentAndNotifiesAfterCommitIntegration(t *testing.T) {
	db, repo, referrer, referred, aggregate, notifier := prepareReferralIntegration(t, datatypes.JSON([]byte("{}")))
	reward := decimal.RequireFromString("2.25")
	wantBalance := decimal.RequireFromString("6.25")
	notifier.expectedBalance = wantBalance
	notifier.err = errors.New("push unavailable")

	first, err := repo.AddReferral(context.Background(), referrer.ID, referred.ID, reward)
	if err != nil {
		t.Fatalf("first AddReferral() error = %v", err)
	}
	second, err := repo.AddReferral(context.Background(), referrer.ID, referred.ID, reward)
	if err != nil {
		t.Fatalf("idempotent AddReferral() error = %v", err)
	}
	if first == nil || second == nil || !first.Equal(wantBalance) || !second.Equal(wantBalance) {
		t.Fatalf("AddReferral() balances = %v/%v, want %s", first, second, wantBalance)
	}

	assertStoredBalance(t, db, referrer.ID, "6.25")
	assertReferralState(t, db, aggregate.ID, referrer.ID, referred.ID, reward, 1)
	if calls, committed := notifier.snapshot(); calls != 1 || !committed {
		t.Fatalf("notification calls/committed state = %d/%v, want 1/true", calls, committed)
	}
}

func TestAddReferralRollsBackBalanceWhenEngagementWriteFailsIntegration(t *testing.T) {
	// Keep the fixture valid for PostgreSQL jsonb while making it invalid for
	// the domain aggregate's map representation. This reaches the actual
	// transactional engagement write instead of failing during fixture setup.
	db, repo, referrer, referred, aggregate, notifier := prepareReferralIntegration(t, datatypes.JSON([]byte("[]")))

	_, err := repo.AddReferral(context.Background(), referrer.ID, referred.ID, decimal.NewFromInt(2))
	var decodeErr *json.UnmarshalTypeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("AddReferral() error = %v, want in-transaction aggregate decode failure", err)
	}
	assertStoredBalance(t, db, referrer.ID, "4")

	var detailCount int64
	if err := db.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND kind = ?", aggregate.ID, models.EngagementKindReferral).
		Count(&detailCount).Error; err != nil {
		t.Fatalf("count referral details: %v", err)
	}
	if detailCount != 0 {
		t.Fatalf("failed referral persisted %d details, want 0", detailCount)
	}
	if calls, _ := notifier.snapshot(); calls != 0 {
		t.Fatalf("rollback sent %d notifications, want 0", calls)
	}
}

func TestAddReferralRollsBackEngagementWhenBalanceWriteFailsIntegration(t *testing.T) {
	db, repo, referrer, referred, aggregate, notifier := prepareReferralIntegration(t, datatypes.JSON([]byte("{}")))
	wantErr := errors.New("forced referral balance failure")
	callbackName := "test:fail_referral_balance:" + uuid.NewString()
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "users" {
			_ = tx.AddError(wantErr)
		}
	}); err != nil {
		t.Fatalf("register balance failure callback: %v", err)
	}
	t.Cleanup(func() { _ = db.Callback().Update().Remove(callbackName) })

	_, err := repo.AddReferral(context.Background(), referrer.ID, referred.ID, decimal.NewFromInt(2))
	if !errors.Is(err, wantErr) {
		t.Fatalf("AddReferral() error = %v, want forced balance failure", err)
	}
	assertStoredBalance(t, db, referrer.ID, "4")

	var detailCount int64
	if err := db.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND kind = ?", aggregate.ID, models.EngagementKindReferral).
		Count(&detailCount).Error; err != nil {
		t.Fatalf("count rolled-back referral details: %v", err)
	}
	if detailCount != 0 {
		t.Fatalf("failed balance write persisted %d referral details, want 0", detailCount)
	}
	if calls, _ := notifier.snapshot(); calls != 0 {
		t.Fatalf("balance rollback sent %d notifications, want 0", calls)
	}
}

func TestAddReferralConcurrentRetriesRewardOnceIntegration(t *testing.T) {
	db := referralRaceIntegrationDB(t)
	referrer, referred, aggregate := createReferralFixture(t, db, datatypes.JSON([]byte("{}")))
	t.Cleanup(func() { cleanupReferralFixture(db, referrer.ID, referred.ID) })

	reward := decimal.RequireFromString("3.50")
	wantBalance := decimal.RequireFromString("7.50")
	notifier := &referralNotificationSpy{db: db, referrerID: referrer.ID, expectedBalance: wantBalance}
	repo := &UserRepository{db: db, engagementRepo: NewEngagementRepository(db), notificationRepo: notifier}

	const attempts = 12
	start := make(chan struct{})
	errorsByAttempt := make(chan error, attempts)
	balances := make(chan decimal.Decimal, attempts)
	var workers sync.WaitGroup
	for i := 0; i < attempts; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			balance, err := repo.AddReferral(context.Background(), referrer.ID, referred.ID, reward)
			if err == nil && balance != nil {
				balances <- *balance
			}
			errorsByAttempt <- err
		}()
	}
	close(start)
	workers.Wait()
	close(errorsByAttempt)
	close(balances)

	for err := range errorsByAttempt {
		if err != nil {
			t.Fatalf("concurrent AddReferral() error = %v", err)
		}
	}
	for balance := range balances {
		if !balance.Equal(wantBalance) {
			t.Fatalf("concurrent AddReferral() balance = %s, want %s", balance, wantBalance)
		}
	}

	assertStoredBalance(t, db, referrer.ID, "7.50")
	assertReferralState(t, db, aggregate.ID, referrer.ID, referred.ID, reward, 1)
	if calls, committed := notifier.snapshot(); calls != 1 || !committed {
		t.Fatalf("concurrent notification calls/committed state = %d/%v, want 1/true", calls, committed)
	}
}

func TestAddReferralDoesNotDeadlockWithReciprocalInteractionIntegration(t *testing.T) {
	db := referralRaceIntegrationDB(t)
	referrer, referred, aggregate := createReferralFixture(t, db, datatypes.JSON([]byte("{}")))
	t.Cleanup(func() { cleanupReferralFixture(db, referrer.ID, referred.ID) })

	reward := decimal.NewFromInt(1)
	wantBalance := decimal.NewFromInt(5)
	notifier := &referralNotificationSpy{db: db, referrerID: referrer.ID, expectedBalance: wantBalance}
	engagementRepo := NewEngagementRepository(db)
	repo := &UserRepository{db: db, engagementRepo: engagementRepo, notificationRepo: notifier}
	intent, err := domainuser.NewSetInteractionState(domainuser.InteractionFollow, true)
	if err != nil {
		t.Fatalf("NewSetInteractionState() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, err := repo.AddReferral(ctx, referrer.ID, referred.ID, reward)
		if err != nil {
			results <- fmt.Errorf("AddReferral: %w", err)
			return
		}
		results <- nil
	}()
	go func() {
		<-start
		_, err := engagementRepo.ApplyReciprocalUserInteraction(ctx, referred.ID, referrer.ID, intent)
		if err != nil {
			results <- fmt.Errorf("ApplyReciprocalUserInteraction: %w", err)
			return
		}
		results <- nil
	}()
	close(start)

	for i := 0; i < 2; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(15 * time.Second):
			t.Fatal("referral and reciprocal interaction deadlocked")
		}
	}

	assertStoredBalance(t, db, referrer.ID, "5")
	assertReferralState(t, db, aggregate.ID, referrer.ID, referred.ID, reward, 1)
	assertReciprocalDetailCount(t, db, referred.ID, referrer.ID, models.EngagementKindFollowing, 1)
	assertReciprocalDetailCount(t, db, referrer.ID, referred.ID, models.EngagementKindFollower, 1)
	if calls, committed := notifier.snapshot(); calls != 1 || !committed {
		t.Fatalf("deadlock test notification calls/committed state = %d/%v, want 1/true", calls, committed)
	}
}

func prepareReferralIntegration(t *testing.T, counts datatypes.JSON) (*gorm.DB, *UserRepository, models.User, models.User, models.Engagement, *referralNotificationSpy) {
	t.Helper()
	db := engagementViewIntegrationDB(t)
	referrer, referred, aggregate := createReferralFixture(t, db, counts)
	notifier := &referralNotificationSpy{
		db:              db,
		referrerID:      referrer.ID,
		expectedBalance: referrer.Balance,
	}
	repo := &UserRepository{db: db, engagementRepo: NewEngagementRepository(db), notificationRepo: notifier}
	return db, repo, referrer, referred, aggregate, notifier
}

func createReferralFixture(t *testing.T, db *gorm.DB, counts datatypes.JSON) (models.User, models.User, models.Engagement) {
	t.Helper()
	if err := db.AutoMigrate(&models.User{}, &models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate referral schema: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	referrer := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "referral-referrer-" + uuid.NewString(), DisplayName: "Referrer",
		UserRole: constants.UserRoleUser, Balance: decimal.NewFromInt(4),
	}
	referred := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "referral-referred-" + uuid.NewString(), DisplayName: "Referred",
		UserRole: constants.UserRoleUser, Balance: decimal.Zero,
	}
	if err := db.Omit(clause.Associations).Create(&[]models.User{referrer, referred}).Error; err != nil {
		t.Fatalf("create referral users: %v", err)
	}

	aggregate := models.Engagement{
		ID:              uuid.New(),
		ContentableID:   referrer.ID,
		ContentableType: models.EngagementContentableTypeUser,
		Counts:          counts,
	}
	if err := db.Omit(clause.Associations).Create(&aggregate).Error; err != nil {
		t.Fatalf("create referral aggregate: %v", err)
	}
	return referrer, referred, aggregate
}

func referralRaceIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" && os.Getenv("ENV") == "test" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open referral race database: %v", err)
	}
	return db
}

func cleanupReferralFixture(db *gorm.DB, referrerID, referredID uuid.UUID) {
	var aggregateIDs []uuid.UUID
	db.Model(&models.Engagement{}).
		Where("contentable_id IN ? AND contentable_type = ?", []uuid.UUID{referrerID, referredID}, models.EngagementContentableTypeUser).
		Pluck("id", &aggregateIDs)
	if len(aggregateIDs) > 0 {
		db.Where("engagement_id IN ?", aggregateIDs).Delete(&models.EngagementDetail{})
		db.Where("id IN ?", aggregateIDs).Delete(&models.Engagement{})
	}
	db.Unscoped().Where("id IN ?", []uuid.UUID{referrerID, referredID}).Delete(&models.User{})
}

func assertReferralState(t *testing.T, db *gorm.DB, aggregateID, referrerID, referredID uuid.UUID, reward decimal.Decimal, expectedDetails int64) {
	t.Helper()
	referral, err := domainuser.NewReferral(referrerID, referredID, reward)
	if err != nil {
		t.Fatalf("NewReferral() assertion setup: %v", err)
	}

	var details []models.EngagementDetail
	if err := db.Where("engagement_id = ? AND kind = ?", aggregateID, models.EngagementKindReferral).Find(&details).Error; err != nil {
		t.Fatalf("load referral details: %v", err)
	}
	if int64(len(details)) != expectedDetails {
		t.Fatalf("referral detail count = %d, want %d", len(details), expectedDetails)
	}
	if expectedDetails > 0 {
		if details[0].DedupeKey == nil || *details[0].DedupeKey != referral.DedupeKey() {
			t.Fatalf("referral dedupe key = %v, want %q", details[0].DedupeKey, referral.DedupeKey())
		}
		var payload map[string]string
		if err := json.Unmarshal(details[0].Details, &payload); err != nil {
			t.Fatalf("decode referral amount details: %v", err)
		}
		if payload["amount"] != reward.String() {
			t.Fatalf("referral amount detail = %q, want %q", payload["amount"], reward.String())
		}
	}

	var aggregate models.Engagement
	if err := db.First(&aggregate, "id = ?", aggregateID).Error; err != nil {
		t.Fatalf("load referral aggregate: %v", err)
	}
	var counts map[string]any
	if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
		t.Fatalf("decode referral aggregate counts: %v", err)
	}
	if got := counts["referral_count"]; got != float64(expectedDetails) {
		t.Fatalf("referral_count = %#v, want %d", got, expectedDetails)
	}
	if expectedDetails > 0 && counts["referral_amount"] != reward.String() {
		t.Fatalf("referral_amount = %#v, want %q", counts["referral_amount"], reward.String())
	}
}
