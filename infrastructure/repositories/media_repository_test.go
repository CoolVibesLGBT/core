package repositories

import (
	"core/models/media"
	"testing"
)

func TestShouldProcessAsync(t *testing.T) {
	if !shouldProcessAsync("image/jpeg") {
		t.Fatalf("image/jpeg should be processed asynchronously")
	}
	if !shouldProcessAsync("video/mp4") {
		t.Fatalf("video/mp4 should be processed asynchronously")
	}
	if shouldProcessAsync("application/pdf") {
		t.Fatalf("application/pdf should not be processed asynchronously")
	}
}

func TestInitialFileVariantsForImage(t *testing.T) {
	variants := initialFileVariants("image/jpeg", "./static/uploads/a.jpg", ".jpg", 1234)
	if variants == nil || variants.Image == nil || variants.Image.Original == nil {
		t.Fatalf("expected initial image variants, got %#v", variants)
	}

	if variants.Image.Original.URL != "/static/uploads/a.jpg" {
		t.Fatalf("unexpected original url: %q", variants.Image.Original.URL)
	}
	if variants.Image.Original.Format != "jpg" {
		t.Fatalf("unexpected original format: %q", variants.Image.Original.Format)
	}
	if variants.Image.Original.Size != 1234 {
		t.Fatalf("unexpected original size: %d", variants.Image.Original.Size)
	}
}

func TestInitialFileVariantsForNonImage(t *testing.T) {
	if variants := initialFileVariants("video/mp4", "./static/uploads/a.mp4", ".mp4", 1234); variants != nil {
		t.Fatalf("expected nil variants for non-image upload, got %#v", variants)
	}
}

func TestChatMediaRolesArePrivate(t *testing.T) {
	for _, role := range []media.MediaRole{media.RoleChatImage, media.RoleChatMedia, media.RoleChatVideo} {
		if isPublicMediaRole(role) {
			t.Fatalf("chat role %q was marked public", role)
		}
	}
	if !isPublicMediaRole(media.RolePost) {
		t.Fatal("regular post media should be public")
	}
}
