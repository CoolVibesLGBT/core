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
