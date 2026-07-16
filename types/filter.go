package types

import (
	"context"
	"core/models"
	"core/models/post"

	"github.com/google/uuid"
)

type Filter struct {
	AuthUser *models.User
	Context  context.Context
	Domain   *string

	PostUUID uuid.UUID
	PostID   int64 // snowflakeid
	IsLive   *bool
	PostKind post.PostKind
	UserUUID uuid.UUID
	UserID   int64
	UserName *string
	Search   *string
	Slug     *string
	Category *string

	Pillar  *string
	Cluster *string

	Name      *string
	City      *string
	Country   *string
	Latitude  *float64
	Longitude *float64
	Distance  *float64

	Cursor  *int64
	Page    *int
	PerPage *int
	Mode    string // "offset" | "cursor"
	Limit   int
}

func (f Filter) PillarStr() string {
	if f.Pillar == nil {
		return ""
	}
	return *f.Pillar
}

func (f Filter) ClusterStr() string {
	if f.Cluster == nil {
		return ""
	}
	return *f.Cluster
}

func (f Filter) SlugStr() string {
	if f.Slug == nil {
		return ""
	}
	return *f.Slug
}
