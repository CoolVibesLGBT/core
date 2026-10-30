package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/models"
	"core/models/chat"
	"github.com/google/uuid"
)

var _ ports.ChatGroupInvites = (*ChatService)(nil)

func (s *ChatService) ExportGroupInvite(ctx context.Context, actor *models.User, id uuid.UUID, rotate bool) (*types.ChatGroupInvite, error) {
	r, ok := s.chatRepo.(ports.ChatGroupInvitesRepository)
	if !ok {
		return nil, chat.ErrPermissionDenied
	}
	entity, err := r.ExportGroupInvite(ctx, actor, id, rotate)
	if err != nil {
		return nil, err
	}
	return &types.ChatGroupInvite{Token: entity.Token, Uses: entity.Uses}, nil
}
func (s *ChatService) PreviewGroup(ctx context.Context, token, username string) (*types.ChatGroupPreview, error) {
	r, ok := s.chatRepo.(ports.ChatGroupInvitesRepository)
	if !ok {
		return nil, chat.ErrGroupLinkInvalid
	}
	entity, count, err := r.PreviewGroup(ctx, token, username)
	if err != nil {
		return nil, err
	}
	return &types.ChatGroupPreview{ID: entity.ID, Title: cloneLocalizedString(entity.Title), Description: cloneLocalizedString(entity.Description), Visibility: entity.Visibility, PublicUsername: entity.PublicUsername, Avatar: chatMediaProjection(entity.Avatar), Cover: chatMediaProjection(entity.Cover), Hero: chatMediaProjection(entity.Hero), MemberCount: count}, nil
}
func (s *ChatService) JoinGroup(ctx context.Context, actor *models.User, token, username string) (*types.Chat, error) {
	r, ok := s.chatRepo.(ports.ChatGroupInvitesRepository)
	if !ok || actor == nil {
		return nil, chat.ErrPermissionDenied
	}
	entity, err := r.JoinGroup(ctx, actor, token, username)
	if err != nil {
		return nil, err
	}
	users := []*models.User{actor}
	for i := range entity.Participants {
		users = append(users, &entity.Participants[i].User)
	}
	s.notifyChatParticipantsChanged(entity.ID, users...)
	return chatProjection(entity, actor.ID, actor), nil
}
