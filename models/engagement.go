package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

// EngagementKind enumu
type EngagementKind string

const (
	EngagementKindTouch           EngagementKind = "touch"
	EngagementKindBanana          EngagementKind = "banana"
	EngagementKindCarrot          EngagementKind = "carrot"
	EngagementKindCoffee          EngagementKind = "coffee"
	EngagementKindKiss            EngagementKind = "kiss"
	EngagementKindLikeGiven       EngagementKind = "like_given"       // Kullanıcının beğendikleri
	EngagementKindLikeReceived    EngagementKind = "like_received"    // Kullanıcıyı beğenenler
	EngagementKindDislikeGiven    EngagementKind = "dislike_given"    // Kullanıcının beğendikleri
	EngagementKindDisLikeReceived EngagementKind = "dislike_received" // Kullanıcıyı beğenenler
	EngagementKindPost            EngagementKind = "post"
	EngagementKindComment         EngagementKind = "comment"
	EngagementKindFollower        EngagementKind = "follower"
	EngagementKindFollowing       EngagementKind = "following"
	EngagementKindBlockedBy       EngagementKind = "blocked_by" // seni engelleyenler
	EngagementKindBlocking        EngagementKind = "blocking"   // senin engellediklerin
	EngagementKindView            EngagementKind = "view"
	EngagementKindBookmark        EngagementKind = "bookmark"
	EngagementKindRating          EngagementKind = "rating"
	EngagementTip                 EngagementKind = "tip"
	EngagementKindGift            EngagementKind = "gift"
	EngagementKindReport          EngagementKind = "report"
	EngagementKindDeposit         EngagementKind = "deposit"
	EngagementKindWithdraw        EngagementKind = "withdraw"
)

var EngagementCountKeys = map[EngagementKind]struct {
	CountKey  string
	AmountKey string // boşsa yok demek
}{
	EngagementKindTouch:  {"touch_count", ""},
	EngagementKindBanana: {"banana_count", ""},
	EngagementKindCarrot: {"carrot_count", ""},
	EngagementKindCoffee: {"coffee_count", ""},
	EngagementKindKiss:   {"kiss_count", ""},

	EngagementKindLikeGiven:    {"like_given_count", ""},
	EngagementKindLikeReceived: {"like_received_count", ""},

	EngagementKindDislikeGiven:    {"dislike_given_count", ""},
	EngagementKindDisLikeReceived: {"dislike_received_count", ""},

	EngagementKindPost:      {"post_count", ""},
	EngagementKindComment:   {"comment_count", ""},
	EngagementKindFollower:  {"follower_count", ""},
	EngagementKindFollowing: {"following_count", ""},

	EngagementKindBlockedBy: {"blocked_by_count", ""},
	EngagementKindBlocking:  {"blocking_count", ""},

	EngagementKindView:     {"view_count", ""},
	EngagementKindBookmark: {"bookmark_count", ""},
	EngagementKindRating:   {"rating_count", "rating_sum"},
	EngagementTip:          {"tip_count", "tip_amount"},
	EngagementKindGift:     {"gift_count", "gift_amount"},
	EngagementKindReport:   {"report_count", ""},
	EngagementKindDeposit:  {"deposit_count", "deposit_amount"},
	EngagementKindWithdraw: {"withdraw_count", "withdraw_amount"},
}

func NewCountsMap() map[string]interface{} {
	counts := make(map[string]interface{})
	for _, v := range EngagementCountKeys {
		counts[v.CountKey] = int64(0)
		if v.AmountKey != "" {
			counts[v.AmountKey] = decimal.NewFromInt(0)
		}
	}

	return counts
}

type Engagement struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ContentableID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"contentable_id"`
	ContentableType string         `gorm:"type:varchar(50);not null;index" json:"contentable_type"`
	Counts          datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"counts"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	EngagementDetails []EngagementDetail `gorm:"foreignKey:EngagementID;constraint:OnDelete:CASCADE;" json:"engagement_details,omitempty"`
}

type EngagementDetail struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EngagementID uuid.UUID `gorm:"type:uuid;not null;index" json:"engagement_id"`

	EngagerID uuid.UUID `gorm:"type:uuid;not null;index" json:"engager_id"`
	Engager   User      `gorm:"foreignKey:EngagerID" json:"engager,omitempty"`

	RecipientID uuid.UUID      `gorm:"type:uuid;index" json:"recipient_id,omitempty"`
	Recipient   User           `gorm:"foreignKey:RecipientID" json:"recipient,omitempty"`
	Kind        EngagementKind `gorm:"type:varchar(50);not null;index" json:"kind"`
	Details     datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"details"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}
