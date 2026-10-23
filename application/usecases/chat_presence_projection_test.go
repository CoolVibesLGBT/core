package usecases

import (
	"core/models"
	chatmodel "core/models/chat"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The conversation header needs the other participant's presence; the viewer's own
// presence and message authors stay out of the chat transport.
func TestChatProjectionExposesCounterpartPresenceOnly(t *testing.T) {
	seen := time.Date(2026, 10, 23, 9, 30, 0, 0, time.UTC)
	viewer := models.User{ID: uuid.New(), PublicID: 920000000000001, UserName: "viewer", IsOnline: true, LastOnline: &seen}
	offline := models.User{ID: uuid.New(), PublicID: 920000000000002, UserName: "offline", IsOnline: false, LastOnline: &seen}
	online := models.User{ID: uuid.New(), PublicID: 920000000000003, UserName: "online", IsOnline: true, LastOnline: &seen}
	entity := &chatmodel.Chat{ID: uuid.New(), Type: chatmodel.ChatTypeGroup, CreatorID: viewer.ID, Participants: []chatmodel.ChatParticipant{
		{UserID: viewer.ID, User: viewer}, {UserID: offline.ID, User: offline}, {UserID: online.ID, User: online},
	}}
	view := chatProjection(entity, viewer.ID)
	byName := map[string]int{}
	for index, participant := range view.Participants {
		byName[participant.User.UserName] = index
	}
	self := view.Participants[byName["viewer"]].User
	if self.IsOnline != nil || self.LastOnline != nil {
		t.Fatalf("viewer presence leaked: %+v", self)
	}
	away := view.Participants[byName["offline"]].User
	if away.IsOnline == nil || *away.IsOnline || away.LastOnline == nil || !away.LastOnline.Equal(seen) {
		t.Fatalf("offline participant presence missing: %+v", away)
	}
	here := view.Participants[byName["online"]].User
	if here.IsOnline == nil || !*here.IsOnline || here.LastOnline != nil {
		t.Fatalf("online participant should report online without a last-seen time: %+v", here)
	}
}
