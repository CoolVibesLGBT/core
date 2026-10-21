package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainevents "core/domain/events"
	domainmedia "core/domain/media"
	domainmoderation "core/domain/moderation"
	domainuser "core/domain/user"
	"core/models"
	"core/models/media"
	"core/models/notifications"
	"core/models/post"
	"core/models/utils"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserService struct {
	mediaRepo                ports.MediaRepository
	userRepo                 ports.UserRepository
	profileWriter            ports.UserProfileWriter
	postRepo                 ports.PostRepository
	engagementRepo           ports.EngagementRepository
	notificationRepo         ports.NotificationRepository
	captchaVerifier          ports.CaptchaVerifier
	passwordHasher           ports.PasswordHasher
	tokenIssuer              ports.TokenIssuer
	publicIDGen              ports.PublicIDGenerator
	remoteImage              ports.RemoteImageFetcher
	eventPublisher           ports.EventPublisher
	privatePhotoBlockRevoker ports.PrivatePhotoBlockRevoker
	privatePhotoRealtime     ports.PrivatePhotoRealtimePublisher
}

type UserServiceOption func(*UserService)

func WithCaptchaVerifier(verifier ports.CaptchaVerifier) UserServiceOption {
	return func(s *UserService) {
		s.captchaVerifier = verifier
	}
}

func WithPasswordHasher(hasher ports.PasswordHasher) UserServiceOption {
	return func(s *UserService) {
		s.passwordHasher = hasher
	}
}

func WithTokenIssuer(issuer ports.TokenIssuer) UserServiceOption {
	return func(s *UserService) {
		s.tokenIssuer = issuer
	}
}

func WithPublicIDGenerator(generator ports.PublicIDGenerator) UserServiceOption {
	return func(s *UserService) {
		s.publicIDGen = generator
	}
}

func WithRemoteImageFetcher(fetcher ports.RemoteImageFetcher) UserServiceOption {
	return func(s *UserService) {
		s.remoteImage = fetcher
	}
}

func WithEventPublisher(publisher ports.EventPublisher) UserServiceOption {
	return func(s *UserService) {
		s.eventPublisher = publisher
	}
}

func WithPrivatePhotoBlockRevoker(revoker ports.PrivatePhotoBlockRevoker) UserServiceOption {
	return func(s *UserService) {
		s.privatePhotoBlockRevoker = revoker
	}
}

func WithPrivatePhotoRealtimePublisher(publisher ports.PrivatePhotoRealtimePublisher) UserServiceOption {
	return func(s *UserService) {
		s.privatePhotoRealtime = publisher
	}
}

func NewUserService(
	userRepo ports.UserRepository,
	postRepo ports.PostRepository,
	mediaRepo ports.MediaRepository,
	engagementRepo ports.EngagementRepository,
	notificationRepo ports.NotificationRepository,
	options ...UserServiceOption,
) *UserService {
	service := &UserService{
		postRepo:         postRepo,
		mediaRepo:        mediaRepo,
		userRepo:         userRepo,
		profileWriter:    userRepo,
		notificationRepo: notificationRepo,
		engagementRepo:   engagementRepo,
		eventPublisher:   ports.NoopEventPublisher(),
	}

	for _, option := range options {
		option(service)
	}

	return service
}

type RegisterInput struct {
	Name           string `form:"name"`     // Profile display name.
	Nickname       string `form:"nickname"` // Unique username used for sign-in.
	Password       string `form:"password"`
	Domain         string `form:"domain"`
	Email          string `form:"email"`
	Captcha        string `form:"captcha"`
	RecaptchaToken string `form:"recaptchaToken"`
	Referral       string `form:"referralCode"`
}

type LoginInput struct {
	UserName string `form:"nickname"`
	Password string `form:"password"`
}

func (s *UserService) RegisterUser(ctx context.Context, input RegisterInput) (*models.User, string, error) {
	registration, err := domainuser.NewRegistration(domainuser.RegistrationInput{
		Name:     input.Name,
		Nickname: input.Nickname,
		Password: input.Password,
		Domain:   input.Domain,
		Email:    input.Email,
	})
	if err != nil {
		return nil, "", err
	}

	captchaToken := input.RecaptchaToken
	if captchaToken == "" {
		captchaToken = input.Captcha
	}

	captchaValid, err := s.verifyCaptcha(ctx, captchaToken)
	if err != nil {
		return nil, "", err
	}
	if !captchaValid {
		return nil, "", constants.ErrInvalidCaptcha
	}

	usernameExists, err := s.userRepo.ExistsByUsername(registration.Nickname)
	if err != nil {
		return nil, "", err
	}
	if usernameExists {
		return nil, "", ErrUsernameAlreadyExists
	}

	if registration.Email != "" {
		emailExists, err := s.userRepo.ExistsByEmail(registration.Email)
		if err != nil {
			return nil, "", err
		}
		if emailExists {
			return nil, "", ErrEmailAlreadyExists
		}
	}

	hash, err := s.hashPassword(registration.Password)
	if err != nil {
		return nil, "", constants.ErrPasswordHashFailed.Format(err)
	}

	publicID, err := s.generatePublicID()
	if err != nil {
		return nil, "", err
	}
	userID := uuid.New()
	userObj := &models.User{
		ID:          userID,
		PublicID:    publicID,
		Domain:      models.DomainKind(registration.Domain),
		UserName:    registration.Nickname,
		DisplayName: registration.Name,
		Email:       registration.Email,
		Password:    hash,
		UserRole:    constants.UserRoleUser,
		IsLive:      false,
		IsBot:       false,
		IsPremium:   false,
	}

	var referral *domainuser.Referral
	if strings.TrimSpace(input.Referral) != "" {
		referrer, err := s.resolveReferralUser(ctx, input.Referral)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				return nil, "", domainuser.ErrInvalidReferralCode
			}
			return nil, "", fmt.Errorf("%w: %w", ports.ErrRegistrationPersistence, err)
		}
		if referrer == nil {
			return nil, "", domainuser.ErrInvalidReferralCode
		}
		value, err := domainuser.NewReferral(referrer.ID, userID, constants.DEFAULT_REFERRAL_REWARD)
		if err != nil {
			return nil, "", err
		}
		referral = &value
	}

	if err := s.userRepo.CreateRegisteredUser(ctx, userObj, referral); err != nil {
		if errors.Is(err, domainuser.ErrReferralUnavailable) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("%w: %w", ports.ErrRegistrationPersistence, err)
	}

	userInfo, err := s.GetUserByID(userObj.ID)
	if err != nil {
		return nil, "", err
	}

	if err := s.publishEvent(ctx, domainuser.NewRegisteredEvent(userObj.ID.String(), userObj.PublicID, registration.Domain, time.Now().UTC())); err != nil {
		return nil, "", err
	}

	token, err := s.generateToken(userObj.ID, userObj.PublicID)
	if err != nil {
		return nil, "", err
	}

	return userInfo, token, nil
}

