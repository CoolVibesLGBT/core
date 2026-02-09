package post

import (
	"coolvibes/models"
	"coolvibes/models/media"
	"coolvibes/models/taxonomy"
	"coolvibes/models/utils"

	"encoding/json"
	"strconv"
	"time"

	"coolvibes/models/post/payloads"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PostKind string
type ContentCategory string

const (
	PostKindStatus     PostKind = "status"
	PostKindTimeline   PostKind = "timeline"
	PostKindPlace      PostKind = "place"
	PostKindClassified PostKind = "classified"
	PostKindGeneric    PostKind = "generic"
	PostKindNews       PostKind = "news"
	PostKindStory      PostKind = "story"
	PostKindChat       PostKind = "chat"
	PostKindMessage    PostKind = "message"
	PostKindPost       PostKind = "post"
	PostKindEvent      PostKind = "event"
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
	Parent   *Post      `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children []Post     `gorm:"foreignKey:ParentID" json:"children,omitempty"`

	PublicID int64 `gorm:"uniqueIndex;not null" json:"public_id"`

	PostKind PostKind          `gorm:"size:50;not null;index;default:'post'" json:"post_kind"`
	Domain   models.DomainKind `gorm:"size:50;not null;index;default:'coolvibes.lgbt'" json:"domain"`

	ContentCategory ContentCategory `gorm:"size:50;not null;index;default:'normal'" json:"content_category"`

	ContentableID   *uuid.UUID `gorm:"type:uuid;index" json:"contentable_id,omitempty"`
	ContentableType *string    `gorm:"size:50;index" json:"contentable_type,omitempty"`

	AuthorID uuid.UUID `gorm:"type:uuid;index;not null" json:"author_id"`

	Title       *utils.LocalizedString `gorm:"type:jsonb" json:"title,omitempty"`
	Slug        *string                `gorm:"size:255;index" json:"slug,omitempty"`
	Content     *utils.LocalizedString `gorm:"type:jsonb" json:"content,omitempty"`
	Summary     *utils.LocalizedString `gorm:"type:jsonb" json:"summary,omitempty"`
	Audience    *string                `gorm:"size:50;index;default:'public'" json:"audience,omitempty"`
	Published   bool                   `gorm:"default:false;index" json:"published"`
	PublishedAt *time.Time             `gorm:"index" json:"published_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   gorm.DeletedAt         `gorm:"index" json:"deleted_at,omitempty"`

	//Extras *map[string]any `gorm:"type:jsonb" json:"extras,omitempty"`
	Extras datatypes.JSON `gorm:"type:jsonb" json:"extras,omitempty"`
	Author models.User    `gorm:"foreignKey:AuthorID;references:ID" json:"author"`

	Clusters    []taxonomy.Cluster `gorm:"many2many:post_clusters;" json:"clusters,omitempty"`
	Attachments []*media.Media     `gorm:"polymorphic:Owner;polymorphicValue:post;constraint:OnDelete:CASCADE" json:"attachments,omitempty"`
	Mentions    []*models.Mention  `gorm:"polymorphic:Mentionable;polymorphicValue:post;constraint:OnDelete:CASCADE" json:"mentions,omitempty"`
	Hashtags    []*models.Hashtag  `gorm:"polymorphic:Taggable;polymorphicValue:post;constraint:OnDelete:CASCADE" json:"hashtags,omitempty"`

	Poll  []*payloads.Poll `gorm:"polymorphic:Contentable;polymorphicValue:post;constraint:OnDelete:CASCADE" json:"poll,omitempty"`
	Event *payloads.Event  `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE" json:"event,omitempty"`

	Location    *utils.Location `gorm:"polymorphic:Contentable;polymorphicValue:post;constraint:OnDelete:CASCADE;" json:"location,omitempty"`
	Contentable any             `gorm:"-" json:"contentable,omitempty"`

	//	Engagements *models.Engagement `gorm:"polymorphic:Contentable;constraint:OnDelete:CASCADE" json:"engagements,omitempty"`
	Engagements *models.Engagement `gorm:"polymorphic:Contentable;polymorphicValue:post;constraint:OnDelete:CASCADE" json:"engagements,omitempty"`
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
