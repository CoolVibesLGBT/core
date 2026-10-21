package usecases

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/chat"
	"github.com/google/uuid"
)

func (s *ChatService) SetChatMuted(ctx context.Context, actor *models.User, chatID uuid.UUID, muted bool) error {
	repository, ok := s.chatRepo.(ports.ChatMutePreference)
	if !ok {
		return chat.ErrPermissionDenied
	}
	return repository.SetChatMuted(ctx, actor, chatID, muted)
}
