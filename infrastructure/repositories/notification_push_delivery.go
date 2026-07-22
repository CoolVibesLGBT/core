package repositories

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	push "core/infrastructure/push"
	"core/models"
)

const (
	notificationPushBatchTimeout  = 3 * time.Second
	notificationPushMaxConcurrent = 4
)

// webPushSender is the outbound delivery boundary consumed by the notification
// persistence adapter. Keeping the interface here lets tests replace the
// external provider without coupling repository behavior to an HTTP client.
type webPushSender interface {
	SendContext(context.Context, *push.Push) error
}

type webPushSenderFactory func(*push.Options) (webPushSender, error)

func defaultWebPushSenderFactory(options *push.Options) (webPushSender, error) {
	return push.NewService(options)
}

func deliverWebPushBatch(ctx context.Context, sender webPushSender, subscriptions []models.Subscription, payload []byte, maxConcurrent int) (int, int) {
	if len(subscriptions) == 0 || sender == nil {
		return 0, 0
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if maxConcurrent > len(subscriptions) {
		maxConcurrent = len(subscriptions)
	}

	jobs := make(chan models.Subscription)
	var attempted atomic.Int64
	var failed atomic.Int64
	var workers sync.WaitGroup
	workers.Add(maxConcurrent)

	for range maxConcurrent {
		go func() {
			defer workers.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case sub, ok := <-jobs:
					if !ok {
						return
					}
					if ctx.Err() != nil {
						return
					}

					attempted.Add(1)
					if err := sender.SendContext(ctx, &push.Push{
						Endpoint:  sub.Endpoint,
						Auth:      sub.Keys.Auth,
						P256DH:    sub.Keys.P256dh,
						Plaintext: payload,
					}); err != nil {
						failed.Add(1)
					}
				}
			}
		}()
	}

sendSubscriptions:
	for _, sub := range subscriptions {
		select {
		case <-ctx.Done():
			break sendSubscriptions
		case jobs <- sub:
		}
	}
	close(jobs)
	workers.Wait()

	return int(attempted.Load()), int(failed.Load())
}
