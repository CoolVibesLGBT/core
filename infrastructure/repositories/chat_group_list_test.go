package repositories

import (
	"context"
	"core/application/ports"
	"core/models/chat"
	modelutils "core/models/utils"
	"github.com/google/uuid"
	"testing"
)

func TestListPublicGroupsFiltersAndPaginates(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&chat.Chat{}, &chat.ChatParticipant{}); err != nil {
		t.Fatal(err)
	}
	owner, _ := privatePhotoTestUsers(t, db)
	repo := NewChatRepository(db, nil, nil, nil, nil)
	title := modelutils.MakeLocalizedString("en", "Discoverable community")
	for _, visibility := range []string{"public", "public", "private"} {
		entity := chat.Chat{ID: uuid.New(), Type: chat.ChatTypeGroup, CreatorID: owner.ID, Visibility: visibility, Title: title}
		if err := db.Create(&entity).Error; err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.ListPublicGroups(context.Background(), ports.PublicGroupQuery{Search: "discoverable", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Groups) != 1 || !first.HasMore || first.Groups[0].Visibility != "public" {
		t.Fatalf("unexpected page: %+v", first)
	}
	cursor := first.Groups[0].ID
	second, err := repo.ListPublicGroups(context.Background(), ports.PublicGroupQuery{Search: "discoverable", Limit: 1, Cursor: &cursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Groups) != 1 || second.HasMore || second.Groups[0].ID == cursor {
		t.Fatalf("unexpected second page: %+v", second)
	}
	empty, err := repo.ListPublicGroups(context.Background(), ports.PublicGroupQuery{Search: "no such group", Limit: 10})
	if err != nil || len(empty.Groups) != 0 {
		t.Fatalf("search = %+v, %v", empty, err)
	}
}
