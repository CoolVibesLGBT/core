package models

import (
	domainmedia "core/domain/media"
	"time"

	"github.com/google/uuid"
)

// PrivatePhotoAccessRequest stores one album-level request per owner/viewer
// pair. A denied row is retained for audit and can later be moved back to
// pending by a new request.
type PrivatePhotoAccessRequest struct {
	ID       uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	PublicID int64     `gorm:"uniqueIndex;not null" json:"public_id"`

	OwnerID  uuid.UUID                            `gorm:"type:uuid;not null;index;uniqueIndex:uidx_private_photo_access_owner_viewer,priority:1" json:"owner_id"`
	ViewerID uuid.UUID                            `gorm:"type:uuid;not null;index;uniqueIndex:uidx_private_photo_access_owner_viewer,priority:2;check:chk_private_photo_access_distinct_users,owner_id <> viewer_id" json:"viewer_id"`
	Status   domainmedia.PrivatePhotoAccessStatus `gorm:"type:varchar(16);not null;default:'pending';index;check:chk_private_photo_access_status,status IN ('pending','approved','denied')" json:"status"`

	Owner  User `gorm:"foreignKey:OwnerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`
	Viewer User `gorm:"foreignKey:ViewerID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"-"`

	RequestedAt time.Time  `gorm:"not null;index" json:"requested_at"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (PrivatePhotoAccessRequest) TableName() string {
	return "private_photo_access_requests"
}
