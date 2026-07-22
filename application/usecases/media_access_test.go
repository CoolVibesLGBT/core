package usecases

import (
	"context"
	"core/application/ports"
	"testing"

	"github.com/google/uuid"
)

type mediaAccessRepositoryFake struct {
	access    ports.MediaFileAccess
	principal ports.MediaAccessPrincipal
}

func (r *mediaAccessRepositoryFake) FindMediaFileAccess(context.Context, string) (ports.MediaFileAccess, error) {
	return r.access, nil
}

func (r *mediaAccessRepositoryFake) FindMediaAccessPrincipal(context.Context, int64) (ports.MediaAccessPrincipal, error) {
	return r.principal, nil
}

func (*mediaAccessRepositoryFake) IsActiveChatParticipant(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

type privatePhotoAuthorizerFake struct {
	approved bool
	calls    int
}

func (a *privatePhotoAuthorizerFake) HasApprovedPrivatePhotoAccess(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	a.calls++
	return a.approved, nil
}

func TestMediaAccessPrivatePhotoGrantCoversOnlyProcessedVariants(t *testing.T) {
	ownerID := uuid.New()
	viewerID := uuid.New()
	repository := &mediaAccessRepositoryFake{
		access: ports.MediaFileAccess{
			StoragePath: "./static/uploads/users/owner/private/photo.jpg",
			OwnerID:     ownerID,
			OwnerType:   "user",
			Role:        "private_photo",
			IsPublic:    false,
		},
		principal: ports.MediaAccessPrincipal{ID: viewerID, Role: "user"},
	}
	authorizer := &privatePhotoAuthorizerFake{approved: true}
	service := NewMediaAccessService(repository, authorizer)
	viewerPublicID := int64(20)

	decision, err := service.Authorize(
		context.Background(),
		"./static/uploads/users/owner/private/photo",
		&viewerPublicID,
		"static/uploads/users/owner/private/photo_landscape_lg.webp",
	)
	if err != nil || !decision.Allowed || decision.Public {
		t.Fatalf("Authorize(processed, approved) = %+v, %v", decision, err)
	}

	decision, err = service.Authorize(
		context.Background(),
		"./static/uploads/users/owner/private/photo",
		&viewerPublicID,
		"static/uploads/users/owner/private/photo.jpg",
	)
	if err != nil || decision.Allowed {
		t.Fatalf("Authorize(raw, approved) = %+v, %v; want denied", decision, err)
	}

	authorizer.approved = false
	decision, err = service.Authorize(
		context.Background(),
		"./static/uploads/users/owner/private/photo",
		&viewerPublicID,
		"static/uploads/users/owner/private/photo_landscape_lg.webp",
	)
	if err != nil || decision.Allowed {
		t.Fatalf("Authorize(processed, revoked) = %+v, %v; want denied", decision, err)
	}
}

func TestMediaAccessPrivatePhotoOwnerCanDeleteOrInspectRawUpload(t *testing.T) {
	ownerID := uuid.New()
	repository := &mediaAccessRepositoryFake{
		access: ports.MediaFileAccess{
			StoragePath: "./static/uploads/users/owner/private/photo.jpg",
			OwnerID:     ownerID,
			OwnerType:   "user",
			Role:        "private_photo",
		},
		principal: ports.MediaAccessPrincipal{ID: ownerID, Role: "user"},
	}
	service := NewMediaAccessService(repository, &privatePhotoAuthorizerFake{})
	ownerPublicID := int64(10)

	decision, err := service.Authorize(
		context.Background(),
		"./static/uploads/users/owner/private/photo",
		&ownerPublicID,
		"static/uploads/users/owner/private/photo.jpg",
	)
	if err != nil || !decision.Allowed {
		t.Fatalf("Authorize(raw, owner) = %+v, %v; want allowed", decision, err)
	}
}
