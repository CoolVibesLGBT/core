package ports

import (
	"context"
	"core/application/types"
	domainuser "core/domain/user"
	"time"

	"github.com/google/uuid"
)

// FriendshipRecord exposes request roles independently of database pair order.
type FriendshipRecord struct {
	PublicID          int64
	RequesterPublicID int64
	RecipientPublicID int64
	Status            domainuser.FriendshipStatus
	RequestedAt       time.Time
	UpdatedAt         time.Time
	OtherUser         types.PublicUserSummary // The participant other than the authenticated viewer.
}

type FriendshipRepository interface {
	RequestFriendship(context.Context, uuid.UUID, int64) (FriendshipRecord, error)
	ChangeFriendship(context.Context, uuid.UUID, int64, domainuser.FriendshipStatus) (FriendshipRecord, error)
	ListFriendships(context.Context, uuid.UUID, domainuser.FriendshipScope, *types.MatchListCursor, int) ([]FriendshipRecord, *types.MatchListCursor, error)
}
type FriendshipStatusReader interface {
	FindFriendship(context.Context, uuid.UUID, int64) (*FriendshipRecord, error)
}

type UserProfileAccess interface {
	CanViewUserProfile(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}
type ViewerUserProfileReader interface {
	FetchUserProfileForViewer(context.Context, string, uuid.UUID) (*types.PublicUserProfile, error)
}
