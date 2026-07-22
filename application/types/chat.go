package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ChatUser is the only user shape that may cross a chat transport boundary.
// ID intentionally mirrors PublicID for clients that still read the legacy
// `id` key. Database UUIDs and account state are not part of this contract.
type ChatUser struct {
	ID          SnowflakeID      `json:"id"`
	PublicID    SnowflakeID      `json:"public_id"`
	UserName    string           `json:"username"`
	DisplayName string           `json:"displayname"`
	Avatar      *PublicUserMedia `json:"avatar,omitempty"`
}

// ChatParticipant exposes the public participant identity. Viewer-specific
// state is populated only for the authenticated participant; another
// participant's read/mute state is never returned.
type ChatParticipant struct {
	UserID      SnowflakeID `json:"user_id"`
	User        ChatUser    `json:"user"`
	UnreadCount *int        `json:"unread_count,omitempty"`
	IsMuted     *bool       `json:"is_muted,omitempty"`
	LastReadAt  *time.Time  `json:"last_read_at,omitempty"`
}

// Chat is an application read model. Chat/message UUIDs remain transport
// identifiers because existing clients use them for room and mutation
// commands; user UUIDs and persistence-only fields are deliberately absent.
type Chat struct {
	ID          uuid.UUID         `json:"id"`
	Type        string            `json:"type"`
	Title       map[string]string `json:"title,omitempty"`
	Description map[string]string `json:"description,omitempty"`
	Avatar      *ChatMedia        `json:"avatar,omitempty"`

	CreatorID   *SnowflakeID `json:"creator_id,omitempty"`
	PinnedMsgID *uuid.UUID   `json:"pinned_msg_id,omitempty"`
	PinnedMsg   *ChatMessage `json:"pinned_msg,omitempty"`
	PinnedByID  *SnowflakeID `json:"pinned_by_id,omitempty"`
	PinnedBy    *ChatUser    `json:"pinned_by,omitempty"`

	LastMessageID        *uuid.UUID   `json:"last_message_id,omitempty"`
	LastMessage          *ChatMessage `json:"last_message,omitempty"`
	LastMessageTimestamp *time.Time   `json:"last_message_timestamp,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Participants []ChatParticipant `json:"participants,omitempty"`
	Messages     []ChatMessage     `json:"messages,omitempty"`

	// These convenience fields mirror the authenticated participant state.
	UnreadCount int  `json:"unread_count"`
	IsMuted     bool `json:"is_muted"`
}

type ChatMessage struct {
	ID       uuid.UUID    `json:"id"`
	ParentID *uuid.UUID   `json:"parent_id,omitempty"`
	Parent   *ChatMessage `json:"parent,omitempty"`

	PublicID SnowflakeID `json:"public_id"`
	PostKind string      `json:"post_kind"`
	Domain   string      `json:"domain"`

	ContentCategory string     `json:"content_category,omitempty"`
	ContentableID   *uuid.UUID `json:"contentable_id,omitempty"`
	ContentableType *string    `json:"contentable_type,omitempty"`

	AuthorID SnowflakeID `json:"author_id"`
	Author   ChatUser    `json:"author"`

	Title   map[string]string `json:"title,omitempty"`
	Content map[string]string `json:"content,omitempty"`
	Summary map[string]string `json:"summary,omitempty"`

	Attachments []ChatMedia          `json:"attachments,omitempty"`
	Location    *ChatMessageLocation `json:"location,omitempty"`

	ExpiresInSeconds *int       `json:"expires_in_seconds,omitempty"`
	OpenedAt         *time.Time `json:"opened_at,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	ViewOnce         bool       `json:"view_once"`
	IsDisappearing   bool       `json:"is_disappearing"`
	ContentHidden    bool       `json:"content_hidden"`
	ViewedOnce       bool       `json:"viewed_once"`
	ClientID         string     `json:"client_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMedia and ChatMediaFile are URL-only media projections. They exclude
// media/file UUIDs, owner IDs, storage paths, processing internals and errors.
type ChatMedia struct {
	ID       SnowflakeID   `json:"id"`
	PublicID SnowflakeID   `json:"public_id"`
	File     ChatMediaFile `json:"file"`
}

type ChatMediaFile struct {
	URL      string          `json:"url,omitempty"`
	MimeType string          `json:"mime_type,omitempty"`
	Name     string          `json:"name,omitempty"`
	Variants json.RawMessage `json:"variants,omitempty"`
}

type ChatMessageLocation struct {
	CountryCode *string  `json:"country_code,omitempty"`
	Display     *string  `json:"display,omitempty"`
	Address     *string  `json:"address,omitempty"`
	City        *string  `json:"city,omitempty"`
	Country     *string  `json:"country,omitempty"`
	Region      *string  `json:"region,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
}

type ChatOpenedMedia struct {
	Name       string `json:"name"`
	MimeType   string `json:"mime_type"`
	DataBase64 string `json:"data_base64"`
}

type ChatOpenMessageResult struct {
	Message *ChatMessage     `json:"message"`
	Media   *ChatOpenedMedia `json:"media,omitempty"`
}

type ChatTypingEvent struct {
	Action string      `json:"action"`
	ChatID uuid.UUID   `json:"chat_id"`
	UserID SnowflakeID `json:"user_id"`
	Typing bool        `json:"typing"`
}

type ChatMessageEvent struct {
	Action  string       `json:"action"`
	Message *ChatMessage `json:"message"`
}

type ChatMessageOpenedEvent struct {
	Action    string      `json:"action"`
	ChatID    uuid.UUID   `json:"chat_id"`
	MessageID uuid.UUID   `json:"message_id"`
	OpenedBy  SnowflakeID `json:"opened_by"`
	OpenedAt  *time.Time  `json:"opened_at,omitempty"`
	ExpiresAt *time.Time  `json:"expires_at,omitempty"`
	ViewOnce  bool        `json:"view_once"`
}

type ChatMessageExpiredEvent struct {
	Action    string    `json:"action"`
	ChatID    uuid.UUID `json:"chat_id"`
	MessageID uuid.UUID `json:"message_id"`
	ExpiredAt time.Time `json:"expired_at"`
}
