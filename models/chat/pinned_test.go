package chat

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPinnedMessageListKeepsOrderAndMirrorsLatest(t *testing.T) {
	first, second, user := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()
	list := PinnedMessageList{}.With(first, user, now).With(second, user, now.Add(time.Second))
	if got := list.IDs(); len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("pins must stay oldest first, got %v", got)
	}
	// Re-pinning moves a message to the newest position without duplicating it.
	list = list.With(first, user, now.Add(2*time.Second))
	if got := list.IDs(); len(got) != 2 || got[1] != first {
		t.Fatalf("re-pin must move to the end, got %v", got)
	}
	if latest := list.Latest(); latest == nil || latest.MessageID != first {
		t.Fatalf("latest must be the newest pin")
	}
	list = list.Without(first)
	if list.Contains(first) || list.Latest().MessageID != second {
		t.Fatalf("unpin must remove only that message")
	}
	if PinnedMessageList(nil).Without(second).Latest() != nil {
		t.Fatalf("empty list has no latest pin")
	}
}

func TestPinnedMessageListRoundTripsJSONB(t *testing.T) {
	list := PinnedMessageList{}.With(uuid.New(), uuid.New(), time.Now())
	value, err := list.Value()
	if err != nil {
		t.Fatal(err)
	}
	var decoded PinnedMessageList
	if err := decoded.Scan([]byte(value.(string))); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded[0].MessageID != list[0].MessageID {
		t.Fatalf("round trip lost data: %v", decoded)
	}
	var empty PinnedMessageList
	if err := empty.Scan(nil); err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("NULL must scan to an empty list")
	}
}
