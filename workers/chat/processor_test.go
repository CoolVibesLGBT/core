package chatworker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeExpirationService struct {
	counts []int
	err    error
	calls  int
	now    time.Time
	limit  int
}

func (s *fakeExpirationService) ExpireMessages(_ context.Context, now time.Time, limit int) (int, error) {
	s.calls++
	s.now = now
	s.limit = limit
	if s.err != nil {
		return 0, s.err
	}
	if len(s.counts) == 0 {
		return 0, nil
	}
	count := s.counts[0]
	s.counts = s.counts[1:]
	return count, nil
}

func TestSweepOnceDrainsFullBatches(t *testing.T) {
	now := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	service := &fakeExpirationService{counts: []int{2, 2, 1}}

	count, err := SweepOnce(context.Background(), service, now, 2)
	if err != nil {
		t.Fatalf("SweepOnce() error = %v", err)
	}
	if count != 5 || service.calls != 3 {
		t.Fatalf("SweepOnce() count=%d calls=%d, want count=5 calls=3", count, service.calls)
	}
	if !service.now.Equal(now) || service.limit != 2 {
		t.Fatalf("unexpected sweep args now=%v limit=%d", service.now, service.limit)
	}
}

func TestSweepOnceReturnsServiceError(t *testing.T) {
	want := errors.New("database unavailable")
	service := &fakeExpirationService{err: want}

	if _, err := SweepOnce(context.Background(), service, time.Now(), 100); !errors.Is(err, want) {
		t.Fatalf("SweepOnce() error = %v, want %v", err, want)
	}
}

func TestSweepOnceStopsBeforeNextBatchWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &fakeExpirationService{counts: []int{100}}

	count, err := SweepOnce(ctx, service, time.Now(), 100)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SweepOnce() error = %v, want %v", err, context.Canceled)
	}
	if count != 0 || service.calls != 0 {
		t.Fatalf("SweepOnce() count=%d calls=%d, want both 0", count, service.calls)
	}
}

func TestProcessorShutdownInterruptsTickerWait(t *testing.T) {
	processor := StartProcessor(&fakeExpirationService{})
	if processor == nil {
		t.Fatal("StartProcessor() returned nil")
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

func TestProcessorStopsWhenParentContextIsCanceled(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	processor := StartProcessorContext(parent, &fakeExpirationService{})
	cancelParent()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := processor.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestNilProcessorShutdownIsSafe(t *testing.T) {
	if processor := StartProcessor(nil); processor != nil {
		t.Fatalf("StartProcessor(nil) = %#v, want nil", processor)
	}

	var processor *Processor
	if err := processor.Shutdown(context.Background()); err != nil {
		t.Fatalf("nil Shutdown() error = %v", err)
	}
}