func (s *UserService) LoginUser(ctx context.Context, input LoginInput) (*models.User, string, error) {
	credentials := domainuser.NewCredentials(input.UserName, input.Password)

	userObj, err := s.userRepo.GetByUserNameOrEmailOrUsername(credentials.UserName)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// Broadcast/system bots are provisioned through trusted workers and must
	// never gain an interactive session. In particular, a blank bot password
	// must not be treated as a first-login password setup flow.
	if !domainuser.CanAuthenticate(domainuser.AccountRole(userObj.UserRole), userObj.IsBot) {
		return nil, "", ErrInvalidCredentials
	}

	ok, err := s.comparePassword(userObj.Password, credentials.Password)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateToken(userObj.ID, userObj.PublicID)
	if err != nil {
		return nil, "", err
	}

	return userObj, token, nil
}

func (s *UserService) verifyCaptcha(ctx context.Context, token string) (bool, error) {
	if s.captchaVerifier == nil {
		return false, ErrCaptchaNotConfigured
	}
	return s.captchaVerifier.VerifyCaptcha(ctx, token)
}

func (s *UserService) hashPassword(password string) (string, error) {
	if s.passwordHasher == nil {
		return "", ErrPasswordNotConfigured
	}
	return s.passwordHasher.HashPassword(password)
}

func (s *UserService) comparePassword(hashed string, raw string) (bool, error) {
	if s.passwordHasher == nil {
		return false, ErrPasswordNotConfigured
	}
	return s.passwordHasher.ComparePassword(hashed, raw)
}

func (s *UserService) generateToken(userID uuid.UUID, publicID int64) (string, error) {
	if s.tokenIssuer == nil {
		return "", ErrTokenIssuerNotConfigured
	}
	return s.tokenIssuer.GenerateUserToken(userID, publicID)
}

func (s *UserService) generatePublicID() (int64, error) {
	if s.publicIDGen == nil {
		return 0, ErrPublicIDNotConfigured
	}
	return s.publicIDGen.GeneratePublicID(), nil
}

func (s *UserService) publishEvent(ctx context.Context, event domainevents.Event) error {
	if s.eventPublisher == nil {
		return nil
	}
	return s.eventPublisher.Publish(ctx, event)
}

func (s *UserService) resolveReferralUser(ctx context.Context, referral string) (*models.User, error) {
	referral = strings.TrimSpace(referral)
	if referral == "" {
		return nil, constants.ErrReferralEmpty
	}

	if referralID, err := strconv.ParseInt(referral, 10, 64); err == nil {
		return s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: referralID})
	}

	return s.userRepo.GetByNameOrMailWithoutRelations(referral)
}

func (s *UserService) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *UserService) FetchUserProfileByUsername(username string) (*models.User, error) {
	return s.userRepo.GetByUserNameOrEmailOrUsername(username)
}

// FetchPublicUserProfileByUsername is the query-side boundary used by the
// public profile action. The legacy entity-returning method above remains for
// trusted workers that need an aggregate, but must not be used by HTTP views.
func (s *UserService) FetchPublicUserProfileByUsername(ctx context.Context, username string) (*types.PublicUserProfile, error) {
	if reader, ok := s.userRepo.(ports.PublicUserProfileReader); ok {
		return reader.FetchPublicUserProfile(ctx, username)
	}

	user, err := s.userRepo.GetByUserNameOrEmailOrUsername(username)
	if err != nil {
		return nil, err
	}
	if !isPublicDiscoverableUser(user) {
		return nil, constants.ErrUserNotFound
	}
	return publicUserProfileFromModel(user), nil
}

func (s *UserService) FetchUserProfileForViewer(ctx context.Context, username string, viewer *models.User) (*types.PublicUserProfile, error) {
	if viewer != nil && viewer.ID != uuid.Nil {
		if reader, ok := s.userRepo.(ports.ViewerUserProfileReader); ok {
			return reader.FetchUserProfileForViewer(ctx, username, viewer.ID)
		}
	}
	return s.FetchPublicUserProfileByUsername(ctx, username)
}

func (s *UserService) GetUserByPublicID(ctx context.Context, publicID int64) (*models.User, error) {
	return s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: publicID})
}

// ViewProfile records one lifetime view per authenticated viewer and target.
// Self views are intentionally ignored.
func (s *UserService) ViewProfile(ctx context.Context, authUser *models.User, targetPublicID int64) (bool, error) {
	if authUser == nil {
		return false, ErrPrivateViewEngagements
	}

	target, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: targetPublicID})
	if err != nil {
		return false, err
	}
	if target == nil {
		return false, constants.ErrUserNotFound
	}
	if target.ID == authUser.ID {
		return false, nil
	}

	return s.engagementRepo.RecordViewOnce(
		ctx,
		authUser.ID,
		target.ID,
		models.EngagementKindViewReceived,
		target.ID,
		models.EngagementContentableTypeUser,
	)
}

