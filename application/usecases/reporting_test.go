package usecases

import (
	"context"
	domainmoderation "core/domain/moderation"
	"core/models"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPostAndUserReportsShareDomainNormalization(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}

	postRepo := &fakePostRepository{}
	postService := NewPostService(&fakeUserRepository{}, postRepo, &fakeMediaRepository{})
	if err := postService.Report(context.Background(), 25, "  spam  ", "  details  ", authUser); err != nil {
		t.Fatalf("post Report() error = %v", err)
	}
	if postRepo.reportKind != "spam" || postRepo.reportDescription != "details" {
		t.Fatalf("post repository fields = %q, %q", postRepo.reportKind, postRepo.reportDescription)
	}

	userRepo := &fakeUserRepository{}
	userService := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeEngagementRepository{}, &fakeNotificationRepository{})
	if err := userService.Report(context.Background(), 20, "  fake_profile  ", "  copied  ", authUser); err != nil {
		t.Fatalf("user Report() error = %v", err)
	}
	if userRepo.reportKind != "fake_profile" || userRepo.reportDescription != "copied" {
		t.Fatalf("user repository fields = %q, %q", userRepo.reportKind, userRepo.reportDescription)
	}
}

func TestPostAndUserReportsRejectTheSameInvalidFieldsBeforeRepositories(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	tests := []struct {
		name        string
		kind        string
		description string
		wantErr     error
	}{
		{name: "missing kind", kind: "  ", wantErr: domainmoderation.ErrInvalidKind},
		{name: "long kind", kind: strings.Repeat("x", domainmoderation.MaxKindLength+1), wantErr: domainmoderation.ErrInvalidKind},
		{name: "long description", kind: "spam", description: strings.Repeat("x", domainmoderation.MaxDescriptionLength+1), wantErr: domainmoderation.ErrInvalidDescription},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postRepo := &fakePostRepository{}
			postService := NewPostService(&fakeUserRepository{}, postRepo, &fakeMediaRepository{})
			if err := postService.Report(context.Background(), 25, tt.kind, tt.description, authUser); !errors.Is(err, tt.wantErr) {
				t.Fatalf("post Report() error = %v, want %v", err, tt.wantErr)
			}
			if postRepo.reportPostID != 0 {
				t.Fatal("post repository was called for invalid report fields")
			}

			userRepo := &fakeUserRepository{}
			userService := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeEngagementRepository{}, &fakeNotificationRepository{})
			if err := userService.Report(context.Background(), 20, tt.kind, tt.description, authUser); !errors.Is(err, tt.wantErr) {
				t.Fatalf("user Report() error = %v, want %v", err, tt.wantErr)
			}
			if userRepo.reportUserPublicID != 0 {
				t.Fatal("user repository was called for invalid report fields")
			}
		})
	}
}
