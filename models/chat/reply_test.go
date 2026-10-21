package chat

import (
	"errors"
	"github.com/google/uuid"
	"testing"
)

func TestParseReplyID(t *testing.T) {
	id := uuid.New()
	for _, values := range [][]string{nil, {""}} {
		if parsed, err := ParseReplyID(values); parsed != nil || err != nil {
			t.Fatalf("empty reply: %v %v", parsed, err)
		}
	}
	if parsed, err := ParseReplyID([]string{id.String()}); err != nil || parsed == nil || *parsed != id {
		t.Fatalf("valid reply: %v %v", parsed, err)
	}
	for _, values := range [][]string{{"bad"}, {uuid.Nil.String()}, {id.String(), id.String()}} {
		if _, err := ParseReplyID(values); !errors.Is(err, ErrInvalidReply) {
			t.Fatalf("invalid reply accepted: %v", values)
		}
	}
}