func (s *UserService) CreateBotUser(ctx context.Context, userObj *models.User) (*models.User, error) {
	publicID, err := s.generatePublicID()
	if err != nil {
		return nil, err
	}
	userObj.ID = uuid.New()
	userObj.PublicID = publicID
	userObj.PrivacyLevel = constants.PrivacyPublic
	userObj.UserRole = constants.UserRoleUser
	userObj.IsBot = true
	userObj.IsLive = true
	userObj.IsOnline = true
	userObj.IsActive = true

	if err := s.userRepo.Create(userObj); err != nil {
		return nil, err
	}
	return userObj, nil
}

func (s *UserService) UpdateAvatar(ctx context.Context, file *domainmedia.UploadedFile, user *models.User) (*media.Media, error) {
	if user == nil || user.ID == uuid.Nil {
		return nil, constants.ErrAuthenticatedUserRequired
	}
	persistedUser, err := s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: user.ID})
	if err != nil {
		return nil, constants.ErrAvatarUserLookupFailed.Format(err)
	}

	newMedia, err := s.mediaRepo.AddMedia(
		user.ID,
		media.OwnerUser,
		user.ID,
		media.RoleAvatar,
		file,
	)
	if err != nil {
		return nil, constants.ErrAvatarUploadFailed.Format(err)
	}

	// User tablosunu güncelle
	persistedUser.AvatarID = &newMedia.ID
	persistedUser.Avatar = newMedia

	if err := s.userRepo.UpdateUser(persistedUser); err != nil {
		return nil, constants.ErrAvatarUpdateFailed.Format(err)
	}
	return newMedia, nil
}

func (s *UserService) UpdateAvatarFromURL(ctx context.Context, imgUrl string, user *models.User) (*media.Media, error) {
	if s.remoteImage == nil {
		return nil, constants.ErrRemoteImageFetcherNotConfigured
	}

	file, err := s.remoteImage.FetchImage(ctx, imgUrl)
	if err != nil {
		return nil, err
	}

	return s.UpdateAvatar(ctx, file, user)
}

func (s *UserService) UpdateCover(ctx context.Context, file *domainmedia.UploadedFile, user *models.User) (*media.Media, error) {
	if user == nil || user.ID == uuid.Nil {
		return nil, constants.ErrAuthenticatedUserRequired
	}
	persistedUser, err := s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: user.ID})
	if err != nil {
		return nil, constants.ErrCoverUserLookupFailed.Format(err)
	}

	newMedia, err := s.mediaRepo.AddMedia(
		user.ID,
		media.OwnerUser,
		user.ID,
		media.RoleCover,
		file,
	)
	if err != nil {
		return nil, constants.ErrCoverUploadFailed.Format(err)
	}
	persistedUser.CoverID = &newMedia.ID
	persistedUser.Cover = newMedia

	if err := s.userRepo.UpdateUser(persistedUser); err != nil {
		return nil, constants.ErrCoverUpdateFailed.Format(err)
	}
	return newMedia, nil
}

func (s *UserService) AddStory(ctx context.Context, values map[string][]string, files []*domainmedia.UploadedFile, user *models.User) (*types.PublicPost, error) {
	values = cloneFormValues(values)
	if values == nil {
		values = make(map[string][]string)
	}

	if caption := firstFormValue(values, "caption"); caption != "" && firstFormValue(values, "content") == "" {
		values["content"] = []string{caption}
	}
	if firstFormValue(values, "audience") == "" {
		values["audience"] = []string{"public"}
	}
	if firstFormValue(values, "slug") == "" && firstFormValue(values, "title") == "" {
		values["slug"] = []string{fmt.Sprintf("story-%d", time.Now().UnixNano())}
	}
	if firstFormValue(values, "extras") == "" {
		extras, err := json.Marshal(map[string]string{
			"expires_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		})
		if err != nil {
			return nil, err
		}
		values["extras"] = []string{string(extras)}
	}

	createdPost, err := s.postRepo.CreateContentablePost(ctx, values, files, user, string(post.PostKindStory), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.FindPublicPostByIDIncludingUnpublished(createdPost.ID)
}

func cloneFormValues(values map[string][]string) map[string][]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string][]string, len(values))
	for key, items := range values {
		cloned[key] = append([]string(nil), items...)
	}
	return cloned
}

