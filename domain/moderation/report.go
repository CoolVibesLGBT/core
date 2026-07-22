// Package moderation contains the reporting bounded context's business rules.
// It deliberately has no persistence or transport dependencies.
package moderation

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	MaxKindLength        = 64
	MaxDescriptionLength = 4000
)

var (
	ErrInvalidTarget           = errors.New("invalid report target")
	ErrTargetIDRequired        = validationError{message: "target id is required", cause: ErrInvalidTarget}
	ErrInvalidTargetType       = validationError{message: "invalid report target type", cause: ErrInvalidTarget}
	ErrInvalidKind             = errors.New("invalid report kind")
	ErrKindTooLong             = validationError{message: "report kind is too long", cause: ErrInvalidKind}
	ErrInvalidDescription      = errors.New("invalid report description")
	ErrDescriptionTooLong      = validationError{message: "report description is too long", cause: ErrInvalidDescription}
	ErrReporterRequired        = errors.New("reporter is required")
	ErrCannotReportSelf        = errors.New("users cannot report themselves")
	ErrInvalidStatus           = errors.New("invalid report status")
	ErrInvalidTransition       = errors.New("invalid report status transition")
	ErrInvalidResolutionAction = errors.New("invalid moderation action")
)

type validationError struct {
	message string
	cause   error
}

func (e validationError) Error() string { return e.message }

func (e validationError) Unwrap() error { return e.cause }

type TargetType string

const (
	TargetPost TargetType = "post"
	TargetUser TargetType = "user"
)

func (t TargetType) IsValid() bool {
	return t == TargetPost || t == TargetUser
}

func ParseTargetType(raw string) (TargetType, error) {
	targetType := TargetType(strings.TrimSpace(raw))
	if !targetType.IsValid() {
		return "", ErrInvalidTargetType
	}
	return targetType, nil
}

// Target is the report target value object. Public IDs are used here because
// target resolution to persistence IDs belongs to the repository adapter.
type Target struct {
	targetType TargetType
	publicID   int64
}

func NewTarget(targetType TargetType, publicID int64) (Target, error) {
	if !targetType.IsValid() {
		return Target{}, ErrInvalidTargetType
	}
	if publicID <= 0 {
		return Target{}, ErrTargetIDRequired
	}
	return Target{targetType: targetType, publicID: publicID}, nil
}

func (t Target) Type() TargetType { return t.targetType }

func (t Target) PublicID() int64 { return t.publicID }

func (t Target) IsValid() bool {
	return t.targetType.IsValid() && t.publicID > 0
}

type Kind string

const (
	KindSpam                   Kind = "spam"
	KindHateSpeech             Kind = "hate_speech"
	KindNudity                 Kind = "nudity"
	KindViolenceThreat         Kind = "violence_threat"
	KindFraud                  Kind = "fraud"
	KindHarassment             Kind = "harassment"
	KindPersonalInfo           Kind = "personal_info"
	KindFalseInfo              Kind = "false_info"
	KindProfanity              Kind = "profanity"
	KindSelfHarm               Kind = "self_harm"
	KindCopyrightInfringement  Kind = "copyright_infringement"
	KindDrugUse                Kind = "drug_use"
	KindTerrorism              Kind = "terrorism"
	KindPoliticalContent       Kind = "political_content"
	KindMisleadingAdvertising  Kind = "misleading_advertising"
	KindSecurityVulnerability  Kind = "security_vulnerability"
	KindFakeProfile            Kind = "fake_profile"
	KindUnderage               Kind = "underage"
	KindImpersonation          Kind = "impersonation"
	KindNonConsensualContent   Kind = "non_consensual_content"
	KindSexualHarassment       Kind = "sexual_harassment"
	KindSolicitation           Kind = "solicitation"
	KindSelfPromotion          Kind = "self_promotion"
	KindGraphicViolence        Kind = "graphic_violence"
	KindDiscriminatoryLanguage Kind = "discriminatory_language"
	KindMalwarePhishing        Kind = "malware_phishing"
	KindInappropriateUsername  Kind = "inappropriate_username"
	KindSelfHarmPromotion      Kind = "self_harm_promotion"
	KindThreatsBullying        Kind = "threats_bullying"
	KindPrivacyViolation       Kind = "privacy_violation"
	KindFakeNews               Kind = "fake_news"
	KindReligiousHateSpeech    Kind = "religious_hate_speech"
	KindPoliticalExtremism     Kind = "political_extremism"
	KindCulturalInsensitivity  Kind = "cultural_insensitivity"
	KindIllegalActivities      Kind = "illegal_activities"
	KindCopyrightViolation     Kind = "copyright_violation"
	KindOther                  Kind = "other"
)

