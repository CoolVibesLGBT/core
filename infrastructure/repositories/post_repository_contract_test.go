package repositories

import (
	"core/application/types"
	"core/models"
	"core/models/media"
	"core/models/notifications"
	"core/models/post"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPostKindForContentableTypeCoversKnownTypes(t *testing.T) {
	tests := []struct {
		name            string
		contentableType string
		wantKind        post.PostKind
		wantPublished   bool
	}{
		{name: "chat", contentableType: "chat", wantKind: post.PostKindMessage, wantPublished: true},
		{name: "post", contentableType: "post", wantKind: post.PostKindPost, wantPublished: true},
		{name: "event", contentableType: "event", wantKind: post.PostKindEvent, wantPublished: true},
		{name: "status", contentableType: "status", wantKind: post.PostKindStatus, wantPublished: true},
		{name: "classified", contentableType: "classified", wantKind: post.PostKindClassified, wantPublished: true},
		{name: "job_offer", contentableType: "job_offer", wantKind: post.PostKindJobOffer, wantPublished: true},
		{name: "job_search", contentableType: "job_search", wantKind: post.PostKindJobSearch, wantPublished: true},
		{name: "news", contentableType: "news", wantKind: post.PostKindNews, wantPublished: false},
		{name: "place", contentableType: "place", wantKind: post.PostKindPlace, wantPublished: true},
		{name: "checkin", contentableType: "checkin", wantKind: post.PostKindCheckIn, wantPublished: true},
		{name: "video", contentableType: "video", wantKind: post.PostKindVideo, wantPublished: true},
		{name: "story", contentableType: "story", wantKind: post.PostKindStory, wantPublished: true},
		{name: "unknown", contentableType: "unknown", wantKind: post.PostKindStatus, wantPublished: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotPublished := postKindForContentableType(tt.contentableType)
			if gotKind != tt.wantKind {
				t.Fatalf("expected kind %q, got %q", tt.wantKind, gotKind)
			}
			if gotPublished != tt.wantPublished {
				t.Fatalf("expected published %v, got %v", tt.wantPublished, gotPublished)
			}
		})
	}
}

func TestEventFieldsResolveGenericPostsToEventKind(t *testing.T) {
	kind, published := resolvePostKindForContentableType("status", true)
	if kind != post.PostKindEvent {
		t.Fatalf("expected event kind for a post with event fields, got %q", kind)
	}
	if !published {
		t.Fatal("expected event post to be published")
	}

	kind, published = resolvePostKindForContentableType("status", false)
	if kind != post.PostKindStatus || !published {
		t.Fatalf("expected regular status behavior without event fields, got kind=%q published=%v", kind, published)
	}
}

func TestMediaOwnerForContentableTypeCoversKnownTypes(t *testing.T) {
	tests := []struct {
		name            string
		contentableType string
		wantOwner       media.OwnerType
		wantRole        media.MediaRole
	}{
		{name: "chat", contentableType: "chat", wantOwner: media.OwnerPost, wantRole: media.RoleChatMedia},
		{name: "news", contentableType: "news", wantOwner: media.OwnerNews, wantRole: media.RolePost},
		{name: "video", contentableType: "video", wantOwner: media.OwnerVideo, wantRole: media.RoleVideo},
		{name: "story", contentableType: "story", wantOwner: media.OwnerPost, wantRole: media.RoleStory},
		{name: "post", contentableType: "post", wantOwner: media.OwnerPost, wantRole: media.RolePost},
		{name: "unknown", contentableType: "unknown", wantOwner: media.OwnerPost, wantRole: media.RolePost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRole := mediaOwnerForContentableType(tt.contentableType)
			if gotOwner != tt.wantOwner {
				t.Fatalf("expected owner %q, got %q", tt.wantOwner, gotOwner)
			}
			if gotRole != tt.wantRole {
				t.Fatalf("expected role %q, got %q", tt.wantRole, gotRole)
			}
		})
	}
}

func TestPostsByKindQueryUsesPostKindAndStoryWindow(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &PostRepository{db: db}

	var posts []post.Post
	tx := repo.postsByKindQuery(types.Filter{PostKind: post.PostKindStory, Limit: 10}).Find(&posts)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	if !strings.Contains(sql, "post_kind") {
		t.Fatalf("expected query to filter by post_kind, got %s", sql)
	}
	if strings.Contains(sql, "contentable_type") {
		t.Fatalf("expected query not to filter stories by contentable_type, got %s", sql)
	}
	if !strings.Contains(sql, "created_at >") {
		t.Fatalf("expected story query to include 24-hour created_at window, got %s", sql)
	}
	if !strings.Contains(strings.ToLower(sql), "published") {
		t.Fatalf("expected public post query to filter published posts, got %s", sql)
	}
}

func TestCommentNotificationPayloadIncludesPostAndCommentIDs(t *testing.T) {
	parent := &post.Post{ID: uuid.New(), PublicID: 100}
	comment := &post.Post{ID: uuid.New(), PublicID: 200}
	author := &models.User{DisplayName: "Jane"}

	payload := commentNotificationPayload(author, parent, comment)
	if payload.Title != "New Comment" {
		t.Fatalf("expected title New Comment, got %q", payload.Title)
	}
	if payload.Body != "Jane commented on your post" {
		t.Fatalf("expected author name in body, got %q", payload.Body)
	}

	data, ok := payload.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload data, got %#v", payload.Data)
	}
	if data["type"] != notifications.NotificationTypeComment {
		t.Fatalf("expected comment notification type, got %#v", data["type"])
	}
	if data["post_id"] != int64(100) || data["comment_id"] != int64(200) {
		t.Fatalf("expected public ids in payload, got %#v", data)
	}
}