func firstFormValue(values map[string][]string, key string) string {
	items := values[key]
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func (s *UserService) UpsertUserPreference(ctx context.Context, userID uuid.UUID, preferenceItemID string, enabled bool) error {
	if userID == uuid.Nil {
		return constants.ErrAuthenticatedUserRequired
	}
	err := s.userRepo.UpsertUserPreference(ctx, userID, preferenceItemID, enabled)
	if err != nil {
		return constants.ErrUserAttributeUpsertFailed.Format(err)
	}
	return err
}

func (s *UserService) GetAllStories(filters types.Filter) (types.PublicPostPage, error) {
	filters.PostKind = post.PostKindStory
	result, err := s.postRepo.FetchPublicPostsByKind(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return result, nil
}

func (s *UserService) FetchNearbyUsers(filters types.Filter) ([]types.NearbyUser, *float64, error) {
	ctx := filters.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	filters.Context = ctx

	users, distance, err := s.userRepo.FetchNearbyUsers(filters)
	if err != nil && ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}
	return users, distance, err
}

func (s *UserService) FetchLiveUsers(filters types.Filter) ([]*models.User, error) {
	return s.userRepo.FetchLiveUsers(filters)
}

func (s *UserService) FetchPublicLiveUsers(filters types.Filter) ([]types.LiveUser, error) {
	users, err := s.userRepo.FetchLiveUsers(filters)
	if err != nil {
		return nil, err
	}
	result := make([]types.LiveUser, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		row := types.LiveUser{PublicUserSummary: publicUserSummaryFromModel(user), IsLive: user.IsLive, CanReceiveGifts: !user.IsBot}
		if user.IsBot {
			var metadata map[string]json.RawMessage
			if json.Unmarshal(user.BroadcastInfo, &metadata) == nil {
				row.BroadcastInfo = make(map[string]json.RawMessage)
				// Viewer credentials are necessary for existing external playback. Never
				// expose publisher credentials, ACLs, provider accounts or arbitrary data.
				for _, key := range []string{"objectId", "provider", "streamDescription", "description", "currentViewers", "totalViewers", "totalLikes", "totalDiamonds", "broadcasterLifetimeDiamonds", "language", "viewerToken", "viewerTokenExpiration", "agoraAppId"} {
					if value, ok := metadata[key]; ok {
						row.BroadcastInfo[key] = value
					}
				}
				_ = json.Unmarshal(metadata["provider"], &row.Provider)
			}
		} else {
			row.Provider = "local"
		}
		result = append(result, row)
	}
	return result, nil
}

func (s *UserService) GetUsersStartingWith(letter string, limit int) ([]models.User, error) {
	return s.userRepo.GetUsersStartingWith(letter, limit)
}

func (s *UserService) SearchUsers(filters types.Filter) ([]models.User, error) {
	type scopedUserSearcher interface {
		SearchUsers(types.Filter) ([]models.User, error)
	}

	if searcher, ok := s.userRepo.(scopedUserSearcher); ok {
		return searcher.SearchUsers(filters)
	}
	if filters.Search == nil {
		return []models.User{}, nil
	}
	return s.userRepo.GetUsersStartingWith(*filters.Search, filters.Limit)
}

func (s *UserService) GetPublicUsersStartingWith(letter string, limit int) ([]types.PublicUserSummary, error) {
	query := strings.TrimSpace(letter)
	return s.SearchPublicUsers(types.Filter{Search: &query, Limit: limit})
}

// SearchPublicUsers returns a public read model even when a test double or a
// legacy adapter has not implemented the optimized projection port yet.
func (s *UserService) SearchPublicUsers(filters types.Filter) ([]types.PublicUserSummary, error) {
	if reader, ok := s.userRepo.(ports.PublicUserSearchReader); ok {
		return reader.SearchPublicUsers(filters)
	}

	users, err := s.SearchUsers(filters)
	if err != nil {
		return nil, err
	}
	result := make([]types.PublicUserSummary, 0, len(users))
	for index := range users {
		if !isPublicDiscoverableUser(&users[index]) {
			continue
		}
		result = append(result, publicUserSummaryFromModel(&users[index]))
	}
	return result, nil
}

func isPublicDiscoverableUser(user *models.User) bool {
	if user == nil || user.IsBot || user.DeletedAt.Valid || user.PrivacyLevel != constants.PrivacyPublic {
		return false
	}
	switch user.UserRole {
	case constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending:
		return false
	default:
		return true
	}
}

func publicUserProfileFromModel(user *models.User) *types.PublicUserProfile {
	if user == nil {
		return nil
	}

	return &types.PublicUserProfile{
		ID:              types.SnowflakeID(user.PublicID),
		PublicID:        types.SnowflakeID(user.PublicID),
		UserName:        user.UserName,
		DisplayName:     user.DisplayName,
		Bio:             cloneLocalizedString(user.Bio),
		Website:         user.Website,
		DateOfBirth:     user.DateOfBirth,
		PrivacyLevel:    string(user.PrivacyLevel),
		IsOnline:        user.IsOnline,
		IsPremium:       user.IsPremium,
		CreatedAt:       user.CreatedAt,
		DefaultLanguage: user.DefaultLanguage,
		Languages:       append([]string(nil), user.Languages...),
		Hobbies:         append([]string(nil), user.Hobbies...),
		MoviesGenres:    append([]string(nil), user.MoviesGenres...),
		TVShowsGenres:   append([]string(nil), user.TVShowsGenres...),
		TheaterGenres:   append([]string(nil), user.TheaterGenres...),
		CinemaGenres:    append([]string(nil), user.CinemaGenres...),
		ArtInterests:    append([]string(nil), user.ArtInterests...),
		Entertainment:   append([]string(nil), user.Entertainment...),
		Location:        publicUserLocationFromModel(user.Location),
		Avatar:          publicUserMediaFromModel(user.Avatar),
		Cover:           publicUserMediaFromModel(user.Cover),
		Engagements: types.PublicUserEngagements{
			Counts: publicUserEngagementCounts(userEngagementCounts(user)),
		},
	}
}

// UserProfileResponse maps an authenticated profile command result onto the
// same persistence-free response boundary used by public profile reads. It is
// safe for HTTP serialization and deliberately omits account credentials,
// authorization state, wallet data and internal identifiers.
func UserProfileResponse(user *models.User) *types.PublicUserProfile {
	return publicUserProfileFromModel(user)
}

// AuthUserResponse is the authenticated HTTP boundary. The service may need
// the persistence identity internally for token issuance and commands, but
// adapters must never serialize that persistence entity directly.
func AuthUserResponse(user *models.User) *types.AuthUser {
	profile := publicUserProfileFromModel(user)
	if profile == nil {
		return nil
	}
	return &types.AuthUser{
		PublicUserProfile: *profile,
		UserRole:          string(user.UserRole),
	}
}

func publicUserSummaryFromModel(user *models.User) types.PublicUserSummary {
	if user == nil {
		return types.PublicUserSummary{}
	}
	return types.PublicUserSummary{
		ID:          types.SnowflakeID(user.PublicID),
		PublicID:    types.SnowflakeID(user.PublicID),
		UserName:    user.UserName,
		DisplayName: user.DisplayName,
		Bio:         cloneLocalizedString(user.Bio),
		IsOnline:    user.IsOnline,
		Location:    publicUserLocationFromModel(user.Location),
		Avatar:      publicUserMediaFromModel(user.Avatar),
	}
}

func cloneLocalizedString(value *utils.LocalizedString) map[string]string {
	if value == nil || len(*value) == 0 {
		return nil
	}
	result := make(map[string]string, len(*value))
	for language, text := range *value {
		result[language] = text
	}
	return result
}

func publicUserLocationFromModel(location *utils.Location) *types.PublicUserLocation {
	if location == nil {
		return nil
	}
	if location.Display == nil && location.City == nil && location.Region == nil && location.Country == nil {
		return nil
	}
	return &types.PublicUserLocation{
		Display: location.Display,
		City:    location.City,
		Region:  location.Region,
		Country: location.Country,
	}
}

func publicUserMediaFromModel(value *media.Media) *types.PublicUserMedia {
	if value == nil {
		return nil
	}

	result := &types.PublicUserMedia{File: types.PublicUserMediaFile{URL: value.File.URL}}
	if value.File.Variants != nil {
		if encoded, err := json.Marshal(value.File.Variants); err == nil && string(encoded) != "null" {
			result.File.Variants = encoded
		}
	}
	if result.File.URL == "" && len(result.File.Variants) == 0 {
		return nil
	}
	return result
}

func userEngagementCounts(user *models.User) []byte {
	if user == nil || user.Engagements == nil {
		return nil
	}
	return user.Engagements.Counts
}

func publicUserEngagementCounts(raw []byte) types.PublicUserEngagementCounts {
	var source map[string]json.RawMessage
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &source)
	}
	return types.PublicUserEngagementCounts{
		FollowerCount:        publicCount(source["follower_count"]),
		FollowingCount:       publicCount(source["following_count"]),
		PostCount:            publicCount(source["post_count"]),
		BlockingCount:        publicCount(source["blocking_count"]),
		BlockedByCount:       publicCount(source["blocked_by_count"]),
		LikeGivenCount:       publicCount(source["like_given_count"]),
		LikeReceivedCount:    publicCount(source["like_received_count"]),
		DislikeGivenCount:    publicCount(source["dislike_given_count"]),
		DislikeReceivedCount: publicCount(source["dislike_received_count"]),
		MatchCount:           publicCount(source["match_count"]),
		ViewReceivedCount:    publicCount(source["view_received_count"]),
	}
}

