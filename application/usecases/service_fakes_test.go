package usecases

import (
	"context"
	"core/application/ports"
	domainevents "core/domain/events"
	"core/models"
	"core/models/chat"
	"core/models/media"
	"core/models/notifications"
	"core/models/post"
	"core/models/taxonomy"
	"core/models/utils"
	"core/types"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeCaptchaVerifier struct {
	valid bool
	err   error
	token string
}

func (f *fakeCaptchaVerifier) VerifyCaptcha(ctx context.Context, response string) (bool, error) {
	f.token = response
	return f.valid, f.err
}

type fakePasswordHasher struct {
	hashErr    error
	compareOK  bool
	compareErr error
}

func (f *fakePasswordHasher) HashPassword(raw string) (string, error) {
	if f.hashErr != nil {
		return "", f.hashErr
	}
	return "hashed:" + raw, nil
}

func (f *fakePasswordHasher) ComparePassword(hashed string, raw string) (bool, error) {
	return f.compareOK, f.compareErr
}

type fakeTokenIssuer struct {
	token string
	err   error
}

func (f *fakeTokenIssuer) GenerateUserToken(userID uuid.UUID, publicID int64) (string, error) {
	return f.token, f.err
}

type fakePublicIDGenerator struct {
	next int64
}

func (f *fakePublicIDGenerator) GeneratePublicID() int64 {
	return f.next
}

type fakeEventPublisher struct {
	events []domainevents.Event
	err    error
}

func (f *fakeEventPublisher) Publish(ctx context.Context, events ...domainevents.Event) error {
	f.events = append(f.events, events...)
	return f.err
}

type fakeUserRepository struct {
	ports.UserRepository

	exists              bool
	existsErr           error
	createErr           error
	created             *models.User
	byID                map[uuid.UUID]*models.User
	byNameOrMail        *models.User
	byNameOrMailErr     error
	byPublicIDDirect    map[int64]*models.User
	byPublicID          map[int64]*models.User
	byUUID              map[uuid.UUID]*models.User
	byUsername          *models.User
	byUsernameErr       error
	userUUIDByPublicID  map[int64]uuid.UUID
	usersStartingWith   []models.User
	updated             *models.User
	updateErr           error
	upsertLocation      *utils.Location
	upsertPreference    *upsertPreferenceCall
	deletedFilter       types.Filter
	fetchNearbyFilter   types.Filter
	fetchNearbyUsers    []*models.User
	fetchNearbyDistance *float64
	fetchLiveFilter     types.Filter
	fetchLiveUsers      []*models.User
	notifications       []*notifications.Notification
	nextNotification    *time.Time
	referralReferrerID  uuid.UUID
	referralReferredID  uuid.UUID
	referralReward      decimal.Decimal
}

type upsertPreferenceCall struct {
	user             models.User
	preferenceItemID string
	bitIndex         string
	enabled          bool
}

func (r *fakeUserRepository) ExistsByNameOrMail(input string) (bool, error) {
	return r.exists, r.existsErr
}

func (r *fakeUserRepository) Create(user *models.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = user
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*models.User)
	}
	r.byID[user.ID] = user
	return nil
}

