package repositories

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"core/models"
	"core/models/notifications"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"sync"
	"testing"
	"time"
)

func prepareBroadcastGift(t *testing.T) (*gorm.DB, *BroadcastGiftRepository, models.User, models.User, ports.BroadcastSessionView) {
	t.Helper()
	db := privatePhotoIntegrationDB(t)
	payer, owner := financialUser(t, db), financialUser(t, db)
	if err := db.Model(&models.User{}).Where("id = ?", payer.ID).Update("balance", decimal.NewFromInt(10)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.User{}).Where("id = ?", owner.ID).Update("balance", decimal.NewFromInt(4)).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewBroadcastGiftRepository(db)
	session, err := repo.StartBroadcastSession(context.Background(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	return db, repo, payer, owner, session
}
func assertGiftRecords(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	for _, model := range []any{&models.BroadcastGift{}, &notifications.Notification{}, &notifications.Delivery{}} {
		q := db.Model(model)
		if _, ok := model.(*notifications.Notification); ok {
			q = q.Where("type = ?", notifications.NotificationTypeBroadcastGift)
		}
		var count int64
		if err := q.Count(&count).Error; err != nil || count != want {
			t.Fatalf("%T count %d, want %d, error %v", model, count, want, err)
		}
	}
	var count int64
	if err := db.Model(&models.EngagementDetail{}).Where("kind = ?", models.EngagementKindGift).Count(&count).Error; err != nil || count != want {
		t.Fatalf("gift ledger %d, want %d, error %v", count, want, err)
	}
}
func TestBroadcastGiftTransfersAndReplaysAfterClosureIntegration(t *testing.T) {
	db, repo, payer, owner, session := prepareBroadcastGift(t)
	ctx := context.Background()
	key := mustTipIdempotencyKey(t, "local-gift-request")
	first, err := repo.SendBroadcastGift(ctx, payer.ID, session.ID, decimal.RequireFromString("2.25"), key)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == uuid.Nil || first.BroadcastID != session.ID || int64(first.RecipientPublicID) != owner.PublicID || !first.Balance.Equal(decimal.RequireFromString("7.75")) {
		t.Fatalf("gift receipt %+v", first)
	}
	if _, err := repo.ChangeBroadcastSession(ctx, owner.ID, session.ID, true); err != nil {
		t.Fatal(err)
	}
	again, err := repo.SendBroadcastGift(ctx, payer.ID, session.ID, decimal.RequireFromString("2.25"), key)
	if err != nil || again.ID != first.ID || !again.Balance.Equal(first.Balance) {
		t.Fatalf("replay %+v, %v", again, err)
	}
	assertStoredBalance(t, db, payer.ID, "7.75")
	assertStoredBalance(t, db, owner.ID, "6.25")
	assertGiftRecords(t, db, 1)
	var aggregate models.Engagement
	if err := db.Where("contentable_id = ? AND contentable_type = ?", owner.ID, models.EngagementContentableTypeUser).Take(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	var counts map[string]any
	if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
		t.Fatal(err)
	}
	if counts["gift_count"] != float64(1) || counts["gift_amount"] != "2.25" {
		t.Fatalf("gift counters %+v", counts)
	}
	if _, err := repo.SendBroadcastGift(ctx, payer.ID, session.ID, decimal.NewFromInt(1), key); !errors.Is(err, domainwallet.ErrIdempotencyConflict) {
		t.Fatalf("different amount %v", err)
	}
}
func TestBroadcastGiftKeyCannotMoveToAnotherSessionIntegration(t *testing.T) {
	db, repo, payer, owner, session := prepareBroadcastGift(t)
	ctx := context.Background()
	key := mustTipIdempotencyKey(t, "gift-cross-session")
	if _, err := repo.SendBroadcastGift(ctx, payer.ID, session.ID, decimal.NewFromInt(2), key); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ChangeBroadcastSession(ctx, owner.ID, session.ID, true); err != nil {
		t.Fatal(err)
	}
	next, err := repo.StartBroadcastSession(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SendBroadcastGift(ctx, payer.ID, next.ID, decimal.NewFromInt(2), key); !errors.Is(err, domainwallet.ErrIdempotencyConflict) {
		t.Fatalf("different session %v", err)
	}
	assertStoredBalance(t, db, payer.ID, "8")
	assertStoredBalance(t, db, owner.ID, "6")
	assertGiftRecords(t, db, 1)
}
func TestBroadcastGiftRollsBackOnOutboxFailureIntegration(t *testing.T) {
	db, repo, payer, owner, session := prepareBroadcastGift(t)
	key := mustTipIdempotencyKey(t, "gift-outbox-failure")
	if err := db.Exec(`ALTER TABLE notification_deliveries ADD CONSTRAINT fail_gift_delivery CHECK (attempts < 0)`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SendBroadcastGift(context.Background(), payer.ID, session.ID, decimal.NewFromInt(2), key); err == nil {
		t.Fatal("gift committed without outbox")
	}
	assertStoredBalance(t, db, payer.ID, "10")
	assertStoredBalance(t, db, owner.ID, "4")
	assertGiftRecords(t, db, 0)
	if err := db.Exec(`ALTER TABLE notification_deliveries DROP CONSTRAINT fail_gift_delivery`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SendBroadcastGift(context.Background(), payer.ID, session.ID, decimal.NewFromInt(2), key); err != nil {
		t.Fatal(err)
	}
	assertGiftRecords(t, db, 1)
}
func TestBroadcastGiftPolicyRejectsWithoutMoneyMovementIntegration(t *testing.T) {
	for _, scenario := range []string{"blocked", "reverse-block", "bot", "banned", "payer-banned", "domain", "expired", "ended", "private", "overflow", "insufficient", "self"} {
		t.Run(scenario, func(t *testing.T) {
			db, repo, payer, owner, session := prepareBroadcastGift(t)
			ctx := context.Background()
			amount := decimal.NewFromInt(2)
			expected := domainwallet.ErrTransferUnavailable
			ownerBalance := "4"
			switch scenario {
			case "blocked", "reverse-block":
				actor, target := owner.ID, payer.ID
				if scenario == "reverse-block" {
					actor, target = target, actor
				}
				if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(ctx, actor, target, socialIntent(t, domainuser.InteractionBlock, true)); err != nil {
					t.Fatal(err)
				}
				expected = domainwallet.ErrTransferBlocked
			case "bot":
				if err := db.Model(&models.User{}).Where("id = ?", owner.ID).Update("is_bot", true).Error; err != nil {
					t.Fatal(err)
				}
			case "banned":
				if err := db.Model(&models.User{}).Where("id = ?", owner.ID).Update("user_role", constants.UserRoleBanned).Error; err != nil {
					t.Fatal(err)
				}
			case "payer-banned":
				if err := db.Model(&models.User{}).Where("id = ?", payer.ID).Update("user_role", constants.UserRoleBanned).Error; err != nil {
					t.Fatal(err)
				}
			case "domain":
				if err := db.Model(&models.User{}).Where("id = ?", owner.ID).Update("domain", "another-domain").Error; err != nil {
					t.Fatal(err)
				}
			case "expired":
				if err := db.Model(&models.BroadcastSession{}).Where("id = ?", session.ID).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
					t.Fatal(err)
				}
				expected = ports.ErrBroadcastSessionUnavailable
			case "ended":
				if _, err := repo.ChangeBroadcastSession(ctx, owner.ID, session.ID, true); err != nil {
					t.Fatal(err)
				}
				expected = ports.ErrBroadcastSessionUnavailable
			case "private":
				if err := db.Model(&models.User{}).Where("id = ?", owner.ID).Update("privacy_level", constants.PrivacyPrivate).Error; err != nil {
					t.Fatal(err)
				}
			case "overflow":
				ownerBalance = "99999999999999999999.999999999999999999"
				if err := db.Model(&models.User{}).Where("id = ?", owner.ID).Update("balance", decimal.RequireFromString(ownerBalance)).Error; err != nil {
					t.Fatal(err)
				}
				expected = domainwallet.ErrAmountOutOfRange
			case "insufficient":
				amount = decimal.NewFromInt(11)
				expected = domainwallet.ErrInsufficientFunds
			case "self":
				payer = owner
				expected = domainwallet.ErrSelfTransfer
			}
			_, err := repo.SendBroadcastGift(ctx, payer.ID, session.ID, amount, mustTipIdempotencyKey(t, "gift-policy-check"))
			if !errors.Is(err, expected) {
				t.Fatalf("gift %v, want %v", err, expected)
			}
			assertStoredBalance(t, db, owner.ID, ownerBalance)
			if scenario != "self" {
				assertStoredBalance(t, db, payer.ID, "10")
			}
			var count int64
			if err := db.Model(&models.BroadcastGift{}).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("gift count %d, %v", count, err)
			}
		})
	}
}
func TestBroadcastSessionOwnershipLeaseAndRestartIntegration(t *testing.T) {
	db, repo, viewer, owner, session := prepareBroadcastGift(t)
	ctx := context.Background()
	same, err := repo.StartBroadcastSession(ctx, owner.ID)
	if err != nil || same.ID != session.ID {
		t.Fatalf("repeated start %+v, %v", same, err)
	}
	for _, end := range []bool{false, true} {
		if _, err := repo.ChangeBroadcastSession(ctx, viewer.ID, session.ID, end); !errors.Is(err, ports.ErrBroadcastSessionForbidden) {
			t.Fatalf("foreign mutation %v", err)
		}
	}
	if _, err := repo.FindBroadcastSession(ctx, viewer.ID, owner.PublicID); err != nil {
		t.Fatal(err)
	}
	heartbeat, err := repo.ChangeBroadcastSession(ctx, owner.ID, session.ID, false)
	if err != nil || heartbeat.ExpiresAt.Before(session.ExpiresAt) {
		t.Fatalf("heartbeat %+v, %v", heartbeat, err)
	}
	if err := db.Model(&models.BroadcastSession{}).Where("id = ?", session.ID).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ChangeBroadcastSession(ctx, owner.ID, session.ID, false); !errors.Is(err, ports.ErrBroadcastSessionUnavailable) {
		t.Fatalf("expired heartbeat %v", err)
	}
	if _, err := repo.FindBroadcastSession(ctx, viewer.ID, owner.PublicID); !errors.Is(err, ports.ErrBroadcastSessionUnavailable) {
		t.Fatalf("expired fetch %v", err)
	}
	next, err := repo.StartBroadcastSession(ctx, owner.ID)
	if err != nil || next.ID == session.ID {
		t.Fatalf("restart %+v, %v", next, err)
	}
	if _, err := repo.ChangeBroadcastSession(ctx, owner.ID, session.ID, true); err != nil {
		t.Fatal(err)
	}
	var saved models.User
	if err := db.Take(&saved, "id = ?", owner.ID).Error; err != nil || !saved.IsLive {
		t.Fatalf("stale close stopped new session: %v", err)
	}
}
func TestBroadcastGiftConcurrentRetriesTransferOnceIntegration(t *testing.T) {
	db, repo, payer, owner, session := prepareBroadcastGift(t)
	key := mustTipIdempotencyKey(t, "gift-concurrent-retry")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan ports.BroadcastGiftReceipt, 12)
	errs := make(chan error, 12)
	for range 12 {
		go func() {
			<-start
			receipt, err := repo.SendBroadcastGift(ctx, payer.ID, session.ID, decimal.NewFromInt(2), key)
			results <- receipt
			errs <- err
		}()
	}
	close(start)
	var first uuid.UUID
	for range 12 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
		receipt := <-results
		if first == uuid.Nil {
			first = receipt.ID
		}
		if receipt.ID != first || !receipt.Balance.Equal(decimal.NewFromInt(8)) {
			t.Fatalf("receipt %+v", receipt)
		}
	}
	assertStoredBalance(t, db, payer.ID, "8")
	assertStoredBalance(t, db, owner.ID, "6")
	assertGiftRecords(t, db, 1)
}
func TestBroadcastConcurrentSessionStartCreatesOneSessionIntegration(t *testing.T) {
	db, repo, _, owner, initial := prepareBroadcastGift(t)
	if _, err := repo.ChangeBroadcastSession(context.Background(), owner.ID, initial.ID, true); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	errs := make(chan error, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; _, err := repo.StartBroadcastSession(ctx, owner.ID); errs <- err }()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := db.Model(&models.BroadcastSession{}).Where("owner_id = ? AND ended_at IS NULL", owner.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("active sessions %d, %v", count, err)
	}
}
func TestConcurrentBroadcastGiftAndTipCannotOverspendIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	db, postRepo, payer, owner, target, _ := prepareTipIntegrationOnDB(t, db, datatypes.JSON([]byte("{}")))
	repo := NewBroadcastGiftRepository(db)
	session, err := repo.StartBroadcastSession(context.Background(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	errs := make(chan error, 2)
	go func() {
		<-start
		_, err := repo.SendBroadcastGift(ctx, payer.ID, session.ID, decimal.NewFromInt(7), mustTipIdempotencyKey(t, "concurrent-gift-tip"))
		errs <- err
	}()
	go func() {
		local := payer
		<-start
		_, err := postRepo.Tip(ctx, target.PublicID, &local, decimal.NewFromInt(7), mustTipIdempotencyKey(t, "concurrent-tip-gift"))
		errs <- err
	}()
	close(start)
	success := 0
	for range 2 {
		err := <-errs
		if err == nil {
			success++
		} else if !errors.Is(err, domainwallet.ErrInsufficientFunds) && err.Error() != constants.ErrInsufficientBalance.String() {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatalf("successes %d", success)
	}
	assertStoredBalance(t, db, payer.ID, "3")
	assertStoredBalance(t, db, owner.ID, "11")
}
func TestExpiredLocalBroadcastsAreHiddenFromLiveListIntegration(t *testing.T) {
	db, _, _, owner, session := prepareBroadcastGift(t)
	bot := financialUser(t, db)
	if err := db.Model(&models.User{}).Where("id = ?", bot.ID).Updates(map[string]any{"is_live": true, "is_bot": true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.BroadcastSession{}).Where("id = ?", session.ID).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	live := true
	var users []models.User
	if err := (&UserRepository{db: db}).liveUsersWithoutLocationQuery(types.Filter{IsLive: &live, Limit: 100}).Find(&users).Error; err != nil {
		t.Fatal(err)
	}
	foundBot := false
	for _, user := range users {
		if user.ID == owner.ID {
			t.Fatal("expired publisher remained live")
		}
		if user.ID == bot.ID {
			foundBot = true
		}
	}
	if !foundBot {
		t.Fatal("external imported broadcaster disappeared")
	}
}

func TestBroadcastGiftDifferentRequestsCannotOverspendIntegration(t *testing.T) {
	db, repo, payer, owner, session := prepareBroadcastGift(t)
	other := financialUser(t, db)
	second, err := repo.StartBroadcastSession(context.Background(), other.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 2)
	firstKey, secondKey := mustTipIdempotencyKey(t, "first-stream-gift"), mustTipIdempotencyKey(t, "second-stream-gift")
	go func() {
		<-start
		_, err := repo.SendBroadcastGift(ctx, payer.ID, session.ID, decimal.NewFromInt(7), firstKey)
		results <- err
	}()
	go func() {
		<-start
		_, err := repo.SendBroadcastGift(ctx, payer.ID, second.ID, decimal.NewFromInt(7), secondKey)
		results <- err
	}()
	close(start)
	success, failure := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, domainwallet.ErrInsufficientFunds) {
			failure++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || failure != 1 {
		t.Fatalf("success/failure %d/%d", success, failure)
	}
	assertStoredBalance(t, db, payer.ID, "3")
	assertGiftRecords(t, db, 1)
	var users []models.User
	if err := db.Where("id IN ?", []uuid.UUID{owner.ID, other.ID}).Find(&users).Error; err != nil {
		t.Fatal(err)
	}
	if len(users) != 2 || !users[0].Balance.Add(users[1].Balance).Equal(decimal.NewFromInt(11)) {
		t.Fatalf("recipient total %+v", users)
	}
}
func TestLocalBroadcastDiscoveryHonorsPrivacyAndBlockingIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	owner, viewer := financialUser(t, db), financialUser(t, db)
	repo := NewBroadcastGiftRepository(db)
	if _, err := repo.StartBroadcastSession(context.Background(), owner.ID); err != nil {
		t.Fatal(err)
	}
	usersRepo := NewUserRepository(db, nil, nil, NewEngagementRepository(db), nil)
	live := true
	fetch := func(viewer *models.User, want int) {
		t.Helper()
		var actor *types.Actor
		if viewer != nil {
			actor = &types.Actor{ID: viewer.ID, PublicID: viewer.PublicID, Role: string(viewer.UserRole)}
		}
		users, err := usersRepo.FetchLiveUsers(types.Filter{IsLive: &live, AuthUser: actor, Limit: 20})
		if err != nil || len(users) != want {
			t.Fatalf("live users %d, %v; want %d", len(users), err, want)
		}
	}
	fetch(nil, 1)
	if err := db.Model(&models.User{}).Where("id = ?", owner.ID).Update("privacy_level", constants.PrivacyPrivate).Error; err != nil {
		t.Fatal(err)
	}
	fetch(nil, 0)
	fetch(&viewer, 0)
	fetch(&owner, 1)
	if err := db.Model(&models.User{}).Where("id = ?", owner.ID).Update("privacy_level", constants.PrivacyPublic).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(context.Background(), owner.ID, viewer.ID, socialIntent(t, domainuser.InteractionBlock, true)); err != nil {
		t.Fatal(err)
	}
	fetch(&viewer, 0)
}
