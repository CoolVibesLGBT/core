package routes

import (
	"testing"

	"github.com/google/uuid"
)

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
