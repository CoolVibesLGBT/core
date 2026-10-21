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
	"time"

	"github.com/google/uuid"
)

type PlaceService struct {
	mediaRepo ports.MediaRepository
	userRepo  ports.UserRepository
	postRepo  ports.PostRepository
	placeRepo ports.PlaceRepository
}

func NewPlaceService(
	userRepo ports.UserRepository,
	postRepo ports.PostRepository,
	mediaRepo ports.MediaRepository,
	placeRepo ports.PlaceRepository) *PlaceService {
	return &PlaceService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, placeRepo: placeRepo}
}

func (s *PlaceService) ServiceName() string {
	return "PlaceService"
}

func (s *PlaceService) CreatePlace(context context.Context, values map[string][]string, files []*domainmedia.UploadedFile, author *models.User) (*types.PublicPost, error) {
	_post, err := s.postRepo.CreateContentablePost(context, values, files, author, string(post.PostKindPlace), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.FindPublicPostByIDIncludingUnpublished(_post.ID)
}

func (s *PlaceService) GetPostByID(id uuid.UUID) (*types.PublicPost, error) {
	postData, err := s.postRepo.FindPublicPostByID(id)
	if err != nil {
		return nil, constants.ErrPostLookupFailed.Format(err)
	}
	return postData, nil
}

func (s *PlaceService) GetNearByPlaces(filters types.Filter) ([]types.PublicPost, types.Cursor, error) {
	ctx := filters.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	filters.Context = ctx

	places, cursor, err := s.placeRepo.GetNearByPlaces(filters)
	if err != nil && ctx.Err() != nil {
		return nil, types.Cursor{}, ctx.Err()
	}
	if err != nil {
		return nil, types.Cursor{}, err
	}
	return places, cursor, nil
}

func (s *PlaceService) GetPlacesCategories(filters types.Filter) ([]taxonomy.Pillar, error) {
	return s.placeRepo.GetPlacesCategories(filters)
}
