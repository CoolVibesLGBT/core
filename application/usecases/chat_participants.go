package usecases

import (
	"context"
	"core/application/types"
	"core/constants"
	"core/models"
	"core/models/chat"
	modelutils "core/models/utils"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func (s *ChatService) CreateGroupChat(ctx context.Context, identifiers []string, actor *models.User, title, description string) (*types.Chat, error) {
	if actor == nil || actor.ID == uuid.Nil {
		return nil, chat.ErrNotParticipant
	}
	if len(identifiers) == 0 {
		return nil, constants.ErrInvalidParticipantsLength
	}
	users := []*models.User{actor}
	ids := make([]uuid.UUID, 0, len(identifiers))
	seen := map[uuid.UUID]bool{actor.ID: true}
	for _, identifier := range identifiers {
		user, err := s.resolveChatParticipant(ctx, identifier)
		if err != nil {
			return nil, err
		}
		if seen[user.ID] {
			continue
		}
		seen[user.ID] = true
		ids = append(ids, user.ID)
		users = append(users, user)
	}
	if len(ids) == 0 {
		return nil, constants.ErrInvalidParticipantsLength
	}
	language := actor.DefaultLanguage
	if language == "" {
		language = "en"
	}
	entity, err := s.chatRepo.CreateGroupChat(ctx, actor.ID, ids, modelutils.MakeLocalizedString(language, strings.TrimSpace(title)), modelutils.MakeLocalizedString(language, strings.TrimSpace(description)))
	if err != nil {
		return nil, err
	}
	s.notifyChatParticipantsChanged(entity.ID, users...)
	return chatProjection(entity, actor.ID, users...), nil
}

func (s *ChatService) AddChatParticipant(ctx context.Context, chatID uuid.UUID, identifier string, actor *models.User) error {
	if actor == nil || actor.ID == uuid.Nil {
		return chat.ErrNotParticipant
	}
	user, err := s.resolveChatParticipant(ctx, identifier)
	if err != nil {
		return err
	}
	if err := s.chatRepo.AddParticipant(ctx, actor, chatID, user.ID); err != nil {
		return err
	}
	s.notifyChatParticipantsChanged(chatID, user)
	return nil
}

func (s *ChatService) RemoveChatParticipant(ctx context.Context, chatID uuid.UUID, identifier string, actor *models.User) error {
	if actor == nil || actor.ID == uuid.Nil {
		return chat.ErrNotParticipant
	}
	user := actor
	if strings.TrimSpace(identifier) != "" {
		var err error
		user, err = s.resolveChatParticipant(ctx, identifier)
		if err != nil {
			return err
		}
	}
	if err := s.chatRepo.RemoveParticipant(ctx, actor, chatID, user.ID); err != nil {
		return err
	}
	s.notifyChatParticipantsChanged(chatID, user)
	return nil
}

func (s *ChatService) GetChatParticipants(ctx context.Context, chatID uuid.UUID, actor *models.User) ([]types.ChatParticipant, error) {
	if actor == nil || actor.ID == uuid.Nil {
		return nil, chat.ErrNotParticipant
	}
	participants, err := s.chatRepo.GetParticipants(ctx, actor, chatID)
	if err != nil {
		return nil, err
	}
	return chatProjection(&chat.Chat{Participants: participants}, actor.ID).Participants, nil
}

func (s *ChatService) notifyChatParticipantsChanged(chatID uuid.UUID, users ...*models.User) {
	if s.socketService == nil {
		return
	}
	payload, err := json.Marshal(struct {
		Action string    `json:"action"`
		ChatID uuid.UUID `json:"chat_id"`
	}{constants.CMD_CHAT_PARTICIPANTS_CHANGED, chatID})
	if err != nil {
		return
	}
	rooms := []string{chatID.String()}
	for _, user := range users {
		if user != nil && user.PublicID > 0 {
			rooms = append(rooms, "user:"+strconv.FormatInt(user.PublicID, 10))
		}
	}
	for _, room := range rooms {
		if err := s.socketService.BroadcastToRoom("/", room, "chat", string(payload)); err != nil {
			log.Printf("chat participants event: %v", err)
		}
	}
}
