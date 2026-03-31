package repositories

import (
	"context"
	"core/constants"
	"core/extensions"
	"core/helpers"
	"core/models"
	"core/models/notifications"
	"core/models/utils"
	"core/types"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oschwald/maxminddb-golang"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db               *gorm.DB
	engagementRepo   *EngagementRepository
	notificationRepo *NotificationRepository
	snowFlakeNode    *helpers.Node
	geoipDB          *maxminddb.Reader
}

func (r *UserRepository) DB() *gorm.DB {
	return r.db
}

func (r *UserRepository) GEOIPDB() *maxminddb.Reader {
	return r.geoipDB
}

func (r *UserRepository) GetEngagementRepository() *EngagementRepository {
	return r.engagementRepo
}

func (r *UserRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewUserRepository(db *gorm.DB, geoipDB *maxminddb.Reader, snowFlakeNode *helpers.Node, engagementRepo *EngagementRepository, notificationRepo *NotificationRepository) *UserRepository {
	return &UserRepository{db: db, geoipDB: geoipDB, snowFlakeNode: snowFlakeNode, engagementRepo: engagementRepo, notificationRepo: notificationRepo}
}

func (r *UserRepository) TestUser() error {
	user := models.User{
		UserName:    "testUser",
		DisplayName: "testUser",
	}

	return r.db.Create(&user).Error
}

func (r *UserRepository) GetByUserNameOrEmailOrUsername(input string) (*models.User, error) {
	var userObj models.User
	err := r.db.
		Preload("Engagements").
		Preload("Engagements.EngagementDetails").
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Avatar.File").
		Preload("Cover.File").
		Where("LOWER(user_name) = LOWER(?) OR LOWER(email) = LOWER(?)", input, input).
		First(&userObj).Error
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func (r *UserRepository) GetUserByNameOrEmailOrNickname(input string) (*models.User, error) {
	var userObj models.User
	err := r.db.
		Where("user_name = ? OR email = ? OR display_name = ?", input, input, input).First(&userObj).Error
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func (r *UserRepository) ExistsByNameOrMail(input string) (bool, error) {
	var exists bool

	err := r.db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE LOWER(user_name) = LOWER(?)
		   OR LOWER(email) = LOWER(?)
		)
	`, input, input).Scan(&exists).Error

	return exists, err
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) UpdateUser(u *models.User) error {
	return r.db.Save(u).Error
}

func (r *UserRepository) DeleteUser(filters types.Filter) error {
	return r.db.
		Where("id = ?", filters.UserID).
		Delete(&models.User{}).Error
}

func (r *UserRepository) Login(username string, password string) error {
	return nil
}

func (r *UserRepository) LoginViaToken(token string) error {
	return nil
}

func (r *UserRepository) GetByID(userID uuid.UUID) (*models.User, error) {
	var u models.User

	err :=
		r.db.
			Preload("Avatar.File").
			Preload("Engagements").
			Preload("Engagements.EngagementDetails").
			Preload("Engagements.EngagementDetails.Engager").
			Preload("Engagements.EngagementDetails.Engagee").
			Preload("Engagements.EngagementDetails.Engager.Avatar.File").
			Preload("Engagements.EngagementDetails.Engagee.Cover.File").
			Preload("Cover.File").
			Preload("Location").
			First(&u, "id = ?", userID).Error

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserUUIDByPublicID(publicID int64) (uuid.UUID, error) {
	var userObj models.User
	err := r.db.Where("public_id = ?", publicID).First(&userObj).Error
	if err != nil {
		return uuid.Nil, err // nil yerine uuid.Nil döneriz
	}
	return userObj.ID, nil
}

func (r *UserRepository) GetUserByPublicId(userID int64) (*models.User, error) {
	var u models.User
	err :=
		r.db.
			Preload("Avatar").
			Preload("Avatar.File").
			Preload("Cover").
			Preload("Cover.File").
			Preload("Location").
			Preload("Engagements").
			Preload("Engagements.EngagementDetails.Engager").
			Preload("Engagements.EngagementDetails.Engagee").
			Preload("Engagements.EngagementDetails.Engager.Avatar.File").
			Preload("Engagements.EngagementDetails.Engagee.Cover.File").
			First(&u, "public_id = ?", userID).Error

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUsersStartingWith(letter string, limit int) ([]models.User, error) {
	var users []models.User
	pattern := strings.ToLower(letter) + "%"

	err := r.db.
		Preload("Avatar").
		Preload("Avatar.File").
		Preload("Cover").
		Preload("Cover.File").
		Limit(limit).
		Where("LOWER(user_name) LIKE ? OR LOWER(display_name) LIKE ?", pattern, pattern).
		Find(&users).Error

	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) GetUserByPublicIdWithoutRelations(filters types.Filter) (*models.User, error) {
	var u models.User
	err :=
		r.db.WithContext(filters.Context).First(&u, "public_id = ?", filters.UserID).Error

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserByUUIDdWithoutRelations(filters types.Filter) (*models.User, error) {
	var u models.User
	err :=
		r.db.WithContext(filters.Context).First(&u, "id = ?", filters.UserUUID).Error

	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByNameOrMailWithoutRelations(input string) (*models.User, error) {
	var userObj models.User
	err := r.db.
		Where("LOWER(user_name) = LOWER(?) OR LOWER(email) = LOWER(?)", input, input).
		First(&userObj).Error
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func (r *UserRepository) UpsertLocation(location *utils.Location) error {
	if location.ID == uuid.Nil {
		location.ID = uuid.New()
	}

	location.UpdatedAt = time.Now()
	if location.CreatedAt.IsZero() {
		location.CreatedAt = time.Now()
	}

	// Polymorphic owner_type + owner_id eşleşmesini kontrol et
	var existing utils.Location
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

func (r *UserRepository) AddStory(userID uuid.UUID, story *models.Story) error {
	story.UserID = userID
	return r.db.Create(story).Error
}

func (r *UserRepository) GetUserStories(userID uuid.UUID, limit int) ([]*models.Story, error) {
	var stories []*models.Story
	if err := r.db.Preload("Media").
		Where("user_id = ? AND is_expired = false", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *UserRepository) GetAllStories(filters types.Filter) ([]*models.Story, error) {
	var stories []*models.Story
	if err := r.db.WithContext(filters.Context).
		Preload("Media.File").
		Preload("User").
		Preload("User.Avatar.File").
		Preload("User.Cover.File").
		Where("is_expired = false").
		Order("created_at DESC").
		Limit(filters.Limit).
		Find(&stories).Error; err != nil {
		return nil, err
	}
	return stories, nil
}

func (r *UserRepository) ExpireOldStories() error {
	return r.db.Model(&models.Story{}).
		Where("expires_at <= ? AND is_expired = false", gorm.Expr("NOW()")).
		Update("is_expired", true).Error
}

func (r *UserRepository) UpsertUserPreferenceEx(ctx context.Context, user models.User, preferenceItemId string, bitIndexStr string, enabled bool) error {

	bitIndex, err := strconv.ParseInt(bitIndexStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid bitIndex: %w", err)
	}

	if enabled {
		err := user.SetPreference(int(bitIndex))
		if err != nil {
			return err
		}
	} else {
		err := user.UnsetPreference(int(bitIndex))
		if err != nil {
			return err
		}
	}

	updateError := r.db.Model(&user).Update("preferences_flags", user.PreferencesFlags).Error

	fmt.Println("USER_ID", user.ID, user.UserName, user.PreferencesFlags)
	return updateError

}

func (r *UserRepository) UpsertUserPreference(ctx context.Context, user models.User, preferenceItemId string, bitIndexStr string, enabled bool) error {
	bitIndex, err := strconv.ParseInt(bitIndexStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid bitIndex: %w", err)
	}

	var pref models.PreferencesData
	if err := r.db.Model(&models.Preferences{}).Select("data").First(&pref).Error; err != nil {
		return err
	}

	allCategories := append(append(pref.Attributes, pref.Interests...), pref.Fantasies...)

	var foundCategory *models.PreferenceCategory
	var foundItem *models.PreferenceItem
	for i, cat := range allCategories {
		for j, item := range cat.Items {
			if item.ID.String() == preferenceItemId {
				foundCategory = &allCategories[i]
				foundItem = &allCategories[i].Items[j]
				break
			}
		}
		if foundItem != nil {
			break
		}
	}

	if foundItem == nil {
		return fmt.Errorf("preference item not found")
	}

	var flags big.Int
	if user.PreferencesFlags != "" {
		bytes, err := hex.DecodeString(user.PreferencesFlags)
		if err != nil {
			return err
		}
		flags.SetBytes(bytes)
	}

	if !foundCategory.AllowMultiple {
		for _, item := range foundCategory.Items {
			flags.SetBit(&flags, int(item.BitIndex), 0)
		}
	}

	if enabled {
		flags.SetBit(&flags, int(bitIndex), 1)
	} else {
		flags.SetBit(&flags, int(bitIndex), 0)
	}

	user.PreferencesFlags = hex.EncodeToString(flags.Bytes())
	updateError := r.db.Model(&user).Update("preferences_flags", user.PreferencesFlags).Error
	if updateError != nil {
		return updateError
	}

	return nil
}

func (r *UserRepository) FetchNearbyUsers(filters types.Filter) ([]*models.User, *float64, error) {
	var authUser *models.User

	if filters.AuthUser != nil {
		authUser = &models.User{}
		if err := r.db.WithContext(filters.Context).Preload("Location").First(authUser, "id = ?", filters.AuthUser.ID).Error; err != nil {
			authUser = nil
		}
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}

	var lat, lng float64
	useLocation := false

	if filters.Latitude != nil && filters.Longitude != nil {
		lat = *filters.Latitude
		lng = *filters.Longitude
		useLocation = true
	} else if authUser != nil &&
		authUser.Location != nil &&
		authUser.Location.Latitude != nil &&
		authUser.Location.Longitude != nil {
		lat = *authUser.Location.Latitude
		lng = *authUser.Location.Longitude
		useLocation = true
	}

	var users []*models.User

	baseQuery := r.db.WithContext(filters.Context).
		Model(&models.User{}).
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Limit(limit)

	if filters.Domain != nil {
		baseQuery = baseQuery.Where("domain = ?", *filters.Domain)
	}

	if useLocation {
		const noLocationDistance = 9999999999.0

		distanceSQL := `
			COALESCE(
				ST_Distance(
					l.location_point,
					ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
				),
				?
			)
		`

		query := baseQuery.Joins(`
			LEFT JOIN locations l
				ON l.contentable_id = users.id
				AND l.contentable_type = ?
				AND l.deleted_at IS NULL
		`, utils.LocationOwnerUser)

		if filters.Cursor != nil {
			if filters.Distance == nil {
				return nil, nil, fmt.Errorf("distance cursor is required for location-based pagination")
			}

			query = query.Where(
				fmt.Sprintf("(%s > ?) OR (%s = ? AND users.public_id > ?)", distanceSQL, distanceSQL),
				lng, lat, noLocationDistance, *filters.Distance,
				lng, lat, noLocationDistance, *filters.Distance, *filters.Cursor,
			)
		}

		if err := query.
			Order(clause.Expr{
				SQL:  distanceSQL + " ASC",
				Vars: []interface{}{lng, lat, noLocationDistance},
			}).
			Order("users.public_id ASC").
			Find(&users).Error; err != nil {
			return nil, nil, err
		}

		if len(users) == 0 {
			return users, nil, nil
		}

		lastDistance := noLocationDistance
		lastUser := users[len(users)-1]
		if lastUser.Location != nil && lastUser.Location.Latitude != nil && lastUser.Location.Longitude != nil {
			lastDistance = lastUser.Location.DistanceTo(lat, lng)
		}

		return users, &lastDistance, nil
	}

	q := baseQuery.Order("public_id ASC")

	if filters.Cursor != nil {
		q = q.Where("public_id > ?", *filters.Cursor)
	}

	if err := q.Find(&users).Error; err != nil {
		return nil, nil, err
	}

	return users, nil, nil
}

func (r *UserRepository) FetchLiveUsers(filters types.Filter) ([]*models.User, error) {
	var users []*models.User
	var authUser *models.User

	if filters.AuthUser != nil {
		if err := r.db.Preload("Location").First(&authUser, "id = ?", filters.AuthUser.ID).Error; err != nil {
			authUser = nil
		}
	}

	if authUser != nil &&
		authUser.Location != nil &&
		authUser.Location.Latitude != nil &&
		authUser.Location.Longitude != nil {

		lat := *authUser.Location.Latitude
		lng := *authUser.Location.Longitude

		raw := r.db.
			Table("users u").
			Select(`
				u.*,
				MIN(
					COALESCE(
						ST_Distance(
							l.location_point,
							ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
						),
						9999999999
					)
				) AS distance
			`, lng, lat).
			Joins(`
				LEFT JOIN locations l 
				ON l.contentable_id = u.id 
				AND l.contentable_type = 'user'
			`).
			Group("u.id")

		if filters.IsLive != nil {
			raw = raw.Where("u.is_live = ?", *filters.IsLive)
		}

		query := r.db.Table("(?) as sub", raw)

		if filters.Cursor != nil && filters.Distance != nil {
			query = query.Where(`
				(sub.distance > ? OR (sub.distance = ? AND sub.public_id > ?))
			`, *filters.Distance, *filters.Distance, *filters.Cursor)
		}

		query = query.
			Order("sub.distance ASC, sub.public_id ASC").
			Limit(filters.Limit)

		if err := query.
			Preload("Location").
			Preload("Avatar.File").
			Preload("Cover.File").
			Find(&users).Error; err != nil {
			return nil, err
		}

		return users, nil
	}

	q := r.db.Model(&models.User{})

	if filters.IsLive != nil {
		q = q.Where("is_live = ?", *filters.IsLive)
	}

	q = q.Order("public_id ASC").Limit(filters.Limit)

	if filters.Cursor != nil {
		q = q.Where("public_id > ?", *filters.Cursor)
	}

	if err := q.
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) VerifyCaptcha(secret string, response string) (bool, error) {
	type recaptchaResponse struct {
		Success bool `json:"success"`
	}

	if response == constants.APPLICATION_NAME {
		return true, nil
	}

	resp, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify",
		url.Values{"secret": {secret}, "response": {response}})
	if err != nil {
		return false, err
	}
	defer func() {
		if cerr := resp.Body.Close(); err == nil {
			err = cerr
		}
	}()

	body, err := io.ReadAll(resp.Body)
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
	result := r.db.Model(&models.User{}).Where("public_id = ?", userID).Updates(updateData)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *UserRepository) FetchUserNotifications(ctx context.Context, auth_user *models.User, cursor *time.Time, limit int) (items []*notifications.Notification, nextCursor *time.Time, err error) {

	db := r.db.WithContext(ctx).
		Where("user_id = ?", auth_user.ID).
		Order("created_at DESC").
		Limit(limit + 1). // +1 => daha fazla var mı görmek için
		Preload("Sender.Avatar.File")

	if cursor != nil {
		db = db.Where("created_at < ?", *cursor)
	}

	if err := db.Find(&items).Error; err != nil {
		return nil, nil, err
	}

	// Eğer fazla varsa next cursor üret
	if len(items) > limit {
		last := items[limit]
		items = items[:limit]        // fazlayı çıkar
		nextCursor = &last.CreatedAt // bir sonraki cursor bu
	} else {
		nextCursor = nil // daha fazla yok
	}

	return items, nextCursor, nil
}

func (r *UserRepository) CheckIn(ctx context.Context) error {
	return nil
}

func (r *UserRepository) GetLocationFromIP(ipStr string) (*utils.Location, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP")
	}

	var record utils.GeoIPCity
	err := r.GEOIPDB().Lookup(ip, &record)
	if err != nil {
		return nil, err
	}

	countryName := record.Country.Names["en"]
	cityName := record.City.Names["en"]
	isoCode := record.Country.IsoCode
	lat := record.Location.Latitude
	lon := record.Location.Longitude
	tz := record.Location.Timezone

	display := cityName
	if display == "" {
		display = countryName
	}

	loc := &utils.Location{
		CountryCode: &isoCode,
		Country:     &countryName,
		City:        &cityName,
		Latitude:    &lat,
		Longitude:   &lon,
		Timezone:    &tz,
		Display:     &display,
		IPAddress:   &ipStr,
	}

	if loc.Latitude != nil && loc.Longitude != nil {
		loc.LocationPoint = &extensions.PostGISPoint{
			Lat: *loc.Latitude,
			Lng: *loc.Longitude,
		}
	}

	return loc, nil
}

func (r *UserRepository) UpdateLocation(context context.Context, authUser *models.User, ipStr string) error {

	loc, err := r.GetLocationFromIP(ipStr)
	if err != nil {
		return err
	}

	if authUser != nil {
		loc.ContentableType = utils.LocationOwnerUser
		loc.ContentableID = authUser.ID
		if authUser.Location == nil {
			return r.UpsertLocation(loc)
		}
	}

	return nil
}

func (r *UserRepository) AddReferral(ctx context.Context, referrerID uuid.UUID, referredUserID uuid.UUID, rewardAmount decimal.Decimal) (*decimal.Decimal, error) {
	exists, err := r.GetEngagementRepository().HasUserEngaged(ctx, referredUserID, referrerID, models.EngagementKindReferral)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("user already referred")
	}

	referralErr := r.GetEngagementRepository().AddTip(ctx, referredUserID, referrerID, rewardAmount, referrerID, models.EngagementContentableTypeUser, models.EngagementKindReferral)
	if referralErr != nil {
		return nil, referralErr
	}

	messageTitle := " New Referral Reward"
	messageText := "You received a new referral reward. Click to read."

	payload := notifications.NotificationPayload{
		Title: messageTitle,
		Body:  messageText,
	}

	canSendNotification := true
	senderUser, err := r.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: referredUserID})
	if err != nil {
		canSendNotification = false
	}
	receiverUser, err := r.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: referrerID})
	if err != nil {
		canSendNotification = false
	}
	if canSendNotification {
		err = r.notificationRepo.SendNotificationToUser(*senderUser, *receiverUser, notifications.NotificationTypeReferral, messageTitle, messageText, payload)
		if err != nil {
			fmt.Printf("Bildirim gönderilemedi user %s: %v\n", referrerID, err)
		}
	}

	return r.AddBalance(ctx, referrerID, rewardAmount)

}

func (r *UserRepository) AddBalance(ctx context.Context, userID uuid.UUID, amount decimal.Decimal) (*decimal.Decimal, error) {

	if amount.IsZero() {
		return nil, errors.New("amount cannot be zero")
	}

	if amount.IsNegative() {
		return nil, errors.New("amount cannot be negative")
	}

	var newBalance decimal.Decimal

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}

		user.Balance = user.Balance.Add(amount)
		newBalance = user.Balance

		if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("balance", newBalance).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &newBalance, nil
}
