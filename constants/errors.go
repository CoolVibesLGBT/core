package constants

type ErrorCode string

const (
	ErrUnknown           ErrorCode = "UNKNOWN_ERROR"
	ErrFileNotFound      ErrorCode = "FILE_NOT_FOUND"
	ErrPermissionDenied  ErrorCode = "PERMISSION_DENIED"
	ErrInvalidInput      ErrorCode = "INVALID_INPUT"
	ErrNetworkError      ErrorCode = "NETWORK_ERROR"
	ErrDatabaseError     ErrorCode = "DATABASE_ERROR"
	ErrResourceNotFound  ErrorCode = "RESOURCE_NOT_FOUND"
	ErrInvalidAction     ErrorCode = "INVALID_ACTION"
	ErrInvalidPassword   ErrorCode = "INVALID_PASSWORD"
	ErrTokenGeneration   ErrorCode = "TOKEN_GENERATION_FAILED"
	ErrUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrDuplicateResource ErrorCode = "DUPLICATE_RESOURCE"
	ErrInternalServer    ErrorCode = "INTERNAL_SERVER_ERROR"

	ErrMediaUploadFailed    ErrorCode = "MEDIA_UPLOAD_FAILED"
	ErrMediaInvalidFile     ErrorCode = "MEDIA_INVALID_FILE"
	ErrMediaUnsupportedType ErrorCode = "MEDIA_UNSUPPORTED_TYPE"
	ErrMediaSaveFailed      ErrorCode = "MEDIA_SAVE_FAILED"

	ErrUserExists              ErrorCode = "USER_EXISTS"
	ErrUserDoesntExists        ErrorCode = "USER_DOESNT_EXISTS"
	ErrEngagementsDoesntExists ErrorCode = "ENGAGEMENTS_DOESNT_EXISTS"

	ErrInvalidEngagementKind ErrorCode = "INVALID_ENGAGEMENT_KIND"
)

var ErrorMessages = map[ErrorCode]string{
	ErrUnknown:                 "An unknown error occurred.",
	ErrFileNotFound:            "The requested file could not be found.",
	ErrPermissionDenied:        "Permission denied.",
	ErrInvalidInput:            "Invalid input provided.",
	ErrNetworkError:            "A network error occurred.",
	ErrDatabaseError:           "A database error occurred.",
	ErrResourceNotFound:        "The requested resource could not be found.",
	ErrInvalidAction:           "The requested action is not valid.",
	ErrInvalidPassword:         "Invalid password.",
	ErrTokenGeneration:         "Failed to generate authentication token.",
	ErrUnauthorized:            "Unauthorized access.",
	ErrDuplicateResource:       "This resource already exists.",
	ErrMediaUploadFailed:       "Failed to upload media.",
	ErrMediaInvalidFile:        "Invalid media file provided.",
	ErrMediaUnsupportedType:    "Unsupported media file type.",
	ErrMediaSaveFailed:         "Failed to save media file.",
	ErrUserExists:              "User already exists",
	ErrUserDoesntExists:        "User doesnt exists",
	ErrInvalidEngagementKind:   "Invalid engagement kind",
	ErrEngagementsDoesntExists: "Engagements doesnt exists",
}

// String returns a readable message for the given error code.
func (e ErrorCode) String() string {
	if msg, ok := ErrorMessages[e]; ok {
		return msg
	}
	return ErrorMessages[ErrUnknown]
}
