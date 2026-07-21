package usecases

import (
	"context"
	"errors"
	"testing"

	"core/application/ports"
	"core/constants"
	"core/models"
	"core/models/post"

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

func (r *fakeModerationRepository) ResolveReport(ctx context.Context, input ports.ModerationResolveInput) (*models.Report, error) {
	r.resolveIn = input
	return &models.Report{ID: input.ReportID, Status: input.Status, ReviewedByID: &input.ReviewedByID}, nil
}

func (r *fakeModerationRepository) SetPostPublished(ctx context.Context, postPublicID int64, published bool, moderatorID uuid.UUID, resolution string) (*post.Post, error) {
	r.setPostID = postPublicID
	r.setPublished = published
	r.setReviewer = moderatorID
	r.setNote = resolution
	return &post.Post{PublicID: postPublicID, Published: published}, nil
}

func TestCanModerate(t *testing.T) {
	allowed := []constants.UserRole{
		constants.UserRoleModerator,
		constants.UserRoleAdmin,
		constants.UserRoleSuperAdmin,
	}
	for _, role := range allowed {
		if !CanModerate(&models.User{UserRole: role}) {
			t.Fatalf("expected role %s to moderate", role)
		}
	}

	if CanModerate(&models.User{UserRole: constants.UserRoleUser}) {
		t.Fatal("expected regular user to be denied")
	}
	if CanModerate(nil) {
		t.Fatal("expected nil user to be denied")
	}
}

func TestModerationServiceFetchReportsRequiresModerator(t *testing.T) {
	repo := &fakeModerationRepository{}
	service := NewModerationService(repo)

	_, err := service.FetchReports(context.Background(), &models.User{UserRole: constants.UserRoleUser}, ports.ModerationReportFilter{})
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
		&models.User{UserRole: constants.UserRoleModerator},
		ports.ModerationReportFilter{Status: models.ReportStatus("bad")},
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

	_, err := service.FetchReports(context.Background(), &models.User{UserRole: constants.UserRoleModerator}, ports.ModerationReportFilter{})
	if err != nil {
		t.Fatalf("FetchReports() error = %v", err)
	}
	if repo.fetchFilter.Status != models.ReportStatusPending {
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
		&models.User{ID: moderatorID, UserRole: constants.UserRoleModerator},
		ports.ModerationResolveInput{
			ReportID: reportID,
			Status:   models.ReportStatusReviewed,
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
	moderator := &models.User{ID: uuid.New(), UserRole: constants.UserRoleModerator}

	_, err := service.ResolveReport(context.Background(), moderator, ports.ModerationResolveInput{
		ReportID: uuid.New(),
		Status:   models.ReportStatusPending,
	})
	if !errors.Is(err, ErrInvalidReportStatus) {
		t.Fatalf("pending resolve error = %v", err)
	}

	publish := false
	_, err = service.ResolveReport(context.Background(), moderator, ports.ModerationResolveInput{
		ReportID:    uuid.New(),
		Status:      models.ReportStatusRejected,
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

	_, err := service.HidePost(context.Background(), &models.User{ID: moderatorID, UserRole: constants.UserRoleAdmin}, 42, "violates rules")
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
