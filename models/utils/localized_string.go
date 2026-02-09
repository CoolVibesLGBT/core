package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type LocalizedString map[string]string // key: dil kodu (en, tr, es), value: içerik

func (ls *LocalizedString) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan LocalizedString: %v", value)
	}
	return json.Unmarshal(bytes, &ls)
}

func (ls LocalizedString) Value() (driver.Value, error) {
	return json.Marshal(ls)
}

func MakeLocalizedString(lang string, text string) *LocalizedString {
	if text == "" {
		return nil
	}
	ls := LocalizedString{lang: text}
	return &ls
}

func StringPtr(s string) *string {
	return &s
}

func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
func (ls LocalizedString) DefaultValue() string {
	if len(ls) == 0 {
		return ""
	}

	for _, v := range ls {
		if v != "" {
			return v
		}
	}
	return ""
}

func (ls LocalizedString) ToString() string {
	if len(ls) == 0 {
		return ""
	}

	var result string
	for _, v := range ls {
		if v != "" {
			result += v + " "
		}
	}

	return strings.ToLower(strings.TrimSpace(result))
}
