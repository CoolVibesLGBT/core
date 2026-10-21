package usecases

import (
	"context"
	"core/application/ports"
	domainuser "core/domain/user"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type recordingMatchesRepository struct {
	ports.MatchesRepository
	calls    int
	reaction domainuser.MatchReaction
}

func (r *recordingMatchesRepository) RecordView(_ context.Context, _, _ uuid.UUID, reaction domainuser.MatchReaction) (bool, error) {
	r.calls++
	r.reaction = reaction
	return false, nil
}

func TestMatchesServiceValidatesReactionAndSelfBeforePersistence(t *testing.T) {
	repository := &recordingMatchesRepository{}
	service := NewMatchService(nil, repository)
	actorID, targetID := uuid.New(), uuid.New()

	if _, err := service.RecordView(context.Background(), actorID, targetID, " LIKE "); err != nil {
		t.Fatal(err)
	}
	if repository.calls != 1 || repository.reaction != domainuser.MatchReactionLike {
		t.Fatalf("repository call = %d, reaction %q", repository.calls, repository.reaction)
	}

	if _, err := service.RecordView(context.Background(), actorID, targetID, "favorite"); !errors.Is(err, domainuser.ErrInvalidMatchReaction) {
		t.Fatalf("invalid reaction error = %v", err)
	}
	if _, err := service.RecordView(context.Background(), actorID, actorID, "like"); !errors.Is(err, domainuser.ErrSelfInteraction) {
		t.Fatalf("self reaction error = %v", err)
	}
	if repository.calls != 1 {
		t.Fatalf("invalid commands reached repository %d times", repository.calls-1)
	}
}

type contextIdentityResolver struct {
	context context.Context
	id      uuid.UUID
}

func (r *contextIdentityResolver) GetUserUUIDByPublicID(ctx context.Context, _ int64) (uuid.UUID, error) {
	r.context = ctx
	return r.id, ctx.Err()
}

func TestMatchIdentityLookupPreservesRequestContextAndDeadline(t *testing.T) {
	resolver := &contextIdentityResolver{id: uuid.New()}
	service := NewMatchService(resolver, nil)
	type marker struct{}
	ctx := context.WithValue(context.Background(), marker{}, "request")
	got, err := service.GetUserUUIDByPublicID(ctx, 42)
	if err != nil || got != resolver.id {
		t.Fatalf("lookup=%v %v", got, err)
	}
	if resolver.context.Value(marker{}) != "request" {
		t.Fatal("lookup lost request values")
	}
	if deadline, ok := resolver.context.Deadline(); !ok || time.Until(deadline) > matchesRepositoryTimeout {
		t.Fatal("lookup has no bounded deadline")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.GetUserUUIDByPublicID(cancelled, 42); !errors.Is(err, context.Canceled) {
		t.Fatalf("lookup ignored cancellation: %v", err)
	}
}
