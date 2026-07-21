package routes

import (
	"testing"

	"github.com/google/uuid"
)

func TestMediaStoragePathRecognizesGeneratedVariants(t *testing.T) {
	id := uuid.NewString()
	requested, prefix, ok := mediaStoragePath("/static/uploads/users/u/posts/2026-07-21/p/" + id + "_landscape_sm.webp")
	if !ok {
		t.Fatal("generated variant path was rejected")
	}
	if requested != "static/uploads/users/u/posts/2026-07-21/p/"+id+"_landscape_sm.webp" {
		t.Fatalf("requested path = %q", requested)
	}
	if prefix != "./static/uploads/users/u/posts/2026-07-21/p/"+id {
		t.Fatalf("original prefix = %q", prefix)
	}
	if _, _, ok := mediaStoragePath("/static/uploads/../../.env"); ok {
		t.Fatal("path traversal was accepted")
	}
}
