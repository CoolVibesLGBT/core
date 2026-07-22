package shared

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

func LookupString(arguments map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		raw, ok := arguments[key]
		if !ok || raw == nil {
			continue
		}

		switch value := raw.(type) {
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed, true
			}
		case json.Number:
			return value.String(), true
		case float64:
			if math.Trunc(value) == value {
				return strconv.FormatInt(int64(value), 10), true
			}
			return strconv.FormatFloat(value, 'f', -1, 64), true
		case float32:
			if math.Trunc(float64(value)) == float64(value) {
				return strconv.FormatInt(int64(value), 10), true
			}
			return strconv.FormatFloat(float64(value), 'f', -1, 64), true
		case int:
			return strconv.Itoa(value), true
		case int64:
			return strconv.FormatInt(value, 10), true
		case int32:
			return strconv.FormatInt(int64(value), 10), true
		}
	}

	return "", false
}

func LookupInt64(arguments map[string]any, keys ...string) (int64, bool) {
	for _, key := range keys {
		raw, ok := arguments[key]
		if !ok || raw == nil {
			continue
		}

		switch value := raw.(type) {
		case int:
			return int64(value), true
		case int64:
			return value, true
		case int32:
			return int64(value), true
		case float64:
			if math.Trunc(value) == value {
				return int64(value), true
			}
		case float32:
			if math.Trunc(float64(value)) == float64(value) {
				return int64(value), true
			}
		case json.Number:
			if parsed, err := value.Int64(); err == nil {
				return parsed, true
			}
		case string:
			parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err == nil {
				return parsed, true
			}
		}
	}

	return 0, false
}

func LookupFloat64(arguments map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		raw, ok := arguments[key]
		if !ok || raw == nil {
			continue
		}

		switch value := raw.(type) {
		case float64:
			return value, true
		case float32:
			return float64(value), true
		case int:
			return float64(value), true
		case int64:
			return float64(value), true
		case json.Number:
			if parsed, err := value.Float64(); err == nil {
				return parsed, true
			}
		case string:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err == nil {
				return parsed, true
			}
		}
	}

	return 0, false
}
