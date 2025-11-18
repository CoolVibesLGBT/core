package services

import (
	"context"
	"coolvibes/constants"
	"coolvibes/extensions"
	"coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/models/media"
	"coolvibes/models/notifications"
	"coolvibes/models/utils"
	"coolvibes/repositories"
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	form "github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	mediaRepo *repositories.MediaRepository
	userRepo  *repositories.UserRepository
	postRepo  *repositories.PostRepository
}

func NewUserService(
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository) *UserService {
	return &UserService{postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo}
}

// Register işlemi
func (s *UserService) Register(request map[string][]string) (*models.User, string, error) {

	type RegisterForm struct {
		Name      string `form:"name"`
		Nickname  string `form:"nickname"`
		Password  string `form:"password"`
		BirthDate string `form:"birthDate"`      // string veya time.Time
		Captcha   string `form:"recaptchaToken"` // string veya time.Time
		// Nested location
		CountryCode string  `form:"location[country_code]"`
		Country     string  `form:"location[country_name]"`
		City        string  `form:"location[city]"`
		Region      string  `form:"location[region]"`
		Lat         float64 `form:"location[lat]"`
		Lng         float64 `form:"location[lng]"`
		Timezone    string  `form:"location[timezone]"`
		Display     string  `form:"location[display]"`
		Address     string  `form:"location[address]"` // varsa
	}
	decoder := form.NewDecoder()
	var formData RegisterForm

	// formValues map[string][]string şeklinde gelecek
	if err := decoder.Decode(&formData, request); err != nil {
		return nil, "", err
	}

	captchaValid, captchaErr := s.userRepo.VerifyCaptcha("6LecaQIsAAAAAE2vz3YKi5jFOWIOzXEpMX4675ox", formData.Captcha)
	if captchaErr != nil {
		return nil, "", errors.New("invalid  captcha")
	}

	if !captchaValid {
		return nil, "", errors.New("invalid captcha")
	}

	formData.Nickname = strings.ToLower(formData.Nickname)
	formData.Password = strings.ToLower(formData.Password)

	// BirthDate
	dateOfBirth, err := time.Parse("2006-01-02", formData.BirthDate)
	if err != nil {
		return nil, "", errors.New("invalid birthDate")
	}

	// Hashle
	hash, err := helpers.HashPasswordArgon2id(formData.Password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create hash password: %w", err)
	}

	existingUser, err := s.userRepo.GetByUserNameOrEmailOrNickname(formData.Nickname)
	if err == nil && existingUser != nil {
		return nil, "", errors.New("username already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// başka bir hata varsa onu da döndür
		return nil, "", err
	}

	locationPoint := &extensions.PostGISPoint{
		Lat: formData.Lat,
		Lng: formData.Lng,
	}

	UserID := uuid.New()
	locationUser := &utils.Location{
		ID:              uuid.New(),
		ContentableType: utils.LocationOwnerUser,
		ContentableID:   UserID,

		CountryCode:   &formData.CountryCode,
		Country:       &formData.Country,
		City:          &formData.City,
		Region:        &formData.Region,
		Display:       &formData.Display,
		Timezone:      &formData.Timezone,
		Address:       &formData.Address,
		Latitude:      &formData.Lat,
		Longitude:     &formData.Lng,
		LocationPoint: locationPoint,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.userRepo.UpsertLocation(locationUser); err != nil {
		return nil, "", err
	}

	userObj := &models.User{

		ID:          UserID,
		PublicID:    s.userRepo.Node().Generate().Int64(),
		UserName:    formData.Name,
		DisplayName: formData.Nickname,
		DateOfBirth: &dateOfBirth,
		Password:    hash,
	}

	if err := s.userRepo.Create(userObj); err != nil {
		return nil, "", err
	}

	userInfo, err := s.GetUserByID(userObj.ID)
	if err != nil {
		return nil, "", err
	}
	token, err := helpers.GenerateUserJWT(userObj.ID, userObj.PublicID)
	if err != nil {
		return nil, "", err
	}

	return userInfo, token, nil
}

func (s *UserService) Login(request map[string][]string) (*models.User, string, error) {
	// Form yapısı
	type LoginForm struct {
		UserName string `form:"nickname"`
		Password string `form:"password"`

		CountryCode string  `form:"location[country_code]"`
		Country     string  `form:"location[country_name]"`
		City        string  `form:"location[city]"`
		Region      string  `form:"location[region]"`
		Lat         float64 `form:"location[lat]"`
		Lng         float64 `form:"location[lng]"`
		Timezone    string  `form:"location[timezone]"`
		Display     string  `form:"location[display]"`
		Address     string  `form:"location[address]"` // varsa
	}

	decoder := form.NewDecoder()
	var formData LoginForm

	if err := decoder.Decode(&formData, request); err != nil {
		return nil, "", err
	}

	formData.Password = strings.ToLower(formData.Password)
	formData.UserName = strings.ToLower(formData.UserName)

	// Kullanıcıyı username ile bul (repo'da buna uygun fonksiyon olmalı)
	userObj, err := s.userRepo.GetByUserNameOrEmailOrNickname(formData.UserName)
	fmt.Println(err)
	if err != nil {
		return nil, "", errors.New("invalid username/email/nickname or password")
	}

	ok, err := helpers.ComparePasswordArgon2id(userObj.Password, formData.Password)
	if err != nil {
		return nil, "", err // Karşılaştırma sırasında hata
	}
	if !ok {
		return nil, "", errors.New("invalid credentials") // Şifre yanlış
	}

	locationPoint := &extensions.PostGISPoint{
		Lat: formData.Lat,
		Lng: formData.Lng,
	}

	locationUser := &utils.Location{
		ID:              uuid.New(),
		ContentableType: utils.LocationOwnerUser,
		ContentableID:   userObj.ID,
		CountryCode:     &formData.CountryCode,
		Country:         &formData.Country,
		City:            &formData.City,
		Region:          &formData.Region,
		Display:         &formData.Display,
		Timezone:        &formData.Timezone,
		Address:         &formData.Address,
		Latitude:        &formData.Lat,
		Longitude:       &formData.Lng,
		LocationPoint:   locationPoint,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.userRepo.UpsertLocation(locationUser); err != nil {
		return nil, "", err
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

// Kullanıcı ID ile getir
func (s *UserService) FetchUserProfileByNickname(nickname string) (*models.User, error) {
	return s.userRepo.GetByUserNameOrEmailOrNickname(nickname)
}

// Register işlemi
func (s *UserService) Test() {

	if err := s.userRepo.TestUser(); err != nil {
		return
	}

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

func (s *UserService) GetAllStories(ctx context.Context, limit int) ([]*models.Story, error) {
	return s.userRepo.GetAllStories(limit)
}

func (s *UserService) FetchNearbyUsers(ctx context.Context, user *models.User, distanceKm int, cursor *int64, limit int) ([]*models.User, error) {
	return s.userRepo.FetchNearbyUsers(user, distanceKm, cursor, limit)
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
	follower, err := s.userRepo.GetUserByPublicIdWithoutRelations(followerID)
	if err != nil {
		return false, errors.New(err.Error())
	}
	followee, err := s.userRepo.GetUserByPublicIdWithoutRelations(followeeID)
	if err != nil {
		return false, errors.New(err.Error())
	}

	engagementRepo := s.userRepo.GetEngagementRepository()

	// Takip edilenin takipçi sayısını toggle et (kind = follower)
	status, err := engagementRepo.ToggleEngagement(ctx, followee.ID, follower.ID, models.EngagementKindFollower, followee.ID, "user")
	if err != nil {
		return status, err
	}

	// Takip edenin takip ettiği kişi sayısını toggle et (kind = following)
	status, err = engagementRepo.ToggleEngagement(ctx, follower.ID, followee.ID, models.EngagementKindFollowing, follower.ID, "user")
	if err != nil {
		return status, err
	}

	return true, nil
}

func (s *UserService) UpdateUserProfile(authUser models.User, request map[string][]string) (*models.User, error) {
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

	// Kullanıcıyı username ile bul (repo'da buna uygun fonksiyon olmalı)
	existsUser, err := s.userRepo.GetByNameOrMailWithoutRelations(formData.UserName)
	if err == nil && existsUser.ID != authUser.ID {
		return nil, errors.New("username already taken")
	}

	userInfo, err := s.userRepo.GetUserByUUIDdWithoutRelations(authUser.ID)
	if err != nil {
		return nil, err
	}

	// Şifre doğrulaması (şifre formdan geliyorsa)
	if formData.CurrentPassword != "" {
		ok, err := helpers.ComparePasswordArgon2id(authUser.Password, formData.CurrentPassword)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("invalid password")
		}
	}

	if formData.DateOfBirth != "" {
		dateOfBirth, err := time.Parse("2006-01-02", formData.DateOfBirth)
		if err == nil {
			userInfo.DateOfBirth = &dateOfBirth
		} else {
			// İstersen hata dönebilirsin, ya da ignore et
		}
	}

	userInfo.UserName = formData.UserName
	userInfo.DisplayName = formData.DisplayName
	userInfo.Bio = utils.MakeLocalizedString("en", formData.Bio)
	//userObj.Website = formData.Website

	userInfo.PrivacyLevel = constants.PrivacyLevel(formData.PrivacyLevel)

	// Update et
	if err := s.userRepo.UpdateUser(userInfo); err != nil {
		return nil, err
	}

	if formData.LocationLatitude != "" && formData.LocationLongitude != "" {

		lat, err := strconv.ParseFloat(formData.LocationLatitude, 64)
		if err != nil {
			return nil, errors.New("invalid latitude")
		}
		lng, err := strconv.ParseFloat(formData.LocationLongitude, 64)
		if err != nil {
			return nil, errors.New("invalid longitude")
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
	likerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(likerId)
	if err != nil {
		return isLike, false, errors.New(err.Error())
	}
	likeeUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(likeeId)
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

	status, err := engagementRepo.ToggleEngagement(ctx, likerUser.ID, likeeUser.ID, engagementKindGiven, likerUser.ID, "user")
	if err != nil {
		return isLike, status, err
	}

	status, err = engagementRepo.ToggleEngagement(ctx, likeeUser.ID, likerUser.ID, engagementKindReceived, likeeUser.ID, "user")
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
	blockerUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(blockerId)
	if err != nil {
		return false, errors.New(err.Error())
	}
	blockedUser, err := s.userRepo.GetUserByPublicIdWithoutRelations(blockedId)
	if err != nil {
		return false, errors.New(err.Error())
	}

	engagementRepo := s.userRepo.GetEngagementRepository()

	var engagementKindGiven models.EngagementKind
	var engagementKindReceived models.EngagementKind

	engagementKindGiven = models.EngagementKindBlocking
	engagementKindReceived = models.EngagementKindBlockedBy

	status, err := engagementRepo.ToggleEngagement(ctx, blockerUser.ID, blockedUser.ID, engagementKindGiven, blockerUser.ID, "user")
	if err != nil {
		return status, err
	}

	status, err = engagementRepo.ToggleEngagement(ctx, blockedUser.ID, blockerUser.ID, engagementKindReceived, blockedUser.ID, "user")
	if err != nil {
		return status, err
	}

	return true, nil
}

func (s *UserService) FetchUserNotifications(ctx context.Context, auth_user *models.User, cursor *time.Time, limit int) ([]*notifications.Notification, error) {
	return s.userRepo.FetchUserNotifications(ctx, auth_user, cursor, limit)
}
