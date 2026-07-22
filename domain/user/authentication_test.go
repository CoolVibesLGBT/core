package user

import "testing"

func TestCanAuthenticateFailsClosed(t *testing.T) {
	tests := []struct {
		role  AccountRole
		bot   bool
		allow bool
	}{
		{role: AccountRoleUser, allow: true},
		{role: AccountRoleModerator, allow: true},
		{role: AccountRoleAdmin, allow: true},
		{role: AccountRoleSuperAdmin, allow: true},
		{role: AccountRoleVerified, allow: true},
		{role: AccountRoleUnverified, allow: true},
		{role: AccountRoleUser, bot: true},
		{role: "banned"},
		{role: "deleted"},
		{role: "pending"},
		{role: ""},
		{role: "future-unreviewed-role"},
	}

	for _, test := range tests {
		if got := CanAuthenticate(test.role, test.bot); got != test.allow {
			t.Fatalf("CanAuthenticate(%q, bot=%v) = %v, want %v", test.role, test.bot, got, test.allow)
		}
	}
}
