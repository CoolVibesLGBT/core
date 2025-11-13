package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
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
