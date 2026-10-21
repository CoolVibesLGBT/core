package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainmedia "core/domain/media"
	"core/models"
	"core/models/post"
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

func (s *ClassifiedService) CreateClassified(context context.Context, values map[string][]string, files []*domainmedia.UploadedFile, author *models.User) (*types.PublicPost, error) {
	_post, err := s.postRepo.CreateContentablePost(context, values, files, author, string(post.PostKindClassified), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.FindPublicPostByIDIncludingUnpublished(_post.ID)
}

func (s *ClassifiedService) GetClassified(filters types.Filter) (*types.PublicPost, error) {
	postData, err := s.listingsRepo.GetClassified(filters)
	if err != nil {
		return nil, constants.ErrPostLookupFailed.Format(err)
	}
	return postData, nil
}

func (s *ClassifiedService) GetJobOffers(filters types.Filter) (types.PublicPostPage, error) {
	result, err := s.listingsRepo.GetJobOffers(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return result, nil
}

func (s *ClassifiedService) GetJobSearches(filters types.Filter) (types.PublicPostPage, error) {
	result, err := s.listingsRepo.GetJobSearches(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return result, nil
}
