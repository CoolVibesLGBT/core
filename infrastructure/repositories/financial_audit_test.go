package repositories

import (
	"context"
	"core/constants"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"core/models"
	"core/models/notifications"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func financialUser(t *testing.T, db *gorm.DB) models.User {
	t.Helper()
	user := models.User{ID: uuid.New(), PublicID: time.Now().UnixNano(), UserName: "audit_" + uuid.NewString(), Domain: models.CoolVibes, UserRole: constants.UserRoleUser}
	if err := db.Omit(clause.Associations).Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

func assertFinancialOutbox(t *testing.T, db *gorm.DB, receiver uuid.UUID, want int64) {
	t.Helper()
	var notices, jobs int64
	if err := db.Model(&notifications.Notification{}).Where("user_id = ? AND type = ?", receiver, notifications.NotificationTypeReferral).Count(&notices).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("notification_deliveries").Joins("JOIN notifications n ON n.id = notification_deliveries.id").Where("n.user_id = ? AND n.type = ?", receiver, notifications.NotificationTypeReferral).Count(&jobs).Error; err != nil {
		t.Fatal(err)
	}
	if notices != want || jobs != want {
		t.Fatalf("reward notices/jobs = %d/%d, want %d", notices, jobs, want)
	}
}

func TestFinancialReferralOnlyOneReferrerIncludingLegacyIntegration(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "new"
		if legacy {
			name = "legacy"
		}
		t.Run(name, func(t *testing.T) {
			db, repo, referrer, referred, _, _ := prepareReferralIntegration(t, datatypes.JSON([]byte("{}")))
			other := financialUser(t, db)
			if _, err := creditReferralForTest(repo, context.Background(), referrer.ID, referred.ID, decimal.NewFromInt(2)); err != nil {
				t.Fatal(err)
			}
			if legacy {
				if err := db.Model(&models.EngagementDetail{}).Where("engager_id = ? AND kind = ?", referred.ID, models.EngagementKindReferral).Update("dedupe_key", "user-referral:"+referred.ID.String()+":"+referrer.ID.String()).Error; err != nil {
					t.Fatal(err)
				}
			}
			if _, err := creditReferralForTest(repo, context.Background(), other.ID, referred.ID, decimal.NewFromInt(2)); !errors.Is(err, domainuser.ErrReferralAlreadyAssigned) {
				t.Fatalf("different referrer = %v", err)
			}
			if _, err := creditReferralForTest(repo, context.Background(), referrer.ID, referred.ID, decimal.NewFromInt(50)); err != nil {
				t.Fatal(err)
			}
			assertStoredBalance(t, db, referrer.ID, "6")
			assertStoredBalance(t, db, other.ID, "0")
			assertFinancialOutbox(t, db, referrer.ID, 1)
			assertFinancialOutbox(t, db, other.ID, 0)
		})
	}
}

func TestFinancialConcurrentDifferentReferrersRewardOnceIntegration(t *testing.T) {
	db, repo, referrer, referred, _, _ := prepareReferralIntegration(t, datatypes.JSON([]byte("{}")))
	other := financialUser(t, db)
	start := make(chan struct{})
	results := make(chan error, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, id := range []uuid.UUID{referrer.ID, other.ID} {
		go func(id uuid.UUID) {
			<-start
			_, err := creditReferralForTest(repo, ctx, id, referred.ID, decimal.NewFromInt(2))
			results <- err
		}(id)
	}
	close(start)
	successes, conflicts := 0, 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domainuser.ErrReferralAlreadyAssigned):
			conflicts++
		default:
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes/conflicts %d/%d", successes, conflicts)
	}
	var users []models.User
	if err := db.Where("id IN ?", []uuid.UUID{referrer.ID, other.ID}).Find(&users).Error; err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || !users[0].Balance.Add(users[1].Balance).Equal(decimal.NewFromInt(6)) {
		t.Fatalf("balances = %+v", users)
	}
}

