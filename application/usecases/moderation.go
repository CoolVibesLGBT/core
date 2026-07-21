package usecases

import (
	"context"
	"core/application/ports"
	"core/constants"
	"core/models"
	"core/models/post"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrModerationForbidden = errors.New("moderation access denied")
	ErrInvalidReportStatus = errors.New("invalid report status")
	ErrReportIDRequired    = errors.New("report_id is required")
	ErrPostIDRequired      = errors.New("post_id is required")
	ErrInvalidReportType   = errors.New("invalid report content type")
)

type ModerationService struct {
	repo ports.ModerationRepository
}

func NewModerationService(repo ports.ModerationRepository) *ModerationService {
	return &ModerationService{repo: repo}
}

func (s *ModerationService) FetchReports(ctx context.Context, authUser *models.User, filter ports.ModerationReportFilter) (ports.ModerationReportPage, error) {
	if !CanModerate(authUser) {
		return ports.ModerationReportPage{}, ErrModerationForbidden
	}
	if filter.Limit <= 0 {
		filter.Limit = constants.DEFAULT_LIMIT
	}
	if filter.Limit > constants.MAXIMUM_LIMIT {
		filter.Limit = constants.MAXIMUM_LIMIT
	}
	if !filter.AllStatuses && filter.Status == "" {
		filter.Status = models.ReportStatusPending
	}
	if filter.Status != "" && !filter.Status.IsValid() {
		return ports.ModerationReportPage{}, ErrInvalidReportStatus
	}
	if filter.ContentableType != "" &&
		filter.ContentableType != models.EngagementContentableTypePost &&
		filter.ContentableType != models.EngagementContentableTypeUser {
		return ports.ModerationReportPage{}, ErrInvalidReportType
	}
	return s.repo.FetchReports(ctx, filter)
}

func (s *ModerationService) ResolveReport(ctx context.Context, authUser *models.User, input ports.ModerationResolveInput) (*models.Report, error) {
	if !CanModerate(authUser) {
		return nil, ErrModerationForbidden
	}
	if input.ReportID == uuid.Nil {
		return nil, ErrReportIDRequired
	}
	if !input.Status.IsValid() || input.Status == models.ReportStatusPending {
		return nil, ErrInvalidReportStatus
	}
	if input.PublishPost != nil {
		if (!*input.PublishPost && input.Status != models.ReportStatusActioned) ||
			(*input.PublishPost && input.Status == models.ReportStatusActioned) {
			return nil, ports.ErrInvalidModerationAction
		}
	}
	input.ReviewedByID = authUser.ID
	return s.repo.ResolveReport(ctx, input)
}

func (s *ModerationService) HidePost(ctx context.Context, authUser *models.User, postPublicID int64, resolution string) (*post.Post, error) {
	return s.setPostPublished(ctx, authUser, postPublicID, false, resolution)
}

func (s *ModerationService) UnhidePost(ctx context.Context, authUser *models.User, postPublicID int64, resolution string) (*post.Post, error) {
	return s.setPostPublished(ctx, authUser, postPublicID, true, resolution)
}

func (s *ModerationService) setPostPublished(ctx context.Context, authUser *models.User, postPublicID int64, published bool, resolution string) (*post.Post, error) {
	if !CanModerate(authUser) {
		return nil, ErrModerationForbidden
	}
	if postPublicID == 0 {
		return nil, ErrPostIDRequired
	}
	return s.repo.SetPostPublished(ctx, postPublicID, published, authUser.ID, resolution)
}

func CanModerate(user *models.User) bool {
	if user == nil {
		return false
	}
	switch user.UserRole {
	case constants.UserRoleModerator,
		constants.UserRoleAdmin,
		constants.UserRoleSuperAdmin:
		return true
	default:
		return false
	}
}
