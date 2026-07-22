package usecases

import (
	"core/application/types"
	"core/models"
	chatmodel "core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"encoding/json"

	"github.com/google/uuid"
)

func chatProjection(entity *chatmodel.Chat, viewerID uuid.UUID, knownUsers ...*models.User) *types.Chat {
	if entity == nil {
		return nil
	}

	users := make(map[uuid.UUID]*models.User, len(entity.Participants)+len(knownUsers)+1)
	for _, user := range knownUsers {
		if user != nil && user.ID != uuid.Nil {
			users[user.ID] = user
		}
	}
	for i := range entity.Participants {
		participant := &entity.Participants[i]
		if participant.User.ID != uuid.Nil {
			users[participant.User.ID] = &participant.User
		}
	}
	if entity.PinnedBy != nil && entity.PinnedBy.ID != uuid.Nil {
		users[entity.PinnedBy.ID] = entity.PinnedBy
	}

	result := &types.Chat{
		ID:                   entity.ID,
		Type:                 string(entity.Type),
		Title:                cloneLocalizedString(entity.Title),
		Description:          cloneLocalizedString(entity.Description),
		Avatar:               chatMediaProjection(entity.Avatar),
		PinnedMsgID:          entity.PinnedMsgID,
		LastMessageID:        entity.LastMessageID,
		LastMessageTimestamp: entity.LastMessageTimestamp,
		CreatedAt:            entity.CreatedAt,
		UpdatedAt:            entity.UpdatedAt,
	}

	if creator := users[entity.CreatorID]; creator != nil && creator.PublicID > 0 {
		creatorID := types.SnowflakeID(creator.PublicID)
		result.CreatorID = &creatorID
	}
	if pinnedBy := users[valueOrNilUUID(entity.PinnedByID)]; pinnedBy != nil && pinnedBy.PublicID > 0 {
		pinnedByID := types.SnowflakeID(pinnedBy.PublicID)
		pinnedByView := chatUserProjection(pinnedBy)
		result.PinnedByID = &pinnedByID
		result.PinnedBy = &pinnedByView
	}

	result.Participants = make([]types.ChatParticipant, 0, len(entity.Participants))
	for i := range entity.Participants {
		participant := &entity.Participants[i]
		user := users[participant.UserID]
		if user == nil && participant.User.ID != uuid.Nil {
			user = &participant.User
		}
		userView := chatUserProjection(user)
		participantView := types.ChatParticipant{
			UserID: types.SnowflakeID(userView.PublicID),
			User:   userView,
		}
		if participant.UserID == viewerID {
			unreadCount := participant.UnreadCount
			isMuted := participant.IsMuted
			participantView.UnreadCount = &unreadCount
			participantView.IsMuted = &isMuted
			participantView.LastReadAt = participant.LastReadAt
			result.UnreadCount = participant.UnreadCount
			result.IsMuted = participant.IsMuted
		}
		result.Participants = append(result.Participants, participantView)
	}

	result.LastMessage = chatMessageProjection(entity.LastMessage, users, 1)
	result.PinnedMsg = chatMessageProjection(entity.PinnedMsg, users, 1)
	result.Messages = make([]types.ChatMessage, 0, len(entity.Messages))
	for i := range entity.Messages {
		if message := chatMessageProjection(&entity.Messages[i], users, 1); message != nil {
			result.Messages = append(result.Messages, *message)
		}
	}
	return result
}

func chatListProjection(entities []chatmodel.Chat, viewerID uuid.UUID) []types.Chat {
	result := make([]types.Chat, 0, len(entities))
	for i := range entities {
		if view := chatProjection(&entities[i], viewerID); view != nil {
			result = append(result, *view)
		}
	}
	return result
}

