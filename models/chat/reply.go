package chat

import (
	"errors"
	"github.com/google/uuid"
	"strings"
)

var ErrInvalidReply = errors.New("reply must reference an available message in this chat")

func ParseReplyID(values []string) (*uuid.UUID, error) {
	if len(values) == 0 || len(values) == 1 && strings.TrimSpace(values[0]) == "" {
		return nil, nil
	}
	if len(values) != 1 {
		return nil, ErrInvalidReply
	}
	id, err := uuid.Parse(values[0])
	if err != nil || id == uuid.Nil {
		return nil, ErrInvalidReply
	}
	return &id, nil
}
