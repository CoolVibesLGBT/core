package ports

import (
	"context"
	legacyviews "core/application/legacyviews"
	"core/application/types"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/notifications"
	"core/models/payment"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"core/models/taxonomy"
	"core/models/utils"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UserRepository interface {
	UserProfileWriter
	ExistsByNameOrMail(input string) (bool, error)
	ExistsByUsername(username string) (bool, error)
	ExistsByEmail(email string) (bool, error)
	Create(user *models.User) error
	GetByID(userID uuid.UUID) (*models.User, error)
	GetByNameOrMailWithoutRelations(input string) (*models.User, error)
	GetByUserNameOrEmailOrUsername(input string) (*models.User, error)
	GetUserByPublicIdWithoutRelations(filters types.Filter) (*models.User, error)
	GetUserByUUIDdWithoutRelations(filters types.Filter) (*models.User, error)
	GetUserUUIDByPublicID(publicID int64) (uuid.UUID, error)
	GetUsersStartingWith(letter string, limit int) ([]models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(filters types.Filter) error
	UpsertLocation(location *utils.Location) error
	UpsertUserPreference(ctx context.Context, userID uuid.UUID, preferenceItemID string, enabled bool) error
	FetchNearbyUsers(filters types.Filter) ([]types.NearbyUser, *float64, error)
	FetchLiveUsers(filters types.Filter) ([]*models.User, error)
	FetchUserNotifications(ctx context.Context, authUser *models.User, cursor *time.Time, limit int) ([]*notifications.Notification, *time.Time, error)
	GetPreferences() (*models.PreferencesData, error)
	AddReferral(ctx context.Context, referrerID uuid.UUID, referredUserID uuid.UUID, rewardAmount decimal.Decimal) (*decimal.Decimal, error)
	Report(ctx context.Context, userPublicID int64, kind string, description string, authUser *models.User) error
}

// PublicUserProfileReader and PublicUserSearchReader are query-side ports.
// They return application read models instead of leaking persistence entities
// through public HTTP actions.
type PublicUserProfileReader interface {
	FetchPublicUserProfile(ctx context.Context, username string) (*types.PublicUserProfile, error)
}

type PublicUserSearchReader interface {
	SearchPublicUsers(filters types.Filter) ([]types.PublicUserSummary, error)
}

type MediaRepository interface {
	AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userID uuid.UUID, role media.MediaRole, file UploadedFile) (*media.Media, error)
}

type MediaFileAccess struct {
	StoragePath     string
	IsPublic        bool
	OwnerID         uuid.UUID
	OwnerType       string
	Role            string
	PostID          *uuid.UUID
	PostAuthorID    *uuid.UUID
	PostKind        string
	ContentableType string
	ChatID          *uuid.UUID
	Published       bool
	Audience        *string
}

type MediaAccessPrincipal struct {
	ID   uuid.UUID
	Role string
}

type MediaAccessRepository interface {
	FindMediaFileAccess(ctx context.Context, storagePrefix string) (MediaFileAccess, error)
	FindMediaAccessPrincipal(ctx context.Context, publicID int64) (MediaAccessPrincipal, error)
	IsActiveChatParticipant(ctx context.Context, chatID, userID uuid.UUID) (bool, error)
}

type PostRepository interface {
	CreateContentablePost(ctx context.Context, form FormData, author *models.User, contentableType string, contentableID *uuid.UUID) (*post.Post, error)
	GetPostByID(id uuid.UUID) (*post.Post, error)
	GetPostByIDIncludingUnpublished(id uuid.UUID) (*post.Post, error)
	GetPostBySlug(filters types.Filter) (*post.Post, error)
	GetPostByPublicID(id int64) (*post.Post, error)
	GetTimeline(filters types.Filter) (legacyviews.TimelineResult, error)
	FindPostsByKind(filters types.Filter) (legacyviews.PostsResult, error)
	GetPostsByKind(filters types.Filter) (legacyviews.PostsResult, error)
	GetUserPosts(userID uuid.UUID, filters types.Filter) ([]post.Post, error)
	GetUserPostReplies(filters types.Filter) ([]post.Post, error)
	GetUserMedias(filters types.Filter) ([]legacyviews.MediaWithUser, *int64, error)
	GetRecentHashtags(filters types.Filter) ([]types.HashtagStats, error)
	GetTimelineVibes(filters types.Filter) (legacyviews.TimelineResult, error)
	Vote(ctx context.Context, choiceID uuid.UUID, weight int, rank int, userID uuid.UUID) error
	SetEventRSVP(ctx context.Context, postPublicID int64, userID uuid.UUID, status *postpayloads.EventAttendanceStatus) (*postpayloads.EventRSVPResult, error)
	Like(filters types.Filter) error
	Dislike(filters types.Filter) error
	Banana(filters types.Filter) error
	Delete(filters types.Filter) error
	Report(ctx context.Context, postID int64, kind string, description string, authUser *models.User) error
	Bookmark(filters types.Filter) error
	View(filters types.Filter) (bool, error)
	Tip(ctx context.Context, postID int64, authUser *models.User, amount decimal.Decimal, idempotencyKey domainwallet.IdempotencyKey) (*decimal.Decimal, error)
	GetPillarsWithClusters(filters types.Filter) ([]taxonomy.Pillar, error)
}

type EngagementRepository interface {
	ApplyReciprocalUserInteraction(ctx context.Context, actorID uuid.UUID, targetID uuid.UUID, intent domainuser.InteractionStateIntent) (domainuser.InteractionStateTransition, error)
	ToggleEngagement(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) (bool, error)
	RecordViewOnce(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) (bool, error)
	HasUserEngaged(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind) (bool, error)
	GetEngagements(ctx context.Context, contentableType models.EngagementContentableType, contentableID uuid.UUID, engagementKind models.EngagementKind, cursor *time.Time, limit int) ([]models.EngagementDetail, *time.Time, error)
}

type NotificationRepository interface {
	SendNotificationToUser(sender models.User, receiver models.User, notificationType string, notificationTitle string, notificationMessage string, payload notifications.NotificationPayload) error
	FetchAndMarkShownNotifications(userID uuid.UUID, limit int) ([]notifications.Notification, error)
}

type NotificationReadMarker interface {
	MarkNotificationAsRead(notificationID string) error
}

type NewsRepository interface {
	GetNews(filters types.Filter) (legacyviews.PostsResult, error)
	Get(filters types.Filter) (*post.Post, error)
	IsNewsExists(filters types.Filter) (bool, error)
	Categories(filters types.Filter) ([]*taxonomy.Pillar, error)
	Category(filters types.Filter) (*taxonomy.Pillar, error)
}

type PlaceRepository interface {
	GetNearByPlaces(filters types.Filter) ([]*post.Post, types.Cursor, error)
	GetPlacesCategories(filters types.Filter) ([]taxonomy.Pillar, error)
}

type ListingRepository interface {
	GetClassified(filters types.Filter) (*post.Post, error)
	GetJobOffers(filters types.Filter) (legacyviews.PostsResult, error)
	GetJobSearches(filters types.Filter) (legacyviews.PostsResult, error)
}

type MatchesRepository interface {
	GetUnseenUsers(ctx context.Context, userID uuid.UUID, limit int) ([]types.PublicUserSummary, error)
	RecordView(ctx context.Context, userID uuid.UUID, targetID uuid.UUID, reaction domainuser.MatchReaction) (bool, error)
	GetMatchesAfter(ctx context.Context, userID uuid.UUID, cursor *types.MatchListCursor, limit int) (types.MatchListPage, error)
	GetLikesAfter(ctx context.Context, userID uuid.UUID, cursor *types.MatchListCursor, limit int) (types.MatchListPage, error)
	GetPassesAfter(ctx context.Context, userID uuid.UUID, cursor *types.MatchListCursor, limit int) (types.MatchListPage, error)
}

// UserPublicIDResolver is the narrow identity lookup needed by interaction
// commands. Services must not depend on the full persistence-shaped user
// repository merely to translate a public identifier.
type UserPublicIDResolver interface {
	GetUserUUIDByPublicID(publicID int64) (uuid.UUID, error)
}

type ChatRepository interface {
	SendTypingEvent(chatID, userID uuid.UUID, typing bool) error
	GetPrivateChatBetweenUsers(fromUser, toUser uuid.UUID) (*chat.Chat, error)
	CreatePrivateChat(fromUser, toUser uuid.UUID) (*chat.Chat, error)
	ListChats(ctx context.Context, query ChatListQuery) (ChatListPage, error)
	AddMessageToChat(ctx context.Context, form FormData, author *models.User) (*post.Post, error)
	ListChatMessages(ctx context.Context, query ChatMessageListQuery) (ChatMessageListPage, error)
	PinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	UnpinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	DeleteMessageForUser(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	DeleteMessageForAll(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	DeleteChatForUser(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error
	DeleteChatForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error
	DeleteChat(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error
	DeleteMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error
	DeleteChatHistoryForUser(ctx context.Context, authUser *models.User, chatID uuid.UUID) error
	DeleteChatHistoryForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error
	MarkChatMessageRead(ctx context.Context, authUser *models.User, chatID uuid.UUID, messages []uuid.UUID) error
	OpenMessage(ctx context.Context, authUser *models.User, chatID, messageID uuid.UUID, now time.Time) (*chat.OpenMessageResult, error)
	ExpireMessages(ctx context.Context, now time.Time, limit int) ([]chat.ExpiredMessage, error)
}

type PaymentRepository interface {
	Deposit(authUser models.User) error
	Withdraw(authUser models.User) error
	Transactions(authUser models.User) error
}

type RealtimeNotifier interface {
	BroadcastToRoom(namespace string, room string, event string, msg string) error
}

type SystemRepository interface {
	GetPreferences(ctx context.Context) (models.PreferencesData, error)
	GetEventKinds(ctx context.Context) ([]postpayloads.EventKind, error)
	GetReportKinds(ctx context.Context) ([]models.ReportKind, error)
	GetVapidPublicKey(ctx context.Context) (string, error)
	SaveVapidSubscription(ctx context.Context, userID uuid.UUID, subscription models.Subscription) error
	GetPaymentMethod(ctx context.Context) (*payment.PaymentMethod, error)
}

type SitemapRepository interface {
	GenerateSitemapIndex(baseURL string) (string, error)
	GeneratePostSitemap(ctx context.Context, baseURL string) ([]byte, error)
	GenerateNewsSitemap(ctx context.Context, baseURL string) ([]byte, error)
	GenerateCategoriesSitemap(ctx context.Context, baseURL string) ([]byte, error)
	GenerateImageSitemap(ctx context.Context, frontendURL string, apiURL string) ([]byte, error)
	GenerateVideoSitemap(ctx context.Context, frontendURL string, apiURL string) ([]byte, error)
}
