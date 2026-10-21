package repositories

import (
	"context"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"core/models/notifications"
	"errors"
	"sync"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var fallbackReferralTransactionLock sync.Mutex

// Registration and its pending reward commit together. A pending promise
// cannot be spent until an independent administrator approves it.
func (r *UserRepository) CreateRegisteredUser(ctx context.Context, user *models.User, referral *domainuser.Referral) error {
	if user == nil || (referral != nil && referral.ReferredID() != user.ID) {
		return domainuser.ErrReferralPartiesRequired
	}
	if r.db.Name() != "postgres" {
		fallbackReferralTransactionLock.Lock()
		defer fallbackReferralTransactionLock.Unlock()
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(user).Error; err != nil {
			return err
		}
		if referral == nil {
			return nil
		}
		_, err := applyReferralInTransaction(tx, *referral)
		return err
	})
}

func (r *UserRepository) AddReferral(ctx context.Context, referrerID, referredUserID uuid.UUID, rewardAmount decimal.Decimal) (*decimal.Decimal, error) {
	referral, err := domainuser.NewReferral(referrerID, referredUserID, rewardAmount)
	if err != nil {
		return nil, err
	}
	if r == nil || r.db == nil {
		return nil, constants.ErrReferralRepositoryNotConfigured
	}
	if r.db.Name() != "postgres" {
		fallbackReferralTransactionLock.Lock()
		defer fallbackReferralTransactionLock.Unlock()
	}
	var balance decimal.Decimal
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		balance, err = applyReferralInTransaction(tx, referral)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &balance, nil
}

func applyReferralInTransaction(tx *gorm.DB, referral domainuser.Referral) (decimal.Decimal, error) {
	// The identity belongs to the referred user, regardless of referrer. Check
	// legacy pair-based ledger keys as well, so upgrading cannot remint a reward.
	if tx.Name() == "postgres" {
		if err := lockViewAggregate(tx, "referral:"+referral.DedupeKey()).Error; err != nil {
			return decimal.Zero, err
		}
	}
	referrer, err := lockReferralUsers(tx, referral)
	if err != nil {
		return decimal.Zero, err
	}
	var previous []models.EngagementDetail
	if err := tx.Where("engager_id = ? AND kind = ?", referral.ReferredID(), models.EngagementKindReferral).Find(&previous).Error; err != nil {
		return decimal.Zero, err
	}
	for _, detail := range previous {
		if detail.EngageeID != referral.ReferrerID() {
			return decimal.Zero, domainuser.ErrReferralAlreadyAssigned
		}
	}
	if len(previous) > 0 {
		return referrer.Balance, nil
	}
	var pending models.ReferralReward
	err = tx.Where("referred_id = ?", referral.ReferredID()).Take(&pending).Error
	if err == nil {
		if pending.ReferrerID != referral.ReferrerID() {
			return decimal.Zero, domainuser.ErrReferralAlreadyAssigned
		}
		return referrer.Balance, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return decimal.Zero, err
	}
	pending = models.ReferralReward{ID: uuid.New(), ReferrerID: referral.ReferrerID(), ReferredID: referral.ReferredID(), Amount: referral.RewardAmount(), Status: domainuser.ReferralRewardPending}
	if err := tx.Create(&pending).Error; err != nil {
		return decimal.Zero, err
	}
	return referrer.Balance, nil
}

func creditApprovedReferralInTransaction(tx *gorm.DB, referral domainuser.Referral, referrer *models.User) error {
	balance, err := referral.Credit(referrer.Balance)
	if err != nil {
		return err
	}
	if err := addTipInTransaction(tx, referral.ReferredID(), referral.ReferrerID(), referral.RewardAmount(), referral.ReferrerID(), models.EngagementContentableTypeUser, models.EngagementKindReferral, withAmountEngagementDedupeKey(referral.DedupeKey())); err != nil {
		return err
	}
	update := tx.Model(&models.User{}).Where("id = ?", referral.ReferrerID()).Update("balance", balance)
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected != 1 {
		return errReferralUserNotFound
	}
	payload := notifications.NotificationPayload{Data: map[string]string{"amount": referral.RewardAmount().String()}}
	_, err = persistQueuedNotification(tx, referral.ReferredID(), referral.ReferrerID(), notifications.NotificationTypeReferral, "New Referral Reward", "Your referral reward was approved and credited.", payload)
	return err
}

func lockReferralUsers(tx *gorm.DB, referral domainuser.Referral) (*models.User, error) {
	var users []models.User
	if err := referralUsersForBalanceUpdateQuery(tx, referral).Find(&users).Error; err != nil {
		return nil, err
	}
	if len(users) != 2 {
		return nil, errReferralUserNotFound
	}
	for _, user := range users {
		if !domainuser.CanAuthenticate(domainuser.AccountRole(user.UserRole), user.IsBot) {
			return nil, domainuser.ErrReferralUnavailable
		}
	}
	if users[0].Domain != users[1].Domain {
		return nil, domainuser.ErrReferralUnavailable
	}
	for i := range users {
		if users[i].ID == referral.ReferrerID() {
			return &users[i], nil
		}
	}
	return nil, errReferralUserNotFound
}

// NO KEY UPDATE is compatible with foreign-key checks from social mutations.
func referralUsersForBalanceUpdateQuery(tx *gorm.DB, referral domainuser.Referral) *gorm.DB {
	return tx.Select("id", "balance", "domain", "user_role", "is_bot").
		Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).
		Where("id IN ?", []uuid.UUID{referral.ReferrerID(), referral.ReferredID()}).Order("id ASC")
}
