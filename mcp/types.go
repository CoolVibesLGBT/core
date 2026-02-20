package mcp

import "time"

type MessageType string

const (
	TypeEvent    MessageType = "event"
	TypeCommand  MessageType = "command"
	TypeResponse MessageType = "response"
	TypeError    MessageType = "error"
)

type Envelope struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	Source    string      `json:"source"`
	Target    string      `json:"target,omitempty"`
	Action    string      `json:"action"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   any         `json:"payload"`
}
