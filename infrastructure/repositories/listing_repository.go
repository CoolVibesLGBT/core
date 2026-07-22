package repositories

import (
	legacyviews "core/application/legacyviews"
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

func (r *ListingRepository) GetJobOffers(filters types.Filter) (legacyviews.PostsResult, error) {
	filters.PostKind = post.PostKindJobOffer
	return r.postRepo.GetPostsByKind(filters)
}

func (r *ListingRepository) GetJobSearches(filters types.Filter) (legacyviews.PostsResult, error) {
	filters.PostKind = post.PostKindJobSearch
	return r.postRepo.GetPostsByKind(filters)
}

func (r *ListingRepository) GetClassified(filters types.Filter) (*post.Post, error) {
	return r.postRepo.GetPostByPublicID(filters.PostID)
}
