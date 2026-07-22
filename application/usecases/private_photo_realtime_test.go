package usecases

import (
	"bytes"
	"context"
	"core/application/ports"
	domainmedia "core/domain/media"
	modelmedia "core/models/media"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type privatePhotoRealtimeTestFile struct{}

func (privatePhotoRealtimeTestFile) Filename() string    { return "private.jpg" }
func (privatePhotoRealtimeTestFile) Size() int64         { return 1024 }
func (privatePhotoRealtimeTestFile) ContentType() string { return "image/jpeg" }
func (privatePhotoRealtimeTestFile) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

type privatePhotoRealtimeRepository struct {
	ports.PrivatePhotoRepository
	usersByPublicID map[int64]ports.PrivatePhotoUser
	access          *ports.PrivatePhotoAccessRecord
	audience        []ports.PrivatePhotoAccessRecord
}

func (r *privatePhotoRealtimeRepository) FindPrivatePhotoUserByPublicID(_ context.Context, publicID int64) (ports.PrivatePhotoUser, error) {
	user, ok := r.usersByPublicID[publicID]
	if !ok {
		return ports.PrivatePhotoUser{}, ports.ErrNotFound
	}
	return user, nil
}

func (r *privatePhotoRealtimeRepository) FindPrivatePhotoUserByID(_ context.Context, userID uuid.UUID) (ports.PrivatePhotoUser, error) {
	for _, user := range r.usersByPublicID {
		if user.ID == userID {
			return user, nil
		}
	}
	return ports.PrivatePhotoUser{}, ports.ErrNotFound
}

func (*privatePhotoRealtimeRepository) CountPrivatePhotos(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}

func (*privatePhotoRealtimeRepository) AddPrivatePhoto(_ context.Context, ownerID uuid.UUID, _ ports.UploadedFile, _ int64) (*modelmedia.Media, error) {
	return &modelmedia.Media{
		ID:               uuid.New(),
		PublicID:         501,
		OwnerID:          ownerID,
		UserID:           ownerID,
		Role:             modelmedia.RolePrivatePhoto,
		ProcessingStatus: modelmedia.ProcessingStatusPending,
	}, nil
}

func (*privatePhotoRealtimeRepository) DeletePrivatePhoto(context.Context, uuid.UUID, int64) error {
	return nil
}

func (*privatePhotoRealtimeRepository) ArePrivatePhotoUsersBlocked(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func (r *privatePhotoRealtimeRepository) RequestPrivatePhotoAccess(_ context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	if r.access != nil && r.access.Status == domainmedia.PrivatePhotoAccessPending {
		copy := *r.access
		return &copy, false, nil
	}
	owner, _ := r.FindPrivatePhotoUserByID(context.Background(), ownerID)
	viewer, _ := r.FindPrivatePhotoUserByID(context.Background(), viewerID)
	r.access = &ports.PrivatePhotoAccessRecord{
		ID:             uuid.New(),
		PublicID:       601,
		OwnerID:        ownerID,
		OwnerPublicID:  owner.PublicID,
		ViewerID:       viewerID,
		ViewerPublicID: viewer.PublicID,
		Status:         domainmedia.PrivatePhotoAccessPending,
		RequestedAt:    now,
		Viewer:         viewer,
	}
	copy := *r.access
	return &copy, true, nil
}

func (r *privatePhotoRealtimeRepository) FindPrivatePhotoAccessByPublicID(_ context.Context, requestPublicID int64) (*ports.PrivatePhotoAccessRecord, error) {
	if r.access == nil || r.access.PublicID != requestPublicID {
		return nil, ports.ErrNotFound
	}
	copy := *r.access
	return &copy, nil
}

func (r *privatePhotoRealtimeRepository) RespondPrivatePhotoAccess(_ context.Context, _ int64, _ uuid.UUID, decision domainmedia.PrivatePhotoAccessStatus, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	r.access.Status = decision
	r.access.RespondedAt = &now
	copy := *r.access
	return &copy, true, nil
}

func (r *privatePhotoRealtimeRepository) RevokePrivatePhotoAccess(_ context.Context, _ uuid.UUID, _ uuid.UUID, now time.Time) (*ports.PrivatePhotoAccessRecord, bool, error) {
	if r.access.Status == domainmedia.PrivatePhotoAccessDenied {
		copy := *r.access
		return &copy, false, nil
	}
	r.access.Status = domainmedia.PrivatePhotoAccessDenied
	r.access.RespondedAt = &now
	copy := *r.access
	return &copy, true, nil
}

func (r *privatePhotoRealtimeRepository) ListPrivatePhotoAccessRequests(context.Context, uuid.UUID) ([]ports.PrivatePhotoAccessRecord, error) {
	return append([]ports.PrivatePhotoAccessRecord(nil), r.audience...), nil
}

type capturedPrivatePhotoRealtimeEvent struct {
	recipients []int64
	event      ports.PrivatePhotoRealtimeEnvelope
}

type privatePhotoRealtimePublisherFake struct {
	events []capturedPrivatePhotoRealtimeEvent
}

func (p *privatePhotoRealtimePublisherFake) PublishPrivatePhotoEvent(_ context.Context, recipients []int64, event ports.PrivatePhotoRealtimeEnvelope) error {
	p.events = append(p.events, capturedPrivatePhotoRealtimeEvent{
		recipients: append([]int64(nil), recipients...),
		event:      event,
	})
	return nil
}

func TestPrivatePhotoRealtimeEventsAreVersionedSafeAndAudienceScoped(t *testing.T) {
	owner := ports.PrivatePhotoUser{ID: uuid.New(), PublicID: 10}
	viewer := ports.PrivatePhotoUser{ID: uuid.New(), PublicID: 20}
	approvedViewer := ports.PrivatePhotoUser{ID: uuid.New(), PublicID: 30}
	deniedViewer := ports.PrivatePhotoUser{ID: uuid.New(), PublicID: 40}
	repository := &privatePhotoRealtimeRepository{
		usersByPublicID: map[int64]ports.PrivatePhotoUser{
			owner.PublicID:          owner,
			viewer.PublicID:         viewer,
			approvedViewer.PublicID: approvedViewer,
			deniedViewer.PublicID:   deniedViewer,
		},
		audience: []ports.PrivatePhotoAccessRecord{
			{ViewerPublicID: viewer.PublicID, Status: domainmedia.PrivatePhotoAccessApproved},
			{ViewerPublicID: approvedViewer.PublicID, Status: domainmedia.PrivatePhotoAccessApproved},
			{ViewerPublicID: deniedViewer.PublicID, Status: domainmedia.PrivatePhotoAccessDenied},
			{ViewerPublicID: viewer.PublicID, Status: domainmedia.PrivatePhotoAccessApproved},
		},
	}
	publisher := &privatePhotoRealtimePublisherFake{}
	service := NewPrivatePhotoService(repository, nil, publisher)
	ownerPrincipal := ports.PrivatePhotoPrincipal{ID: owner.ID, PublicID: owner.PublicID}
	viewerPrincipal := ports.PrivatePhotoPrincipal{ID: viewer.ID, PublicID: viewer.PublicID}
	ctx := context.Background()

	if _, err := service.Upload(ctx, ownerPrincipal, []ports.UploadedFile{privatePhotoRealtimeTestFile{}}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	assertPrivatePhotoRealtimeEvent(t, publisher.events[0], []int64{10, 20, 30}, ports.PrivatePhotoEventAlbumChanged, "")

	if err := service.Delete(ctx, ownerPrincipal, 501); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	assertPrivatePhotoRealtimeEvent(t, publisher.events[1], []int64{10, 20, 30}, ports.PrivatePhotoEventAlbumChanged, "")

	if _, err := service.RequestAccess(ctx, viewerPrincipal, owner.PublicID); err != nil {
		t.Fatalf("RequestAccess() error = %v", err)
	}
	assertPrivatePhotoRealtimeEvent(t, publisher.events[2], []int64{10, 20}, ports.PrivatePhotoEventAccessRequested, "pending")
	if _, err := service.RequestAccess(ctx, viewerPrincipal, owner.PublicID); err != nil {
		t.Fatalf("idempotent RequestAccess() error = %v", err)
	}
	if len(publisher.events) != 3 {
		t.Fatalf("idempotent request emitted %d events; want 3 total", len(publisher.events))
	}

	if _, err := service.RespondAccess(ctx, ownerPrincipal, 601, "approved"); err != nil {
		t.Fatalf("RespondAccess() error = %v", err)
	}
	assertPrivatePhotoRealtimeEvent(t, publisher.events[3], []int64{10, 20}, ports.PrivatePhotoEventAccessUpdated, "approved")

	if _, err := service.RevokeAccess(ctx, ownerPrincipal, viewer.PublicID); err != nil {
		t.Fatalf("RevokeAccess() error = %v", err)
	}
	assertPrivatePhotoRealtimeEvent(t, publisher.events[4], []int64{10, 20}, ports.PrivatePhotoEventAccessRevoked, "denied")

	repository.audience[0].Status = domainmedia.PrivatePhotoAccessDenied
	repository.audience[3].Status = domainmedia.PrivatePhotoAccessDenied
	service.MediaProcessingUpdated(ctx, &modelmedia.Media{
		ID:       uuid.New(),
		PublicID: 501,
		OwnerID:  owner.ID,
		UserID:   owner.ID,
		Role:     modelmedia.RolePrivatePhoto,
	}, modelmedia.ProcessingStatusReady)
	assertPrivatePhotoRealtimeEvent(t, publisher.events[5], []int64{10, 30}, ports.PrivatePhotoEventMediaProcessingUpdated, "ready")
	if publisher.events[5].event.Data.PhotoID != "501" {
		t.Fatalf("processing event photo ID = %q, want 501", publisher.events[5].event.Data.PhotoID)
	}

	for _, captured := range publisher.events {
		encoded, err := json.Marshal(captured.event)
		if err != nil {
			t.Fatalf("marshal event: %v", err)
		}
		payload := string(encoded)
		for _, forbidden := range []string{"storage_path", `"url"`, owner.ID.String(), viewer.ID.String(), approvedViewer.ID.String()} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("realtime event leaked %q: %s", forbidden, payload)
			}
		}
	}
}

func assertPrivatePhotoRealtimeEvent(
	t *testing.T,
	captured capturedPrivatePhotoRealtimeEvent,
	wantRecipients []int64,
	wantType ports.PrivatePhotoRealtimeEventType,
	wantStatus string,
) {
	t.Helper()
	if !reflect.DeepEqual(captured.recipients, wantRecipients) {
		t.Fatalf("recipients = %v, want %v", captured.recipients, wantRecipients)
	}
	if captured.event.Version != ports.PrivatePhotoRealtimeVersion || captured.event.Type != wantType {
		t.Fatalf("event version/type = %q/%q, want %q/%q", captured.event.Version, captured.event.Type, ports.PrivatePhotoRealtimeVersion, wantType)
	}
	if _, err := uuid.Parse(captured.event.EventID); err != nil {
		t.Fatalf("event ID %q is not an ephemeral UUID: %v", captured.event.EventID, err)
	}
	if captured.event.OccurredAt.IsZero() || captured.event.Data.OwnerID != "10" || captured.event.Data.Status != wantStatus {
		t.Fatalf("event envelope = %+v", captured.event)
	}
}
