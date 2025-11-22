package services

import (
	"context"
	"coolvibes/models"
	"coolvibes/models/post"

	"coolvibes/repositories"
	"coolvibes/types"
	"fmt"
	"mime/multipart"

	"github.com/google/uuid"
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

func (s *PostService) CreatePost(request map[string][]string, files []*multipart.FileHeader, author *models.User) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(request, files, author, "post", nil)
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

func (s *PostService) GetPostByPublicID(id int64) (*post.Post, error) {
	postData, err := s.postRepo.GetPostByPublicID(id)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return postData, nil
}

func (s *PostService) GetTimeline(limit int, cursor *int64) (types.TimelineResult, error) {
	// Repo fonksiyonunu çağırıyoruz
	posts, err := s.postRepo.GetTimeline(limit, cursor)
	if err != nil {
		return types.TimelineResult{}, err
	}
	return posts, nil
}

func (s *PostService) GetPostsByUserID(id int64, limit int, cursor *int64) ([]post.Post, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(id)
	if err != nil {
		return nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	posts, err := s.postRepo.GetUserPosts(userId, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return posts, nil
}

func (s *PostService) GetUserPostReplies(id int64, limit int, cursor *int64) ([]post.Post, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(id)
	if err != nil {
		return nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	posts, err := s.postRepo.GetUserPostReplies(userId, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return posts, nil
}

func (s *PostService) GetUserMedias(id int64, limit int, cursor *int64) ([]types.MediaWithUser, *int64, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(id)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	medias, lastCursor, err := s.postRepo.GetUserMedias(userId, cursor, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserMedias error: %w", err)
	}
	return medias, lastCursor, nil
}

func (s *PostService) GetRecentHashtags(limit int) ([]types.HashtagStats, error) {
	hashtags, err := s.postRepo.GetRecentHashtags(limit)
	if err != nil {
		return nil, fmt.Errorf("GetRecentHashtags error: %w", err)
	}
	return hashtags, nil
}

func (s *PostService) GetTimelineVibes(limit int, cursor *int64) (types.TimelineResult, error) {
	// Repo fonksiyonunu çağırıyoruz
	posts, err := s.postRepo.GetTimelineVibes(limit, cursor)
	if err != nil {
		return types.TimelineResult{}, err
	}
	return posts, nil
}

func (s *PostService) Vote(ctx context.Context, choiceId uuid.UUID, weight int, rank int, userId uuid.UUID) error {
	return s.postRepo.Vote(ctx, choiceId, weight, rank, userId)
}

func (s *PostService) Like(ctx context.Context, postId int64, authUser *models.User) error {
	return s.postRepo.Like(ctx, postId, authUser)
}

func (s *PostService) Dislike(ctx context.Context, postId int64, authUser *models.User) error {
	return s.postRepo.Dislike(ctx, postId, authUser)
}

func (s *PostService) Banana(ctx context.Context, postId int64, authUser *models.User) error {
	return s.postRepo.Banana(ctx, postId, authUser)
}

func (s *PostService) Report(ctx context.Context, postId int64, authUser *models.User) error {
	return s.postRepo.Report(ctx, postId, authUser)
}

func (s *PostService) Bookmark(ctx context.Context, postId int64, authUser *models.User) error {
	return s.postRepo.Bookmark(ctx, postId, authUser)
}

func (s *PostService) View(ctx context.Context, postId int64, authUser *models.User) error {
	return s.postRepo.View(ctx, postId, authUser)
}
