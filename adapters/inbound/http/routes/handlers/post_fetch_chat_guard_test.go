package handlers

import (
	"core/application/ports"
	"core/application/usecases"
	"core/models/post"
	"testing"
)

type chatGuardPostRepository struct {
	ports.PostRepository
	item *post.Post
}

func (r *chatGuardPostRepository) GetPostByPublicID(id int64) (*post.Post, error) {
	return r.item, nil
}

func TestHandleFetchPostRejectsChatMessagesFromPublicEndpoint(t *testing.T) {
	contentableType := string(post.PostKindChat)
	repo := &chatGuardPostRepository{item: &post.Post{
		PublicID: 8181, PostKind: post.PostKindMessage, ContentableType: &contentableType,
	}}
	service := usecases.NewPostService(nil, repo, nil)
	resp := performMultipartHandlerRequest(t, HandleFetchPost(service), nil, map[string]string{"post_id": "8181"}, nil)
	if resp.StatusCode != 404 {
		t.Fatalf("expected public chat-message fetch to return 404, got %d", resp.StatusCode)
	}
}

var _ ports.PostRepository = (*chatGuardPostRepository)(nil)
