package constants

import "fmt"

type ErrorCode string

const (
	ErrUnknown              ErrorCode = "UNKNOWN_ERROR"
	ErrFileNotFound         ErrorCode = "FILE_NOT_FOUND"
	ErrPermissionDenied     ErrorCode = "PERMISSION_DENIED"
	ErrInvalidInput         ErrorCode = "INVALID_INPUT"
	ErrInvalidForm          ErrorCode = "INVALID_FORM"
	ErrNetworkError         ErrorCode = "NETWORK_ERROR"
	ErrRequestTimeout       ErrorCode = "REQUEST_TIMEOUT"
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
	ErrUserExists            ErrorCode = "USER_EXISTS"
	ErrUserNotFound          ErrorCode = "USER_NOT_FOUND"
	ErrUsernameRequired      ErrorCode = "USERNAME_REQUIRED"
	ErrUsernameInvalidFormat ErrorCode = "USERNAME_INVALID_FORMAT"
	ErrUnauthorized          ErrorCode = "UNAUTHORIZED"
	ErrUserUnauthorized      ErrorCode = "USER_UNAUTHORIZED"
	ErrUsernameTaken         ErrorCode = "USERNAME_TAKEN"

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
	ErrPostNotFound         ErrorCode = "POST_NOT_FOUND"
	ErrPostDeleteDenied     ErrorCode = "POST_DELETE_DENIED"
	ErrPostDeleteFailed     ErrorCode = "POST_DELETE_FAILED"
	ErrPostCreateFailed     ErrorCode = "POST_CREATE_FAILED"
	ErrPostUpdateFailed     ErrorCode = "POST_UPDATE_FAILED"
	ErrPostSaveFailed       ErrorCode = "POST_SAVE_FAILED"
	ErrPostUnsaveFailed     ErrorCode = "POST_UNSAVE_FAILED"
	ErrPostShareFailed      ErrorCode = "POST_SHARE_FAILED"
	ErrPostReportFailed     ErrorCode = "POST_REPORT_FAILED"
	ErrPostLikeFailed       ErrorCode = "POST_LIKE_FAILED"
	ErrPostDislikeFailed    ErrorCode = "POST_DISLIKE_FAILED"
	ErrPostUnlikeFailed     ErrorCode = "POST_UNLIKE_FAILED"
	ErrPostBookmarkFailed   ErrorCode = "POST_BOOKMARK_FAILED"
	ErrPostUnbookmarkFailed ErrorCode = "POST_UNBOOKMARK_FAILED"
	ErrFailedToBookmarkPost ErrorCode = "FAILED_TO_BOOKMARK_POST"

	ErrPostSubscribeFailed     ErrorCode = "POST_SUBSCRIBE_FAILED"
	ErrPostUnsubscribeFailed   ErrorCode = "POST_UNSUBSCRIBE_FAILED"
	ErrFailedToSubscribePost   ErrorCode = "FAILED_TO_SUBSCRIBE_POST"
	ErrFailedToUnsubscribePost ErrorCode = "FAILED_TO_UNSUBSCRIBE_POST"

	ErrInsufficientBalance ErrorCode = "INSUFFICIENT_BALANCE"
	ErrInvalidAmount       ErrorCode = "INVALID_AMOUNT"
	ErrFailedToDislikePost ErrorCode = "FAILED_TO_DISLIKE_POST"
	ErrFailedToLikePost    ErrorCode = "FAILED_TO_LIKE_POST"
	ErrFailedToReportPost  ErrorCode = "FAILED_TO_REPORT_POST"
	ErrFailedToSavePost    ErrorCode = "FAILED_TO_SAVE_POST"
	ErrFailedToUnsavePost  ErrorCode = "FAILED_TO_UNSAVE_POST"
	ErrFailedToSharePost   ErrorCode = "FAILED_TO_SHARE_POST"
	ErrFailedToUnlikePost  ErrorCode = "FAILED_TO_UNLIKE_POST"
	ErrFailedToBananaPost  ErrorCode = "FAILED_TO_BANANA_POST"
	ErrCannotTipOwnPost    ErrorCode = "CANNOT_TIP_OWN_POST"

	// Place
	ErrPlaceNotFound     ErrorCode = "PLACE_NOT_FOUND"
	ErrPlaceDeleteDenied ErrorCode = "PLACE_DELETE_DENIED"
	ErrPlaceDeleteFailed ErrorCode = "PLACE_DELETE_FAILED"
	ErrPlaceCreateFailed ErrorCode = "PLACE_CREATE_FAILED"
	// Chat
	ErrGroupOwnerCannotLeave            ErrorCode = "GROUP_OWNER_CANNOT_LEAVE"
	ErrSelfChatNotAllowed               ErrorCode = "SELF_CHAT_NOT_ALLOWED"
	ErrUnsupportedChatType              ErrorCode = "UNSUPPORTED_CHAT_TYPE"
	ErrInvalidParticipantID             ErrorCode = "INVALID_PARTICIPANT_ID"
	ErrInvalidParticipantsLength        ErrorCode = "INVALID_PARTICIPANTS_LENGTH"
	ErrFailedToLoadMessages             ErrorCode = "FAILED_TO_LOAD_MESSAGES"
	ErrInvalidChatID                    ErrorCode = "INVALID_CHAT_ID"
	ErrInvalidMessageID                 ErrorCode = "INVALID_MESSAGE_ID"
	ErrFailedToPinMessage               ErrorCode = "FAILED_TO_PIN_MESSAGE"
	ErrFailedToUnpinMessage             ErrorCode = "FAILED_TO_UNPIN_MESSAGE"
	ErrFailedToDeleteMessage            ErrorCode = "FAILED_TO_DELETE_MESSAGE"
	ErrFailedToDeleteMessageForUser     ErrorCode = "FAILED_TO_DELETE_MESSAGE_FOR_USER"
	ErrFailedToDeleteMessageForAll      ErrorCode = "FAILED_TO_DELETE_MESSAGE_FOR_ALL"
	ErrFailedToDeleteChatForAll         ErrorCode = "FAILED_TO_DELETE_CHAT_FOR_ALL"
	ErrFailedToDeleteChatForUser        ErrorCode = "FAILED_TO_DELETE_CHAT_FOR_USER"
	ErrFailedToDeleteChatHistoryForUser ErrorCode = "FAILED_TO_DELETE_CHAT_HISTORY_FOR_USER"
	ErrFailedToDeleteChatHistoryForAll  ErrorCode = "FAILED_TO_DELETE_CHAT_HISTORY_FOR_ALL"

	// Location / Geo
	ErrInvalidLatitude  ErrorCode = "INVALID_LATITUDE"
	ErrInvalidLongitude ErrorCode = "INVALID_LONGITUDE"

	// Payment
	ErrPaymentMethodNotFound    ErrorCode = "PAYMENT_METHOD_NOT_FOUND"
	ErrPaymentMethodFetchFailed ErrorCode = "PAYMENT_METHOD_FETCH_FAILED"

	// News
	ErrNewsCreateFailed ErrorCode = "NEWS_CREATE_FAILED"

	// Others
	ErrOGFetchFailed   ErrorCode = "OG_FETCH_FAILED"
	ErrHTMLParseFailed ErrorCode = "HTML_PARSE_FAILED"

	//Check In
	ErrFailedtoCheckIn ErrorCode = "FAILED_TO_CHECK_IN"

	// Repository and service errors.
	ErrAIProviderNotConfigured                     ErrorCode = "AI_PROVIDER_NOT_CONFIGURED"
	ErrBroadcastPageSizeExceeded                   ErrorCode = "BROADCAST_PAGE_SIZE_EXCEEDED"
	ErrBroadcastDescriptionTooLong                 ErrorCode = "BROADCAST_DESCRIPTION_TOO_LONG"
	ErrBroadcastIDRequired                         ErrorCode = "BROADCAST_ID_REQUIRED"
	ErrBroadcastGuestIdentifiersRequired           ErrorCode = "BROADCAST_GUEST_IDENTIFIERS_REQUIRED"
	ErrBroadcastLikeInputRequired                  ErrorCode = "BROADCAST_LIKE_INPUT_REQUIRED"
	ErrChatOpenMessageResultEmpty                  ErrorCode = "CHAT_OPEN_MESSAGE_RESULT_EMPTY"
	ErrPostLookupFailed                            ErrorCode = "POST_LOOKUP_FAILED"
	ErrMatchesRepositoryNotConfigured              ErrorCode = "MATCHES_REPOSITORY_NOT_CONFIGURED"
	ErrModerationForbidden                         ErrorCode = "MODERATION_FORBIDDEN"
	ErrReportIDRequired                            ErrorCode = "REPORT_ID_REQUIRED"
	ErrPostIDRequired                              ErrorCode = "POST_ID_REQUIRED"
	ErrInvalidReportType                           ErrorCode = "INVALID_REPORT_TYPE"
	ErrPostSlugLookupFailed                        ErrorCode = "POST_SLUG_LOOKUP_FAILED"
	ErrUserUUIDLookupFailed                        ErrorCode = "USER_UUID_LOOKUP_FAILED"
	ErrUserMediaLookupFailed                       ErrorCode = "USER_MEDIA_LOOKUP_FAILED"
	ErrRecentHashtagsLookupFailed                  ErrorCode = "RECENT_HASHTAGS_LOOKUP_FAILED"
	ErrAuthenticatedUserRequired                   ErrorCode = "AUTHENTICATED_USER_REQUIRED"
	ErrPrivatePhotoOwnerRequired                   ErrorCode = "PRIVATE_PHOTO_OWNER_REQUIRED"
	ErrPrivatePhotoRequestRequired                 ErrorCode = "PRIVATE_PHOTO_REQUEST_REQUIRED"
	ErrPrivatePhotoViewerRequired                  ErrorCode = "PRIVATE_PHOTO_VIEWER_REQUIRED"
	ErrPrivatePhotoIDRequired                      ErrorCode = "PRIVATE_PHOTO_ID_REQUIRED"
	ErrPrivatePhotoUserNotFound                    ErrorCode = "PRIVATE_PHOTO_USER_NOT_FOUND"
	ErrPrivatePhotoNotFound                        ErrorCode = "PRIVATE_PHOTO_NOT_FOUND"
	ErrPrivatePhotoRequestNotFound                 ErrorCode = "PRIVATE_PHOTO_REQUEST_NOT_FOUND"
	ErrPrivatePhotoForbidden                       ErrorCode = "PRIVATE_PHOTO_FORBIDDEN"
	ErrPrivatePhotoSelfRequest                     ErrorCode = "PRIVATE_PHOTO_SELF_REQUEST"
	ErrPrivatePhotoFilesRequired                   ErrorCode = "PRIVATE_PHOTO_FILES_REQUIRED"
	ErrPrivatePhotoUploadLimit                     ErrorCode = "PRIVATE_PHOTO_UPLOAD_LIMIT"
	ErrPrivatePhotoLimitReached                    ErrorCode = "PRIVATE_PHOTO_LIMIT_REACHED"
	ErrPhotoDestinationRequired                    ErrorCode = "PHOTO_DESTINATION_REQUIRED"
	ErrProfilePhotoRepositoryNotConfigured         ErrorCode = "PROFILE_PHOTO_REPOSITORY_NOT_CONFIGURED"
	ErrSessionUserNotFound                         ErrorCode = "SESSION_USER_NOT_FOUND"
	ErrPrivateViewEngagements                      ErrorCode = "PRIVATE_VIEW_ENGAGEMENTS"
	ErrPrivateEngagements                          ErrorCode = "PRIVATE_ENGAGEMENTS"
	ErrUserIDRequired                              ErrorCode = "USER_ID_REQUIRED"
	ErrCaptchaNotConfigured                        ErrorCode = "CAPTCHA_NOT_CONFIGURED"
	ErrPasswordNotConfigured                       ErrorCode = "PASSWORD_NOT_CONFIGURED"
	ErrTokenIssuerNotConfigured                    ErrorCode = "TOKEN_ISSUER_NOT_CONFIGURED"
	ErrPublicIDNotConfigured                       ErrorCode = "PUBLIC_ID_NOT_CONFIGURED"
	ErrInvalidCredentials                          ErrorCode = "INVALID_CREDENTIALS"
	ErrInvalidCurrentPassword                      ErrorCode = "INVALID_CURRENT_PASSWORD"
	ErrLegacyPasswordField                         ErrorCode = "LEGACY_PASSWORD_FIELD"
	ErrLocationCoordinates                         ErrorCode = "LOCATION_COORDINATES"
	ErrInvalidLocationOwner                        ErrorCode = "INVALID_LOCATION_OWNER"
	ErrInvalidCaptcha                              ErrorCode = "INVALID_CAPTCHA"
	ErrPasswordHashFailed                          ErrorCode = "PASSWORD_HASH_FAILED"
	ErrReferralEmpty                               ErrorCode = "REFERRAL_EMPTY"
	ErrAvatarUserLookupFailed                      ErrorCode = "AVATAR_USER_LOOKUP_FAILED"
	ErrAvatarUploadFailed                          ErrorCode = "AVATAR_UPLOAD_FAILED"
	ErrAvatarUpdateFailed                          ErrorCode = "AVATAR_UPDATE_FAILED"
	ErrRemoteImageFetcherNotConfigured             ErrorCode = "REMOTE_IMAGE_FETCHER_NOT_CONFIGURED"
	ErrCoverUserLookupFailed                       ErrorCode = "COVER_USER_LOOKUP_FAILED"
	ErrCoverUploadFailed                           ErrorCode = "COVER_UPLOAD_FAILED"
	ErrCoverUpdateFailed                           ErrorCode = "COVER_UPDATE_FAILED"
	ErrUserAttributeUpsertFailed                   ErrorCode = "USER_ATTRIBUTE_UPSERT_FAILED"
	ErrBroadcastIntegrationDisabled                ErrorCode = "BROADCAST_INTEGRATION_DISABLED"
	ErrBroadcastProviderUnsupported                ErrorCode = "BROADCAST_PROVIDER_UNSUPPORTED"
	ErrBroadcastInvalidInput                       ErrorCode = "BROADCAST_INVALID_INPUT"
	ErrBroadcastUpstream                           ErrorCode = "BROADCAST_UPSTREAM"
	ErrBroadcastUnauthorized                       ErrorCode = "BROADCAST_UNAUTHORIZED"
	ErrBroadcastSessionUnavailable                 ErrorCode = "BROADCAST_SESSION_UNAVAILABLE"
	ErrBroadcastSessionForbidden                   ErrorCode = "BROADCAST_SESSION_FORBIDDEN"
	ErrRegistrationPersistence                     ErrorCode = "REGISTRATION_PERSISTENCE"
	ErrNotFound                                    ErrorCode = "NOT_FOUND"
	ErrPrivatePhotoAccessBlocked                   ErrorCode = "PRIVATE_PHOTO_ACCESS_BLOCKED"
	ErrPaymentProcessingNotImplemented             ErrorCode = "PAYMENT_PROCESSING_NOT_IMPLEMENTED"
	ErrReportTargetNotFound                        ErrorCode = "REPORT_TARGET_NOT_FOUND"
	ErrReportNotFound                              ErrorCode = "REPORT_NOT_FOUND"
	ErrChatVisibilityIdentifiersRequired           ErrorCode = "CHAT_VISIBILITY_IDENTIFIERS_REQUIRED"
	ErrViewOnceMediaReadFailed                     ErrorCode = "VIEW_ONCE_MEDIA_READ_FAILED"
	ErrEngagementDetailAlreadyExists               ErrorCode = "ENGAGEMENT_DETAIL_ALREADY_EXISTS"
	ErrEngagementCounterMissing                    ErrorCode = "ENGAGEMENT_COUNTER_MISSING"
	ErrEngagementAggregateNotUpdated               ErrorCode = "ENGAGEMENT_AGGREGATE_NOT_UPDATED"
	ErrEngagementAggregateRequired                 ErrorCode = "ENGAGEMENT_AGGREGATE_REQUIRED"
	ErrReciprocalInteractionConflict               ErrorCode = "RECIPROCAL_INTERACTION_CONFLICT"
	ErrInteractionUserIdentifiersRequired          ErrorCode = "INTERACTION_USER_IDENTIFIERS_REQUIRED"
	ErrViewIdentifiersRequired                     ErrorCode = "VIEW_IDENTIFIERS_REQUIRED"
	ErrUnsupportedViewEngagementKind               ErrorCode = "UNSUPPORTED_VIEW_ENGAGEMENT_KIND"
	ErrViewEngagementCounterMissing                ErrorCode = "VIEW_ENGAGEMENT_COUNTER_MISSING"
	ErrViewAggregateNotUpdated                     ErrorCode = "VIEW_AGGREGATE_NOT_UPDATED"
	ErrEngagementDetailRequired                    ErrorCode = "ENGAGEMENT_DETAIL_REQUIRED"
	ErrTransactionRequired                         ErrorCode = "TRANSACTION_REQUIRED"
	ErrEngagementRecordNotFound                    ErrorCode = "ENGAGEMENT_RECORD_NOT_FOUND"
	ErrUnknownEngagementKind                       ErrorCode = "UNKNOWN_ENGAGEMENT_KIND"
	ErrEngagementAmountInvalid                     ErrorCode = "ENGAGEMENT_AMOUNT_INVALID"
	ErrEngagementRepositoryNotConfigured           ErrorCode = "ENGAGEMENT_REPOSITORY_NOT_CONFIGURED"
	ErrEngagementDetailIDRequired                  ErrorCode = "ENGAGEMENT_DETAIL_ID_REQUIRED"
	ErrEngagementDetailOwnerInvalid                ErrorCode = "ENGAGEMENT_DETAIL_OWNER_INVALID"
	ErrEngagementDetailAggregateMismatch           ErrorCode = "ENGAGEMENT_DETAIL_AGGREGATE_MISMATCH"
	ErrEngagementCountInvalid                      ErrorCode = "ENGAGEMENT_COUNT_INVALID"
	ErrEngagementIdentifiersRequired               ErrorCode = "ENGAGEMENT_IDENTIFIERS_REQUIRED"
	ErrAmountEngagementToggleDenied                ErrorCode = "AMOUNT_ENGAGEMENT_TOGGLE_DENIED"
	ErrAmountEngagementRequiresAddTip              ErrorCode = "AMOUNT_ENGAGEMENT_REQUIRES_ADD_TIP"
	ErrAmountEngagementIdentifiersRequired         ErrorCode = "AMOUNT_ENGAGEMENT_IDENTIFIERS_REQUIRED"
	ErrEngagementAmountUnsupported                 ErrorCode = "ENGAGEMENT_AMOUNT_UNSUPPORTED"
	ErrFriendshipIDGeneratorRequired               ErrorCode = "FRIENDSHIP_ID_GENERATOR_REQUIRED"
	ErrMatchEngagementAggregateRequired            ErrorCode = "MATCH_ENGAGEMENT_AGGREGATE_REQUIRED"
	ErrVideoDimensionsProbeFailed                  ErrorCode = "VIDEO_DIMENSIONS_PROBE_FAILED"
	ErrMediaCommandFailed                          ErrorCode = "MEDIA_COMMAND_FAILED"
	ErrFFProbeOutputInvalid                        ErrorCode = "FF_PROBE_OUTPUT_INVALID"
	ErrMediaProcessingStatusFailed                 ErrorCode = "MEDIA_PROCESSING_STATUS_FAILED"
	ErrUploadRootResolveFailed                     ErrorCode = "UPLOAD_ROOT_RESOLVE_FAILED"
	ErrUploadRootSymlinksResolveFailed             ErrorCode = "UPLOAD_ROOT_SYMLINKS_RESOLVE_FAILED"
	ErrStoredUploadResolveFailed                   ErrorCode = "STORED_UPLOAD_RESOLVE_FAILED"
	ErrStoredUploadSymlinksResolveFailed           ErrorCode = "STORED_UPLOAD_SYMLINKS_RESOLVE_FAILED"
	ErrStoredUploadPathCompareFailed               ErrorCode = "STORED_UPLOAD_PATH_COMPARE_FAILED"
	ErrUploadPathOutsideRoot                       ErrorCode = "UPLOAD_PATH_OUTSIDE_ROOT"
	ErrStoredUploadRemoveFailed                    ErrorCode = "STORED_UPLOAD_REMOVE_FAILED"
	ErrReportTargetChanged                         ErrorCode = "REPORT_TARGET_CHANGED"
	ErrNewsPostIDRequired                          ErrorCode = "NEWS_POST_ID_REQUIRED"
	ErrPostPublicIDNotFound                        ErrorCode = "POST_PUBLIC_ID_NOT_FOUND"
	ErrPostUUIDNotFound                            ErrorCode = "POST_UUID_NOT_FOUND"
	ErrMatchNotificationParticipantsMissing        ErrorCode = "MATCH_NOTIFICATION_PARTICIPANTS_MISSING"
	ErrFollowNotificationParticipantsMissing       ErrorCode = "FOLLOW_NOTIFICATION_PARTICIPANTS_MISSING"
	ErrPushDeliveryFailed                          ErrorCode = "PUSH_DELIVERY_FAILED"
	ErrDistanceCursorRequired                      ErrorCode = "DISTANCE_CURSOR_REQUIRED"
	ErrInvalidEventRSVPStatus                      ErrorCode = "INVALID_EVENT_RSVP_STATUS"
	ErrPostUUIDLookupNotFound                      ErrorCode = "POST_UUID_LOOKUP_NOT_FOUND"
	ErrPostSlugRequired                            ErrorCode = "POST_SLUG_REQUIRED"
	ErrPostSlugLookupNotFound                      ErrorCode = "POST_SLUG_LOOKUP_NOT_FOUND"
	ErrPostPublicIDLookupNotFound                  ErrorCode = "POST_PUBLIC_ID_LOOKUP_NOT_FOUND"
	ErrParentPostEngagementNotUpdated              ErrorCode = "PARENT_POST_ENGAGEMENT_NOT_UPDATED"
	ErrCommentEngagementIdentifiersRequired        ErrorCode = "COMMENT_ENGAGEMENT_IDENTIFIERS_REQUIRED"
	ErrInvalidParentID                             ErrorCode = "INVALID_PARENT_ID"
	ErrPostLocationCoordinates                     ErrorCode = "POST_LOCATION_COORDINATES"
	ErrEventDateTimeRequired                       ErrorCode = "EVENT_DATE_TIME_REQUIRED"
	ErrEventDateTimeInvalid                        ErrorCode = "EVENT_DATE_TIME_INVALID"
	ErrEventPaidFlagInvalid                        ErrorCode = "EVENT_PAID_FLAG_INVALID"
	ErrEventOnlineFlagInvalid                      ErrorCode = "EVENT_ONLINE_FLAG_INVALID"
	ErrEventPriceInvalid                           ErrorCode = "EVENT_PRICE_INVALID"
	ErrPollChoiceNotFound                          ErrorCode = "POLL_CHOICE_NOT_FOUND"
	ErrCommentEngagementLookupIdentifiersRequired  ErrorCode = "COMMENT_ENGAGEMENT_LOOKUP_IDENTIFIERS_REQUIRED"
	ErrTipRepositoryNotConfigured                  ErrorCode = "TIP_REPOSITORY_NOT_CONFIGURED"
	ErrPillarAlreadyExists                         ErrorCode = "PILLAR_ALREADY_EXISTS"
	ErrClusterAlreadyExists                        ErrorCode = "CLUSTER_ALREADY_EXISTS"
	ErrSynonymAlreadyExists                        ErrorCode = "SYNONYM_ALREADY_EXISTS"
	ErrPrimarySynonymAlreadyExists                 ErrorCode = "PRIMARY_SYNONYM_ALREADY_EXISTS"
	ErrClusterSlugAmbiguous                        ErrorCode = "CLUSTER_SLUG_AMBIGUOUS"
	ErrPrivatePhotoNotificationParticipantsMissing ErrorCode = "PRIVATE_PHOTO_NOTIFICATION_PARTICIPANTS_MISSING"
	ErrProfilePhotoDestinationUnsupported          ErrorCode = "PROFILE_PHOTO_DESTINATION_UNSUPPORTED"
	ErrPrivatePhotoMediaRepositoryNotConfigured    ErrorCode = "PRIVATE_PHOTO_MEDIA_REPOSITORY_NOT_CONFIGURED"
	ErrPrivatePhotoProtectionRequired              ErrorCode = "PRIVATE_PHOTO_PROTECTION_REQUIRED"
	ErrPrivatePhotoPublicIDNotConfigured           ErrorCode = "PRIVATE_PHOTO_PUBLIC_ID_NOT_CONFIGURED"
	ErrPrivatePhotoDistinctUsersRequired           ErrorCode = "PRIVATE_PHOTO_DISTINCT_USERS_REQUIRED"
	ErrReferralRepositoryNotConfigured             ErrorCode = "REFERRAL_REPOSITORY_NOT_CONFIGURED"
	ErrReferralUserNotFound                        ErrorCode = "REFERRAL_USER_NOT_FOUND"
	ErrBroadcastUserIDRequired                     ErrorCode = "BROADCAST_USER_ID_REQUIRED"
	ErrUserUUIDRequiredForDelete                   ErrorCode = "USER_UUID_REQUIRED_FOR_DELETE"
	ErrLocationRequired                            ErrorCode = "LOCATION_REQUIRED"
	ErrPreferenceItemNotFound                      ErrorCode = "PREFERENCE_ITEM_NOT_FOUND"
	ErrStoredPreferenceBitIndexInvalid             ErrorCode = "STORED_PREFERENCE_BIT_INDEX_INVALID"
	ErrInvalidIP                                   ErrorCode = "INVALID_IP"
	ErrAmountZero                                  ErrorCode = "AMOUNT_ZERO"
	ErrAmountNegative                              ErrorCode = "AMOUNT_NEGATIVE"
)

