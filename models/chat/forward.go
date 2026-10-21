package chat

import (
	"encoding/json"
	"time"
)

// ForwardOrigin contains presentation data only: no source chat or private user identifiers.
type ForwardOrigin struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`
}

func ForwardOriginFromExtras(raw []byte) *ForwardOrigin {
	var extras struct {
		Origin *ForwardOrigin `json:"forwarded_from"`
	}
	if json.Unmarshal(raw, &extras) != nil || extras.Origin == nil || extras.Origin.Name == "" || extras.Origin.Date.IsZero() {
		return nil
	}
	return extras.Origin
}
