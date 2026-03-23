package repositories

import (
	"context"
	"core/models/post"
	"core/models/taxonomy"
	"core/types"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewsRepositoryGetRejectsNonNewsPosts(t *testing.T) {
	newsID := uuid.New()
	placeID := uuid.New()

	repo := &NewsRepository{
		postRepo: newsPostRepositoryStub{
			byUUID: map[uuid.UUID]*post.Post{
				newsID:  {ID: newsID, PostKind: post.PostKindNews},
				placeID: {ID: placeID, PostKind: post.PostKindPlace},
			},
			byPublicID: map[int64]*post.Post{
				1001: {ID: newsID, PublicID: 1001, PostKind: post.PostKindNews},
				2002: {ID: placeID, PublicID: 2002, PostKind: post.PostKindPlace},
			},
		},
	}

	t.Run("returns news by uuid", func(t *testing.T) {
		postData, err := repo.Get(types.Filter{PostUUID: newsID})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if postData == nil || postData.PostKind != post.PostKindNews {
			t.Fatalf("expected news post, got %#v", postData)
		}
	})

	t.Run("returns news by public id", func(t *testing.T) {
		postData, err := repo.Get(types.Filter{PostID: 1001})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if postData == nil || postData.PostKind != post.PostKindNews {
			t.Fatalf("expected news post, got %#v", postData)
		}
	})

	t.Run("rejects non-news uuid", func(t *testing.T) {
		_, err := repo.Get(types.Filter{PostUUID: placeID})
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected non-news uuid to be rejected, got %v", err)
		}
	})

	t.Run("rejects non-news public id", func(t *testing.T) {
		_, err := repo.Get(types.Filter{PostID: 2002})
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected non-news public id to be rejected, got %v", err)
		}
	})
}

type newsPostRepositoryStub struct {
	byUUID     map[uuid.UUID]*post.Post
	byPublicID map[int64]*post.Post
}

func (s newsPostRepositoryStub) CreatePost(*post.Post) error {
	return nil
}

func (s newsPostRepositoryStub) GetPostsByKind(types.Filter) (types.PostsResult, error) {
	return types.PostsResult{}, nil
}

func (s newsPostRepositoryStub) GetPostByID(id uuid.UUID) (*post.Post, error) {
	postData, ok := s.byUUID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return postData, nil
}

func (s newsPostRepositoryStub) GetPostByPublicID(id int64) (*post.Post, error) {
	postData, ok := s.byPublicID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return postData, nil
}

func (s newsPostRepositoryStub) ExistsBySlug(types.Filter) (bool, error) {
	return false, nil
}

func (s newsPostRepositoryStub) GetPillarsWithClustersWithSlug(context.Context, string) ([]taxonomy.Pillar, error) {
	return nil, nil
}
