package helpers

import (
	"encoding/json"
	"fmt"
	"strings"
)

func DecodeJSONItems(raw []byte) ([]json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, nil
	}

	switch trimmed[0] {
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
		return items, nil
	case '{':
		return []json.RawMessage{append(json.RawMessage(nil), raw...)}, nil
	default:
		return nil, fmt.Errorf("unsupported json payload")
	}
}
