package user

import (
	"errors"
	"strings"
)

var (
	ErrInvalidMatchReaction   = errors.New("invalid match reaction")
	ErrMatchTargetUnavailable = errors.New("match target is unavailable")
)

// MatchReaction is the decision recorded by the matching aggregate for one
// actor/target pair. A swipe is a set command, not a toggle: replaying the same
// command must leave the relationship in the same state.
type MatchReaction string

const (
	MatchReactionLike    MatchReaction = "like"
	MatchReactionDislike MatchReaction = "dislike"
)

func ParseMatchReaction(value string) (MatchReaction, error) {
	reaction := MatchReaction(strings.ToLower(strings.TrimSpace(value)))
	switch reaction {
	case MatchReactionLike, MatchReactionDislike:
		return reaction, nil
	default:
		return "", ErrInvalidMatchReaction
	}
}

func (r MatchReaction) IsLike() bool {
	return r == MatchReactionLike
}

func (r MatchReaction) EngagementPairs() (selected EngagementPair, opposite EngagementPair, err error) {
	switch r {
	case MatchReactionLike:
		selected, err = InteractionEngagementPair(InteractionLike, true)
		if err != nil {
			return EngagementPair{}, EngagementPair{}, err
		}
		opposite, err = InteractionEngagementPair(InteractionLike, false)
	case MatchReactionDislike:
		selected, err = InteractionEngagementPair(InteractionLike, false)
		if err != nil {
			return EngagementPair{}, EngagementPair{}, err
		}
		opposite, err = InteractionEngagementPair(InteractionLike, true)
	default:
		err = ErrInvalidMatchReaction
	}
	return selected, opposite, err
}
