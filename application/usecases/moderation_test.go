package usecases

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"core/application/ports"
	domainmoderation "core/domain/moderation"

	"github.com/google/uuid"
)

type fakeModerationRepository struct {
	fetchCalled  bool
	fetchFilter  ports.ModerationReportFilter
	resolveIn    ports.ModerationResolveInput
	setPostID    int64
	setPublished bool
	setReviewer  uuid.UUID
	setNote      string
}

func (r *fakeModerationRepository) FetchReports(ctx context.Context, filter ports.ModerationReportFilter) (ports.ModerationReportPage, error) {
	r.fetchCalled = true
	r.fetchFilter = filter
	return ports.ModerationReportPage{Limit: filter.Limit}, nil
}

func (r *fakeModerationRepository) ResolveReport(ctx context.Context, input ports.ModerationResolveInput) (*ports.ModerationReportView, error) {
	r.resolveIn = input
	return &ports.ModerationReportView{
		ID:           input.ReportID,
		Status:       input.Status,
		ReviewedByID: &input.ReviewedByID,
	}, nil
}

func (r *fakeModerationRepository) SetPostPublished(ctx context.Context, postPublicID int64, published bool, moderatorID uuid.UUID, resolution string) (ports.ModerationPostView, error) {
	r.setPostID = postPublicID
	r.setPublished = published
	r.setReviewer = moderatorID
	r.setNote = resolution
	return ports.ModerationPostView{"public_id": strconv.FormatInt(postPublicID, 10), "published": published}, nil
}

func TestCanModerate(t *testing.T) {
	allowed := []ports.ModeratorRole{
		ports.ModeratorRoleModerator,
		ports.ModeratorRoleAdmin,
		ports.ModeratorRoleSuperAdmin,
	}
	for _, role := range allowed {
		if !CanModerate(ports.ModeratorPrincipal{ID: uuid.New(), Role: role}) {
			t.Fatalf("expected role %s to moderate", role)
		}
	}

	if CanModerate(ports.ModeratorPrincipal{ID: uuid.New(), Role: "user"}) {
		t.Fatal("expected regular user to be denied")
	}
	if CanModerate(ports.ModeratorPrincipal{Role: ports.ModeratorRoleModerator}) {
		t.Fatal("expected missing moderator identity to be denied")
	}
}

func TestModerationServiceFetchReportsRequiresModerator(t *testing.T) {
	repo := &fakeModerationRepository{}
	service := NewModerationService(repo)

	_, err := service.FetchReports(context.Background(), ports.ModeratorPrincipal{ID: uuid.New(), Role: "user"}, ports.ModerationReportFilter{})
	if !errors.Is(err, ErrModerationForbidden) {
		t.Fatalf("expected ErrModerationForbidden, got %v", err)
	}
	if repo.fetchCalled {
		t.Fatal("repository should not be called when user cannot moderate")
	}
}

func TestModerationServiceFetchReportsRejectsInvalidStatus(t *testing.T) {
	repo := &fakeModerationRepository{}
	service := NewModerationService(repo)

	_, err := service.FetchReports(
		context.Background(),
		ports.ModeratorPrincipal{ID: uuid.New(), Role: ports.ModeratorRoleModerator},
		ports.ModerationReportFilter{Status: domainmoderation.Status("bad")},
	)
	if !errors.Is(err, ErrInvalidReportStatus) {
		t.Fatalf("expected ErrInvalidReportStatus, got %v", err)
	}
	if repo.fetchCalled {
		t.Fatal("repository should not be called for invalid status")
	}
}

func TestModerationServiceFetchReportsDefaultsToPending(t *testing.T) {
	repo := &fakeModerationRepository{}
	service := NewModerationService(repo)

	_, err := service.FetchReports(context.Background(), ports.ModeratorPrincipal{ID: uuid.New(), Role: ports.ModeratorRoleModerator}, ports.ModerationReportFilter{})
	if err != nil {
		t.Fatalf("FetchReports() error = %v", err)
	}
	if repo.fetchFilter.Status != domainmoderation.StatusPending {
		t.Fatalf("status = %q, want pending", repo.fetchFilter.Status)
	}
}

