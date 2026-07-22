package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"core/constants"
	domainwallet "core/domain/wallet"
	"core/models"
	"core/models/post"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestTipMovesBalancesAndRecordsEngagementAtomicallyIntegration(t *testing.T) {
	db, postRepo, payer, payee, target, aggregate := prepareTipIntegration(t, datatypes.JSON([]byte("{}")))

	key := mustTipIdempotencyKey(t, "tip-success-request")
	balance, err := postRepo.Tip(context.Background(), target.PublicID, &payer, decimal.RequireFromString("2.25"), key)
	if err != nil {
		t.Fatalf("Tip() error = %v", err)
	}
	if balance == nil || !balance.Equal(decimal.RequireFromString("7.75")) {
		t.Fatalf("Tip() payer balance = %v, want 7.75", balance)
	}

	assertStoredBalance(t, db, payer.ID, "7.75")
	assertStoredBalance(t, db, payee.ID, "6.25")

	var detail models.EngagementDetail
	if err := db.Where("engagement_id = ? AND kind = ?", aggregate.ID, models.EngagementKindTip).First(&detail).Error; err != nil {
		t.Fatalf("load tip detail: %v", err)
	}
	var details map[string]string
	if err := json.Unmarshal(detail.Details, &details); err != nil {
		t.Fatalf("decode tip detail: %v", err)
	}
	if details["amount"] != "2.25" {
		t.Fatalf("tip detail amount = %q, want 2.25", details["amount"])
	}
	if details[tipResultingBalanceDetailKey] != "7.75" {
		t.Fatalf("tip resulting balance = %q, want 7.75", details[tipResultingBalanceDetailKey])
	}
	if details["post_public_id"] != strconv.FormatInt(target.PublicID, 10) {
		t.Fatalf("tip post public id = %q, want %d", details["post_public_id"], target.PublicID)
	}
	if detail.DedupeKey == nil || *detail.DedupeKey != scopedTipDedupeKey(payer.ID, key) {
		t.Fatalf("tip detail dedupe key = %v", detail.DedupeKey)
	}
	assertTipAggregateTotals(t, db, aggregate.ID, 1, "2.25")
}

func TestTipReplaySurvivesPostBecomingHiddenIntegration(t *testing.T) {
	db, postRepo, payer, payee, target, _ := prepareTipIntegration(t, datatypes.JSON([]byte("{}")))
	key := mustTipIdempotencyKey(t, "tip-hidden-replay")

	firstBalance, err := postRepo.Tip(context.Background(), target.PublicID, &payer, decimal.NewFromInt(2), key)
	if err != nil {
		t.Fatalf("first Tip() error = %v", err)
	}
	if err := db.Model(&post.Post{}).Where("id = ?", target.ID).Update("published", false).Error; err != nil {
		t.Fatalf("hide tipped post: %v", err)
	}

	replayedBalance, err := postRepo.Tip(context.Background(), target.PublicID, &payer, decimal.NewFromInt(2), key)
	if err != nil {
		t.Fatalf("hidden-post replay error = %v", err)
	}
	if firstBalance == nil || replayedBalance == nil || !firstBalance.Equal(decimal.NewFromInt(8)) || !replayedBalance.Equal(*firstBalance) {
		t.Fatalf("hidden-post replay balances = %v/%v, want 8/8", firstBalance, replayedBalance)
	}
	assertStoredBalance(t, db, payer.ID, "8")
	assertStoredBalance(t, db, payee.ID, "6")
}

func TestTipRollsBackBalancesWhenEngagementWriteFailsIntegration(t *testing.T) {
	// PostgreSQL rejects malformed JSON before the transaction under test can
	// begin. A valid JSON array persists in jsonb but cannot be decoded as the
	// aggregate's counts map, forcing the intended in-transaction failure.
	db, postRepo, payer, payee, target, aggregate := prepareTipIntegration(t, datatypes.JSON([]byte("[]")))

	_, err := postRepo.Tip(context.Background(), target.PublicID, &payer, decimal.NewFromInt(2), mustTipIdempotencyKey(t, "tip-rollback-request"))
	var decodeErr *json.UnmarshalTypeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("Tip() error = %v, want in-transaction aggregate decode failure", err)
	}

	assertStoredBalance(t, db, payer.ID, "10")
	assertStoredBalance(t, db, payee.ID, "4")
	if !payer.Balance.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("failed Tip() mutated caller balance to %s", payer.Balance)
	}

	var count int64
	if err := db.Model(&models.EngagementDetail{}).Where("engagement_id = ?", aggregate.ID).Count(&count).Error; err != nil {
		t.Fatalf("count tip details: %v", err)
	}
	if count != 0 {
		t.Fatalf("failed Tip() persisted %d engagement details", count)
	}
}