func publicCount(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}
	var number int64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		value, _ := strconv.ParseInt(text, 10, 64)
		return value
	}
	return 0
}

func (s *UserService) Follow(ctx context.Context, followerID, followeeID int64) (bool, error) {
	return s.HandleFollow(ctx, followerID, followeeID, true)
}

func (s *UserService) Unfollow(ctx context.Context, followerID, followeeID int64) (bool, error) {
	return s.HandleFollow(ctx, followerID, followeeID, false)
}

func (s *UserService) HandleFollow(ctx context.Context, followerID, followeeID int64, isFollow bool) (bool, error) {
	intent, err := domainuser.NewSetInteractionState(domainuser.InteractionFollow, isFollow)
	if err != nil {
		return false, err
	}
	return s.applyFollowIntent(ctx, followerID, followeeID, intent)
}

func (s *UserService) ToggleFollow(ctx context.Context, followerID, followeeID int64) (bool, error) {
	intent, err := domainuser.NewToggleInteractionState(domainuser.InteractionFollow)
	if err != nil {
		return false, err
	}
	return s.applyFollowIntent(ctx, followerID, followeeID, intent)
}

func (s *UserService) applyFollowIntent(ctx context.Context, followerID, followeeID int64, intent domainuser.InteractionStateIntent) (bool, error) {
	if err := domainuser.EnsureDifferentPublicUsers(followerID, followeeID, domainuser.InteractionFollow); err != nil {
		return false, err
	}

	followerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: followerID})
	if err != nil {
		return false, err
	}
	followeeUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: followeeID})
	if err != nil {
		return false, err
	}

	transition, err := s.engagementRepo.ApplyReciprocalUserInteraction(ctx, followerUser.ID, followeeUser.ID, intent)
	if err != nil {
		return false, err
	}
	if !transition.Changed {
		return transition.Enabled, nil
	}

	if !s.socialNotificationsTransactional() {
		if transition.Enabled {
			// Follow started
			notificationTitleToFollowee := "New Follower"
			notificationBodyToFollowee := followerUser.UserName + " started following you."

			payloadToFollowee := notifications.NotificationPayload{
				Title: notificationTitleToFollowee,
				Body:  notificationBodyToFollowee,
			}
			err := s.notificationRepo.SendNotificationToUser(ctx, *followerUser, *followeeUser, notifications.NotificationTypeFollow, notificationTitleToFollowee, notificationBodyToFollowee, payloadToFollowee)
			if err != nil {
				fmt.Printf("Failed to send notification to user %d: %v\n", followeeUser.ID, err)
			}

			notificationTitleToFollower := "Follow Started"
			notificationBodyToFollower := "You started following " + followeeUser.UserName + "."

			payloadToFollower := notifications.NotificationPayload{
				Title: notificationTitleToFollower,
				Body:  notificationBodyToFollower,
			}
			err = s.notificationRepo.SendNotificationToUser(ctx, *followeeUser, *followerUser, notifications.NotificationTypeFollow, notificationTitleToFollower, notificationBodyToFollower, payloadToFollower)
			if err != nil {
				fmt.Printf("Failed to send notification to user %d: %v\n", followerUser.ID, err)
			}

		} else {
			// Follow stopped
			notificationTitleToFollowee := "Unfollowed"
			notificationBodyToFollowee := followerUser.UserName + " unfollowed you."

			payloadToFollowee := notifications.NotificationPayload{
				Title: notificationTitleToFollowee,
				Body:  notificationBodyToFollowee,
			}
			err := s.notificationRepo.SendNotificationToUser(ctx, *followerUser, *followeeUser, notifications.NotificationTypeUnFollow, notificationTitleToFollowee, notificationBodyToFollowee, payloadToFollowee)
			if err != nil {
				fmt.Printf("Failed to send notification to user %d: %v\n", followeeUser.ID, err)
			}

			notificationTitleToFollower := "Unfollowed"
			notificationBodyToFollower := "You unfollowed " + followeeUser.UserName + "."

			payloadToFollower := notifications.NotificationPayload{
				Title: notificationTitleToFollower,
				Body:  notificationBodyToFollower,
			}
			err = s.notificationRepo.SendNotificationToUser(ctx, *followeeUser, *followerUser, notifications.NotificationTypeUnFollow, notificationTitleToFollower, notificationBodyToFollower, payloadToFollower)
			if err != nil {
				fmt.Printf("Failed to send notification to user %d: %v\n", followerUser.ID, err)
			}
		}

	}

	if err := s.publishEvent(ctx, domainuser.NewInteractionToggledEvent(followerID, followeeID, domainuser.InteractionFollow, transition.Enabled, time.Now().UTC())); err != nil {
		return transition.Enabled, err
	}

	return transition.Enabled, nil
}

