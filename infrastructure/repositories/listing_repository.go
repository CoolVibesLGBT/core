package repositories

import (
	"core/application/types"
	"core/helpers"
	"core/models/post"

	"gorm.io/gorm"
)

type ListingRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	mediaRepo        *MediaRepository
	userRepo         *UserRepository
	notificationRepo *NotificationRepository
	postRepo         *PostRepository
}

func NewListingRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository, notificationRepo *NotificationRepository, postRepo *PostRepository) *ListingRepository {
	return &ListingRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo, userRepo: userRepo, notificationRepo: notificationRepo, postRepo: postRepo}
}

func (r *ListingRepository) Create(listing *post.Post) error {
	return r.postRepo.CreatePost(listing)
}

func (r *ListingRepository) GetJobOffers(filters types.Filter) (types.PublicPostPage, error) {
	filters.PostKind = post.PostKindJobOffer
	return r.postRepo.FetchPublicPostsByKind(filters)
}

func (r *ListingRepository) GetJobSearches(filters types.Filter) (types.PublicPostPage, error) {
	filters.PostKind = post.PostKindJobSearch
	return r.postRepo.FetchPublicPostsByKind(filters)
}

func (r *ListingRepository) GetClassified(filters types.Filter) (*types.PublicPost, error) {
	return r.postRepo.FindPublicPostByPublicID(filters.PostID)
}
