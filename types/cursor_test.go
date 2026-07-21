package types

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPaginationCursorKeepsLegacyFiberWireFormat(t *testing.T) {
	values := map[string]any{CursorKeyPublicID: "42"}
	cursor, err := NewPaginationCursor(values)
	if err != nil {
		t.Fatalf("NewPaginationCursor() error = %v", err)
	}
	want := base64.RawURLEncoding.EncodeToString([]byte(`{"public_id":"42"}`))
	if cursor == nil || *cursor != want {
		t.Fatalf("cursor = %#v, want %q", cursor, want)
	}
}

func TestPaginationCursorRejectsOversizedToken(t *testing.T) {
	if _, ok := DecodePaginationCursor(strings.Repeat("a", maxCursorLength+1)); ok {
		t.Fatal("oversized cursor must be rejected")
	}
}

func TestPublicIDDistanceCursorRoundTrip(t *testing.T) {
	publicID := int64(9223372036854775000)
	distance := 42.5
	cursor, err := NewPublicIDDistanceCursor(publicID, &distance)
	if err != nil {
		t.Fatalf("NewPublicIDDistanceCursor() error = %v", err)
	}
	if cursor == nil || *cursor == "9223372036854775000" {
		t.Fatalf("expected opaque fiber cursor token, got %#v", cursor)
	}

	values, ok := DecodePaginationCursor(*cursor)
	if !ok {
		t.Fatalf("expected cursor to decode")
	}
	gotPublicID, ok := CursorPublicID(values)
	if !ok || gotPublicID != 9223372036854775000 {
		t.Fatalf("expected public_id 9223372036854775000, got %d ok=%v", gotPublicID, ok)
	}
	gotDistance, ok := CursorDistance(values)
	if !ok || gotDistance != distance {
		t.Fatalf("expected distance %v, got %v ok=%v", distance, gotDistance, ok)
	}
}

func TestTimeUUIDCursorRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	id := uuid.New()
	cursor, err := NewTimeUUIDCursor(now, id)
	if err != nil {
		t.Fatalf("NewTimeUUIDCursor() error = %v", err)
	}
	values, ok := DecodePaginationCursor(*cursor)
	if !ok {
		t.Fatal("cursor could not be decoded")
	}
	createdAt, ok := CursorCreatedAt(values)
	if !ok || !createdAt.Equal(now) {
		t.Fatalf("created_at = %v, %v", createdAt, ok)
	}
	gotID, ok := CursorUUID(values)
	if !ok || gotID != id {
		t.Fatalf("id = %v, %v", gotID, ok)
	}
}

func TestTimeCursorRoundTrip(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 30, 5, 123, time.UTC)
	cursor, err := NewTimeCursor(now)
	if err != nil {
		t.Fatalf("NewTimeCursor() error = %v", err)
	}
	if cursor == nil || *cursor == now.Format(time.RFC3339Nano) {
		t.Fatalf("expected opaque fiber cursor token, got %#v", cursor)
	}

	values, ok := DecodePaginationCursor(*cursor)
	if !ok {
		t.Fatalf("expected cursor to decode")
	}
	createdAt, ok := CursorCreatedAt(values)
	if !ok || !createdAt.Equal(now) {
		t.Fatalf("expected created_at %s, got %s ok=%v", now, createdAt, ok)
	}
}
