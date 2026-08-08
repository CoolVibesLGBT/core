package ports

import (
	"context"
	domainmedia "core/domain/media"
	modelmedia "core/models/media"
	"time"

	"github.com/google/uuid"
)

type PrivatePhotoPrincipal struct {
	ID       uuid.UUID
	PublicID int64
}

type PrivatePhotoUser struct {
	ID          uuid.UUID
	PublicID    int64
	UserName    string
	DisplayName string
	Avatar      *modelmedia.Media
}

type PrivatePhotoAccessRecord struct {
	ID             uuid.UUID
	PublicID       int64
	OwnerID        uuid.UUID
	OwnerPublicID  int64
	ViewerID       uuid.UUID
	ViewerPublicID int64
	Status         domainmedia.PrivatePhotoAccessStatus
	RequestedAt    time.Time
	RespondedAt    *time.Time
	Viewer         PrivatePhotoUser
}

type PrivatePhotoRepository interface {
	FindPrivatePhotoUserByPublicID(ctx context.Context, publicID int64) (PrivatePhotoUser, error)
	FindPrivatePhotoUserByID(ctx context.Context, userID uuid.UUID) (PrivatePhotoUser, error)
	CountPrivatePhotos(ctx context.Context, ownerID uuid.UUID) (int64, error)
	ListPrivatePhotos(ctx context.Context, ownerID uuid.UUID) ([]modelmedia.Media, error)
	AddPrivatePhoto(ctx context.Context, ownerID uuid.UUID, file UploadedFile, maxCount int64) (*modelmedia.Media, error)
	DeletePrivatePhoto(ctx context.Context, ownerID uuid.UUID, photoPublicID int64) error
	ArePrivatePhotoUsersBlocked(ctx context.Context, firstUserID, secondUserID uuid.UUID) (bool, error)

	GetPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID) (*PrivatePhotoAccessRecord, error)
	RequestPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*PrivatePhotoAccessRecord, bool, error)
	FindPrivatePhotoAccessByPublicID(ctx context.Context, requestPublicID int64) (*PrivatePhotoAccessRecord, error)
	RespondPrivatePhotoAccess(ctx context.Context, requestPublicID int64, ownerID uuid.UUID, decision domainmedia.PrivatePhotoAccessStatus, now time.Time) (*PrivatePhotoAccessRecord, bool, error)
	RevokePrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID, now time.Time) (*PrivatePhotoAccessRecord, bool, error)
	ListPrivatePhotoAccessRequests(ctx context.Context, ownerID uuid.UUID) ([]PrivatePhotoAccessRecord, error)
}

// ProfilePhotoRepository owns the public/private album boundary. It is kept
// separate from PrivatePhotoRepository so access-only adapters do not need to
// know about public profile media.
type ProfilePhotoRepository interface {
	ListProfilePhotos(ctx context.Context, ownerID uuid.UUID) ([]modelmedia.Media, error)
	MoveProfilePhoto(ctx context.Context, ownerID uuid.UUID, photoPublicID int64, destination modelmedia.MediaRole, maxPrivateCount int64) (*modelmedia.Media, error)
}

// PrivatePhotoAccessAuthorizer is deliberately separate from
// MediaAccessRepository so existing media-access adapters can remain unaware
// of the private album workflow. The static file service denies access when
// this optional capability is absent.
type PrivatePhotoAccessAuthorizer interface {
	HasApprovedPrivatePhotoAccess(ctx context.Context, ownerID, viewerID uuid.UUID) (bool, error)
}

// PrivatePhotoBlockRevoker permanently removes grants in both directions when
// either user blocks the other. This prevents an old approval from silently
// becoming valid again after a later unblock.
type PrivatePhotoBlockRevoker interface {
	RevokePrivatePhotoAccessBetween(ctx context.Context, firstUserID, secondUserID uuid.UUID, now time.Time) error
}

type PrivatePhotoNotifier interface {
	NotifyPrivatePhotoAccessRequested(ctx context.Context, owner, viewer PrivatePhotoUser, request PrivatePhotoAccessRecord) error
	NotifyPrivatePhotoAccessResponded(ctx context.Context, owner, viewer PrivatePhotoUser, request PrivatePhotoAccessRecord) error
}
