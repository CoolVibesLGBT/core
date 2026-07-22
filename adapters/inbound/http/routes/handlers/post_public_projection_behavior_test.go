package handlers

import (
	legacyviews "core/application/legacyviews"
	"core/application/ports"
	"core/application/types"
	"core/application/usecases"
	"core/models"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type publicTimelineProjectionRepo struct {
	ports.PostRepository
	item post.Post
}

func (r *publicTimelineProjectionRepo) GetTimeline(_ types.Filter) (legacyviews.TimelineResult, error) {
	return legacyviews.TimelineResult{Posts: []post.Post{r.item}}, nil
}

func TestTimelineHandlerNeverSerializesRawPostEntities(t *testing.T) {
	authorID := uuid.New()
	postID := uuid.New()
	storagePath := "/private/uploads/secret.png"
	repo := &publicTimelineProjectionRepo{item: post.Post{
		ID: postID, PublicID: 501, AuthorID: authorID, PostKind: post.PostKindPost, Published: true,
		Author: models.User{ID: authorID, PublicID: 601, UserName: "author", Balance: decimal.NewFromInt(44), PreferencesFlags: "secret-bits"},
		Attachments: []*media.Media{{
			ID: uuid.New(), PublicID: 701, OwnerID: postID, UserID: authorID, IsPublic: true,
			File: modelutils.FileMetadata{ID: uuid.New(), URL: "https://cdn.test/safe.png", StoragePath: storagePath, Name: "secret.png"},
		}},
	}}
	service := usecases.NewPostService(&handlerUserRepo{}, repo, &handlerMediaRepo{})
	response := performMultipartHandlerRequest(t, HandleTimeline(service), nil, nil, nil)
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	})
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("status = %d, body=%s", response.StatusCode, body)
	}
	if !json.Valid(body) {
		t.Fatalf("invalid JSON: %s", body)
	}
	payload := string(body)
	for _, forbidden := range []string{authorID.String(), postID.String(), storagePath, "secret.png", "secret-bits", `"balance"`, `"storage_path"`} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("timeline response leaked %q: %s", forbidden, payload)
		}
	}
}

var _ ports.PostRepository = (*publicTimelineProjectionRepo)(nil)
