package media

import (
	"context"
	mediamodel "core/models/media"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

type observerProcessorRepository struct {
	claimed atomic.Bool
}

func (r *observerProcessorRepository) ClaimNextPendingMedia(time.Time) (*mediamodel.Media, error) {
	if !r.claimed.CompareAndSwap(false, true) {
		return nil, nil
	}
	return &mediamodel.Media{ID: uuid.New(), PublicID: 55, Role: mediamodel.RolePrivatePhoto}, nil
}

func (*observerProcessorRepository) ProcessClaimedMedia(item *mediamodel.Media) error {
	item.ProcessingStatus = mediamodel.ProcessingStatusReady
	return nil
}

func (*observerProcessorRepository) RequeueStaleProcessing(time.Duration) (int64, error) {
	return 0, nil
}

type processingObserverFake struct {
	events chan mediamodel.ProcessingStatus
}

func (o *processingObserverFake) MediaProcessingUpdated(_ context.Context, _ *mediamodel.Media, status mediamodel.ProcessingStatus) {
	o.events <- status
}

func TestProcessorPublishesTerminalMediaStatus(t *testing.T) {
	repository := &observerProcessorRepository{}
	observer := &processingObserverFake{events: make(chan mediamodel.ProcessingStatus, 1)}
	processor := startProcessor(context.Background(), repository, observer)

	select {
	case status := <-observer.events:
		if status != mediamodel.ProcessingStatusReady {
			t.Fatalf("observer status = %q, want %q", status, mediamodel.ProcessingStatusReady)
		}
	case <-time.After(time.Second):
		t.Fatal("processor did not publish terminal media status")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := processor.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
