package repositories

import (
	"coolvibes/helpers"
	"coolvibes/models/post"
	"coolvibes/types"
	"strconv"

	"gorm.io/gorm"
)

type NewsRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	mediaRepo        *MediaRepository
	userRepo         *UserRepository
	notificationRepo *NotificationRepository
	postRepo         *PostRepository
}

func (r *NewsRepository) DB() *gorm.DB {
	return r.db
}

func (r *NewsRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func (r *NewsRepository) MediaRepo() *MediaRepository {
	return r.mediaRepo
}

func (r *NewsRepository) UserRepo() *UserRepository {
	return r.userRepo
}

func (r *NewsRepository) NotificationRepo() *NotificationRepository {
	return r.notificationRepo
}

func (r *NewsRepository) PostRepo() *PostRepository {
	return r.postRepo
}

func NewNewsRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository, notificationRepo *NotificationRepository, postRepo *PostRepository) *NewsRepository {
	return &NewsRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo, userRepo: userRepo, notificationRepo: notificationRepo, postRepo: postRepo}
}

func (r *NewsRepository) Create(news *post.Post) error {
	return r.postRepo.CreatePost(news)
}

func (r *NewsRepository) GetNews(filters types.Filter) ([]*post.Post, types.Cursor, error) {

	var prevCursor *string
	if filters.Cursor != nil {
		s := strconv.FormatInt(*filters.Cursor, 10)
		prevCursor = &s
	}

	cursorInfo := types.Cursor{
		Prev: prevCursor,
		Next: nil,
	}

	return nil, cursorInfo, nil
}

func (r *NewsRepository) IsNewsExists(filters types.Filter) (bool, error) {
	return r.PostRepo().ExistsBySlug(filters)
}
