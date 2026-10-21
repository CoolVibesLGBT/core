package ports

import (
	"context"
	"core/application/types"
	"core/models"
	"core/models/post"
	"github.com/google/uuid"
)

type ChatMessageEditor interface {
	EditMessage(context.Context, *models.User, uuid.UUID, uuid.UUID, string) (*types.ChatMessage, error)
}

type ChatMessageEditorRepository interface {
	EditMessage(context.Context, *models.User, uuid.UUID, uuid.UUID, string) (*post.Post, error)
}
