package types

import (
	"core/models"
	"core/models/media"
	"core/models/post"
	"strconv"
)

type Int64String int64

type TimelineResult struct {
	Posts  []post.Post `json:"posts"`
	Cursor *string     `json:"cursor"`
}

type PostsResult struct {
	Posts  []post.Post `json:"posts"`
	Cursor *string     `json:"cursor"`
}

type SearchResult struct {
	Posts  []post.Post   `json:"posts"`
	User   []models.User `json:"users"`
	Cursor *string       `json:"cursor"`
}

type MediaWithUser struct {
	media.Media `json:",inline"` // embedded struct, alanları direkt üstte olacak
	User        models.User      `gorm:"embedded;embeddedPrefix:user_" json:"user"`
	Cursor      *int64           `json:"cursor"` // Bir sonraki sayfa için cursor (PublicID)
}

func (i Int64String) MarshalJSON() ([]byte, error) {
	s := strconv.FormatInt(int64(i), 10)
	return []byte(`"` + s + `"`), nil
}