func TestFinancialRegistrationRollsBackAccountOnPendingRewardFailureIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	referrer := financialUser(t, db)
	repo := NewUserRepository(db, nil, nil, NewEngagementRepository(db), nil)
	user := models.User{ID: uuid.New(), PublicID: time.Now().UnixNano(), UserName: "new_signup", Domain: models.CoolVibes, UserRole: constants.UserRoleUser}
	referral, err := domainuser.NewReferral(referrer.ID, user.ID, decimal.NewFromInt(50))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`ALTER TABLE referral_rewards ADD CONSTRAINT fail_reward_promise CHECK (amount < 0)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRegisteredUser(context.Background(), &user, &referral); err == nil {
		t.Fatal("signup succeeded despite broken outbox")
	}
	var count int64
	if err := db.Model(&models.User{}).Where("id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed reward left a registered account")
	}
	assertStoredBalance(t, db, referrer.ID, "0")
	assertFinancialOutbox(t, db, referrer.ID, 0)
	if err := db.Model(&models.EngagementDetail{}).Where("engager_id = ?", user.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("reward details %d, %v", count, err)
	}
	if err := db.Exec(`ALTER TABLE referral_rewards DROP CONSTRAINT fail_reward_promise`).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRegisteredUser(context.Background(), &user, &referral); err != nil {
		t.Fatal(err)
	}
	assertStoredBalance(t, db, referrer.ID, "0")
	assertFinancialOutbox(t, db, referrer.ID, 0)
	var reward models.ReferralReward
	if err := db.Where("referred_id = ?", user.ID).Take(&reward).Error; err != nil || reward.Status != domainuser.ReferralRewardPending {
		t.Fatalf("pending referral = %+v, %v", reward, err)
	}
}

func TestFinancialReferralRejectsUnavailableAccountsIntegration(t *testing.T) {
	for _, scenario := range []string{"banned", "bot", "domain"} {
		t.Run(scenario, func(t *testing.T) {
			db, repo, referrer, referred, _, _ := prepareReferralIntegration(t, datatypes.JSON([]byte("{}")))
			field, value := "user_role", any(constants.UserRoleBanned)
			switch scenario {
			case "bot":
				field, value = "is_bot", true
			case "domain":
				field, value = "domain", "another-domain"
			}
			if err := db.Model(&models.User{}).Where("id = ?", referrer.ID).Update(field, value).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := creditReferralForTest(repo, context.Background(), referrer.ID, referred.ID, decimal.NewFromInt(2)); !errors.Is(err, domainuser.ErrReferralUnavailable) {
				t.Fatalf("reward = %v", err)
			}
			assertStoredBalance(t, db, referrer.ID, "4")
			assertFinancialOutbox(t, db, referrer.ID, 0)
		})
	}
}

func TestFinancialTipRejectsIneligibleRecipientsWithoutMovingMoneyIntegration(t *testing.T) {
	for _, scenario := range []string{"blocked-by-payee", "blocked-by-payer", "banned", "bot", "domain", "post-domain", "overflow", "insufficient"} {
		t.Run(scenario, func(t *testing.T) {
			db, repo, payer, payee, target, _ := prepareTipIntegration(t, datatypes.JSON([]byte("{}")))
			expected := domainwallet.ErrTransferUnavailable
			payeeBalance := "4"
			switch scenario {
			case "blocked-by-payee", "blocked-by-payer":
				actor, other := payee.ID, payer.ID
				if scenario == "blocked-by-payer" {
					actor, other = other, actor
				}
				if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(context.Background(), actor, other, socialIntent(t, domainuser.InteractionBlock, true)); err != nil {
					t.Fatal(err)
				}
				expected = domainwallet.ErrTransferBlocked
			case "banned":
				if err := db.Model(&models.User{}).Where("id = ?", payee.ID).Update("user_role", constants.UserRoleBanned).Error; err != nil {
					t.Fatal(err)
				}
			case "bot":
				if err := db.Model(&models.User{}).Where("id = ?", payee.ID).Update("is_bot", true).Error; err != nil {
					t.Fatal(err)
				}
			case "domain":
				if err := db.Model(&models.User{}).Where("id = ?", payee.ID).Update("domain", "another-domain").Error; err != nil {
					t.Fatal(err)
				}
			case "post-domain":
				if err := db.Table("posts").Where("id = ?", target.ID).Update("domain", "another-domain").Error; err != nil {
					t.Fatal(err)
				}
			case "overflow":
				payeeBalance = "99999999999999999999.999999999999999999"
				if err := db.Model(&models.User{}).Where("id = ?", payee.ID).Update("balance", decimal.RequireFromString(payeeBalance)).Error; err != nil {
					t.Fatal(err)
				}
				expected = domainwallet.ErrAmountOutOfRange
			case "insufficient":
				expected = nil
			}
			amount := decimal.NewFromInt(2)
			if scenario == "insufficient" {
				amount = decimal.NewFromInt(11)
			}
			_, err := repo.Tip(context.Background(), target.PublicID, &payer, amount, mustTipIdempotencyKey(t, "financial-policy-check"))
			if expected != nil && !errors.Is(err, expected) {
				t.Fatalf("tip error = %v, want %v", err, expected)
			}
			if scenario == "insufficient" && (err == nil || err.Error() != constants.ErrInsufficientBalance.String()) {
				t.Fatalf("insufficient tip = %v", err)
			}
			assertStoredBalance(t, db, payer.ID, "10")
			assertStoredBalance(t, db, payee.ID, payeeBalance)
			var count int64
			if err := db.Model(&models.EngagementDetail{}).Where("kind = ?", models.EngagementKindTip).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("tip records = %d, %v", count, err)
			}
		})
	}
}

func TestFinancialConcurrentTipsCannotOverspendIntegration(t *testing.T) {
	db, repo, payer, payee, target, aggregate := prepareTipIntegration(t, datatypes.JSON([]byte("{}")))
	start := make(chan struct{})
	results := make(chan error, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, raw := range []string{"first-tip-request", "second-tip-request"} {
		key := mustTipIdempotencyKey(t, raw)
		go func() {
			local := payer
			<-start
			_, err := repo.Tip(ctx, target.PublicID, &local, decimal.NewFromInt(7), key)
			results <- err
		}()
	}
	close(start)
	success, failure := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			success++
		} else if err.Error() == constants.ErrInsufficientBalance.String() {
			failure++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || failure != 1 {
		t.Fatalf("success/failure %d/%d", success, failure)
	}
	assertStoredBalance(t, db, payer.ID, "3")
	assertStoredBalance(t, db, payee.ID, "11")
	assertTipAggregateTotals(t, db, aggregate.ID, 1, "7")
}

func TestFinancialStaleProfileCannotOverwriteTipsOrRewardsIntegration(t *testing.T) {
	db, repo, payer, payee, target, _ := prepareTipIntegration(t, datatypes.JSON([]byte("{}")))
	stalePayer, stalePayee := payer, payee
	referred := financialUser(t, db)
	if _, err := creditReferralForTest(repo.userRepo, context.Background(), payee.ID, referred.ID, decimal.NewFromInt(50)); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Tip(context.Background(), target.PublicID, &payer, decimal.NewFromInt(2), mustTipIdempotencyKey(t, "stale-profile-tip")); err != nil {
		t.Fatal(err)
	}
	for _, user := range []*models.User{&stalePayer, &stalePayee} {
		user.DisplayName = "Updated profile"
		if err := repo.userRepo.UpdateUser(user); err != nil {
			t.Fatal(err)
		}
	}
	assertStoredBalance(t, db, payer.ID, "8")
	assertStoredBalance(t, db, payee.ID, "56")
}

func TestFinancialReferralLookupIgnoresAmbiguousDisplayNamesIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, attacker := financialUser(t, db), financialUser(t, db)
	if err := db.Model(&models.User{}).Where("id = ?", attacker.ID).Update("display_name", owner.UserName).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewUserRepository(db, nil, nil, nil, nil)
	user, err := repo.GetByNameOrMailWithoutRelations(owner.UserName)
	if err != nil || user.ID != owner.ID {
		t.Fatalf("referrer resolved to %+v, %v", user, err)
	}
	if _, err := repo.GetByNameOrMailWithoutRelations("does-not-exist"); err == nil {
		t.Fatal("unknown code accepted")
	}
}

func TestFinancialAddBalanceRejectsUnsafePrecisionBeforeDatabase(t *testing.T) {
	repo := &UserRepository{}
	for _, amount := range []decimal.Decimal{decimal.New(1, math.MaxInt32), decimal.New(1, math.MinInt32)} {
		if _, err := repo.AddBalance(context.Background(), uuid.New(), amount); !errors.Is(err, domainwallet.ErrAmountOutOfRange) {
			t.Fatalf("unsafe credit error %v", err)
		}
	}
}

func TestFinancialConcurrentDifferentReferralsDoNotLoseCreditsIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner := financialUser(t, db)
	repo := NewUserRepository(db, nil, nil, NewEngagementRepository(db), nil)
	users := []models.User{financialUser(t, db), financialUser(t, db), financialUser(t, db)}
	var wg sync.WaitGroup
	results := make(chan error, len(users))
	start := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, user := range users {
		wg.Add(1)
		go func(user models.User) {
			defer wg.Done()
			<-start
			_, err := creditReferralForTest(repo, ctx, owner.ID, user.ID, decimal.NewFromInt(50))
			results <- err
		}(user)
	}
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertStoredBalance(t, db, owner.ID, "150")
	assertFinancialOutbox(t, db, owner.ID, 3)
}
