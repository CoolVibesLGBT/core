package media

import (
	"context"
	mediamodel "core/models/media"
	"log"
	"sync"
	"time"
)

const (
	processorWorkerCount = 2
	processorIdleDelay   = time.Second
	staleProcessingAfter = 10 * time.Minute
	staleSweepInterval   = time.Minute
)

// Repository is the outbound port required by the media processor. Its
// implementation is selected by the composition root; the worker must not
// construct a persistence adapter itself.
type Repository interface {
	ClaimNextPendingMedia(time.Time) (*mediamodel.Media, error)
	ProcessClaimedMedia(*mediamodel.Media) error
	RequeueStaleProcessing(time.Duration) (int64, error)
}

type ProcessingObserver interface {
	MediaProcessingUpdated(ctx context.Context, item *mediamodel.Media, status mediamodel.ProcessingStatus)
}

type Processor struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func StartProcessor(repo Repository, observers ...ProcessingObserver) *Processor {
	return StartProcessorContext(context.Background(), repo, observers...)
}

func StartProcessorContext(ctx context.Context, repo Repository, observers ...ProcessingObserver) *Processor {
	if repo == nil {
		return nil
	}
	return startProcessor(ctx, repo, observers...)
}

func startProcessor(parent context.Context, repo Repository, observers ...ProcessingObserver) *Processor {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	processor := &Processor{
		cancel: cancel,
		done:   make(chan struct{}),
	}
	var observer ProcessingObserver
	if len(observers) > 0 {
		observer = observers[0]
	}

	var wg sync.WaitGroup
	wg.Add(processorWorkerCount + 1)

	for i := 0; i < processorWorkerCount; i++ {
		go func() {
			defer wg.Done()
			runWorker(ctx, repo, observer)
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

func runWorker(ctx context.Context, repo Repository, observer ProcessingObserver) {
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
		if observer != nil && (item.ProcessingStatus == mediamodel.ProcessingStatusReady || item.ProcessingStatus == mediamodel.ProcessingStatusFailed) {
			observer.MediaProcessingUpdated(context.WithoutCancel(ctx), item, item.ProcessingStatus)
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

func runStaleSweep(ctx context.Context, repo Repository) {
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
