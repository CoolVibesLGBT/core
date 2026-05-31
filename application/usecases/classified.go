package usecases

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/post"
	"core/types"
	"fmt"

	"mime/multipart"
)

type ClassifiedService struct {
	mediaRepo    ports.MediaRepository
	userRepo     ports.UserRepository
	postRepo     ports.PostRepository
	placeRepo    ports.PlaceRepository
	listingsRepo ports.ListingRepository
}

func NewClassifiedService(
	userRepo ports.UserRepository,
	postRepo ports.PostRepository,
	mediaRepo ports.MediaRepository,
	placeRepo ports.PlaceRepository,
	listingsRepo ports.ListingRepository) *ClassifiedService {
	return &ClassifiedService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, placeRepo: placeRepo, listingsRepo: listingsRepo}
}

func (s *ClassifiedService) ServiceName() string {
	return "ClassifiedService"
}

func (s *ClassifiedService) CreateClassified(context context.Context, request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(context, request, files, author, string(post.PostKindClassified), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.GetPostByID(_post.ID)
}

func (s *ClassifiedService) GetClassified(filters types.Filter) (*post.Post, error) {
	postData, err := s.listingsRepo.GetClassified(filters)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return postData, nil
}

func (s *ClassifiedService) GetJobOffers(filters types.Filter) (types.PostsResult, error) {
	return s.listingsRepo.GetJobOffers(filters)
}

func (s *ClassifiedService) GetJobSearches(filters types.Filter) (types.PostsResult, error) {
	return s.listingsRepo.GetJobSearches(filters)
}
