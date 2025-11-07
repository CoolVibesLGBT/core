package types

import (
	"coolvibes/models/media"
	"coolvibes/models/post"
	userModel "coolvibes/models/user"
)

type TimelineResult struct {
	Posts      []post.Post `json:"posts"`       // Döndürülen postlar
	NextCursor *int64      `json:"next_cursor"` // Bir sonraki sayfa için cursor (PublicID)
}

type MediaWithUser struct {
	media.Media `json:",inline"` // embedded struct, alanları direkt üstte olacak
	User        userModel.User   `gorm:"embedded;embeddedPrefix:user_" json:"user"`
}
