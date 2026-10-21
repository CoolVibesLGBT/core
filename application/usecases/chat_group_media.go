package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	domainmedia "core/domain/media"
	"core/models"
	"core/models/chat"
	"github.com/google/uuid"
)

var _ ports.ChatGroupMediaUseCase = (*ChatService)(nil)

func (s *ChatService) UpdateGroup(ctx context.Context, actor *models.User, id uuid.UUID, title, description string, avatar *domainmedia.UploadedFile, remove bool, options ...chat.GroupOptions) (*types.Chat, error) {
	repo, ok := s.chatRepo.(ports.ChatGroupMediaRepository)
	if !ok || actor == nil {
		return nil, chat.ErrPermissionDenied
	}
	entity, err := repo.UpdateGroup(ctx, actor, id, title, description, avatar, remove, options...)
	if err != nil {
		return nil, err
	}
	users := []*models.User{actor}
	for i := range entity.Participants {
		users = append(users, &entity.Participants[i].User)
	}
	s.notifyChatParticipantsChanged(id, users...)
	return chatProjection(entity, actor.ID, actor), nil
}
