package types

import "time"

// UserEngagement exposes the relationship and public identities, without
// account records, aggregate identifiers or internal engagement payloads.
type UserEngagement struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Engager   PublicUserSummary `json:"engager"`
	Engagee   PublicUserSummary `json:"engagee"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
