package post

import (
	"errors"
	"testing"
)

func TestPollVotePolicyEnforcesKinds(t *testing.T) {
	tests := []struct {
		name     string
		policy   PollVotePolicy
		current  int
		weight   int
		rank     int
		rankUsed bool
		wantErr  error
	}{
		{name: "single replaces", policy: PollVotePolicy{Kind: PollSingle, MaxSelectable: 1, ChoiceCount: 3}, current: 1, weight: 1},
		{name: "single rejects rank", policy: PollVotePolicy{Kind: PollSingle, ChoiceCount: 3}, weight: 1, rank: 1, wantErr: ErrInvalidPollRank},
		{name: "multiple limit", policy: PollVotePolicy{Kind: PollMultiple, MaxSelectable: 2, ChoiceCount: 3}, current: 2, weight: 1, wantErr: ErrPollSelectionLimit},
		{name: "rank required", policy: PollVotePolicy{Kind: PollRanked, MaxSelectable: 3, ChoiceCount: 3}, weight: 1, wantErr: ErrInvalidPollRank},
		{name: "rank unique", policy: PollVotePolicy{Kind: PollRanked, MaxSelectable: 3, ChoiceCount: 3}, weight: 1, rank: 2, rankUsed: true, wantErr: ErrDuplicatePollRank},
		{name: "weighted positive", policy: PollVotePolicy{Kind: PollWeighted, MaxSelectable: 2, ChoiceCount: 3}, weight: 5},
		{name: "weighted rejects zero", policy: PollVotePolicy{Kind: PollWeighted, ChoiceCount: 3}, weight: 0, wantErr: ErrInvalidPollWeight},
		{name: "unknown kind", policy: PollVotePolicy{Kind: "surprise", ChoiceCount: 3}, weight: 1, wantErr: ErrInvalidPollKind},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.policy.ValidateNewVote(test.current, test.weight, test.rank, test.rankUsed)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ValidateNewVote() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestNewPollDefinitionValidatesCreationInvariant(t *testing.T) {
	definition, err := NewPollDefinition("  Where?  ", PollMultiple, 2, []string{" Home ", "Away"})
	if err != nil {
		t.Fatal(err)
	}
	if definition.Question != "Where?" || definition.Kind != PollMultiple || definition.MaxSelectable != 2 || definition.Options[0] != "Home" {
		t.Fatalf("normalized definition = %#v", definition)
	}

	tests := []struct {
		name     string
		question string
		kind     PollKind
		maximum  int
		options  []string
		wantErr  error
	}{
		{name: "question", kind: PollSingle, maximum: 1, options: []string{"a", "b"}, wantErr: ErrPollQuestionRequired},
		{name: "kind", question: "q", kind: "surprise", maximum: 1, options: []string{"a", "b"}, wantErr: ErrInvalidPollKind},
		{name: "few options", question: "q", kind: PollSingle, maximum: 1, options: []string{"a"}, wantErr: ErrPollOptionsRequired},
		{name: "empty option", question: "q", kind: PollSingle, maximum: 1, options: []string{"a", " "}, wantErr: ErrInvalidPollChoiceData},
		{name: "duplicate option", question: "q", kind: PollSingle, maximum: 1, options: []string{"Yes", " yes "}, wantErr: ErrDuplicatePollOption},
		{name: "single maximum", question: "q", kind: PollSingle, maximum: 2, options: []string{"a", "b"}, wantErr: ErrInvalidPollMaximum},
		{name: "multiple maximum", question: "q", kind: PollMultiple, maximum: 3, options: []string{"a", "b"}, wantErr: ErrInvalidPollMaximum},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewPollDefinition(test.question, test.kind, test.maximum, test.options)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("NewPollDefinition() error = %v; want %v", err, test.wantErr)
			}
		})
	}
}
