package media

import (
	"context"
	mediamodel "core/models/media"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type idleProcessorRepository struct {
	started chan struct{}
	once    sync.Once
	claims  atomic.Int64
}

func (r *idleProcessorRepository) ClaimNextPendingMedia(time.Time) (*mediamodel.Media, error) {
	r.claims.Add(1)
	r.once.Do(func() { close(r.started) })
	return nil, nil
}

func (*idleProcessorRepository) ProcessClaimedMedia(*mediamodel.Media) error { return nil }

func (*idleProcessorRepository) RequeueStaleProcessing(time.Duration) (int64, error) {
	return 0, nil
}

func TestProcessorShutdownInterruptsIdleWait(t *testing.T) {
	repo := &idleProcessorRepository{started: make(chan struct{})}
	processor := startProcessor(context.Background(), repo)

	select {
	case <-repo.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := processor.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	select {
	case <-processor.done:
	default:
		t.Fatal("processor did not close its done channel")
	}
}

type blockingProcessorRepository struct {
	claimed atomic.Bool
	started chan struct{}
	release chan struct{}
}

func (r *blockingProcessorRepository) ClaimNextPendingMedia(time.Time) (*mediamodel.Media, error) {
	if r.claimed.CompareAndSwap(false, true) {
		return &mediamodel.Media{}, nil
	}
	return nil, nil
}

func (r *blockingProcessorRepository) ProcessClaimedMedia(*mediamodel.Media) error {
	close(r.started)
	<-r.release
	return nil
}

func (*blockingProcessorRepository) RequeueStaleProcessing(time.Duration) (int64, error) {
	return 0, nil
}

func TestProcessorShutdownHonorsDeadlineForActiveJob(t *testing.T) {
	repo := &blockingProcessorRepository{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	processor := startProcessor(context.Background(), repo)

	select {
	case <-repo.started:
	case <-time.After(time.Second):
		t.Fatal("processor did not claim a job")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := processor.Shutdown(ctx); err != context.DeadlineExceeded {
		t.Fatalf("Shutdown() error = %v, want %v", err, context.DeadlineExceeded)
	}

	close(repo.release)
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Second)
	defer cleanupCancel()
	if err := processor.Shutdown(cleanupCtx); err != nil {
		t.Fatalf("cleanup Shutdown() error = %v", err)
	}
}
