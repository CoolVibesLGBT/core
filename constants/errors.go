package constants

type ErrorCode string

const (
	ErrUnknown              ErrorCode = "UNKNOWN_ERROR"
	ErrFileNotFound         ErrorCode = "FILE_NOT_FOUND"
	ErrPermissionDenied     ErrorCode = "PERMISSION_DENIED"
	ErrInvalidInput         ErrorCode = "INVALID_INPUT"
	ErrInvalidForm          ErrorCode = "INVALID_FORM"
	ErrNetworkError         ErrorCode = "NETWORK_ERROR"
	ErrDatabaseError        ErrorCode = "DATABASE_ERROR"
	ErrResourceNotFound     ErrorCode = "RESOURCE_NOT_FOUND"
	ErrInvalidAction        ErrorCode = "INVALID_ACTION"
	ErrInvalidPassword      ErrorCode = "INVALID_PASSWORD"
	ErrTokenGeneration      ErrorCode = "TOKEN_GENERATION_FAILED"
	ErrDuplicateResource    ErrorCode = "DUPLICATE_RESOURCE"
	ErrInternalServer       ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrMethodNotImplemented ErrorCode = "METHOD_NOT_IMPLEMENTED"

	// Preferences
	ErrPreferencesFetchFailed ErrorCode = "PREFERENCES_FETCH_FAILED"

	// EventKinds
	ErrEventKindsFetchFailed ErrorCode = "EVENT_KINDS_FETCH_FAILED"

	// ReportKinds
	ErrReportKindsFetchFailed ErrorCode = "REPORT_KINDS_FETCH_FAILED"

	// Media
	ErrMediaUploadFailed    ErrorCode = "MEDIA_UPLOAD_FAILED"
	ErrMediaInvalidFile     ErrorCode = "MEDIA_INVALID_FILE"
	ErrMediaUnsupportedType ErrorCode = "MEDIA_UNSUPPORTED_TYPE"
	ErrMediaSaveFailed      ErrorCode = "MEDIA_SAVE_FAILED"

	ErrVapidKeyGenerationFailed ErrorCode = "VAPID_KEY_GENERATION_FAILED"
	ErrVapidKeyNotFound         ErrorCode = "VAPID_KEY_NOT_FOUND"
	ErrVapidSubscriptionFailed  ErrorCode = "VAPID_SUBSCRIPTION_FAILED"

	// Notifications
	ErrNotificationsFetchFailed ErrorCode = "NOTIFICATIONS_FETCH_FAILED"

	// User
	ErrUserExists       ErrorCode = "USER_EXISTS"
	ErrUserNotFound     ErrorCode = "USER_NOT_FOUND"
	ErrUsernameRequired ErrorCode = "USERNAME_REQUIRED"
	ErrUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrUserUnauthorized ErrorCode = "USER_UNAUTHORIZED"
	ErrUsernameTaken    ErrorCode = "USERNAME_TAKEN"

	// Engagement
	ErrEngagementNotFound    ErrorCode = "ENGAGEMENT_NOT_FOUND"
	ErrInvalidEngagementKind ErrorCode = "INVALID_ENGAGEMENT_KIND"

	// Poll
	ErrPollTitleEmpty   ErrorCode = "POLL_TITLE_EMPTY"
	ErrPollOptionsEmpty ErrorCode = "POLL_OPTIONS_EMPTY"
	ErrChoiceIDInvalid  ErrorCode = "CHOICE_ID_INVALID"
	ErrWeightInvalid    ErrorCode = "WEIGHT_INVALID"
	ErrRankInvalid      ErrorCode = "RANK_INVALID"
	ErrVoteFailed       ErrorCode = "VOTE_FAILED"
	// Post
	ErrPostNotFound     ErrorCode = "POST_NOT_FOUND"
	ErrPostDeleteDenied ErrorCode = "POST_DELETE_DENIED"
	ErrPostDeleteFailed ErrorCode = "POST_DELETE_FAILED"
	ErrPostCreateFailed ErrorCode = "POST_CREATE_FAILED"

	ErrInsufficientBalance  ErrorCode = "INSUFFICIENT_BALANCE"
	ErrInvalidAmount        ErrorCode = "INVALID_AMOUNT"
	ErrFailedToBookmarkPost ErrorCode = "FAILED_TO_BOOKMARK_POST"

	// Place
	ErrPlaceNotFound     ErrorCode = "PLACE_NOT_FOUND"
	ErrPlaceDeleteDenied ErrorCode = "PLACE_DELETE_DENIED"
	ErrPlaceDeleteFailed ErrorCode = "PLACE_DELETE_FAILED"
	ErrPlaceCreateFailed ErrorCode = "PLACE_CREATE_FAILED"
	// Chat
	ErrSelfChatNotAllowed           ErrorCode = "SELF_CHAT_NOT_ALLOWED"
	ErrUnsupportedChatType          ErrorCode = "UNSUPPORTED_CHAT_TYPE"
	ErrInvalidParticipantID         ErrorCode = "INVALID_PARTICIPANT_ID"
	ErrInvalidParticipantsLength    ErrorCode = "INVALID_PARTICIPANTS_LENGTH"
	ErrFailedToLoadMessages         ErrorCode = "FAILED_TO_LOAD_MESSAGES"
	ErrInvalidChatID                ErrorCode = "INVALID_CHAT_ID"
	ErrInvalidMessageID             ErrorCode = "INVALID_MESSAGE_ID"
	ErrFailedToPinMessage           ErrorCode = "FAILED_TO_PIN_MESSAGE"
	ErrFailedToUnpinMessage         ErrorCode = "FAILED_TO_UNPIN_MESSAGE"
	ErrFailedToDeleteMessageForUser ErrorCode = "FAILED_TO_DELETE_MESSAGE_FOR_USER"
	ErrFailedToDeleteChatForAll     ErrorCode = "FAILED_TO_DELETE_CHAT_FOR_ALL"
	ErrFailedToDeleteChatForUser    ErrorCode = "FAILED_TO_DELETE_CHAT_FOR_USER"

	// Location / Geo
	ErrInvalidLatitude  ErrorCode = "INVALID_LATITUDE"
	ErrInvalidLongitude ErrorCode = "INVALID_LONGITUDE"
)

