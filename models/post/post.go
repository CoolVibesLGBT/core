package post

import (
	"bifrost/models"
	"bifrost/models/media"
	"bifrost/models/user"

	"encoding/json"
	"strconv"
	"time"

	"bifrost/models/post/payloads"
	"bifrost/models/post/shared"
	global_shared "bifrost/models/shared"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostType string
type ContentCategory string

const (
	PostTypeStatus     PostType = "status"
	PostTypeTimeline   PostType = "timeline"
	PostTypePlace      PostType = "place"
	PostTypeClassified PostType = "classified"
	PostTypeGeneric    PostType = "generic"
	PostTypeNews       PostType = "news"
	PostTypeStory      PostType = "story"
	PostTypeChat       PostType = "chat"
)

const (
	ContentNormal       ContentCategory = "normal"       // Standart içerik
	ContentErotic       ContentCategory = "erotic"       // Erotik / yetişkin içerik
	ContentViolence     ContentCategory = "violence"     // Şiddet içerik
	ContentSpam         ContentCategory = "spam"         // Reklam / spam
	ContentPolitical    ContentCategory = "political"    // Politik içerik
	ContentSensitive    ContentCategory = "sensitive"    // Hassas konular (ör: depresyon, travma)
	ContentNSFW         ContentCategory = "nsfw"         // 18+ genel içerik
	ContentSelfPromo    ContentCategory = "self_promo"   // Kendi reklamı / promosyon
	ContentEvent        ContentCategory = "event"        // Etkinlik duyurusu
	ContentAnnouncement ContentCategory = "announcement" // Duyuru
	ContentReview       ContentCategory = "review"       // Yorum / inceleme
	ContentNews         ContentCategory = "news"         // Haber içerik
	ContentArt          ContentCategory = "art"          // Sanat / görsel içerik
	ContentTutorial     ContentCategory = "tutorial"     // Eğitim / rehber
	ContentOther        ContentCategory = "other"        // Diğer
)

type Post struct {
	ID       uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ParentID *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Children []Post     `gorm:"foreignKey:ParentID" json:"children,omitempty"`

	PublicID int64 `gorm:"uniqueIndex;not null" json:"public_id"`

	// 🔸 PostType alanının yeni ismi
	PostKind PostType `gorm:"size:50;not null;index;default:'status'" json:"post_kind"`

	// 🔹 İçerik kategorisi
	ContentCategory ContentCategory `gorm:"size:50;not null;index;default:'normal'" json:"content_category"`

	ContentableID   *uuid.UUID `gorm:"type:uuid;index" json:"contentable_id,omitempty"`
	ContentableType *string    `gorm:"size:50;index" json:"contentable_type,omitempty"`

	AuthorID uuid.UUID `gorm:"type:uuid;index;not null" json:"author_id"`

	Title   *shared.LocalizedString `gorm:"type:jsonb" json:"title,omitempty"`
	Slug    *string                 `gorm:"size:255;uniqueIndex" json:"slug,omitempty"`
	Content *shared.LocalizedString `gorm:"type:jsonb" json:"content,omitempty"`
	Summary *shared.LocalizedString `gorm:"type:jsonb" json:"summary,omitempty"`

	Published   bool           `gorm:"default:false;index" json:"published"`
	PublishedAt *time.Time     `gorm:"index" json:"published_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Extras *map[string]any `gorm:"type:jsonb" json:"extras,omitempty"`

	Author user.User `gorm:"foreignKey:AuthorID;references:ID" json:"author"`

	Attachments []*media.Media `gorm:"polymorphic:Owner;polymorphicValue:post;constraint:OnDelete:CASCADE" json:"attachments,omitempty"`

	//Mentions []*models.Mention `gorm:"polymorphic:Mentionable;polymorphicValue:post" json:"mentions,omitempty"`
	Mentions []*models.Mention `gorm:"polymorphic:Mentionable;polymorphicValue:post;constraint:OnDelete:CASCADE" json:"mentions,omitempty"`
	Hashtags []*models.Hashtag `gorm:"polymorphic:Taggable;polymorphicValue:post;constraint:OnDelete:CASCADE" json:"hashtags,omitempty"`

	Poll  []*payloads.Poll `gorm:"polymorphic:Contentable;constraint:OnDelete:CASCADE" json:"poll,omitempty"`
	Event *payloads.Event  `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE" json:"event,omitempty"`

	Location    *global_shared.Location `gorm:"polymorphic:Contentable;polymorphicValue:post;constraint:OnDelete:CASCADE;" json:"location,omitempty"`
	Contentable any                     `gorm:"-" json:"contentable,omitempty"`
}

func (Post) TableName() string {
	return "posts"
}

func (u Post) MarshalJSON() ([]byte, error) {
	type Alias Post // recursive çağrıyı önlemek için alias
	aux := struct {
		PublicID string `json:"public_id"`
		Alias
	}{
		PublicID: strconv.FormatInt(u.PublicID, 10),
		Alias:    (Alias)(u),
	}

	return json.Marshal(aux)
}
