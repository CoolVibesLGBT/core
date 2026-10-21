package routes

import (
	"context"
	"core/application/ports"
	"core/application/usecases"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

type mediaFileAccessFake struct {
	access   ports.MediaFileAccess
	ownerID  uuid.UUID
	viewerID uuid.UUID
	approved bool
}

func (r *mediaFileAccessFake) FindMediaFileAccess(context.Context, string) (ports.MediaFileAccess, error) {
	return r.access, nil
}

func (r *mediaFileAccessFake) FindMediaAccessPrincipal(_ context.Context, id int64) (ports.MediaAccessPrincipal, error) {
	switch id {
	case 10:
		return ports.MediaAccessPrincipal{ID: r.ownerID, Role: "user"}, nil
	case 20:
		return ports.MediaAccessPrincipal{ID: r.viewerID, Role: "user"}, nil
	}
	return ports.MediaAccessPrincipal{}, ports.ErrNotFound
}

func (*mediaFileAccessFake) IsActiveChatParticipant(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

func (r *mediaFileAccessFake) HasApprovedPrivatePhotoAccess(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return r.approved, nil
}

type mediaTokenDecoderFake struct{}

func (mediaTokenDecoderFake) DecodeUserPublicID(token string) (int64, error) {
	switch token {
	case "owner":
		return 10, nil
	case "viewer":
		return 20, nil
	}
	return 0, errors.New("invalid token")
}

func TestMediaFileChecksGrantsAndPrivacyOnEveryRequest(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("static/uploads/private", 0o755); err != nil {
		t.Fatal(err)
	}
	stem := "/static/uploads/private/" + uuid.NewString()
	variant, raw := stem+"_square_sm.webp", stem+".jpg"
	for _, name := range []string{variant, raw} {
		if err := os.WriteFile("."+name, []byte("photo"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	repo := &mediaFileAccessFake{ownerID: uuid.New(), viewerID: uuid.New()}
	repo.access = ports.MediaFileAccess{OwnerID: repo.ownerID, OwnerType: "user", Role: "private_photo"}
	app := NewRouter(Dependencies{
		MediaAccessService: usecases.NewMediaAccessService(repo, repo),
		TokenDecoder:       mediaTokenDecoderFake{},
	}).GetFiber()
	check := func(method, name, token string, want int) *http.Response {
		t.Helper()
		request := httptest.NewRequest(method, name, nil)
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = response.Body.Close() })
		if response.StatusCode != want {
			t.Fatalf("%s %s token=%q: status=%d, want %d", method, name, token, response.StatusCode, want)
		}
		if want == http.StatusOK && response.Header.Get("Cache-Control") != "private, no-store" {
			t.Fatalf("cache policy = %q; want private, no-store", response.Header.Get("Cache-Control"))
		}
		return response
	}
	check(http.MethodGet, variant, "", http.StatusNotFound)
	check(http.MethodGet, variant, "invalid", http.StatusNotFound)
	check(http.MethodGet, variant, "viewer", http.StatusNotFound)
	repo.approved = true
	check(http.MethodGet, variant, "viewer", http.StatusOK)
	check(http.MethodHead, variant, "viewer", http.StatusOK)
	check(http.MethodGet, raw, "viewer", http.StatusNotFound)
	check(http.MethodGet, raw, "owner", http.StatusOK)
	repo.approved = false
	check(http.MethodGet, variant, "viewer", http.StatusNotFound)
	check(http.MethodHead, variant, "viewer", http.StatusNotFound)
	repo.access.IsPublic, repo.access.Role = true, "profile"
	check(http.MethodGet, variant, "", http.StatusOK)
	repo.access.IsPublic, repo.access.Role = false, "private_photo"
	check(http.MethodGet, variant, "", http.StatusNotFound)
}

func TestStaticUploadsCannotBypassAuthorization(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("static/uploads/private", 0o755); err != nil {
		t.Fatal(err)
	}
	name := uuid.NewString() + "_square_sm.webp"
	if err := os.WriteFile(filepath.Join("static/uploads/private", name), []byte("private photo"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewRouter(Dependencies{}).GetFiber()
	for _, prefix := range []string{
		"/static/uploads/private/",
		"/static/%75ploads/private/",
		"/static/uploads%2fprivate/",
		"/static//uploads/private/",
		"/static/data/../uploads/private/",
		"/static/data/%2e%2e/uploads/private/",
		"/static/%2575ploads/private/",
		"/static/data/%252e%252e/uploads/private/",
		"/static/UPLOADS/private/",
	} {
		t.Run(prefix, func(t *testing.T) {
			response, err := app.Test(httptest.NewRequest(http.MethodGet, prefix+name, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("unauthorized file status = %d, want 404", response.StatusCode)
			}
		})
	}
	if err := os.MkdirAll("static/data", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("static/data/public.txt", []byte("public asset"), 0o600); err != nil {
		t.Fatal(err)
	}
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/static/data/public.txt", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusOK || string(body) != "public asset" {
		t.Fatalf("public static asset = %d %q, %v", response.StatusCode, body, err)
	}
}

func TestMediaStoragePathRecognizesGeneratedVariants(t *testing.T) {
	id := uuid.NewString()
	stem := "1786147200_" + id
	requested, prefix, ok := mediaStoragePath("/static/uploads/users/u/posts/2026-07-21/p/" + stem + "_landscape_sm.webp")
	if !ok {
		t.Fatal("generated variant path was rejected")
	}
	if requested != "static/uploads/users/u/posts/2026-07-21/p/"+stem+"_landscape_sm.webp" {
		t.Fatalf("requested path = %q", requested)
	}
	if prefix != "./static/uploads/users/u/posts/2026-07-21/p/"+stem {
		t.Fatalf("original prefix = %q", prefix)
	}

	if _, _, ok := mediaStoragePath("/static/uploads/users/u/posts/2026-07-21/p/" + id + "_landscape_sm.webp"); !ok {
		t.Fatal("legacy UUID variant path was rejected")
	}
	if _, _, ok := mediaStoragePath("/static/uploads/../../.env"); ok {
		t.Fatal("path traversal was accepted")
	}
	for _, invalidStem := range []string{
		"not-a-media-id",
		"prefix_" + id,
		"0_" + id,
		"1786147200_" + id + "_extra",
	} {
		if _, _, ok := mediaStoragePath("/static/uploads/users/u/" + invalidStem + "_landscape_sm.webp"); ok {
			t.Fatalf("invalid media stem %q was accepted", invalidStem)
		}
	}
}
