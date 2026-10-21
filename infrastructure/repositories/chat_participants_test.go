package repositories

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"core/models/chat"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func groupParticipantFixture(t *testing.T) (*ChatRepository, models.User, models.User, models.User, *chat.Chat) {
	t.Helper()
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&chat.Chat{}, &chat.ChatParticipant{}); err != nil {
		t.Fatal(err)
	}
	owner, member := privatePhotoTestUsers(t, db)
	outsider := models.User{ID: uuid.New(), PublicID: 30, UserName: "outsider", UserRole: constants.UserRoleUser}
	if err := db.Omit(clause.Associations).Create(&outsider).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewChatRepository(db, nil, nil, nil, nil)
	entity, err := repo.CreateGroupChat(context.Background(), owner.ID, []uuid.UUID{member.ID, member.ID, owner.ID}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return repo, owner, member, outsider, entity
}

func TestCreateGroupChatParticipantLifecycleIntegration(t *testing.T) {
	repo, owner, member, outsider, entity := groupParticipantFixture(t)
	ctx := context.Background()
	if len(entity.Participants) != 2 {
		t.Fatalf("participants=%d", len(entity.Participants))
	}
	for _, p := range entity.Participants {
		if p.ID == uuid.Nil || p.JoinedAt.IsZero() || p.ChatID != entity.ID {
			t.Fatalf("invalid participant: %+v", p)
		}
		if p.UserID == owner.ID && p.Role != chat.ParticipantRoleOwner {
			t.Fatal("creator is not owner")
		}
	}
	if err := repo.AddParticipant(ctx, &member, entity.ID, outsider.ID); !errors.Is(err, chat.ErrPermissionDenied) {
		t.Fatalf("member added outsider: %v", err)
	}
	if err := repo.AddParticipant(ctx, &outsider, entity.ID, outsider.ID); !errors.Is(err, chat.ErrNotParticipant) {
		t.Fatalf("outsider joined: %v", err)
	}
	if _, err := repo.GetParticipants(ctx, &outsider, entity.ID); !errors.Is(err, chat.ErrNotParticipant) {
		t.Fatalf("outsider read participants: %v", err)
	}
	if err := repo.RemoveParticipant(ctx, &member, entity.ID, owner.ID); !errors.Is(err, chat.ErrPermissionDenied) {
		t.Fatalf("member removed owner: %v", err)
	}
	if err := repo.RemoveParticipant(ctx, &owner, entity.ID, owner.ID); !errors.Is(err, constants.ErrGroupOwnerCannotLeave) {
		t.Fatalf("owner orphaned group: %v", err)
	}
	if err := repo.AddParticipant(ctx, &owner, entity.ID, outsider.ID); err != nil {
		t.Fatal(err)
	}
	var original chat.ChatParticipant
	if err := repo.db.Where("chat_id = ? AND user_id = ?", entity.ID, outsider.ID).Take(&original).Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); failures <- repo.AddParticipant(ctx, &owner, entity.ID, outsider.ID) }()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	repo.db.Model(&chat.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", entity.ID, outsider.ID).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate memberships: %d", count)
	}
	duplicate := original
	duplicate.ID = uuid.New()
	if err := repo.db.Omit(clause.Associations).Create(&duplicate).Error; err == nil {
		t.Fatal("DB accepted duplicate membership")
	}
	if err := repo.RemoveParticipant(ctx, &outsider, entity.ID, outsider.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.SendTypingEvent(entity.ID, outsider.ID, true); !errors.Is(err, chat.ErrNotParticipant) {
		t.Fatalf("left member retained access: %v", err)
	}
	participants, err := repo.GetParticipants(ctx, &owner, entity.ID)
	if err != nil || len(participants) != 2 {
		t.Fatalf("participants after leave=%d, %v", len(participants), err)
	}
	if err := repo.AddParticipant(ctx, &owner, entity.ID, outsider.ID); err != nil {
		t.Fatal(err)
	}
	var rejoined chat.ChatParticipant
	if err := repo.db.Where("chat_id = ? AND user_id = ?", entity.ID, outsider.ID).Take(&rejoined).Error; err != nil {
		t.Fatal(err)
	}
	if rejoined.ID != original.ID || rejoined.LeftAt != nil || rejoined.JoinedAt.Before(original.JoinedAt) {
		t.Fatalf("bad rejoin: %+v", rejoined)
	}
	if err := repo.SendTypingEvent(entity.ID, outsider.ID, true); err != nil {
		t.Fatal(err)
	}
	page, err := repo.ListChats(ctx, ports.ChatListQuery{UserID: outsider.ID, Limit: 20})
	if err != nil || len(page.Chats) != 1 || page.Chats[0].ID != entity.ID || len(page.Chats[0].Participants) != 3 {
		t.Fatalf("rejoined group missing from chat list: %+v, %v", page, err)
	}
	for _, member := range page.Chats[0].Participants {
		if member.User.PublicID <= 0 {
			t.Fatal("participant identity was not loaded")
		}
	}
	if err := repo.RemoveParticipant(ctx, &owner, entity.ID, outsider.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveParticipant(ctx, &member, entity.ID, member.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveParticipant(ctx, &owner, entity.ID, owner.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.db.First(&chat.Chat{}, "id = ?", entity.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("empty group not closed: %v", err)
	}
}

func TestGroupParticipantsRejectUnavailableUsersAndDeletedChatsIntegration(t *testing.T) {
	repo, owner, member, outsider, entity := groupParticipantFixture(t)
	ctx := context.Background()
	if err := repo.db.Model(&outsider).Update("domain", models.DomainKind("other")).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.AddParticipant(ctx, &owner, entity.ID, outsider.ID); !errors.Is(err, domainuser.ErrInteractionUnavailable) {
		t.Fatalf("cross domain member added: %v", err)
	}
	if _, err := repo.CreateGroupChat(ctx, owner.ID, []uuid.UUID{member.ID, outsider.ID}, nil, nil); !errors.Is(err, domainuser.ErrInteractionUnavailable) {
		t.Fatalf("cross domain group: %v", err)
	}
	var groups int64
	repo.db.Model(&chat.Chat{}).Count(&groups)
	if groups != 1 {
		t.Fatalf("partial group persisted: %d", groups)
	}
	if err := repo.db.Model(&outsider).Update("domain", models.CoolVibes).Error; err != nil {
		t.Fatal(err)
	}
	// Apply an existing social block rather than inventing a group-only policy.
	if err := repo.db.Transaction(func(tx *gorm.DB) error {
		aggregate := models.Engagement{ID: uuid.New(), ContentableID: owner.ID, ContentableType: models.EngagementContentableTypeUser}
		if err := tx.Create(&aggregate).Error; err != nil {
			return err
		}
		return tx.Create(&models.EngagementDetail{ID: uuid.New(), EngagementID: aggregate.ID, EngagerID: owner.ID, EngageeID: outsider.ID, Kind: models.EngagementKindBlocking, CreatedAt: time.Now()}).Error
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddParticipant(ctx, &owner, entity.ID, outsider.ID); !errors.Is(err, domainuser.ErrInteractionBlocked) {
		t.Fatalf("blocked member added: %v", err)
	}
	private, err := repo.CreatePrivateChat(owner.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AddParticipant(ctx, &owner, private.ID, outsider.ID); !errors.Is(err, constants.ErrUnsupportedChatType) {
		t.Fatalf("private chat became group: %v", err)
	}
	if err := repo.DeleteChatForAll(ctx, &owner, entity.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddParticipant(ctx, &owner, entity.ID, outsider.ID); !errors.Is(err, chat.ErrChatNotFound) {
		t.Fatalf("deleted group accepted member: %v", err)
	}
}
