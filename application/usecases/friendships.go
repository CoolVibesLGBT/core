package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	domainuser "core/domain/user"
	"time"
)

type FriendshipView struct {
	RequestID   string                      `json:"request_id"`
	RequesterID string                      `json:"requester_id"`
	RecipientID string                      `json:"recipient_id"`
	Status      domainuser.FriendshipStatus `json:"status"`
	User        types.PublicUserSummary     `json:"user"`
	RequestedAt time.Time                   `json:"requested_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}

type FriendshipService struct{ repository ports.FriendshipRepository }

func NewFriendshipService(repository ports.FriendshipRepository) *FriendshipService {
	return &FriendshipService{repository: repository}
}

func (s *FriendshipService) Status(ctx context.Context, principal ports.PrivatePhotoPrincipal, target int64) (*FriendshipView, error) {
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return nil, err
	}
	if target <= 0 {
		return nil, ErrUserIDRequired
	}
	reader, ok := s.repository.(ports.FriendshipStatusReader)
	if !ok {
		return nil, ports.ErrNotFound
	}
	record, err := reader.FindFriendship(ctx, principal.ID, target)
	if err != nil || record == nil {
		return nil, err
	}
	view := friendshipView(*record)
	return &view, nil
}

func (s *FriendshipService) Request(ctx context.Context, principal ports.PrivatePhotoPrincipal, target int64) (FriendshipView, error) {
	ctx, cancel := matchesContext(ctx)
	defer cancel()
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return FriendshipView{}, err
	}
	if target <= 0 {
		return FriendshipView{}, ErrUserIDRequired
	}
	if target == principal.PublicID {
		return FriendshipView{}, domainuser.ErrSelfInteraction
	}
	record, err := s.repository.RequestFriendship(ctx, principal.ID, target)
	if err != nil {
		return FriendshipView{}, err
	}
	return friendshipView(record), nil
}
func (s *FriendshipService) Change(ctx context.Context, principal ports.PrivatePhotoPrincipal, request int64, next domainuser.FriendshipStatus) (FriendshipView, error) {
	ctx, cancel := matchesContext(ctx)
	defer cancel()
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return FriendshipView{}, err
	}
	if request <= 0 {
		return FriendshipView{}, ErrUserIDRequired
	}
	record, err := s.repository.ChangeFriendship(ctx, principal.ID, request, next)
	if err != nil {
		return FriendshipView{}, err
	}
	return friendshipView(record), nil
}
func (s *FriendshipService) List(ctx context.Context, principal ports.PrivatePhotoPrincipal, scope domainuser.FriendshipScope, cursor *types.MatchListCursor, limit int) ([]FriendshipView, *types.MatchListCursor, error) {
	ctx, cancel := matchesContext(ctx)
	defer cancel()
	if err := validatePrivatePhotoPrincipal(principal); err != nil {
		return nil, nil, err
	}
	records, next, err := s.repository.ListFriendships(ctx, principal.ID, scope, cursor, limit)
	if err != nil {
		return nil, nil, err
	}
	views := make([]FriendshipView, 0, len(records))
	for _, record := range records {
		views = append(views, friendshipView(record))
	}
	return views, next, nil
}
func friendshipView(record ports.FriendshipRecord) FriendshipView {
	return FriendshipView{
		RequestID:   publicIDString(record.PublicID),
		RequesterID: publicIDString(record.RequesterPublicID),
		RecipientID: publicIDString(record.RecipientPublicID),
		Status:      record.Status,
		User:        record.OtherUser,
		RequestedAt: record.RequestedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}
