package media

import (
	"context"
	"core/helpers"
	"core/infrastructure/repositories"
	mediamodel "core/models/media"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	processorWorkerCount = 2
	processorIdleDelay   = time.Second
	staleProcessingAfter = 10 * time.Minute
	staleSweepInterval   = time.Minute
)

type processorRepository interface {
	ClaimNextPendingMedia(time.Time) (*mediamodel.Media, error)
	ProcessClaimedMedia(*mediamodel.Media) error
	RequeueStaleProcessing(time.Duration) (int64, error)
}

type Processor struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func StartProcessor(db *gorm.DB, node *helpers.Node) *Processor {
	return StartProcessorContext(context.Background(), db, node)
}

func StartProcessorContext(ctx context.Context, db *gorm.DB, node *helpers.Node) *Processor {
	repo := repositories.NewMediaRepository(db, node)
	return startProcessor(ctx, repo)
}

func startProcessor(parent context.Context, repo processorRepository) *Processor {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	processor := &Processor{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	var wg sync.WaitGroup
	wg.Add(processorWorkerCount + 1)

	for i := 0; i < processorWorkerCount; i++ {
		go func() {
			defer wg.Done()
			runWorker(ctx, repo)
		}()
	}

	go func() {
		defer wg.Done()
		runStaleSweep(ctx, repo)
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

func runWorker(ctx context.Context, repo processorRepository) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		item, err := repo.ClaimNextPendingMedia(time.Now())
		if err != nil {
			log.Printf("[MediaProcessor] claim failed: %v", err)
			if !waitForNextAttempt(ctx, processorIdleDelay) {
				return
			}
			continue
		}
		if item == nil {
			if !waitForNextAttempt(ctx, processorIdleDelay) {
				return
			}
			continue
		}

		if err := repo.ProcessClaimedMedia(item); err != nil {
			log.Printf("[MediaProcessor] processing media %s failed: %v", item.ID, err)
		}
	}
}

func waitForNextAttempt(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func runStaleSweep(ctx context.Context, repo processorRepository) {
	ticker := time.NewTicker(staleSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, err := repo.RequeueStaleProcessing(staleProcessingAfter)
			if err != nil {
				log.Printf("[MediaProcessor] stale sweep failed: %v", err)
				continue
			}
			if rows > 0 {
				log.Printf("[MediaProcessor] re-queued %d stale media rows", rows)
			}
		}
	}
}
