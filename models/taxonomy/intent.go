package taxonomy

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IntentKey string

const (
	IntentInformational IntentKey = "informational"
	IntentNavigational  IntentKey = "navigational"
	IntentTransactional IntentKey = "transactional"
	IntentCommercial    IntentKey = "commercial"
	IntentNews          IntentKey = "news"
	IntentAnalysis      IntentKey = "analysis"
	IntentGuide         IntentKey = "guide"
)

type Intent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Key       IntentKey `gorm:"type:varchar(40);not null;uniqueIndex" json:"key"`
	Label     string    `gorm:"size:80;not null" json:"label"`
	IsActive  bool      `gorm:"default:true;index" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (i *Intent) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	i.Label = strings.TrimSpace(i.Label)
	return nil
}
