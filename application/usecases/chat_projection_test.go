package usecases

import (
	"context"
	"core/application/ports"
	"core/constants"
	"core/models"
	chatmodel "core/models/chat"
	"core/models/media"
	"core/models/post"
	modelutils "core/models/utils"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

func TestChatProjectionJSONExcludesPersistenceUserAndFileMetadata(t *testing.T) {
	viewerID := uuid.New()
	otherID := uuid.New()
	avatarMediaID := uuid.New()
	avatarFileID := uuid.New()
	attachmentMediaID := uuid.New()
	attachmentFileID := uuid.New()
	storagePath := "/srv/private/storage/chat-secret.jpg"
	adminRole := chatmodel.ParticipantRoleAdmin
	viewer := models.User{
		ID:               viewerID,
		PublicID:         910000000000001,
		UserName:         "viewer",
		DisplayName:      "Viewer",
		Balance:          decimal.RequireFromString("123456.789"),
		PreferencesFlags: "private-preference-bits",
		UserRole:         constants.UserRoleAdmin,
		BroadcastInfo:    datatypes.JSON(`{"socket_token":"socket-secret"}`),
		Subscriptions:    datatypes.JSON(`{"endpoint":"subscription-secret"}`),
	}
	other := models.User{
		ID:               otherID,
		PublicID:         910000000000002,
		UserName:         "safe-user",
		DisplayName:      "Safe User",
		Email:            "private-chat@example.test",
		Password:         "private-password-hash",
		Balance:          decimal.RequireFromString("987654.321"),
		PreferencesFlags: "do-not-serialize",
		UserRole:         constants.UserRoleAdmin,
		BroadcastInfo:    datatypes.JSON(`{"room":"broadcast-secret"}`),
		Avatar: &media.Media{
			ID:       avatarMediaID,
			PublicID: 810000000000001,
			FileID:   avatarFileID,
			OwnerID:  otherID,
			UserID:   otherID,
			Role:     media.RoleAvatar,
			File: modelutils.FileMetadata{
				ID:          avatarFileID,
				URL:         "https://cdn.example.test/avatar.jpg",
				StoragePath: "/srv/private/storage/avatar.jpg",
				Variants: &modelutils.FileVariants{Image: &modelutils.ImageVariants{
					Small: &modelutils.VariantInfo{URL: "https://cdn.example.test/avatar-small.jpg"},
				}},
			},
		},
	}
	messageID := uuid.New()
	chatID := uuid.New()
	message := post.Post{
		ID:              messageID,
		PublicID:        710000000000001,
		PostKind:        post.PostKindMessage,
		AuthorID:        other.ID,
		Author:          other,
		ContentableID:   &chatID,
		ContentableType: stringPointer(string(post.PostKindChat)),
		Content:         &modelutils.LocalizedString{"en": "hello"},
		Metadata:        datatypes.JSON(`{"private_metadata":"metadata-secret"}`),
		Extras:          datatypes.JSON(`{"private_extra":"extra-secret"}`),
		Attachments: []*media.Media{{
			ID:       attachmentMediaID,
			PublicID: 810000000000002,
			FileID:   attachmentFileID,
			OwnerID:  messageID,
			UserID:   otherID,
			Role:     media.RoleChatImage,
			File: modelutils.FileMetadata{
				ID:          attachmentFileID,
				URL:         "https://cdn.example.test/chat.jpg",
				StoragePath: storagePath,
				MimeType:    "image/jpeg",
				Name:        "chat.jpg",
			},
		}},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	entity := &chatmodel.Chat{
		ID:            chatID,
		Type:          chatmodel.ChatTypePrivate,
		CreatorID:     viewer.ID,
		LastMessageID: &messageID,
		LastMessage:   &message,
		Participants: []chatmodel.ChatParticipant{
			{UserID: viewer.ID, User: viewer, Role: adminRole, UnreadCount: 3, IsMuted: true},
			{UserID: other.ID, User: other, Role: adminRole, UnreadCount: 99, IsMuted: true},
		},
		Messages:  []post.Post{message},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	encoded, err := json.Marshal(chatProjection(entity, viewerID))
	if err != nil {
		t.Fatal(err)
	}
	payload := string(encoded)
	for _, forbidden := range []string{
		viewerID.String(), otherID.String(), avatarMediaID.String(), avatarFileID.String(),
		attachmentMediaID.String(), attachmentFileID.String(), storagePath,
		`"balance"`, `"preferences_flags"`, `"user_role"`, `"email"`, `"password"`, `"socket_id"`,
		`"broadcast_info"`, `"subscriptions"`, `"storage_path"`, `"file_id"`,
		`"owner_id"`, `"processing_status"`, `"role"`,
		"metadata-secret", "extra-secret", "socket-secret", "subscription-secret",
		"private-chat@example.test", "private-password-hash",
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("chat projection leaked %q: %s", forbidden, payload)
		}
	}
	for _, expected := range []string{
		`"id":"910000000000002"`, `"public_id":"910000000000002"`,
		`"username":"safe-user"`, `"displayname":"Safe User"`,
		"https://cdn.example.test/avatar.jpg", "https://cdn.example.test/avatar-small.jpg",
		"https://cdn.example.test/chat.jpg",
	} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("chat projection omitted %q: %s", expected, payload)
		}
	}

	var decoded struct {
		Participants []map[string]json.RawMessage `json:"participants"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Participants) != 2 {
		t.Fatalf("participants = %d, want 2", len(decoded.Participants))
	}
	if _, ok := decoded.Participants[0]["unread_count"]; !ok {
		t.Fatal("authenticated participant lost viewer-specific unread_count")
	}
	for _, field := range []string{"unread_count", "is_muted", "last_read_at"} {
		if _, ok := decoded.Participants[1][field]; ok {
			t.Fatalf("other participant's %s leaked: %s", field, payload)
		}
	}
	if _, ok := decoded.Participants[1]["role"]; ok {
		t.Fatalf("participant role leaked: %s", payload)
	}
}

func TestChatOpenProjectionOmitsOpenedMediaUUID(t *testing.T) {
	viewer := &models.User{ID: uuid.New(), PublicID: 910000000000010, UserName: "viewer"}
	chatID := uuid.New()
	messageID := uuid.New()
	mediaID := uuid.New()
	repo := &fakeChatRepository{openResult: &chatmodel.OpenMessageResult{
		Message: &post.Post{ID: messageID, PublicID: 710000000000010, ContentableID: &chatID, Author: *viewer, AuthorID: viewer.ID},
		Media:   &chatmodel.OpenedMedia{ID: mediaID, Name: "once.jpg", MimeType: "image/jpeg", DataBase64: "c2FmZQ=="},
	}}
	service := NewChatService(&fakeRealtimeNotifier{}, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, repo, &fakeNotificationRepository{})

	result, err := service.OpenMessage(context.Background(), viewer, chatID, messageID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), mediaID.String()) || strings.Contains(string(encoded), `"media":{"id"`) {
		t.Fatalf("opened media UUID leaked: %s", encoded)
	}
}

func TestChatMessageSocketUsesSafeApplicationProjection(t *testing.T) {
	chatID := uuid.New()
	userID := uuid.New()
	mediaID := uuid.New()
	fileID := uuid.New()
	storagePath := "/private/socket-only/message.jpg"
	author := &models.User{
		ID: userID, PublicID: 930000000000001, UserName: "socket-user", DisplayName: "Socket User",
		Email: "socket-private@example.test", Password: "socket-password-hash",
		Balance: decimal.RequireFromString("999.99"), PreferencesFlags: "socket-private-flags",
		UserRole: constants.UserRoleAdmin, BroadcastInfo: datatypes.JSON(`{"token":"socket-credential"}`),
	}
	message := &post.Post{
		ID: uuid.New(), PublicID: 730000000000001, PostKind: post.PostKindMessage,
		ContentableID: &chatID, AuthorID: userID, Author: *author,
		Content: &modelutils.LocalizedString{"en": "socket-safe"},
		Attachments: []*media.Media{{
			ID: mediaID, PublicID: 830000000000001, FileID: fileID, OwnerID: chatID, UserID: userID,
			Role: media.RoleChatImage, File: modelutils.FileMetadata{
				ID: fileID, URL: "https://cdn.example.test/socket-message.jpg", StoragePath: storagePath, MimeType: "image/jpeg",
			},
		}},
	}
	notifier := &fakeRealtimeNotifier{}
	service := NewChatService(notifier, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, &fakeChatRepository{message: message}, &fakeNotificationRepository{})

	created, err := service.AddMessageToChat(context.Background(), ports.FormData{}, author)
	if err != nil {
		t.Fatal(err)
	}
	responseJSON, err := json.Marshal(created)
	if err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string]string{"socket": notifier.msg, "response": string(responseJSON)} {
		for _, forbidden := range []string{
			userID.String(), mediaID.String(), fileID.String(), storagePath,
			`"balance"`, `"preferences_flags"`, `"user_role"`, `"email"`, `"password"`, `"broadcast_info"`,
			`"storage_path"`, `"file_id"`, `"owner_id"`, `"role"`, "socket-credential",
			"socket-private@example.test", "socket-password-hash",
		} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("%s payload leaked %q: %s", name, forbidden, payload)
			}
		}
		for _, expected := range []string{
			`"author_id":"930000000000001"`, `"username":"socket-user"`,
			"https://cdn.example.test/socket-message.jpg",
		} {
			if !strings.Contains(payload, expected) {
				t.Fatalf("%s payload omitted %q: %s", name, expected, payload)
			}
		}
	}
}

func stringPointer(value string) *string { return &value }
