package services

import (
	"bytes"
	"context"
	"core/constants"
	"core/extensions"
	"core/helpers"
	"core/models"
	"core/models/media"
	"core/models/notifications"
	"core/models/post"
	"core/models/utils"
	"core/repositories"
	"core/types"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	form "github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	mediaRepo        *repositories.MediaRepository
	userRepo         *repositories.UserRepository
	postRepo         *repositories.PostRepository
	engagementRepo   *repositories.EngagementRepository
	notificationRepo *repositories.NotificationRepository
}

func NewUserService(
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository,
	engagementRepo *repositories.EngagementRepository,
	notificationRepo *repositories.NotificationRepository,
) *UserService {
	return &UserService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo, notificationRepo: notificationRepo, engagementRepo: engagementRepo}
}

func (s *UserService) UserRepository() *repositories.UserRepository {
	return s.userRepo
}

// Register işlemi
func (s *UserService) Register(context context.Context, request map[string][]string) (*models.User, string, error) {

	type RegisterForm struct {
		Name     string `form:"name"`
		Nickname string `form:"nickname"`
		Password string `form:"password"`
		Domain   string `form:"domain"`
		Email    string `form:"email"`
		Captcha  string `form:"recaptchaToken"`
		Referral string `form:"referralCode"`
	}
	decoder := form.NewDecoder()
	var formData RegisterForm

	if err := decoder.Decode(&formData, request); err != nil {
		return nil, "", err
	}

	captchaSecret := os.Getenv("CAPTCHA_SECRET_KEY")
	if len(captchaSecret) == 0 {
		log.Println("ENV CAPTCHA_SECRET_KEY is not set")
	}

	captchaValid, captchaErr := s.userRepo.VerifyCaptcha(captchaSecret, formData.Captcha)
	if captchaErr != nil {
		return nil, "", errors.New(captchaErr.Error())
	}

	if !captchaValid {
		return nil, "", errors.New("invalid captcha")
	}

	formData.Nickname = strings.ToLower(formData.Nickname)
	formData.Password = strings.ToLower(formData.Password)
	formData.Email = strings.ToLower(formData.Email)

	if len(formData.Email) > 0 {
		if !helpers.IsValidEmail(formData.Email) {
			return nil, "", errors.New("invalid email")
		}
	}

	domain := models.GetDomainKind(formData.Domain)
	if !models.IsValidDomainByKind(domain) {
		return nil, "", errors.New("invalid domain")
	}

	hash, err := helpers.HashPasswordArgon2id(formData.Password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create hash password: %w", err)
	}

	existingUser, err := s.userRepo.GetByUserNameOrEmailOrUsername(formData.Nickname)
	if err == nil && existingUser != nil {
		return nil, "", errors.New("username already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}

	UserID := uuid.New()

	userObj := &models.User{
		ID:          UserID,
		PublicID:    s.userRepo.Node().Generate().Int64(),
		Domain:      models.GetDomainKind(formData.Domain),
		UserName:    formData.Name,
		DisplayName: formData.Nickname,
		Email:       formData.Email,
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

	if len(formData.Referral) > 0 {
		referralId, err := helpers.StrToInt64(formData.Referral)
		if err == nil {
			referralUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: context, UserID: referralId})
			if err == nil {
				reward := constants.DEFAULT_REFERRAL_REWARD
				newBalance, err := s.userRepo.AddReferral(context, referralUser.ID, userInfo.ID, reward)
				if err != nil {
					fmt.Println("REFERRAL ERROR3", err)
				}
				fmt.Println("NEW BALANCE", newBalance)
			} else {
				fmt.Println("REFERRAL ERROR2", err)
			}
		} else {
			fmt.Println("REFERRAL ERROR1", err)
		}
	}

	token, err := helpers.GenerateUserJWT(userObj.ID, userObj.PublicID)
	if err != nil {
		return nil, "", err
	}

	return userInfo, token, nil
}

func (s *UserService) Login(context context.Context, request map[string][]string) (*models.User, string, error) {
	// Form yapısı
	type LoginForm struct {
		UserName string `form:"nickname"`
		Password string `form:"password"`
	}

	decoder := form.NewDecoder()
	var formData LoginForm

	if err := decoder.Decode(&formData, request); err != nil {
		return nil, "", err
	}

	formData.Password = strings.ToLower(formData.Password)
	formData.UserName = strings.ToLower(formData.UserName)

	// Kullanıcıyı username ile bul (repo'da buna uygun fonksiyon olmalı)
	userObj, err := s.userRepo.GetByUserNameOrEmailOrUsername(formData.UserName)
	fmt.Println(err)
	if err != nil {
		return nil, "", errors.New("invalid username/email/nickname or password")
	}

	if userObj.IsBot && userObj.Password == "" {
		if formData.Password == "" {
			return nil, "", errors.New("password cannot be empty")
		}
		hash, err := helpers.HashPasswordArgon2id(formData.Password)
		if err != nil {
			return nil, "", fmt.Errorf("failed to hash password: %w", err)
		}
		userObj.Password = hash
		if err := s.userRepo.UpdateUser(userObj); err != nil {
			return nil, "", fmt.Errorf("failed to update bot password: %w", err)
		}
	} else {
		ok, err := helpers.ComparePasswordArgon2id(userObj.Password, formData.Password)
		if err != nil {
			return nil, "", err // Karşılaştırma sırasında hata
		}
		if !ok {
			return nil, "", errors.New("invalid credentials") // Şifre yanlış
		}
	}

	// Token üret
	token, err := helpers.GenerateUserJWT(userObj.ID, userObj.PublicID)
	if err != nil {
		return nil, "", err
	}

	return userObj, token, nil
}

func (s *UserService) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *UserService) FetchUserProfileByUsername(username string) (*models.User, error) {
	return s.userRepo.GetByUserNameOrEmailOrUsername(username)
}

