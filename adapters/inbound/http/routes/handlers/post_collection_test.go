package handlers

import (
	"core/application/ports"
	"core/application/types"
	"core/application/usecases"
	"core/models"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type collectionTestRepo struct {
	ports.PostRepository
	called    bool
	bookmarks bool
}

func (r *collectionTestRepo) FetchPublicUserPostCollection(f types.Filter, bookmarks bool) ([]types.PublicPost, error) {
	r.called, r.bookmarks = true, bookmarks
	return []types.PublicPost{{PublicID: 42, ID: 42}}, nil
}

func TestPostCollectionsReturnPostsAndCursor(t *testing.T) {
	for _, bookmarks := range []bool{false, true} {
		repo := &collectionTestRepo{}
		service := usecases.NewPostService(nil, repo, nil)
		actor := &models.User{ID: uuid.New(), PublicID: 10}
		response := performMultipartHandlerRequest(t, HandleGetUserPostCollection(service, bookmarks), actor, map[string]string{"limit": "1"}, nil)
		defer func() {
			if err := response.Body.Close(); err != nil {
				t.Errorf("close response body: %v", err)
			}
		}()
		var body struct {
			Data struct {
				Posts  []types.PublicPost `json:"posts"`
				Cursor *string            `json:"cursor"`
			} `json:"data"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 200 || !repo.called || repo.bookmarks != bookmarks || len(body.Data.Posts) != 1 || body.Data.Cursor == nil {
			t.Fatalf("invalid collection response: status=%d body=%+v", response.StatusCode, body)
		}
	}
}

func TestBookmarksRejectAnotherUsersCollection(t *testing.T) {
	repo := &collectionTestRepo{}
	service := usecases.NewPostService(nil, repo, nil)
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	response := performMultipartHandlerRequest(t, HandleGetUserPostCollection(service, true), actor, map[string]string{"user_id": uuid.NewString()}, nil)
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	if response.StatusCode != 403 || repo.called {
		t.Fatalf("status=%d called=%v", response.StatusCode, repo.called)
	}
}

func TestCreatePostMalformedFormUsesStandardError(t *testing.T) {
	app := fiber.New()
	app.Post("/", HandleCreate(nil))
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	var body struct {
		Code    string `json:"code"`
		Success bool   `json:"success"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 400 || body.Code != "INVALID_FORM" || body.Success {
		t.Fatalf("status=%d body=%+v", response.StatusCode, body)
	}
}

func TestPostCollectionsRequireAuthentication(t *testing.T) {
	app := fiber.New()
	app.Post("/", HandleGetUserPostCollection(nil, false))
	response, err := app.Test(httptest.NewRequest(fiber.MethodPost, "/", nil))
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusUnauthorized)
	}
}
