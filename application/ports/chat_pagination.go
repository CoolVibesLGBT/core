package ports

import (
	"core/models/chat"
	"core/models/post"
	"time"

	"github.com/google/uuid"
)

// ChatListCursor is a stable keyset cursor. ActivityAt is the chat's last
// message timestamp, falling back to the chat creation time for empty chats.
// ChatID breaks timestamp ties so pagination cannot skip or repeat rows.
type ChatListCursor struct {
	ActivityAt time.Time
	ChatID     uuid.UUID
}

type ChatListQuery struct {
	UserID uuid.UUID
	Cursor *ChatListCursor
	Limit  int
}

type ChatListPage struct {
	Chats   []chat.Chat
	HasMore bool
}

// ChatMessageListCursor uses the globally unique snowflake public ID. Message
// pages are selected newest-first in SQL and returned oldest-first to preserve
// the mobile API's established chronological array order.
type ChatMessageListCursor struct {
	PublicID int64
}

type ChatMessageListQuery struct {
	UserID uuid.UUID
	ChatID uuid.UUID
	Cursor *ChatMessageListCursor
	Limit  int
}

type ChatMessageListPage struct {
	Messages []post.Post
	HasMore  bool
}