func (r *fakeUserRepository) GetByID(userID uuid.UUID) (*models.User, error) {
	if user, ok := r.byID[userID]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (r *fakeUserRepository) GetByUserNameOrEmailOrUsername(input string) (*models.User, error) {
	if r.byUsernameErr != nil {
		return nil, r.byUsernameErr
	}
	if r.byUsername == nil {
		return nil, errors.New("user not found")
	}
	return r.byUsername, nil
}

func (r *fakeUserRepository) GetUserByPublicId(publicID int64) (*models.User, error) {
	if user, ok := r.byPublicIDDirect[publicID]; ok {
		return user, nil
	}
	if user, ok := r.byPublicID[publicID]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (r *fakeUserRepository) GetByNameOrMailWithoutRelations(input string) (*models.User, error) {
	if r.byNameOrMailErr != nil {
		return nil, r.byNameOrMailErr
	}
	if r.byNameOrMail != nil {
		return r.byNameOrMail, nil
	}
	return nil, errors.New("user not found")
}

func (r *fakeUserRepository) GetUserByPublicIdWithoutRelations(filters types.Filter) (*models.User, error) {
	if user, ok := r.byPublicID[filters.UserID]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (r *fakeUserRepository) GetUserByUUIDdWithoutRelations(filters types.Filter) (*models.User, error) {
	if user, ok := r.byUUID[filters.UserUUID]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (r *fakeUserRepository) GetUserUUIDByPublicID(publicID int64) (uuid.UUID, error) {
	if id, ok := r.userUUIDByPublicID[publicID]; ok {
		return id, nil
	}
	return uuid.Nil, errors.New("user not found")
}

func (r *fakeUserRepository) UpdateUser(user *models.User) error {
	r.updated = user
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*models.User)
	}
	r.byID[user.ID] = user
	return r.updateErr
}

func (r *fakeUserRepository) DeleteUser(filters types.Filter) error {
	r.deletedFilter = filters
	return nil
}

func (r *fakeUserRepository) UpsertLocation(location *utils.Location) error {
	r.upsertLocation = location
	return nil
}

func (r *fakeUserRepository) UpsertUserPreference(ctx context.Context, user models.User, preferenceItemID string, bitIndex string, enabled bool) error {
	r.upsertPreference = &upsertPreferenceCall{
		user: user, preferenceItemID: preferenceItemID, bitIndex: bitIndex, enabled: enabled,
	}
	return nil
}

func (r *fakeUserRepository) FetchNearbyUsers(filters types.Filter) ([]*models.User, *float64, error) {
	r.fetchNearbyFilter = filters
	return r.fetchNearbyUsers, r.fetchNearbyDistance, nil
}

func (r *fakeUserRepository) FetchLiveUsers(filters types.Filter) ([]*models.User, error) {
	r.fetchLiveFilter = filters
	return r.fetchLiveUsers, nil
}

func (r *fakeUserRepository) GetUsersStartingWith(letter string, limit int) ([]models.User, error) {
	return r.usersStartingWith, nil
}

func (r *fakeUserRepository) FetchUserNotifications(ctx context.Context, authUser *models.User, cursor *time.Time, limit int) ([]*notifications.Notification, *time.Time, error) {
	return r.notifications, r.nextNotification, nil
}

func (r *fakeUserRepository) GetPreferences() (*models.PreferencesData, error) {
	return &models.PreferencesData{}, nil
}

func (r *fakeUserRepository) AddReferral(ctx context.Context, referrerID uuid.UUID, referredUserID uuid.UUID, rewardAmount decimal.Decimal) (*decimal.Decimal, error) {
	r.referralReferrerID = referrerID
	r.referralReferredID = referredUserID
	r.referralReward = rewardAmount
	balance := decimal.NewFromInt(1)
	return &balance, nil
}

type fakeEngagementRepository struct {
	ports.EngagementRepository

	toggles []engagementToggle
	has     map[models.EngagementKind]bool
}

type engagementToggle struct {
	engagerID       uuid.UUID
	engageeID       uuid.UUID
	kind            models.EngagementKind
	contentableID   uuid.UUID
	contentableType models.EngagementContentableType
}

func (r *fakeEngagementRepository) ToggleEngagement(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) (bool, error) {
	r.toggles = append(r.toggles, engagementToggle{
		engagerID: engagerID, engageeID: engageeID, kind: kind, contentableID: contentableID, contentableType: contentableType,
	})
	return true, nil
}

func (r *fakeEngagementRepository) HasUserEngaged(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind) (bool, error) {
	if r.has == nil {
		return false, nil
	}
	return r.has[kind], nil
}

func (r *fakeEngagementRepository) GetEngagements(ctx context.Context, contentableType models.EngagementContentableType, contentableID uuid.UUID, engagementKind models.EngagementKind, cursor *time.Time, limit int) ([]models.EngagementDetail, *time.Time, error) {
	return nil, nil, nil
}

type fakeNotificationRepository struct {
	ports.NotificationRepository
	sent []string
}

func (r *fakeNotificationRepository) SendNotificationToUser(sender models.User, receiver models.User, notificationType string, notificationTitle string, notificationMessage string, payload notifications.NotificationPayload) error {
	r.sent = append(r.sent, notificationType)
	return nil
}

type fakePostRepository struct {
	ports.PostRepository

	createContentableType string
	createForm            ports.FormData
	createAuthor          *models.User
	createdPostID         uuid.UUID
	getPostByID           *post.Post
	getPostBySlugFilter   types.Filter
	getPostBySlug         *post.Post
	getPostByPublicID     *post.Post
	getPostByPublicIDArg  int64
	timelineFilter        types.Filter
	userPostsUserID       uuid.UUID
	userPostsFilter       types.Filter
	userPosts             []post.Post
	userRepliesFilter     types.Filter
	userReplies           []post.Post
	userMediasFilter      types.Filter
	userMedias            []types.MediaWithUser
	userMediasCursor      *int64
	recentHashtagsFilter  types.Filter
	recentHashtags        []types.HashtagStats
	timeline              types.TimelineResult
	timelineVibesFilter   types.Filter
	timelineVibes         types.TimelineResult
	searchFilter          types.Filter
	search                types.PostsResult
	voteChoiceID          uuid.UUID
	voteWeight            int
	voteRank              int
	voteUserID            uuid.UUID
	likeFilter            types.Filter
	dislikeFilter         types.Filter
	bananaFilter          types.Filter
	deleteFilter          types.Filter
	reportPostID          int64
	reportKind            string
	reportDescription     string
	reportAuthUser        *models.User
	bookmarkFilter        types.Filter
	viewFilter            types.Filter
	tipPostID             int64
	tipAuthUser           *models.User
	tipAmount             decimal.Decimal
	tipBalance            decimal.Decimal
	pillarsFilter         types.Filter
	pillars               []taxonomy.Pillar
}

func (r *fakePostRepository) CreateContentablePost(ctx context.Context, form ports.FormData, author *models.User, contentableType string, contentableID *uuid.UUID) (*post.Post, error) {
	r.createContentableType = contentableType
	r.createForm = form
	r.createAuthor = author
	if r.createdPostID == uuid.Nil {
		r.createdPostID = uuid.New()
	}
	return &post.Post{ID: r.createdPostID, PostKind: post.PostKind(contentableType)}, nil
}

func (r *fakePostRepository) GetPostByID(id uuid.UUID) (*post.Post, error) {
	if r.getPostByID != nil {
		return r.getPostByID, nil
	}
	kind := post.PostKindPost
	if r.createContentableType != "" {
		kind = post.PostKind(r.createContentableType)
	}
	return &post.Post{ID: id, PostKind: kind}, nil
}

func (r *fakePostRepository) GetPostBySlug(filters types.Filter) (*post.Post, error) {
	r.getPostBySlugFilter = filters
	if r.getPostBySlug != nil {
		return r.getPostBySlug, nil
	}
	return &post.Post{ID: uuid.New(), Slug: filters.Slug}, nil
}

func (r *fakePostRepository) GetPostByPublicID(id int64) (*post.Post, error) {
	r.getPostByPublicIDArg = id
	if r.getPostByPublicID != nil {
		return r.getPostByPublicID, nil
	}
	return &post.Post{ID: uuid.New(), PublicID: id}, nil
}

func (r *fakePostRepository) GetTimeline(filters types.Filter) (types.TimelineResult, error) {
	r.timelineFilter = filters
	return r.timeline, nil
}

func (r *fakePostRepository) FindPostsByKind(filters types.Filter) (types.PostsResult, error) {
	r.searchFilter = filters
	return r.search, nil
}

func (r *fakePostRepository) GetUserPosts(userID uuid.UUID, filters types.Filter) ([]post.Post, error) {
	r.userPostsUserID = userID
	r.userPostsFilter = filters
	return r.userPosts, nil
}

func (r *fakePostRepository) GetUserPostReplies(filters types.Filter) ([]post.Post, error) {
	r.userRepliesFilter = filters
	return r.userReplies, nil
}

func (r *fakePostRepository) GetUserMedias(filters types.Filter) ([]types.MediaWithUser, *int64, error) {
	r.userMediasFilter = filters
	return r.userMedias, r.userMediasCursor, nil
}

func (r *fakePostRepository) GetRecentHashtags(filters types.Filter) ([]types.HashtagStats, error) {
	r.recentHashtagsFilter = filters
	return r.recentHashtags, nil
}

func (r *fakePostRepository) GetTimelineVibes(filters types.Filter) (types.TimelineResult, error) {
	r.timelineVibesFilter = filters
	return r.timelineVibes, nil
}

func (r *fakePostRepository) Vote(ctx context.Context, choiceID uuid.UUID, weight int, rank int, userID uuid.UUID) error {
	r.voteChoiceID = choiceID
	r.voteWeight = weight
	r.voteRank = rank
	r.voteUserID = userID
	return nil
}

func (r *fakePostRepository) Like(filters types.Filter) error {
	r.likeFilter = filters
	return nil
}

func (r *fakePostRepository) Dislike(filters types.Filter) error {
	r.dislikeFilter = filters
	return nil
}

func (r *fakePostRepository) Banana(filters types.Filter) error {
	r.bananaFilter = filters
	return nil
}

func (r *fakePostRepository) Delete(filters types.Filter) error {
	r.deleteFilter = filters
	return nil
}

func (r *fakePostRepository) Report(ctx context.Context, postID int64, kind string, description string, authUser *models.User) error {
	r.reportPostID = postID
	r.reportKind = kind
	r.reportDescription = description
	r.reportAuthUser = authUser
	return nil
}

func (r *fakePostRepository) Bookmark(filters types.Filter) error {
	r.bookmarkFilter = filters
	return nil
}

func (r *fakePostRepository) View(filters types.Filter) error {
	r.viewFilter = filters
	return nil
}

func (r *fakePostRepository) Tip(ctx context.Context, postID int64, authUser *models.User, amount decimal.Decimal) (*decimal.Decimal, error) {
	r.tipPostID = postID
	r.tipAuthUser = authUser
	r.tipAmount = amount
	if r.tipBalance.IsZero() {
		r.tipBalance = decimal.NewFromInt(10)
	}
	return &r.tipBalance, nil
}

func (r *fakePostRepository) GetPillarsWithClusters(filters types.Filter) ([]taxonomy.Pillar, error) {
	r.pillarsFilter = filters
	return r.pillars, nil
}

func (r *fakePostRepository) GetPostsByKind(filters types.Filter) (types.PostsResult, error) {
	return types.PostsResult{Posts: []post.Post{{PostKind: filters.PostKind}}}, nil
}

type fakeChatRepository struct {
	ports.ChatRepository

	privateChat        *chat.Chat
	privateChatErr     error
	createdPrivate     *chat.Chat
	chatsByUserID      []chat.Chat
	chatsByUserIDArg   uuid.UUID
	message            *post.Post
	messagesByChatID   []post.Post
	messageUserID      uuid.UUID
	messageChatID      uuid.UUID
	typingPayload      map[string]interface{}
	action             string
	actionAuthUser     *models.User
	actionChatID       uuid.UUID
	actionUserID       uuid.UUID
	actionMessageID    uuid.UUID
	markedReadChatID   uuid.UUID
	markedReadMessages []uuid.UUID
}

func (r *fakeChatRepository) SendTypingEvent(chatID, userID uuid.UUID, typing bool) (map[string]interface{}, error) {
	if r.typingPayload != nil {
		return r.typingPayload, nil
	}
	return map[string]interface{}{"chat_id": chatID.String(), "user_id": userID.String(), "typing": typing}, nil
}

func (r *fakeChatRepository) GetPrivateChatBetweenUsers(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	if r.privateChatErr != nil {
		return nil, r.privateChatErr
	}
	if r.privateChat != nil {
		return r.privateChat, nil
	}
	return nil, errors.New("not found")
}

func (r *fakeChatRepository) CreatePrivateChat(fromUser, toUser uuid.UUID) (*chat.Chat, error) {
	r.createdPrivate = &chat.Chat{ID: uuid.New(), CreatorID: fromUser}
	return r.createdPrivate, nil
}

func (r *fakeChatRepository) GetChatsByUserID(userID uuid.UUID) ([]chat.Chat, error) {
	r.chatsByUserIDArg = userID
	return r.chatsByUserID, nil
}

func (r *fakeChatRepository) AddMessageToChat(ctx context.Context, form ports.FormData, author *models.User) (*post.Post, error) {
	if r.message != nil {
		return r.message, nil
	}
	chatID := uuid.New()
	return &post.Post{ID: uuid.New(), AuthorID: author.ID, ContentableID: &chatID}, nil
}

func (r *fakeChatRepository) GetMessagesByChatID(userID uuid.UUID, chatID uuid.UUID) ([]post.Post, error) {
	r.messageUserID = userID
	r.messageChatID = chatID
	return r.messagesByChatID, nil
}

func (r *fakeChatRepository) PinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	r.recordChatAction("pin", authUser, chatID, userID, messageID)
	return nil
}

func (r *fakeChatRepository) UnpinMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	r.recordChatAction("unpin", authUser, chatID, userID, messageID)
	return nil
}

