package user

import (
	"core/domain/events"
	"time"
)

const EventUserRegistered = "user.registered"

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
