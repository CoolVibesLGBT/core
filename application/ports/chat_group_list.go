package ports

import (
	"context"
	"core/application/types"
	"core/models/chat"
	"github.com/google/uuid"
)

type PublicGroupQuery struct {
	Search string
	Cursor *uuid.UUID
	Limit  int
}
type PublicGroupRow struct {
	chat.Chat
	MemberCount int64
}
type PublicGroupPage struct {
	Groups  []PublicGroupRow
	HasMore bool
}
type ChatGroupListRepository interface {
	ListPublicGroups(context.Context, PublicGroupQuery) (PublicGroupPage, error)
}
type ChatGroupList interface {
	ListPublicGroups(context.Context, PublicGroupQuery) ([]types.ChatGroupPreview, *string, error)
}
