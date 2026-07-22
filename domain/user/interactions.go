package user

import (
	"errors"
	"fmt"
)

var (
	ErrSelfInteraction = errors.New("self interaction is not allowed")
	ErrActorMismatch   = errors.New("interaction actor does not match authenticated principal")
)

type InteractionKind string

const (
	InteractionFollow    InteractionKind = "follow"
	InteractionLike      InteractionKind = "like"
	InteractionBlock     InteractionKind = "block"
	InteractionSubscribe InteractionKind = "subscribe"
	InteractionChat      InteractionKind = "chat"
)

func EnsureDifferentPublicUsers(actorID int64, targetID int64, kind InteractionKind) error {
	if actorID == targetID {
		return fmt.Errorf("%w: %s", ErrSelfInteraction, kind)
	}
	return nil
}

func EnsureActorMatchesPrincipal(principalID int64, actorID int64) error {
	if principalID <= 0 || actorID <= 0 || principalID != actorID {
		return ErrActorMismatch
	}
	return nil
}

func EnsureDifferentUsers(actorID string, targetID string, kind InteractionKind) error {
	if actorID == targetID {
		return fmt.Errorf("%w: %s", ErrSelfInteraction, kind)
	}
	return nil
}
