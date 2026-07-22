package types

import (
	"encoding/json"
	"strconv"
	"time"
)

// SnowflakeID keeps 64-bit public identifiers lossless for JavaScript clients.
// It belongs to the application response model rather than the persistence
// entity so list endpoints do not have to expose a full database record.
type SnowflakeID int64

func (id SnowflakeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(id), 10))
}

// NearbyUser is the intentionally narrow read model for nearby discovery.
// Sensitive persistence fields (balance, role, preferences, storage path,
// internal user UUID and account timestamps) never cross the application port.
type NearbyUser struct {
	PublicID    SnowflakeID      `json:"public_id"`
	UserName    string           `json:"username"`
	DisplayName string           `json:"displayname"`
	DateOfBirth *time.Time       `json:"date_of_birth,omitempty"`
	IsOnline    bool             `json:"is_online"`
	IsPremium   bool             `json:"is_premium"`
	Location    *NearbyLocation  `json:"location,omitempty"`
	Avatar      *NearbyUserMedia `json:"avatar,omitempty"`
}

type NearbyLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type NearbyUserMedia struct {
	PublicID SnowflakeID         `json:"public_id"`
	File     NearbyUserMediaFile `json:"file"`
}

type NearbyUserMediaFile struct {
	URL      string          `json:"url,omitempty"`
	Variants json.RawMessage `json:"variants,omitempty"`
}
