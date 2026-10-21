package ports

import (
	"context"
	"core/application/types"
	domainmedia "core/domain/media"
	"core/models"
	"time"

	"github.com/google/uuid"
)

// ChatUseCase is the inbound application boundary consumed by HTTP adapters.
// Query and command results use transport-safe application projections; raw
// persistence entities are never returned to an adapter for serialization.
type ChatUseCase interface {
	CreateGroupChat(ctx context.Context, participantIdentifiers []string, actor *models.User, title, description string) (*types.Chat, error)
	AddChatParticipant(ctx context.Context, chatID uuid.UUID, participantIdentifier string, actor *models.User) error
	RemoveChatParticipant(ctx context.Context, chatID uuid.UUID, participantIdentifier string, actor *models.User) error
	GetChatParticipants(ctx context.Context, chatID uuid.UUID, actor *models.User) ([]types.ChatParticipant, error)
	SendTypingEvent(chatID uuid.UUID, user *models.User, typing bool) error
	CreateChatFromIdentifier(ctx context.Context, participantIdentifier string, actor *models.User, chatType string) (*types.Chat, error)
	FetchChats(ctx context.Context, userID uuid.UUID, cursor *ChatListCursor, limit int) ([]types.Chat, *string, error)
	AddMessageToChat(ctx context.Context, values map[string][]string, files []*domainmedia.UploadedFile, author *models.User) (*types.ChatMessage, error)
	FetchChatMessages(ctx context.Context, userID, chatID uuid.UUID, cursor *ChatMessageListCursor, limit int) ([]types.ChatMessage, *string, error)
	MarkChatMessageRead(ctx context.Context, authUser *models.User, chatID uuid.UUID, messageIDs []uuid.UUID) error
	OpenMessage(ctx context.Context, authUser *models.User, chatID, messageID uuid.UUID, now time.Time) (*types.ChatOpenMessageResult, error)
	PinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	UnpinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	DeleteMessageForUser(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	DeleteMessageForAll(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	DeleteChatForUser(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error
	DeleteChatForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error
	DeleteChat(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error
	DeleteMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	DeleteChatHistoryForUser(ctx context.Context, authUser *models.User, chatID uuid.UUID) error
	DeleteChatHistoryForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error
}
