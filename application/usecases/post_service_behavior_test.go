package usecases

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/post"
	"core/types"
	"testing"

	"github.com/google/uuid"
)

func TestPostServiceCreatePostCreatesAndHydratesPost(t *testing.T) {
	postID := uuid.New()
	postRepo := &fakePostRepository{
		createdPostID: postID,
		getPostByID:   &post.Post{ID: postID, PostKind: post.PostKindStatus, PublicID: 99},
	}
	service := NewPostService(&fakeUserRepository{}, postRepo, &fakeMediaRepository{})
	author := &models.User{ID: uuid.New(), PublicID: 7}

	created, err := service.CreatePost(context.Background(), ports.FormData{
		Values: map[string][]string{"content": {"hello"}},
	}, author, post.PostKindStatus)
	if err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}

	if created.ID != postID || created.PublicID != 99 {
		t.Fatalf("expected hydrated post, got %#v", created)
	}
	if postRepo.createContentableType != string(post.PostKindStatus) {
		t.Fatalf("expected status contentable type, got %q", postRepo.createContentableType)
	}
	if postRepo.createAuthor != author {
		t.Fatalf("expected author to pass through")
	}
}

func TestPostServiceUserScopedListsResolvePublicUserID(t *testing.T) {
	userUUID := uuid.New()
	userRepo := &fakeUserRepository{userUUIDByPublicID: map[int64]uuid.UUID{42: userUUID}}
	postRepo := &fakePostRepository{
		userPosts:   []post.Post{{ID: uuid.New(), PostKind: post.PostKindPost}},
		userReplies: []post.Post{{ID: uuid.New(), PostKind: post.PostKindPost}},
	}
	service := NewPostService(userRepo, postRepo, &fakeMediaRepository{})

	posts, err := service.GetPostsByUserID(types.Filter{UserID: 42, PostKind: post.PostKindPost, Limit: 10})
	if err != nil {
		t.Fatalf("GetPostsByUserID() error = %v", err)
	}
	if len(posts) != 1 || postRepo.userPostsUserID != userUUID {
		t.Fatalf("expected user posts for uuid %s, got posts=%#v user=%s", userUUID, posts, postRepo.userPostsUserID)
	}

	replies, err := service.GetUserPostReplies(types.Filter{UserID: 42, PostKind: post.PostKindPost, Limit: 10})
	if err != nil {
		t.Fatalf("GetUserPostReplies() error = %v", err)
	}
	if len(replies) != 1 || postRepo.userRepliesFilter.UserUUID != userUUID {
		t.Fatalf("expected replies filter user uuid %s, got %#v", userUUID, postRepo.userRepliesFilter)
	}
}

func TestPostServicePassesEngagementFiltersToRepository(t *testing.T) {
	postRepo := &fakePostRepository{}
	service := NewPostService(&fakeUserRepository{}, postRepo, &fakeMediaRepository{})
	filter := types.Filter{PostID: 123, AuthUser: &models.User{ID: uuid.New()}}

	if err := service.Like(filter); err != nil {
		t.Fatalf("Like() error = %v", err)
	}
	if postRepo.likeFilter.PostID != 123 {
		t.Fatalf("expected like filter post id 123, got %#v", postRepo.likeFilter)
	}

	if err := service.Delete(filter); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if postRepo.deleteFilter.PostID != 123 {
		t.Fatalf("expected delete filter post id 123, got %#v", postRepo.deleteFilter)
	}
}
