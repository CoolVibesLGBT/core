package docs

import (
	"core/constants"
	"testing"
)

func TestFinancialActionsDocumentedWithJSON(t *testing.T) {
	wanted := map[string]bool{constants.CMD_BROADCAST_GIFT: false, constants.CMD_BROADCAST_SESSION_START: false, constants.CMD_BROADCAST_SESSION_HEARTBEAT: false, constants.CMD_BROADCAST_SESSION_END: false, constants.CMD_BROADCAST_SESSION_FETCH: false, constants.CMD_USER_REFERRAL_FETCH: false, constants.CMD_ADMIN_REFERRAL_FETCH: false, constants.CMD_ADMIN_REFERRAL_REVIEW: false}
	for _, doc := range ActionDocuments() {
		if _, ok := wanted[doc.Action]; ok {
			if doc.Auth != AuthRequired {
				t.Fatalf("financial action %s lacks authentication docs", doc.Action)
			}
			wanted[doc.Action] = true
		}
	}
	for action, found := range wanted {
		if !found || !actionSupportsJSON(action) {
			t.Fatalf("financial action %s missing documentation or JSON support", action)
		}
	}
}