var standardKinds = map[Kind]struct{}{
	KindSpam: {}, KindHateSpeech: {}, KindNudity: {}, KindViolenceThreat: {},
	KindFraud: {}, KindHarassment: {}, KindPersonalInfo: {}, KindFalseInfo: {},
	KindProfanity: {}, KindSelfHarm: {}, KindCopyrightInfringement: {}, KindDrugUse: {},
	KindTerrorism: {}, KindPoliticalContent: {}, KindMisleadingAdvertising: {},
	KindSecurityVulnerability: {}, KindFakeProfile: {}, KindUnderage: {}, KindImpersonation: {},
	KindNonConsensualContent: {}, KindSexualHarassment: {}, KindSolicitation: {},
	KindSelfPromotion: {}, KindGraphicViolence: {}, KindDiscriminatoryLanguage: {},
	KindMalwarePhishing: {}, KindInappropriateUsername: {}, KindSelfHarmPromotion: {},
	KindThreatsBullying: {}, KindPrivacyViolation: {}, KindFakeNews: {},
	KindReligiousHateSpeech: {}, KindPoliticalExtremism: {}, KindCulturalInsensitivity: {},
	KindIllegalActivities: {}, KindCopyrightViolation: {}, KindOther: {},
}

// NewKind validates the value object's shape. Whether a syntactically valid
// custom kind is configured is checked by the repository against report_kinds.
func NewKind(raw string) (Kind, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ErrInvalidKind
	}
	if utf8.RuneCountInString(value) > MaxKindLength {
		return "", ErrKindTooLong
	}
	return Kind(value), nil
}

func (k Kind) String() string { return string(k) }

func (k Kind) IsValid() bool {
	validated, err := NewKind(string(k))
	return err == nil && validated == k
}

func IsStandardKind(raw string) bool {
	_, ok := standardKinds[Kind(raw)]
	return ok
}

type Description string

func NewDescription(raw string) (Description, error) {
	value := strings.TrimSpace(raw)
	if utf8.RuneCountInString(value) > MaxDescriptionLength {
		return "", ErrDescriptionTooLong
	}
	return Description(value), nil
}

func (d Description) String() string { return string(d) }

func (d Description) IsValid() bool {
	validated, err := NewDescription(string(d))
	return err == nil && validated == d
}

type Status string

const (
	StatusPending  Status = "pending"
	StatusReviewed Status = "reviewed"
	StatusRejected Status = "rejected"
	StatusActioned Status = "actioned"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusReviewed, StatusRejected, StatusActioned:
		return true
	default:
		return false
	}
}

// CanTransitionTo describes moderator-driven state changes. Repeating a valid
// state is allowed so moderation client retries remain idempotent.
func (s Status) CanTransitionTo(next Status) bool {
	if !s.IsValid() || !next.IsValid() {
		return false
	}
	if s == next {
		return true
	}
	switch s {
	case StatusPending:
		return next == StatusReviewed || next == StatusRejected || next == StatusActioned
	case StatusReviewed:
		return next == StatusRejected || next == StatusActioned
	default:
		return false
	}
}

func (s Status) TransitionTo(next Status) (Status, error) {
	if !s.CanTransitionTo(next) {
		return s, ErrInvalidTransition
	}
	return next, nil
}

// Report is the aggregate used while accepting a new report and applying its
// lifecycle transitions. Persistence models map to and from these values.
type Report struct {
	target      Target
	kind        Kind
	description Description
	status      Status
}

func NewReport(target Target, kind Kind, description Description) (Report, error) {
	return RestoreReport(target, kind, description, StatusPending)
}

// RestoreReport rehydrates an aggregate without allowing an invalid status to
// enter the domain. Persistence-specific IDs and loading remain outside.
func RestoreReport(target Target, kind Kind, description Description, status Status) (Report, error) {
	if !target.IsValid() {
		return Report{}, ErrInvalidTarget
	}
	if !status.IsValid() {
		return Report{}, ErrInvalidStatus
	}
	validatedKind, err := NewKind(kind.String())
	if err != nil {
		return Report{}, err
	}
	validatedDescription, err := NewDescription(description.String())
	if err != nil {
		return Report{}, err
	}
	return Report{
		target:      target,
		kind:        validatedKind,
		description: validatedDescription,
		status:      status,
	}, nil
}

func (r Report) Target() Target           { return r.target }
func (r Report) Kind() Kind               { return r.kind }
func (r Report) Description() Description { return r.description }
func (r Report) Status() Status           { return r.status }

func (r Report) ValidateReporter(reporterPublicID int64) error {
	if reporterPublicID <= 0 {
		return ErrReporterRequired
	}
	if r.target.Type() == TargetUser && r.target.PublicID() == reporterPublicID {
		return ErrCannotReportSelf
	}
	return nil
}

func (r *Report) TransitionTo(next Status) error {
	status, err := r.status.TransitionTo(next)
	if err != nil {
		return err
	}
	r.status = status
	return nil
}

// ValidateResolution applies the invariant between a report decision and the
// optional post visibility action performed atomically with that decision.
func ValidateResolution(status Status, publishPost *bool) error {
	if !status.IsValid() || status == StatusPending {
		return ErrInvalidStatus
	}
	if publishPost == nil {
		return nil
	}
	if (!*publishPost && status != StatusActioned) || (*publishPost && status == StatusActioned) {
		return ErrInvalidResolutionAction
	}
	return nil
}
