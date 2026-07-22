package db

import (
	"core/models"
	"core/models/chat"
	chatpayloads "core/models/chat/payloads"
	"core/models/listings"
	"core/models/media"
	"core/models/notifications"
	"core/models/payment"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"core/models/taxonomy"
	modelutils "core/models/utils"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm/schema"
)

// TestPersistenceUUIDFieldsDeclarePostgresUUID prevents a model parsed in
// isolation from falling back to google/uuid's driver.Value (string) and thus
// presenting an existing PostgreSQL uuid column to AutoMigrate as text.
//
// Keep supplemental persistence models here too, even when the main migration
// does not create their table yet. Repository-local AutoMigrate calls and
// future migration registration must obey the same storage contract.
func TestPersistenceUUIDFieldsDeclarePostgresUUID(t *testing.T) {
	persistenceModels := []any{
		&models.CheckInTag{},
		&models.Engagement{},
		&models.EngagementDetail{},
		&models.Hashtag{},
		&models.Mention{},
		&models.Preferences{},
		&models.PrivatePhotoAccessRequest{},
		&models.Report{},
		&models.User{},
		&models.Wallet{},
		&chat.Chat{},
		&chat.ChatParticipant{},
		&chat.MessageView{},
		&chatpayloads.Call{},
		&listings.Listing{},
		&media.Media{},
		&notifications.Notification{},
		&payment.PaymentMethod{},
		&post.Post{},
		&postpayloads.Event{},
		&postpayloads.EventAttendee{},
		&postpayloads.Poll{},
		&postpayloads.PollChoice{},
		&postpayloads.PollVote{},
		&taxonomy.Cluster{},
		&taxonomy.ClusterEntity{},
		&taxonomy.ClusterIntent{},
		&taxonomy.Entity{},
		&taxonomy.Intent{},
		&taxonomy.Pillar{},
		&taxonomy.Synonym{},
		&modelutils.BaseContentable{},
		&modelutils.FileMetadata{},
		&modelutils.Location{},
	}

	uuidType := reflect.TypeOf(uuid.UUID{})
	for _, model := range persistenceModels {
		parsed, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			t.Fatalf("parse %T: %v", model, err)
		}
		for _, field := range parsed.Fields {
			if field.IndirectFieldType != uuidType || field.DBName == "" {
				continue
			}
			if !strings.EqualFold(field.TagSettings["TYPE"], "uuid") {
				t.Errorf("%s.%s (%s) has gorm type %q; want explicit uuid", parsed.Name, field.Name, field.DBName, field.TagSettings["TYPE"])
			}
		}
	}
}
