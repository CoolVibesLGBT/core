package repositories

import (
	"context"
	"core/application/ports"
	"core/application/types"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"core/models"
	"core/models/notifications"
	"errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type BroadcastGiftRepository struct{ db *gorm.DB }

func NewBroadcastGiftRepository(db *gorm.DB) *BroadcastGiftRepository {
	return &BroadcastGiftRepository{db: db}
}
func livePrincipal(tx *gorm.DB, actor uuid.UUID) (models.User, error) {
	var user models.User
	if err := tx.Take(&user, "id = ?", actor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, domainwallet.ErrTransferUnavailable
		}
		return user, err
	}
	if !domainuser.CanAuthenticate(domainuser.AccountRole(user.UserRole), user.IsBot) {
		return user, domainwallet.ErrTransferUnavailable
	}
	return user, nil
}
func broadcastSessionView(session models.BroadcastSession, owner int64) ports.BroadcastSessionView {
	return ports.BroadcastSessionView{ID: session.ID, OwnerPublicID: types.SnowflakeID(owner), ExpiresAt: session.ExpiresAt, EndedAt: session.EndedAt}
}
func (r *BroadcastGiftRepository) StartBroadcastSession(ctx context.Context, actor uuid.UUID) (ports.BroadcastSessionView, error) {
	var result ports.BroadcastSessionView
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Name() == "postgres" {
			if err := lockViewAggregate(tx, "broadcast-owner:"+actor.String()).Error; err != nil {
				return err
			}
		}
		user, err := livePrincipal(tx, actor)
		if err != nil {
			return err
		}
		var session models.BroadcastSession
		now := time.Now().UTC()
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("owner_id = ? AND ended_at IS NULL", actor).Take(&session).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && session.ExpiresAt.After(now) {
			result = broadcastSessionView(session, user.PublicID)
			return nil
		}
		if err == nil {
			if err := tx.Model(&session).Update("ended_at", now).Error; err != nil {
				return err
			}
		}
		session = models.BroadcastSession{ID: uuid.New(), OwnerID: actor, ExpiresAt: now.Add(ports.BroadcastSessionTTL)}
		if err := tx.Create(&session).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.User{}).Where("id = ?", actor).Update("is_live", true).Error; err != nil {
			return err
		}
		result = broadcastSessionView(session, user.PublicID)
		return nil
	})
	return result, err
}
func (r *BroadcastGiftRepository) ChangeBroadcastSession(ctx context.Context, actor, id uuid.UUID, end bool) (ports.BroadcastSessionView, error) {
	var result ports.BroadcastSessionView
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Name() == "postgres" {
			if err := lockViewAggregate(tx, "broadcast-owner:"+actor.String()).Error; err != nil {
				return err
			}
		}
		user, err := livePrincipal(tx, actor)
		if err != nil {
			return err
		}
		var session models.BroadcastSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&session, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrBroadcastSessionUnavailable
			}
			return err
		}
		if session.OwnerID != actor {
			return ports.ErrBroadcastSessionForbidden
		}
		now := time.Now().UTC()
		if end {
			if session.EndedAt == nil {
				session.EndedAt = &now
				if err := tx.Model(&session).Update("ended_at", now).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.User{}).Where("id = ?", actor).Update("is_live", false).Error; err != nil {
					return err
				}
			}
		} else {
			if session.EndedAt != nil || !session.ExpiresAt.After(now) {
				return ports.ErrBroadcastSessionUnavailable
			}
			session.ExpiresAt = now.Add(ports.BroadcastSessionTTL)
			if err := tx.Model(&session).Update("expires_at", session.ExpiresAt).Error; err != nil {
				return err
			}
		}
		result = broadcastSessionView(session, user.PublicID)
		return nil
	})
	return result, err
}
func (r *BroadcastGiftRepository) FindBroadcastSession(ctx context.Context, actor uuid.UUID, owner int64) (ports.BroadcastSessionView, error) {
	tx := r.db.WithContext(ctx)
	if _, err := livePrincipal(tx, actor); err != nil {
		return ports.BroadcastSessionView{}, err
	}
	var target models.User
	if err := profileViewerScope(tx.Model(&models.User{}).Table("users"), actor).Where("users.public_id = ?", owner).Take(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.BroadcastSessionView{}, ports.ErrBroadcastSessionUnavailable
		}
		return ports.BroadcastSessionView{}, err
	}
	if !domainuser.CanAuthenticate(domainuser.AccountRole(target.UserRole), target.IsBot) {
		return ports.BroadcastSessionView{}, ports.ErrBroadcastSessionUnavailable
	}
	var session models.BroadcastSession
	if err := tx.Where("owner_id = ? AND ended_at IS NULL AND expires_at > ?", target.ID, time.Now().UTC()).Take(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.BroadcastSessionView{}, ports.ErrBroadcastSessionUnavailable
		}
		return ports.BroadcastSessionView{}, err
	}
	return broadcastSessionView(session, target.PublicID), nil
}
func giftReceipt(gift models.BroadcastGift) ports.BroadcastGiftReceipt {
	return ports.BroadcastGiftReceipt{ID: gift.ID, BroadcastID: gift.BroadcastID, RecipientPublicID: types.SnowflakeID(gift.RecipientPublicID), Amount: gift.Amount, Balance: gift.ResultingBalance, CreatedAt: gift.CreatedAt}
}
func loadGiftReplay(tx *gorm.DB, payer, broadcast uuid.UUID, amount decimal.Decimal, key domainwallet.IdempotencyKey) (models.BroadcastGift, bool, error) {
	var gift models.BroadcastGift
	err := tx.Where("payer_id = ? AND idempotency_key = ?", payer, key.String()).Take(&gift).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return gift, false, nil
	}
	if err != nil {
		return gift, false, err
	}
	if gift.BroadcastID != broadcast || !gift.Amount.Equal(amount) {
		return gift, false, domainwallet.ErrIdempotencyConflict
	}
	return gift, true, nil
}
func (r *BroadcastGiftRepository) SendBroadcastGift(ctx context.Context, payerID, broadcastID uuid.UUID, amount decimal.Decimal, key domainwallet.IdempotencyKey) (ports.BroadcastGiftReceipt, error) {
	if payerID == uuid.Nil || broadcastID == uuid.Nil {
		return ports.BroadcastGiftReceipt{}, ports.ErrBroadcastInvalidInput
	}
	if err := domainwallet.ValidateTipAmount(amount); err != nil {
		return ports.BroadcastGiftReceipt{}, err
	}
	key, err := domainwallet.NewIdempotencyKey(key.String())
	if err != nil {
		return ports.BroadcastGiftReceipt{}, err
	}
	var gift models.BroadcastGift
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if tx.Name() == "postgres" {
			if err := lockViewAggregate(tx, "broadcast-gift:"+payerID.String()+":"+key.String()).Error; err != nil {
				return err
			}
		}
		existing, found, err := loadGiftReplay(tx, payerID, broadcastID, amount, key)
		if err != nil {
			return err
		}
		if found {
			gift = existing
			return nil
		}
		var session models.BroadcastSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&session, "id = ?", broadcastID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrBroadcastSessionUnavailable
			}
			return err
		}
		if session.OwnerID == payerID {
			return domainwallet.ErrSelfTransfer
		}
		if err := lockPrivatePhotoAccessPair(tx, payerID, session.OwnerID); err != nil {
			return err
		}
		var users []models.User
		if err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Where("id IN ?", []uuid.UUID{payerID, session.OwnerID}).Order("id ASC").Find(&users).Error; err != nil {
			return err
		}
		if len(users) != 2 {
			return domainwallet.ErrTransferUnavailable
		}
		var payer, payee models.User
		for _, user := range users {
			if !domainuser.CanAuthenticate(domainuser.AccountRole(user.UserRole), user.IsBot) {
				return domainwallet.ErrTransferUnavailable
			}
			if user.ID == payerID {
				payer = user
			} else {
				payee = user
			}
		}
		if payer.Domain != payee.Domain {
			return domainwallet.ErrTransferUnavailable
		}
		if err := ensureSocialInteractionAllowed(tx, payer.ID, payee.ID, domainuser.InteractionFollow); err != nil {
			if errors.Is(err, domainuser.ErrInteractionBlocked) {
				return domainwallet.ErrTransferBlocked
			}
			return err
		}
		now := time.Now().UTC()
		if session.EndedAt != nil || !session.ExpiresAt.After(now) {
			return ports.ErrBroadcastSessionUnavailable
		}
		var visible int64
		if err := profileViewerScope(tx.Model(&models.User{}).Table("users"), payer.ID).Where("users.id = ?", payee.ID).Count(&visible).Error; err != nil {
			return err
		}
		if visible != 1 {
			return domainwallet.ErrTransferUnavailable
		}
		transfer, err := domainwallet.NewTransfer(payer.ID, payee.ID, amount, domainwallet.MinimumTipAmount(), payer.Balance)
		if err != nil {
			return err
		}
		if err := domainwallet.ValidateMoneyRepresentation(payee.Balance); err != nil {
			return err
		}
		from, to := transfer.Apply(payer.Balance, payee.Balance)
		if err := domainwallet.ValidateMoneyRepresentation(from); err != nil {
			return err
		}
		if err := domainwallet.ValidateMoneyRepresentation(to); err != nil {
			return err
		}
		for _, update := range []struct {
			id      uuid.UUID
			balance decimal.Decimal
		}{{payer.ID, from}, {payee.ID, to}} {
			result := tx.Model(&models.User{}).Where("id = ?", update.id).Update("balance", update.balance)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return domainwallet.ErrTransferUnavailable
			}
		}
		gift = models.BroadcastGift{ID: uuid.New(), BroadcastID: session.ID, PayerID: payer.ID, RecipientID: payee.ID, RecipientPublicID: payee.PublicID, IdempotencyKey: key.String(), Amount: amount, ResultingBalance: from, CreatedAt: now}
		if err := tx.Create(&gift).Error; err != nil {
			return err
		}
		if err := addTipInTransaction(tx, payer.ID, payee.ID, amount, payee.ID, models.EngagementContentableTypeUser, models.EngagementKindGift, withAmountEngagementDedupeKey("broadcast-gift:"+gift.ID.String())); err != nil {
			return err
		}
		_, err = persistQueuedNotification(tx, payer.ID, payee.ID, notifications.NotificationTypeBroadcastGift, "New live gift", "You received a gift during your broadcast.", notifications.NotificationPayload{Data: map[string]string{"amount": amount.String(), "broadcast_id": session.ID.String(), "gift_id": gift.ID.String()}})
		return err
	})
	return giftReceipt(gift), err
}