func TestModerationServiceResolveReportSetsReviewer(t *testing.T) {
	repo := &fakeModerationRepository{}
	service := NewModerationService(repo)
	moderatorID := uuid.New()
	reportID := uuid.New()

	_, err := service.ResolveReport(
		context.Background(),
		ports.ModeratorPrincipal{ID: moderatorID, Role: ports.ModeratorRoleModerator},
		ports.ModerationResolveInput{
			ReportID: reportID,
			Status:   domainmoderation.StatusReviewed,
		},
	)
	if err != nil {
		t.Fatalf("resolve report returned error: %v", err)
	}
	if repo.resolveIn.ReportID != reportID {
		t.Fatalf("expected report id %s, got %s", reportID, repo.resolveIn.ReportID)
	}
	if repo.resolveIn.ReviewedByID != moderatorID {
		t.Fatalf("expected reviewer id %s, got %s", moderatorID, repo.resolveIn.ReviewedByID)
	}
}

func TestModerationServiceResolveRejectsPendingAndContradictoryVisibility(t *testing.T) {
	repo := &fakeModerationRepository{}
	service := NewModerationService(repo)
	moderator := ports.ModeratorPrincipal{ID: uuid.New(), Role: ports.ModeratorRoleModerator}

	_, err := service.ResolveReport(context.Background(), moderator, ports.ModerationResolveInput{
		ReportID: uuid.New(),
		Status:   domainmoderation.StatusPending,
	})
	if !errors.Is(err, ErrInvalidReportStatus) {
		t.Fatalf("pending resolve error = %v", err)
	}

	publish := false
	_, err = service.ResolveReport(context.Background(), moderator, ports.ModerationResolveInput{
		ReportID:    uuid.New(),
		Status:      domainmoderation.StatusRejected,
		PublishPost: &publish,
	})
	if !errors.Is(err, ports.ErrInvalidModerationAction) {
		t.Fatalf("contradictory visibility error = %v", err)
	}
}

func TestModerationServiceHidePostSetsPostUnpublished(t *testing.T) {
	repo := &fakeModerationRepository{}
	service := NewModerationService(repo)
	moderatorID := uuid.New()

	_, err := service.HidePost(context.Background(), ports.ModeratorPrincipal{ID: moderatorID, Role: ports.ModeratorRoleAdmin}, 42, "violates rules")
	if err != nil {
		t.Fatalf("hide post returned error: %v", err)
	}
	if repo.setPostID != 42 {
		t.Fatalf("expected post id 42, got %d", repo.setPostID)
	}
	if repo.setPublished {
		t.Fatal("expected hidden post to be unpublished")
	}
	if repo.setReviewer != moderatorID {
		t.Fatalf("expected reviewer id %s, got %s", moderatorID, repo.setReviewer)
	}
	if repo.setNote != "violates rules" {
		t.Fatalf("expected resolution to be passed through, got %q", repo.setNote)
	}
}

func TestModerationServiceRejectsNonPositivePostID(t *testing.T) {
	repo := &fakeModerationRepository{}
	service := NewModerationService(repo)
	moderator := ports.ModeratorPrincipal{ID: uuid.New(), Role: ports.ModeratorRoleModerator}

	for _, postID := range []int64{0, -1} {
		if _, err := service.HidePost(context.Background(), moderator, postID, "invalid"); !errors.Is(err, ErrPostIDRequired) {
			t.Fatalf("HidePost(%d) error = %v", postID, err)
		}
	}
	if repo.setPostID != 0 {
		t.Fatalf("invalid post id reached repository: %d", repo.setPostID)
	}
}
