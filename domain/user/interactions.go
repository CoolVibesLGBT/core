package user

import (
	"errors"
	"fmt"
)

var ErrSelfInteraction = errors.New("self interaction is not allowed")

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

func EnsureDifferentUsers(actorID string, targetID string, kind InteractionKind) error {
	if actorID == targetID {
		return fmt.Errorf("%w: %s", ErrSelfInteraction, kind)
	}
	return nil
}
