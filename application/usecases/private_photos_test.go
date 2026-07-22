package usecases

import (
	"bytes"
	"context"
	"core/application/ports"
	domainmedia "core/domain/media"
	modelmedia "core/models/media"
	modelutils "core/models/utils"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type privatePhotoTestFile struct {
	size int64
}

func (f privatePhotoTestFile) Filename() string    { return "photo.jpg" }
func (f privatePhotoTestFile) Size() int64         { return f.size }
func (f privatePhotoTestFile) ContentType() string { return "image/jpeg" }
func (f privatePhotoTestFile) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

type privatePhotoRepositoryFake struct {
	ports.PrivatePhotoRepository
	users          map[int64]ports.PrivatePhotoUser
	count          int64
	countCalls     int
	blocked        bool
	access         *ports.PrivatePhotoAccessRecord
	photos         []modelmedia.Media
	listCalls      int
	addCalls       int
	addedSize      int64
	respondCalls   int
	deletedOwner   uuid.UUID
	deletedPhotoID int64
}

func (r *privatePhotoRepositoryFake) FindPrivatePhotoUserByPublicID(_ context.Context, publicID int64) (ports.PrivatePhotoUser, error) {
	user, ok := r.users[publicID]
	if !ok {
		return ports.PrivatePhotoUser{}, ports.ErrNotFound
	}
	return user, nil
}

func (r *privatePhotoRepositoryFake) CountPrivatePhotos(context.Context, uuid.UUID) (int64, error) {
	r.countCalls++
	return r.count, nil
}

func (r *privatePhotoRepositoryFake) ListPrivatePhotos(context.Context, uuid.UUID) ([]modelmedia.Media, error) {
	r.listCalls++
	return r.photos, nil
}

func (r *privatePhotoRepositoryFake) AddPrivatePhoto(_ context.Context, ownerID uuid.UUID, file ports.UploadedFile, _ int64) (*modelmedia.Media, error) {
	r.addCalls++
	r.addedSize = file.Size()
	return &modelmedia.Media{ID: uuid.New(), OwnerID: ownerID, PublicID: 1}, nil
}

func (r *privatePhotoRepositoryFake) DeletePrivatePhoto(_ context.Context, ownerID uuid.UUID, photoPublicID int64) error {
	r.deletedOwner = ownerID
	r.deletedPhotoID = photoPublicID
	return nil
}

func (r *privatePhotoRepositoryFake) ArePrivatePhotoUsersBlocked(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return r.blocked, nil
}

func (r *privatePhotoRepositoryFake) GetPrivatePhotoAccess(context.Context, uuid.UUID, uuid.UUID) (*ports.PrivatePhotoAccessRecord, error) {
	if r.access == nil {
		return nil, ports.ErrNotFound
	}
	copy := *r.access
	return &copy, nil
}

func (r *privatePhotoRepositoryFake) RequestPrivatePhotoAccess(_ context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	changed := false
	if r.access == nil {
		r.access = &ports.PrivatePhotoAccessRecord{
			ID:          uuid.New(),
			PublicID:    77,
			OwnerID:     ownerID,
			ViewerID:    viewerID,
			Status:      domainmedia.PrivatePhotoAccessPending,
			RequestedAt: now,
			Viewer:      privatePhotoViewer(r.users, viewerID),
		}
		changed = true
	} else if r.access.Status == domainmedia.PrivatePhotoAccessDenied {
		r.access.Status = domainmedia.PrivatePhotoAccessPending
		r.access.RequestedAt = now
		changed = true
	}
	copy := *r.access
	return &copy, changed, nil
}

func (r *privatePhotoRepositoryFake) FindPrivatePhotoAccessByPublicID(_ context.Context, requestPublicID int64) (*ports.PrivatePhotoAccessRecord, error) {
	if r.access == nil || r.access.PublicID != requestPublicID {
		return nil, ports.ErrNotFound
	}
	copy := *r.access
	return &copy, nil
}

func (r *privatePhotoRepositoryFake) RespondPrivatePhotoAccess(_ context.Context, _ int64, _ uuid.UUID, decision domainmedia.PrivatePhotoAccessStatus, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	r.respondCalls++
	r.access.Status = decision
	r.access.RespondedAt = &now
	copy := *r.access
	return &copy, true, nil
}

func privatePhotoViewer(users map[int64]ports.PrivatePhotoUser, viewerID uuid.UUID) ports.PrivatePhotoUser {
	for _, user := range users {
		if user.ID == viewerID {
			return user
		}
	}
	return ports.PrivatePhotoUser{}
}

type privatePhotoNotifierFake struct {
	requested int
}

func (n *privatePhotoNotifierFake) NotifyPrivatePhotoAccessRequested(context.Context, ports.PrivatePhotoUser, ports.PrivatePhotoUser, ports.PrivatePhotoAccessRecord) error {
	n.requested++
	return nil
}

func (*privatePhotoNotifierFake) NotifyPrivatePhotoAccessResponded(context.Context, ports.PrivatePhotoUser, ports.PrivatePhotoUser, ports.PrivatePhotoAccessRecord) error {
	return nil
}

func TestPrivatePhotoFetchRequiresApprovalAndReturnsSafeMediaDTO(t *testing.T) {
	ownerID := uuid.New()
	viewerID := uuid.New()
	owner := ports.PrivatePhotoUser{ID: ownerID, PublicID: 10}
	largeURL := "/static/uploads/users/owner/private/photo_landscape_lg.webp"
	rawURL := "/static/uploads/users/owner/private/photo.jpg"
	repository := &privatePhotoRepositoryFake{
		users: map[int64]ports.PrivatePhotoUser{10: owner},
		count: 1,
		photos: []modelmedia.Media{{
			ID:               uuid.New(),
			PublicID:         99,
			FileID:           uuid.New(),
			OwnerID:          ownerID,
			UserID:           ownerID,
			Role:             modelmedia.RolePrivatePhoto,
			ProcessingStatus: modelmedia.ProcessingStatusReady,
			File: modelutils.FileMetadata{
				ID:          uuid.New(),
				URL:         rawURL,
				StoragePath: "." + rawURL,
				Name:        "original-secret.jpg",
				MimeType:    "image/jpeg",
				Variants: &modelutils.FileVariants{Image: &modelutils.ImageVariants{
					Large:    &modelutils.VariantInfo{URL: largeURL, Format: "webp"},
					Original: &modelutils.VariantInfo{URL: rawURL, Format: "jpg"},
				}},
			},
		}, {
			PublicID:         100,
			OwnerID:          ownerID,
			UserID:           ownerID,
			Role:             modelmedia.RolePrivatePhoto,
			ProcessingStatus: modelmedia.ProcessingStatusFailed,
		}},
	}
	service := NewPrivatePhotoService(repository, nil, nil)
	principal := ports.PrivatePhotoPrincipal{ID: viewerID, PublicID: 20}

	result, err := service.Fetch(context.Background(), principal, owner.PublicID)
	if err != nil {
		t.Fatalf("Fetch(unapproved) error = %v", err)
	}
	if result.Access.Status != "none" || len(result.Photos) != 0 || repository.listCalls != 0 {
		t.Fatalf("unapproved result = %+v, listCalls=%d", result, repository.listCalls)
	}

	repository.access = &ports.PrivatePhotoAccessRecord{
		PublicID: 77,
		OwnerID:  ownerID,
		ViewerID: viewerID,
		Status:   domainmedia.PrivatePhotoAccessApproved,
	}
	result, err = service.Fetch(context.Background(), principal, owner.PublicID)
	if err != nil {
		t.Fatalf("Fetch(approved) error = %v", err)
	}
	if result.Count != 1 || len(result.Photos) != 1 || result.Photos[0].PublicID != "99" || result.Photos[0].File.URL != largeURL {
		t.Fatalf("approved safe result = %+v", result)
	}
	encoded, err := json.Marshal(result.Photos[0])
	if err != nil {
		t.Fatal(err)
	}
	response := string(encoded)
	for _, forbidden := range []string{"storage_path", "file_id", "owner_id", "user_id", "original-secret.jpg", rawURL, `"original"`} {
		if strings.Contains(response, forbidden) {
			t.Fatalf("safe media response leaked %q: %s", forbidden, response)
		}
	}

	ownerResult, err := service.Fetch(context.Background(), ports.PrivatePhotoPrincipal{ID: ownerID, PublicID: 10}, owner.PublicID)
	if err != nil {
		t.Fatalf("Fetch(owner) error = %v", err)
	}
	if ownerResult.Count != 2 || len(ownerResult.Photos) != 2 || ownerResult.Photos[1].ProcessingStatus != modelmedia.ProcessingStatusFailed {
		t.Fatalf("owner cannot inspect/delete failed quota item: %+v", ownerResult)
	}
}

func TestPrivatePhotoFetchFailsClosedForBlockedUsersBeforeCounting(t *testing.T) {
	ownerID := uuid.New()
	viewerID := uuid.New()
	repository := &privatePhotoRepositoryFake{
		users:   map[int64]ports.PrivatePhotoUser{10: {ID: ownerID, PublicID: 10}},
		blocked: true,
		count:   12,
	}
	service := NewPrivatePhotoService(repository, nil, nil)

	_, err := service.Fetch(context.Background(), ports.PrivatePhotoPrincipal{ID: viewerID, PublicID: 20}, 10)
	if !errors.Is(err, ErrPrivatePhotoForbidden) {
		t.Fatalf("Fetch(blocked) error = %v; want ErrPrivatePhotoForbidden", err)
	}
	if repository.countCalls != 0 || repository.listCalls != 0 {
		t.Fatalf("blocked fetch leaked album state: countCalls=%d listCalls=%d", repository.countCalls, repository.listCalls)
	}
}

func TestPrivatePhotoRequestIsIdempotentWhilePending(t *testing.T) {
	ownerID := uuid.New()
	viewerID := uuid.New()
	repository := &privatePhotoRepositoryFake{users: map[int64]ports.PrivatePhotoUser{
		10: {ID: ownerID, PublicID: 10, UserName: "owner"},
		20: {ID: viewerID, PublicID: 20, UserName: "viewer"},
	}}
	notifier := &privatePhotoNotifierFake{}
	service := NewPrivatePhotoService(repository, notifier, nil)
	principal := ports.PrivatePhotoPrincipal{ID: viewerID, PublicID: 20}

	first, err := service.RequestAccess(context.Background(), principal, 10)
	if err != nil {
		t.Fatalf("first RequestAccess() error = %v", err)
	}
	second, err := service.RequestAccess(context.Background(), principal, 10)
	if err != nil {
		t.Fatalf("second RequestAccess() error = %v", err)
	}
	if first.Status != "pending" || second.Status != "pending" || first.RequestID == nil || second.RequestID == nil || *first.RequestID != *second.RequestID {
		t.Fatalf("request results are not idempotent: first=%+v second=%+v", first, second)
	}
	if notifier.requested != 1 {
		t.Fatalf("idempotent request sent %d notifications; want 1", notifier.requested)
	}
}

func TestPrivatePhotoOwnerOnlyMutationsUseAuthenticatedOwner(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	repository := &privatePhotoRepositoryFake{
		users: map[int64]ports.PrivatePhotoUser{10: {ID: ownerID, PublicID: 10}},
		access: &ports.PrivatePhotoAccessRecord{
			PublicID: 44,
			OwnerID:  ownerID,
			ViewerID: uuid.New(),
			Status:   domainmedia.PrivatePhotoAccessPending,
		},
	}
	service := NewPrivatePhotoService(repository, nil, nil)

	_, err := service.RespondAccess(context.Background(), ports.PrivatePhotoPrincipal{ID: otherID, PublicID: 20}, 44, "approved")
	if !errors.Is(err, ErrPrivatePhotoForbidden) || repository.respondCalls != 0 {
		t.Fatalf("RespondAccess(non-owner) error=%v respondCalls=%d", err, repository.respondCalls)
	}

	principal := ports.PrivatePhotoPrincipal{ID: ownerID, PublicID: 10}
	if err := service.Delete(context.Background(), principal, 55); err != nil {
		t.Fatalf("Delete(owner) error = %v", err)
	}
	if repository.deletedOwner != ownerID || repository.deletedPhotoID != 55 {
		t.Fatalf("Delete() target owner=%s photo=%d", repository.deletedOwner, repository.deletedPhotoID)
	}
}

func TestPrivatePhotoUploadHasNoApplicationByteSizeLimit(t *testing.T) {
	repository := &privatePhotoRepositoryFake{}
	service := NewPrivatePhotoService(repository, nil, nil)
	principal := ports.PrivatePhotoPrincipal{ID: uuid.New(), PublicID: 10}
	largeFileSize := int64(128 << 20)

	photos, err := service.Upload(context.Background(), principal, []ports.UploadedFile{
		privatePhotoTestFile{size: largeFileSize},
	})
	if err != nil {
		t.Fatalf("Upload(128 MiB) error = %v", err)
	}
	if len(photos) != 1 || repository.addCalls != 1 || repository.addedSize != largeFileSize {
		t.Fatalf("large upload was not persisted: photos=%d addCalls=%d size=%d", len(photos), repository.addCalls, repository.addedSize)
	}
}