var ErrorMessages = map[ErrorCode]string{
	ErrInvalidForm:          "Invalid form data.",
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

	ErrVapidKeyGenerationFailed: "Failed to generate VAPID keys.",
	ErrVapidKeyNotFound:         "VAPID key not found.",
	ErrVapidSubscriptionFailed:  "Failed to save VAPID subscription.",

	// Preferences
	ErrPreferencesFetchFailed: "Failed to fetch user preferences.",

	// EventKinds
	ErrEventKindsFetchFailed: "Failed to fetch event kinds.",

	// ReportKinds
	ErrReportKindsFetchFailed: "Failed to fetch report kinds.",

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
	ErrUsernameTaken:    "Username already taken",

	// Engagement
	ErrEngagementNotFound:    "Engagement record not found.",
	ErrInvalidEngagementKind: "Invalid engagement type.",

	// Poll
	ErrPollTitleEmpty:   "Poll title cannot be empty.",
	ErrPollOptionsEmpty: "Poll options cannot be empty.",
	ErrChoiceIDInvalid:  "The provided choice ID is invalid.",
	ErrWeightInvalid:    "Weight must be a positive integer.",
	ErrRankInvalid:      "Rank must be zero or a positive integer.",
	ErrVoteFailed:       "Failed to register your vote. Please try again.",

	// Post
	ErrPostNotFound:         "Post not found.",
	ErrPostDeleteDenied:     "You are not allowed to delete this post.",
	ErrPostDeleteFailed:     "Failed to delete the post.",
	ErrInsufficientBalance:  "Insufficient balance.",    // yeni mesaj eklendi
	ErrInvalidAmount:        "Invalid amount provided.", // yeni mesaj
	ErrPostCreateFailed:     "Failed to create post",
	ErrFailedToBookmarkPost: "Failed to bookmark post",

	// Place
	ErrPlaceCreateFailed: "Failed to create place",
	ErrPlaceNotFound:     "Place not found",
	ErrPlaceDeleteDenied: "You are not allowed to delete this place",
	ErrPlaceDeleteFailed: "Failed to delete the place",

	// Chat
	ErrSelfChatNotAllowed:           "You cannot create a chat with yourself.",
	ErrUnsupportedChatType:          "Unsupported chat type.",
	ErrInvalidParticipantID:         "Invalid participant ID.",
	ErrInvalidParticipantsLength:    "Invalid participants length.",
	ErrFailedToLoadMessages:         "Failed to load messages. Please try again later.",
	ErrInvalidChatID:                "The provided chat ID is invalid.",
	ErrInvalidMessageID:             "The provided message ID is invalid.",
	ErrFailedToPinMessage:           "Failed to pin message.",
	ErrFailedToUnpinMessage:         "Failed to unpin the message. Please try again later.",
	ErrFailedToDeleteMessageForUser: "Failed to delete the message for the user.",
	ErrFailedToDeleteChatForAll:     "Failed to delete the chat for all.",
	ErrFailedToDeleteChatForUser:    "Failed to delete the chat for the user.",

	// Location / Geo
	ErrInvalidLatitude:  "Invalid latitude.",
	ErrInvalidLongitude: "Invalid longitude.",

	// Notifications
	ErrNotificationsFetchFailed: "Failed to fetch notifications.",
}

func (e ErrorCode) String() string {
	if msg, ok := ErrorMessages[e]; ok {
		return msg
	}
	return ErrorMessages[ErrUnknown]
}
