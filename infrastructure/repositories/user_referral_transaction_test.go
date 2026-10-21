package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type referralNotificationSpy struct {
	db              *gorm.DB
	referrerID      uuid.UUID
	expectedBalance decimal.Decimal
}

// Observe durable notifications and the committed reward from a separate query.
func (s *referralNotificationSpy) snapshot() (int, bool) {
	var count int64
	var user models.User
	err := s.db.Model(&notifications.Notification{}).Where("user_id = ? AND type = ?", s.referrerID, notifications.NotificationTypeReferral).Count(&count).Error
	visible := s.db.Select("id", "balance").First(&user, "id = ?", s.referrerID).Error == nil && user.Balance.Equal(s.expectedBalance)
	return int(count), err == nil && visible
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
	if !strings.Contains(lockSQL, "transaction_locks") || !strings.Contains(lockSQL, "for update") {
		t.Fatalf("referral advisory lock SQL is incomplete: %s", lock.Statement.SQL.String())
	}
	if len(lock.Statement.Vars) < 1 || lock.Statement.Vars[0] != lockKey {
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

func TestReferralApprovalIsAtomicIdempotentAndQueuesNotificationIntegration(t *testing.T) {
	db, repo, referrer, referred, aggregate, notifier := prepareReferralIntegration(t, datatypes.JSON([]byte("{}")))
	reward := decimal.RequireFromString("2.25")
	wantBalance := decimal.RequireFromString("6.25")
	notifier.expectedBalance = wantBalance

	first, err := creditReferralForTest(repo, context.Background(), referrer.ID, referred.ID, reward)
	if err != nil {
		t.Fatalf("first AddReferral() error = %v", err)
	}
	second, err := creditReferralForTest(repo, context.Background(), referrer.ID, referred.ID, reward)
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

func TestReferralApprovalRollsBackBalanceWhenEngagementWriteFailsIntegration(t *testing.T) {
	// Keep the fixture valid for PostgreSQL jsonb while making it invalid for
	// the domain aggregate's map representation. This reaches the actual
	// transactional engagement write instead of failing during fixture setup.
	db, repo, referrer, referred, aggregate, notifier := prepareReferralIntegration(t, datatypes.JSON([]byte("[]")))

	_, err := creditReferralForTest(repo, context.Background(), referrer.ID, referred.ID, decimal.NewFromInt(2))
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

func TestReferralApprovalRollsBackEngagementWhenBalanceWriteFailsIntegration(t *testing.T) {
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

	_, err := creditReferralForTest(repo, context.Background(), referrer.ID, referred.ID, decimal.NewFromInt(2))
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

func TestReferralApprovalConcurrentRetriesRewardOnceIntegration(t *testing.T) {
	db := referralRaceIntegrationDB(t)
	referrer, referred, aggregate := createReferralFixture(t, db, datatypes.JSON([]byte("{}")))
	t.Cleanup(func() { cleanupReferralFixture(db, referrer.ID, referred.ID) })

	reward := decimal.RequireFromString("3.50")
	wantBalance := decimal.RequireFromString("7.50")
	notifier := &referralNotificationSpy{db: db, referrerID: referrer.ID, expectedBalance: wantBalance}
	repo := &UserRepository{db: db, engagementRepo: NewEngagementRepository(db)}

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
			balance, err := creditReferralForTest(repo, context.Background(), referrer.ID, referred.ID, reward)
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

func TestReferralApprovalDoesNotDeadlockWithReciprocalInteractionIntegration(t *testing.T) {
	db := referralRaceIntegrationDB(t)
	referrer, referred, aggregate := createReferralFixture(t, db, datatypes.JSON([]byte("{}")))
	t.Cleanup(func() { cleanupReferralFixture(db, referrer.ID, referred.ID) })

	reward := decimal.NewFromInt(1)
	wantBalance := decimal.NewFromInt(5)
	notifier := &referralNotificationSpy{db: db, referrerID: referrer.ID, expectedBalance: wantBalance}
	engagementRepo := NewEngagementRepository(db)
	repo := &UserRepository{db: db, engagementRepo: engagementRepo}
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
		_, err := creditReferralForTest(repo, ctx, referrer.ID, referred.ID, reward)
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
	db := privatePhotoIntegrationDB(t)
	referrer, referred, aggregate := createReferralFixture(t, db, counts)
	notifier := &referralNotificationSpy{
		db:              db,
		referrerID:      referrer.ID,
		expectedBalance: referrer.Balance,
	}
	repo := &UserRepository{db: db, engagementRepo: NewEngagementRepository(db)}
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
	return privatePhotoIntegrationDB(t)
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

// Monetary tests explicitly pass both stages: record the referral, then review
// it through an independent trusted administrator. Production registration only
// performs the first stage.
func creditReferralForTest(repo *UserRepository, ctx context.Context, referrer, referred uuid.UUID, amount decimal.Decimal) (*decimal.Decimal, error) {
	if _, err := repo.AddReferral(ctx, referrer, referred, amount); err != nil {
		return nil, err
	}
	admin := models.User{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), PublicID: -999, UserName: "financial_test_admin", Domain: models.CoolVibes, UserRole: constants.UserRoleAdmin}
	if err := repo.db.WithContext(ctx).Omit(clause.Associations).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).Create(&admin).Error; err != nil {
		return nil, err
	}
	var reward models.ReferralReward
	if err := repo.db.WithContext(ctx).Where("referred_id = ?", referred).Take(&reward).Error; err != nil {
		return nil, err
	}
	review, err := domainuser.NewReferralReview("approve", "Verified genuine referral in test", "test-identity:"+referred.String())
	if err != nil {
		return nil, err
	}
	if _, err := NewReferralRewardRepository(repo.db).ReviewReferralReward(ctx, admin.ID, reward.ID, review); err != nil {
		return nil, err
	}
	var user models.User
	if err := repo.db.WithContext(ctx).Select("balance").Take(&user, "id = ?", referrer).Error; err != nil {
		return nil, err
	}
	return &user.Balance, nil
}
