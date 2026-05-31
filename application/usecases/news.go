package usecases

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/post"
	"core/models/taxonomy"
	"core/types"
	"fmt"

	"mime/multipart"

	"github.com/google/uuid"
)

type NewsService struct {
	mediaRepo ports.MediaRepository
	userRepo  ports.UserRepository
	postRepo  ports.PostRepository
	placeRepo ports.PlaceRepository
	newsRepo  ports.NewsRepository
}

func NewNewsService(
	userRepo ports.UserRepository,
	postRepo ports.PostRepository,
	mediaRepo ports.MediaRepository,
	placeRepo ports.PlaceRepository,
	newsRepo ports.NewsRepository) *NewsService {
	return &NewsService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, placeRepo: placeRepo, newsRepo: newsRepo}
}

func (s *NewsService) ServiceName() string {
	return "NewsService"
}

func (s *NewsService) CreateNews(context context.Context, request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(context, request, files, author, string(post.PostKindNews), nil)
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

func (s *NewsService) GetNews(filters types.Filter) (types.PostsResult, error) {
	return s.newsRepo.GetNews(filters)
}

func (s *NewsService) Get(filters types.Filter) (*post.Post, error) {
	return s.newsRepo.Get(filters)
}

func (s *NewsService) IsNewsExists(filters types.Filter) (bool, error) {
	return s.newsRepo.IsNewsExists(filters)
}

func (s *NewsService) Categories(filters types.Filter) ([]*taxonomy.Pillar, error) {
	return s.newsRepo.Categories(filters)
}

func (s *NewsService) Category(filters types.Filter) (*taxonomy.Pillar, error) {
	return s.newsRepo.Category(filters)
}
