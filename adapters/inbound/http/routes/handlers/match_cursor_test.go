package handlers

import (
	"core/application/types"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseMatchListCursorPreservesExactTieBreaker(t *testing.T) {
	occurredAt := time.Date(2026, time.July, 22, 3, 4, 5, 678000000, time.UTC)
	detailID := uuid.New()
	encoded, err := types.NewTimeUUIDCursor(occurredAt, detailID)
	if err != nil {
		t.Fatal(err)
	}

	cursor, err := parseMatchListCursor(*encoded)
	if err != nil || cursor == nil || !cursor.OccurredAt.Equal(occurredAt) || cursor.DetailID != detailID {
		t.Fatalf("parseMatchListCursor() = %#v, %v", cursor, err)
	}
}

func TestParseMatchListCursorAcceptsLegacyTimestampButRejectsBadOpaqueID(t *testing.T) {
	legacy := "2026-07-22T03:04:05.123456789Z"
	cursor, err := parseMatchListCursor(legacy)
	if err != nil || cursor == nil || cursor.DetailID != uuid.Nil {
		t.Fatalf("legacy cursor = %#v, %v", cursor, err)
	}

	bad, err := types.NewPaginationCursor(map[string]any{
		types.CursorKeyCreatedAt: legacy,
		types.CursorKeyUUID:      "not-a-uuid",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseMatchListCursor(*bad); err == nil {
		t.Fatal("invalid opaque cursor ID was accepted")
	}
}
