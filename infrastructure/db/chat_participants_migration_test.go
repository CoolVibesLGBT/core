package db

import (
	"core/models/chat"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestMigrateChatParticipantsRepairsDuplicatesAndInstallsUniqueIndex(t *testing.T) {
	database := migrationIntegrationDB(t)
	if err := database.AutoMigrate(&chat.ChatParticipant{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrator().DropIndex(&chat.ChatParticipant{}, "idx_chat_participants_chat_user"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	chatID, userID := uuid.New(), uuid.New()
	members := []chat.ChatParticipant{
		{ID: uuid.New(), ChatID: chatID, UserID: userID, UpdatedAt: now.Add(-time.Hour), ClearedAt: &now},
		{ID: uuid.Nil, ChatID: chatID, UserID: userID, UpdatedAt: now, CreatedAt: now, LeftAt: &now, Role: chat.ParticipantRoleAdmin},
	}
	if err := database.Create(&members).Error; err != nil {
		t.Fatal(err)
	}
	if err := MigrateChatParticipants(database); err != nil {
		t.Fatal(err)
	}
	if err := MigrateChatParticipants(database); err != nil {
		t.Fatal(err)
	}
	var result []chat.ChatParticipant
	if err := database.Find(&result).Error; err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].ID == uuid.Nil || result[0].LeftAt == nil || result[0].ClearedAt == nil || result[0].JoinedAt.IsZero() || result[0].Role != chat.ParticipantRoleAdmin {
		t.Fatalf("migration lost membership state: %+v", result)
	}
	// A nested savepoint keeps the expected unique violation from aborting the fixture.
	if err := database.SavePoint("duplicate_member").Error; err != nil {
		t.Fatal(err)
	}
	duplicate := result[0]
	duplicate.ID = uuid.New()
	if err := database.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate accepted after migration")
	}
	if err := database.RollbackTo("duplicate_member").Error; err != nil {
		t.Fatal(err)
	}
}
