package ports

import (
	"context"
	"core/application/types"
	"core/models"
	"core/models/post"
	"github.com/google/uuid"
)

type ChatMessageForwarder interface {
	ForwardMessages(context.Context, *models.User, uuid.UUID, uuid.UUID, []uuid.UUID, string) ([]types.ChatMessage, error)
}

type ChatMessageForwarderRepository interface {
	ForwardMessages(context.Context, *models.User, uuid.UUID, uuid.UUID, []uuid.UUID, string) ([]post.Post, error)
}
