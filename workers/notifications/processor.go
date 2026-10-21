package notifications

import (
	"context"
	"log"
	"time"
)

type Repository interface {
	ProcessNextNotification(context.Context, time.Time) (bool, error)
}
type Processor struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func StartProcessorContext(parent context.Context, repo Repository) *Processor {
	if repo == nil {
		return nil
	}
	ctx, cancel := context.WithCancel(parent)
	p := &Processor{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(p.done)
		for ctx.Err() == nil {
			worked, err := repo.ProcessNextNotification(ctx, time.Now().UTC())
			if err != nil && ctx.Err() == nil {
				log.Printf("notification delivery deferred: %v", err)
			}
			if worked && err == nil {
				continue
			}
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	return p
}

func (p *Processor) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.cancel()
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
