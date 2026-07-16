package payloads

import (
	"errors"
	"strings"
	"time"

	"core/models/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// Pride, LGBTQ+ ve Topluluk Etkinlikleri
	EventKindPrideParade         = "pride_parade"         // Gurur Yürüyüşü
	EventKindLGBTQMeetup         = "lgbtq_meetup"         // LGBTQ+ buluşması
	EventKindSupportGroup        = "support_group"        // Destek grubu
	EventKindWorkshop            = "workshop"             // Atölye çalışması
	EventKindPanelDiscussion     = "panel_discussion"     // Panel tartışması
	EventKindWebinar             = "webinar"              // Online seminer
	EventKindFilmScreening       = "film_screening"       // Film gösterimi
	EventKindArtExhibition       = "art_exhibition"       // Sanat sergisi
	EventKindFundraiser          = "fundraiser"           // Bağış toplama etkinliği
	EventKindConference          = "conference"           // Konferans
	EventKindParty               = "party"                // Parti
	EventKindAwarenessCampaign   = "awareness_campaign"   // Farkındalık kampanyası
	EventKindVolunteering        = "volunteering"         // Gönüllülük etkinliği
	EventKindMarch               = "march"                // Yürüyüş, miting
	EventKindNetworkingEvent     = "networking_event"     // Ağ oluşturma etkinliği
	EventKindBookClub            = "book_club"            // Kitap kulübü
	EventKindYogaSession         = "yoga_session"         // Yoga seansı
	EventKindHealthCheckup       = "health_checkup"       // Sağlık taraması
	EventKindDragShow            = "drag_show"            // Drag şov
	EventKindMusicConcert        = "music_concert"        // Müzik konseri
	EventKindDanceParty          = "dance_party"          // Dans partisi
	EventKindFundingPitch        = "funding_pitch"        // Fonlama sunumu
	EventKindCommunityFestival   = "community_festival"   // Topluluk festivali
	EventKindTraining            = "training"             // Eğitim
	EventKindDebate              = "debate"               // Münazara
	EventKindMovieNight          = "movie_night"          // Film gecesi
	EventKindSupportCounseling   = "support_counseling"   // Destek danışmanlığı
	EventKindYouthOutreach       = "youth_outreach"       // Gençlere yönelik etkinlik
	EventKindMentalHealthForum   = "mental_health_forum"  // Ruh sağlığı forumu
	EventKindTransgenderRights   = "transgender_rights"   // Trans hakları etkinliği
	EventKindQueerHistoryTalk    = "queer_history_talk"   // Queer tarih konuşması
	EventKindAllyTraining        = "ally_training"        // Destekçi eğitimi
	EventKindFamilySupport       = "family_support"       // Aile destek etkinliği
	EventKindCulturalCelebration = "cultural_celebration" // Kültürel kutlama

	// HIV ve Cinsel Sağlık Odağı
	EventKindHIVAwareness          = "hiv_awareness"           // HIV farkındalık etkinliği
	EventKindHIVTesting            = "hiv_testing"             // HIV test etkinliği
	EventKindHIVSupportGroup       = "hiv_support_group"       // HIV destek grubu
	EventKindHIVPrevention         = "hiv_prevention"          // HIV önleme çalışması
	EventKindHIVTreatmentInfo      = "hiv_treatment_info"      // HIV tedavi bilgisi
	EventKindSexualHealthClinic    = "sexual_health_clinic"    // Cinsel sağlık kliniği
	EventKindSafeSexWorkshop       = "safe_sex_workshop"       // Güvenli seks atölyesi
	EventKindSTDAwareness          = "std_awareness"           // Cinsel yolla bulaşan hastalık farkındalığı
	EventKindPrEPInfoSession       = "prep_info_session"       // PrEP bilgilendirme
	EventKindPEPWorkshop           = "pep_workshop"            // PEP atölyesi
	EventKindNeedleExchange        = "needle_exchange"         // İğne değişim programı
	EventKindCounselingSession     = "counseling_session"      // Danışmanlık seansı
	EventKindHarmReduction         = "harm_reduction"          // Zarar azaltma etkinliği
	EventKindSexualViolenceSupport = "sexual_violence_support" // Cinsel şiddet destek etkinliği

	// Sosyal ve Toplumsal Etkinlikler
	EventKindCommunityMeetup       = "community_meetup"        // Topluluk buluşması
	EventKindOpenMicNight          = "open_mic_night"          // Açık mikrofon gecesi
	EventKindDragBrunch            = "drag_brunch"             // Drag brunch etkinliği
	EventKindKaraokeNight          = "karaoke_night"           // Karaoke gecesi
	EventKindFundraisingGala       = "fundraising_gala"        // Bağış galası
	EventKindArtWorkshop           = "art_workshop"            // Sanat atölyesi
	EventKindPoetryReading         = "poetry_reading"          // Şiir dinletisi
	EventKindLGBTQFilmFestival     = "lgbtq_film_festival"     // LGBTQ film festivali
	EventKindQueerDanceClass       = "queer_dance_class"       // Queer dans dersi
	EventKindMeditationSession     = "meditation_session"      // Meditasyon seansı
	EventKindSexEdClass            = "sex_ed_class"            // Cinsel eğitim sınıfı
	EventKindHealthFair            = "health_fair"             // Sağlık fuarı
	EventKindStorytellingNight     = "storytelling_night"      // Hikaye anlatımı gecesi
	EventKindQueerYouthCamp        = "queer_youth_camp"        // Queer genç kampı
	EventKindVolunteerMeetup       = "volunteer_meetup"        // Gönüllü buluşması
	EventKindSocialJusticeForum    = "social_justice_forum"    // Sosyal adalet forumu
	EventKindMentalWellnessRetreat = "mental_wellness_retreat" // Ruh sağlığı kampı
)

