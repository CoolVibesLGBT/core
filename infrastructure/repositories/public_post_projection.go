package repositories

import (
	legacyviews "core/application/legacyviews"
	"core/application/ports"
	"core/application/types"

	"github.com/google/uuid"
)

var _ ports.PublicPostReader = (*PostRepository)(nil)

func (r *PostRepository) FindPublicPostByID(id uuid.UUID) (*types.PublicPost, error) {
	result, err := r.GetPostByID(id)
	if err != nil {
		return nil, err
	}
	projected := legacyviews.ProjectPublicPost(*result)
	return &projected, nil
}

func (r *PostRepository) FindPublicPostBySlug(filters types.Filter) (*types.PublicPost, error) {
	result, err := r.GetPostBySlug(filters)
	if err != nil {
		return nil, err
	}
	projected := legacyviews.ProjectPublicPost(*result)
	return &projected, nil
}

func (r *PostRepository) FindPublicPostByPublicID(id int64) (*types.PublicPost, error) {
	result, err := r.GetPostByPublicID(id)
	if err != nil {
		return nil, err
	}
	projected := legacyviews.ProjectPublicPost(*result)
	return &projected, nil
}

func (r *PostRepository) FetchPublicTimeline(filters types.Filter) (types.PublicPostPage, error) {
	result, err := r.GetTimeline(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return legacyviews.ProjectPublicPostPage(result), nil
}

func (r *PostRepository) SearchPublicPosts(filters types.Filter) (types.PublicPostPage, error) {
	result, err := r.FindPostsByKind(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return legacyviews.ProjectPublicPostsResult(result), nil
}

func (r *PostRepository) FetchPublicUserPosts(userID uuid.UUID, filters types.Filter) ([]types.PublicPost, error) {
	result, err := r.GetUserPosts(userID, filters)
	if err != nil {
		return nil, err
	}
	return legacyviews.ProjectPublicPosts(result), nil
}

func (r *PostRepository) FetchPublicUserPostReplies(filters types.Filter) ([]types.PublicPost, error) {
	result, err := r.GetUserPostReplies(filters)
	if err != nil {
		return nil, err
	}
	return legacyviews.ProjectPublicPosts(result), nil
}

func (r *PostRepository) FetchPublicUserMedia(filters types.Filter) (types.PublicPostMediaPage, error) {
	result, next, err := r.GetUserMedias(filters)
	if err != nil {
		return types.PublicPostMediaPage{}, err
	}
	return types.PublicPostMediaPage{
		Items:        legacyviews.ProjectPublicMediaItems(result),
		NextPublicID: next,
	}, nil
}

func (r *PostRepository) FetchPublicTimelineVibes(filters types.Filter) (types.PublicPostPage, error) {
	result, err := r.GetTimelineVibes(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return legacyviews.ProjectPublicPostPage(result), nil
}
