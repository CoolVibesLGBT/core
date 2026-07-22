package ports

import (
	"core/application/types"

	"github.com/google/uuid"
)

// PublicPostReader is the query-side port used by public post HTTP actions.
// Its result contract is persistence-free: adapters cannot return models.Post,
// models.User or models.Media across this boundary.
type PublicPostReader interface {
	FindPublicPostByID(id uuid.UUID) (*types.PublicPost, error)
	FindPublicPostBySlug(filters types.Filter) (*types.PublicPost, error)
	FindPublicPostByPublicID(id int64) (*types.PublicPost, error)
	FetchPublicTimeline(filters types.Filter) (types.PublicPostPage, error)
	SearchPublicPosts(filters types.Filter) (types.PublicPostPage, error)
	FetchPublicUserPosts(userID uuid.UUID, filters types.Filter) ([]types.PublicPost, error)
	FetchPublicUserPostReplies(filters types.Filter) ([]types.PublicPost, error)
	FetchPublicUserMedia(filters types.Filter) (types.PublicPostMediaPage, error)
	FetchPublicTimelineVibes(filters types.Filter) (types.PublicPostPage, error)
}
