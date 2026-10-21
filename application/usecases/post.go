package usecases

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainmedia "core/domain/media"
	domainmoderation "core/domain/moderation"
	domainwallet "core/domain/wallet"
	"core/models"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"core/models/taxonomy"

	"core/application/types"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PostService struct {
	mediaRepo ports.MediaRepository
	userRepo  ports.UserRepository
	postRepo  ports.PostRepository
}

func NewPostService(userRepo ports.UserRepository, postRepo ports.PostRepository, mediaRepo ports.MediaRepository) *PostService {
	return &PostService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo}
}

func (s *PostService) ServiceName() string {
	return "PostService"
}

func (s *PostService) CreatePost(context context.Context, values map[string][]string, files []*domainmedia.UploadedFile, author *models.User, postKind post.PostKind) (*types.PublicPost, error) {
	_post, err := s.postRepo.CreateContentablePost(context, values, files, author, string(postKind), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.FindPublicPostByIDIncludingUnpublished(_post.ID)
}

func (s *PostService) GetPostByID(id uuid.UUID) (*types.PublicPost, error) {
	postData, err := s.postRepo.FindPublicPostByID(id)
	if err != nil {
		return nil, constants.ErrPostLookupFailed.Format(err)
	}
	return postData, nil
}

func (s *PostService) GetPostBySlug(filters types.Filter) (*types.PublicPost, error) {
	postData, err := s.postRepo.FindPublicPostBySlug(filters)
	if err != nil {
		return nil, constants.ErrPostSlugLookupFailed.Format(err)
	}
	return postData, nil
}

func (s *PostService) GetPostByPublicID(id int64) (*types.PublicPost, error) {
	postData, err := s.postRepo.FindPublicPostByPublicID(id)
	if err != nil {
		return nil, constants.ErrPostLookupFailed.Format(err)
	}
	return postData, nil
}

func (s *PostService) GetTimeline(filters types.Filter) (types.PublicPostPage, error) {
	posts, err := s.postRepo.FetchPublicTimeline(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return posts, nil
}

func (s *PostService) SearchPost(filters types.Filter) (types.PublicPostPage, error) {
	posts, err := s.postRepo.SearchPublicPosts(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return posts, nil
}

func (s *PostService) GetPostsByUserID(filters types.Filter) ([]types.PublicPost, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(filters.Context, filters.UserID)
	if err != nil {
		return nil, constants.ErrUserUUIDLookupFailed.Format(err)
	}
	posts, err := s.postRepo.FetchPublicUserPosts(userId, filters)
	if err != nil {
		return nil, constants.ErrPostLookupFailed.Format(err)
	}
	return posts, nil
}

func (s *PostService) GetUserPostReplies(filters types.Filter) ([]types.PublicPost, error) {
	userUUID, err := s.userRepo.GetUserUUIDByPublicID(filters.Context, filters.UserID)
	if err != nil {
		return nil, constants.ErrUserUUIDLookupFailed.Format(err)
	}
	filters.UserUUID = userUUID
	posts, err := s.postRepo.FetchPublicUserPostReplies(filters)
	if err != nil {
		return nil, constants.ErrPostLookupFailed.Format(err)
	}
	return posts, nil
}

func (s *PostService) GetUserMedias(filters types.Filter) ([]types.PublicPostMediaWithUser, *int64, error) {
	userId, err := s.userRepo.GetUserUUIDByPublicID(filters.Context, filters.UserID)
	if err != nil {
		return nil, nil, constants.ErrUserUUIDLookupFailed.Format(err)
	}
	filters.UserUUID = userId
	page, err := s.postRepo.FetchPublicUserMedia(filters)
	if err != nil {
		return nil, nil, constants.ErrUserMediaLookupFailed.Format(err)
	}
	return page.Items, page.NextPublicID, nil
}

func (s *PostService) GetRecentHashtags(filters types.Filter) ([]types.HashtagStats, error) {
	hashtags, err := s.postRepo.GetRecentHashtags(filters)
	if err != nil {
		return nil, constants.ErrRecentHashtagsLookupFailed.Format(err)
	}
	return hashtags, nil
}

func (s *PostService) GetTimelineVibes(filters types.Filter) (types.PublicPostPage, error) {
	posts, err := s.postRepo.FetchPublicTimelineVibes(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return posts, nil
}

func (s *PostService) Vote(ctx context.Context, choiceId uuid.UUID, weight int, rank int, userId uuid.UUID) error {
	return s.postRepo.Vote(ctx, choiceId, weight, rank, userId)
}

func (s *PostService) SetEventRSVP(ctx context.Context, postPublicID int64, authUser *models.User, status *postpayloads.EventAttendanceStatus) (*postpayloads.EventRSVPResult, error) {
	if authUser == nil {
		return nil, constants.ErrAuthenticatedUserRequired
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
