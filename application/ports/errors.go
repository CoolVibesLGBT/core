package ports

import (
	domainmoderation "core/domain/moderation"
	"errors"
)

var ErrNotFound = errors.New("resource not found")
var ErrPrivatePhotoLimitReached = errors.New("private photo album limit reached")
var ErrPaymentProcessingNotImplemented = errors.New("payment processing is not implemented")

var (
	ErrInvalidReportKind       = domainmoderation.ErrInvalidKind
	ErrReportTargetNotFound    = errors.New("report target not found")
	ErrReportNotFound          = errors.New("report not found")
	ErrInvalidReportTransition = domainmoderation.ErrInvalidTransition
	ErrInvalidModerationAction = domainmoderation.ErrInvalidResolutionAction
)
