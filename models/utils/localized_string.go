package utils

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LocalizedString map[string]string

func (ls *LocalizedString) Scan(value interface{}) error {
	if err := ScanJSON(value, ls); err != nil {
		return fmt.Errorf("failed to scan LocalizedString: %w", err)
	}
	return nil
}

func (ls LocalizedString) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return JSONBGormValue(ctx, db, ls)
}

func (ls *LocalizedString) GetLocalizedString(lang string) string {
	if ls == nil {
		return ""
	}
	if val, ok := (*ls)[lang]; ok && val != "" {
		return val
	}
	return ls.DefaultValue()
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
