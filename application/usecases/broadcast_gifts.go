package usecases

import (
	"context"
	"core/application/ports"
	domainwallet "core/domain/wallet"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BroadcastGiftService struct{ repo ports.BroadcastGiftRepository }

func NewBroadcastGiftService(repo ports.BroadcastGiftRepository) *BroadcastGiftService {
	return &BroadcastGiftService{repo: repo}
}
func (s *BroadcastGiftService) Start(ctx context.Context, actor uuid.UUID) (ports.BroadcastSessionView, error) {
	if actor == uuid.Nil {
		return ports.BroadcastSessionView{}, ports.ErrBroadcastUnauthorized
	}
	return s.repo.StartBroadcastSession(ctx, actor)
}
func (s *BroadcastGiftService) Change(ctx context.Context, actor, id uuid.UUID, end bool) (ports.BroadcastSessionView, error) {
	if actor == uuid.Nil {
		return ports.BroadcastSessionView{}, ports.ErrBroadcastUnauthorized
	}
	if id == uuid.Nil {
		return ports.BroadcastSessionView{}, ports.ErrBroadcastInvalidInput
	}
	return s.repo.ChangeBroadcastSession(ctx, actor, id, end)
}
func (s *BroadcastGiftService) Find(ctx context.Context, actor uuid.UUID, owner int64) (ports.BroadcastSessionView, error) {
	if actor == uuid.Nil {
		return ports.BroadcastSessionView{}, ports.ErrBroadcastUnauthorized
	}
	if owner <= 0 {
		return ports.BroadcastSessionView{}, ports.ErrBroadcastInvalidInput
	}
	return s.repo.FindBroadcastSession(ctx, actor, owner)
}
func (s *BroadcastGiftService) Gift(ctx context.Context, actor, id uuid.UUID, amount decimal.Decimal, rawKey string) (ports.BroadcastGiftReceipt, error) {
	if actor == uuid.Nil {
		return ports.BroadcastGiftReceipt{}, ports.ErrBroadcastUnauthorized
	}
	if id == uuid.Nil {
		return ports.BroadcastGiftReceipt{}, ports.ErrBroadcastInvalidInput
	}
	if err := domainwallet.ValidateTipAmount(amount); err != nil {
		return ports.BroadcastGiftReceipt{}, err
	}
	key, err := domainwallet.NewIdempotencyKey(rawKey)
	if err != nil {
		return ports.BroadcastGiftReceipt{}, err
	}
	return s.repo.SendBroadcastGift(ctx, actor, id, amount, key)
}
