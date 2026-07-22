package usecases

import (
	"core/application/ports"
	"core/application/types"
	"testing"

	"github.com/google/uuid"
)

type projectionAwarePostRepository struct {
	*fakePostRepository
	timelineCalled bool
}

var _ ports.PublicPostReader = (*projectionAwarePostRepository)(nil)

func (r *projectionAwarePostRepository) FindPublicPostByID(id uuid.UUID) (*types.PublicPost, error) {
	return &types.PublicPost{ID: 1, PublicID: 1}, nil
}

func (r *projectionAwarePostRepository) FindPublicPostBySlug(filters types.Filter) (*types.PublicPost, error) {
	return &types.PublicPost{ID: 2, PublicID: 2}, nil
}

func (r *projectionAwarePostRepository) FindPublicPostByPublicID(id int64) (*types.PublicPost, error) {
	return &types.PublicPost{ID: types.SnowflakeID(id), PublicID: types.SnowflakeID(id)}, nil
}

func (r *projectionAwarePostRepository) FetchPublicTimeline(filters types.Filter) (types.PublicPostPage, error) {
	r.timelineCalled = true
	return types.PublicPostPage{Posts: []types.PublicPost{{ID: 10, PublicID: 10}}}, nil
}

func (r *projectionAwarePostRepository) SearchPublicPosts(filters types.Filter) (types.PublicPostPage, error) {
	return types.PublicPostPage{}, nil
}

func (r *projectionAwarePostRepository) FetchPublicUserPosts(userID uuid.UUID, filters types.Filter) ([]types.PublicPost, error) {
	return nil, nil
}

func (r *projectionAwarePostRepository) FetchPublicUserPostReplies(filters types.Filter) ([]types.PublicPost, error) {
	return nil, nil
}

func (r *projectionAwarePostRepository) FetchPublicUserMedia(filters types.Filter) (types.PublicPostMediaPage, error) {
	return types.PublicPostMediaPage{}, nil
}

func (r *projectionAwarePostRepository) FetchPublicTimelineVibes(filters types.Filter) (types.PublicPostPage, error) {
	return types.PublicPostPage{}, nil
}

func TestPostServiceUsesPersistenceFreePublicPostReaderPort(t *testing.T) {
	repository := &projectionAwarePostRepository{fakePostRepository: &fakePostRepository{}}
	service := NewPostService(&fakeUserRepository{}, repository, &fakeMediaRepository{})

	page, err := service.GetTimeline(types.Filter{Limit: 10})
	if err != nil {
		t.Fatalf("GetTimeline() error = %v", err)
	}
	if !repository.timelineCalled || len(page.Posts) != 1 || page.Posts[0].PublicID != 10 {
		t.Fatalf("public reader was not used: called=%v page=%#v", repository.timelineCalled, page)
	}
	if repository.timelineFilter.Limit != 0 {
		t.Fatalf("legacy raw post reader was called: %#v", repository.timelineFilter)
	}
}
