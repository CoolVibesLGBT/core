package handlers

import (
	"bytes"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/models"
	"core/models/media"
	"core/models/post"
	"core/types"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type searchHandlerUserRepository struct {
	ports.UserRepository
	users []models.User
}

func (r *searchHandlerUserRepository) GetUsersStartingWith(_ string, _ int) ([]models.User, error) {
	return r.users, nil
}

type searchHandlerPostRepository struct {
	ports.PostRepository
	filters []types.Filter
}

func (r *searchHandlerPostRepository) FindPostsByKind(filters types.Filter) (types.PostsResult, error) {
	r.filters = append(r.filters, filters)
	result := post.Post{ID: uuid.New(), PublicID: 100, PostKind: filters.PostKind}
	return types.PostsResult{Posts: []post.Post{result}}, nil
}

type searchHandlerMediaRepository struct {
	ports.MediaRepository
}

func (r *searchHandlerMediaRepository) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userID uuid.UUID, role media.MediaRole, file ports.UploadedFile) (*media.Media, error) {
	return nil, errors.New("not used")
}

func TestHandleGlobalSearchReturnsGroupedResults(t *testing.T) {
	userRepo := &searchHandlerUserRepository{users: []models.User{{PublicID: 7, UserName: "pride_user"}}}
	postRepo := &searchHandlerPostRepository{}
	userService := usecases.NewUserService(userRepo, postRepo, &searchHandlerMediaRepository{}, nil, nil)
	postService := usecases.NewPostService(userRepo, postRepo, &searchHandlerMediaRepository{})

	app := fiber.New()
	app.Post("/", HandleGlobalSearch(userService, postService))

	req := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewBufferString("query=pride&scope=all&limit=5&domain=coolvibes.lgbt"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Success || !bytes.Contains(payload.Data, []byte(`"query":"pride"`)) {
		t.Fatalf("unexpected grouped response: %s", payload.Data)
	}
	if len(postRepo.filters) != 3 {
		t.Fatalf("expected posts, events, and places searches, got %d", len(postRepo.filters))
	}
	if postRepo.filters[0].PostKind != "" {
		t.Fatalf("expected unscoped social search first, got %q", postRepo.filters[0].PostKind)
	}
	if postRepo.filters[1].PostKind != post.PostKindEvent {
		t.Fatalf("expected event search second, got %q", postRepo.filters[1].PostKind)
	}
	if postRepo.filters[2].PostKind != post.PostKindPlace {
		t.Fatalf("expected place search third, got %q", postRepo.filters[2].PostKind)
	}
	if postRepo.filters[0].Search == nil || *postRepo.filters[0].Search != "pride" {
		t.Fatalf("expected normalized search text, got %#v", postRepo.filters[0].Search)
	}
}

func TestHandleGlobalSearchRejectsShortQueries(t *testing.T) {
	userRepo := &searchHandlerUserRepository{}
	postRepo := &searchHandlerPostRepository{}
	userService := usecases.NewUserService(userRepo, postRepo, &searchHandlerMediaRepository{}, nil, nil)
	postService := usecases.NewPostService(userRepo, postRepo, &searchHandlerMediaRepository{})

	app := fiber.New()
	app.Post("/", HandleGlobalSearch(userService, postService))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("query=p"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	if len(postRepo.filters) != 0 {
		t.Fatalf("expected no repository calls for a short query, got %d", len(postRepo.filters))
	}
}
