package user

import (
	domainwallet "core/domain/wallet"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrReferralPartiesRequired = errors.New("referral parties are required")
	ErrSelfReferral            = errors.New("users cannot refer themselves")
	ErrInvalidReferralReward   = errors.New("referral reward must be positive")
)

// Referral is the validated, idempotent reward instruction for one referred
// user and one referrer. Persistence adapters use DedupeKey as a stable
// at-most-once identity across retries and concurrent requests.
type Referral struct {
	referrerID   uuid.UUID
	referredID   uuid.UUID
	rewardAmount decimal.Decimal
}

func NewReferral(referrerID, referredID uuid.UUID, rewardAmount decimal.Decimal) (Referral, error) {
	if referrerID == uuid.Nil || referredID == uuid.Nil {
		return Referral{}, ErrReferralPartiesRequired
	}
	if referrerID == referredID {
		return Referral{}, ErrSelfReferral
	}
	if err := domainwallet.ValidateMoneyRepresentation(rewardAmount); err != nil {
		return Referral{}, fmt.Errorf("%w: %v", ErrInvalidReferralReward, err)
	}
	if rewardAmount.Coefficient().Sign() <= 0 {
		return Referral{}, ErrInvalidReferralReward
	}
	return Referral{
		referrerID:   referrerID,
		referredID:   referredID,
		rewardAmount: rewardAmount,
	}, nil
}

func (r Referral) ReferrerID() uuid.UUID         { return r.referrerID }
func (r Referral) ReferredID() uuid.UUID         { return r.referredID }
func (r Referral) RewardAmount() decimal.Decimal { return r.rewardAmount }

func (r Referral) DedupeKey() string {
	return fmt.Sprintf("user-referral:%s:%s", r.referredID, r.referrerID)
}

func (r Referral) Credit(balance decimal.Decimal) (decimal.Decimal, error) {
	if err := domainwallet.ValidateMoneyRepresentation(balance); err != nil {
		return decimal.Zero, err
	}
	credited := balance.Add(r.rewardAmount)
	if err := domainwallet.ValidateMoneyRepresentation(credited); err != nil {
		return decimal.Zero, err
	}
	return credited, nil
}
