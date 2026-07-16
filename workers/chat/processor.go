package chatworker

import (
	"context"
	"log"
	"time"
)

const (
	expiryBatchSize = 100
	expiryInterval  = time.Second
)

type ExpirationService interface {
	ExpireMessages(ctx context.Context, now time.Time, limit int) (int, error)
}

func StartProcessor(service ExpirationService) {
	if service == nil {
		return
	}
	go runWorker(service)
}

func runWorker(service ExpirationService) {
	ticker := time.NewTicker(expiryInterval)
	defer ticker.Stop()

	for now := range ticker.C {
		if _, err := SweepOnce(context.Background(), service, now.UTC(), expiryBatchSize); err != nil {
			log.Printf("[ChatExpiry] sweep failed: %v", err)
		}
	}
}

// SweepOnce drains every currently due batch. It is exported so the worker can
// be exercised deterministically without waiting for a ticker in tests.
func SweepOnce(ctx context.Context, service ExpirationService, now time.Time, batchSize int) (int, error) {
	if service == nil || batchSize <= 0 {
		return 0, nil
	}

	total := 0
	for {
		count, err := service.ExpireMessages(ctx, now, batchSize)
		if err != nil {
			return total, err
		}
		total += count
		if count < batchSize {
			return total, nil
		}
	}
}
