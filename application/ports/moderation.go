package ports

import (
	"context"
	domainmoderation "core/domain/moderation"
	"time"

	"github.com/google/uuid"
)

// ModeratorRole is the application-facing authorization role. Transport
// adapters translate their authenticated user representation to this type.
type ModeratorRole string

const (
	ModeratorRoleModerator  ModeratorRole = "moderator"
	ModeratorRoleAdmin      ModeratorRole = "admin"
	ModeratorRoleSuperAdmin ModeratorRole = "super_admin"
)

// ModeratorPrincipal contains only the identity and authority facts required
// by moderation use cases. It deliberately does not expose a persistence user.
type ModeratorPrincipal struct {
	ID   uuid.UUID
	Role ModeratorRole
}

type ModerationReportFilter struct {
	Status           domainmoderation.Status
	AllStatuses      bool
	ContentableType  domainmoderation.TargetType
	PostPublicID     int64
	UserPublicID     int64
	ReporterPublicID int64
	Limit            int
	Cursor           *time.Time
	CursorID         *uuid.UUID
}

type ModerationLocalizedText map[string]string

type ModerationReportKindView struct {
	Key          string                  `json:"key"`
	DisplayOrder int                     `json:"display_order"`
	Name         ModerationLocalizedText `json:"name"`
	Description  ModerationLocalizedText `json:"description"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

// ModerationReportView is the application-owned report read model. Related
// user and post resources remain named snapshots so the HTTP adapter can keep
// their existing JSON shape without exposing persistence entities.
type ModerationReportView struct {
	ID              uuid.UUID                   `json:"id"`
	ContentableID   uuid.UUID                   `json:"contentable_id"`
	ContentableType domainmoderation.TargetType `json:"contentable_type"`
	ReporterID      uuid.UUID                   `json:"reporter_id"`
	Reporter        ModerationUserView          `json:"reporter,omitempty"`
	ReportKindKey   string                      `json:"report_kind_key"`
	ReportKind      *ModerationReportKindView   `json:"report_kind,omitempty"`
	Reason          string                      `json:"reason"`
	Status          domainmoderation.Status     `json:"status"`
	ReviewedByID    *uuid.UUID                  `json:"reviewed_by_id,omitempty"`
	ReviewedBy      ModerationUserView          `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time                  `json:"reviewed_at,omitempty"`
	Resolution      string                      `json:"resolution,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
}

// The named resource read models preserve the existing related-resource JSON
// shape while preventing GORM entities from crossing the repository boundary.
// Persistence adapters produce these detached response snapshots.
type ModerationPostView map[string]any
type ModerationUserView map[string]any

type ModerationReportItem struct {
	Report ModerationReportView `json:"report"`
	Post   ModerationPostView   `json:"post,omitempty"`
	User   ModerationUserView   `json:"user,omitempty"`
}

type ModerationReportPage struct {
	Items  []ModerationReportItem `json:"items"`
	Cursor *string                `json:"cursor,omitempty"`
	Count  int                    `json:"count"`
	Limit  int                    `json:"limit"`
}

type ModerationResolveInput struct {
	ReportID     uuid.UUID
	Status       domainmoderation.Status
	ReviewedByID uuid.UUID
	Resolution   string
	PublishPost  *bool
}

type ModerationRepository interface {
	FetchReports(ctx context.Context, filter ModerationReportFilter) (ModerationReportPage, error)
	ResolveReport(ctx context.Context, input ModerationResolveInput) (*ModerationReportView, error)
	SetPostPublished(ctx context.Context, postPublicID int64, published bool, moderatorID uuid.UUID, resolution string) (ModerationPostView, error)
}
