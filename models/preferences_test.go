package models

import (
	"context"
	"core/models/utils"
	"testing"
)

func TestPreferencesDataGormValueReturnsJSONString(t *testing.T) {
	expr := (PreferencesData{
		Attributes: []PreferenceCategory{{Slug: utils.StringPtr("profile")}},
	}).GormValue(context.Background(), nil)
	if expr.SQL != "?" {
		t.Fatalf("GormValue() SQL = %q, want %q", expr.SQL, "?")
	}
	if len(expr.Vars) != 1 {
		t.Fatalf("GormValue() vars = %#v", expr.Vars)
	}
}

func TestPreferencesDataScanAcceptsString(t *testing.T) {
	var data PreferencesData
	if err := (&data).Scan(`{"attributes":[{"slug":"profile"}]}`); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if len(data.Attributes) != 1 || data.Attributes[0].Slug == nil || *data.Attributes[0].Slug != "profile" {
		t.Fatalf("Scan() decoded unexpected data: %#v", data)
	}
}
