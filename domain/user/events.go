package user

import (
	"core/domain/events"
	"time"
)

const EventUserRegistered = "user.registered"

const (
	EventUserFollowToggled    = "user.follow_toggled"
	EventUserReactionToggled  = "user.reaction_toggled"
	EventUserBlockToggled     = "user.block_toggled"
	EventUserSubscribeToggled = "user.subscribe_toggled"
)

type RegisteredEvent struct {
	events.BaseEvent
	UserID   string
	PublicID int64
	Domain   DomainKind
}

func NewRegisteredEvent(userID string, publicID int64, domain DomainKind, occurredAt time.Time) RegisteredEvent {
	return RegisteredEvent{
		BaseEvent: events.NewBaseEvent(EventUserRegistered, occurredAt),
		UserID:    userID,
		PublicID:  publicID,
		Domain:    domain,
	}
}

type InteractionToggledEvent struct {
	events.BaseEvent
	ActorPublicID  int64
	TargetPublicID int64
	Interaction    InteractionKind
	Enabled        bool
}

func NewInteractionToggledEvent(actorPublicID int64, targetPublicID int64, interaction InteractionKind, enabled bool, occurredAt time.Time) InteractionToggledEvent {
	eventName := EventUserReactionToggled
	switch interaction {
	case InteractionFollow:
		eventName = EventUserFollowToggled
	case InteractionBlock:
		eventName = EventUserBlockToggled
	case InteractionSubscribe:
		eventName = EventUserSubscribeToggled
	}

	return InteractionToggledEvent{
		BaseEvent:      events.NewBaseEvent(eventName, occurredAt),
		ActorPublicID:  actorPublicID,
		TargetPublicID: targetPublicID,
		Interaction:    interaction,
		Enabled:        enabled,
	}
}
