package constants

type ErrorCode string

const (
	ErrUnknown              ErrorCode = "UNKNOWN_ERROR"
	ErrFileNotFound         ErrorCode = "FILE_NOT_FOUND"
	ErrPermissionDenied     ErrorCode = "PERMISSION_DENIED"
	ErrInvalidInput         ErrorCode = "INVALID_INPUT"
	ErrNetworkError         ErrorCode = "NETWORK_ERROR"
	ErrDatabaseError        ErrorCode = "DATABASE_ERROR"
	ErrResourceNotFound     ErrorCode = "RESOURCE_NOT_FOUND"
	ErrInvalidAction        ErrorCode = "INVALID_ACTION"
	ErrInvalidPassword      ErrorCode = "INVALID_PASSWORD"
	ErrTokenGeneration      ErrorCode = "TOKEN_GENERATION_FAILED"
	ErrDuplicateResource    ErrorCode = "DUPLICATE_RESOURCE"
	ErrInternalServer       ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrMethodNotImplemented ErrorCode = "METHOD_NOT_IMPLEMENTED"
	// Media
	ErrMediaUploadFailed    ErrorCode = "MEDIA_UPLOAD_FAILED"
	ErrMediaInvalidFile     ErrorCode = "MEDIA_INVALID_FILE"
	ErrMediaUnsupportedType ErrorCode = "MEDIA_UNSUPPORTED_TYPE"
	ErrMediaSaveFailed      ErrorCode = "MEDIA_SAVE_FAILED"

	// User
	ErrUserExists       ErrorCode = "USER_EXISTS"
	ErrUserNotFound     ErrorCode = "USER_NOT_FOUND"
	ErrUsernameRequired ErrorCode = "USERNAME_REQUIRED"
	ErrUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrUserUnauthorized ErrorCode = "USER_UNAUTHORIZED"

	// Engagement
	ErrEngagementNotFound    ErrorCode = "ENGAGEMENT_NOT_FOUND"
	ErrInvalidEngagementKind ErrorCode = "INVALID_ENGAGEMENT_KIND"

	// Poll
	ErrPollTitleEmpty   ErrorCode = "POLL_TITLE_EMPTY"
	ErrPollOptionsEmpty ErrorCode = "POLL_OPTIONS_EMPTY"

	// Post
	ErrPostNotFound        ErrorCode = "POST_NOT_FOUND"
	ErrPostDeleteDenied    ErrorCode = "POST_DELETE_DENIED"
	ErrPostDeleteFailed    ErrorCode = "POST_DELETE_FAILED"
	ErrInsufficientBalance ErrorCode = "INSUFFICIENT_BALANCE"
	ErrInvalidAmount       ErrorCode = "INVALID_AMOUNT" // yeni hata kodu

	// Chat
	ErrSelfChatNotAllowed        ErrorCode = "SELF_CHAT_NOT_ALLOWED"
	ErrUnsupportedChatType       ErrorCode = "UNSUPPORTED_CHAT_TYPE"
	ErrInvalidParticipantID      ErrorCode = "INVALID_PARTICIPANT_ID"
	ErrInvalidParticipantsLength ErrorCode = "INVALID_PARTICIPANTS_LENGTH"
)

var ErrorMessages = map[ErrorCode]string{
	ErrUnknown:              "An unknown error occurred.",
	ErrFileNotFound:         "The requested file could not be found.",
	ErrPermissionDenied:     "Permission denied.",
	ErrInvalidInput:         "Invalid input provided.",
	ErrNetworkError:         "A network error occurred.",
	ErrDatabaseError:        "A database error occurred.",
	ErrResourceNotFound:     "The requested resource could not be found.",
	ErrInvalidAction:        "The requested action is not valid.",
	ErrInvalidPassword:      "Invalid password.",
	ErrTokenGeneration:      "Failed to generate authentication token.",
	ErrUnauthorized:         "Unauthorized access.",
	ErrDuplicateResource:    "This resource already exists.",
	ErrInternalServer:       "An internal server error occurred.",
	ErrMethodNotImplemented: "The requested method is not implemented.",
	// Media
	ErrMediaUploadFailed:    "Failed to upload media.",
	ErrMediaInvalidFile:     "Invalid media file.",
	ErrMediaUnsupportedType: "Unsupported media type.",
	ErrMediaSaveFailed:      "Failed to save media.",

	// User
	ErrUserExists:       "User already exists.",
	ErrUserNotFound:     "User not found.",
	ErrUsernameRequired: "username or nickname is required",
	ErrUserUnauthorized: "User is not authorized to perform this action.",

	// Engagement
	ErrEngagementNotFound:    "Engagement record not found.",
	ErrInvalidEngagementKind: "Invalid engagement type.",

	// Poll
	ErrPollTitleEmpty:   "Poll title cannot be empty.",
	ErrPollOptionsEmpty: "Poll options cannot be empty.",

	// Post
	ErrPostNotFound:        "Post not found.",
	ErrPostDeleteDenied:    "You are not allowed to delete this post.",
	ErrPostDeleteFailed:    "Failed to delete the post.",
	ErrInsufficientBalance: "Insufficient balance.",    // yeni mesaj eklendi
	ErrInvalidAmount:       "Invalid amount provided.", // yeni mesaj

	// Chat
	ErrSelfChatNotAllowed:        "You cannot create a chat with yourself.",
	ErrUnsupportedChatType:       "Unsupported chat type.",
	ErrInvalidParticipantID:      "Invalid participant ID.",
	ErrInvalidParticipantsLength: "Invalid participants length.",
}

func (e ErrorCode) String() string {
	if msg, ok := ErrorMessages[e]; ok {
		return msg
	}
	return ErrorMessages[ErrUnknown]
}