var ErrorMessages = map[ErrorCode]string{
	// Repository and service errors.
	ErrAIProviderNotConfigured:                     "no AI provider is configured",
	ErrBroadcastPageSizeExceeded:                   "%w: page size exceeds %d",
	ErrBroadcastDescriptionTooLong:                 "%w: description is too long",
	ErrBroadcastIDRequired:                         "%w: broadcast ID is required",
	ErrBroadcastGuestIdentifiersRequired:           "%w: broadcast ID and stream client ID are required",
	ErrBroadcastLikeInputRequired:                  "%w: broadcast ID, viewer ID and a positive like count are required",
	ErrChatOpenMessageResultEmpty:                  "chat repository returned an empty open-message result",
	ErrPostLookupFailed:                            "GetPostByID error: %w",
	ErrMatchesRepositoryNotConfigured:              "matches repository is not configured",
	ErrModerationForbidden:                         "moderation access denied",
	ErrReportIDRequired:                            "report_id is required",
	ErrPostIDRequired:                              "post_id is required",
	ErrInvalidReportType:                           "invalid report content type",
	ErrPostSlugLookupFailed:                        "GetPostBySlug error: %w",
	ErrUserUUIDLookupFailed:                        "GetUserUUIDByPublicID error: %w",
	ErrUserMediaLookupFailed:                       "GetUserMedias error: %w",
	ErrRecentHashtagsLookupFailed:                  "GetRecentHashtags error: %w",
	ErrAuthenticatedUserRequired:                   "authenticated user is required",
	ErrPrivatePhotoOwnerRequired:                   "owner_id is required",
	ErrPrivatePhotoRequestRequired:                 "request_id is required",
	ErrPrivatePhotoViewerRequired:                  "viewer_id is required",
	ErrPrivatePhotoIDRequired:                      "photo_id is required",
	ErrPrivatePhotoUserNotFound:                    "private photo user not found",
	ErrPrivatePhotoNotFound:                        "private photo not found",
	ErrPrivatePhotoRequestNotFound:                 "private photo access request not found",
	ErrPrivatePhotoForbidden:                       "private photo action is forbidden",
	ErrPrivatePhotoSelfRequest:                     "cannot request your own private photos",
	ErrPrivatePhotoFilesRequired:                   "at least one private photo is required",
	ErrPrivatePhotoUploadLimit:                     "private photo upload limit exceeded",
	ErrPrivatePhotoLimitReached:                    "private photo album limit reached",
	ErrPhotoDestinationRequired:                    "destination must be public or private",
	ErrProfilePhotoRepositoryNotConfigured:         "profile photo repository is not configured",
	ErrSessionUserNotFound:                         "session user not found",
	ErrPrivateViewEngagements:                      "profile view engagements are private",
	ErrPrivateEngagements:                          "these user engagements are private",
	ErrUserIDRequired:                              "user_id is required",
	ErrCaptchaNotConfigured:                        "captcha verifier is not configured",
	ErrPasswordNotConfigured:                       "password hasher is not configured",
	ErrTokenIssuerNotConfigured:                    "token issuer is not configured",
	ErrPublicIDNotConfigured:                       "public ID generator is not configured",
	ErrInvalidCredentials:                          "invalid username/email/nickname or password",
	ErrInvalidCurrentPassword:                      "current password is invalid",
	ErrLegacyPasswordField:                         "password field is unsupported; use current_password, new_password, and new_password_confirmation",
	ErrLocationCoordinates:                         "both location latitude and longitude are required",
	ErrInvalidLocationOwner:                        "profile location contentable_type must be user",
	ErrInvalidCaptcha:                              "invalid captcha",
	ErrPasswordHashFailed:                          "failed to create hash password: %w",
	ErrReferralEmpty:                               "referral is empty",
	ErrAvatarUserLookupFailed:                      "failed to load user for avatar update: %w",
	ErrAvatarUploadFailed:                          "failed to upload avatar: %w",
	ErrAvatarUpdateFailed:                          "failed to update user avatar: %w",
	ErrRemoteImageFetcherNotConfigured:             "remote image fetcher is not configured",
	ErrCoverUserLookupFailed:                       "failed to load user for cover update: %w",
	ErrCoverUploadFailed:                           "failed to upload cover: %w",
	ErrCoverUpdateFailed:                           "failed to update user cover: %w",
	ErrUserAttributeUpsertFailed:                   "failed to upsert user attribute: %w",
	ErrBroadcastIntegrationDisabled:                "broadcast integration is disabled",
	ErrBroadcastProviderUnsupported:                "broadcast provider is unsupported",
	ErrBroadcastInvalidInput:                       "broadcast input is invalid",
	ErrBroadcastUpstream:                           "broadcast upstream request failed",
	ErrBroadcastUnauthorized:                       "broadcast operation requires authentication",
	ErrBroadcastSessionUnavailable:                 "broadcast session is unavailable",
	ErrBroadcastSessionForbidden:                   "broadcast session belongs to another user",
	ErrRegistrationPersistence:                     "registration could not be saved",
	ErrNotFound:                                    "resource not found",
	ErrPrivatePhotoAccessBlocked:                   "private photo users are blocked",
	ErrPaymentProcessingNotImplemented:             "payment processing is not implemented",
	ErrReportTargetNotFound:                        "report target not found",
	ErrReportNotFound:                              "report not found",
	ErrChatVisibilityIdentifiersRequired:           "chat visibility identifiers are required",
	ErrViewOnceMediaReadFailed:                     "read view-once media: %w",
	ErrEngagementDetailAlreadyExists:               "engagement detail already exists",
	ErrEngagementCounterMissing:                    "missing engagement counter for kind %s",
	ErrEngagementAggregateNotUpdated:               "engagement aggregate was not updated",
	ErrEngagementAggregateRequired:                 "engagement aggregate is required",
	ErrReciprocalInteractionConflict:               "reciprocal interaction detail conflict",
	ErrInteractionUserIdentifiersRequired:          "interaction user identifiers are required",
	ErrViewIdentifiersRequired:                     "view identifiers are required",
	ErrUnsupportedViewEngagementKind:               "unsupported view engagement kind: %s",
	ErrViewEngagementCounterMissing:                "missing counter for view engagement kind: %s",
	ErrViewAggregateNotUpdated:                     "view aggregate was not updated",
	ErrEngagementDetailRequired:                    "detail is nil",
	ErrTransactionRequired:                         "transaction is nil",
	ErrEngagementRecordNotFound:                    "engagement record not found for engagement_id: %s",
	ErrUnknownEngagementKind:                       "unknown engagement kind: %s",
	ErrEngagementAmountInvalid:                     "engagement amount must be a decimal string",
	ErrEngagementRepositoryNotConfigured:           "engagement repository is not configured",
	ErrEngagementDetailIDRequired:                  "engagement detail identifier is required",
	ErrEngagementDetailOwnerInvalid:                "engagement detail owner is invalid",
	ErrEngagementDetailAggregateMismatch:           "locked engagement detail does not belong to aggregate",
	ErrEngagementCountInvalid:                      "invalid engagement count %q: %w",
	ErrEngagementIdentifiersRequired:               "engagement identifiers are required",
	ErrAmountEngagementToggleDenied:                "amount engagement kinds cannot be toggled",
	ErrAmountEngagementRequiresAddTip:              "amount engagement kinds require AddTip",
	ErrAmountEngagementIdentifiersRequired:         "amount engagement identifiers are required",
	ErrEngagementAmountUnsupported:                 "engagement kind does not support an amount: %s",
	ErrFriendshipIDGeneratorRequired:               "friendship ID generator is required",
	ErrMatchEngagementAggregateRequired:            "match engagement aggregate is required",
	ErrVideoDimensionsProbeFailed:                  "probe video dimensions: %w",
	ErrMediaCommandFailed:                          "%s %v failed: %v - %s",
	ErrFFProbeOutputInvalid:                        "unexpected ffprobe output: %s",
	ErrMediaProcessingStatusFailed:                 "process media: %w; mark failed: %v",
	ErrUploadRootResolveFailed:                     "resolve upload root: %w",
	ErrUploadRootSymlinksResolveFailed:             "resolve upload root symlinks: %w",
	ErrStoredUploadResolveFailed:                   "resolve stored upload: %w",
	ErrStoredUploadSymlinksResolveFailed:           "resolve stored upload symlinks: %w",
	ErrStoredUploadPathCompareFailed:               "compare stored upload path: %w",
	ErrUploadPathOutsideRoot:                       "refusing to remove path outside upload root: %s",
	ErrStoredUploadRemoveFailed:                    "remove stored upload %s: %w",
	ErrReportTargetChanged:                         "report target changed during moderation",
	ErrNewsPostIDRequired:                          "post id is required",
	ErrPostPublicIDNotFound:                        "post with id %d not found",
	ErrPostUUIDNotFound:                            "post with id %s not found",
	ErrMatchNotificationParticipantsMissing:        "match notification participants missing",
	ErrFollowNotificationParticipantsMissing:       "follow notification participants missing",
	ErrPushDeliveryFailed:                          "push delivery failed for %d subscriptions",
	ErrDistanceCursorRequired:                      "distance cursor is required for location-based pagination",
	ErrInvalidEventRSVPStatus:                      "invalid event RSVP status: %s",
	ErrPostUUIDLookupNotFound:                      "%w: post with id %s",
	ErrPostSlugRequired:                            "post slug is required",
	ErrPostSlugLookupNotFound:                      "%w: post with slug %s",
	ErrPostPublicIDLookupNotFound:                  "%w: post with id %d",
	ErrParentPostEngagementNotUpdated:              "parent post engagement aggregate was not updated",
	ErrCommentEngagementIdentifiersRequired:        "comment engagement identifiers are required",
	ErrInvalidParentID:                             "invalid parentId %s: %w",
	ErrPostLocationCoordinates:                     "latitude and longitude must be provided together",
	ErrEventDateTimeRequired:                       "event date and time must be provided together",
	ErrEventDateTimeInvalid:                        "invalid event date or time: %w",
	ErrEventPaidFlagInvalid:                        "invalid event paid flag: %w",
	ErrEventOnlineFlagInvalid:                      "invalid event online flag: %w",
	ErrEventPriceInvalid:                           "invalid event price",
	ErrPollChoiceNotFound:                          "choice not found: %w",
	ErrCommentEngagementLookupIdentifiersRequired:  "comment engagement lookup identifiers are required",
	ErrTipRepositoryNotConfigured:                  "tip repository dependencies are not configured",
	ErrPillarAlreadyExists:                         "pillar already exists",
	ErrClusterAlreadyExists:                        "cluster already exists",
	ErrSynonymAlreadyExists:                        "synonym already exists",
	ErrPrimarySynonymAlreadyExists:                 "primary synonym already exists for this cluster",
	ErrClusterSlugAmbiguous:                        "cluster slug %q is ambiguous within pillar %s",
	ErrPrivatePhotoNotificationParticipantsMissing: "private photo notification participants missing",
	ErrProfilePhotoDestinationUnsupported:          "unsupported profile photo destination",
	ErrPrivatePhotoMediaRepositoryNotConfigured:    "private photo media repository is not configured",
	ErrPrivatePhotoProtectionRequired:              "private photo was not stored as protected media",
	ErrPrivatePhotoPublicIDNotConfigured:           "private photo public ID generator is not configured",
	ErrPrivatePhotoDistinctUsersRequired:           "two different private photo users are required",
	ErrReferralRepositoryNotConfigured:             "referral repository is not configured",
	ErrReferralUserNotFound:                        "referral user not found",
	ErrBroadcastUserIDRequired:                     "broadcast user ID is required",
	ErrUserUUIDRequiredForDelete:                   "missing user uuid for delete",
	ErrLocationRequired:                            "location is required",
	ErrPreferenceItemNotFound:                      "preference item not found",
	ErrStoredPreferenceBitIndexInvalid:             "invalid stored preference bit index",
	ErrInvalidIP:                                   "invalid IP",
	ErrAmountZero:                                  "amount cannot be zero",
	ErrAmountNegative:                              "amount cannot be negative",

	ErrInvalidForm:          "Invalid form data.",
	ErrUnknown:              "An unknown error occurred.",
	ErrFileNotFound:         "The requested file could not be found.",
	ErrPermissionDenied:     "Permission denied.",
	ErrInvalidInput:         "Invalid input provided.",
	ErrNetworkError:         "A network error occurred.",
	ErrRequestTimeout:       "The request timed out.",
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
	ErrUserExists:            "User already exists.",
	ErrUserNotFound:          "User not found.",
	ErrUsernameRequired:      "Username is required.",
	ErrUsernameInvalidFormat: "Username may only contain lowercase letters, digits, and underscores, and must be 3–30 characters.",
	ErrUserUnauthorized:      "User is not authorized to perform this action.",
	ErrUsernameTaken:         "Username already taken.",

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
	ErrPostUpdateFailed:        "Failed to update post.",
	ErrPostSaveFailed:          "Failed to save post.",
	ErrPostUnsaveFailed:        "Failed to unsave post.",
	ErrPostShareFailed:         "Failed to share post.",
	ErrPostReportFailed:        "Failed to report post.",
	ErrPostLikeFailed:          "Failed to like post.",
	ErrPostDislikeFailed:       "Failed to dislike post.",
	ErrPostUnlikeFailed:        "Failed to unlike post.",
	ErrPostBookmarkFailed:      "Failed to bookmark post.",
	ErrPostUnbookmarkFailed:    "Failed to unbookmark post.",
	ErrPostNotFound:            "Post not found.",
	ErrPostDeleteDenied:        "You are not allowed to delete this post.",
	ErrPostDeleteFailed:        "Failed to delete the post.",
	ErrInsufficientBalance:     "Insufficient balance.",
	ErrInvalidAmount:           "Invalid amount provided.",
	ErrPostCreateFailed:        "Failed to create post",
	ErrFailedToBookmarkPost:    "Failed to bookmark post",
	ErrFailedToDislikePost:     "Failed to dislike post",
	ErrFailedToLikePost:        "Failed to like post",
	ErrFailedToUnlikePost:      "Failed to unlike post",
	ErrFailedToReportPost:      "Failed to report post",
	ErrFailedToSavePost:        "Failed to save post",
	ErrFailedToUnsavePost:      "Failed to unsave post",
	ErrFailedToSharePost:       "Failed to share post",
	ErrCannotTipOwnPost:        "You cannot tip your own post.",
	ErrFailedToBananaPost:      "Failed to banana post",
	ErrPostSubscribeFailed:     "Failed to subscribe to post",
	ErrPostUnsubscribeFailed:   "Failed to unsubscribe from post",
	ErrFailedToSubscribePost:   "Failed to subscribe to post",
	ErrFailedToUnsubscribePost: "Failed to unsubscribe from post",
	// Place
	ErrPlaceCreateFailed: "Failed to create place",
	ErrPlaceNotFound:     "Place not found",
	ErrPlaceDeleteDenied: "You are not allowed to delete this place",
	ErrPlaceDeleteFailed: "Failed to delete the place",

	// Chat
	ErrGroupOwnerCannotLeave:            "The group owner cannot leave while other participants remain.",
	ErrSelfChatNotAllowed:               "You cannot create a chat with yourself.",
	ErrUnsupportedChatType:              "Unsupported chat type.",
	ErrInvalidParticipantID:             "Invalid participant ID.",
	ErrInvalidParticipantsLength:        "Invalid participants length.",
	ErrFailedToLoadMessages:             "Failed to load messages. Please try again later.",
	ErrInvalidChatID:                    "The provided chat ID is invalid.",
	ErrInvalidMessageID:                 "The provided message ID is invalid.",
	ErrFailedToPinMessage:               "Failed to pin message.",
	ErrFailedToUnpinMessage:             "Failed to unpin the message. Please try again later.",
	ErrFailedToDeleteMessage:            "Failed to delete the message.",
	ErrFailedToDeleteMessageForUser:     "Failed to delete the message for the user.",
	ErrFailedToDeleteMessageForAll:      "Failed to delete the message for all.",
	ErrFailedToDeleteChatForAll:         "Failed to delete the chat for all.",
	ErrFailedToDeleteChatForUser:        "Failed to delete the chat for the user.",
	ErrFailedToDeleteChatHistoryForUser: "Failed to delete the chat history for the user.",
	ErrFailedToDeleteChatHistoryForAll:  "Failed to delete the chat history for all.",

	// Location / Geo
	ErrInvalidLatitude:  "Invalid latitude.",
	ErrInvalidLongitude: "Invalid longitude.",

	// Notifications
	ErrNotificationsFetchFailed: "Failed to fetch notifications.",

	// Payment
	ErrPaymentMethodNotFound:    "Payment method not found.",
	ErrPaymentMethodFetchFailed: "Failed to fetch payment method.",

	// News
	ErrNewsCreateFailed: "Failed to create news",

	// Others
	ErrOGFetchFailed:   "Failed to fetch OG data from the URL.",
	ErrHTMLParseFailed: "Failed to parse HTML from the URL.",

	// CheckIn
	ErrFailedtoCheckIn: "Failed to check in",
}

func (e ErrorCode) String() string {
	if msg, ok := ErrorMessages[e]; ok {
		return msg
	}
	return ErrorMessages[ErrUnknown]
}

// Error lets callers return a registered code without allocating a new error.
func (e ErrorCode) Error() string {
	return e.String()
}

// Format applies the registered message template while preserving both the
// error code and any causes wrapped with %w for errors.Is and errors.As.
func (e ErrorCode) Format(args ...any) error {
	return &formattedError{code: e, cause: fmt.Errorf(e.String(), args...)}
}

type formattedError struct {
	code  ErrorCode
	cause error
}

func (e *formattedError) Error() string {
	return e.cause.Error()
}

func (e *formattedError) Unwrap() []error {
	return []error{e.code, e.cause}
}
