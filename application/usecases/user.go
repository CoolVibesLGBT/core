package usecases

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainevents "core/domain/events"
	domainuser "core/domain/user"
	"core/models"
	"core/models/media"
	"core/models/notifications"
	"core/models/post"
	"core/models/utils"
	"core/types"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserService struct {
	mediaRepo        ports.MediaRepository
	userRepo         ports.UserRepository
	postRepo         ports.PostRepository
	engagementRepo   ports.EngagementRepository
	notificationRepo ports.NotificationRepository
	captchaVerifier  ports.CaptchaVerifier
	passwordHasher   ports.PasswordHasher
	tokenIssuer      ports.TokenIssuer
	publicIDGen      ports.PublicIDGenerator
	remoteImage      ports.RemoteImageFetcher
	eventPublisher   ports.EventPublisher
}

var (
	ErrPrivateViewEngagements   = errors.New("profile view engagements are private")
	ErrCannotReportSelf         = errors.New("users cannot report themselves")
	ErrUserIDRequired           = errors.New("user_id is required")
	ErrCaptchaNotConfigured     = errors.New("captcha verifier is not configured")
	ErrPasswordNotConfigured    = errors.New("password hasher is not configured")
	ErrTokenIssuerNotConfigured = errors.New("token issuer is not configured")
	ErrPublicIDNotConfigured    = errors.New("public ID generator is not configured")
)

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
	Name           string `form:"name"`
	Nickname       string `form:"nickname"`
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
		return nil, "", errors.New("invalid captcha")
	}

	hash, err := s.hashPassword(registration.Password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create hash password: %w", err)
	}

	exists, err := s.userRepo.ExistsByNameOrMail(registration.Nickname)
	if err != nil {
		return nil, "", err
	}
	if exists {
		return nil, "", errors.New("username already exists")
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
		UserName:    registration.Name,
		DisplayName: registration.Nickname,
		Email:       registration.Email,
		Password:    hash,
		UserRole:    constants.UserRoleUser,
		IsLive:      false,
		IsBot:       false,
		IsPremium:   false,
	}

	if err := s.userRepo.Create(userObj); err != nil {
		return nil, "", err
	}

	userInfo, err := s.GetUserByID(userObj.ID)
	if err != nil {
		return nil, "", err
	}

	if input.Referral != "" {
		s.applyReferral(ctx, input.Referral, userInfo.ID)
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
		return nil, "", errors.New("invalid username/email/nickname or password")
	}

	if userObj.IsBot && userObj.Password == "" {
		if credentials.Password == "" {
			return nil, "", errors.New("password cannot be empty")
		}
		hash, err := s.hashPassword(credentials.Password)
		if err != nil {
			return nil, "", fmt.Errorf("failed to hash password: %w", err)
		}
		userObj.Password = hash
		if err := s.userRepo.UpdateUser(userObj); err != nil {
			return nil, "", fmt.Errorf("failed to update bot password: %w", err)
		}
	} else {
		ok, err := s.comparePassword(userObj.Password, credentials.Password)
		if err != nil {
			return nil, "", err
		}
		if !ok {
			return nil, "", errors.New("invalid credentials")
		}
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

func (s *UserService) applyReferral(ctx context.Context, referral string, userID uuid.UUID) {
	referralUser, err := s.resolveReferralUser(ctx, referral)
	if err != nil {
		return
	}
	if referralUser.ID == userID {
		return
	}

	if _, err := s.userRepo.AddReferral(ctx, referralUser.ID, userID, constants.DEFAULT_REFERRAL_REWARD); err != nil {
		return
	}
}

func (s *UserService) resolveReferralUser(ctx context.Context, referral string) (*models.User, error) {
	referral = strings.TrimSpace(referral)
	if referral == "" {
		return nil, errors.New("referral is empty")
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
		return false, errors.New(constants.ErrUserNotFound.String())
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

func (s *UserService) UpdateAvatar(ctx context.Context, file ports.UploadedFile, user *models.User) (*media.Media, error) {
	newMedia, err := s.mediaRepo.AddMedia(
		user.ID,
		media.OwnerUser,
		user.ID,
		media.RoleAvatar,
		file,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload avatar: %w", err)
	}

	// User tablosunu güncelle
	user.AvatarID = &newMedia.ID
	user.Avatar = newMedia

	if err := s.userRepo.UpdateUser(user); err != nil {
		return nil, fmt.Errorf("failed to update user avatar: %w", err)
	}
	return newMedia, nil
}

func (s *UserService) UpdateAvatarFromURL(ctx context.Context, imgUrl string, user *models.User) (*media.Media, error) {
	if s.remoteImage == nil {
		return nil, errors.New("remote image fetcher is not configured")
	}

	file, err := s.remoteImage.FetchImage(ctx, imgUrl)
	if err != nil {
		return nil, err
	}

	return s.UpdateAvatar(ctx, file, user)
}

func (s *UserService) UpdateCover(ctx context.Context, file ports.UploadedFile, user *models.User) (*media.Media, error) {
	//
	newMedia, err := s.mediaRepo.AddMedia(
		user.ID,
		media.OwnerUser,
		user.ID,
		media.RoleCover,
		file,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload avatar: %w", err)
	}
	user.CoverID = &newMedia.ID
	user.Cover = newMedia

	if err := s.userRepo.UpdateUser(user); err != nil {
		return nil, fmt.Errorf("failed to update user avatar: %w", err)
	}
	return newMedia, nil
}

func (s *UserService) AddStory(ctx context.Context, form ports.FormData, user *models.User) (*post.Post, error) {
	form.Values = cloneFormValues(form.Values)
	if form.Values == nil {
		form.Values = make(map[string][]string)
	}

	if caption := firstFormValue(form.Values, "caption"); caption != "" && firstFormValue(form.Values, "content") == "" {
		form.Values["content"] = []string{caption}
	}
	if firstFormValue(form.Values, "audience") == "" {
		form.Values["audience"] = []string{"public"}
	}
	if firstFormValue(form.Values, "slug") == "" && firstFormValue(form.Values, "title") == "" {
		form.Values["slug"] = []string{fmt.Sprintf("story-%d", time.Now().UnixNano())}
	}
	if firstFormValue(form.Values, "extras") == "" {
		extras, err := json.Marshal(map[string]string{
			"expires_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		})
		if err != nil {
			return nil, err
		}
		form.Values["extras"] = []string{string(extras)}
	}

	createdPost, err := s.postRepo.CreateContentablePost(ctx, form, user, string(post.PostKindStory), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.GetPostByIDIncludingUnpublished(createdPost.ID)
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

func (s *UserService) UpsertUserPreference(ctx context.Context, user models.User, preferenceItemId string, bitIndexStr string, enabled bool) error {
	err := s.userRepo.UpsertUserPreference(ctx, user, preferenceItemId, bitIndexStr, enabled)
	if err != nil {
		return fmt.Errorf("failed to upsert user attribute: %w", err)
	}
	return err
}

func (s *UserService) GetAllStories(filters types.Filter) (types.PostsResult, error) {
	filters.PostKind = post.PostKindStory
	return s.postRepo.GetPostsByKind(filters)
}

func (s *UserService) FetchNearbyUsers(filters types.Filter) ([]*models.User, *float64, error) {
	return s.userRepo.FetchNearbyUsers(filters)
}

func (s *UserService) FetchLiveUsers(filters types.Filter) ([]*models.User, error) {
	return s.userRepo.FetchLiveUsers(filters)
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

func (s *UserService) Follow(ctx context.Context, followerID, followeeID int64) (bool, error) {
	return s.HandleFollow(ctx, followerID, followeeID, true)
}

func (s *UserService) Unfollow(ctx context.Context, followerID, followeeID int64) (bool, error) {
	return s.HandleFollow(ctx, followerID, followeeID, false)
}

func (s *UserService) HandleFollow(ctx context.Context, followerID, followeeID int64, isFollow bool) (bool, error) {
	return s.ToggleFollow(ctx, followerID, followeeID)
}

func (s *UserService) ToggleFollow(ctx context.Context, followerID, followeeID int64) (bool, error) {
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

	pair, err := domainuser.InteractionEngagementPair(domainuser.InteractionFollow, true)
	if err != nil {
		return false, err
	}

	//takip et
	_, err = s.engagementRepo.ToggleEngagement(ctx, followerUser.ID, followeeUser.ID, models.EngagementKind(pair.Given), followerUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return false, err
	}
	// takipcilere yaz
	_, err = s.engagementRepo.ToggleEngagement(ctx, followeeUser.ID, followerUser.ID, models.EngagementKind(pair.Received), followeeUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return false, err
	}

	isFollowing, err := s.engagementRepo.HasUserEngaged(ctx, followerUser.ID, followeeUser.ID, models.EngagementKind(pair.Given))
	if err != nil {
		return false, err
	}

	fmt.Println("isFollowing:", isFollowing)

	if isFollowing {
		// Follow started
		notificationTitleToFollowee := "New Follower"
		notificationBodyToFollowee := followerUser.UserName + " started following you."

		payloadToFollowee := notifications.NotificationPayload{
			Title: notificationTitleToFollowee,
			Body:  notificationBodyToFollowee,
		}
		err := s.notificationRepo.SendNotificationToUser(*followerUser, *followeeUser, notifications.NotificationTypeFollow, notificationTitleToFollowee, notificationBodyToFollowee, payloadToFollowee)
		if err != nil {
			fmt.Printf("Failed to send notification to user %d: %v\n", followeeUser.ID, err)
		}

		notificationTitleToFollower := "Follow Started"
		notificationBodyToFollower := "You started following " + followeeUser.UserName + "."

		payloadToFollower := notifications.NotificationPayload{
			Title: notificationTitleToFollower,
			Body:  notificationBodyToFollower,
		}
		err = s.notificationRepo.SendNotificationToUser(*followeeUser, *followerUser, notifications.NotificationTypeFollow, notificationTitleToFollower, notificationBodyToFollower, payloadToFollower)
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
		err := s.notificationRepo.SendNotificationToUser(*followerUser, *followeeUser, notifications.NotificationTypeUnFollow, notificationTitleToFollowee, notificationBodyToFollowee, payloadToFollowee)
		if err != nil {
			fmt.Printf("Failed to send notification to user %d: %v\n", followeeUser.ID, err)
		}

		notificationTitleToFollower := "Unfollowed"
		notificationBodyToFollower := "You unfollowed " + followeeUser.UserName + "."

		payloadToFollower := notifications.NotificationPayload{
			Title: notificationTitleToFollower,
			Body:  notificationBodyToFollower,
		}
		err = s.notificationRepo.SendNotificationToUser(*followeeUser, *followerUser, notifications.NotificationTypeUnFollow, notificationTitleToFollower, notificationBodyToFollower, payloadToFollower)
		if err != nil {
			fmt.Printf("Failed to send notification to user %d: %v\n", followerUser.ID, err)
		}
	}

	if err := s.publishEvent(ctx, domainuser.NewInteractionToggledEvent(followerID, followeeID, domainuser.InteractionFollow, isFollowing, time.Now().UTC())); err != nil {
		return false, err
	}

	return isFollowing, nil
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

func (s *UserService) UpdateUserProfile(context context.Context, authUser models.User, input UpdateUserProfileInput) (*models.User, error) {
	username := domainuser.NormalizeUsername(input.UserName)
	if username != "" {
		existsUser, err := s.userRepo.GetByNameOrMailWithoutRelations(username)
		if err == nil && existsUser.ID != authUser.ID {
			return nil, errors.New(constants.ErrUsernameTaken.String())
		}
	}

	userInfo, err := s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: context, UserUUID: authUser.ID})
	if err != nil {
		return nil, err
	}

	if input.CurrentPassword != "" {
		ok, err := s.comparePassword(authUser.Password, input.CurrentPassword)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New(constants.ErrInvalidPassword.String())
		}
	}

	dateOfBirth, err := domainuser.ParseBirthDate(input.DateOfBirth, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if dateOfBirth != nil {
		userInfo.DateOfBirth = dateOfBirth
	}

	userInfo.UserName = defaultIfEmpty(username, userInfo.UserName)
	userInfo.DisplayName = defaultIfEmpty(domainuser.NormalizeDisplayName(input.DisplayName), userInfo.DisplayName)

	userInfo.Bio = utils.MakeLocalizedString(userInfo.DefaultLanguage, defaultIfEmpty(input.Bio, userInfo.Bio.GetLocalizedString(userInfo.DefaultLanguage)))

	//userObj.Website = formData.Website

	privacyLevel, hasPrivacyLevel, err := domainuser.ParsePrivacyLevel(input.PrivacyLevel)
	if err != nil {
		return nil, err
	}
	if hasPrivacyLevel {
		userInfo.PrivacyLevel = constants.PrivacyLevel(privacyLevel)
	}

	if err := s.userRepo.UpdateUser(userInfo); err != nil {
		return nil, err
	}

	if input.LocationLatitude != "" && input.LocationLongitude != "" {

		lat, err := strconv.ParseFloat(input.LocationLatitude, 64)
		if err != nil {
			return nil, errors.New(constants.ErrInvalidLatitude.String())
		}
		lng, err := strconv.ParseFloat(input.LocationLongitude, 64)
		if err != nil {
			return nil, errors.New(constants.ErrInvalidLongitude.String())
		}

		coordinates, err := domainuser.NewCoordinates(lat, lng)
		if err != nil {
			return nil, err
		}

		locationUser := &utils.Location{
			ID:              uuid.New(),
			ContentableType: utils.LocationOwnerUser,
			ContentableID:   userInfo.ID,
			CountryCode:     &input.LocationCountryCode,
			Country:         &input.LocationCountry,
			City:            &input.LocationCity,
			Region:          &input.LocationRegion,
			Display:         &input.LocationDisplay,
			Timezone:        &input.LocationTimezone,
			Address:         &input.LocationAddress,
			Latitude:        &coordinates.Latitude,
			Longitude:       &coordinates.Longitude,
			LocationPoint:   utils.NewLocationPoint(coordinates.Latitude, coordinates.Longitude),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := s.userRepo.UpsertLocation(locationUser); err != nil {
			return nil, err
		}
	}

	return s.GetUserByID(authUser.ID)
}

func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
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
	if err := domainuser.EnsureDifferentPublicUsers(likerId, likeeId, domainuser.InteractionLike); err != nil {
		return isLike, false, err
	}

	likerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: likerId})
	if err != nil {
		return isLike, false, errors.New(err.Error())
	}
	likeeUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: likeeId})
	if err != nil {
		return isLike, false, errors.New(err.Error())
	}

	pair, err := domainuser.InteractionEngagementPair(domainuser.InteractionLike, isLike)
	if err != nil {
		return isLike, false, err
	}
	engagementKindGiven := models.EngagementKind(pair.Given)
	engagementKindReceived := models.EngagementKind(pair.Received)

	status, err := s.engagementRepo.ToggleEngagement(ctx, likerUser.ID, likeeUser.ID, engagementKindGiven, likerUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return isLike, status, err
	}
	enabled := status

	status, err = s.engagementRepo.ToggleEngagement(ctx, likeeUser.ID, likerUser.ID, engagementKindReceived, likeeUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return isLike, status, err
	}

	if err := s.publishEvent(ctx, domainuser.NewInteractionToggledEvent(likerId, likeeId, domainuser.InteractionLike, enabled, time.Now().UTC())); err != nil {
		return isLike, false, err
	}

	return isLike, enabled, nil
}

// return Params : bool isLike, bool success, error
func (s *UserService) Block(ctx context.Context, authUser models.User, blockerId, blockedId int64) (bool, error) {
	return s.HandleBlock(ctx, authUser, blockerId, blockedId, true)
}

func (s *UserService) Unblock(ctx context.Context, authUser models.User, blockerId, blockedId int64) (bool, error) {
	return s.HandleBlock(ctx, authUser, blockerId, blockedId, false)
}

func (s *UserService) HandleBlock(ctx context.Context, authUser models.User, blockerId, blockedId int64, isBlock bool) (bool, error) {
	return s.ToggleBlock(ctx, authUser, blockerId, blockedId)
}

func (s *UserService) ToggleBlock(ctx context.Context, authUser models.User, blockerId, blockedId int64) (bool, error) {

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

	pair, err := domainuser.InteractionEngagementPair(domainuser.InteractionBlock, true)
	if err != nil {
		return false, err
	}
	engagementKindGiven := models.EngagementKind(pair.Given)
	engagementKindReceived := models.EngagementKind(pair.Received)

	isBlocked, _ := s.engagementRepo.HasUserEngaged(ctx, blockerUser.ID, blockedUser.ID, engagementKindGiven)

	status, err := s.engagementRepo.ToggleEngagement(ctx, blockerUser.ID, blockedUser.ID, engagementKindGiven, blockerUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return status, err
	}

	status, err = s.engagementRepo.ToggleEngagement(ctx, blockedUser.ID, blockerUser.ID, engagementKindReceived, blockedUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return status, err
	}

	enabled := !isBlocked
	if err := s.publishEvent(ctx, domainuser.NewInteractionToggledEvent(blockerId, blockedId, domainuser.InteractionBlock, enabled, time.Now().UTC())); err != nil {
		return false, err
	}

	return enabled, nil
}

func (s *UserService) ToggleSubscribe(ctx context.Context, authUser models.User, subscriberId, subscribedId int64) (bool, error) {

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

	pair, err := domainuser.InteractionEngagementPair(domainuser.InteractionSubscribe, true)
	if err != nil {
		return false, err
	}
	engagementKindGiven := models.EngagementKind(pair.Given)
	engagementKindReceived := models.EngagementKind(pair.Received)

	isBlocked, _ := s.engagementRepo.HasUserEngaged(ctx, subscriberUser.ID, subscribedUser.ID, engagementKindGiven)

	status, err := s.engagementRepo.ToggleEngagement(ctx, subscriberUser.ID, subscribedUser.ID, engagementKindGiven, subscriberUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return status, err
	}

	status, err = s.engagementRepo.ToggleEngagement(ctx, subscribedUser.ID, subscriberUser.ID, engagementKindReceived, subscribedUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return status, err
	}

	enabled := !isBlocked
	if err := s.publishEvent(ctx, domainuser.NewInteractionToggledEvent(subscriberId, subscribedId, domainuser.InteractionSubscribe, enabled, time.Now().UTC())); err != nil {
		return false, err
	}

	return enabled, nil
}

func (s *UserService) FetchUserNotifications(ctx context.Context, authUser *models.User, cursor *time.Time, limit int) (items []*notifications.Notification, nextCursor *time.Time, err error) {
	return s.userRepo.FetchUserNotifications(ctx, authUser, cursor, limit)
}

func (s *UserService) Report(ctx context.Context, userPublicID int64, kind string, description string, authUser *models.User) error {
	if userPublicID <= 0 {
		return ErrUserIDRequired
	}
	if authUser == nil {
		return errors.New("authenticated user is required")
	}
	if authUser.PublicID == userPublicID {
		return ErrCannotReportSelf
	}
	return s.userRepo.Report(ctx, userPublicID, kind, description, authUser)
}

func (s *UserService) FetchUserEngagements(ctx context.Context, authUser *models.User, contentableID uuid.UUID, contentableType models.EngagementContentableType, engagementKind models.EngagementKind, cursor *time.Time, limit int) ([]models.EngagementDetail, *time.Time, error) {
	if (engagementKind == models.EngagementKindViewGiven || engagementKind == models.EngagementKindViewReceived) &&
		(authUser == nil || authUser.ID != contentableID) {
		return nil, nil, ErrPrivateViewEngagements
	}
	return s.engagementRepo.GetEngagements(ctx, contentableType, contentableID, engagementKind, cursor, limit)
}

func (s *UserService) CheckIn(context context.Context, form ports.FormData, author *models.User, postKind post.PostKind) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(context, form, author, string(postKind), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.GetPostByIDIncludingUnpublished(_post.ID)
}

func (s *UserService) FetchCheckIns(filters types.Filter) (types.PostsResult, error) {
	return s.postRepo.GetPostsByKind(filters)
}

func (s *UserService) DeleteUser(filters types.Filter) error {
	return s.userRepo.DeleteUser(filters)

}
