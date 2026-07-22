package repositories

import (
	"context"
	"core/infrastructure/push"
	"core/models"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

type trackingWebPushSender struct {
	send      func(context.Context, *push.Push) error
	active    atomic.Int64
	maxActive atomic.Int64
	calls     atomic.Int64
}

func (s *trackingWebPushSender) SendContext(ctx context.Context, message *push.Push) error {
	active := s.active.Add(1)
	defer s.active.Add(-1)
	s.calls.Add(1)
	for {
		maximum := s.maxActive.Load()
		if active <= maximum || s.maxActive.CompareAndSwap(maximum, active) {
			break
		}
	}
	return s.send(ctx, message)
}

func pushSubscriptions(count int) []models.Subscription {
	items := make([]models.Subscription, count)
	for i := range items {
		items[i].Endpoint = fmt.Sprintf("https://push.example.test/%d", i)
	}
	return items
}

func TestDeliverWebPushBatchBoundsConcurrency(t *testing.T) {
	sender := &trackingWebPushSender{
		send: func(context.Context, *push.Push) error {
			time.Sleep(15 * time.Millisecond)
			return nil
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	attempted, failed := deliverWebPushBatch(ctx, sender, pushSubscriptions(12), []byte("payload"), 3)

	if attempted != 12 || failed != 0 {
		t.Fatalf("deliverWebPushBatch() = attempted %d, failed %d; want 12, 0", attempted, failed)
	}
	if maximum := sender.maxActive.Load(); maximum > 3 {
		t.Fatalf("maximum concurrent deliveries = %d; want <= 3", maximum)
	}
	if maximum := sender.maxActive.Load(); maximum < 2 {
		t.Fatalf("maximum concurrent deliveries = %d; worker pool did not fan out", maximum)
	}
}

func TestDeliverWebPushBatchHonorsTotalDeadline(t *testing.T) {
	sender := &trackingWebPushSender{
		send: func(ctx context.Context, _ *push.Push) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()

	attempted, failed := deliverWebPushBatch(ctx, sender, pushSubscriptions(20), []byte("payload"), 2)
	elapsed := time.Since(started)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("batch returned after %s; total deadline was not respected", elapsed)
	}
	if attempted == 0 || attempted > 2 {
		t.Fatalf("attempted deliveries = %d; want between 1 and worker limit 2", attempted)
	}
	if failed != attempted {
		t.Fatalf("failed deliveries = %d; want attempted count %d", failed, attempted)
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("context error = %v; want deadline exceeded", ctx.Err())
	}
	if maximum := sender.maxActive.Load(); maximum > 2 {
		t.Fatalf("maximum concurrent deliveries = %d; want <= 2", maximum)
	}
}
