package services

import (
	"coolvibes/models"
	"coolvibes/models/post"
	"coolvibes/types"
	"fmt"

	"coolvibes/repositories"
	"mime/multipart"

	"github.com/google/uuid"
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

func (s *PlaceService) ServiceName() string {
	return "PlaceService"
}

func (s *PlaceService) CreatePlace(request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(request, files, author, string(post.PostKindPlace), nil)
	if err != nil {
		return nil, err
	}
	return s.GetPostByID(_post.ID)
}

func (s *PlaceService) GetPostByID(id uuid.UUID) (*post.Post, error) {
	postData, err := s.postRepo.GetPostByID(id)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return postData, nil
}

func (s *PlaceService) GetNearByPlaces(filters types.Filter) ([]*post.Post, types.Cursor, error) {
	return s.placeRepo.GetNearByPlaces(filters)
}
