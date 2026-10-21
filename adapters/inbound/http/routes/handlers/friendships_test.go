package handlers

import (
	"bytes"
	"context"
	"core/application/ports"
	"core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type friendshipHandlerRepository struct {
	ports.FriendshipRepository
	actor       uuid.UUID
	target      int64
	action      domainuser.FriendshipStatus
	changeCalls int
}

func (r *friendshipHandlerRepository) RequestFriendship(_ context.Context, actor uuid.UUID, target int64) (ports.FriendshipRecord, error) {
	r.actor, r.target = actor, target
	return ports.FriendshipRecord{PublicID: 9007199254740993, RequesterPublicID: 10, RecipientPublicID: 20, Status: domainuser.FriendshipPending}, nil
}
func (r *friendshipHandlerRepository) ChangeFriendship(_ context.Context, actor uuid.UUID, _ int64, action domainuser.FriendshipStatus) (ports.FriendshipRecord, error) {
	r.actor, r.action = actor, action
	r.changeCalls++
	return ports.FriendshipRecord{}, nil
}

func TestFriendshipJSONUsesSessionActorAndStringPublicIDs(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	repo := &friendshipHandlerRepository{}
	service := usecases.NewFriendshipService(repo)
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", actor)
		return HandleFriendshipMutation(service, constants.CMD_USER_FRIEND_REQUEST)(c)
	})
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"user_id":"20","requester_id":"999","recipient_id":"888"}`))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	if response.StatusCode != http.StatusOK || repo.actor != actor.ID || repo.target != 20 {
		t.Fatalf("actor spoof/input parse: %d %s %d", response.StatusCode, repo.actor, repo.target)
	}
	raw, _ := io.ReadAll(response.Body)
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"request_id":"9007199254740993"`, `"requester_id":"10"`, `"recipient_id":"20"`} {
		if !bytes.Contains(raw, []byte(expected)) {
			t.Fatalf("missing directional public ID %s: %s", expected, raw)
		}
	}
	for _, forbidden := range []string{`"first_user_id"`, `"second_user_id"`, actor.ID.String(), `"999"`, `"888"`} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("invalid friendship identity %s: %s", forbidden, raw)
		}
	}
}

func TestFriendshipRespondRejectsNonDecisionActions(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	repo := &friendshipHandlerRepository{}
	service := usecases.NewFriendshipService(repo)
	response := performMultipartHandlerRequest(t, HandleFriendshipMutation(service, constants.CMD_USER_FRIEND_RESPOND), actor, map[string]string{"request_id": "100", "decision": "remove"}, nil)
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()
	if response.StatusCode != http.StatusBadRequest || repo.changeCalls != 0 {
		t.Fatalf("invalid decision reached repository: %d calls=%d", response.StatusCode, repo.changeCalls)
	}
}

func TestFriendshipMutationUsesTypedTargetStatus(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	for _, test := range []struct {
		name, action, decision string
		want                   domainuser.FriendshipStatus
		valid                  bool
	}{
		{"accept", constants.CMD_USER_FRIEND_RESPOND, "accepted", domainuser.FriendshipAccepted, true},
		{"deny", constants.CMD_USER_FRIEND_RESPOND, "denied", domainuser.FriendshipDenied, true},
		{"cancel", constants.CMD_USER_FRIEND_CANCEL, "accepted", domainuser.FriendshipCancelled, true},
		{"remove", constants.CMD_USER_FRIEND_REMOVE, "accepted", domainuser.FriendshipRemoved, true},
		{"pending is not a decision", constants.CMD_USER_FRIEND_RESPOND, "pending", "", false},
		{"cancel is not a decision", constants.CMD_USER_FRIEND_RESPOND, "cancelled", "", false},
		{"unknown decision", constants.CMD_USER_FRIEND_RESPOND, "approved", "", false},
		{"unknown action", "unknown", "accepted", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &friendshipHandlerRepository{}
			response := performMultipartHandlerRequest(t, HandleFriendshipMutation(usecases.NewFriendshipService(repo), test.action), actor,
				map[string]string{"request_id": "100", "decision": test.decision}, nil)
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			}()
			if !test.valid {
				if response.StatusCode != http.StatusBadRequest || repo.changeCalls != 0 {
					t.Fatalf("invalid transition reached repository: status=%d calls=%d", response.StatusCode, repo.changeCalls)
				}
				return
			}
			if response.StatusCode != http.StatusOK || repo.changeCalls != 1 || repo.action != test.want {
				t.Fatalf("status=%d target=%s calls=%d; want %s", response.StatusCode, repo.action, repo.changeCalls, test.want)
			}
		})
	}
}
