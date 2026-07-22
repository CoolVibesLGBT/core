package usecases

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainmoderation "core/domain/moderation"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrModerationForbidden = errors.New("moderation access denied")
	ErrInvalidReportStatus = domainmoderation.ErrInvalidStatus
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

func (s *ModerationService) FetchReports(ctx context.Context, principal ports.ModeratorPrincipal, filter ports.ModerationReportFilter) (ports.ModerationReportPage, error) {
	if !CanModerate(principal) {
		return ports.ModerationReportPage{}, ErrModerationForbidden
	}
	if filter.Limit <= 0 {
		filter.Limit = constants.DEFAULT_LIMIT
	}
	if filter.Limit > constants.MAXIMUM_LIMIT {
		filter.Limit = constants.MAXIMUM_LIMIT
	}
	if !filter.AllStatuses && filter.Status == "" {
		filter.Status = domainmoderation.StatusPending
	}
	if filter.Status != "" && !filter.Status.IsValid() {
		return ports.ModerationReportPage{}, ErrInvalidReportStatus
	}
	if filter.ContentableType != "" {
		if _, err := domainmoderation.ParseTargetType(string(filter.ContentableType)); err != nil {
			return ports.ModerationReportPage{}, ErrInvalidReportType
		}
	}
	return s.repo.FetchReports(ctx, filter)
}

func (s *ModerationService) ResolveReport(ctx context.Context, principal ports.ModeratorPrincipal, input ports.ModerationResolveInput) (*ports.ModerationReportView, error) {
	if !CanModerate(principal) {
		return nil, ErrModerationForbidden
	}
	if input.ReportID == uuid.Nil {
		return nil, ErrReportIDRequired
	}
	if err := domainmoderation.ValidateResolution(input.Status, input.PublishPost); err != nil {
		return nil, err
	}
	input.ReviewedByID = principal.ID
	return s.repo.ResolveReport(ctx, input)
}

func (s *ModerationService) HidePost(ctx context.Context, principal ports.ModeratorPrincipal, postPublicID int64, resolution string) (ports.ModerationPostView, error) {
	return s.setPostPublished(ctx, principal, postPublicID, false, resolution)
}

func (s *ModerationService) UnhidePost(ctx context.Context, principal ports.ModeratorPrincipal, postPublicID int64, resolution string) (ports.ModerationPostView, error) {
	return s.setPostPublished(ctx, principal, postPublicID, true, resolution)
}

func (s *ModerationService) setPostPublished(ctx context.Context, principal ports.ModeratorPrincipal, postPublicID int64, published bool, resolution string) (ports.ModerationPostView, error) {
	if !CanModerate(principal) {
		return nil, ErrModerationForbidden
	}
	if postPublicID <= 0 {
		return nil, ErrPostIDRequired
	}
	return s.repo.SetPostPublished(ctx, postPublicID, published, principal.ID, resolution)
}

func CanModerate(principal ports.ModeratorPrincipal) bool {
	if principal.ID == uuid.Nil {
		return false
	}
	switch principal.Role {
	case ports.ModeratorRoleModerator,
		ports.ModeratorRoleAdmin,
		ports.ModeratorRoleSuperAdmin:
		return true
	default:
		return false
	}
}