type UpdateUserProfileInput struct {
	UserName                string
	Password                string
	CurrentPassword         string
	NewPassword             string
	NewPasswordConfirmation string
	Email                   string
	DisplayName             string
	Bio                     string
	Website                 string
	DateOfBirth             string
	PrivacyLevel            string
	LocationContentableType string
	LocationCountryCode     string
	LocationAddress         string
	LocationCity            string
	LocationCountry         string
	LocationRegion          string
	LocationTimezone        string
	LocationDisplay         string
	LocationLatitude        string
	LocationLongitude       string
}

func (s *UserService) UpdateUserProfile(ctx context.Context, authUser models.User, input UpdateUserProfileInput) (*models.User, error) {
	if authUser.ID == uuid.Nil {
		return nil, ports.ErrNotFound
	}
	if input.Password != "" {
		return nil, ErrLegacyPasswordField
	}

	dateOfBirth, err := domainuser.ParseBirthDate(input.DateOfBirth, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	privacyLevel, hasPrivacyLevel, err := domainuser.ParsePrivacyLevel(input.PrivacyLevel)
	if err != nil {
		return nil, err
	}
	email, hasEmail, err := domainuser.NormalizeEmail(input.Email)
	if err != nil {
		return nil, err
	}
	website, hasWebsite, err := domainuser.NormalizeWebsite(input.Website)
	if err != nil {
		return nil, err
	}
	changePassword, err := domainuser.ValidatePasswordChange(input.CurrentPassword, input.NewPassword, input.NewPasswordConfirmation)
	if err != nil {
		return nil, err
	}
	location, err := profileLocationUpdate(input)
	if err != nil {
		return nil, err
	}

	userInfo, err := s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: authUser.ID})
	if err != nil {
		return nil, err
	}

	username := domainuser.NormalizeUsername(input.UserName)
	usernameChanged := username != "" && username != userInfo.UserName
	if usernameChanged && !strings.EqualFold(username, userInfo.UserName) {
		exists, err := s.userRepo.ExistsByUsername(username)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUsernameAlreadyExists
		}
	}
	emailChanged := hasEmail && email != userInfo.Email
	if emailChanged && !strings.EqualFold(email, userInfo.Email) {
		exists, err := s.userRepo.ExistsByEmail(email)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrEmailAlreadyExists
		}
	}

	if input.CurrentPassword != "" {
		// The authenticated request user is a narrow session projection and does
		// not contain credentials. Compare against the explicitly reloaded row.
		ok, err := s.comparePassword(userInfo.Password, input.CurrentPassword)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrInvalidCurrentPassword
		}
	}

	update := ports.UserProfileUpdate{UserID: authUser.ID, Location: location}
	if usernameChanged {
		update.UserName = &username
	}
	if displayName := domainuser.NormalizeDisplayName(input.DisplayName); displayName != "" && displayName != userInfo.DisplayName {
		update.DisplayName = &displayName
	}
	if emailChanged {
		update.Email = &email
	}
	if hasWebsite && website != userInfo.Website {
		update.Website = &website
	}
	if input.Bio != "" {
		update.Bio = map[string]string{userInfo.DefaultLanguage: input.Bio}
	}
	if dateOfBirth != nil {
		update.DateOfBirth = dateOfBirth
	}
	if hasPrivacyLevel {
		value := string(privacyLevel)
		update.PrivacyLevel = &value
	}
	if changePassword {
		passwordHash, err := s.hashPassword(input.NewPassword)
		if err != nil {
			return nil, err
		}
		update.PasswordHash = &passwordHash
	}

	if err := s.profileWriter.UpdateUserProfile(ctx, update); err != nil {
		return nil, err
	}
	return s.GetUserByID(authUser.ID)
}

func profileLocationUpdate(input UpdateUserProfileInput) (*ports.UserProfileLocationUpdate, error) {
	latitude := strings.TrimSpace(input.LocationLatitude)
	longitude := strings.TrimSpace(input.LocationLongitude)
	hasLocationMetadata := strings.TrimSpace(input.LocationContentableType) != "" ||
		strings.TrimSpace(input.LocationCountryCode) != "" || strings.TrimSpace(input.LocationAddress) != "" ||
		strings.TrimSpace(input.LocationCity) != "" || strings.TrimSpace(input.LocationCountry) != "" ||
		strings.TrimSpace(input.LocationRegion) != "" || strings.TrimSpace(input.LocationTimezone) != "" ||
		strings.TrimSpace(input.LocationDisplay) != ""

	if latitude == "" && longitude == "" && !hasLocationMetadata {
		return nil, nil
	}
	ownerType := strings.TrimSpace(input.LocationContentableType)
	if ownerType != "" && !strings.EqualFold(ownerType, string(utils.LocationOwnerUser)) {
		return nil, ErrInvalidLocationOwner
	}
	if latitude == "" || longitude == "" {
		return nil, ErrLocationCoordinates
	}

	lat, err := strconv.ParseFloat(latitude, 64)
	if err != nil {
		return nil, domainuser.ErrInvalidLatitude
	}
	lng, err := strconv.ParseFloat(longitude, 64)
	if err != nil {
		return nil, domainuser.ErrInvalidLongitude
	}
	coordinates, err := domainuser.NewCoordinates(lat, lng)
	if err != nil {
		return nil, err
	}

	return &ports.UserProfileLocationUpdate{
		CountryCode: input.LocationCountryCode,
		Address:     input.LocationAddress,
		City:        input.LocationCity,
		Country:     input.LocationCountry,
		Region:      input.LocationRegion,
		Timezone:    input.LocationTimezone,
		Display:     input.LocationDisplay,
		Latitude:    coordinates.Latitude,
		Longitude:   coordinates.Longitude,
	}, nil
}