func (s *UserService) CreateBotUser(ctx context.Context, userObj *models.User) (*models.User, error) {
	userObj.ID = uuid.New()
	userObj.PublicID = s.userRepo.Node().Generate().Int64()
	userObj.Domain = models.CoolVibes
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

func (s *UserService) UpdateAvatar(ctx context.Context, file *multipart.FileHeader, user *models.User) (*media.Media, error) {
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
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(imgUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image body: %w", err)
	}

	ext := filepath.Ext(imgUrl)
	if ext == "" {
		ext = ".jpg"
	}
	indexQuery := strings.Index(ext, "?")
	if indexQuery != -1 {
		ext = ext[:indexQuery]
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(data)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	h.Set("Content-Type", mimeType)

	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(data) + 1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read form: %w", err)
	}

	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		return nil, errors.New("file header is empty")
	}
	fh := fileHeaders[0]

	return s.UpdateAvatar(ctx, fh, user)
}

func (s *UserService) UpdateCover(ctx context.Context, file *multipart.FileHeader, user *models.User) (*media.Media, error) {
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

func (s *UserService) AddStory(ctx context.Context, file *multipart.FileHeader, user *models.User) (*models.Story, error) {
	storyMedia, err := s.mediaRepo.AddMedia(
		user.ID,
		media.OwnerUser,
		user.ID,
		media.RoleStory,
		file,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upload avatar: %w", err)
	}

	story := &models.Story{
		ID:         uuid.New(),
		UserID:     user.ID,
		MediaID:    storyMedia.ID,
		Caption:    nil,                            // istersen ekleyebilirsin
		ExpiresAt:  time.Now().Add(24 * time.Hour), // örneğin 24 saat sonra silinecek
		IsExpired:  false,
		IsArchived: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.userRepo.AddStory(user.ID, story); err != nil {
		return nil, fmt.Errorf("failed to update user avatar: %w", err)
	}
	story.Media = storyMedia
	return story, nil
}

func (s *UserService) UpsertUserPreference(ctx context.Context, user models.User, preferenceItemId string, bitIndexStr string, enabled bool) error {
	err := s.userRepo.UpsertUserPreference(ctx, user, preferenceItemId, bitIndexStr, enabled)
	if err != nil {
		return fmt.Errorf("failed to upsert user attribute: %w", err)
	}
	return err
}

func (s *UserService) GetAllStories(filters types.Filter) ([]*models.Story, error) {
	return s.userRepo.GetAllStories(filters)
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
	followerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: followerID})
	if err != nil {
		return false, err
	}
	followeeUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: followeeID})
	if err != nil {
		return false, err
	}

	engagementRepo := s.userRepo.GetEngagementRepository()
	notificationRepo := s.notificationRepo

	//takip et
	_, err = engagementRepo.ToggleEngagement(ctx, followerUser.ID, followeeUser.ID, models.EngagementKindFollowing, followerUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return false, err
	}
	// takipcilere yaz
	_, err = engagementRepo.ToggleEngagement(ctx, followeeUser.ID, followerUser.ID, models.EngagementKindFollower, followeeUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return false, err
	}

	isFollowing, err := engagementRepo.HasUserEngaged(ctx, followerUser.ID, followeeUser.ID, models.EngagementKindFollowing)
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
		err := notificationRepo.SendNotificationToUser(*followerUser, *followeeUser, notifications.NotificationTypeFollow, notificationTitleToFollowee, notificationBodyToFollowee, payloadToFollowee)
		if err != nil {
			fmt.Printf("Failed to send notification to user %d: %v\n", followeeUser.ID, err)
		}

		notificationTitleToFollower := "Follow Started"
		notificationBodyToFollower := "You started following " + followeeUser.UserName + "."

		payloadToFollower := notifications.NotificationPayload{
			Title: notificationTitleToFollower,
			Body:  notificationBodyToFollower,
		}
		err = notificationRepo.SendNotificationToUser(*followeeUser, *followerUser, notifications.NotificationTypeFollow, notificationTitleToFollower, notificationBodyToFollower, payloadToFollower)
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
		err := notificationRepo.SendNotificationToUser(*followerUser, *followeeUser, notifications.NotificationTypeUnFollow, notificationTitleToFollowee, notificationBodyToFollowee, payloadToFollowee)
		if err != nil {
			fmt.Printf("Failed to send notification to user %d: %v\n", followeeUser.ID, err)
		}

		notificationTitleToFollower := "Unfollowed"
		notificationBodyToFollower := "You unfollowed " + followeeUser.UserName + "."

		payloadToFollower := notifications.NotificationPayload{
			Title: notificationTitleToFollower,
			Body:  notificationBodyToFollower,
		}
		err = notificationRepo.SendNotificationToUser(*followeeUser, *followerUser, notifications.NotificationTypeUnFollow, notificationTitleToFollower, notificationBodyToFollower, payloadToFollower)
		if err != nil {
			fmt.Printf("Failed to send notification to user %d: %v\n", followerUser.ID, err)
		}
	}

	return isFollowing, nil
}

