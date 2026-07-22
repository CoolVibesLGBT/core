package notifications

import (
	"context"
	"core/models"
	modelutils "core/models/utils"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	NotificationTypeChatMessage                = "chat_message"   // Yeni mesaj bildirimi
	NotificationTypeNewMatch                   = "new_match"      // Yeni eşleşme bildirimi
	NotificationTypeProfileVisit               = "profile_visit"  // Profil ziyareti bildirimi
	NotificationTypeFriendRequest              = "friend_request" // Arkadaşlık isteği (isteğe bağlı)
	NotificationTypeEventReminder              = "event_reminder" // Etkinlik hatırlatıcısı
	NotificationTypeSystemAlert                = "system_alert"   // Sistem uyarısı veya güncelleme
	NotificationTypeLike                       = "like"           // Profil beğenisi bildirimi
	NotificationTypeComment                    = "comment"        // Post yorum bildirimi
	NotificationTypeGiftReceived               = "gift"           // Profil beğenisi bildirimi
	NotificationTypeFollow                     = "follow"         // Takip bildirimi (eğer varsa)
	NotificationTypeUnFollow                   = "unfollow"       // Takip birakma bildirimi (eğer varsa)
	NotificationTypeSuperLike                  = "super_like"     // Özel beğeni bildirimi (örn. Tinder’daki gibi)
	NotificationTypeMessageRead                = "message_read"   // Mesaj okundu bildirimi
	NotificationTypeMatchUnmatch               = "match_unmatch"  // Eşleşme iptali bildirimi
	NotificationTypeReferral                   = "referral"       // Profil beğenisi bildirimi
	NotificationTypePrivatePhotoAccessRequest  = "private_photo_access_request"
	NotificationTypePrivatePhotoAccessResponse = "private_photo_access_response"
)

type Notification struct {
	ID        uuid.UUID           `gorm:"type:uuid;primaryKey" json:"id"`
	SenderID  *uuid.UUID          `gorm:"type:uuid;index" json:"sender_id,omitempty"`
	Sender    *models.User        `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	UserID    uuid.UUID           `gorm:"type:uuid;index;not null" json:"user_id"`
	Type      string              `gorm:"size:64;index;not null" json:"type"`
	Title     string              `gorm:"size:255" json:"title"`
	Message   string              `gorm:"type:text" json:"message"`
	Payload   NotificationPayload `gorm:"type:jsonb" json:"payload"`
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
	if err := modelutils.ScanJSON(value, p); err != nil {
		return errors.New("failed to scan NotificationPayload: " + err.Error())
	}
	return nil
}

func (p NotificationPayload) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return modelutils.JSONBGormValue(ctx, db, p)
}