// return Params : bool isLike, bool success, error
func (s *UserService) Like(ctx context.Context, authUser models.User, likerId, likeeId int64) (bool, bool, error) {
	return s.HandleLike(ctx, authUser, likerId, likeeId, true)
}

func (s *UserService) Dislike(ctx context.Context, authUser models.User, likerId, likeeId int64) (bool, bool, error) {
	return s.HandleLike(ctx, authUser, likerId, likeeId, false)
}

func (s *UserService) HandleLike(ctx context.Context, authUser models.User, likerId, likeeId int64, isLike bool) (bool, bool, error) {
	return s.ToggleLike(ctx, authUser, likerId, likeeId, isLike)
}

func (s *UserService) ToggleLike(ctx context.Context, authUser models.User, likerId, likeeId int64, isLike bool) (bool, bool, error) {
	if err := domainuser.EnsureActorMatchesPrincipal(authUser.PublicID, likerId); err != nil {
		return isLike, false, err
	}
	if err := domainuser.EnsureDifferentPublicUsers(likerId, likeeId, domainuser.InteractionLike); err != nil {
		return isLike, false, err
	}

	likerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: likerId})
	if err != nil {
		return isLike, false, err
	}
	likeeUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: likeeId})
	if err != nil {
		return isLike, false, err
	}

	intent, err := domainuser.NewToggleReactionState(isLike)
	if err != nil {
		return isLike, false, err
	}
	transition, err := s.engagementRepo.ApplyReciprocalUserInteraction(ctx, likerUser.ID, likeeUser.ID, intent)
	if err != nil {
		return isLike, false, err
	}
	if transition.NewMatch && !s.socialNotificationsTransactional() {
		s.notifyNewUserMatch(ctx, likerUser, likeeUser)
	}

	if transition.Changed {
		if err := s.publishEvent(ctx, domainuser.NewInteractionToggledEvent(likerId, likeeId, domainuser.InteractionLike, transition.Enabled, time.Now().UTC())); err != nil {
			return isLike, false, err
		}
	}

	return isLike, transition.Enabled, nil
}

func (s *UserService) notifyNewUserMatch(ctx context.Context, first, second *models.User) {
	if s.notificationRepo == nil {
		return
	}
	for _, pair := range [][2]*models.User{{first, second}, {second, first}} {
		payload := notifications.NotificationPayload{
			Data: map[string]string{"match_user_id": strconv.FormatInt(pair[0].PublicID, 10)},
		}
		_ = s.notificationRepo.SendNotificationToUser(ctx, *pair[0], *pair[1], notifications.NotificationTypeNewMatch,
			"It's a Match!", "You have a new match.", payload)
	}
}

// return Params : bool isLike, bool success, error
func (s *UserService) Block(ctx context.Context, authUser models.User, blockerId, blockedId int64) (bool, error) {
	return s.HandleBlock(ctx, authUser, blockerId, blockedId, true)
}

func (s *UserService) Unblock(ctx context.Context, authUser models.User, blockerId, blockedId int64) (bool, error) {
	return s.HandleBlock(ctx, authUser, blockerId, blockedId, false)
}

func (s *UserService) HandleBlock(ctx context.Context, authUser models.User, blockerId, blockedId int64, isBlock bool) (bool, error) {
	intent, err := domainuser.NewSetInteractionState(domainuser.InteractionBlock, isBlock)
	if err != nil {
		return false, err
	}
	return s.applyBlockIntent(ctx, authUser, blockerId, blockedId, intent)
}

func (s *UserService) ToggleBlock(ctx context.Context, authUser models.User, blockerId, blockedId int64) (bool, error) {
	intent, err := domainuser.NewToggleInteractionState(domainuser.InteractionBlock)
	if err != nil {
		return false, err
	}
	return s.applyBlockIntent(ctx, authUser, blockerId, blockedId, intent)
}

func (s *UserService) applyBlockIntent(ctx context.Context, authUser models.User, blockerId, blockedId int64, intent domainuser.InteractionStateIntent) (bool, error) {
	if err := domainuser.EnsureActorMatchesPrincipal(authUser.PublicID, blockerId); err != nil {
		return false, err
	}
	if err := domainuser.EnsureDifferentPublicUsers(blockerId, blockedId, domainuser.InteractionBlock); err != nil {
		return false, err
	}

	blockerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: blockerId})
	if err != nil {
		return false, err
	}
	blockedUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: blockedId})
	if err != nil {
		return false, err
	}

	transition, err := s.engagementRepo.ApplyReciprocalUserInteraction(ctx, blockerUser.ID, blockedUser.ID, intent)
	if err != nil {
		return false, err
	}
	if transition.Enabled {
		var revokeErr error
		if s.privatePhotoBlockRevoker != nil {
			revokeErr = s.privatePhotoBlockRevoker.RevokePrivatePhotoAccessBetween(
				ctx,
				blockerUser.ID,
				blockedUser.ID,
				time.Now().UTC(),
			)
		}

		// The committed block itself denies access, even when cleaning up the
		// stored grant fails. Publish fail-closed invalidations before returning
		// that cleanup error so either user's open viewer closes immediately.
		s.publishPrivatePhotoBlockInvalidations(ctx, blockerId, blockedId)
		if revokeErr != nil {
			return transition.Enabled, revokeErr
		}
	}
	if !transition.Changed {
		return transition.Enabled, nil
	}

	if err := s.publishEvent(ctx, domainuser.NewInteractionToggledEvent(blockerId, blockedId, domainuser.InteractionBlock, transition.Enabled, time.Now().UTC())); err != nil {
		return transition.Enabled, err
	}

	return transition.Enabled, nil
}

