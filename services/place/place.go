package services

import (
	"context"
	"coolvibes/models"
	"coolvibes/models/post"

	"coolvibes/repositories"
	"mime/multipart"
)

type PlaceService struct {
	mediaRepo *repositories.MediaRepository
	userRepo  *repositories.UserRepository
	postRepo  *repositories.PostRepository
	placeRepo *repositories.PlaceRepository
}

func NewPlaceService(
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository,
	placeRepo *repositories.PlaceRepository) *PlaceService {
	return &PlaceService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, placeRepo: placeRepo}
}

func (s *PlaceService) CreatePlace(ctx context.Context, request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(request, files, author, "post", nil)
	if err != nil {
		return nil, err
	}
	return s.placeRepo.GetPlaceByID(_post.ID)
}
