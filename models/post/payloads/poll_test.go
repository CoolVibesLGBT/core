package payloads

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestPollContentableIDRemainsUUIDInMigrations(t *testing.T) {
	parsed, err := schema.Parse(&Poll{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	field := parsed.LookUpField("ContentableID")
	if field == nil || field.DataType != schema.DataType("uuid") {
		t.Fatalf("Poll.ContentableID data type = %#v; want uuid", field)
	}
}