func TestTipRejectsOwnPostWithDomainErrorIntegration(t *testing.T) {
	db, postRepo, _, author, target, aggregate := prepareTipIntegration(t, datatypes.JSON([]byte("{}")))

	_, err := postRepo.Tip(context.Background(), target.PublicID, &author, decimal.NewFromInt(1), mustTipIdempotencyKey(t, "tip-own-post-request"))
	if err == nil || err.Error() != constants.ErrCannotTipOwnPost.String() {
		t.Fatalf("Tip(own post) error = %v, want %s", err, constants.ErrCannotTipOwnPost.String())
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Tip(own post) leaked persistence error: %v", err)
	}
	assertStoredBalance(t, db, author.ID, "4")

	var count int64
	if err := db.Model(&models.EngagementDetail{}).Where("engagement_id = ?", aggregate.ID).Count(&count).Error; err != nil {
		t.Fatalf("count own-tip details: %v", err)
	}
	if count != 0 {
		t.Fatalf("own Tip() persisted %d engagement details", count)
	}
}

func TestTipReplaysSameRequestWithoutMovingMoneyTwiceIntegration(t *testing.T) {
	db, postRepo, payer, payee, target, aggregate := prepareTipIntegration(t, datatypes.JSON([]byte("{}")))
	key := mustTipIdempotencyKey(t, "tip-replay-request")

	firstBalance, err := postRepo.Tip(context.Background(), target.PublicID, &payer, decimal.NewFromInt(2), key)
	if err != nil {
		t.Fatalf("first Tip() error = %v", err)
	}
	secondBalance, err := postRepo.Tip(context.Background(), target.PublicID, &payer, decimal.NewFromInt(2), key)
	if err != nil {
		t.Fatalf("replayed Tip() error = %v", err)
	}
	if firstBalance == nil || secondBalance == nil || !firstBalance.Equal(decimal.NewFromInt(8)) || !secondBalance.Equal(*firstBalance) {
		t.Fatalf("Tip() replay balances = %v/%v, want 8/8", firstBalance, secondBalance)
	}

	assertStoredBalance(t, db, payer.ID, "8")
	assertStoredBalance(t, db, payee.ID, "6")
	var detailCount int64
	if err := db.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND kind = ?", aggregate.ID, models.EngagementKindTip).
		Count(&detailCount).Error; err != nil {
		t.Fatalf("count replayed tip details: %v", err)
	}
	if detailCount != 1 {
		t.Fatalf("replayed Tip() detail count = %d, want 1", detailCount)
	}
	assertTipAggregateTotals(t, db, aggregate.ID, 1, "2")
}

func TestTipRejectsIdempotencyKeyReuseForDifferentAmountIntegration(t *testing.T) {
	db, postRepo, payer, payee, target, aggregate := prepareTipIntegration(t, datatypes.JSON([]byte("{}")))
	key := mustTipIdempotencyKey(t, "tip-conflict-request")

	if _, err := postRepo.Tip(context.Background(), target.PublicID, &payer, decimal.NewFromInt(2), key); err != nil {
		t.Fatalf("first Tip() error = %v", err)
	}
	if _, err := postRepo.Tip(context.Background(), target.PublicID, &payer, decimal.NewFromInt(3), key); !errors.Is(err, domainwallet.ErrIdempotencyConflict) {
		t.Fatalf("conflicting Tip() error = %v, want ErrIdempotencyConflict", err)
	}

	assertStoredBalance(t, db, payer.ID, "8")
	assertStoredBalance(t, db, payee.ID, "6")
	var detailCount int64
	if err := db.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND kind = ?", aggregate.ID, models.EngagementKindTip).
		Count(&detailCount).Error; err != nil {
		t.Fatalf("count conflicting tip details: %v", err)
	}
	if detailCount != 1 {
		t.Fatalf("conflicting Tip() detail count = %d, want 1", detailCount)
	}
}

func prepareTipIntegration(t *testing.T, counts datatypes.JSON) (*gorm.DB, *PostRepository, models.User, models.User, post.Post, models.Engagement) {
	t.Helper()
	db := engagementViewIntegrationDB(t)
	return prepareTipIntegrationOnDB(t, db, counts)
}

