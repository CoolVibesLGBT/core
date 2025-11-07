package types

import (
	"coolvibes/models/media"
	"coolvibes/models/post"
	userModel "coolvibes/models/user"
	"strconv"
)

type Int64String int64

type TimelineResult struct {
	Posts      []post.Post `json:"posts"`
	NextCursor *string     `json:"next_cursor"`
}
type MediaWithUser struct {
	media.Media `json:",inline"` // embedded struct, alanları direkt üstte olacak
	User        userModel.User   `gorm:"embedded;embeddedPrefix:user_" json:"user"`
	NextCursor  *int64           `json:"next_cursor"` // Bir sonraki sayfa için cursor (PublicID)
}

func (i Int64String) MarshalJSON() ([]byte, error) {
	s := strconv.FormatInt(int64(i), 10)
	return []byte(`"` + s + `"`), nil
}
