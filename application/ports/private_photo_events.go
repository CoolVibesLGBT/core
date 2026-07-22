package ports

import (
	"context"
	"time"
)

const (
	PrivatePhotoRealtimeVersion     = "1"
	PrivatePhotoRealtimeSocketEvent = "private_photo_changed"
)

type PrivatePhotoRealtimeEventType string

const (
	PrivatePhotoEventAlbumChanged           PrivatePhotoRealtimeEventType = "private_photos.album.changed"
	PrivatePhotoEventMediaProcessingUpdated PrivatePhotoRealtimeEventType = "private_photos.media.processing_updated"
	PrivatePhotoEventAccessRequested        PrivatePhotoRealtimeEventType = "private_photos.access.requested"
	PrivatePhotoEventAccessUpdated          PrivatePhotoRealtimeEventType = "private_photos.access.updated"
	PrivatePhotoEventAccessRevoked          PrivatePhotoRealtimeEventType = "private_photos.access.revoked"
	PrivatePhotoEventAccessInvalidated      PrivatePhotoRealtimeEventType = "private_photos.access.invalidated"
)

// PrivatePhotoRealtimeEnvelope is intentionally a narrow invalidation event.
// It contains public identifiers only: media URLs, storage paths, account
// UUIDs, and private-photo database UUIDs must never enter the socket payload.
type PrivatePhotoRealtimeEnvelope struct {
	Version    string                        `json:"version"`
	EventID    string                        `json:"event_id"`
	Type       PrivatePhotoRealtimeEventType `json:"type"`
	OccurredAt time.Time                     `json:"occurred_at"`
	Data       PrivatePhotoRealtimeEventData `json:"data"`
}

type PrivatePhotoRealtimeEventData struct {
	RequestID string `json:"request_id,omitempty"`
	OwnerID   string `json:"owner_id"`
	ViewerID  string `json:"viewer_id,omitempty"`
	PhotoID   string `json:"photo_id,omitempty"`
	Status    string `json:"status,omitempty"`
}

type PrivatePhotoRealtimePublisher interface {
	PublishPrivatePhotoEvent(ctx context.Context, recipientPublicIDs []int64, event PrivatePhotoRealtimeEnvelope) error
}
