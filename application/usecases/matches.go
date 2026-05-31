package usecases

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/types"
	"time"

	"github.com/google/uuid"
)

type MatchesService struct {
	mediaRepo   ports.MediaRepository
	userRepo    ports.UserRepository
	postRepo    ports.PostRepository
	matchesRepo ports.MatchesRepository
}

func NewMatchService(
	userRepo ports.UserRepository,
	postRepo ports.PostRepository,
	mediaRepo ports.MediaRepository,
	matchesRepo ports.MatchesRepository) *MatchesService {
	return &MatchesService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, matchesRepo: matchesRepo}
}

func (s *MatchesService) GetUserUUIDByPublicID(publicID int64) (uuid.UUID, error) {
	return s.userRepo.GetUserUUIDByPublicID(publicID)
}

func (service *MatchesService) GetUnseenUsers(ctx context.Context, userId uuid.UUID, limit int) ([]models.User, error) {
	return service.matchesRepo.GetUnseenUsers(ctx, userId, limit)
}

func (service *MatchesService) RecordView(ctx context.Context, userId, targetId uuid.UUID, reaction types.ReactionType) (bool, error) {
	return service.matchesRepo.RecordView(ctx, userId, targetId, reaction)
}

func (m *MatchesService) GetMatchesAfter(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]models.User, error) {
	return m.matchesRepo.GetMatchesAfter(ctx, userID, cursor, limit)
}

func (m *MatchesService) GetLikesAfter(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]models.User, error) {
	return m.matchesRepo.GetLikesAfter(ctx, userID, cursor, limit)

}

func (m *MatchesService) GetPassesAfter(ctx context.Context, userID uuid.UUID, cursor *time.Time, limit int) ([]models.User, error) {
	return m.matchesRepo.GetPassesAfter(ctx, userID, cursor, limit)
}
