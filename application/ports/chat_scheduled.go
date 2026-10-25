package ports

import (
	"context"
	"core/application/types"
	"core/models"
	"core/models/post"
	"time"

	"github.com/google/uuid"
)

// ChatScheduledMessages is the Telegram "Scheduled Messages" use case.
type ChatScheduledMessages interface {
	FetchScheduledMessages(ctx context.Context, userID, chatID uuid.UUID) ([]types.ChatMessage, error)
	SendScheduledNow(ctx context.Context, author *models.User, chatID, messageID uuid.UUID) error
}

type ChatScheduledMessagesRepository interface {
	ListScheduledMessages(ctx context.Context, userID, chatID uuid.UUID) ([]post.Post, error)
	SendScheduledNow(ctx context.Context, author *models.User, chatID, messageID uuid.UUID) error
	PublishScheduledMessages(ctx context.Context, now time.Time, limit int) ([]post.Post, error)
}
