package models

import (
	"coolvibes/models/utils"
	"time"
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