func (r *fakeChatRepository) DeleteMessageForUser(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	r.recordChatAction("delete_message_for_user", authUser, chatID, userID, messageID)
	return nil
}

func (r *fakeChatRepository) DeleteMessageForAll(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	r.recordChatAction("delete_message_for_all", authUser, chatID, userID, messageID)
	return nil
}

func (r *fakeChatRepository) DeleteChatForUser(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error {
	r.recordChatAction("delete_chat_for_user", authUser, chatID, userID, uuid.Nil)
	return nil
}

func (r *fakeChatRepository) DeleteChatForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	r.recordChatAction("delete_chat_for_all", authUser, chatID, uuid.Nil, uuid.Nil)
	return nil
}

func (r *fakeChatRepository) DeleteChat(ctx context.Context, authUser *models.User, chatID, userID uuid.UUID) error {
	r.recordChatAction("delete_chat", authUser, chatID, userID, uuid.Nil)
	return nil
}

func (r *fakeChatRepository) DeleteMessage(ctx context.Context, authUser *models.User, chatID, userID, messageID uuid.UUID) error {
	r.recordChatAction("delete_message", authUser, chatID, userID, messageID)
	return nil
}

func (r *fakeChatRepository) DeleteChatHistoryForUser(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	r.recordChatAction("delete_history_for_user", authUser, chatID, uuid.Nil, uuid.Nil)
	return nil
}

