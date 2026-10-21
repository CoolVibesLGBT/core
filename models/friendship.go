package models

import (
	domainuser "core/domain/user"
	"time"

	"github.com/google/uuid"
)

// Friendship stores a canonical unordered pair for uniqueness. FirstUserID
// and SecondUserID follow UUID order, while RequestedByID defines the requester.
// Application read models expose requester/recipient roles instead of pair order.
type Friendship struct {
	ID            uuid.UUID                   `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	PublicID      int64                       `gorm:"uniqueIndex;not null"`
	FirstUserID   uuid.UUID                   `gorm:"type:uuid;not null;uniqueIndex:uidx_friendship_pair,priority:1;index"`
	SecondUserID  uuid.UUID                   `gorm:"type:uuid;not null;uniqueIndex:uidx_friendship_pair,priority:2;index;check:chk_friendship_pair,first_user_id < second_user_id"`
	RequestedByID uuid.UUID                   `gorm:"type:uuid;not null;check:chk_friendship_requester,requested_by_id = first_user_id OR requested_by_id = second_user_id"`
	Status        domainuser.FriendshipStatus `gorm:"type:varchar(16);not null;index;check:chk_friendship_status,status IN ('pending','accepted','denied','cancelled','removed')"`
	RequestedAt   time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time `gorm:"index"`
}