func chatMessagesProjection(messages []post.Post, knownUsers ...*models.User) []types.ChatMessage {
	users := make(map[uuid.UUID]*models.User, len(messages)+len(knownUsers))
	for _, user := range knownUsers {
		if user != nil && user.ID != uuid.Nil {
			users[user.ID] = user
		}
	}
	for i := range messages {
		if messages[i].Author.ID != uuid.Nil {
			users[messages[i].Author.ID] = &messages[i].Author
		}
	}

	result := make([]types.ChatMessage, 0, len(messages))
	for i := range messages {
		if message := chatMessageProjection(&messages[i], users, 1); message != nil {
			result = append(result, *message)
		}
	}
	return result
}

func chatMessageProjection(message *post.Post, users map[uuid.UUID]*models.User, parentDepth int) *types.ChatMessage {
	if message == nil {
		return nil
	}

	author := &message.Author
	if knownAuthor := users[message.AuthorID]; knownAuthor != nil {
		author = knownAuthor
	}
	authorView := chatUserProjection(author)
	result := &types.ChatMessage{
		ID:               message.ID,
		ParentID:         message.ParentID,
		PublicID:         types.SnowflakeID(message.PublicID),
		PostKind:         string(message.PostKind),
		Domain:           string(message.Domain),
		ContentCategory:  string(message.ContentCategory),
		ContentableID:    message.ContentableID,
		ContentableType:  message.ContentableType,
		AuthorID:         types.SnowflakeID(authorView.PublicID),
		Author:           authorView,
		Title:            cloneLocalizedString(message.Title),
		Content:          cloneLocalizedString(message.Content),
		Summary:          cloneLocalizedString(message.Summary),
		Location:         chatLocationProjection(message.Location),
		ExpiresInSeconds: message.ExpiresInSeconds,
		OpenedAt:         message.OpenedAt,
		ExpiresAt:        message.ExpiresAt,
		ViewOnce:         message.ViewOnce,
		IsDisappearing:   message.ViewOnce || message.ExpiresInSeconds != nil,
		ContentHidden:    message.ContentHidden,
		ViewedOnce:       message.ViewedOnce,
		ClientID:         message.ClientID,
		CreatedAt:        message.CreatedAt,
		UpdatedAt:        message.UpdatedAt,
	}
	if parentDepth > 0 {
		result.Parent = chatMessageProjection(message.Parent, users, parentDepth-1)
	}
	result.Attachments = make([]types.ChatMedia, 0, len(message.Attachments))
	for _, attachment := range message.Attachments {
		if view := chatMediaProjection(attachment); view != nil {
			result.Attachments = append(result.Attachments, *view)
		}
	}
	return result
}

func chatUserProjection(user *models.User) types.ChatUser {
	if user == nil {
		return types.ChatUser{}
	}
	publicID := types.SnowflakeID(user.PublicID)
	return types.ChatUser{
		ID:          publicID,
		PublicID:    publicID,
		UserName:    user.UserName,
		DisplayName: user.DisplayName,
		Avatar:      publicUserMediaFromModel(user.Avatar),
	}
}

func chatMediaProjection(value *media.Media) *types.ChatMedia {
	if value == nil {
		return nil
	}
	publicID := types.SnowflakeID(value.PublicID)
	result := &types.ChatMedia{
		ID:       publicID,
		PublicID: publicID,
		File: types.ChatMediaFile{
			URL:      value.File.URL,
			MimeType: value.File.MimeType,
			Name:     value.File.Name,
		},
	}
	if value.File.Variants != nil {
		if encoded, err := json.Marshal(value.File.Variants); err == nil && string(encoded) != "null" {
			result.File.Variants = encoded
		}
	}
	return result
}

func chatLocationProjection(value *modelutils.Location) *types.ChatMessageLocation {
	if value == nil {
		return nil
	}
	return &types.ChatMessageLocation{
		CountryCode: value.CountryCode,
		Display:     value.Display,
		Address:     value.Address,
		City:        value.City,
		Country:     value.Country,
		Region:      value.Region,
		Latitude:    value.Latitude,
		Longitude:   value.Longitude,
	}
}

func valueOrNilUUID(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}
