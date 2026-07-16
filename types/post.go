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

type UsersResult struct {
	Users  []post.Post `json:"users"`
	Cursor *string     `json:"cursor"`
}

type SearchResult struct {
	Posts PostsResult `json:"posts"`
	Users UsersResult `json:"users"`
}

// GlobalSearchResult is the stable response contract used by the web search
// screen. Each collection is kept separate so the client can render filters
// without having to infer a result type from a polymorphic payload.
type GlobalSearchResult struct {
	Query  string        `json:"query"`
	Users  []models.User `json:"users"`
	Events []post.Post   `json:"events"`
	Posts  []post.Post   `json:"posts"`
	Places []*post.Post  `json:"places"`
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
