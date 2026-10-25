package chat

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// Telegram-style scheduled sends: at least a few seconds ahead and no more
// than a year out.
const (
	MinScheduleDelay = 10 * time.Second
	MaxScheduleDelay = 366 * 24 * time.Hour
)

var ErrInvalidSchedule = errors.New("scheduled_at must be between 10 seconds and one year from now")

// ParseScheduledAt reads the optional scheduled_at form value as Unix seconds
// or RFC 3339. A missing or blank value sends immediately.
func ParseScheduledAt(values []string, now time.Time) (*time.Time, error) {
	if len(values) == 0 || strings.TrimSpace(values[0]) == "" {
		return nil, nil
	}
	raw := strings.TrimSpace(values[0])
	var at time.Time
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		at = time.Unix(seconds, 0)
	} else if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		at = parsed
	} else {
		return nil, ErrInvalidSchedule
	}
	at = at.UTC()
	if at.Before(now.Add(MinScheduleDelay)) || at.After(now.Add(MaxScheduleDelay)) {
		return nil, ErrInvalidSchedule
	}
	return &at, nil
}
