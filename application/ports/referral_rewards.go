package ports

import (
	"context"
	"core/application/types"
	domainuser "core/domain/user"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type ReferralRewardView struct {
	ID               uuid.UUID                       `json:"id"`
	ReferrerPublicID types.SnowflakeID               `json:"referrer_id"`
	ReferredPublicID types.SnowflakeID               `json:"referred_id"`
	Amount           decimal.Decimal                 `json:"amount"`
	Status           domainuser.ReferralRewardStatus `json:"status"`
	CreatedAt        time.Time                       `json:"created_at"`
	ReviewedAt       *time.Time                      `json:"reviewed_at,omitempty"`
}
type ReferralRewardRepository interface {
	ListReferralRewards(context.Context, uuid.UUID, bool, domainuser.ReferralRewardStatus, *types.MatchListCursor, int) ([]ReferralRewardView, *types.MatchListCursor, error)
	ReviewReferralReward(context.Context, uuid.UUID, uuid.UUID, domainuser.ReferralReview) (ReferralRewardView, error)
}
