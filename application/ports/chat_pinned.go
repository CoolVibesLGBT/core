package ports

import (
	"context"
	"core/application/types"
	"core/models/post"

	"github.com/google/uuid"
)

// ChatPinnedMessagesFetcher lists a chat's pinned messages for a viewer.
type ChatPinnedMessagesFetcher interface {
	FetchPinnedMessages(ctx context.Context, userID, chatID uuid.UUID) ([]types.ChatMessage, error)
}

type ChatPinnedMessagesRepository interface {
	ListPinnedMessages(ctx context.Context, userID, chatID uuid.UUID) ([]post.Post, error)
}
