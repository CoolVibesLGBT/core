package usecases

import (
	"context"
	"core/application/types"
	"core/models"
	"core/models/post"
	"errors"
	"github.com/google/uuid"
	"testing"
)

type postMutationFailure struct{ *fakePostRepository }

func (r *postMutationFailure) Like(types.Filter) error { return errors.New("write failed") }

func TestPostMutationInvalidatesWithoutBroadcastingPrivateContent(t *testing.T) {
	realtime := &fakeRealtimeNotifier{}
	repo := &fakePostRepository{createdPostID: uuid.New(), getPostByID: &types.PublicPost{ID: 99, PublicID: 99}}
	service := NewPostService(&fakeUserRepository{}, repo, &fakeMediaRepository{}, realtime)
	_, err := service.CreatePost(context.Background(), map[string][]string{"content": {"private draft"}}, nil, &models.User{ID: uuid.New(), PublicID: 7}, post.PostKindStatus)
	if err != nil {
		t.Fatal(err)
	}
	if realtime.room != "broadcast" || realtime.event != "post" || realtime.msg != `{"action":"post.changed"}` {
		t.Fatalf("unexpected invalidation: %#v", realtime)
	}
}

func TestPostMutationFailureDoesNotInvalidate(t *testing.T) {
	realtime := &fakeRealtimeNotifier{}
	service := NewPostService(&fakeUserRepository{}, &postMutationFailure{&fakePostRepository{}}, &fakeMediaRepository{}, realtime)
	if err := service.Like(types.Filter{}); err == nil {
		t.Fatal("write failure was lost")
	}
	if realtime.event != "" {
		t.Fatal("failed write was broadcast")
	}
}

func TestPostSocketFailureCannotTurnCommittedMutationIntoFailure(t *testing.T) {
	realtime := &fakeRealtimeNotifier{err: errors.New("offline")}
	service := NewPostService(&fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, realtime)
	if err := service.Delete(types.Filter{}); err != nil {
		t.Fatal(err)
	}
	if realtime.event != "post" {
		t.Fatal("missing invalidation attempt")
	}
}
