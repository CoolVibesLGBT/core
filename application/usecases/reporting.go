package usecases

import (
	domainmoderation "core/domain/moderation"
	"core/models"
	"errors"
)

var ErrAuthenticatedUserRequired = errors.New("authenticated user is required")

// validateReportSubmission is the single application entry point for both
// post and user reports. It keeps transport compatibility concerns out of the
// domain while ensuring every caller gets the same reporting invariants.
func validateReportSubmission(
	targetType domainmoderation.TargetType,
	targetPublicID int64,
	kind string,
	description string,
	authUser *models.User,
) (domainmoderation.Report, error) {
	target, err := domainmoderation.NewTarget(targetType, targetPublicID)
	if err != nil {
		return domainmoderation.Report{}, err
	}
	if authUser == nil {
		return domainmoderation.Report{}, ErrAuthenticatedUserRequired
	}
	reportKind, err := domainmoderation.NewKind(kind)
	if err != nil {
		return domainmoderation.Report{}, err
	}
	reportDescription, err := domainmoderation.NewDescription(description)
	if err != nil {
		return domainmoderation.Report{}, err
	}
	report, err := domainmoderation.NewReport(target, reportKind, reportDescription)
	if err != nil {
		return domainmoderation.Report{}, err
	}
	if err := report.ValidateReporter(authUser.PublicID); err != nil {
		return domainmoderation.Report{}, err
	}
	return report, nil
}
