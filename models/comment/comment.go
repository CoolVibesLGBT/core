package comment

import (
	"coolvibes/models/user"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	CommentTargetBlog    = "blog"
	CommentTargetPost    = "post"
	CommentTargetNews    = "news"
	CommentTargetVideo   = "video"
	CommentTargetPhoto   = "photo"
	CommentTargetProduct = "product"
	CommentTargetUser    = "user"
)

const (
	InteractionLike    = "like"
	InteractionDislike = "dislike"
	InteractionStar    = "star"
	InteractionGift    = "gift"
)

type Comment struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	AuthorID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	TargetID   uuid.UUID  `gorm:"type:uuid;not null;index"`        // İçeriğin ID'si
	TargetType string     `gorm:"type:varchar(32);not null;index"` // İçerik tipi: blog, video, product vs.
	ParentID   *uuid.UUID `gorm:"type:uuid;index"`                 // Üst yorum varsa
	Content    string     `gorm:"type:text;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`

	// Relations
	Author       user.User            `gorm:"foreignKey:AuthorID;constraint:OnDelete:CASCADE"`
	Parent       *Comment             `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
	Replies      []Comment            `gorm:"foreignKey:ParentID;constraint:OnDelete:CASCADE"`
	Interactions []CommentInteraction `gorm:"foreignKey:CommentID"`
}

type CommentInteraction struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	CommentID uuid.UUID `gorm:"type:uuid;not null;index"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Type      string    `gorm:"type:varchar(16);not null;index"` // like, dislike, star, gift
	Metadata  *string   `gorm:"type:text"`                       // opsiyonel: mesaj, token, değer vs.
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relations
	User    user.User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Comment Comment   `gorm:"foreignKey:CommentID;constraint:OnDelete:CASCADE"`
}

func (Comment) TableName() string {
	return "shared_comments"
}

func (CommentInteraction) TableName() string {
	return "shared_comment_interactions"
}
