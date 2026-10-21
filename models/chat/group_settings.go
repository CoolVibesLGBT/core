package chat

import (
	"errors"
	"github.com/google/uuid"
	"regexp"
	"strings"
	"time"
)

var ErrInvalidGroupSettings = errors.New("invalid group settings")
var ErrGroupLinkInvalid = errors.New("group invite link is invalid")
var publicGroupName = regexp.MustCompile(`^[a-z][a-z0-9_]{4,31}$`)

type GroupOptions struct {
	Visibility     string
	PublicUsername string
}

func (o GroupOptions) Normalize() (string, *string, error) {
	visibility := strings.TrimSpace(o.Visibility)
	if visibility == "" {
		visibility = "private"
	}
	if visibility == "private" {
		return visibility, nil, nil
	}
	name := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(o.PublicUsername), "@"))
	if visibility != "public" || !publicGroupName.MatchString(name) {
		return "", nil, ErrInvalidGroupSettings
	}
	return visibility, &name, nil
}

type GroupInvite struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	ChatID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"-"`
	Token     string    `gorm:"size:64;uniqueIndex;not null" json:"-"`
	CreatedBy uuid.UUID `gorm:"type:uuid;not null" json:"-"`
	Uses      int64     `gorm:"not null;default:0" json:"uses"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
