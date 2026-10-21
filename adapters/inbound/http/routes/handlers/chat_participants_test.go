package handlers

import (
	"context"
	"core/application/usecases"
	"core/constants"
	"core/models"
	"core/models/chat"
	modelutils "core/models/utils"
	"encoding/json"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type groupHandlerRepo struct {
	chatHandlerRepo
	group    *chat.Chat
	groupIDs []uuid.UUID
}

func (r *groupHandlerRepo) CreateGroupChat(_ context.Context, creatorID uuid.UUID, ids []uuid.UUID, title, description *modelutils.LocalizedString) (*chat.Chat, error) {
	r.groupIDs = ids
	r.group = &chat.Chat{ID: uuid.New(), CreatorID: creatorID, Type: chat.ChatTypeGroup, Title: title, Description: description, Participants: []chat.ChatParticipant{{UserID: creatorID}}}
	for _, id := range ids {
		r.group.Participants = append(r.group.Participants, chat.ChatParticipant{UserID: id})
	}
	return r.group, nil
}
func (r *groupHandlerRepo) AddParticipant(_ context.Context, actor *models.User, chatID, userID uuid.UUID) error {
	return r.recordMutation(constants.CMD_CHAT_PARTICIPANT_ADD, actor, chatID, userID, uuid.Nil)
}
func (r *groupHandlerRepo) RemoveParticipant(_ context.Context, actor *models.User, chatID, userID uuid.UUID) error {
	return r.recordMutation(constants.CMD_CHAT_PARTICIPANT_REMOVE, actor, chatID, userID, uuid.Nil)
}
func (r *groupHandlerRepo) GetParticipants(_ context.Context, actor *models.User, chatID uuid.UUID) ([]chat.ChatParticipant, error) {
	if r.mutationErr != nil {
		return nil, r.mutationErr
	}
	member := models.User{ID: uuid.New(), PublicID: 55, UserName: "member"}
	return []chat.ChatParticipant{{UserID: actor.ID, User: *actor}, {UserID: member.ID, User: member}}, nil
}

func TestCreateGroupChatResolvesAndDeduplicatesParticipantsThroughHTTP(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10, DefaultLanguage: "tr"}
	first, second := &models.User{ID: uuid.New(), PublicID: 20}, &models.User{ID: uuid.New(), PublicID: 30}
	for _, urlEncoded := range []bool{false, true} {
		t.Run(map[bool]string{false: "json multipart", true: "repeated urlencoded"}[urlEncoded], func(t *testing.T) {
			repo := &groupHandlerRepo{}
			users := &chatHandlerUserRepo{byUUID: map[uuid.UUID]*models.User{first.ID: first}, byPublicID: map[int64]*models.User{10: actor, 20: first, 30: second}}
			service := usecases.NewChatService(nil, users, nil, nil, nil, repo, nil)
			handler := HandleCreateChat(service)
			var body []byte
			if urlEncoded {
				app := fiber.New()
				app.Post("/", func(c fiber.Ctx) error { c.Locals("authenticatedUser", actor); return handler(c) })
				fields := url.Values{"type": {"group"}, "title": {"Arkadaşlar"}, "participant_ids[]": {"20", first.ID.String(), "30", "10"}}
				request := httptest.NewRequest("POST", "/", strings.NewReader(fields.Encode()))
				request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				response, err := app.Test(request)
				if err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := response.Body.Close(); err != nil {
						t.Errorf("close response body: %v", err)
					}
				}()
				if response.StatusCode != 200 {
					t.Fatalf("status=%d", response.StatusCode)
				}
				body, _ = io.ReadAll(response.Body)
			} else {
				response := performMultipartHandlerRequest(t, handler, actor, map[string]string{"type": "group", "title": "Arkadaşlar", "participant_ids": `["20","` + first.ID.String() + `",30,"10"]`}, nil)
				defer func() {
					if err := response.Body.Close(); err != nil {
						t.Errorf("close response body: %v", err)
					}
				}()
				if response.StatusCode != 200 {
					t.Fatalf("status=%d", response.StatusCode)
				}
				body, _ = io.ReadAll(response.Body)
			}
			if len(repo.groupIDs) != 2 || repo.groupIDs[0] != first.ID || repo.groupIDs[1] != second.ID {
				t.Fatalf("resolved IDs=%v", repo.groupIDs)
			}
			if repo.group.Title.GetLocalizedString("tr") != "Arkadaşlar" {
				t.Fatal("group title lost")
			}
			if strings.Contains(string(body), actor.ID.String()) || strings.Contains(string(body), first.ID.String()) {
				t.Fatal("user UUID leaked")
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), `"user_id":"20"`) || !strings.Contains(string(body), `"user_id":"30"`) {
				t.Fatalf("missing public participants: %s", body)
			}
		})
	}
}

func TestChatParticipantCommandsResolveIdentityAndMapAuthorization(t *testing.T) {
	actor, target := &models.User{ID: uuid.New(), PublicID: 10}, &models.User{ID: uuid.New(), PublicID: 20}
	repo := &groupHandlerRepo{}
	service := usecases.NewChatService(nil, &chatHandlerUserRepo{byPublicID: map[int64]*models.User{20: target}}, nil, nil, nil, repo, nil)
	chatID := uuid.New()
	response := performMultipartHandlerRequest(t, HandleAddChatParticipant(service), actor, map[string]string{"chat_id": chatID.String(), "participant_id": "20"}, nil)
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
	if response.StatusCode != 200 || repo.mutationUserID != target.ID || repo.mutationAuth != actor {
		t.Fatal("add did not resolve identity")
	}
	response = performMultipartHandlerRequest(t, HandleRemoveChatParticipant(service), actor, map[string]string{"chat_id": chatID.String()}, nil)
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
	if response.StatusCode != 200 || repo.mutationUserID != actor.ID {
		t.Fatal("leave did not use authenticated participant")
	}
	response = performMultipartHandlerRequest(t, HandleGetChatParticipants(service), actor, map[string]string{"chat_id": chatID.String()}, nil)
	body, _ := io.ReadAll(response.Body)
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
	if response.StatusCode != 200 || !strings.Contains(string(body), `"user_id":"55"`) || strings.Contains(string(body), actor.ID.String()) {
		t.Fatalf("invalid participant projection: %s", body)
	}
	repo.mutationErr = chat.ErrNotParticipant
	response = performMultipartHandlerRequest(t, HandleAddChatParticipant(service), actor, map[string]string{"chat_id": chatID.String(), "participant_id": "20"}, nil)
	if err := response.Body.Close(); err != nil {
		t.Errorf("close response body: %v", err)
	}
	if response.StatusCode != 403 {
		t.Fatalf("authorization status=%d", response.StatusCode)
	}
}
