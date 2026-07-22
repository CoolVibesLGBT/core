package usecases

import (
	"context"
	legacyviews "core/application/legacyviews"
	"core/application/ports"
	"core/application/types"
	"core/models"
	"core/models/post"
	"core/models/taxonomy"
	"fmt"
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

func (s *PlaceService) CreatePlace(context context.Context, form ports.FormData, author *models.User) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(context, form, author, string(post.PostKindPlace), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.GetPostByIDIncludingUnpublished(_post.ID)
}

func (s *PlaceService) GetPostByID(id uuid.UUID) (*post.Post, error) {
	postData, err := s.postRepo.GetPostByID(id)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
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
	return legacyviews.ProjectPublicPostPointers(places), cursor, nil
}

func (s *PlaceService) GetPlacesCategories(filters types.Filter) ([]taxonomy.Pillar, error) {
	return s.placeRepo.GetPlacesCategories(filters)
}
