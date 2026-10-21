package handlers

import (
	"core/application/ports"
	"core/application/types"
	"core/application/usecases"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type publicTimelineProjectionRepo struct {
	ports.PostRepository
	item types.PublicPost
}

func (r *publicTimelineProjectionRepo) FetchPublicTimeline(_ types.Filter) (types.PublicPostPage, error) {
	return types.PublicPostPage{Posts: []types.PublicPost{r.item}}, nil
}

func TestTimelineHandlerNeverSerializesRawPostEntities(t *testing.T) {
	authorID := uuid.New()
	postID := uuid.New()
	storagePath := "/private/uploads/secret.png"
	repo := &publicTimelineProjectionRepo{item: types.PublicPost{
		ID: 501, PublicID: 501, AuthorID: 601, PostKind: "post", Published: true,
		Author:      types.PublicPostAuthor{ID: 601, PublicID: 601, UserName: "author"},
		Attachments: []types.PublicPostMedia{{ID: 701, PublicID: 701, IsPublic: true, File: types.PublicMediaFile{URL: "https://cdn.test/safe.png"}}},
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
