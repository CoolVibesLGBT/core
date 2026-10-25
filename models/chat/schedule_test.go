package chat

import (
	"testing"
	"time"
)

func TestParseScheduledAt(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	if at, err := ParseScheduledAt(nil, now); err != nil || at != nil {
		t.Fatalf("missing value sends now")
	}
	at, err := ParseScheduledAt([]string{"2026-10-25T13:00:00Z"}, now)
	if err != nil || !at.Equal(now.Add(time.Hour)) {
		t.Fatalf("RFC 3339 value: %v %v", at, err)
	}
	at, err = ParseScheduledAt([]string{"1793019600"}, now) // Unix seconds
	if err != nil || at == nil {
		t.Fatalf("unix value: %v", err)
	}
	for _, bad := range []string{"2026-10-25T12:00:05Z", "2028-01-01T00:00:00Z", "soon"} {
		if _, err := ParseScheduledAt([]string{bad}, now); err == nil {
			t.Fatalf("%q must be rejected", bad)
		}
	}
}
