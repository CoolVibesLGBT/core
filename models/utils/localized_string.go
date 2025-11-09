package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type LocalizedString map[string]string // key: dil kodu (en, tr, es), value: içerik

// Scanner implements sql.Scanner
func (ls *LocalizedString) Scan(value interface{}) error {
	if value == nil {
		*ls = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan LocalizedString: %v", value)
	}
	return json.Unmarshal(bytes, ls)
}

// Value implements driver.Valuer
func (ls LocalizedString) Value() (driver.Value, error) {
	if ls == nil {
		return nil, nil
	}
	return json.Marshal(ls)
}

func MakeLocalizedString(lang string, text string) *LocalizedString {
	if text == "" {
		return nil
	}
	ls := LocalizedString{lang: text}
	return &ls
}
