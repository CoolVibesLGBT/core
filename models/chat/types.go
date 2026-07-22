package chat

import (
	"core/models/post"
	"time"

	"github.com/google/uuid"
)

type MessageType string
type ParticipantRole string
type MessageStatus string
type ChatType string

type ExpiredMessage struct {
	ChatID    uuid.UUID `json:"chat_id"`
	MessageID uuid.UUID `json:"message_id"`
	ExpiredAt time.Time `json:"expired_at"`
}

type OpenedMedia struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	MimeType   string    `json:"mime_type"`
	DataBase64 string    `json:"data_base64"`
}

type OpenMessageResult struct {
	Message *post.Post   `json:"message"`
	Media   *OpenedMedia `json:"media,omitempty"`
}

const (
	Text      MessageType = "text"
	Image     MessageType = "image"
	Video     MessageType = "video"
	Audio     MessageType = "audio"
	GIF       MessageType = "gif"
	Sticker   MessageType = "sticker"
	File      MessageType = "file"
	Location  MessageType = "location"
	System    MessageType = "system"
	Gift      MessageType = "gift"
	Poll      MessageType = "poll"
	CallAudio MessageType = "call_audio"
	CallVideo MessageType = "call_video"
)

const (
	Pending   MessageStatus = "pending"
	Delivered MessageStatus = "delivered"
	Seen      MessageStatus = "seen"
	Deleted   MessageStatus = "deleted"
)

const (
	ChatTypePrivate ChatType = "private"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

const (
	ParticipantRoleMember ParticipantRole = "member"
	ParticipantRoleAdmin  ParticipantRole = "admin"
	ParticipantRoleOwner  ParticipantRole = "owner"
)
