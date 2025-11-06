package repositories

import (
	"bifrost/constants"
	"bifrost/helpers"
	global_shared "bifrost/models/shared"
	userModel "bifrost/models/user"
	"bifrost/models/user/payloads"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
}

func (r *UserRepository) DB() *gorm.DB {
	return r.db
}

func (r *UserRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewUserRepository(db *gorm.DB, snowFlakeNode *helpers.Node) *UserRepository {
	return &UserRepository{db: db, snowFlakeNode: snowFlakeNode}
}

func (r *UserRepository) TestUser() error {
	user := userModel.User{
		UserName:    "testUser",
		DisplayName: "testUser",
	}

	return r.db.Create(&user).Error
}

func (r *UserRepository) GetByUserNameOrEmailOrNickname(input string) (*userModel.User, error) {
	var userObj userModel.User
	err := r.db.
		Preload("Fantasies.Fantasy").
		Preload("Interests.InterestItem.Interest").
		Preload("Avatar.File").
		Preload("Cover.File").
		Preload("GenderIdentities").
		Preload("SexualOrientations").
		Preload("SexualRole").
		Preload("UserAttributes.Attribute").
		Preload("Media").
		Preload("Followees.Followee"). // Kullanıcının takip ettikleri
		Preload("Followers.Follower"). // Kullanıcıyı takip edenler
		Preload("SocialRelations.Likes").
		Preload("SocialRelations.LikedBy").
		Preload("SocialRelations.Matches").
		Preload("SocialRelations.Favorites").
		Preload("SocialRelations.FavoritedBy").
		Preload("SocialRelations.BlockedUsers").
		Preload("SocialRelations.BlockedByUsers").
		Where("user_name = ? OR email = ?", input, input).First(&userObj).Error
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func (r *UserRepository) GetUserByNameOrEmailOrNickname(input string) (*userModel.User, error) {
	var userObj userModel.User
	err := r.db.
		Where("user_name = ? OR email = ?", input, input).First(&userObj).Error
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func (r *UserRepository) Create(user *userModel.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) UpdateUser(u *userModel.User) error {
	return r.db.Save(u).Error
}

func (r *UserRepository) DeleteUser(userID uuid.UUID) error {
	return r.db.
		Where("id = ?", userID).
		Delete(&userModel.User{}).Error
}

func (r *UserRepository) Login(username string, password string) error {
	return nil
}

func (r *UserRepository) LoginViaToken(token string) error {
	return nil
}

// Kullanıcıyı takip et
func (r *UserRepository) Follow(followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return errors.New(constants.ErrInvalidAction.String()) // Kendini takip edemezsin
	}

	// Zaten takip ediyor mu kontrol et
	var existing userModel.Follow
	if err := r.db.
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		First(&existing).Error; err == nil {
		return errors.New(constants.ErrDuplicateResource.String()) // Zaten takip ediyor
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New(constants.ErrDatabaseError.String()) // DB hatası
	}

	follow := userModel.Follow{
		FollowerID: followerID,
		FolloweeID: followeeID,
		Status:     "following",
	}

	if err := r.db.Create(&follow).Error; err != nil {
		return errors.New(constants.ErrDatabaseError.String())
	}

	return nil
}

// Takipten çık
func (r *UserRepository) Unfollow(followerID, followeeID uuid.UUID) error {
	if err := r.db.
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&userModel.Follow{}).Error; err != nil {
		return errors.New(constants.ErrDatabaseError.String())
	}
	return nil
}

// Kullanıcının başka bir kullanıcıyı takip edip etmediğini kontrol eder
func (r *UserRepository) IsFollowing(followerID, followeeID uuid.UUID) (bool, error) {
	if followerID == followeeID {
		// Kendini takip etme durumu her zaman false olabilir, ya da hata dönebilirsin
		return false, nil
	}

	var follow userModel.Follow
	err := r.db.
		Where("follower_id = ? AND followee_id = ? AND status = ?", followerID, followeeID, "following").
		First(&follow).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Takip yok
			return false, nil
		}
		// Başka bir DB hatası var
		return false, errors.New(constants.ErrDatabaseError.String())
	}

	// Kayıt bulundu, takip ediliyor
	return true, nil
}

