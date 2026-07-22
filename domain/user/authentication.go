package user

// AccountRole is the domain-level account status/authorization role relevant
// to interactive authentication. Keeping the policy here prevents individual
// transports and token consumers from making different suspension decisions.
type AccountRole string

const (
	AccountRoleUser       AccountRole = "user"
	AccountRoleModerator  AccountRole = "moderator"
	AccountRoleAdmin      AccountRole = "admin"
	AccountRoleSuperAdmin AccountRole = "super_admin"
	AccountRoleVerified   AccountRole = "verified"
	AccountRoleUnverified AccountRole = "unverified"
)

// CanAuthenticate rejects non-human service accounts and fails closed for
// suspended or unknown roles. Existing tokens are subject to the same policy
// on every authenticated request.
func CanAuthenticate(role AccountRole, isBot bool) bool {
	if isBot {
		return false
	}
	switch role {
	case AccountRoleUser,
		AccountRoleModerator,
		AccountRoleAdmin,
		AccountRoleSuperAdmin,
		AccountRoleVerified,
		AccountRoleUnverified:
		return true
	default:
		return false
	}
}
