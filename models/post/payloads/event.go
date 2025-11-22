package payloads

import (
	"time"

	"coolvibes/models/utils"

	"github.com/google/uuid"
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

type EventKind struct {
	Kind         string                `gorm:"primaryKey;size:64" json:"kind"`
	DisplayOrder int                   `gorm:"default:0" json:"display_order"`
	Name         utils.LocalizedString `gorm:"type:jsonb" json:"name"`        // Çoklu dil destekli isim
	Description  utils.LocalizedString `gorm:"type:jsonb" json:"description"` // Çoklu dil destekli açıklama

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EventAttendee struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	EventID uuid.UUID `gorm:"type:uuid;not null;index" json:"event_id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`

	Status   string    `gorm:"size:32;default:'interested'" json:"status"` // "going", "interested", "invited", "declined"
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joined_at"`
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

	Status string `gorm:"size:32;default:'scheduled'" json:"status"`

	Attendees []EventAttendee `gorm:"foreignKey:EventID;constraint:OnDelete:CASCADE" json:"attendees,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (Event) TableName() string {
	return "events"
}

func (EventAttendee) TableName() string {
	return "event_attendees"
}
