package models

import (
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
			Tag:       HasPlace,
			Name:      utils.LocalizedString{"en": "Has Place", "tr": "Mekanı Var"},
			Icon:      utils.StringPtr("home-outline"),
			IsVisible: true,
		},
		{
			Tag:       HasVehicle,
			Name:      utils.LocalizedString{"en": "Has Vehicle", "tr": "Aracı Var"},
			Icon:      utils.StringPtr("car-outline"),
			IsVisible: true,
		},
		{
			Tag:       OwnsHome,
			Name:      utils.LocalizedString{"en": "Owns Home", "tr": "Evi Var"},
			Icon:      utils.StringPtr("office-building"),
			IsVisible: true,
		},
		{
			Tag:       HasMoney,
			Name:      utils.LocalizedString{"en": "Has Money", "tr": "Parası Var"},
			Icon:      utils.StringPtr("wallet-outline"),
			IsVisible: true,
		},

		// ---- Intent ----
		{
			Tag:       SeekingLove,
			Name:      utils.LocalizedString{"en": "Seeking Love", "tr": "Aşk Arıyor"},
			Icon:      utils.StringPtr("heart-outline"),
			IsVisible: true,
		},
		{
			Tag:       SeekingFun,
			Name:      utils.LocalizedString{"en": "Seeking Fun", "tr": "Takılmak İstiyor"},
			Icon:      utils.StringPtr("flash-outline"),
			IsVisible: true,
		},
		{
			Tag:       SeekingChat,
			Name:      utils.LocalizedString{"en": "Seeking Chat", "tr": "Sohbet Etmek İstiyor"},
			Icon:      utils.StringPtr("message-text-outline"),
			IsVisible: true,
		},
		{
			Tag:       PaidMeeting,
			Name:      utils.LocalizedString{"en": "Paid Meeting", "tr": "Ücretli Görüşüyor"},
			Icon:      utils.StringPtr("cash-multiple"),
			IsVisible: true,
		},

		// ---- Availability ----
		{
			Tag:       AvailableNow,
			Name:      utils.LocalizedString{"en": "Available Now", "tr": "Şu An Müsait"},
			Icon:      utils.StringPtr("clock-check-outline"),
			IsVisible: true,
		},
		{
			Tag:       NightOnly,
			Name:      utils.LocalizedString{"en": "Night Only", "tr": "Gece Uygun"},
			Icon:      utils.StringPtr("moon-waning-crescent"),
			IsVisible: true,
		},

		// ---- Personality ----
		{
			Tag:       Chill,
			Name:      utils.LocalizedString{"en": "Chill", "tr": "Sakin"},
			Icon:      utils.StringPtr("coffee-outline"),
			IsVisible: true,
		},
		{
			Tag:       FunPerson,
			Name:      utils.LocalizedString{"en": "Fun", "tr": "Eğlenceli"},
			Icon:      utils.StringPtr("emoticon-happy-outline"),
			IsVisible: true,
		},

		// ---- Safety ----
		{
			Tag:       Respectful,
			Name:      utils.LocalizedString{"en": "Respectful", "tr": "Saygılı"},
			Icon:      utils.StringPtr("shield-check-outline"),
			IsVisible: true,
		},
		{
			Tag:       NoPressure,
			Name:      utils.LocalizedString{"en": "No Pressure", "tr": "Israrcı Değil"},
			Icon:      utils.StringPtr("hand-back-left-outline"),
			IsVisible: true,
		},
	}
}
