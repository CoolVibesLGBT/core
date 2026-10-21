package repositories

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainmedia "core/domain/media"
	domainuser "core/domain/user"
	"core/models"
	"core/models/chat"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	modelutils "core/models/utils"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestGORMNullableLegacyBlockDoesNotHideUnrelatedUsersIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	aggregate := models.Engagement{ID: uuid.New(), ContentableID: viewer.ID, ContentableType: models.EngagementContentableTypeUser}
	if err := db.Create(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.EngagementDetail{}).Create(map[string]any{
		"id": uuid.New(), "engagement_id": aggregate.ID, "engager_id": viewer.ID, "engagee_id": nil, "kind": models.EngagementKindBlocking,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Omit(clause.Associations).Create(&models.PrivatePhotoAccessRequest{
		ID: uuid.New(), PublicID: 1, OwnerID: owner.ID, ViewerID: viewer.ID, Status: domainmedia.PrivatePhotoAccessApproved, RequestedAt: time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}
	allowed, err := (&PrivatePhotoRepository{db: db}).HasApprovedPrivatePhotoAccess(context.Background(), owner.ID, viewer.ID)
	if err != nil || !allowed {
		t.Fatalf("unrelated null target revoked access: %v %v", allowed, err)
	}
	users, err := NewMatchesRepository(db, nil).GetUnseenUsers(context.Background(), viewer.ID, 10)
	if err != nil || len(users) != 1 || int64(users[0].PublicID) != owner.PublicID {
		t.Fatalf("null target hid unrelated profile: %v %v", users, err)
	}
}

func TestGORMSubscriptionLookupKeepsSourceAndIDOnSameElementIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	user := financialUser(t, db)
	repo := &UserRepository{db: db}
	for _, value := range []string{`{"source":"provider","id":"123"}`, `[{"source":"provider","id":"123"}]`} {
		if err := db.Model(&user).UpdateColumn("subscriptions", datatypes.JSON(value)).Error; err != nil {
			t.Fatal(err)
		}
		got, err := repo.GetBySubscriptionSourceID("provider", "123")
		if err != nil || got.ID != user.ID {
			t.Fatalf("lookup %s: %v %v", value, got, err)
		}
	}
	for _, value := range []string{`[{"source":"provider","id":"other"},{"source":"other","id":"123"}]`, `null`, `[]`, `"provider"`} {
		if err := db.Model(&user).UpdateColumn("subscriptions", datatypes.JSON(value)).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := repo.GetBySubscriptionSourceID("provider", "123"); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("unrelated subscription matched %s: %v", value, err)
		}
	}
}

