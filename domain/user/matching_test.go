package user

import (
	"errors"
	"testing"
)

func TestParseMatchReactionAcceptsOnlySwipeDecisions(t *testing.T) {
	for input, expected := range map[string]MatchReaction{
		"like":      MatchReactionLike,
		" LIKE ":    MatchReactionLike,
		"dislike":   MatchReactionDislike,
		" DisLike ": MatchReactionDislike,
	} {
		actual, err := ParseMatchReaction(input)
		if err != nil || actual != expected {
			t.Fatalf("ParseMatchReaction(%q) = %q, %v; want %q", input, actual, err, expected)
		}
	}

	for _, input := range []string{"", "favorite", "matched", "superlike"} {
		if _, err := ParseMatchReaction(input); !errors.Is(err, ErrInvalidMatchReaction) {
			t.Fatalf("ParseMatchReaction(%q) error = %v; want ErrInvalidMatchReaction", input, err)
		}
	}
}

func TestMatchReactionReturnsExclusiveEngagementPairs(t *testing.T) {
	like, dislike, err := MatchReactionLike.EngagementPairs()
	if err != nil {
		t.Fatal(err)
	}
	if like.Given != EngagementLikeGiven || dislike.Given != EngagementDislikeGiven {
		t.Fatalf("like pairs = %+v, %+v", like, dislike)
	}

	dislike, like, err = MatchReactionDislike.EngagementPairs()
	if err != nil {
		t.Fatal(err)
	}
	if dislike.Given != EngagementDislikeGiven || like.Given != EngagementLikeGiven {
		t.Fatalf("dislike pairs = %+v, %+v", dislike, like)
	}
}
