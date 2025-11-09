package services

import (
	"context"
	"coolvibes/constants"
	"coolvibes/extensions"
	"coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/models/media"
	user_payloads "coolvibes/models/user_payloads"
	"coolvibes/models/utils"
	"coolvibes/repositories"
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
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

	fmt.Println("INSERT:FANTASIES")

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

func (s *UserService) GetAttribute(ctx context.Context, attributeID uuid.UUID) (*user_payloads.Attribute, error) {
	return s.userRepo.GetAttribute(attributeID)
}

func (s *UserService) GetInterestItem(ctx context.Context, interestId uuid.UUID) (*user_payloads.InterestItem, error) {
	return s.userRepo.GetInterestItem(interestId)
}

// Kullanıcı ID ile getir
func (s *UserService) GetFantasy(ctx context.Context, id uuid.UUID) (*user_payloads.Fantasy, error) {
	return s.userRepo.GetFantasy(id)
}

func (s *UserService) UpsertUserSexualIdentify(
	ctx context.Context,
	userID uuid.UUID,
	genderIDs []string,
	sexualIDs []string,
	sexRoleIDs []string,
) error {

	// Kullanıcıyı repo'dan çekiyoruz (ilişkilerle birlikte)
	user, err := s.userRepo.GetUserWithSexualRelations(userID)
	if err != nil {
		return err
	}

	// GenderIdentities güncelle
	if genderIDs != nil {
		if len(genderIDs) == 0 {
			if err := s.userRepo.ClearGenderIdentities(user); err != nil {
				return err
			}
		} else {
			ids, err := parseUUIDs(genderIDs)
			if err != nil {
				return err
			}
			if err := s.userRepo.ReplaceGenderIdentities(user, ids); err != nil {
				return err
			}
		}
	}

	// SexualOrientations güncelle
	if sexualIDs != nil {
		if len(sexualIDs) == 0 {
			if err := s.userRepo.ClearSexualOrientations(user); err != nil {
				return err
			}
		} else {
			ids, err := parseUUIDs(sexualIDs)
			if err != nil {
				return err
			}
			if err := s.userRepo.ReplaceSexualOrientations(user, ids); err != nil {
				return err
			}
		}
	}

	// SexRole güncelle (tek ilişki)
	if sexRoleIDs != nil {
		if len(sexRoleIDs) == 0 {
			if err := s.userRepo.ClearSexRole(user); err != nil {
				return err
			}
		} else {
			id, err := uuid.Parse(sexRoleIDs[0])
			if err != nil {
				return err
			}

			fmt.Println("SET ROLE SEX GHERE", user.DisplayName, id)
			if err := s.userRepo.SetSexRole(user, id); err != nil {
				fmt.Println("SET ROLE HATA OLDU GHERE", user.DisplayName)

				return err
			}
		}
	}

	return nil
}

func parseUUIDs(strIDs []string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for _, strID := range strIDs {
		id, err := uuid.Parse(strID)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *UserService) UpsertUserAttribute(ctx context.Context, attr *user_payloads.UserAttribute) error {
	if attr == nil {
		return fmt.Errorf("attribute cannot be nil")
	}

	if attr.UserID == uuid.Nil {
		return fmt.Errorf("user_id is required")
	}

	if attr.AttributeID == uuid.Nil {
		return fmt.Errorf("attribute_id is required")
	}

	// Repository'yi çağır
	err := s.userRepo.UpsertUserAttribute(attr)
	if err != nil {
		return fmt.Errorf("failed to upsert user attribute: %w", err)
	}

	return nil
}

func (s *UserService) UpsertUserInterest(ctx context.Context, interest *user_payloads.UserInterest) error {
	if interest == nil {
		return fmt.Errorf("attribute cannot be nil")
	}

	if interest.UserID == uuid.Nil {
		return fmt.Errorf("user_id is required")
	}

	if interest.InterestItemID == uuid.Nil {
		return fmt.Errorf("attribute_id is required")
	}

	// Repository'yi çağır
	err := s.userRepo.ToggleUserInterest(interest)
	if err != nil {
		return fmt.Errorf("failed to upsert user attribute: %w", err)
	}

	return nil
}

func (s *UserService) UpsertUserFantasy(ctx context.Context, fantasy *user_payloads.UserFantasy) error {
	if fantasy == nil {
		return fmt.Errorf("fantasy cannot be nil")
	}

	if fantasy.UserID == uuid.Nil {
		return fmt.Errorf("user_id is required")
	}

	if fantasy.FantasyID == uuid.Nil {
		return fmt.Errorf("fantasy is required")
	}

	// Repository'yi çağır
	err := s.userRepo.ToggleUserFantasy(fantasy)
	if err != nil {
		return fmt.Errorf("failed to upsert user attribute: %w", err)
	}

	return nil
}

func (s *UserService) GetAllStories(ctx context.Context, limit int) ([]*models.Story, error) {
	return s.userRepo.GetAllStories(limit)
}

func (s *UserService) FetchNearbyUsers(ctx context.Context, user *models.User, distanceKm int, cursor *int64, limit int) ([]*models.User, error) {
	return s.userRepo.FetchNearbyUsers(user, distanceKm, cursor, limit)
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

func (s *UserService) GetUsersStartingWith(letter string, limit int) ([]models.User, error) {
	return s.userRepo.GetUsersStartingWith(letter, limit)
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

	// Latitude ve Longitude string olarak geldiği için dönüştür
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

	// Tarih formatını parse et

	userInfo.PrivacyLevel = constants.PrivacyLevel(formData.PrivacyLevel)

	// Update et
	if err := s.userRepo.UpdateUser(userInfo); err != nil {
		return nil, err
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

	return s.GetUserByID(authUser.ID)
}
