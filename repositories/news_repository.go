package repositories

import (
	"context"
	"core/helpers"
	"core/models/post"
	"core/models/taxonomy"
	"core/types"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type newsPostRepository interface {
	CreatePost(*post.Post) error
	GetPostsByKind(types.Filter) (types.PostsResult, error)
	GetPostByID(uuid.UUID) (*post.Post, error)
	GetPostByPublicID(int64) (*post.Post, error)
	ExistsBySlug(types.Filter) (bool, error)
	GetPillarsWithClustersWithSlug(context.Context, string) ([]taxonomy.Pillar, error)
}

type NewsRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	mediaRepo        *MediaRepository
	userRepo         *UserRepository
	notificationRepo *NotificationRepository
	postRepo         newsPostRepository
	rawPostRepo      *PostRepository
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
	return r.rawPostRepo
}

func NewNewsRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository, notificationRepo *NotificationRepository, postRepo *PostRepository) *NewsRepository {
	return &NewsRepository{
		db:               db,
		snowFlakeNode:    snowFlakeNode,
		mediaRepo:        mediaRepo,
		userRepo:         userRepo,
		notificationRepo: notificationRepo,
		postRepo:         postRepo,
		rawPostRepo:      postRepo,
	}
}

func (r *NewsRepository) Create(news *post.Post) error {
	return r.postRepo.CreatePost(news)
}

func (r *NewsRepository) GetNews(filters types.Filter) (types.PostsResult, error) {
	filters.PostKind = post.PostKindNews
	return r.postRepo.GetPostsByKind(filters)
}

func (r *NewsRepository) Get(filters types.Filter) (*post.Post, error) {
	postData, err := r.loadPost(filters)
	if err != nil {
		return nil, err
	}

	if postData == nil || postData.PostKind != post.PostKindNews {
		return nil, newsNotFoundError(filters)
	}

	return postData, nil
}

func (r *NewsRepository) loadPost(filters types.Filter) (*post.Post, error) {
	if filters.PostID == 0 && filters.PostUUID == uuid.Nil {
		return nil, fmt.Errorf("post id is required")
	}

	if filters.PostID != 0 {
		return r.postRepo.GetPostByPublicID(filters.PostID)
	}

	return r.postRepo.GetPostByID(filters.PostUUID)
}

func newsNotFoundError(filters types.Filter) error {
	if filters.PostID != 0 {
		return fmt.Errorf("post with id %d not found", filters.PostID)
	}
	return fmt.Errorf("post with id %s not found", filters.PostUUID)
}

func (r *NewsRepository) IsNewsExists(filters types.Filter) (bool, error) {
	return r.PostRepo().ExistsBySlug(filters)
}

func (r *NewsRepository) Categories(filters types.Filter) ([]*taxonomy.Pillar, error) {
	ctx := filters.Context
	if ctx == nil {
		ctx = context.Background()
	}

	pillars, err := r.postRepo.GetPillarsWithClustersWithSlug(ctx, "news")
	if err != nil {
		return nil, err
	}

	result := make([]*taxonomy.Pillar, 0, len(pillars))
	for index := range pillars {
		result = append(result, &pillars[index])
	}

	return result, nil
}

func (r *NewsRepository) Category(filters types.Filter) (*taxonomy.Pillar, error) {
	categories, err := r.Categories(filters)
	if err != nil {
		return nil, err
	}
	if len(categories) == 0 {
		return nil, nil
	}

	return categories[0], nil
}
