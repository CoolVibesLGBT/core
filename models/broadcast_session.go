package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type BroadcastSession struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uidx_active_broadcast_owner,where:ended_at IS NULL"`
	ExpiresAt time.Time `gorm:"not null;index"`
	EndedAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Gift records survive stream closure and remain the immutable replay receipt.
type BroadcastGift struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey"`
	BroadcastID       uuid.UUID       `gorm:"type:uuid;not null;index"`
	PayerID           uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:uidx_broadcast_gift_request,priority:1;check:chk_gift_parties,payer_id <> recipient_id"`
	RecipientID       uuid.UUID       `gorm:"type:uuid;not null;index"`
	RecipientPublicID int64           `gorm:"not null"`
	IdempotencyKey    string          `gorm:"size:128;not null;uniqueIndex:uidx_broadcast_gift_request,priority:2"`
	Amount            decimal.Decimal `gorm:"type:numeric(38,18);not null;check:chk_gift_amount,amount > 0"`
	ResultingBalance  decimal.Decimal `gorm:"type:numeric(38,18);not null;check:chk_gift_balance,resulting_balance >= 0"`
	CreatedAt         time.Time
}