func prepareTipIntegrationOnDB(t *testing.T, db *gorm.DB, counts datatypes.JSON) (*gorm.DB, *PostRepository, models.User, models.User, post.Post, models.Engagement) {
	t.Helper()
	if err := db.AutoMigrate(&models.User{}, &post.Post{}, &models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate tip schema: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	payer := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "tip-payer-" + uuid.NewString(), DisplayName: "Payer",
		UserRole: constants.UserRoleUser, Balance: decimal.NewFromInt(10),
	}
	payee := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "tip-payee-" + uuid.NewString(), DisplayName: "Payee",
		UserRole: constants.UserRoleUser, Balance: decimal.NewFromInt(4),
	}
	if err := db.Omit(clause.Associations).Create(&[]models.User{payer, payee}).Error; err != nil {
		t.Fatalf("create tip users: %v", err)
	}

	audience := "public"
	target := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 2, AuthorID: payee.ID,
		PostKind: post.PostKindPost, Domain: models.CoolVibes,
		Published: true, Audience: &audience,
	}
	if err := db.Omit(clause.Associations).Create(&target).Error; err != nil {
		t.Fatalf("create tip target post: %v", err)
	}

	aggregate := models.Engagement{
		ID: target.ID, ContentableID: target.ID,
		ContentableType: models.EngagementContentableTypePost,
		Counts:          counts,
	}
	if err := db.Omit(clause.Associations).Create(&aggregate).Error; err != nil {
		t.Fatalf("create tip engagement aggregate: %v", err)
	}

	engagementRepo := NewEngagementRepository(db)
	userRepo := NewUserRepository(db, nil, nil, engagementRepo, nil)
	postRepo := NewPostRepository(db, nil, nil, userRepo, nil)
	return db, postRepo, payer, payee, target, aggregate
}

func TestConcurrentTipRetriesWithSameKeyMoveMoneyOnceIntegration(t *testing.T) {
	db := locationRaceIntegrationDB(t)
	db, postRepo, payer, payee, target, aggregate := prepareTipIntegrationOnDB(t, db, datatypes.JSON([]byte("{}")))
	cleanupTipIntegrationFixture(t, db, []models.User{payer, payee}, []post.Post{target})
	key := mustTipIdempotencyKey(t, "tip-concurrent-retry")

	type outcome struct {
		balance *decimal.Decimal
		err     error
	}
	outcomes := make(chan outcome, 2)
	var writers sync.WaitGroup
	for range 2 {
		writers.Add(1)
		go func() {
			defer writers.Done()
			requestUser := payer
			balance, err := postRepo.Tip(context.Background(), target.PublicID, &requestUser, decimal.NewFromInt(2), key)
			outcomes <- outcome{balance: balance, err: err}
		}()
	}
	writers.Wait()
	close(outcomes)
	for result := range outcomes {
		if result.err != nil || result.balance == nil || !result.balance.Equal(decimal.NewFromInt(8)) {
			t.Fatalf("concurrent retry outcome = %v, %v; want balance 8", result.balance, result.err)
		}
	}

	assertStoredBalance(t, db, payer.ID, "8")
	assertStoredBalance(t, db, payee.ID, "6")
	var detailCount int64
	if err := db.Model(&models.EngagementDetail{}).
		Where("engagement_id = ? AND kind = ?", aggregate.ID, models.EngagementKindTip).
		Count(&detailCount).Error; err != nil {
		t.Fatalf("count concurrent retry details: %v", err)
	}
	if detailCount != 1 {
		t.Fatalf("concurrent retry detail count = %d, want 1", detailCount)
	}
	assertTipAggregateTotals(t, db, aggregate.ID, 1, "2")
}

