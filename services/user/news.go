package services

import (
	"context"
	"coolvibes/models"
	"coolvibes/models/post"
	"coolvibes/types"
	"fmt"

	"coolvibes/repositories"
	"mime/multipart"

	"github.com/google/uuid"
)

type NewsService struct {
	mediaRepo *repositories.MediaRepository
	userRepo  *repositories.UserRepository
	postRepo  *repositories.PostRepository
	placeRepo *repositories.PlaceRepository
	newsRepo  *repositories.NewsRepository
}

func NewNewsService(
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository,
	placeRepo *repositories.PlaceRepository,
	newsRepo *repositories.NewsRepository) *NewsService {
	return &NewsService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, placeRepo: placeRepo, newsRepo: newsRepo}
}

func (s *NewsService) ServiceName() string {
	return "NewsService"
}

func (s *NewsService) CreateNews(request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(request, files, author, string(post.PostKindNews), nil)
	if err != nil {
		return nil, err
	}
	return s.GetPostByID(_post.ID)
}

func (s *NewsService) GetPostByID(id uuid.UUID) (*post.Post, error) {
	postData, err := s.postRepo.GetPostByID(id)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return postData, nil
}

func (s *NewsService) GetNews(ctx context.Context, authUser *models.User, filters types.Filters) ([]*post.Post, types.Cursor, error) {
	return s.newsRepo.GetNews(ctx, authUser, filters)
}
