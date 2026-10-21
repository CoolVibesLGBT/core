package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	"time"

	"github.com/google/uuid"
)

type MatchesService struct {
	userRepo    ports.UserPublicIDResolver
	matchesRepo ports.MatchesRepository
}

const matchesRepositoryTimeout = 4 * time.Second

func matchesContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, matchesRepositoryTimeout)
}

func NewMatchService(
	userRepo ports.UserPublicIDResolver,
	matchesRepo ports.MatchesRepository) *MatchesService {
	return &MatchesService{userRepo: userRepo, matchesRepo: matchesRepo}
}

func (s *MatchesService) GetUserUUIDByPublicID(ctx context.Context, publicID int64) (uuid.UUID, error) {
	requestContext, cancel := matchesContext(ctx)
	defer cancel()
	return s.userRepo.GetUserUUIDByPublicID(requestContext, publicID)
}

func (service *MatchesService) GetUnseenUsers(ctx context.Context, userId uuid.UUID, limit int, criteria ...domainuser.MatchCriteria) ([]types.PublicUserSummary, error) {
	if len(criteria) > 1 {
		return nil, domainuser.ErrInvalidMatchCriteria
	}
	if len(criteria) > 0 {
		if err := criteria[0].Validate(); err != nil {
			return nil, err
		}
	}
	requestContext, cancel := matchesContext(ctx)
	defer cancel()
	return service.matchesRepo.GetUnseenUsers(requestContext, userId, limit, criteria...)
}

func (service *MatchesService) RecordView(ctx context.Context, userID, targetID uuid.UUID, rawReaction string) (bool, error) {
	if userID == uuid.Nil || targetID == uuid.Nil {
		return false, domainuser.ErrInvalidMatchReaction
	}
	if userID == targetID {
		return false, domainuser.ErrSelfInteraction
	}
	reaction, err := domainuser.ParseMatchReaction(rawReaction)
	if err != nil {
		return false, err
	}
	if service.matchesRepo == nil {
		return false, constants.ErrMatchesRepositoryNotConfigured
	}
	requestContext, cancel := matchesContext(ctx)
	defer cancel()
	return service.matchesRepo.RecordView(requestContext, userID, targetID, reaction)
}

func (service *MatchesService) RemoveMatch(ctx context.Context, userID, targetID uuid.UUID) error {
	if userID == uuid.Nil || targetID == uuid.Nil {
		return domainuser.ErrInvalidMatchReaction
	}
	if userID == targetID {
		return domainuser.ErrSelfInteraction
	}
	requestContext, cancel := matchesContext(ctx)
	defer cancel()
	return service.matchesRepo.RemoveMatch(requestContext, userID, targetID)
}

func (m *MatchesService) GetMatchesAfter(ctx context.Context, userID uuid.UUID, cursor *types.MatchListCursor, limit int) (types.MatchListPage, error) {
	requestContext, cancel := matchesContext(ctx)
	defer cancel()
	return m.matchesRepo.GetMatchesAfter(requestContext, userID, cursor, limit)
}

func (m *MatchesService) GetLikesAfter(ctx context.Context, userID uuid.UUID, cursor *types.MatchListCursor, limit int) (types.MatchListPage, error) {
	requestContext, cancel := matchesContext(ctx)
	defer cancel()
	return m.matchesRepo.GetLikesAfter(requestContext, userID, cursor, limit)

}

func (m *MatchesService) GetPassesAfter(ctx context.Context, userID uuid.UUID, cursor *types.MatchListCursor, limit int) (types.MatchListPage, error) {
	requestContext, cancel := matchesContext(ctx)
	defer cancel()
	return m.matchesRepo.GetPassesAfter(requestContext, userID, cursor, limit)
}
