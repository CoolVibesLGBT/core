package chatworker

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	expiryBatchSize = 100
	expiryInterval  = time.Second
)

type ExpirationService interface {
	ExpireMessages(ctx context.Context, now time.Time, limit int) (int, error)
}

type Processor struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func StartProcessor(service ExpirationService) *Processor {
	return StartProcessorContext(context.Background(), service)
}

func StartProcessorContext(parent context.Context, service ExpirationService) *Processor {
	if service == nil {
		return nil
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	processor := &Processor{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runWorker(ctx, service)
	}()
	go func() {
		wg.Wait()
		close(processor.done)
	}()

	return processor
}

func (p *Processor) Stop() {
	if p != nil && p.cancel != nil {
		p.cancel()
	}
}

func (p *Processor) Wait() {
	if p != nil && p.done != nil {
		<-p.done
	}
}

func (p *Processor) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.Stop()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func runWorker(ctx context.Context, service ExpirationService) {
	ticker := time.NewTicker(expiryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if _, err := SweepOnce(ctx, service, now.UTC(), expiryBatchSize); err != nil && ctx.Err() == nil {
				log.Printf("[ChatExpiry] sweep failed: %v", err)
			}
		}
	}
}

// SweepOnce drains every currently due batch. It is exported so the worker can
// be exercised deterministically without waiting for a ticker in tests.
func SweepOnce(ctx context.Context, service ExpirationService, now time.Time, batchSize int) (int, error) {
	if service == nil || batchSize <= 0 {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	total := 0
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
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