func (s *UserService) UpdateUserProfile(context context.Context, authUser models.User, request map[string][]string) (*models.User, error) {
	// Form yapısı
	type UserProfileForm struct {
		UserName                string `form:"username"`
		Password                string `form:"password"`                  // Şifre formda geliyorsa
		CurrentPassword         string `form:"current_password"`          // Şifre formda geliyorsa
		NewPassword             string `form:"new_password"`              // Şifre formda geliyorsa
		NewPasswordConfirmation string `form:"new_password_confirmation"` // Şifre formda geliyorsa
		Email                   string `form:"email"`
		DisplayName             string `form:"displayname"`
		Bio                     string `form:"bio"`
		Website                 string `form:"website"`
		DateOfBirth             string `form:"date_of_birth"`
		PrivacyLevel            string `form:"privacy_level"`
		LocationContentableType string `form:"location[contentable_type]"`
		LocationCountryCode     string `form:"location[country_code]"`
		LocationAddress         string `form:"location[address]"`
		LocationCity            string `form:"location[city]"`
		LocationCountry         string `form:"location[country]"`
		LocationRegion          string `form:"location[region]"`
		LocationTimezone        string `form:"location[timezone]"`
		LocationDisplay         string `form:"location[display]"`
		LocationLatitude        string `form:"location[latitude]"`
		LocationLongitude       string `form:"location[longitude]"`
	}

	decoder := form.NewDecoder()
	var formData UserProfileForm

	if err := decoder.Decode(&formData, request); err != nil {
		return nil, err
	}

	existsUser, err := s.userRepo.GetByNameOrMailWithoutRelations(formData.UserName)
	if err == nil && existsUser.ID != authUser.ID {
		return nil, errors.New(constants.ErrUsernameTaken.String())
	}

	userInfo, err := s.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: context, UserUUID: authUser.ID})
	if err != nil {
		return nil, err
	}

	if formData.CurrentPassword != "" {
		ok, err := helpers.ComparePasswordArgon2id(authUser.Password, formData.CurrentPassword)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New(constants.ErrInvalidPassword.String())
		}
	}

	if formData.DateOfBirth != "" {
		dateOfBirth, err := time.Parse("2006-01-02", formData.DateOfBirth)
		if err == nil {
			userInfo.DateOfBirth = &dateOfBirth
		}
	}

	userInfo.UserName = helpers.DefaultIfEmpty(formData.UserName, userInfo.UserName)
	userInfo.DisplayName = helpers.DefaultIfEmpty(formData.DisplayName, userInfo.DisplayName)

	userInfo.Bio = utils.MakeLocalizedString(userInfo.DefaultLanguage, helpers.DefaultIfEmpty(formData.Bio, userInfo.Bio.GetLocalizedString(userInfo.DefaultLanguage)))

	//userObj.Website = formData.Website

	userInfo.PrivacyLevel = constants.PrivacyLevel(formData.PrivacyLevel)

	if err := s.userRepo.UpdateUser(userInfo); err != nil {
		return nil, err
	}

	if formData.LocationLatitude != "" && formData.LocationLongitude != "" {

		lat, err := helpers.ParseFloat(formData.LocationLatitude)
		if err != nil {
			return nil, errors.New(constants.ErrInvalidLatitude.String())
		}
		lng, err := helpers.ParseFloat(formData.LocationLongitude)
		if err != nil {
			return nil, errors.New(constants.ErrInvalidLongitude.String())
		}

		locationPoint := &extensions.PostGISPoint{
			Lat: lat,
			Lng: lng,
		}

		locationUser := &utils.Location{
			ID:              uuid.New(),
			ContentableType: utils.LocationOwnerUser,
			ContentableID:   userInfo.ID,
			CountryCode:     &formData.LocationCountryCode,
			Country:         &formData.LocationCountry,
			City:            &formData.LocationCity,
			Region:          &formData.LocationRegion,
			Display:         &formData.LocationDisplay,
			Timezone:        &formData.LocationTimezone,
			Address:         &formData.LocationAddress,
			Latitude:        &lat,
			Longitude:       &lng,
			LocationPoint:   locationPoint,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		if err := s.userRepo.UpsertLocation(locationUser); err != nil {
			return nil, err
		}
	}

	return s.GetUserByID(authUser.ID)
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
	likerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: likerId})
	if err != nil {
		return isLike, false, errors.New(err.Error())
	}
	likeeUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: likeeId})
	if err != nil {
		return isLike, false, errors.New(err.Error())
	}

	engagementRepo := s.userRepo.GetEngagementRepository()

	var engagementKindGiven models.EngagementKind
	var engagementKindReceived models.EngagementKind

	switch {
	case isLike:
		engagementKindGiven = models.EngagementKindLikeGiven
		engagementKindReceived = models.EngagementKindLikeReceived
	default:
		engagementKindGiven = models.EngagementKindDislikeGiven
		engagementKindReceived = models.EngagementKindDisLikeReceived

	}

	status, err := engagementRepo.ToggleEngagement(ctx, likerUser.ID, likeeUser.ID, engagementKindGiven, likerUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return isLike, status, err
	}

	status, err = engagementRepo.ToggleEngagement(ctx, likeeUser.ID, likerUser.ID, engagementKindReceived, likeeUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return isLike, status, err
	}

	return isLike, true, nil
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

	if blockerId == blockedId {
		return false, errors.New("you cannot block yourself")
	}

	blockerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: blockerId})
	if err != nil {
		return false, err
	}
	blockedUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(types.Filter{Context: ctx, UserID: blockedId})
	if err != nil {
		return false, err
	}

	engagementRepo := s.userRepo.GetEngagementRepository()

	var engagementKindGiven models.EngagementKind
	var engagementKindReceived models.EngagementKind

	engagementKindGiven = models.EngagementKindBlocking
	engagementKindReceived = models.EngagementKindBlockedBy

	isBlocked, _ := engagementRepo.HasUserEngaged(ctx, blockerUser.ID, blockedUser.ID, engagementKindGiven)

	status, err := engagementRepo.ToggleEngagement(ctx, blockerUser.ID, blockedUser.ID, engagementKindGiven, blockerUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return status, err
	}

	status, err = engagementRepo.ToggleEngagement(ctx, blockedUser.ID, blockerUser.ID, engagementKindReceived, blockedUser.ID, models.EngagementContentableTypeUser)
	if err != nil {
		return status, err
	}

	return !isBlocked, nil
}

