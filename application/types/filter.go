package types

import (
	"context"
	domainpost "core/domain/post"

	"github.com/google/uuid"
)

// Actor is the minimal authenticated identity carried by application query
// objects. Persistence entities must not cross this boundary.
type Actor struct {
	ID       uuid.UUID
	PublicID int64
	Role     string
}

type Filter struct {
	AuthUser *Actor
	Context  context.Context
	Domain   *string

	PostUUID uuid.UUID
	PostID   int64 // snowflakeid
	IsLive   *bool
	PostKind domainpost.Kind
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
