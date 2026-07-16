package notifications

import (
	"context"
	"testing"
)

func TestNotificationPayloadGormValueReturnsJSONString(t *testing.T) {
	expr := (NotificationPayload{Title: "hi", Body: "there"}).GormValue(context.Background(), nil)
	if expr.SQL != "?" {
		t.Fatalf("GormValue() SQL = %q, want %q", expr.SQL, "?")
	}
	if len(expr.Vars) != 1 {
		t.Fatalf("GormValue() vars = %#v", expr.Vars)
	}
}

func TestNotificationPayloadScanAcceptsString(t *testing.T) {
	var payload NotificationPayload
	if err := (&payload).Scan(`{"title":"hi","body":"there"}`); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if payload.Title != "hi" || payload.Body != "there" {
		t.Fatalf("Scan() decoded unexpected payload: %#v", payload)
	}
}
