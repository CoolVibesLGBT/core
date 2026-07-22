package models

import (
	domainmoderation "core/domain/moderation"
	"core/models/utils"
	"time"

	"github.com/google/uuid"
)

const (
	ReportKindSpam                   = string(domainmoderation.KindSpam)
	ReportKindHateSpeech             = string(domainmoderation.KindHateSpeech)
	ReportKindNudity                 = string(domainmoderation.KindNudity)
	ReportKindViolenceThreat         = string(domainmoderation.KindViolenceThreat)
	ReportKindFraud                  = string(domainmoderation.KindFraud)
	ReportKindHarassment             = string(domainmoderation.KindHarassment)
	ReportKindPersonalInfo           = string(domainmoderation.KindPersonalInfo)
	ReportKindFalseInfo              = string(domainmoderation.KindFalseInfo)
	ReportKindProfanity              = string(domainmoderation.KindProfanity)
	ReportKindSelfHarm               = string(domainmoderation.KindSelfHarm)
	ReportKindCopyrightInfringement  = string(domainmoderation.KindCopyrightInfringement)
	ReportKindDrugUse                = string(domainmoderation.KindDrugUse)
	ReportKindTerrorism              = string(domainmoderation.KindTerrorism)
	ReportKindPoliticalContent       = string(domainmoderation.KindPoliticalContent)
	ReportKindMisleadingAdvertising  = string(domainmoderation.KindMisleadingAdvertising)
	ReportKindSecurityVulnerability  = string(domainmoderation.KindSecurityVulnerability)
	ReportKindFakeProfile            = string(domainmoderation.KindFakeProfile)
	ReportKindUnderage               = string(domainmoderation.KindUnderage)
	ReportKindImpersonation          = string(domainmoderation.KindImpersonation)
	ReportKindNonConsensualContent   = string(domainmoderation.KindNonConsensualContent)
	ReportKindSexualHarassment       = string(domainmoderation.KindSexualHarassment)
	ReportKindSolicitation           = string(domainmoderation.KindSolicitation)
	ReportKindSelfPromotion          = string(domainmoderation.KindSelfPromotion)
	ReportKindGraphicViolence        = string(domainmoderation.KindGraphicViolence)
	ReportKindDiscriminatoryLanguage = string(domainmoderation.KindDiscriminatoryLanguage)
	ReportKindMalwarePhishing        = string(domainmoderation.KindMalwarePhishing)
	ReportKindInappropriateUsername  = string(domainmoderation.KindInappropriateUsername)
	ReportKindSelfHarmPromotion      = string(domainmoderation.KindSelfHarmPromotion)
	ReportKindThreatsBullying        = string(domainmoderation.KindThreatsBullying)
	ReportKindPrivacyViolation       = string(domainmoderation.KindPrivacyViolation)
	ReportKindFakeNews               = string(domainmoderation.KindFakeNews)
	ReportKindReligiousHateSpeech    = string(domainmoderation.KindReligiousHateSpeech)
	ReportKindPoliticalExtremism     = string(domainmoderation.KindPoliticalExtremism)
	ReportKindCulturalInsensitivity  = string(domainmoderation.KindCulturalInsensitivity)
	ReportKindIllegalActivities      = string(domainmoderation.KindIllegalActivities)
	ReportKindCopyrightViolation     = string(domainmoderation.KindCopyrightViolation)
	ReportKindOther                  = string(domainmoderation.KindOther)
)

func IsStandardReportKind(key string) bool {
	return domainmoderation.IsStandardKind(key)
}

type ReportKind struct {
	Key          string                `gorm:"primaryKey;size:64" json:"key"`
	DisplayOrder int                   `gorm:"default:0" json:"display_order"`
	Name         utils.LocalizedString `gorm:"type:jsonb" json:"name"`
	Description  utils.LocalizedString `gorm:"type:jsonb" json:"description"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// ReportStatus remains a models-owned type for JSON/GORM compatibility while
// delegating all lifecycle rules to the moderation domain.
type ReportStatus string

const (
	ReportStatusPending  ReportStatus = ReportStatus(domainmoderation.StatusPending)
	ReportStatusReviewed ReportStatus = ReportStatus(domainmoderation.StatusReviewed)
	ReportStatusRejected ReportStatus = ReportStatus(domainmoderation.StatusRejected)
	ReportStatusActioned ReportStatus = ReportStatus(domainmoderation.StatusActioned)
)

func (s ReportStatus) IsValid() bool {
	return domainmoderation.Status(s).IsValid()
}

func (s ReportStatus) CanTransitionTo(next ReportStatus) bool {
	return domainmoderation.Status(s).CanTransitionTo(domainmoderation.Status(next))
}

func (s ReportStatus) TransitionTo(next ReportStatus) (ReportStatus, error) {
	status, err := domainmoderation.Status(s).TransitionTo(domainmoderation.Status(next))
	return ReportStatus(status), err
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
