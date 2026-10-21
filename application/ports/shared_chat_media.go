package ports

import (
	"context"
	"github.com/google/uuid"
)

// A forwarded file can have several independent, authorized chat attachments.
type SharedChatMediaAccess interface {
	HasVisibleChatFileAccess(context.Context, string, uuid.UUID) (bool, error)
}
type PublicGroupAvatarAccess interface {
	IsPublicGroupAvatar(context.Context, string) (bool, error)
}
