package usecases

import (
	"context"
	"errors"
	"testing"

	"core/models"

	"github.com/google/uuid"
)

func TestUserServiceReportDelegatesAndPreventsSelfReport(t *testing.T) {
	repo := &fakeUserRepository{}
	service := NewUserService(repo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeEngagementRepository{}, &fakeNotificationRepository{})
	authUser := &models.User{ID: uuid.New(), PublicID: 10}

	if err := service.Report(context.Background(), 20, models.ReportKindFakeProfile, "copied profile", authUser); err != nil {
		t.Fatalf("Report() error = %v", err)
	}
	if repo.reportUserPublicID != 20 || repo.reportKind != models.ReportKindFakeProfile || repo.reportDescription != "copied profile" || repo.reportAuthUser != authUser {
		t.Fatalf("repository arguments = %#v", repo)
	}

	if err := service.Report(context.Background(), authUser.PublicID, models.ReportKindOther, "", authUser); !errors.Is(err, ErrCannotReportSelf) {
		t.Fatalf("self Report() error = %v, want %v", err, ErrCannotReportSelf)
	}
	if repo.reportUserPublicID != 20 {
		t.Fatal("self-report should not call the repository")
	}
}
