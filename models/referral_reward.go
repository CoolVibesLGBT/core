package models

import (
	domainuser "core/domain/user"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

// Pending rewards are promises, not spendable balance. Review evidence stays
// internal and the verified identity is stored only as a digest.
type ReferralReward struct {
	ID                   uuid.UUID                       `gorm:"type:uuid;primaryKey"`
	ReferrerID           uuid.UUID                       `gorm:"type:uuid;not null;index;check:chk_referral_distinct,referrer_id <> referred_id"`
	ReferredID           uuid.UUID                       `gorm:"type:uuid;not null;uniqueIndex"`
	Amount               decimal.Decimal                 `gorm:"type:numeric(38,18);not null;check:chk_referral_positive,amount > 0"`
	Status               domainuser.ReferralRewardStatus `gorm:"type:varchar(16);not null;index;check:chk_referral_status,status IN ('pending','paid','rejected')"`
	VerifiedIdentityHash *string                         `gorm:"size:64;uniqueIndex"`
	ReviewedByID         *uuid.UUID                      `gorm:"type:uuid"`
	ReviewNote           string                          `gorm:"type:text"`
	ReviewedAt           *time.Time
	CreatedAt            time.Time `gorm:"index"`
	UpdatedAt            time.Time
}
