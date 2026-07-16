package payloads

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseEventAttendanceStatusCanonicalAndLegacy(t *testing.T) {
	tests := map[string]EventAttendanceStatus{
		"going":      EventAttendanceGoing,
		"not_going":  EventAttendanceNotGoing,
		"maybe":      EventAttendanceMaybe,
		"interested": EventAttendanceMaybe,
		"declined":   EventAttendanceNotGoing,
	}
	for input, want := range tests {
		got, ok := ParseEventAttendanceStatus(input)
		if !ok || got != want {
			t.Fatalf("ParseEventAttendanceStatus(%q) = %q, %v; want %q, true", input, got, ok, want)
		}
	}
	if _, ok := ParseEventAttendanceStatus("invited"); ok {
		t.Fatal("invited must not be treated as an active RSVP")
	}
}

func TestEventAfterFindNormalizesDeduplicatesAndCountsAttendees(t *testing.T) {
	userA := uuid.New()
	userB := uuid.New()
	event := Event{
		Attendees: []EventAttendee{
			{ID: uuid.New(), UserID: userA, UserPublicID: 10, Status: EventAttendanceStatus("interested"), JoinedAt: time.Now()},
			{ID: uuid.New(), UserID: userA, UserPublicID: 10, Status: EventAttendanceGoing, JoinedAt: time.Now().Add(-time.Hour)},
			{ID: uuid.New(), UserID: userB, UserPublicID: 20, Status: EventAttendanceStatus("declined")},
			{ID: uuid.New(), UserID: uuid.New(), UserPublicID: 30, Status: EventAttendanceStatus("invited")},
		},
	}

	if err := event.AfterFind(nil); err != nil {
		t.Fatalf("AfterFind() error = %v", err)
	}
	if len(event.Attendees) != 2 {
		t.Fatalf("attendees = %#v; want two canonical unique RSVPs", event.Attendees)
	}
	if event.Attendees[0].Status != EventAttendanceMaybe || event.Attendees[1].Status != EventAttendanceNotGoing {
		t.Fatalf("normalized statuses = %#v", event.Attendees)
	}
	if event.GoingCount != 0 || event.MaybeCount != 1 || event.NotGoingCount != 1 {
		t.Fatalf("counts = %#v", event.AttendanceCounts())
	}
}

func TestEventAfterFindKeepsStoredCountsWhenAttendeesWereNotPreloaded(t *testing.T) {
	event := Event{GoingCount: 4, MaybeCount: 3, NotGoingCount: 2, Attendees: nil}
	if err := event.AfterFind(nil); err != nil {
		t.Fatalf("AfterFind() error = %v", err)
	}
	if event.GoingCount != 4 || event.MaybeCount != 3 || event.NotGoingCount != 2 {
		t.Fatalf("stored counts were reset without attendee preload: %#v", event.AttendanceCounts())
	}
}

func TestEventIsRSVPClosedAt(t *testing.T) {
	now := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Second)

	for _, test := range []struct {
		name   string
		event  Event
		closed bool
	}{
		{name: "scheduled future event", event: Event{Status: "scheduled", EndTime: &future}},
		{name: "past event", event: Event{Status: "scheduled", EndTime: &past}, closed: true},
		{name: "cancelled event", event: Event{Status: "cancelled", EndTime: &future}, closed: true},
		{name: "completed event", event: Event{Status: "COMPLETED"}, closed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.event.IsRSVPClosedAt(now); got != test.closed {
				t.Fatalf("IsRSVPClosedAt() = %v; want %v", got, test.closed)
			}
		})
	}
}

func TestEventAttendeeJSONUsesOnlyPublicIdentity(t *testing.T) {
	attendee := EventAttendee{
		ID:           uuid.New(),
		EventID:      uuid.New(),
		UserID:       uuid.New(),
		UserPublicID: 123,
		Username:     "safe-user",
		DisplayName:  "Safe User",
		Status:       EventAttendanceGoing,
	}
	payload, err := json.Marshal(attendee)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	jsonText := string(payload)
	if strings.Contains(jsonText, attendee.UserID.String()) || strings.Contains(jsonText, "user_id") {
		t.Fatalf("private user UUID leaked in attendee JSON: %s", jsonText)
	}
	for _, publicField := range []string{`"user_public_id":"123"`, `"username":"safe-user"`, `"displayname":"Safe User"`} {
		if !strings.Contains(jsonText, publicField) {
			t.Fatalf("public attendee field %s missing from JSON: %s", publicField, jsonText)
		}
	}
}
