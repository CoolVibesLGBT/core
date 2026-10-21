package ports

import (
	"context"
	"core/application/types"
	domainwallet "core/domain/wallet"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

const BroadcastSessionTTL = 2 * time.Minute

type BroadcastSessionView struct {
	ID            uuid.UUID         `json:"id"`
	OwnerPublicID types.SnowflakeID `json:"owner_id"`
	ExpiresAt     time.Time         `json:"expires_at"`
	EndedAt       *time.Time        `json:"ended_at,omitempty"`
}
type BroadcastGiftReceipt struct {
	ID                uuid.UUID         `json:"id"`
	BroadcastID       uuid.UUID         `json:"broadcast_id"`
	RecipientPublicID types.SnowflakeID `json:"recipient_id"`
	Amount            decimal.Decimal   `json:"amount"`
	Balance           decimal.Decimal   `json:"balance"`
	CreatedAt         time.Time         `json:"created_at"`
}
type BroadcastGiftRepository interface {
	StartBroadcastSession(context.Context, uuid.UUID) (BroadcastSessionView, error)
	ChangeBroadcastSession(context.Context, uuid.UUID, uuid.UUID, bool) (BroadcastSessionView, error)
	FindBroadcastSession(context.Context, uuid.UUID, int64) (BroadcastSessionView, error)
	SendBroadcastGift(context.Context, uuid.UUID, uuid.UUID, decimal.Decimal, domainwallet.IdempotencyKey) (BroadcastGiftReceipt, error)
}
