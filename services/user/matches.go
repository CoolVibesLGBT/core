package services

import (
	"coolvibes/models"
	"coolvibes/repositories"
	"coolvibes/types"
	"time"

	"github.com/google/uuid"
)

type MatchesService struct {
	mediaRepo   *repositories.MediaRepository
	userRepo    *repositories.UserRepository
	postRepo    *repositories.PostRepository
	matchesRepo *repositories.MatchesRepository
}

func NewMatchService(
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository,
	matchesRepo *repositories.MatchesRepository) *MatchesService {
	return &MatchesService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, matchesRepo: matchesRepo}
}

func (s *MatchesService) UserRepo() *repositories.UserRepository {
	return s.userRepo
}

func (service *MatchesService) GetUnseenUsers(userId uuid.UUID, limit int) ([]models.User, error) {
	return service.matchesRepo.GetUnseenUsers(userId, limit)
}

func (service *MatchesService) RecordView(userId, targetId uuid.UUID, reaction types.ReactionType) (bool, error) {
	return service.matchesRepo.RecordView(userId, targetId, reaction)
}

func (m *MatchesService) GetMatchesAfter(userID uuid.UUID, cursor *time.Time, limit int) ([]models.User, error) {
	return m.matchesRepo.GetMatchesAfter(userID, cursor, limit)
}

func (m *MatchesService) GetLikesAfter(userID uuid.UUID, cursor *time.Time, limit int) ([]models.User, error) {
	return m.matchesRepo.GetLikesAfter(userID, cursor, limit)

}

func (m *MatchesService) GetPassesAfter(userID uuid.UUID, cursor *time.Time, limit int) ([]models.User, error) {
	return m.matchesRepo.GetPassesAfter(userID, cursor, limit)
}
