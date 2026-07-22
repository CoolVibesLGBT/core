package usecases

import (
	"context"
	legacyviews "core/application/legacyviews"
	"core/application/ports"
	domainmoderation "core/domain/moderation"
	domainwallet "core/domain/wallet"
	"core/models"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"core/models/taxonomy"

	"core/application/types"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PostService struct {
	mediaRepo        ports.MediaRepository
	userRepo         ports.UserRepository
	postRepo         ports.PostRepository
	publicPostReader ports.PublicPostReader
}

// legacyPublicPostReader keeps test/during-migration PostRepository
// implementations compatible while the production adapter implements the
// persistence-free PublicPostReader port directly.
type legacyPublicPostReader struct{ repo ports.PostRepository }

func NewPostService(
	userRepo ports.UserRepository,
	postRepo ports.PostRepository,
	mediaRepo ports.MediaRepository) *PostService {
	reader, ok := postRepo.(ports.PublicPostReader)
	if !ok {
		reader = legacyPublicPostReader{repo: postRepo}
	}
	return &PostService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, publicPostReader: reader}
}

func (s *PostService) ServiceName() string {
	return "PostService"
}

func (s *PostService) CreatePost(context context.Context, form ports.FormData, author *models.User, postKind post.PostKind) (*types.PublicPost, error) {
	_post, err := s.postRepo.CreateContentablePost(context, form, author, string(postKind), nil)
	if err != nil {
		return nil, err
	}
	created, err := s.postRepo.GetPostByIDIncludingUnpublished(_post.ID)
	if err != nil {
		return nil, err
	}
	result := legacyviews.ProjectPublicPost(*created)
	return &result, nil
}

func (s *PostService) GetPostByID(id uuid.UUID) (*types.PublicPost, error) {
	postData, err := s.publicPostReader.FindPublicPostByID(id)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return postData, nil
}

func (s *PostService) GetPostBySlug(filters types.Filter) (*types.PublicPost, error) {
	postData, err := s.publicPostReader.FindPublicPostBySlug(filters)
	if err != nil {
		return nil, fmt.Errorf("GetPostBySlug error: %w", err)
	}
	return postData, nil
}

func (s *PostService) GetPostByPublicID(id int64) (*types.PublicPost, error) {
	postData, err := s.publicPostReader.FindPublicPostByPublicID(id)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return postData, nil
}

