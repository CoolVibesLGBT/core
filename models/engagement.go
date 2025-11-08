package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Engagement struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	// Polymorphic ilişki için contentable
	ContentableID   uuid.UUID `gorm:"type:uuid;index;not null" json:"contentable_id"`
	ContentableType string    `gorm:"size:255;index;not null" json:"contentable_type"`

	LikeCount     int64 `gorm:"default:0" json:"like_count"`
	CommentCount  int64 `gorm:"default:0" json:"comment_count"`
	ViewCount     int64 `gorm:"default:0" json:"view_count"`
	BookmarkCount int64 `gorm:"default:0" json:"bookmark_count"`

	RatingCount int64 `gorm:"default:0" json:"rating_count"`
	RatingSum   int64 `gorm:"default:0" json:"rating_sum"`

	ReportCount         int64 `gorm:"default:0;index" json:"report_count"`
	ReportUpvoteCount   int64 `gorm:"default:0;index" json:"report_upvote_count"`
	ReportDownvoteCount int64 `gorm:"default:0;index" json:"report_downvote_count"`

	TipCount  int64           `gorm:"default:0" json:"tip_count"`
	TipAmount decimal.Decimal `gorm:"type:numeric(38,18);default:0" json:"tip_amount"`

	Kind string `gorm:"size:50;index;not null" json:"kind"`

	Details []EngagementDetail `gorm:"foreignKey:EngagementID" json:"details,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type EngagementDetail struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`

	EngagementID uuid.UUID  `gorm:"type:uuid;index;not null" json:"engagement_id"`
	Engagement   Engagement `gorm:"foreignKey:EngagementID;constraint:OnDelete:CASCADE" json:"-"`

	UserID uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"` // Kim yaptı
	User   User      `gorm:"foreignKey:UserID" json:"user"`

	TargetUserID *uuid.UUID `gorm:"type:uuid;index" json:"target_user_id,omitempty"` // Kime yapıldı (isteğe bağlı)
	TargetUser   *User      `gorm:"foreignKey:TargetUserID" json:"target_user,omitempty"`

	Kind string `gorm:"size:50;index;not null" json:"kind"`

	TipAmount *decimal.Decimal `gorm:"type:numeric(38,18);default:null" json:"tip_amount,omitempty"`

	Rating *int8 `gorm:"default:null" json:"rating,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}
