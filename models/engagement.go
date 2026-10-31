package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

type EngagementContentableType string

const (
	EngagementContentableTypeUser    EngagementContentableType = "user"
	EngagementContentableTypePost    EngagementContentableType = "post"
	EngagementContentableTypeChat    EngagementContentableType = "chat"
	EngagementContentableTypeMessage EngagementContentableType = "message"
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
	EngagementKindMatched         EngagementKind = "matched"          // Kullanıcının eslestikleri
	EngagementKindReferral        EngagementKind = "referral"         // referral işlemi

	EngagementKindViewGiven    EngagementKind = "view_given"    // Görüntüleyen kişi (benim baktıklarım)
	EngagementKindViewReceived EngagementKind = "view_received" // Görülen kişi (bana bakanlar)

	EngagementKindPost      EngagementKind = "post"
	EngagementKindComment   EngagementKind = "comment"
	EngagementKindFollower  EngagementKind = "follower"
	EngagementKindFollowing EngagementKind = "following"
	EngagementKindBlockedBy EngagementKind = "blocked_by" // seni engelleyenler
	EngagementKindBlocking  EngagementKind = "blocking"   // senin engellediklerin
	EngagementKindView      EngagementKind = "view"

	EngagementKindSubscribedBy EngagementKind = "subscribed_by" // seni takip edenler (aboneler)
	EngagementKindSubscribing  EngagementKind = "subscribing"   // senin abone oldukların

	EngagementKindBookmark EngagementKind = "bookmark"
	EngagementKindRating   EngagementKind = "rating"
	EngagementKindTip      EngagementKind = "tip"

	EngagementKindGift     EngagementKind = "gift"
	EngagementKindReport   EngagementKind = "report"
	EngagementKindDeposit  EngagementKind = "deposit"
	EngagementKindWithdraw EngagementKind = "withdraw"

	EngagementKindMessageDeletedForMe  EngagementKind = "message_deleted_for_me"
	EngagementKindMessageDeletedForAll EngagementKind = "message_deleted_for_all"

	EngagementKindChatDeletedForMe  EngagementKind = "chat_deleted_for_me"
	EngagementKindChatDeletedForAll EngagementKind = "chat_deleted_for_all"

	EngagementKindChatMessageRead EngagementKind = "chat_mark_message_read"

	// EngagementKindChatMessageOpened records a disappearing/view-once
	// message's one-time consumption per recipient.
	EngagementKindChatMessageOpened EngagementKind = "chat_message_opened"

	// EngagementKindFriendship stores the single canonical relationship row
	// for a user pair. Unlike toggleable kinds, its EngagementDetail.Details
	// JSON carries a status state machine (pending/accepted/denied/
	// cancelled/removed) plus the requester, so the row persists across
	// terminal states instead of being deleted like a toggle.
	EngagementKindFriendship EngagementKind = "friendship"

	// EngagementKindPrivatePhotoAccess stores one row per (owner, viewer)
	// direction — EngagerID is always the album owner, EngageeID the
	// viewer. Its Details JSON carries the pending/approved/denied status
	// state machine, so (unlike a toggle) the row persists across terminal
	// states instead of being deleted.
	EngagementKindPrivatePhotoAccess EngagementKind = "private_photo_access"
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
	EngagementKindMatched:      {"match_count", ""},

	EngagementKindViewGiven:    {"view_given_count", ""},
	EngagementKindViewReceived: {"view_received_count", ""},

	EngagementKindDislikeGiven:    {"dislike_given_count", ""},
	EngagementKindDisLikeReceived: {"dislike_received_count", ""},

	EngagementKindPost:      {"post_count", ""},
	EngagementKindComment:   {"comment_count", ""},
	EngagementKindFollower:  {"follower_count", ""},
	EngagementKindFollowing: {"following_count", ""},

	EngagementKindBlockedBy: {"blocked_by_count", ""},
	EngagementKindBlocking:  {"blocking_count", ""},

	EngagementKindView:         {"view_count", ""},
	EngagementKindBookmark:     {"bookmark_count", ""},
	EngagementKindSubscribedBy: {"subscribed_by_count", ""},
	EngagementKindSubscribing:  {"subscribing_count", ""},

	EngagementKindRating:   {"rating_count", "rating_sum"},
	EngagementKindTip:      {"tip_count", "tip_amount"},
	EngagementKindGift:     {"gift_count", "gift_amount"},
	EngagementKindReport:   {"report_count", ""},
	EngagementKindDeposit:  {"deposit_count", "deposit_amount"},
	EngagementKindWithdraw: {"withdraw_count", "withdraw_amount"},

	EngagementKindReferral:             {"referral_count", "referral_amount"},
	EngagementKindMessageDeletedForMe:  {"message_deleted_for_me_count", ""},
	EngagementKindMessageDeletedForAll: {"message_deleted_for_all_count", ""},
	EngagementKindChatDeletedForMe:     {"chat_deleted_for_me_count", ""},
	EngagementKindChatDeletedForAll:    {"chat_deleted_for_all_count", ""},
	EngagementKindChatMessageRead:      {"chat_message_read_count", ""},
	EngagementKindChatMessageOpened:    {"chat_message_opened_count", ""},
	EngagementKindFriendship:           {"friendship_count", ""},
	EngagementKindPrivatePhotoAccess:   {"private_photo_access_count", ""},
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
	ID              uuid.UUID                 `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ContentableID   uuid.UUID                 `gorm:"type:uuid;not null;index" json:"contentable_id"`
	ContentableType EngagementContentableType `gorm:"type:varchar(50);not null;index" json:"contentable_type"`
	Counts          datatypes.JSON            `gorm:"type:jsonb;default:'{}'" json:"counts"`
	CreatedAt       time.Time                 `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time                 `gorm:"autoUpdateTime" json:"updated_at"`

	EngagementDetails []EngagementDetail `gorm:"foreignKey:EngagementID;constraint:OnDelete:CASCADE;" json:"engagement_details,omitempty"`
}

