package repositories

import (
	"coolvibes/helpers"
	"coolvibes/models/post"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlaceRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	mediaRepo        *MediaRepository
	userRepo         *UserRepository
	postRepo         *PostRepository
	notificationRepo *NotificationRepository
}

func (r *PlaceRepository) DB() *gorm.DB {
	return r.db
}

func (r *PlaceRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func (r *PlaceRepository) MediaRepo() *MediaRepository {
	return r.mediaRepo
}

func (r *PlaceRepository) UserRepo() *UserRepository {
	return r.userRepo
}

func (r *PlaceRepository) NotificationRepo() *NotificationRepository {
	return r.notificationRepo
}

func (r *PlaceRepository) PostRepo() *PostRepository {
	return r.postRepo
}

func NewPlaceRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository, notificationRepo *NotificationRepository, postRepo *PostRepository) *PlaceRepository {
	return &PlaceRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo, userRepo: userRepo, notificationRepo: notificationRepo, postRepo: postRepo}
}

func (r *PlaceRepository) Create(place *post.Post) error {
	return r.postRepo.CreatePost(place)
}

func (r *PlaceRepository) GetPlaceByID(id uuid.UUID) (*post.Post, error) {
	return r.postRepo.GetPostByID(id)
}
