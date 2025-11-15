package notifications

import (
	"coolvibes/models"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	NotificationTypeChatMessage   = "chat_message"   // Yeni mesaj bildirimi
	NotificationTypeNewMatch      = "new_match"      // Yeni eşleşme bildirimi
	NotificationTypeProfileVisit  = "profile_visit"  // Profil ziyareti bildirimi
	NotificationTypeFriendRequest = "friend_request" // Arkadaşlık isteği (isteğe bağlı)
	NotificationTypeEventReminder = "event_reminder" // Etkinlik hatırlatıcısı
	NotificationTypeSystemAlert   = "system_alert"   // Sistem uyarısı veya güncelleme

	NotificationTypeLike         = "like"          // Profil beğenisi bildirimi
	NotificationTypeFollow       = "follow"        // Takip bildirimi (eğer varsa)
	NotificationTypeSuperLike    = "super_like"    // Özel beğeni bildirimi (örn. Tinder’daki gibi)
	NotificationTypeMessageRead  = "message_read"  // Mesaj okundu bildirimi
	NotificationTypeMatchUnmatch = "match_unmatch" // Eşleşme iptali bildirimi
)

type Notification struct {
	ID        uuid.UUID           `gorm:"type:uuid;primaryKey" json:"id"`
	SenderID  *uuid.UUID          `gorm:"type:uuid;index" json:"sender_id,omitempty"`  // Gönderen kullanıcı ID'si
	Sender    *models.User        `gorm:"foreignKey:SenderID" json:"sender,omitempty"` // GORM relation
	UserID    uuid.UUID           `gorm:"type:uuid;index;not null" json:"user_id"`     // Bildirimin hedef kullanıcısı
	Type      string              `gorm:"size:64;index;not null" json:"type"`          // Tip: "chat_message", "friend_request" vb.
	Title     string              `gorm:"size:255" json:"title"`
	Message   string              `gorm:"type:text" json:"message"`
	Payload   NotificationPayload `gorm:"type:jsonb" json:"payload"` // JSONB olarak saklanacak
	IsRead    bool                `gorm:"default:false;index" json:"is_read"`
	IsShown   bool                `gorm:"default:false" json:"is_shown"`
	CreatedAt time.Time           `gorm:"autoCreateTime;index" json:"created_at"`
	ReadAt    *time.Time          `json:"read_at,omitempty"`
	ShownAt   *time.Time          `json:"shown_at,omitempty"`
	DeletedAt *time.Time          `gorm:"index" json:"deleted_at,omitempty"`
}

type NotificationPayload struct {
	Title              string   `json:"title"`
	Body               string   `json:"body"`
	Icon               string   `json:"icon,omitempty"`               // Küçük ikon URL'si
	Image              string   `json:"image,omitempty"`              // Daha büyük görsel URL'si
	Badge              string   `json:"badge,omitempty"`              // Küçük ikon (örneğin durum çubuğu için)
	Tag                string   `json:"tag,omitempty"`                // Bildirim gruplama için
	Color              string   `json:"color,omitempty"`              // Bildirim rengi (hex veya isim)
	Vibrate            []int    `json:"vibrate,omitempty"`            // Titreşim deseni (milisaniye cinsinden)
	Timestamp          int64    `json:"timestamp,omitempty"`          // Bildirim zamanı (unix timestamp)
	Actions            []Action `json:"actions,omitempty"`            // Bildirim butonları
	RequireInteraction bool     `json:"requireInteraction,omitempty"` // Bildirim kullanıcı kapatana kadar açık kalır
	Silent             bool     `json:"silent,omitempty"`             // Sessiz bildirim
	URL                string   `json:"url,omitempty"`                // Bildirim tıklanınca açılacak URL
	Data               any      `json:"data,omitempty"`               // Ek custom veri (mesaj ID vs)
}

type Action struct {
	Action string `json:"action"`         // Butonun adı, örn: "reply", "archive"
	Title  string `json:"title"`          // Butonun ekranda görünen yazısı
	Icon   string `json:"icon,omitempty"` // Buton ikonu (URL)
}

func (p *NotificationPayload) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan NotificationPayload: type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, p)
}

// Value implements the driver.Valuer interface
func (p NotificationPayload) Value() (driver.Value, error) {
	return json.Marshal(p)
}