func TestGORMChatDeletionRemainsScopedToViewerAndAggregateIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&post.Post{}, &chat.Chat{}, &chat.ChatParticipant{}); err != nil {
		t.Fatal(err)
	}
	viewer, other := privatePhotoTestUsers(t, db)
	chatID, messageID := uuid.New(), uuid.New()
	contentType := string(post.PostKindChat)
	message := post.Post{ID: messageID, PublicID: 1, AuthorID: viewer.ID, PostKind: post.PostKindMessage, ContentableType: &contentType, ContentableID: &chatID}
	conversation := chat.Chat{ID: chatID, Type: chat.ChatType("private"), CreatorID: viewer.ID}
	participant := chat.ChatParticipant{ID: uuid.New(), ChatID: chatID, UserID: viewer.ID}
	for _, row := range []any{&message, &conversation, &participant} {
		if err := db.Omit(clause.Associations).Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := &ChatRepository{db: db}
	assertVisible := func(wantChats, wantMessages int64) {
		t.Helper()
		var chats, messages int64
		if err := repo.chatsByUserIDQuery(context.Background(), ports.ChatListQuery{UserID: viewer.ID}).Count(&chats).Error; err != nil {
			t.Fatal(err)
		}
		if err := repo.messagesByChatIDQuery(viewer.ID, chatID, nil).Count(&messages).Error; err != nil {
			t.Fatal(err)
		}
		if chats != wantChats || messages != wantMessages {
			t.Fatalf("visible chats/messages=%d/%d want %d/%d", chats, messages, wantChats, wantMessages)
		}
	}
	assertVisible(1, 1)
	aggregate := models.Engagement{ID: uuid.New(), ContentableID: messageID, ContentableType: models.EngagementContentableTypeMessage}
	if err := db.Create(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	detail := models.EngagementDetail{ID: uuid.New(), EngagementID: aggregate.ID, EngagerID: other.ID, Kind: models.EngagementKindMessageDeletedForMe}
	if err := db.Omit(clause.Associations).Create(&detail).Error; err != nil {
		t.Fatal(err)
	}
	assertVisible(1, 1)
	if err := db.Model(&detail).UpdateColumn("engager_id", viewer.ID).Error; err != nil {
		t.Fatal(err)
	}
	assertVisible(1, 0)
	if err := db.Model(&detail).UpdateColumn("kind", models.EngagementKindMessageDeletedForAll).Error; err != nil {
		t.Fatal(err)
	}
	if err := repairChatLastMessage(db, chatID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&conversation, "id = ?", chatID).Error; err != nil || conversation.LastMessageID != nil {
		t.Fatalf("deleted message retained as last: %v %v", conversation.LastMessageID, err)
	}
	if err := db.Model(&aggregate).Updates(map[string]any{"contentable_id": chatID, "contentable_type": models.EngagementContentableTypeChat}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&detail).UpdateColumn("kind", models.EngagementKindChatDeletedForAll).Error; err != nil {
		t.Fatal(err)
	}
	assertVisible(0, 1)
}

type gormSearchLocation struct {
	ID                                     uuid.UUID `gorm:"type:uuid;primaryKey"`
	ContentableID                          uuid.UUID `gorm:"type:uuid"`
	ContentableType                        string
	Address, City, Town, Province, Country string
	DeletedAt                              gorm.DeletedAt
}

func TestGORMPostSearchMatchesPayloadLocationAndHashtagIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&post.Post{}, &postpayloads.Event{}, &models.Hashtag{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Table("locations").AutoMigrate(&gormSearchLocation{}); err != nil {
		t.Fatal(err)
	}
	rows := make([]post.Post, 6)
	for i := range rows {
		rows[i] = post.Post{ID: uuid.New(), PublicID: int64(i + 1), PostKind: post.PostKindPost, Published: true}
	}
	if err := db.Omit(clause.Associations).Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&rows[0]).UpdateColumn("title", datatypes.JSON(`{"en":"needle"}`)).Error; err != nil {
		t.Fatal(err)
	}
	eventID := uuid.New()
	if err := db.Table("events").Create(map[string]any{"id": uuid.New(), "post_id": rows[1].ID, "title": datatypes.JSON(`{"en":"needle"}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("events").Create(map[string]any{"id": eventID, "post_id": rows[3].ID}).Error; err != nil {
		t.Fatal(err)
	}
	for _, location := range []gormSearchLocation{{ID: uuid.New(), ContentableID: rows[2].ID, ContentableType: "post", City: "needle"}, {ID: uuid.New(), ContentableID: eventID, ContentableType: "event", City: "needle"}} {
		if err := db.Table("locations").Create(&location).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.Hashtag{ID: uuid.New(), TaggableID: rows[4].ID, TaggableType: "post", Tag: "needle"}).Error; err != nil {
		t.Fatal(err)
	}
	var got []uuid.UUID
	if err := applyPostTextSearch(db.Model(&post.Post{}), "%needle%").Order("public_id ASC").Pluck("id", &got).Error; err != nil {
		t.Fatal(err)
	}
	want := make([]uuid.UUID, 5)
	for i := range want {
		want[i] = rows[i].ID
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("payload search=%v want %v", got, want)
	}
}

func TestGORMResourceLockSerializesFirstWritersAndRollsBackIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	user := financialUser(t, db)
	if err := db.Model(&user).UpdateColumn("balance", 0).Error; err != nil {
		t.Fatal(err)
	}
	const workers = 12
	start := make(chan struct{})
	errorsCh := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errorsCh <- db.Transaction(func(tx *gorm.DB) error {
				if err := lockTransactionResource(tx, "test:first-writers").Error; err != nil {
					return err
				}
				var current models.User
				if err := tx.Select("id", "balance").First(&current, "id = ?", user.ID).Error; err != nil {
					return err
				}
				return tx.Model(&models.User{}).Where("id = ?", user.ID).UpdateColumn("balance", current.Balance.Add(decimal.NewFromInt(1))).Error
			})
		}()
	}
	close(start)
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := db.First(&user, "id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !user.Balance.Equal(decimal.NewFromInt(workers)) {
		t.Fatalf("lost concurrent update: %s", user.Balance)
	}
	abort := errors.New("abort transaction")
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockTransactionResource(tx, "test:rollback").Error; err != nil {
			return err
		}
		return abort
	}); !errors.Is(err, abort) {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&models.TransactionLock{}).Where("key = ?", "test:rollback").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("rolled-back lock row: count=%d err=%v", count, err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error { return lockTransactionResource(tx, "test:rollback").Error }); err != nil {
		t.Fatal(err)
	}
}

func TestGORMMatchingRanksAcrossBatchesAndAppliesCriteriaIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	principal := models.User{Hobbies: pq.StringArray{"MUSIC"}, Languages: pq.StringArray{"en"}}
	candidates := make([]models.User, 205)
	ids := make([]uuid.UUID, len(candidates))
	for i := range candidates {
		ids[i] = uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1))
		candidates[i] = models.User{ID: ids[i], PublicID: int64(i + 1), UserRole: constants.UserRoleUser, Domain: models.CoolVibes}
	}
	for _, i := range []int{203, 204} {
		candidates[i].Hobbies = pq.StringArray{" music "}
		candidates[i].Languages = pq.StringArray{"EN"}
	}
	// Online breaks a score tie before public_id, even across batch boundaries.
	candidates[203].IsOnline = true
	if err := db.Omit(clause.Associations).Create(&candidates).Error; err != nil {
		t.Fatal(err)
	}
	query := func() *gorm.DB { return db.Model(&models.User{}).Where("id IN ?", ids) }
	got, err := rankMatchingUsers(query(), principal, domainuser.MatchCriteria{}, 2)
	if err != nil || !reflect.DeepEqual(got, []uuid.UUID{ids[203], ids[204]}) {
		t.Fatalf("global rank=%v err=%v", got, err)
	}
	birth := time.Now().UTC().AddDate(-25, 0, -1)
	if err := db.Model(&models.User{}).Where("id = ?", ids[204]).UpdateColumn("date_of_birth", birth).Error; err != nil {
		t.Fatal(err)
	}
	minAge, maxAge := 18, 26
	got, err = rankMatchingUsers(query(), principal, domainuser.MatchCriteria{MinAge: &minAge, MaxAge: &maxAge}, 10)
	if err != nil || !reflect.DeepEqual(got, []uuid.UUID{ids[204]}) {
		t.Fatalf("age filter=%v err=%v", got, err)
	}
	zero, one := 0.0, 1.0
	principal.Location = &modelutils.Location{Latitude: &zero, Longitude: &zero}
	for _, index := range []int{203, 204} {
		longitude := zero
		if index == 203 {
			longitude = one
		}
		if err := db.Table("locations").Create(map[string]any{"id": uuid.New(), "contentable_id": ids[index], "contentable_type": modelutils.LocationOwnerUser, "latitude": zero, "longitude": longitude}).Error; err != nil {
			t.Fatal(err)
		}
	}
	distance := 50.0
	got, err = rankMatchingUsers(query(), principal, domainuser.MatchCriteria{MaxDistanceKM: &distance}, 10)
	if err != nil || !reflect.DeepEqual(got, []uuid.UUID{ids[204]}) {
		t.Fatalf("distance filter=%v err=%v", got, err)
	}
}

func TestGORMPostTreePreservesVisibilityAndTerminatesCyclesIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&post.Post{}); err != nil {
		t.Fatal(err)
	}
	root, child, hidden, grandchild := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	rows := []post.Post{
		{ID: root, PublicID: 1, PostKind: post.PostKindPost, Published: true},
		{ID: child, PublicID: 2, ParentID: &root, PostKind: post.PostKindStatus, Published: true},
		{ID: hidden, PublicID: 3, ParentID: &root, PostKind: post.PostKindStatus, Published: false},
		{ID: grandchild, PublicID: 4, ParentID: &hidden, PostKind: post.PostKindStatus, Published: true},
	}
	if err := db.Omit(clause.Associations).Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	for _, all := range []bool{false, true} {
		got, err := postTreeIDs(db, db.Model(&post.Post{}).Where("id = ?", root), all)
		want := 2
		if all {
			want = 4
		}
		if err != nil || len(got) != want {
			t.Fatalf("visibility=%v tree=%v err=%v", all, got, err)
		}
	}
	if err := db.Model(&post.Post{}).Where("id = ?", root).UpdateColumn("parent_id", child).Error; err != nil {
		t.Fatal(err)
	}
	got, err := postTreeIDs(db, db.Model(&post.Post{}).Where("id = ?", root), false)
	if err != nil || len(got) != 2 {
		t.Fatalf("cyclic tree=%v err=%v", got, err)
	}
	if err := db.Delete(&post.Post{}, "id = ?", root).Error; err != nil {
		t.Fatal(err)
	}
	got, err = postTreeIDs(db, db.Model(&post.Post{}).Where("id = ?", root), true)
	if err != nil || len(got) != 0 {
		t.Fatalf("deleted root tree=%v err=%v", got, err)
	}
}
