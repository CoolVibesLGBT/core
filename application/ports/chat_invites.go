package ports

import (
	"context"
	"core/application/types"
	"core/models"
	"core/models/chat"
	"github.com/google/uuid"
)

type ChatGroupInvites interface {
	ExportGroupInvite(context.Context, *models.User, uuid.UUID, bool) (*types.ChatGroupInvite, error)
	PreviewGroup(context.Context, string, string) (*types.ChatGroupPreview, error)
	JoinGroup(context.Context, *models.User, string, string) (*types.Chat, error)
}
type ChatGroupInvitesRepository interface {
	ExportGroupInvite(context.Context, *models.User, uuid.UUID, bool) (*chat.GroupInvite, error)
	PreviewGroup(context.Context, string, string) (*chat.Chat, int64, error)
	JoinGroup(context.Context, *models.User, string, string) (*chat.Chat, error)
}