var (
	ErrEventNotFound   = errors.New("event not found")
	ErrEventClosed     = errors.New("event is not accepting RSVPs")
	ErrEventAtCapacity = errors.New("event is at capacity")
)

type EventKind struct {
	Kind         string                `gorm:"primaryKey;size:64" json:"kind"`
	DisplayOrder int                   `gorm:"default:0" json:"display_order"`
	Name         utils.LocalizedString `gorm:"type:jsonb" json:"name"`        // Çoklu dil destekli isim
	Description  utils.LocalizedString `gorm:"type:jsonb" json:"description"` // Çoklu dil destekli açıklama

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EventAttendanceStatus string

const (
	EventAttendanceGoing    EventAttendanceStatus = "going"
	EventAttendanceNotGoing EventAttendanceStatus = "not_going"
	EventAttendanceMaybe    EventAttendanceStatus = "maybe"
)

func ParseEventAttendanceStatus(value string) (EventAttendanceStatus, bool) {
	switch EventAttendanceStatus(value) {
	case EventAttendanceGoing:
		return EventAttendanceGoing, true
	case EventAttendanceNotGoing:
		return EventAttendanceNotGoing, true
	case EventAttendanceMaybe:
		return EventAttendanceMaybe, true
	// Backward compatibility for rows created against the original model.
	case "declined":
		return EventAttendanceNotGoing, true
	case "interested":
		return EventAttendanceMaybe, true
	default:
		return "", false
	}
}

type EventAttendanceCounts struct {
	Going    int64 `json:"going"`
	NotGoing int64 `json:"not_going"`
	Maybe    int64 `json:"maybe"`
}

type EventAttendee struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	EventID uuid.UUID `gorm:"type:uuid;not null;index" json:"event_id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`

	Status    EventAttendanceStatus `gorm:"size:32;default:'maybe'" json:"status"`
	JoinedAt  time.Time             `gorm:"autoCreateTime" json:"joined_at"`
	UpdatedAt time.Time             `gorm:"autoUpdateTime" json:"updated_at"`

	// Safe, read-only public projection populated by attendee preload queries.
	UserPublicID int64   `gorm:"column:user_public_id;->;-:migration" json:"user_public_id,string"`
	Username     string  `gorm:"column:username;->;-:migration" json:"username"`
	DisplayName  string  `gorm:"column:displayname;->;-:migration" json:"displayname"`
	AvatarURL    *string `gorm:"column:avatar_url;->;-:migration" json:"avatar_url,omitempty"`
}

