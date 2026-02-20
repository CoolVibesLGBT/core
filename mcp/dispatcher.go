package mcp

import (
	"time"

	"github.com/google/uuid"
)

func NewMessage(
	source string,
	target string,
	action string,
	payload any,
	msgType MessageType,
) Envelope {

	return Envelope{
		ID:        uuid.NewString(),
		Type:      msgType,
		Source:    source,
		Target:    target,
		Action:    action,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	}
}
