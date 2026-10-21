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

var _ ports.ChatMessageForwarder = (*ChatService)(nil)

func (s *ChatService) ForwardMessages(ctx context.Context, actor *models.User, sourceID, targetID uuid.UUID, ids []uuid.UUID, comment string) ([]types.ChatMessage, error) {
	repository, ok := s.chatRepo.(ports.ChatMessageForwarderRepository)
	if !ok || actor == nil {
		return nil, chat.ErrPermissionDenied
	}
	messages, err := repository.ForwardMessages(ctx, actor, sourceID, targetID, ids, comment)
	if err != nil {
		return nil, err
	}
	views := chatMessagesProjection(messages, actor)
	if s.socketService != nil {
		for i := range views {
			payload, _ := json.Marshal(types.ChatMessageEvent{Action: constants.CMD_SEND_MESSAGE, Message: &views[i]})
			_ = s.socketService.BroadcastToRoom("/", targetID.String(), "chat", string(payload))
		}
	}
	return views, nil
}