// ID ile kullanıcıyı al
func (r *UserRepository) GetByID(userID uuid.UUID) (*userModel.User, error) {
	var u userModel.User

	err :=
		r.db.
			Preload("Fantasies.Fantasy").
			Preload("Interests.InterestItem.Interest").
			Preload("Avatar.File").
			Preload("Cover.File").
			Preload("GenderIdentities").
			Preload("SexualOrientations").
			Preload("SexualRole").
			Preload("UserAttributes.Attribute").
			Preload("Media").
			Preload("Followees.Followee"). // Kullanıcının takip ettikleri
			Preload("Followers.Follower"). // Kullanıcıyı takip edenler
			Preload("SocialRelations.Likes").
			Preload("SocialRelations.LikedBy").
			Preload("SocialRelations.Matches").
			Preload("SocialRelations.Favorites").
			Preload("SocialRelations.FavoritedBy").
			Preload("SocialRelations.BlockedUsers").
			Preload("SocialRelations.BlockedByUsers").
			First(&u, "id = ?", userID).Error

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserUUIDByPublicID(publicID int64) (uuid.UUID, error) {
	var userObj userModel.User
	err := r.db.Where("public_id = ?", publicID).First(&userObj).Error
	if err != nil {
		return uuid.Nil, err // nil yerine uuid.Nil döneriz
	}
	return userObj.ID, nil
}

func (r *UserRepository) GetUserByPublicId(userID int64) (*userModel.User, error) {
	var u userModel.User
	err :=
		r.db.
			Preload("Fantasies.Fantasy").
			Preload("Avatar").
			Preload("Cover").
			Preload("Media").
			Preload("Followees.Followee"). // Kullanıcının takip ettikleri
			Preload("Followers.Follower"). // Kullanıcıyı takip edenler
			Preload("SocialRelations.Likes").
			Preload("SocialRelations.LikedBy").
			Preload("SocialRelations.Matches").
			Preload("SocialRelations.Favorites").
			Preload("SocialRelations.FavoritedBy").
			Preload("SocialRelations.BlockedUsers").
			Preload("SocialRelations.BlockedByUsers").
			First(&u, "public_id = ?", userID).Error

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUsersStartingWith(letter string, limit int) ([]userModel.User, error) {
	var users []userModel.User
	pattern := strings.ToLower(letter) + "%"

	err := r.db.
		Preload("Avatar").
		Preload("Avatar.File").
		Limit(limit).
		Where("LOWER(user_name) LIKE ? OR LOWER(display_name) LIKE ?", pattern, pattern).
		Find(&users).Error

	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) GetUserByPublicIdWithoutRelations(userID int64) (*userModel.User, error) {
	var u userModel.User
	err :=
		r.db.First(&u, "public_id = ?", userID).Error

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserByUUIDdWithoutRelations(userID uuid.UUID) (*userModel.User, error) {
	var u userModel.User
	err :=
		r.db.First(&u, "id = ?", userID).Error

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) UpsertLocation(location *global_shared.Location) error {
	if location.ID == uuid.Nil {
		location.ID = uuid.New()
	}

	location.UpdatedAt = time.Now()
	if location.CreatedAt.IsZero() {
		location.CreatedAt = time.Now()
	}

	// Polymorphic owner_type + owner_id eşleşmesini kontrol et
	var existing global_shared.Location
	err := r.db.Where("contentable_type = ? AND contentable_id = ?", location.ContentableType, location.ContentableID).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Yeni ekle
			return r.db.Create(location).Error
		}
		return err
	}

	// Güncelle
	location.ID = existing.ID
	return r.db.Model(&existing).Updates(location).Error
}

func (r *UserRepository) AddStory(userID uuid.UUID, story *userModel.Story) error {
	story.UserID = userID
	return r.db.Create(story).Error
}

func (r *UserRepository) GetUserStories(userID uuid.UUID, limit int) ([]*userModel.Story, error) {
	var stories []*userModel.Story
	if err := r.db.Preload("Media").
		Where("user_id = ? AND is_expired = false", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *UserRepository) GetAllStories(limit int) ([]*userModel.Story, error) {
	var stories []*userModel.Story
	if err := r.db.
		Preload("Media.File").
		Preload("User").
		Preload("User.Avatar.File").
		Preload("User.Cover.File").
		Where("is_expired = false").
		Order("created_at DESC").
		Limit(limit).
		Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *UserRepository) ExpireOldStories() error {
	return r.db.Model(&userModel.Story{}).
		Where("expires_at <= ? AND is_expired = false", gorm.Expr("NOW()")).
		Update("is_expired", true).Error
}

func (r *UserRepository) GetAttribute(attributeID uuid.UUID) (*payloads.Attribute, error) {
	var attr payloads.Attribute
	if err := r.db.Where("id = ?", attributeID).First(&attr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attr, nil
}

func (r *UserRepository) GetInterestItem(interestId uuid.UUID) (*payloads.InterestItem, error) {
	var interest payloads.InterestItem
	if err := r.db.Where("id = ?", interestId).First(&interest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &interest, nil
}

func (r *UserRepository) GetFantasy(fantasyId uuid.UUID) (*payloads.Fantasy, error) {
	var fantasy payloads.Fantasy
	if err := r.db.Where("id = ?", fantasyId).First(&fantasy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &fantasy, nil
}

func (r *UserRepository) UpsertUserAttribute(attr *payloads.UserAttribute) error {

	fmt.Println("USER", attr.AttributeID, attr.UserID)
	attr.ID = uuid.New()
	if attr.AttributeID == uuid.Nil {
		return fmt.Errorf("invalid attribute")

	}

	if attr.UserID == uuid.Nil {
		return fmt.Errorf("invalid user")
	}

	var existing payloads.UserAttribute
	err := r.db.Where("user_id = ? AND category_type = ?", attr.UserID, attr.CategoryType).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Kayıt yoksa ekle
			if attr.ID == uuid.Nil {
				attr.ID = uuid.New()
			}
			return r.db.Create(attr).Error
		}
		return err
	}
	existing.AttributeID = attr.AttributeID
	existing.Notes = attr.Notes
	return r.db.Save(&existing).Error
}

func (r *UserRepository) ToggleUserInterest(interest *payloads.UserInterest) error {
	if interest.InterestItemID == uuid.Nil {
		return fmt.Errorf("invalid interest_item_id")
	}

	if interest.UserID == uuid.Nil {
		return fmt.Errorf("invalid user_id")
	}

	var existing payloads.UserInterest
	err := r.db.Where("user_id = ? AND interest_item_id = ?", interest.UserID, interest.InterestItemID).First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Kayıt yok → ekle
		if interest.ID == uuid.Nil {
			interest.ID = uuid.New()
		}
		return r.db.Create(interest).Error
	} else if err != nil {
		return err
	}

	// Kayıt varsa → sil
	return r.db.Delete(&existing).Error
}

func (r *UserRepository) ToggleUserFantasy(fantasy *payloads.UserFantasy) error {
	if fantasy.FantasyID == uuid.Nil {
		return fmt.Errorf("invalid fantasy_id")
	}

	if fantasy.UserID == uuid.Nil {
		return fmt.Errorf("invalid user_id")
	}

	var existing payloads.UserFantasy
	err := r.db.Where("user_id = ? AND fantasy_id = ?", fantasy.UserID, fantasy.FantasyID).First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Kayıt yok → ekle
		if fantasy.ID == uuid.Nil {
			fantasy.ID = uuid.New()
		}
		return r.db.Create(fantasy).Error
	} else if err != nil {
		return err
	}

	// Kayıt varsa → sil
	return r.db.Delete(&existing).Error
}

func (r *UserRepository) GetUserWithSexualRelations(userID uuid.UUID) (*userModel.User, error) {
	var user userModel.User
	err := r.db.Preload("GenderIdentities").
		Preload("SexualOrientations").
		Preload("SexualRole").
		First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) ClearGenderIdentities(user *userModel.User) error {
	return r.db.Model(user).Association("GenderIdentities").Clear()
}

func (r *UserRepository) ReplaceGenderIdentities(user *userModel.User, ids []uuid.UUID) error {
	var genders []payloads.GenderIdentity
	for _, id := range ids {
		genders = append(genders, payloads.GenderIdentity{ID: id})
	}
	return r.db.Model(user).Association("GenderIdentities").Replace(genders)
}

func (r *UserRepository) ClearSexualOrientations(user *userModel.User) error {
	return r.db.Model(user).Association("SexualOrientations").Clear()
}

func (r *UserRepository) ReplaceSexualOrientations(user *userModel.User, ids []uuid.UUID) error {
	var sexuals []payloads.SexualOrientation
	for _, id := range ids {
		sexuals = append(sexuals, payloads.SexualOrientation{ID: id})
	}
	return r.db.Model(user).Association("SexualOrientations").Replace(sexuals)
}

func (r *UserRepository) ClearSexRole(user *userModel.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) SetSexRole(user *userModel.User, sexRoleID uuid.UUID) error {
	var dbUser userModel.User
	if err := r.db.First(&dbUser, "id = ?", user.ID).Error; err != nil {
		return err
	}
	dbUser.SexualRoleID = &sexRoleID
	if err := r.db.Save(&dbUser).Error; err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) FetchNearbyUsers(auth_user *userModel.User, distance int, cursor *int64, limit int) ([]*userModel.User, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 100 {
		limit = 100
	}

	if auth_user != nil {
		fmt.Println("GELEN CURSOR ", *cursor)
	} else {
		fmt.Println("CURSOR NILL")
	}

	var users []*userModel.User
	meters := float64(distance * 100000)

	var user *userModel.User
	if auth_user != nil {
		r.db.Preload("Location").First(&user, "id = ?", auth_user.ID)
	}

	// Eğer kullanıcı konumu yoksa -> tüm kullanıcıları çek (cursor + limit uygula)
	if user == nil || user.Location == nil || user.Location.Latitude == nil || user.Location.Longitude == nil {
		q := r.db.Model(&userModel.User{}).
			Order("public_id ASC").
			Limit(limit)

		if cursor != nil {
			fmt.Println("CURSOR EKLENDI")
			q = q.Where("public_id > ?", *cursor)
		}

		// Preload ilişkiler ihtiyaca göre arttır
		if err := q.Preload("Location").
			Preload("Avatar").
			Preload("Avatar.File").
			Preload("Cover").
			Preload("Cover.File").
			Preload("Fantasies.Fantasy").
			Preload("Interests.InterestItem.Interest").
			Preload("Avatar.File").
			Preload("Cover.File").
			Preload("GenderIdentities").
			Preload("SexualOrientations").
			Preload("SexualRole").
			Preload("UserAttributes.Attribute").
			Find(&users).Error; err != nil {
			return nil, err
		}

		return users, nil
	}

	raw := r.db.
		Table("users u").
		Joins("JOIN locations l ON l.contentable_id = u.id AND l.contentable_type = 'user'").
		Select(`
			u.*,
			ST_Distance(
				l.location_point,
				ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
			) AS distance
		`, *user.Location.Longitude, *user.Location.Latitude).
		Where("l.location_point IS NOT NULL").
		Where(`
			ST_DWithin(
				l.location_point,
				ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography,
				?
			)
		`, *user.Location.Longitude, *user.Location.Latitude, meters).
		Order("distance ASC, u.display_name ASC").
		Limit(limit)

	if cursor != nil {
		raw = raw.Where("u.public_id > ?", *cursor)
	}

	if err := raw.Preload("Location").
		Preload("Fantasies.Fantasy").
		Preload("Interests.InterestItem.Interest").
		Preload("Avatar").
		Preload("Avatar.File").
		Preload("Cover").
		Preload("Cover.File").
		Preload("GenderIdentities").
		Preload("SexualOrientations").
		Preload("SexualRole").
		Preload("UserAttributes.Attribute").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) VerifyCaptcha(secret string, response string) (bool, error) {
	type recaptchaResponse struct {
		Success bool `json:"success"`
	}
	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify",
		url.Values{"secret": {secret}, "response": {response}})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var captchaResponse recaptchaResponse
	err = json.Unmarshal(body, &captchaResponse)
	if err != nil {
		return false, err
	}

	return captchaResponse.Success, nil
}

func (r *UserRepository) UpdateUserSocket(userID int64, socketID string) error {
	now := time.Now()

	updateData := map[string]interface{}{
		"last_online": now,
		"socket_id":   socketID,
	}
	result := r.db.Model(&userModel.User{}).Where("public_id = ?", userID).Updates(updateData)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