func (s *UserService) publishPrivatePhotoBlockInvalidations(ctx context.Context, firstPublicID, secondPublicID int64) {
	if s.privatePhotoRealtime == nil || firstPublicID <= 0 || secondPublicID <= 0 || firstPublicID == secondPublicID {
		return
	}
	now := time.Now().UTC()
	for _, pair := range [][2]int64{{firstPublicID, secondPublicID}, {secondPublicID, firstPublicID}} {
		_ = s.privatePhotoRealtime.PublishPrivatePhotoEvent(ctx, []int64{firstPublicID, secondPublicID}, ports.PrivatePhotoRealtimeEnvelope{
			Version:    ports.PrivatePhotoRealtimeVersion,
			EventID:    uuid.NewString(),
			Type:       ports.PrivatePhotoEventAccessInvalidated,
			OccurredAt: now,
			Data: ports.PrivatePhotoRealtimeEventData{
				OwnerID:  publicIDString(pair[0]),
				ViewerID: publicIDString(pair[1]),
				Status:   string(domainmedia.PrivatePhotoAccessDenied),
			},
		})
	}
}

func (s *UserService) ToggleSubscribe(ctx context.Context, authUser models.User, subscriberId, subscribedId int64) (bool, error) {
	if err := domainuser.EnsureActorMatchesPrincipal(authUser.PublicID, subscriberId); err != nil {
		return false, err
	}
	if err := domainuser.EnsureDifferentPublicUsers(subscriberId, subscribedId, domainuser.InteractionSubscribe); err != nil {
		return false, err
	}

	subscriberUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: subscriberId})
	if err != nil {
		return false, err
	}
	subscribedUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: subscribedId})
	if err != nil {
		return false, err
	}

	intent, err := domainuser.NewToggleInteractionState(domainuser.InteractionSubscribe)
	if err != nil {
		return false, err
	}
	transition, err := s.engagementRepo.ApplyReciprocalUserInteraction(ctx, subscriberUser.ID, subscribedUser.ID, intent)
	if err != nil {
		return false, err
	}
	if transition.Changed {
		if err := s.publishEvent(ctx, domainuser.NewInteractionToggledEvent(subscriberId, subscribedId, domainuser.InteractionSubscribe, transition.Enabled, time.Now().UTC())); err != nil {
			return transition.Enabled, err
		}
	}

	return transition.Enabled, nil
}

func (s *UserService) FetchUserNotifications(ctx context.Context, authUser *models.User, cursor *time.Time, limit int) (items []*notifications.Notification, nextCursor *time.Time, err error) {
	return s.userRepo.FetchUserNotifications(ctx, authUser, cursor, limit)
}

func (s *UserService) Report(ctx context.Context, userPublicID int64, kind string, description string, authUser *models.User) error {
	report, err := validateReportSubmission(domainmoderation.TargetUser, userPublicID, kind, description, authUser)
	if err != nil {
		if errors.Is(err, domainmoderation.ErrInvalidTarget) {
			return ErrUserIDRequired
		}
		return err
	}
	return s.userRepo.Report(
		ctx,
		report.Target().PublicID(),
		report.Kind().String(),
		report.Description().String(),
		authUser,
	)
}

func (s *UserService) FetchUserEngagements(ctx context.Context, authUser *models.User, contentableID uuid.UUID, contentableType models.EngagementContentableType, engagementKind models.EngagementKind, cursor *types.MatchListCursor, limit int) ([]types.UserEngagement, *types.MatchListCursor, error) {
	if (engagementKind == models.EngagementKindViewGiven || engagementKind == models.EngagementKindViewReceived) &&
		(authUser == nil || authUser.ID != contentableID) {
		return nil, nil, ErrPrivateViewEngagements
	}
	if authUser == nil || authUser.ID == uuid.Nil || contentableType != models.EngagementContentableTypeUser {
		return nil, nil, ErrPrivateEngagements
	}
	if authUser.ID != contentableID {
		if engagementKind != models.EngagementKindFollowing && engagementKind != models.EngagementKindFollower {
			return nil, nil, ErrPrivateEngagements
		}
		owner, err := s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: contentableID})
		if err != nil {
			return nil, nil, err
		}
		visible := isPublicDiscoverableUser(owner) && owner.Domain == authUser.Domain
		if access, ok := s.userRepo.(ports.UserProfileAccess); ok {
			var err error
			visible, err = access.CanViewUserProfile(ctx, contentableID, authUser.ID)
			if err != nil {
				return nil, nil, err
			}
		}
		if !visible {
			return nil, nil, ErrPrivateEngagements
		}
		for _, pair := range [][2]uuid.UUID{{authUser.ID, contentableID}, {contentableID, authUser.ID}} {
			blocked, err := s.engagementRepo.HasUserEngaged(ctx, pair[0], pair[1], models.EngagementKindBlocking)
			if err != nil {
				return nil, nil, err
			}
			if blocked {
				return nil, nil, ErrPrivateEngagements
			}
		}
	}
	rows, next, err := s.engagementRepo.GetEngagements(ctx, contentableType, contentableID, engagementKind, cursor, limit, authUser.ID)
	if err != nil {
		return nil, nil, err
	}
	var views []types.UserEngagement
	if rows != nil {
		views = make([]types.UserEngagement, 0, len(rows))
	}
	for _, row := range rows {
		views = append(views, types.UserEngagement{
			ID: row.ID.String(), Kind: string(row.Kind), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
			Engager: publicUserSummaryFromModel(&row.Engager), Engagee: publicUserSummaryFromModel(&row.Engagee),
		})
	}
	return views, next, nil
}

func (s *UserService) CheckIn(context context.Context, values map[string][]string, files []*domainmedia.UploadedFile, author *models.User, postKind post.PostKind) (*types.PublicPost, error) {
	_post, err := s.postRepo.CreateContentablePost(context, values, files, author, string(postKind), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.FindPublicPostByIDIncludingUnpublished(_post.ID)
}

func (s *UserService) FetchCheckIns(filters types.Filter) (types.PublicPostPage, error) {
	result, err := s.postRepo.FetchPublicPostsByKind(filters)
	if err != nil {
		return types.PublicPostPage{}, err
	}
	return result, nil
}

func (s *UserService) DeleteUser(filters types.Filter) error {
	return s.userRepo.DeleteUser(filters)

}

func (s *UserService) socialNotificationsTransactional() bool {
	adapter, ok := s.engagementRepo.(ports.TransactionalSocialNotifications)
	return ok && adapter.SocialNotificationsTransactional()
}
