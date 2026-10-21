package ports

import (
	"core/constants"
	domainmoderation "core/domain/moderation"
)

var ErrRegistrationPersistence error = constants.ErrRegistrationPersistence

var ErrChatGIFUnavailable error = constants.ErrChatGIFUnavailable

var ErrNotFound error = constants.ErrNotFound
var ErrPrivatePhotoLimitReached error = constants.ErrPrivatePhotoLimitReached
var ErrPrivatePhotoAccessBlocked error = constants.ErrPrivatePhotoAccessBlocked
var ErrPaymentProcessingNotImplemented error = constants.ErrPaymentProcessingNotImplemented

var (
	ErrInvalidReportKind             = domainmoderation.ErrInvalidKind
	ErrReportTargetNotFound    error = constants.ErrReportTargetNotFound
	ErrReportNotFound          error = constants.ErrReportNotFound
	ErrInvalidReportTransition       = domainmoderation.ErrInvalidTransition
	ErrInvalidModerationAction       = domainmoderation.ErrInvalidResolutionAction
)

var (
	ErrBroadcastIntegrationDisabled error = constants.ErrBroadcastIntegrationDisabled
	ErrBroadcastProviderUnsupported error = constants.ErrBroadcastProviderUnsupported
	ErrBroadcastInvalidInput        error = constants.ErrBroadcastInvalidInput
	ErrBroadcastUpstream            error = constants.ErrBroadcastUpstream
	ErrBroadcastUnauthorized        error = constants.ErrBroadcastUnauthorized
)

var (
	ErrBroadcastSessionUnavailable error = constants.ErrBroadcastSessionUnavailable
	ErrBroadcastSessionForbidden   error = constants.ErrBroadcastSessionForbidden
)
