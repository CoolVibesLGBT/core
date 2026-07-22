package user

import "testing"

func TestSetInteractionStateIsIdempotent(t *testing.T) {
	intent, err := NewSetInteractionState(InteractionFollow, true)
	if err != nil {
		t.Fatalf("NewSetInteractionState() error = %v", err)
	}

	transition, err := intent.Transition(false)
	if err != nil || !transition.Enabled || !transition.Changed {
		t.Fatalf("first transition = %+v, %v; want enabled and changed", transition, err)
	}

	transition, err = intent.Transition(true)
	if err != nil || !transition.Enabled || transition.Changed {
		t.Fatalf("repeated transition = %+v, %v; want enabled and unchanged", transition, err)
	}
}

func TestToggleInteractionStateUsesCurrentState(t *testing.T) {
	intent, err := NewToggleInteractionState(InteractionBlock)
	if err != nil {
		t.Fatalf("NewToggleInteractionState() error = %v", err)
	}

	for current, expected := range map[bool]bool{false: true, true: false} {
		transition, err := intent.Transition(current)
		if err != nil || transition.Enabled != expected || !transition.Changed {
			t.Fatalf("Transition(%v) = %+v, %v; want enabled=%v changed=true", current, transition, err, expected)
		}
	}
}

func TestToggleReactionStateSelectsLikeAndDislikePairs(t *testing.T) {
	for positive, expected := range map[bool]EngagementPair{
		true:  {Given: EngagementLikeGiven, Received: EngagementLikeReceived},
		false: {Given: EngagementDislikeGiven, Received: EngagementDislikeReceived},
	} {
		intent, err := NewToggleReactionState(positive)
		if err != nil {
			t.Fatalf("NewToggleReactionState(%v) error = %v", positive, err)
		}
		pair, err := intent.EngagementPair()
		if err != nil || pair != expected {
			t.Fatalf("NewToggleReactionState(%v) pair = %+v, %v; want %+v", positive, pair, err, expected)
		}
	}
}

func TestInteractionStateRejectsUnsupportedInteraction(t *testing.T) {
	if _, err := NewSetInteractionState(InteractionChat, true); err != ErrUnsupportedInteraction {
		t.Fatalf("NewSetInteractionState(chat) error = %v; want %v", err, ErrUnsupportedInteraction)
	}
	if _, err := NewToggleInteractionState(InteractionKind("unknown")); err != ErrUnsupportedInteraction {
		t.Fatalf("NewToggleInteractionState(unknown) error = %v; want %v", err, ErrUnsupportedInteraction)
	}
}

func TestEnsureActorMatchesPrincipal(t *testing.T) {
	if err := EnsureActorMatchesPrincipal(42, 42); err != nil {
		t.Fatalf("matching principal error = %v", err)
	}
	for _, ids := range [][2]int64{{0, 42}, {42, 0}, {42, 43}} {
		if err := EnsureActorMatchesPrincipal(ids[0], ids[1]); err != ErrActorMismatch {
			t.Fatalf("EnsureActorMatchesPrincipal(%d, %d) error = %v; want %v", ids[0], ids[1], err, ErrActorMismatch)
		}
	}
}
