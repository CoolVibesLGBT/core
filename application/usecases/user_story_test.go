package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	domainmedia "core/domain/media"
	"core/models"
	"core/models/post"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

type storyPostRepository struct {
	ports.PostRepository

	createValues          map[string][]string
	createFiles           []*domainmedia.UploadedFile
	createAuthor          *models.User
	createContentableType string
	createContentableID   *uuid.UUID
	getPostsFilter        types.Filter
}

func (r *storyPostRepository) CreateContentablePost(ctx context.Context, values map[string][]string, files []*domainmedia.UploadedFile, author *models.User, contentableType string, contentableID *uuid.UUID) (*post.Post, error) {
	r.createValues, r.createFiles = values, files
	r.createAuthor = author
	r.createContentableType = contentableType
	r.createContentableID = contentableID

	return &post.Post{
		ID:       uuid.New(),
		PublicID: 1001,
		PostKind: post.PostKindStory,
		AuthorID: author.ID,
	}, nil
}

func (r *storyPostRepository) FindPublicPostByID(id uuid.UUID) (*types.PublicPost, error) {
	return &types.PublicPost{
		ID:       1001,
		PublicID: 1001,
		PostKind: string(post.PostKindStory),
	}, nil
}

func (r *storyPostRepository) FindPublicPostByIDIncludingUnpublished(id uuid.UUID) (*types.PublicPost, error) {
	return r.FindPublicPostByID(id)
}

func (r *storyPostRepository) FetchPublicPostsByKind(filters types.Filter) (types.PublicPostPage, error) {
	r.getPostsFilter = filters
	cursor := "1001"
	return types.PublicPostPage{
		Posts:  []types.PublicPost{{ID: 1001, PublicID: 1001, PostKind: string(post.PostKindStory)}},
		Cursor: &cursor,
	}, nil
}

func TestUserServiceAddStoryCreatesStoryPost(t *testing.T) {
	repo := &storyPostRepository{}
	service := &UserService{postRepo: repo}
	user := &models.User{ID: uuid.New(), PublicID: 42, DefaultLanguage: "en", Domain: models.CoolVibes}

	created, err := service.AddStory(context.Background(), map[string][]string{
		"caption": {"story caption"},
	}, nil, user)
	if err != nil {
		t.Fatalf("AddStory() error = %v", err)
	}

	if created == nil || created.PostKind != string(post.PostKindStory) {
		t.Fatalf("expected hydrated story post, got %#v", created)
	}
	if repo.createAuthor != user {
		t.Fatalf("expected story author to be passed through")
	}
	if repo.createContentableType != string(post.PostKindStory) {
		t.Fatalf("expected contentable type story, got %q", repo.createContentableType)
	}
	if repo.createContentableID != nil {
		t.Fatalf("expected story contentable id to be nil, got %s", repo.createContentableID)
	}
	if got := firstFormValue(repo.createValues, "content"); got != "story caption" {
		t.Fatalf("expected caption to be copied into content, got %q", got)
	}
	if got := firstFormValue(repo.createValues, "audience"); got != "public" {
		t.Fatalf("expected default audience public, got %q", got)
	}
	if got := firstFormValue(repo.createValues, "slug"); got == "" {
		t.Fatalf("expected generated story slug")
	}

	var extras struct {
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal([]byte(firstFormValue(repo.createValues, "extras")), &extras); err != nil {
		t.Fatalf("expected valid story extras json: %v", err)
	}
	expiresAt, err := time.Parse(time.RFC3339, extras.ExpiresAt)
	if err != nil {
		t.Fatalf("expected RFC3339 expires_at, got %q", extras.ExpiresAt)
	}
	if time.Until(expiresAt) < 23*time.Hour || time.Until(expiresAt) > 25*time.Hour {
		t.Fatalf("expected story expires_at about 24h in future, got %s", expiresAt)
	}
}

func TestUserServiceAddStoryPreservesExplicitPostFields(t *testing.T) {
	repo := &storyPostRepository{}
	service := &UserService{postRepo: repo}
	user := &models.User{ID: uuid.New(), PublicID: 42, DefaultLanguage: "en", Domain: models.CoolVibes}

	_, err := service.AddStory(context.Background(), map[string][]string{
		"caption":  {"story caption"},
		"content":  {"explicit content"},
		"audience": {"followers"},
		"slug":     {"explicit-story"},
		"extras":   {`{"expires_at":"custom"}`},
	}, nil, user)
	if err != nil {
		t.Fatalf("AddStory() error = %v", err)
	}

	if got := firstFormValue(repo.createValues, "content"); got != "explicit content" {
		t.Fatalf("expected explicit content to win, got %q", got)
	}
	if got := firstFormValue(repo.createValues, "audience"); got != "followers" {
		t.Fatalf("expected explicit audience to win, got %q", got)
	}
	if got := firstFormValue(repo.createValues, "slug"); got != "explicit-story" {
		t.Fatalf("expected explicit slug to win, got %q", got)
	}
	if got := firstFormValue(repo.createValues, "extras"); got != `{"expires_at":"custom"}` {
		t.Fatalf("expected explicit extras to win, got %q", got)
	}
}

func TestUserServiceGetAllStoriesForcesPostKindStory(t *testing.T) {
	repo := &storyPostRepository{}
	service := &UserService{postRepo: repo}

	result, err := service.GetAllStories(types.Filter{PostKind: post.PostKindPost, Limit: 5})
	if err != nil {
		t.Fatalf("GetAllStories() error = %v", err)
	}

	if repo.getPostsFilter.PostKind != post.PostKindStory {
		t.Fatalf("expected PostKindStory filter, got %q", repo.getPostsFilter.PostKind)
	}
	if len(result.Posts) != 1 || result.Posts[0].PostKind != string(post.PostKindStory) {
		t.Fatalf("expected story posts result, got %#v", result)
	}
	if result.Cursor == nil || *result.Cursor != "1001" {
		t.Fatalf("expected cursor to pass through, got %#v", result.Cursor)
	}
}