type Event struct {
	ID          uuid.UUID             `gorm:"type:uuid;primaryKey" json:"id"`
	PostID      uuid.UUID             `gorm:"type:uuid;uniqueIndex;not null" json:"post_id"`
	Title       utils.LocalizedString `gorm:"type:jsonb" json:"title"`
	Description utils.LocalizedString `gorm:"type:jsonb" json:"description"`
	Kind        string                `gorm:"size:64;index" json:"kind"`
	StartTime   *time.Time            `json:"start_time,omitempty"`
	EndTime     *time.Time            `json:"end_time,omitempty"`
	Location    *utils.Location       `gorm:"polymorphic:Contentable;polymorphicValue:event;constraint:OnDelete:CASCADE" json:"location,omitempty"`

	Capacity  *int     `json:"capacity,omitempty"`
	IsPaid    bool     `gorm:"default:false" json:"is_paid"`
	Price     *float64 `json:"price,omitempty"`
	Currency  *string  `gorm:"size:8" json:"currency,omitempty"`
	IsOnline  bool     `gorm:"default:false" json:"is_online"`
	OnlineURL *string  `gorm:"size:255" json:"online_url,omitempty"`

	Status        string `gorm:"size:32;default:'scheduled'" json:"status"`
	GoingCount    int64  `gorm:"not null;default:0" json:"going_count"`
	NotGoingCount int64  `gorm:"not null;default:0" json:"not_going_count"`
	MaybeCount    int64  `gorm:"not null;default:0" json:"maybe_count"`

	Attendees []EventAttendee `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE" json:"attendees,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (e *Event) AttendanceCounts() EventAttendanceCounts {
	return EventAttendanceCounts{
		Going:    e.GoingCount,
		NotGoing: e.NotGoingCount,
		Maybe:    e.MaybeCount,
	}
}

func (e *Event) IsRSVPClosedAt(now time.Time) bool {
	switch strings.ToLower(strings.TrimSpace(e.Status)) {
	case "cancelled", "canceled", "completed":
		return true
	}
	return e.EndTime != nil && !e.EndTime.After(now)
}

func (e *Event) normalizeAttendance() {
	if e.Attendees == nil {
		return
	}

	seenUsers := make(map[uuid.UUID]struct{}, len(e.Attendees))
	normalized := make([]EventAttendee, 0, len(e.Attendees))
	counts := EventAttendanceCounts{}

	for _, attendee := range e.Attendees {
		status, ok := ParseEventAttendanceStatus(string(attendee.Status))
		if !ok {
			continue
		}
		// Preloads are newest-first. This also shields readers from historical
		// duplicate rows until the next RSVP write repairs them physically.
		if _, exists := seenUsers[attendee.UserID]; exists {
			continue
		}
		seenUsers[attendee.UserID] = struct{}{}

		attendee.Status = status
		normalized = append(normalized, attendee)
		switch status {
		case EventAttendanceGoing:
			counts.Going++
		case EventAttendanceNotGoing:
			counts.NotGoing++
		case EventAttendanceMaybe:
			counts.Maybe++
		}
	}

	e.Attendees = normalized
	e.GoingCount = counts.Going
	e.NotGoingCount = counts.NotGoing
	e.MaybeCount = counts.Maybe
}

func (e *Event) AfterFind(_ *gorm.DB) error {
	e.normalizeAttendance()
	return nil
}

type EventRSVPResult struct {
	Status    *EventAttendanceStatus `json:"status"`
	Attendees []EventAttendee        `json:"attendees"`
	Counts    EventAttendanceCounts  `json:"counts"`
}

func (Event) TableName() string {
	return "events"
}

func (EventAttendee) TableName() string {
	return "event_attendees"
}