func (s *UserService) FetchUserNotifications(ctx context.Context, authUser *models.User, cursor *time.Time, limit int) (items []*notifications.Notification, nextCursor *time.Time, err error) {
	return s.userRepo.FetchUserNotifications(ctx, authUser, cursor, limit)
}

func (s *UserService) FetchUserEngagements(ctx context.Context, authUser *models.User, contentableID uuid.UUID, contentableType models.EngagementContentableType, engagementKind models.EngagementKind, cursor *time.Time, limit int) ([]models.EngagementDetail, *time.Time, error) {
	return s.engagementRepo.GetEngagements(ctx, contentableType, contentableID, engagementKind, cursor, limit)
}

func (s *UserService) CheckIn(context context.Context, request map[string][]string, files []*multipart.FileHeader, author *models.User, postKind post.PostKind) (*post.Post, error) {
	_post, err := s.postRepo.CreateContentablePost(context, request, files, author, string(postKind), nil)
	if err != nil {
		return nil, err
	}
	return s.postRepo.GetPostByID(_post.ID)
}

func (s *UserService) FetchCheckIns(filters types.Filter) (types.PostsResult, error) {
	return s.postRepo.GetPostsByKind(filters)
}

func (s *UserService) DeleteUser(filters types.Filter) error {
	return s.userRepo.DeleteUser(filters)

}
