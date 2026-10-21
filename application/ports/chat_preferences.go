package ports

import (
	"context"
	"core/models"
	"github.com/google/uuid"
)

type ChatMutePreference interface {
	SetChatMuted(context.Context, *models.User, uuid.UUID, bool) error
}