func (r *fakeChatRepository) DeleteChatHistoryForAll(ctx context.Context, authUser *models.User, chatID uuid.UUID) error {
	r.recordChatAction("delete_history_for_all", authUser, chatID, uuid.Nil, uuid.Nil)
	return nil
}

func (r *fakeChatRepository) MarkChatMessageRead(ctx context.Context, authUser *models.User, chatID uuid.UUID, messages []uuid.UUID) error {
	r.markedReadChatID = chatID
	r.markedReadMessages = messages
	return nil
}

func (r *fakeChatRepository) recordChatAction(action string, authUser *models.User, chatID, userID, messageID uuid.UUID) {
	r.action = action
	r.actionAuthUser = authUser
	r.actionChatID = chatID
	r.actionUserID = userID
	r.actionMessageID = messageID
}

type fakeRealtimeNotifier struct {
	room  string
	event string
	msg   string
	err   error
}

func (r *fakeRealtimeNotifier) BroadcastToRoom(namespace string, room string, event string, msg string) error {
	r.room = room
	r.event = event
	r.msg = msg
	return r.err
}

type fakeMediaRepository struct {
	ports.MediaRepository
}

func (r *fakeMediaRepository) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userID uuid.UUID, role media.MediaRole, file ports.UploadedFile) (*media.Media, error) {
	return &media.Media{ID: uuid.New(), OwnerID: ownerID, OwnerType: ownerType, UserID: userID, Role: role}, nil
}

type fakeMatchesRepository struct {
	ports.MatchesRepository
}
