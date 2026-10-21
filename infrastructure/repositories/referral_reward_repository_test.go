package repositories

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"core/models"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"strings"
	"testing"
	"time"
)

func referralAdmin(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	admin := financialUser(t, db)
	if err := db.Model(&models.User{}).Where("id = ?", admin.ID).Update("user_role", constants.UserRoleAdmin).Error; err != nil {
		t.Fatal(err)
	}
	admin.UserRole = constants.UserRoleAdmin
	return admin
}
func pendingReferral(t *testing.T, db *gorm.DB, referrer, referred models.User) models.ReferralReward {
	t.Helper()
	repo := NewUserRepository(db, nil, nil, NewEngagementRepository(db), nil)
	if _, err := repo.AddReferral(context.Background(), referrer.ID, referred.ID, decimal.NewFromInt(50)); err != nil {
		t.Fatal(err)
	}
	var reward models.ReferralReward
	if err := db.Where("referred_id = ?", referred.ID).Take(&reward).Error; err != nil {
		t.Fatal(err)
	}
	return reward
}
func referralReview(t *testing.T, decision, reference string) domainuser.ReferralReview {
	t.Helper()
	review, err := domainuser.NewReferralReview(decision, "Checked the referral evidence manually", reference)
	if err != nil {
		t.Fatal(err)
	}
	return review
}
func assertRewardStatus(t *testing.T, db *gorm.DB, id uuid.UUID, status domainuser.ReferralRewardStatus) {
	t.Helper()
	var reward models.ReferralReward
	if err := db.Take(&reward, "id = ?", id).Error; err != nil || reward.Status != status {
		t.Fatalf("reward status %+v, %v; want %s", reward, err, status)
	}
}
func TestReferralPendingRewardsAreNotSpendableIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, referred := financialUser(t, db), financialUser(t, db)
	reward := pendingReferral(t, db, owner, referred)
	if reward.Status != domainuser.ReferralRewardPending || !reward.Amount.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("promise %+v", reward)
	}
	if _, err := NewUserRepository(db, nil, nil, nil, nil).AddReferral(context.Background(), owner.ID, referred.ID, decimal.NewFromInt(100)); err != nil {
		t.Fatal(err)
	}
	var unchanged models.ReferralReward
	if err := db.Take(&unchanged, "id = ?", reward.ID).Error; err != nil || !unchanged.Amount.Equal(decimal.NewFromInt(50)) {
		t.Fatal("retry changed the promised reward")
	}
	gifts := NewBroadcastGiftRepository(db)
	session, err := gifts.StartBroadcastSession(context.Background(), referred.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gifts.SendBroadcastGift(context.Background(), owner.ID, session.ID, decimal.NewFromInt(1), mustTipIdempotencyKey(t, "pending-cannot-gift")); !errors.Is(err, domainwallet.ErrInsufficientFunds) {
		t.Fatalf("spent pending promise: %v", err)
	}
	assertStoredBalance(t, db, owner.ID, "0")
	assertFinancialOutbox(t, db, owner.ID, 0)
	admin := referralAdmin(t, db)
	repo := NewReferralRewardRepository(db)
	review := referralReview(t, "approve", "trusted-id:person-one")
	first, err := repo.ReviewReferralReward(context.Background(), admin.ID, reward.ID, review)
	if err != nil || first.Status != domainuser.ReferralRewardPaid {
		t.Fatalf("approval %+v, %v", first, err)
	}
	if _, err := repo.ReviewReferralReward(context.Background(), admin.ID, reward.ID, review); err != nil {
		t.Fatal(err)
	}
	assertStoredBalance(t, db, owner.ID, "50")
	assertFinancialOutbox(t, db, owner.ID, 1)
	if _, err := repo.ReviewReferralReward(context.Background(), admin.ID, reward.ID, referralReview(t, "reject", "")); !errors.Is(err, domainuser.ErrReferralReviewConflict) {
		t.Fatalf("reversed approval: %v", err)
	}
}
func TestReferralRequiresIndependentCurrentAdminIntegration(t *testing.T) {
	for _, scenario := range []string{"ordinary-user", "moderator", "demoted", "referrer", "referred", "bot-admin"} {
		t.Run(scenario, func(t *testing.T) {
			db := privatePhotoIntegrationDB(t)
			owner, referred := financialUser(t, db), financialUser(t, db)
			reward := pendingReferral(t, db, owner, referred)
			admin := referralAdmin(t, db)
			actor := admin.ID
			switch scenario {
			case "ordinary-user", "demoted":
				if err := db.Model(&models.User{}).Where("id = ?", admin.ID).Update("user_role", constants.UserRoleUser).Error; err != nil {
					t.Fatal(err)
				}
			case "moderator":
				if err := db.Model(&models.User{}).Where("id = ?", admin.ID).Update("user_role", constants.UserRoleModerator).Error; err != nil {
					t.Fatal(err)
				}
			case "referrer":
				actor = owner.ID
				if err := db.Model(&models.User{}).Where("id = ?", actor).Update("user_role", constants.UserRoleAdmin).Error; err != nil {
					t.Fatal(err)
				}
			case "referred":
				actor = referred.ID
				if err := db.Model(&models.User{}).Where("id = ?", actor).Update("user_role", constants.UserRoleSuperAdmin).Error; err != nil {
					t.Fatal(err)
				}
			case "bot-admin":
				if err := db.Model(&models.User{}).Where("id = ?", admin.ID).Update("is_bot", true).Error; err != nil {
					t.Fatal(err)
				}
			}
			if _, err := NewReferralRewardRepository(db).ReviewReferralReward(context.Background(), actor, reward.ID, referralReview(t, "approve", "trusted-person-reference")); !errors.Is(err, domainuser.ErrReferralReviewForbidden) {
				t.Fatalf("approval %v", err)
			}
			assertRewardStatus(t, db, reward.ID, domainuser.ReferralRewardPending)
			assertStoredBalance(t, db, owner.ID, "0")
		})
	}
}
func TestReferralConcurrentApprovalsPayOnceIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, referred := financialUser(t, db), financialUser(t, db)
	reward := pendingReferral(t, db, owner, referred)
	admin := referralAdmin(t, db)
	repo := NewReferralRewardRepository(db)
	review := referralReview(t, "approve", "trusted-person-reference")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 12)
	for range 12 {
		go func() { <-start; _, err := repo.ReviewReferralReward(ctx, admin.ID, reward.ID, review); results <- err }()
	}
	close(start)
	for range 12 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	assertStoredBalance(t, db, owner.ID, "50")
	assertFinancialOutbox(t, db, owner.ID, 1)
	assertRewardStatus(t, db, reward.ID, domainuser.ReferralRewardPaid)
	var count int64
	if err := db.Model(&models.EngagementDetail{}).Where("engager_id = ? AND kind = ?", referred.ID, models.EngagementKindReferral).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("ledger %d, %v", count, err)
	}
}
func TestReferralVerifiedIdentityCannotQualifyMultipleAccountsIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner := financialUser(t, db)
	admin := referralAdmin(t, db)
	first := pendingReferral(t, db, owner, financialUser(t, db))
	second := pendingReferral(t, db, owner, financialUser(t, db))
	repo := NewReferralRewardRepository(db)
	review := referralReview(t, "approve", "trusted-person-reference")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, id := range []uuid.UUID{first.ID, second.ID} {
		go func(id uuid.UUID) {
			<-start
			_, err := repo.ReviewReferralReward(ctx, admin.ID, id, review)
			results <- err
		}(id)
	}
	close(start)
	paid, denied := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			paid++
		} else if errors.Is(err, domainuser.ErrReferralIdentityUsed) {
			denied++
		} else {
			t.Fatal(err)
		}
	}
	if paid != 1 || denied != 1 {
		t.Fatalf("paid/denied %d/%d", paid, denied)
	}
	assertStoredBalance(t, db, owner.ID, "50")
	assertFinancialOutbox(t, db, owner.ID, 1)
}
func TestReferralRejectedRewardCannotBeRecreatedIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, referred := financialUser(t, db), financialUser(t, db)
	reward := pendingReferral(t, db, owner, referred)
	admin := referralAdmin(t, db)
	repo := NewReferralRewardRepository(db)
	rejection := referralReview(t, "reject", "")
	if _, err := repo.ReviewReferralReward(context.Background(), admin.ID, reward.ID, rejection); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReviewReferralReward(context.Background(), admin.ID, reward.ID, rejection); err != nil {
		t.Fatal(err)
	}
	if _, err := NewUserRepository(db, nil, nil, nil, nil).AddReferral(context.Background(), owner.ID, referred.ID, decimal.NewFromInt(50)); err != nil {
		t.Fatal(err)
	}
	assertRewardStatus(t, db, reward.ID, domainuser.ReferralRewardRejected)
	assertStoredBalance(t, db, owner.ID, "0")
	assertFinancialOutbox(t, db, owner.ID, 0)
	if _, err := repo.ReviewReferralReward(context.Background(), admin.ID, reward.ID, referralReview(t, "approve", "trusted-person-reference")); !errors.Is(err, domainuser.ErrReferralReviewConflict) {
		t.Fatalf("reopened rejection %v", err)
	}
}
func TestReferralApprovalOutboxFailureKeepsPromiseRetryableIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, referred := financialUser(t, db), financialUser(t, db)
	reward := pendingReferral(t, db, owner, referred)
	admin := referralAdmin(t, db)
	repo := NewReferralRewardRepository(db)
	review := referralReview(t, "approve", "trusted-person-reference")
	if err := db.Exec(`ALTER TABLE notification_deliveries ADD CONSTRAINT fail_approval_delivery CHECK (attempts < 0)`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReviewReferralReward(context.Background(), admin.ID, reward.ID, review); err == nil {
		t.Fatal("approval committed with broken outbox")
	}
	assertRewardStatus(t, db, reward.ID, domainuser.ReferralRewardPending)
	assertStoredBalance(t, db, owner.ID, "0")
	assertFinancialOutbox(t, db, owner.ID, 0)
	var stored models.ReferralReward
	if err := db.Take(&stored, "id = ?", reward.ID).Error; err != nil || stored.VerifiedIdentityHash != nil || stored.ReviewedAt != nil {
		t.Fatalf("failed review consumed identity: %+v, %v", stored, err)
	}
	if err := db.Exec(`ALTER TABLE notification_deliveries DROP CONSTRAINT fail_approval_delivery`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReviewReferralReward(context.Background(), admin.ID, reward.ID, review); err != nil {
		t.Fatal(err)
	}
	assertStoredBalance(t, db, owner.ID, "50")
	assertFinancialOutbox(t, db, owner.ID, 1)
}
func TestReferralListIsPrivateAndCursorStableIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, outsider := financialUser(t, db), financialUser(t, db)
	admin := referralAdmin(t, db)
	for range 3 {
		pendingReferral(t, db, owner, financialUser(t, db))
	}
	if err := db.Model(&models.ReferralReward{}).Where("referrer_id = ?", owner.ID).Update("created_at", time.Now().UTC().Truncate(time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewReferralRewardRepository(db)
	first, next, err := repo.ListReferralRewards(context.Background(), owner.ID, false, "", nil, 2)
	if err != nil || len(first) != 2 || next == nil {
		t.Fatalf("first page %+v, %v", first, err)
	}
	second, end, err := repo.ListReferralRewards(context.Background(), owner.ID, false, "", next, 2)
	if err != nil || len(second) != 1 || end != nil || second[0].ID == first[0].ID || second[0].ID == first[1].ID {
		t.Fatalf("second page %+v, %v", second, err)
	}
	rows, _, err := repo.ListReferralRewards(context.Background(), outsider.ID, false, "", nil, 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("outsider rows %+v, %v", rows, err)
	}
	if _, _, err := repo.ListReferralRewards(context.Background(), outsider.ID, true, "", nil, 10); !errors.Is(err, domainuser.ErrReferralReviewForbidden) {
		t.Fatalf("outsider queue %v", err)
	}
	rows, _, err = repo.ListReferralRewards(context.Background(), admin.ID, true, domainuser.ReferralRewardPending, nil, 10)
	if err != nil || len(rows) != 3 {
		t.Fatalf("queue %+v, %v", rows, err)
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"verified_identity_hash", "review_note", "reviewed_by_id", owner.ID.String()} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("private reward detail leaked: %s", secret)
		}
	}
}
func TestLegacyPaidReferralCannotCreateNewPendingRewardIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, referred := financialUser(t, db), financialUser(t, db)
	repo := NewUserRepository(db, nil, nil, NewEngagementRepository(db), nil)
	if _, err := creditReferralForTest(repo, context.Background(), owner.ID, referred.ID, decimal.NewFromInt(50)); err != nil {
		t.Fatal(err)
	}
	if err := db.Where("referred_id = ?", referred.ID).Delete(&models.ReferralReward{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.EngagementDetail{}).Where("engager_id = ? AND kind = ?", referred.ID, models.EngagementKindReferral).Update("dedupe_key", "user-referral:"+referred.ID.String()+":"+owner.ID.String()).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddReferral(context.Background(), owner.ID, referred.ID, decimal.NewFromInt(50)); err != nil {
		t.Fatal(err)
	}
	other := financialUser(t, db)
	if _, err := repo.AddReferral(context.Background(), other.ID, referred.ID, decimal.NewFromInt(50)); !errors.Is(err, domainuser.ErrReferralAlreadyAssigned) {
		t.Fatalf("legacy ownership bypass %v", err)
	}
	var count int64
	if err := db.Model(&models.ReferralReward{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("reminted pending %d, %v", count, err)
	}
	assertStoredBalance(t, db, owner.ID, "50")
}

var _ ports.ReferralRewardRepository = (*ReferralRewardRepository)(nil)
