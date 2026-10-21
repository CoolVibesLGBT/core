package ports

import (
	"context"
	"core/application/types"
	domainmedia "core/domain/media"
	"core/models"
	"core/models/chat"
	modelutils "core/models/utils"
	"github.com/google/uuid"
)

type ChatGroupMediaUseCase interface {
	CreateGroupChatWithAvatar(context.Context, []string, *models.User, string, string, *domainmedia.UploadedFile, ...chat.GroupOptions) (*types.Chat, error)
	UpdateGroup(context.Context, *models.User, uuid.UUID, string, string, *domainmedia.UploadedFile, bool, ...chat.GroupOptions) (*types.Chat, error)
}
type ChatGroupMediaRepository interface {
	CreateGroupChatWithAvatar(context.Context, uuid.UUID, []uuid.UUID, *modelutils.LocalizedString, *modelutils.LocalizedString, *domainmedia.UploadedFile, ...chat.GroupOptions) (*chat.Chat, error)
	UpdateGroup(context.Context, *models.User, uuid.UUID, string, string, *domainmedia.UploadedFile, bool, ...chat.GroupOptions) (*chat.Chat, error)
}
