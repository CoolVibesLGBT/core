package models

import (
	"core/constants"
	"core/models/utils"

	"github.com/google/uuid"
)

type CheckInTagType string

const (
	// ---- Capacity / Assets ----
	HasPlace   CheckInTagType = "has_place"
	HasVehicle CheckInTagType = "has_vehicle"
	OwnsHome   CheckInTagType = "owns_home"
	HasMoney   CheckInTagType = "has_money"

	// ---- Intent ----
	SeekingLove CheckInTagType = "seeking_love"
	SeekingFun  CheckInTagType = "seeking_fun"
	SeekingChat CheckInTagType = "seeking_chat"
	PaidMeeting CheckInTagType = "paid_meeting"

	// ---- Availability ----
	AvailableNow CheckInTagType = "available_now"
	NightOnly    CheckInTagType = "night_only"

	// ---- Hygiene ----
	CleanHygienic CheckInTagType = "clean_hygienic"

	// ---- Body / Appearance ----
	Hairless CheckInTagType = "hairless"
	Hairy    CheckInTagType = "hairy"
	Muscular CheckInTagType = "muscular"
	Chubby   CheckInTagType = "chubby"

	// ---- Personality ----
	Chill     CheckInTagType = "chill"
	FunPerson CheckInTagType = "fun"

	// ---- Safety / Boundaries ----
	Respectful CheckInTagType = "respectful"
	NoPressure CheckInTagType = "no_pressure"
)

type CheckInTag struct {
	ID        uuid.UUID             `gorm:"type:uuid;primaryKey" json:"id"`
	Tag       CheckInTagType        `gorm:"size:64;uniqueIndex" json:"tag"`
	Name      utils.LocalizedString `gorm:"type:jsonb" json:"name"`
	Icon      *string               `gorm:"size:255" json:"icon,omitempty"`
	IsVisible bool                  `json:"is_visible" gorm:"-"`
}

func GetAllCheckInTagTypes() []CheckInTag {
	return []CheckInTag{
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(HasPlace)),
			Tag:       HasPlace,
			Name:      utils.LocalizedString{"en": "Has Place", "tr": "Mekanım Var"},
			Icon:      utils.StringPtr("home"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(HasVehicle)),
			Tag:       HasVehicle,
			Name:      utils.LocalizedString{"en": "Has Vehicle", "tr": "Aracım Var"},
			Icon:      utils.StringPtr("car"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(OwnsHome)),
			Tag:       OwnsHome,
			Name:      utils.LocalizedString{"en": "Owns Home", "tr": "Evim Var"},
			Icon:      utils.StringPtr("building"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(HasMoney)),
			Tag:       HasMoney,
			Name:      utils.LocalizedString{"en": "Has Money", "tr": "Param Var"},
			Icon:      utils.StringPtr("wallet"),
			IsVisible: true,
		},

		// ---- Intent ----
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(SeekingLove)),
			Tag:       SeekingLove,
			Name:      utils.LocalizedString{"en": "Seeking Love", "tr": "Aşk Arıyorum"},
			Icon:      utils.StringPtr("heart"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(SeekingFun)),
			Tag:       SeekingFun,
			Name:      utils.LocalizedString{"en": "Seeking Fun", "tr": "Takılmak İstiyorum"},
			Icon:      utils.StringPtr("zap"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(SeekingChat)),
			Tag:       SeekingChat,
			Name:      utils.LocalizedString{"en": "Seeking Chat", "tr": "Sohbet Etmek İstiyorum"},
			Icon:      utils.StringPtr("message-circle"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(PaidMeeting)),
			Tag:       PaidMeeting,
			Name:      utils.LocalizedString{"en": "Paid Meeting", "tr": "Ücretli Görüşüyorum"},
			Icon:      utils.StringPtr("banknote"),
			IsVisible: true,
		},

		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(CleanHygienic)),
			Tag:       CleanHygienic,
			Name:      utils.LocalizedString{"en": "Clean & Hygienic", "tr": "Temiz ve Hijyenik"},
			Icon:      utils.StringPtr("sparkles"),
			IsVisible: true,
		},

		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(Hairless)),
			Tag:       Hairless,
			Name:      utils.LocalizedString{"en": "Hairless", "tr": "Kılsız / Tüysüz"},
			Icon:      utils.StringPtr("droplet"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(Hairy)),
			Tag:       Hairy,
			Name:      utils.LocalizedString{"en": "Hairy", "tr": "Kıllı / Tüylü"},
			Icon:      utils.StringPtr("feather"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(Muscular)),
			Tag:       Muscular,
			Name:      utils.LocalizedString{"en": "Muscular", "tr": "Kaslı"},
			Icon:      utils.StringPtr("dumbbell"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(Chubby)),
			Tag:       Chubby,
			Name:      utils.LocalizedString{"en": "Chubby", "tr": "Şişman"},
			Icon:      utils.StringPtr("flask-round"),
			IsVisible: true,
		},

		// ---- Availability ----
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(AvailableNow)),
			Tag:       AvailableNow,
			Name:      utils.LocalizedString{"en": "Available Now", "tr": "Şu An Müsaitim"},
			Icon:      utils.StringPtr("clock"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(NightOnly)),
			Tag:       NightOnly,
			Name:      utils.LocalizedString{"en": "Night Only", "tr": "Gece Uygunum"},
			Icon:      utils.StringPtr("moon"),
			IsVisible: true,
		},

		// ---- Personality ----
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(Chill)),
			Tag:       Chill,
			Name:      utils.LocalizedString{"en": "Chill", "tr": "Sakin"},
			Icon:      utils.StringPtr("coffee"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(FunPerson)),
			Tag:       FunPerson,
			Name:      utils.LocalizedString{"en": "Fun", "tr": "Eğlenceli"},
			Icon:      utils.StringPtr("smile"),
			IsVisible: true,
		},

		// ---- Safety ----
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(Respectful)),
			Tag:       Respectful,
			Name:      utils.LocalizedString{"en": "Respectful", "tr": "Saygılı"},
			Icon:      utils.StringPtr("shield-check"),
			IsVisible: true,
		},
		{
			ID:        uuid.NewSHA1(constants.NameSpace, []byte(NoPressure)),
			Tag:       NoPressure,
			Name:      utils.LocalizedString{"en": "No Pressure", "tr": "Israrcı Değil"},
			Icon:      utils.StringPtr("hand"),
			IsVisible: true,
		},
	}
}
