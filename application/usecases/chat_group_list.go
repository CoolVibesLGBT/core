package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
)

func (s *ChatService) ListPublicGroups(ctx context.Context, query ports.PublicGroupQuery) ([]types.ChatGroupPreview, *string, error) {
	repo, ok := s.chatRepo.(ports.ChatGroupListRepository)
	if !ok {
		return nil, nil, constants.ErrInternalServer
	}
	if query.Limit <= 0 {
		query.Limit = constants.DEFAULT_LIMIT
	}
	if query.Limit > constants.MAXIMUM_LIMIT {
		query.Limit = constants.MAXIMUM_LIMIT
	}
	page, err := repo.ListPublicGroups(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	groups := make([]types.ChatGroupPreview, 0, len(page.Groups))
	for _, row := range page.Groups {
		entity := row.Chat
		groups = append(groups, types.ChatGroupPreview{ID: entity.ID, Title: cloneLocalizedString(entity.Title), Description: cloneLocalizedString(entity.Description), Visibility: entity.Visibility, PublicUsername: entity.PublicUsername, Avatar: chatMediaProjection(entity.Avatar), Cover: chatMediaProjection(entity.Cover), Hero: chatMediaProjection(entity.Hero), MemberCount: row.MemberCount})
	}
	var cursor *string
	if page.HasMore && len(page.Groups) > 0 {
		value := page.Groups[len(page.Groups)-1].ID.String()
		cursor = &value
	}
	return groups, cursor, nil
}
