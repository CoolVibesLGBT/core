package user

import (
	"errors"
	"math"
	"testing"
)

func TestFriendshipRequiresRecipientApproval(t *testing.T) {
	if _, _, err := DecideFriendship(FriendshipPending, FriendshipAccepted, true); !errors.Is(err, ErrFriendshipForbidden) {
		t.Fatalf("requester approved: %v", err)
	}
	status, changed, err := DecideFriendship(FriendshipPending, FriendshipAccepted, false)
	if err != nil || !changed || status != FriendshipAccepted {
		t.Fatalf("recipient decision: %v %v %v", status, changed, err)
	}
	if _, changed, err := DecideFriendship(status, FriendshipAccepted, false); err != nil || changed {
		t.Fatalf("repeat decision: %v %v", changed, err)
	}
	if _, _, err := DecideFriendship(FriendshipDenied, FriendshipAccepted, false); !errors.Is(err, ErrFriendshipTransition) {
		t.Fatal("denial silently reversed")
	}
	if status, _, err := DecideFriendship(FriendshipAccepted, FriendshipRemoved, true); err != nil || status != FriendshipRemoved {
		t.Fatal("requester could not remove friend")
	}
}

func TestMatchCompatibilityNormalizesSetsAndMissingSignals(t *testing.T) {
	distance := 0.0
	full := MatchCompatibility([]string{" MUSIC ", "music"}, []string{"music"}, []string{"EN"}, []string{"en"}, &distance)
	if full != 100 {
		t.Fatalf("full score=%v", full)
	}
	if score := MatchCompatibility(nil, nil, nil, nil, nil); score != 0 {
		t.Fatalf("empty score=%v", score)
	}
	far := 1000.0
	if MatchCompatibility(nil, nil, nil, nil, &distance) <= MatchCompatibility(nil, nil, nil, nil, &far) {
		t.Fatal("nearer did not rank higher")
	}
	nan := math.NaN()
	if score := MatchCompatibility(nil, nil, nil, nil, &nan); score != 0 {
		t.Fatalf("invalid distance score=%v", score)
	}
	low, high := 30, 20
	if err := (MatchCriteria{MinAge: &low, MaxAge: &high}).Validate(); err == nil {
		t.Fatal("invalid age range accepted")
	}
}