func (s *PostService) GetTimeline(filters types.Filter) (types.PublicPostPage, error) {
	posts, err := s.publicPostReader.FetchPublicTimeline(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return posts, nil
}

func (s *PostService) SearchPost(filters types.Filter) (types.PublicPostPage, error) {
	posts, err := s.publicPostReader.SearchPublicPosts(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return posts, nil
}

func (s *PostService) GetPostsByUserID(filters types.Filter) ([]types.PublicPost, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(filters.UserID)
	if err != nil {
		return nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	posts, err := s.publicPostReader.FetchPublicUserPosts(userId, filters)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return posts, nil
}

func (s *PostService) GetUserPostReplies(filters types.Filter) ([]types.PublicPost, error) {
	userUUID, err := s.userRepo.GetUserUUIDByPublicID(filters.UserID)
	if err != nil {
		return nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	filters.UserUUID = userUUID
	posts, err := s.publicPostReader.FetchPublicUserPostReplies(filters)
	if err != nil {
		return nil, fmt.Errorf("GetPostByID error: %w", err)
	}
	return posts, nil
}

func (s *PostService) GetUserMedias(filters types.Filter) ([]types.PublicPostMediaWithUser, *int64, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(filters.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserUUIDByPublicID error: %w", err)
	}
	filters.UserUUID = userId
	page, err := s.publicPostReader.FetchPublicUserMedia(filters)
	if err != nil {
		return nil, nil, fmt.Errorf("GetUserMedias error: %w", err)
	}
	return page.Items, page.NextPublicID, nil
}

func (s *PostService) GetRecentHashtags(filters types.Filter) ([]types.HashtagStats, error) {
	hashtags, err := s.postRepo.GetRecentHashtags(filters)
	if err != nil {
		return nil, fmt.Errorf("GetRecentHashtags error: %w", err)
	}
	return hashtags, nil
}

func (s *PostService) GetTimelineVibes(filters types.Filter) (types.PublicPostPage, error) {
	posts, err := s.publicPostReader.FetchPublicTimelineVibes(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return posts, nil
}

func (r legacyPublicPostReader) FindPublicPostByID(id uuid.UUID) (*types.PublicPost, error) {
	item, err := r.repo.GetPostByID(id)
	if err != nil {
		return nil, err
	}
	result := legacyviews.ProjectPublicPost(*item)
	return &result, nil
}

func (r legacyPublicPostReader) FindPublicPostBySlug(filters types.Filter) (*types.PublicPost, error) {
	item, err := r.repo.GetPostBySlug(filters)
	if err != nil {
		return nil, err
	}
	result := legacyviews.ProjectPublicPost(*item)
	return &result, nil
}

func (r legacyPublicPostReader) FindPublicPostByPublicID(id int64) (*types.PublicPost, error) {
	item, err := r.repo.GetPostByPublicID(id)
	if err != nil {
		return nil, err
	}
	result := legacyviews.ProjectPublicPost(*item)
	return &result, nil
}

func (r legacyPublicPostReader) FetchPublicTimeline(filters types.Filter) (types.PublicPostPage, error) {
	result, err := r.repo.GetTimeline(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return legacyviews.ProjectPublicPostPage(result), nil
}

func (r legacyPublicPostReader) SearchPublicPosts(filters types.Filter) (types.PublicPostPage, error) {
	result, err := r.repo.FindPostsByKind(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return legacyviews.ProjectPublicPostsResult(result), nil
}

func (r legacyPublicPostReader) FetchPublicUserPosts(userID uuid.UUID, filters types.Filter) ([]types.PublicPost, error) {
	result, err := r.repo.GetUserPosts(userID, filters)
	if err != nil {
		return nil, err
	}
	return legacyviews.ProjectPublicPosts(result), nil
}

func (r legacyPublicPostReader) FetchPublicUserPostReplies(filters types.Filter) ([]types.PublicPost, error) {
	result, err := r.repo.GetUserPostReplies(filters)
	if err != nil {
		return nil, err
	}
	return legacyviews.ProjectPublicPosts(result), nil
}

func (r legacyPublicPostReader) FetchPublicUserMedia(filters types.Filter) (types.PublicPostMediaPage, error) {
	result, next, err := r.repo.GetUserMedias(filters)
	if err != nil {
		return types.PublicPostMediaPage{}, err
	}
	return types.PublicPostMediaPage{Items: legacyviews.ProjectPublicMediaItems(result), NextPublicID: next}, nil
}

func (r legacyPublicPostReader) FetchPublicTimelineVibes(filters types.Filter) (types.PublicPostPage, error) {
	result, err := r.repo.GetTimelineVibes(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return legacyviews.ProjectPublicPostPage(result), nil
}

func (s *PostService) Vote(ctx context.Context, choiceId uuid.UUID, weight int, rank int, userId uuid.UUID) error {
	return s.postRepo.Vote(ctx, choiceId, weight, rank, userId)
}

func (s *PostService) SetEventRSVP(ctx context.Context, postPublicID int64, authUser *models.User, status *postpayloads.EventAttendanceStatus) (*postpayloads.EventRSVPResult, error) {
	if authUser == nil {
		return nil, errors.New("authenticated user is required")
	}
	return s.postRepo.SetEventRSVP(ctx, postPublicID, authUser.ID, status)
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

func (s *PostService) Report(ctx context.Context, postID int64, kind string, description string, authUser *models.User) error {
	report, err := validateReportSubmission(domainmoderation.TargetPost, postID, kind, description, authUser)
	if err != nil {
		if errors.Is(err, domainmoderation.ErrInvalidTarget) {
			return ErrPostIDRequired
		}
		return err
	}
	return s.postRepo.Report(
		ctx,
		report.Target().PublicID(),
		report.Kind().String(),
		report.Description().String(),
		authUser,
	)
}

func (s *PostService) Bookmark(filters types.Filter) error {
	return s.postRepo.Bookmark(filters)
}

func (s *PostService) View(filters types.Filter) (bool, error) {
	return s.postRepo.View(filters)
}

func (s *PostService) Tip(ctx context.Context, postId int64, authUser *models.User, amount decimal.Decimal, rawIdempotencyKey string) (*decimal.Decimal, error) {
	if err := domainwallet.ValidateTipAmount(amount); err != nil {
		return nil, err
	}
	idempotencyKey, err := domainwallet.NewIdempotencyKey(rawIdempotencyKey)
	if err != nil {
		return nil, err
	}
	return s.postRepo.Tip(ctx, postId, authUser, amount, idempotencyKey)
}

func (s *PostService) GetPillarsWithClusters(filters types.Filter) ([]taxonomy.Pillar, error) {
	return s.postRepo.GetPillarsWithClusters(filters)
}

func (s *PostService) FetchNearbyUsers(filters types.Filter) ([]types.NearbyUser, *float64, error) {
	return s.userRepo.FetchNearbyUsers(filters)
}

func (s *PostService) GetPreferences() (*models.PreferencesData, error) {
	return s.userRepo.GetPreferences()
}
