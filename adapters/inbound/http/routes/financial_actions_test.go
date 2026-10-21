package routes

import (
	"core/constants"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFinancialActionsRegisteredAndRequireAuthentication(t *testing.T) {
	app := NewRouter(Dependencies{}).GetFiber()
	for _, action := range []string{constants.CMD_BROADCAST_GIFT, constants.CMD_BROADCAST_SESSION_START, constants.CMD_BROADCAST_SESSION_HEARTBEAT, constants.CMD_BROADCAST_SESSION_END, constants.CMD_BROADCAST_SESSION_FETCH, constants.CMD_USER_REFERRAL_FETCH, constants.CMD_ADMIN_REFERRAL_FETCH, constants.CMD_ADMIN_REFERRAL_REVIEW} {
		request := httptest.NewRequest(http.MethodPost, "/packet", nil)
		request.Header.Set("X-Action", action)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("action %s status %d", action, response.StatusCode)
		}
	}
}