func TestConcurrentTipKeyReuseAcrossPostsCommitsOnlyOneOperationIntegration(t *testing.T) {
	db := locationRaceIntegrationDB(t)
	db, postRepo, payer, firstPayee, firstTarget, _ := prepareTipIntegrationOnDB(t, db, datatypes.JSON([]byte("{}")))

	secondPayee := models.User{
		ID: uuid.New(), PublicID: time.Now().UTC().UnixNano(), Domain: models.CoolVibes,
		UserName: "tip-second-payee-" + uuid.NewString(), DisplayName: "Second Payee",
		UserRole: constants.UserRoleUser, Balance: decimal.NewFromInt(4),
	}
	if err := db.Omit(clause.Associations).Create(&secondPayee).Error; err != nil {
		t.Fatalf("create second tip payee: %v", err)
	}
	audience := "public"
	secondTarget := post.Post{
		ID: uuid.New(), PublicID: time.Now().UTC().UnixNano() + 1, AuthorID: secondPayee.ID,
		PostKind: post.PostKindPost, Domain: models.CoolVibes, Published: true, Audience: &audience,
	}
	if err := db.Omit(clause.Associations).Create(&secondTarget).Error; err != nil {
		t.Fatalf("create second tip target: %v", err)
	}
	cleanupTipIntegrationFixture(t, db, []models.User{payer, firstPayee, secondPayee}, []post.Post{firstTarget, secondTarget})

	key := mustTipIdempotencyKey(t, "tip-cross-post-race")
	type outcome struct {
		postID int64
		err    error
	}
	outcomes := make(chan outcome, 2)
	var writers sync.WaitGroup
	for _, postID := range []int64{firstTarget.PublicID, secondTarget.PublicID} {
		postID := postID
		writers.Add(1)
		go func() {
			defer writers.Done()
			requestUser := payer
			_, err := postRepo.Tip(context.Background(), postID, &requestUser, decimal.NewFromInt(2), key)
			outcomes <- outcome{postID: postID, err: err}
		}()
	}
	writers.Wait()
	close(outcomes)

	successes := 0
	conflicts := 0
	for result := range outcomes {
		switch {
		case result.err == nil:
			successes++
		case errors.Is(result.err, domainwallet.ErrIdempotencyConflict):
			conflicts++
		default:
			t.Fatalf("cross-post tip %d error = %v", result.postID, result.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("cross-post outcomes = %d successes/%d conflicts, want 1/1", successes, conflicts)
	}
	assertStoredBalance(t, db, payer.ID, "8")

	var payees []models.User
	if err := db.Select("id", "balance").Where("id IN ?", []uuid.UUID{firstPayee.ID, secondPayee.ID}).Find(&payees).Error; err != nil {
		t.Fatalf("load cross-post payee balances: %v", err)
	}
	creditedTotal := decimal.Zero
	for _, payee := range payees {
		creditedTotal = creditedTotal.Add(payee.Balance)
	}
	if !creditedTotal.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("cross-post payee balance total = %s, want 10", creditedTotal)
	}

	var detailCount int64
	if err := db.Model(&models.EngagementDetail{}).
		Where("dedupe_key = ?", scopedTipDedupeKey(payer.ID, key)).
		Count(&detailCount).Error; err != nil {
		t.Fatalf("count cross-post dedupe details: %v", err)
	}
	if detailCount != 1 {
		t.Fatalf("cross-post dedupe detail count = %d, want 1", detailCount)
	}
}

func assertStoredBalance(t *testing.T, db *gorm.DB, userID uuid.UUID, expected string) {
	t.Helper()
	var user models.User
	if err := db.Select("id", "balance").First(&user, "id = ?", userID).Error; err != nil {
		t.Fatalf("load balance for %s: %v", userID, err)
	}
	want := decimal.RequireFromString(expected)
	if !user.Balance.Equal(want) {
		t.Fatalf("stored balance for %s = %s, want %s", userID, user.Balance, want)
	}
}

func assertTipAggregateTotals(t *testing.T, db *gorm.DB, aggregateID uuid.UUID, expectedCount int64, expectedAmount string) {
	t.Helper()
	var aggregate models.Engagement
	if err := db.Select("id", "counts").First(&aggregate, "id = ?", aggregateID).Error; err != nil {
		t.Fatalf("load tip aggregate: %v", err)
	}
	var counts map[string]interface{}
	if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
		t.Fatalf("decode tip aggregate counts: %v", err)
	}
	if counts["tip_count"] != float64(expectedCount) || counts["tip_amount"] != expectedAmount {
		t.Fatalf("tip aggregate counts = %#v, want count %d amount %s", counts, expectedCount, expectedAmount)
	}
}

func cleanupTipIntegrationFixture(t *testing.T, db *gorm.DB, users []models.User, posts []post.Post) {
	t.Helper()
	t.Cleanup(func() {
		postIDs := make([]uuid.UUID, 0, len(posts))
		for _, target := range posts {
			postIDs = append(postIDs, target.ID)
		}
		var aggregateIDs []uuid.UUID
		db.Model(&models.Engagement{}).
			Where("contentable_id IN ? AND contentable_type = ?", postIDs, models.EngagementContentableTypePost).
			Pluck("id", &aggregateIDs)
		if len(aggregateIDs) > 0 {
			db.Where("engagement_id IN ?", aggregateIDs).Delete(&models.EngagementDetail{})
			db.Where("id IN ?", aggregateIDs).Delete(&models.Engagement{})
		}
		db.Unscoped().Where("id IN ?", postIDs).Delete(&post.Post{})
		userIDs := make([]uuid.UUID, 0, len(users))
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
		}
		db.Unscoped().Where("id IN ?", userIDs).Delete(&models.User{})
	})
}

func mustTipIdempotencyKey(t *testing.T, raw string) domainwallet.IdempotencyKey {
	t.Helper()
	key, err := domainwallet.NewIdempotencyKey(raw)
	if err != nil {
		t.Fatalf("NewIdempotencyKey(%q): %v", raw, err)
	}
	return key
}
