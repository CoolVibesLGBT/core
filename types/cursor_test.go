package types

import (
	"testing"
	"time"
)

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
