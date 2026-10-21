package usecases

import (
	"core/constants"
	domainmedia "core/domain/media"
	domainmoderation "core/domain/moderation"
	domainuser "core/domain/user"
)

var (
	ErrModerationForbidden error = constants.ErrModerationForbidden
	ErrInvalidReportStatus       = domainmoderation.ErrInvalidStatus
	ErrReportIDRequired    error = constants.ErrReportIDRequired
	ErrPostIDRequired      error = constants.ErrPostIDRequired
	ErrInvalidReportType   error = constants.ErrInvalidReportType
)

var (
	ErrPrivatePhotoPrincipalRequired error = constants.ErrAuthenticatedUserRequired
	ErrPrivatePhotoOwnerRequired     error = constants.ErrPrivatePhotoOwnerRequired
	ErrPrivatePhotoRequestRequired   error = constants.ErrPrivatePhotoRequestRequired
	ErrPrivatePhotoViewerRequired    error = constants.ErrPrivatePhotoViewerRequired
	ErrPrivatePhotoIDRequired        error = constants.ErrPrivatePhotoIDRequired
	ErrPrivatePhotoUserNotFound      error = constants.ErrPrivatePhotoUserNotFound
	ErrPrivatePhotoNotFound          error = constants.ErrPrivatePhotoNotFound
	ErrPrivatePhotoRequestNotFound   error = constants.ErrPrivatePhotoRequestNotFound
	ErrPrivatePhotoForbidden         error = constants.ErrPrivatePhotoForbidden
	ErrPrivatePhotoSelfRequest       error = constants.ErrPrivatePhotoSelfRequest
	ErrPrivatePhotoFilesRequired     error = constants.ErrPrivatePhotoFilesRequired
	ErrPrivatePhotoUploadLimit       error = constants.ErrPrivatePhotoUploadLimit
	ErrPrivatePhotoOwnerLimit        error = constants.ErrPrivatePhotoLimitReached
	ErrPrivatePhotoImageRequired           = domainmedia.ErrPrivatePhotoImageRequired
	ErrPrivatePhotoImageDimensions         = domainmedia.ErrPrivatePhotoImageDimensions
	ErrPhotoDestinationRequired      error = constants.ErrPhotoDestinationRequired
)

var ErrAuthenticatedUserRequired error = constants.ErrAuthenticatedUserRequired

var ErrSessionUserNotFound error = constants.ErrSessionUserNotFound

var (
	ErrPrivateViewEngagements   error = constants.ErrPrivateViewEngagements
	ErrPrivateEngagements       error = constants.ErrPrivateEngagements
	ErrCannotReportSelf               = domainmoderation.ErrCannotReportSelf
	ErrUserIDRequired           error = constants.ErrUserIDRequired
	ErrCaptchaNotConfigured     error = constants.ErrCaptchaNotConfigured
	ErrPasswordNotConfigured    error = constants.ErrPasswordNotConfigured
	ErrTokenIssuerNotConfigured error = constants.ErrTokenIssuerNotConfigured
	ErrPublicIDNotConfigured    error = constants.ErrPublicIDNotConfigured
	ErrInvalidCredentials       error = constants.ErrInvalidCredentials
	ErrUsernameAlreadyExists          = domainuser.ErrUsernameAlreadyExists
	ErrEmailAlreadyExists             = domainuser.ErrEmailAlreadyExists
	ErrInvalidCurrentPassword   error = constants.ErrInvalidCurrentPassword
	ErrLegacyPasswordField      error = constants.ErrLegacyPasswordField
	ErrLocationCoordinates      error = constants.ErrLocationCoordinates
	ErrInvalidLocationOwner     error = constants.ErrInvalidLocationOwner
)
