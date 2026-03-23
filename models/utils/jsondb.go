package utils

import (
	"context"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MarshalJSONB(v any) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func JSONBGormValue(_ context.Context, db *gorm.DB, v any) clause.Expr {
	data, err := MarshalJSONB(v)
	if err != nil {
		return gorm.Expr("NULL")
	}

	if db != nil && db.Name() == "postgres" {
		return gorm.Expr("?::jsonb", data)
	}

	return gorm.Expr("?", data)
}

func ScanJSON(value any, dst any) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	case fmt.Stringer:
		bytes = []byte(v.String())
	default:
		return fmt.Errorf("cannot convert %T to JSON bytes", value)
	}

	return json.Unmarshal(bytes, dst)
}
