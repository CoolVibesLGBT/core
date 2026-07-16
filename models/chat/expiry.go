package chat

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	MinExpiresInSeconds int64 = 10
	MaxExpiresInSeconds int64 = 7 * 24 * 60 * 60
)

// ParseExpiresInSeconds parses the optional expires_in_seconds form value.
// A missing, blank, or zero value represents a message that does not expire.
func ParseExpiresInSeconds(values []string) (*int, error) {
	if len(values) == 0 {
		return nil, nil
	}

	raw := strings.TrimSpace(values[0])
	if raw == "" || raw == "0" {
		return nil, nil
	}

	seconds, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("expires_in_seconds must be an integer between %d and %d", MinExpiresInSeconds, MaxExpiresInSeconds)
	}
	if seconds < MinExpiresInSeconds || seconds > MaxExpiresInSeconds {
		return nil, fmt.Errorf("expires_in_seconds must be 0 or between %d and %d", MinExpiresInSeconds, MaxExpiresInSeconds)
	}

	value := int(seconds)
	return &value, nil
}
