package usecases

import (
	"context"
	"core/application/ports"
	"core/models"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestBlockPublishesBidirectionalPrivatePhotoInvalidations(t *testing.T) {
	blockerID := uuid.New()
	blockedID := uuid.New()
	userRepository := &fakeUserRepository{byPublicID: map[int64]*models.User{
		1: {ID: blockerID, PublicID: 1},
		2: {ID: blockedID, PublicID: 2},
	}}
	revoker := &privatePhotoBlockRevokerFake{}
	publisher := &privatePhotoRealtimePublisherFake{}
	service := NewUserService(
		userRepository,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithPrivatePhotoBlockRevoker(revoker),
		WithPrivatePhotoRealtimePublisher(publisher),
	)

	for attempt := 0; attempt < 2; attempt++ {
		enabled, err := service.Block(context.Background(), models.User{ID: blockerID, PublicID: 1}, 1, 2)
		if err != nil || !enabled {
			t.Fatalf("Block() attempt %d = %v, %v; want true, nil", attempt+1, enabled, err)
		}
	}
	if revoker.calls != 2 {
		t.Fatalf("revoker calls = %d, want 2", revoker.calls)
	}
	if len(publisher.events) != 4 {
		t.Fatalf("bidirectional invalidation events = %d, want 4 after two idempotent block calls", len(publisher.events))
	}

	for offset := 0; offset < len(publisher.events); offset += 2 {
		forward := publisher.events[offset]
		reverse := publisher.events[offset+1]
		for _, captured := range []capturedPrivatePhotoRealtimeEvent{forward, reverse} {
			if !reflect.DeepEqual(captured.recipients, []int64{1, 2}) {
				t.Fatalf("invalidation recipients = %v, want [1 2]", captured.recipients)
			}
			if captured.event.Version != ports.PrivatePhotoRealtimeVersion || captured.event.Type != ports.PrivatePhotoEventAccessInvalidated || captured.event.Data.Status != "denied" {
				t.Fatalf("invalidation event = %+v", captured.event)
			}
		}
		if forward.event.Data.OwnerID != "1" || forward.event.Data.ViewerID != "2" {
			t.Fatalf("forward invalidation data = %+v", forward.event.Data)
		}
		if reverse.event.Data.OwnerID != "2" || reverse.event.Data.ViewerID != "1" {
			t.Fatalf("reverse invalidation data = %+v", reverse.event.Data)
		}
	}
}

func TestBlockPublishesFailClosedInvalidationsWhenGrantCleanupFails(t *testing.T) {
	blockerID := uuid.New()
	blockedID := uuid.New()
	revokeErr := errors.New("private photo grant cleanup failed")
	userRepository := &fakeUserRepository{byPublicID: map[int64]*models.User{
		1: {ID: blockerID, PublicID: 1},
		2: {ID: blockedID, PublicID: 2},
	}}
	publisher := &privatePhotoRealtimePublisherFake{}
	service := NewUserService(
		userRepository,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithPrivatePhotoBlockRevoker(&privatePhotoBlockRevokerFake{err: revokeErr}),
		WithPrivatePhotoRealtimePublisher(publisher),
	)

	enabled, err := service.Block(context.Background(), models.User{ID: blockerID, PublicID: 1}, 1, 2)
	if !enabled || !errors.Is(err, revokeErr) {
		t.Fatalf("Block() = %v, %v; want true, cleanup error", enabled, err)
	}
	if len(publisher.events) != 2 {
		t.Fatalf("fail-closed invalidation events = %d, want 2", len(publisher.events))
	}
	if publisher.events[0].event.Type != ports.PrivatePhotoEventAccessInvalidated ||
		publisher.events[1].event.Type != ports.PrivatePhotoEventAccessInvalidated {
		t.Fatalf("fail-closed event types = %q, %q", publisher.events[0].event.Type, publisher.events[1].event.Type)
	}
}
