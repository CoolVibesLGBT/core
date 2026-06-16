package models

import (
	"core/models/utils"
	"time"

	"github.com/google/uuid"
)

const (
	ReportKindSpam                   = "spam"
	ReportKindHateSpeech             = "hate_speech"
	ReportKindNudity                 = "nudity"
	ReportKindViolenceThreat         = "violence_threat"
	ReportKindFraud                  = "fraud"
	ReportKindHarassment             = "harassment"
	ReportKindPersonalInfo           = "personal_info"
	ReportKindFalseInfo              = "false_info"
	ReportKindProfanity              = "profanity"
	ReportKindSelfHarm               = "self_harm"
	ReportKindCopyrightInfringement  = "copyright_infringement"
	ReportKindDrugUse                = "drug_use"
	ReportKindTerrorism              = "terrorism"
	ReportKindPoliticalContent       = "political_content"
	ReportKindMisleadingAdvertising  = "misleading_advertising"
	ReportKindSecurityVulnerability  = "security_vulnerability"
	ReportKindFakeProfile            = "fake_profile"
	ReportKindUnderage               = "underage"
	ReportKindImpersonation          = "impersonation"
	ReportKindNonConsensualContent   = "non_consensual_content"
	ReportKindSexualHarassment       = "sexual_harassment"
	ReportKindSolicitation           = "solicitation"
	ReportKindSelfPromotion          = "self_promotion"
	ReportKindGraphicViolence        = "graphic_violence"
	ReportKindDiscriminatoryLanguage = "discriminatory_language"
	ReportKindMalwarePhishing        = "malware_phishing"
	ReportKindInappropriateUsername  = "inappropriate_username"
	ReportKindSelfHarmPromotion      = "self_harm_promotion"
	ReportKindThreatsBullying        = "threats_bullying"
	ReportKindPrivacyViolation       = "privacy_violation"
	ReportKindFakeNews               = "fake_news"
	ReportKindReligiousHateSpeech    = "religious_hate_speech"
	ReportKindPoliticalExtremism     = "political_extremism"
	ReportKindCulturalInsensitivity  = "cultural_insensitivity"
	ReportKindIllegalActivities      = "illegal_activities"
	ReportKindCopyrightViolation     = "copyright_violation"
	ReportKindOther                  = "other"
)

type ReportKind struct {
	Key          string                `gorm:"primaryKey;size:64" json:"key"`
	DisplayOrder int                   `gorm:"default:0" json:"display_order"`
	Name         utils.LocalizedString `gorm:"type:jsonb" json:"name"`
	Description  utils.LocalizedString `gorm:"type:jsonb" json:"description"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

type ReportStatus string

const (
	ReportStatusPending  ReportStatus = "pending"
	ReportStatusReviewed ReportStatus = "reviewed"
	ReportStatusRejected ReportStatus = "rejected"
	ReportStatusActioned ReportStatus = "actioned"
)

func (s ReportStatus) IsValid() bool {
	switch s {
	case ReportStatusPending,
		ReportStatusReviewed,
		ReportStatusRejected,
		ReportStatusActioned:
		return true
	default:
		return false
	}
}

type Report struct {
	ID              uuid.UUID                 `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ContentableID   uuid.UUID                 `gorm:"type:uuid;not null;index" json:"contentable_id"`
	ContentableType EngagementContentableType `gorm:"type:varchar(50);not null;index" json:"contentable_type"`

	ReporterID uuid.UUID `gorm:"type:uuid;not null;index" json:"reporter_id"`
	Reporter   *User     `gorm:"foreignKey:ReporterID;references:ID" json:"reporter,omitempty"`

	ReportKindKey string      `gorm:"type:varchar(64);not null;index" json:"report_kind_key"` // Rapor türü
	ReportKind    *ReportKind `gorm:"foreignKey:ReportKindKey;references:Key" json:"report_kind,omitempty"`
	Reason        string      `gorm:"type:text" json:"reason"` // Kullanıcı açıklaması

	Status       ReportStatus `gorm:"type:varchar(32);not null;default:'pending';index" json:"status"`
	ReviewedByID *uuid.UUID   `gorm:"type:uuid;index" json:"reviewed_by_id,omitempty"`
	ReviewedBy   *User        `gorm:"foreignKey:ReviewedByID;references:ID" json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time   `json:"reviewed_at,omitempty"`
	Resolution   string       `gorm:"type:text" json:"resolution,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
