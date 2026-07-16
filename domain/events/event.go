package events

import "time"

type Event interface {
	Name() string
	OccurredAt() time.Time
}

type BaseEvent struct {
	name       string
	occurredAt time.Time
}

func NewBaseEvent(name string, occurredAt time.Time) BaseEvent {
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return BaseEvent{name: name, occurredAt: occurredAt}
}

func (e BaseEvent) Name() string {
	return e.name
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.occurredAt
}
