package usecases

import (
	"core/application/ports"
	"core/models/media"
	"mime/multipart"

	"github.com/google/uuid"
)

type MediaService struct {
	repo ports.MediaRepository
}

func NewMediaService(repo ports.MediaRepository) *MediaService {
	return &MediaService{repo: repo}
}

func (s *MediaService) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userId uuid.UUID, role media.MediaRole, file *multipart.FileHeader) (*media.Media, error) {
	return s.repo.AddMedia(ownerID, ownerType, userId, role, file)
}
