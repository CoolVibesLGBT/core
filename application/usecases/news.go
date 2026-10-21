package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainmedia "core/domain/media"
	"core/models"
	"core/models/post"
	"core/models/taxonomy"

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

func (s *NewsService) CreateNews(context context.Context, values map[string][]string, files []*domainmedia.UploadedFile, author *models.User) (*types.PublicPost, error) {
	_post, err := s.postRepo.CreateContentablePost(context, values, files, author, string(post.PostKindNews), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.FindPublicPostByIDIncludingUnpublished(_post.ID)
}

func (s *NewsService) GetPostByID(id uuid.UUID) (*types.PublicPost, error) {
	postData, err := s.postRepo.FindPublicPostByID(id)
	if err != nil {
		return nil, constants.ErrPostLookupFailed.Format(err)
	}
	return postData, nil
}

func (s *NewsService) GetNews(filters types.Filter) (types.PublicPostPage, error) {
	result, err := s.newsRepo.GetNews(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return result, nil
}

func (s *NewsService) Get(filters types.Filter) (*types.PublicPost, error) {
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
