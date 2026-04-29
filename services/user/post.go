package services

import (
	"context"
	"core/models"
	"core/models/post"
	"core/models/taxonomy"

	"core/repositories"
	"core/types"
	"fmt"
	"mime/multipart"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PostService struct {
	mediaRepo *repositories.MediaRepository
	userRepo  *repositories.UserRepository
	postRepo  *repositories.PostRepository
}

func NewPostService(
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository) *PostService {
	return &PostService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo}
}

func (s *PostService) ServiceName() string {
	return "PostService"
}

func (s *PostService) UserRepo() *repositories.UserRepository {
	return s.userRepo
}

func (s *PostService) CreatePost(context context.Context, request map[string][]string, files []*multipart.FileHeader, author *models.User, postKind post.PostKind) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(context, request, files, author, string(postKind), nil)
	if err != nil {
		return nil, err
	}
	return s.GetPostByID(_post.ID)
}

func (s *PostService) GetPostByID(id uuid.UUID) (*post.Post, error) {
	postData, err := s.postRepo.GetPostByID(id)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return postData, nil
}

func (s *PostService) GetPostBySlug(filters types.Filter) (*post.Post, error) {
	postData, err := s.postRepo.GetPostBySlug(filters)
	if err != nil {
		return nil, fmt.Errorf("GetPostBySlug error: %w", err)
	}
	return postData, nil
}

func (s *PostService) GetPostByPublicID(id int64) (*post.Post, error) {
	postData, err := s.postRepo.GetPostByPublicID(id)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return postData, nil
}

func (s *PostService) GetTimeline(filters types.Filter) (types.TimelineResult, error) {
	posts, err := s.postRepo.GetTimeline(filters)
	if err != nil {
		return types.TimelineResult{}, err
	}
	return posts, nil
}

func (s *PostService) SearchPost(filters types.Filter) (types.PostsResult, error) {
	posts, err := s.postRepo.FindPostsByKind(filters)
	if err != nil {
		return types.PostsResult{}, err
	}
	return posts, nil
}

func (s *PostService) GetPostsByUserID(filters types.Filter) ([]post.Post, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(filters.UserID)
	if err != nil {
		return nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	posts, err := s.postRepo.GetUserPosts(userId, filters)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return posts, nil
}

func (s *PostService) GetUserPostReplies(filters types.Filter) ([]post.Post, error) {
	userUUID, err := s.userRepo.GetUserUUIDByPublicID(filters.UserID)
	if err != nil {
		return nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	filters.UserUUID = userUUID
	posts, err := s.postRepo.GetUserPostReplies(filters)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return posts, nil
}

func (s *PostService) GetUserMedias(filters types.Filter) ([]types.MediaWithUser, *int64, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(filters.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	filters.UserUUID = userId
	medias, lastCursor, err := s.postRepo.GetUserMedias(filters)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserMedias error: %w", err)
	}
	return medias, lastCursor, nil
}

func (s *PostService) GetRecentHashtags(filters types.Filter) ([]types.HashtagStats, error) {
	hashtags, err := s.postRepo.GetRecentHashtags(filters)
	if err != nil {
		return nil, fmt.Errorf("GetRecentHashtags error: %w", err)
	}
	return hashtags, nil
}

func (s *PostService) GetTimelineVibes(filters types.Filter) (types.TimelineResult, error) {
	posts, err := s.postRepo.GetTimelineVibes(filters)
	if err != nil {
		return types.TimelineResult{}, err
	}
	return posts, nil
}

func (s *PostService) Vote(ctx context.Context, choiceId uuid.UUID, weight int, rank int, userId uuid.UUID) error {
	return s.postRepo.Vote(ctx, choiceId, weight, rank, userId)
}

func (s *PostService) Like(filters types.Filter) error {
	return s.postRepo.Like(filters)
}

func (s *PostService) Dislike(filters types.Filter) error {
	return s.postRepo.Dislike(filters)
}

func (s *PostService) Banana(filters types.Filter) error {
	return s.postRepo.Banana(filters)
}

func (s *PostService) Delete(filters types.Filter) error {
	return s.postRepo.Delete(filters)
}

func (s *PostService) Report(ctx context.Context, postId int64, reason string, description string, authUser *models.User) error {
	return s.postRepo.Report(ctx, postId, reason, description, authUser)
}

func (s *PostService) Bookmark(filters types.Filter) error {
	return s.postRepo.Bookmark(filters)
}

func (s *PostService) View(filters types.Filter) error {
	return s.postRepo.View(filters)
}

func (s *PostService) Tip(ctx context.Context, postId int64, authUser *models.User, amount decimal.Decimal) (*decimal.Decimal, error) {
	return s.postRepo.Tip(ctx, postId, authUser, amount)
}

func (s *PostService) GetPillarsWithClusters(filters types.Filter) ([]taxonomy.Pillar, error) {
	return s.postRepo.GetPillarsWithClusters(filters)
}
