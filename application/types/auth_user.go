package types

// AuthUser is the allowlisted account projection returned after register,
// login and authenticated user-info checks. It deliberately reuses the safe
// profile fields while adding only the authenticated user's UI role. Token
// claims, database UUIDs, credentials, wallet state, preference bitsets,
// realtime metadata, subscriptions and storage paths are never represented.
type AuthUser struct {
	PublicUserProfile
	UserRole string `json:"user_role,omitempty"`
}
