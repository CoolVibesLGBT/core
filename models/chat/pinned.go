package chat

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// MaxPinnedMessages bounds a chat's pinned list (Telegram keeps a short
// history that the pinned bar steps through).
const MaxPinnedMessages = 100

// PinnedMessage is one entry of a chat's pinned list.
type PinnedMessage struct {
	MessageID  uuid.UUID `json:"message_id"`
	PinnedByID uuid.UUID `json:"pinned_by_id"`
	PinnedAt   time.Time `json:"pinned_at"`
}

// PinnedMessageList is stored as a JSONB array, oldest pin first.
type PinnedMessageList []PinnedMessage

func (list PinnedMessageList) Value() (driver.Value, error) {
	if list == nil {
		return "[]", nil
	}
	encoded, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

func (list *PinnedMessageList) Scan(value any) error {
	var raw []byte
	switch typed := value.(type) {
	case nil:
		*list = PinnedMessageList{}
		return nil
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return errors.New("pinned messages: unsupported column type")
	}
	if len(raw) == 0 {
		*list = PinnedMessageList{}
		return nil
	}
	var decoded PinnedMessageList
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*list = decoded
	return nil
}

// Contains reports whether the message is pinned.
func (list PinnedMessageList) Contains(messageID uuid.UUID) bool {
	for _, entry := range list {
		if entry.MessageID == messageID {
			return true
		}
	}
	return false
}

// With pins the message as the newest entry; re-pinning moves it to the end.
func (list PinnedMessageList) With(messageID, pinnedBy uuid.UUID, at time.Time) PinnedMessageList {
	result := make(PinnedMessageList, 0, len(list)+1)
	for _, entry := range list {
		if entry.MessageID != messageID {
			result = append(result, entry)
		}
	}
	result = append(result, PinnedMessage{MessageID: messageID, PinnedByID: pinnedBy, PinnedAt: at.UTC()})
	if len(result) > MaxPinnedMessages {
		result = result[len(result)-MaxPinnedMessages:]
	}
	return result
}

// Without removes the message from the list.
func (list PinnedMessageList) Without(messageID uuid.UUID) PinnedMessageList {
	result := make(PinnedMessageList, 0, len(list))
	for _, entry := range list {
		if entry.MessageID != messageID {
			result = append(result, entry)
		}
	}
	return result
}

// Latest is the most recently pinned entry, mirrored into pinned_msg_id for
// clients that only know a single pinned message.
func (list PinnedMessageList) Latest() *PinnedMessage {
	if len(list) == 0 {
		return nil
	}
	latest := list[len(list)-1]
	return &latest
}

// IDs lists the pinned message identifiers, oldest first.
func (list PinnedMessageList) IDs() []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(list))
	for _, entry := range list {
		ids = append(ids, entry.MessageID)
	}
	return ids
}
