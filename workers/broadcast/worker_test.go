package broadcast

import (
	"context"
	"core/models"
	mediamodel "core/models/media"
	"core/workers"
	"testing"
	"time"

	"github.com/google/uuid"
)

type broadcastRepositoryFake struct {
	user       *models.User
	found      bool
	updatedID  uuid.UUID
	updatedRaw []byte
}

func (*broadcastRepositoryFake) ResetBotBroadcastPresence(context.Context) error { return nil }

func (r *broadcastRepositoryFake) FindBroadcastUser(context.Context, []string) (*models.User, bool, error) {
	return r.user, r.found, nil
}

func (r *broadcastRepositoryFake) UpdateBroadcastState(_ context.Context, userID uuid.UUID, raw []byte) error {
	r.updatedID = userID
	r.updatedRaw = append([]byte(nil), raw...)
	return nil
}

type broadcastUserServiceFake struct{}

func (*broadcastUserServiceFake) CreateBotUser(context.Context, *models.User) (*models.User, error) {
	return nil, nil
}

func (*broadcastUserServiceFake) UpdateAvatarFromURL(context.Context, string, *models.User) (*mediamodel.Media, error) {
	return nil, nil
}

func TestProcessBroadcastDataUsesInjectedRepository(t *testing.T) {
	userID := uuid.New()
	repository := &broadcastRepositoryFake{
		user:  &models.User{ID: userID},
		found: true,
	}
	data := []byte(`{"result":{"broadcasts":[{"language":"en","userDetails":{"networkUserId":"123","memberId":"456"}}]}}`)

	err := processBroadcastData(context.Background(), Dependencies{
		Repository: repository,
		Users:      &broadcastUserServiceFake{},
	}, data, "provider")
	if err != nil {
		t.Fatalf("processBroadcastData() error = %v", err)
	}
	if repository.updatedID != userID {
		t.Fatalf("updated user ID = %s, want %s", repository.updatedID, userID)
	}
	if len(repository.updatedRaw) == 0 {
		t.Fatal("broadcast payload was not persisted")
	}
}

func TestProcessBroadcastDataRejectsMissingDependencies(t *testing.T) {
	if err := processBroadcastData(context.Background(), Dependencies{}, []byte(`{}`), "provider"); err == nil {
		t.Fatal("processBroadcastData() error = nil, want dependency error")
	}
}

func TestFetcherShutdownWaitsForScheduledWork(t *testing.T) {
	dispatcher := workers.NewDispatcher(1, 1)
	dispatcher.Run()
	fetcher := StartFetcher(dispatcher, Dependencies{})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := fetcher.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
