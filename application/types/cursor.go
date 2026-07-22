package types

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const maxCursorLength = 2048

var ErrCursorEncode = errors.New("failed to encode pagination cursor")

const (
	CursorKeyPublicID  = "public_id"
	CursorKeyDistance  = "distance"
	CursorKeyCreatedAt = "created_at"
	CursorKeyUUID      = "id"
)

type Cursor struct {
	Prev     *string  `json:"prev,omitempty"`
	Next     *string  `json:"next,omitempty"`
	Distance *float64 `json:"distance,omitempty"`
}

func NewPaginationCursor(values map[string]any) (*string, error) {
	data, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCursorEncode, err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	if len(encoded) > maxCursorLength {
		return nil, fmt.Errorf("%w: token exceeds %d bytes", ErrCursorEncode, maxCursorLength)
	}
	return &encoded, nil
}

func NewPublicIDCursor(publicID int64) (*string, error) {
	return NewPaginationCursor(map[string]any{
		CursorKeyPublicID: strconv.FormatInt(publicID, 10),
	})
}

func NewPublicIDDistanceCursor(publicID int64, distance *float64) (*string, error) {
	values := map[string]any{
		CursorKeyPublicID: strconv.FormatInt(publicID, 10),
	}
	if distance != nil {
		values[CursorKeyDistance] = *distance
	}
	return NewPaginationCursor(values)
}

func NewTimeCursor(createdAt time.Time) (*string, error) {
	return NewPaginationCursor(map[string]any{
		CursorKeyCreatedAt: createdAt.UTC().Format(time.RFC3339Nano),
	})
}

func NewTimeUUIDCursor(createdAt time.Time, id uuid.UUID) (*string, error) {
	return NewPaginationCursor(map[string]any{
		CursorKeyCreatedAt: createdAt.UTC().Format(time.RFC3339Nano),
		CursorKeyUUID:      id.String(),
	})
}

func DecodePaginationCursor(raw string) (map[string]any, bool) {
	if raw == "" || len(raw) > maxCursorLength {
		return nil, false
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, false
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil || values == nil {
		return nil, false
	}
	return values, true
}

func CursorPublicID(values map[string]any) (int64, bool) {
	return cursorInt64(values, CursorKeyPublicID)
}

func CursorDistance(values map[string]any) (float64, bool) {
	value, ok := values[CursorKeyDistance]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func CursorCreatedAt(values map[string]any) (time.Time, bool) {
	value, ok := values[CursorKeyCreatedAt]
	if !ok {
		return time.Time{}, false
	}
	switch v := value.(type) {
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, v)
		}
		return parsed, err == nil
	default:
		return time.Time{}, false
	}
}

func CursorUUID(values map[string]any) (uuid.UUID, bool) {
	value, ok := values[CursorKeyUUID]
	if !ok {
		return uuid.Nil, false
	}
	raw, ok := value.(string)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	return id, err == nil
}

func cursorInt64(values map[string]any, key string) (int64, bool) {
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		i, err := v.Int64()
		return i, err == nil
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}
