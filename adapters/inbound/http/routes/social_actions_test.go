package routes

import (
	"core/constants"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSocialActionsAreRegisteredAndRequireAuthentication(t *testing.T) {
	app := NewRouter(Dependencies{}).GetFiber()
	for _, action := range []string{constants.CMD_USER_FRIEND_REQUEST, constants.CMD_USER_FRIEND_RESPOND, constants.CMD_USER_FRIEND_CANCEL, constants.CMD_USER_FRIEND_REMOVE, constants.CMD_USER_FRIEND_FETCH, constants.CMD_USER_FRIEND_REQUESTS,
		constants.CMD_MATCH_UPDATE, constants.CMD_MATCH_FETCH, constants.CMD_MATCH_DELETE, constants.CMD_USER_FETCH_FOLLOWERS, constants.CMD_USER_FETCH_FOLLOWINGS} {
		request := httptest.NewRequest(http.MethodPost, "/packet", nil)
		request.Header.Set("X-Action", action)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("action %s unregistered or unprotected: %d", action, response.StatusCode)
		}
	}
}
