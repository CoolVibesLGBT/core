package handlers

import (
	"core/application/ports"
	"core/constants"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

func TestChatListAndMessageHandlersNeverSerializeRawUserModels(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 920000000000001, UserName: "viewer"}
	otherID := uuid.New()
	mediaID := uuid.New()
	fileID := uuid.New()
	storagePath := "/var/lib/coolvibes/private/chat-leak.png"
	other := models.User{
		ID:               otherID,
		PublicID:         920000000000002,
		UserName:         "public-name",
		DisplayName:      "Public Name",
		Email:            "handler-private@example.test",
		Password:         "handler-password-hash",
		Balance:          decimal.RequireFromString("444.55"),
		PreferencesFlags: "sensitive-flags",
		UserRole:         constants.UserRoleAdmin,
		BroadcastInfo:    datatypes.JSON(`{"socket":"broadcast-secret"}`),
		Subscriptions:    datatypes.JSON(`{"endpoint":"subscription-secret"}`),
		Avatar: &media.Media{ID: mediaID, PublicID: 820000000000001, FileID: fileID, OwnerID: otherID, UserID: otherID, File: modelutils.FileMetadata{
			ID: fileID, URL: "https://cdn.example.test/safe-avatar.png", StoragePath: "/private/avatar.png",
		}},
	}
	chatID := uuid.New()
	messageID := uuid.New()
	message := post.Post{
		ID: messageID, PublicID: 720000000000001, PostKind: post.PostKindMessage,
		ContentableID: &chatID, AuthorID: otherID, Author: other,
		Content:  &modelutils.LocalizedString{"en": "safe message"},
		Metadata: datatypes.JSON(`{"secret":"metadata-secret"}`),
		Attachments: []*media.Media{{
			ID: uuid.New(), PublicID: 820000000000002, FileID: uuid.New(), OwnerID: messageID, UserID: otherID,
			Role: media.RoleChatImage, File: modelutils.FileMetadata{URL: "https://cdn.example.test/safe-chat.png", StoragePath: storagePath, MimeType: "image/png"},
		}},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	repo := &chatHandlerRepo{
		chatListPage: ports.ChatListPage{Chats: []chat.Chat{{
			ID: chatID, Type: chat.ChatTypePrivate, CreatorID: authUser.ID,
			Participants: []chat.ChatParticipant{
				{UserID: authUser.ID, User: *authUser, Role: chat.ParticipantRoleOwner, UnreadCount: 1},
				{UserID: otherID, User: other, Role: chat.ParticipantRoleAdmin, UnreadCount: 999},
			},
			LastMessageID: &messageID, LastMessage: &message,
		}}},
		messagePage: ports.ChatMessageListPage{Messages: []post.Post{message}},
	}
	service := newChatHandlerService(&chatHandlerUserRepo{}, repo, &chatHandlerNotifier{})

	chatResponse := performMultipartHandlerRequest(t, HandleGetChatsByUserID(service), authUser, nil, nil)
	assertSafeChatHandlerResponse(t, chatResponse.Body, otherID, mediaID, fileID, storagePath)

	messageResponse := performMultipartHandlerRequest(t, HandleGetMessagesByChatID(service), authUser, map[string]string{"chat_id": chatID.String()}, nil)
	assertSafeChatHandlerResponse(t, messageResponse.Body, otherID, mediaID, fileID, storagePath)
}

func assertSafeChatHandlerResponse(t *testing.T, body io.ReadCloser, forbiddenIDs ...interface{}) {
	t.Helper()
	defer func() { _ = body.Close() }()
	encoded, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(encoded)
	for _, id := range forbiddenIDs {
		var value string
		switch typed := id.(type) {
		case uuid.UUID:
			value = typed.String()
		case string:
			value = typed
		}
		if value != "" && strings.Contains(payload, value) {
			t.Fatalf("handler response leaked persistence value %q: %s", value, payload)
		}
	}
	for _, forbidden := range []string{
		`"balance"`, `"preferences_flags"`, `"user_role"`, `"email"`, `"password"`, `"broadcast_info"`,
		`"subscriptions"`, `"storage_path"`, `"file_id"`, `"owner_id"`,
		`"role"`, "metadata-secret", "broadcast-secret", "subscription-secret",
		"handler-private@example.test", "handler-password-hash",
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("handler response leaked %q: %s", forbidden, payload)
		}
	}
	for _, expected := range []string{
		`"id":"920000000000002"`, `"public_id":"920000000000002"`,
		`"username":"public-name"`, `"displayname":"Public Name"`,
		"https://cdn.example.test/safe-avatar.png", "https://cdn.example.test/safe-chat.png",
	} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("handler response omitted %q: %s", expected, payload)
		}
	}
}
