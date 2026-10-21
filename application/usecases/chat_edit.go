package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	"core/models"
	"core/models/chat"
	"encoding/json"
	"github.com/google/uuid"
)

var _ ports.ChatMessageEditor = (*ChatService)(nil)

func (s *ChatService) EditMessage(ctx context.Context, actor *models.User, chatID, messageID uuid.UUID, text string) (*types.ChatMessage, error) {
	editor, ok := s.chatRepo.(ports.ChatMessageEditorRepository)
	if !ok || actor == nil {
		return nil, chat.ErrPermissionDenied
	}
	message, err := editor.EditMessage(ctx, actor, chatID, messageID, text)
	if err != nil {
		return nil, err
	}
	view := chatMessageProjection(message, map[uuid.UUID]*models.User{actor.ID: actor}, 1)
	if s.socketService != nil {
		payload, _ := json.Marshal(types.ChatMessageEvent{Action: constants.CMD_EDIT_MESSAGE, Message: view})
		_ = s.socketService.BroadcastToRoom("/", chatID.String(), "chat", string(payload))
	}
	return view, nil
}
