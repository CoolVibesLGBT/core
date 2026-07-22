package repositories

import (
	"core/application/types"
	"core/models/post"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEnsureNewsPost(t *testing.T) {
	newsID := uuid.New()
	placeID := uuid.New()
	loadErr := errors.New("boom")

	t.Run("returns load error", func(t *testing.T) {
		_, err := ensureNewsPost(types.Filter{PostUUID: newsID}, nil, loadErr)
		if !errors.Is(err, loadErr) {
			t.Fatalf("expected load error, got %v", err)
		}
	})

	t.Run("returns news by uuid", func(t *testing.T) {
		postData, err := ensureNewsPost(
			types.Filter{PostUUID: newsID},
			&post.Post{ID: newsID, PostKind: post.PostKindNews},
			nil,
		)
		if err != nil {
			t.Fatalf("ensureNewsPost() error = %v", err)
		}
		if postData == nil || postData.PostKind != post.PostKindNews {
			t.Fatalf("expected news post, got %#v", postData)
		}
	})

	t.Run("returns news by public id", func(t *testing.T) {
		postData, err := ensureNewsPost(
			types.Filter{PostID: 1001},
			&post.Post{ID: newsID, PublicID: 1001, PostKind: post.PostKindNews},
			nil,
		)
		if err != nil {
			t.Fatalf("ensureNewsPost() error = %v", err)
		}
		if postData == nil || postData.PostKind != post.PostKindNews {
			t.Fatalf("expected news post, got %#v", postData)
		}
	})

	t.Run("rejects nil post", func(t *testing.T) {
		_, err := ensureNewsPost(types.Filter{PostUUID: placeID}, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected nil post to be rejected, got %v", err)
		}
	})

	t.Run("rejects non-news uuid", func(t *testing.T) {
		_, err := ensureNewsPost(
			types.Filter{PostUUID: placeID},
			&post.Post{ID: placeID, PostKind: post.PostKindPlace},
			nil,
		)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected non-news uuid to be rejected, got %v", err)
		}
	})

	t.Run("rejects non-news public id", func(t *testing.T) {
		_, err := ensureNewsPost(
			types.Filter{PostID: 2002},
			&post.Post{ID: placeID, PublicID: 2002, PostKind: post.PostKindPlace},
			nil,
		)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected non-news public id to be rejected, got %v", err)
		}
	})
}
