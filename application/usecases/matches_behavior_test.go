package usecases

import (
	"context"
	"core/application/ports"
	domainuser "core/domain/user"
	"errors"
	"testing"

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
