package routes

import (
	"core/constants"
	"net/http/httptest"
	"testing"
)

func TestChatParticipantActionsRequireAuthentication(t *testing.T) {
	app := NewRouter(Dependencies{}).GetFiber()
	for _, action := range []string{constants.CMD_CHAT_PARTICIPANT_ADD, constants.CMD_CHAT_PARTICIPANT_REMOVE, constants.CMD_CHAT_PARTICIPANTS_FETCH} {
		request := httptest.NewRequest("POST", "/packet", nil)
		request.Header.Set("X-Action", action)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != 401 {
			t.Fatalf("%s unregistered or unprotected: %d", action, response.StatusCode)
		}
	}
}
