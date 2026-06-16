package usecases

import (
	"context"
	"core/models"
	"core/models/chat"
	"core/models/post"
	"testing"

	"github.com/google/uuid"
)

func TestChatServiceListAndReadMethodsDelegateIDs(t *testing.T) {
	userID := uuid.New()
	chatID := uuid.New()
	messageID := uuid.New()
	chatRepo := &fakeChatRepository{
		chatsByUserID:    []chat.Chat{{ID: chatID, CreatorID: userID}},
		messagesByChatID: []post.Post{{ID: messageID, ContentableID: &chatID}},
	}
	service := NewChatService(&fakeRealtimeNotifier{}, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	chats, err := service.GetChatsByUserID(userID)
	if err != nil || len(chats) != 1 || chatRepo.chatsByUserIDArg != userID {
		t.Fatalf("GetChatsByUserID() = %#v, arg=%s, err=%v", chats, chatRepo.chatsByUserIDArg, err)
	}

	messages, err := service.GetMessagesByChatID(userID, chatID)
	if err != nil || len(messages) != 1 || chatRepo.messageUserID != userID || chatRepo.messageChatID != chatID {
		t.Fatalf("GetMessagesByChatID() = %#v, user=%s chat=%s err=%v", messages, chatRepo.messageUserID, chatRepo.messageChatID, err)
	}
}

func TestChatServiceMessageActionsDelegateArguments(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	messageID := uuid.New()
	chatRepo := &fakeChatRepository{}
	service := NewChatService(&fakeRealtimeNotifier{}, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	checkAction := func(name string, call func() error, wantUserID uuid.UUID, wantMessageID uuid.UUID) {
		t.Helper()
		if err := call(); err != nil {
			t.Fatalf("%s() error = %v", name, err)
		}
		if chatRepo.action != name || chatRepo.actionAuthUser != authUser || chatRepo.actionChatID != chatID || chatRepo.actionUserID != wantUserID || chatRepo.actionMessageID != wantMessageID {
			t.Fatalf("%s() action mismatch: action=%q auth=%p chat=%s user=%s message=%s", name, chatRepo.action, chatRepo.actionAuthUser, chatRepo.actionChatID, chatRepo.actionUserID, chatRepo.actionMessageID)
		}
	}

	checkAction("pin", func() error {
		return service.PinMessage(context.Background(), authUser, chatID, authUser.ID, messageID)
	}, authUser.ID, messageID)
	checkAction("unpin", func() error {
		return service.UnpinMessage(context.Background(), authUser, chatID, authUser.ID, messageID)
	}, authUser.ID, messageID)
	checkAction("delete_message_for_user", func() error {
		return service.DeleteMessageForUser(context.Background(), authUser, chatID, authUser.ID, messageID)
	}, authUser.ID, messageID)
	checkAction("delete_message_for_all", func() error {
		return service.DeleteMessageForAll(context.Background(), authUser, chatID, authUser.ID, messageID)
	}, authUser.ID, messageID)
	checkAction("delete_message", func() error {
		return service.DeleteMessage(context.Background(), authUser, chatID, authUser.ID, messageID)
	}, authUser.ID, messageID)
}

func TestChatServiceChatDeleteAndHistoryActionsDelegateArguments(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	chatRepo := &fakeChatRepository{}
	service := NewChatService(&fakeRealtimeNotifier{}, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	checkAction := func(name string, call func() error, wantUserID uuid.UUID) {
		t.Helper()
		if err := call(); err != nil {
			t.Fatalf("%s() error = %v", name, err)
		}
		if chatRepo.action != name || chatRepo.actionAuthUser != authUser || chatRepo.actionChatID != chatID || chatRepo.actionUserID != wantUserID {
			t.Fatalf("%s() action mismatch: action=%q auth=%p chat=%s user=%s", name, chatRepo.action, chatRepo.actionAuthUser, chatRepo.actionChatID, chatRepo.actionUserID)
		}
	}

	checkAction("delete_chat_for_user", func() error {
		return service.DeleteChatForUser(context.Background(), authUser, chatID, authUser.ID)
	}, authUser.ID)
	checkAction("delete_chat_for_all", func() error {
		return service.DeleteChatForAll(context.Background(), authUser, chatID)
	}, uuid.Nil)
	checkAction("delete_chat", func() error {
		return service.DeleteChat(context.Background(), authUser, chatID, authUser.ID)
	}, authUser.ID)
	checkAction("delete_history_for_user", func() error {
		return service.DeleteChatHistoryForUser(context.Background(), authUser, chatID)
	}, uuid.Nil)
	checkAction("delete_history_for_all", func() error {
		return service.DeleteChatHistoryForAll(context.Background(), authUser, chatID)
	}, uuid.Nil)
}

func TestChatServiceMarkReadDelegatesMessageIDs(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	chatID := uuid.New()
	messageIDs := []uuid.UUID{uuid.New(), uuid.New()}
	chatRepo := &fakeChatRepository{}
	service := NewChatService(&fakeRealtimeNotifier{}, &fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeMatchesRepository{}, chatRepo, &fakeNotificationRepository{})

	if err := service.MarkChatMessageRead(context.Background(), authUser, chatID, messageIDs); err != nil {
		t.Fatalf("MarkChatMessageRead() error = %v", err)
	}
	if chatRepo.markedReadChatID != chatID || len(chatRepo.markedReadMessages) != 2 || chatRepo.markedReadMessages[1] != messageIDs[1] {
		t.Fatalf("expected read markers to pass through, got chat=%s messages=%#v", chatRepo.markedReadChatID, chatRepo.markedReadMessages)
	}
}
