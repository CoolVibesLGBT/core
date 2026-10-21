package notifications

import (
	"context"
	"testing"
	"time"
)

type cancelledDelivery struct{ started chan struct{} }

func (r *cancelledDelivery) ProcessNextNotification(ctx context.Context, _ time.Time) (bool, error) {
	close(r.started)
	<-ctx.Done()
	return true, ctx.Err()
}
func TestNotificationProcessorCancelsDeliveryOnShutdown(t *testing.T) {
	repo := &cancelledDelivery{started: make(chan struct{})}
	processor := StartProcessorContext(context.Background(), repo)
	select {
	case <-repo.started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := processor.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
}