type EngagementDetail struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EngagementID uuid.UUID      `gorm:"type:uuid;not null;index" json:"engagement_id"`
	DedupeKey    *string        `gorm:"type:varchar(255);uniqueIndex" json:"-"`
	EngagerID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"engager_id"` //etkilesimi yapan ornegin: takip eden
	Engager      User           `gorm:"foreignKey:EngagerID" json:"engager,omitempty"`
	EngageeID    uuid.UUID      `gorm:"type:uuid;index" json:"engagee_id,omitempty"` //etkilesimi alan ornegin: takip edilen
	Engagee      User           `gorm:"foreignKey:EngageeID" json:"engagee,omitempty"`
	Kind         EngagementKind `gorm:"type:varchar(50);not null;index" json:"kind"`
	Details      datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"details"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (e EngagementKind) IsValid() bool {
	switch e {
	case
		EngagementKindTouch,
		EngagementKindBanana,
		EngagementKindCarrot,
		EngagementKindCoffee,
		EngagementKindKiss,
		EngagementKindLikeGiven,
		EngagementKindLikeReceived,
		EngagementKindDislikeGiven,
		EngagementKindDisLikeReceived,
		EngagementKindMatched,
		EngagementKindReferral,
		EngagementKindViewGiven,
		EngagementKindViewReceived,
		EngagementKindPost,
		EngagementKindComment,
		EngagementKindFollower,
		EngagementKindFollowing,
		EngagementKindBlockedBy,
		EngagementKindBlocking,
		EngagementKindView,
		EngagementKindBookmark,
		EngagementKindSubscribedBy,
		EngagementKindSubscribing,
		EngagementKindRating,
		EngagementKindTip,
		EngagementKindGift,
		EngagementKindReport,
		EngagementKindDeposit,
		EngagementKindWithdraw,
		EngagementKindMessageDeletedForMe,
		EngagementKindMessageDeletedForAll,
		EngagementKindChatDeletedForMe,
		EngagementKindChatDeletedForAll,
		EngagementKindChatMessageRead,
		EngagementKindChatMessageOpened,
		EngagementKindFriendship,
		EngagementKindPrivatePhotoAccess:
		return true
	default:
		return false
	}
}
