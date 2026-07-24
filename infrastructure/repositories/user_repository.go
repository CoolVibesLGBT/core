package repositories

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	"core/extensions"
	"core/helpers"
	"core/models"
	"core/models/notifications"
	"core/models/utils"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"github.com/oschwald/maxminddb-golang"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db               *gorm.DB
	engagementRepo   *EngagementRepository
	notificationRepo referralNotificationSender
	snowFlakeNode    *helpers.Node
	geoipDB          *maxminddb.Reader
}

var (
	_ ports.UserRepository          = (*UserRepository)(nil)
	_ ports.SessionRepository       = (*UserRepository)(nil)
	_ ports.PublicUserProfileReader = (*UserRepository)(nil)
	_ ports.PublicUserSearchReader  = (*UserRepository)(nil)
)

type referralNotificationSender interface {
	SendNotificationToUser(sender models.User, receiver models.User, notificationType string, notificationTitle string, notificationMessage string, payload notifications.NotificationPayload) error
}

var fallbackReferralTransactionLock sync.Mutex

var errReferralUserNotFound = errors.New("referral user not found")

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
	var notifier referralNotificationSender
	if notificationRepo != nil {
		notifier = notificationRepo
	}
	return &UserRepository{db: db, geoipDB: geoipDB, snowFlakeNode: snowFlakeNode, engagementRepo: engagementRepo, notificationRepo: notifier}
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
	err := r.publicProfileQuery(input).
		First(&userObj).Error
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

// ResetBotBroadcastPresence is the persistence adapter for the broadcast
// worker's presence reset port.
func (r *UserRepository) ResetBotBroadcastPresence(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("is_bot = ?", true).
		Updates(map[string]interface{}{"is_live": false, "is_online": true}).Error
}

// FindBroadcastUser hides PostgreSQL/GORM query details behind the worker's
// narrow persistence boundary.
func (r *UserRepository) FindBroadcastUser(ctx context.Context, externalIDs []string) (*models.User, bool, error) {
	if len(externalIDs) == 0 {
		return nil, false, nil
	}

	var user models.User
	err := r.db.WithContext(ctx).
		Where(`
			broadcast_info->'userDetails'->>'networkUserId' IN ?
			OR broadcast_info->'userDetails'->>'memberId' IN ?
		`, externalIDs, externalIDs).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &user, true, nil
}

func (r *UserRepository) UpdateBroadcastState(ctx context.Context, userID uuid.UUID, broadcastInfo []byte) error {
	if userID == uuid.Nil {
		return errors.New("broadcast user ID is required")
	}
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"broadcast_info": datatypes.JSON(broadcastInfo),
			"is_live":        true,
			"is_online":      true,
		}).Error
}

func (r *UserRepository) publicProfileQuery(input string) *gorm.DB {
	return r.db.
		Preload("Engagements").
		Preload("Avatar.File").
		Preload("Cover.File").
		Where("LOWER(user_name) = LOWER(?) OR LOWER(email) = LOWER(?)", input, input)
}

type publicUserProjectionRow struct {
	CursorCreatedAt time.Time              `gorm:"column:cursor_created_at"`
	CursorID        uuid.UUID              `gorm:"column:cursor_id"`
	PublicID        int64                  `gorm:"column:public_id"`
	UserName        string                 `gorm:"column:user_name"`
	DisplayName     string                 `gorm:"column:display_name"`
	Bio             *utils.LocalizedString `gorm:"column:bio;type:jsonb"`
	Website         string                 `gorm:"column:website"`
	DateOfBirth     *time.Time             `gorm:"column:date_of_birth"`
	PrivacyLevel    string                 `gorm:"column:privacy_level"`
	IsOnline        bool                   `gorm:"column:is_online"`
	IsPremium       bool                   `gorm:"column:is_premium"`
	CreatedAt       time.Time              `gorm:"column:created_at"`
	DefaultLanguage string                 `gorm:"column:default_language"`
	Languages       pq.StringArray         `gorm:"column:languages;type:text[]"`
	Hobbies         pq.StringArray         `gorm:"column:hobbies;type:text[]"`
	MoviesGenres    pq.StringArray         `gorm:"column:movies_genres;type:text[]"`
	TVShowsGenres   pq.StringArray         `gorm:"column:tv_shows_genres;type:text[]"`
	TheaterGenres   pq.StringArray         `gorm:"column:theater_genres;type:text[]"`
	CinemaGenres    pq.StringArray         `gorm:"column:cinema_genres;type:text[]"`
	ArtInterests    pq.StringArray         `gorm:"column:art_interests;type:text[]"`
	Entertainment   pq.StringArray         `gorm:"column:entertainment;type:text[]"`

	LocationDisplay *string `gorm:"column:location_display"`
	LocationCity    *string `gorm:"column:location_city"`
	LocationRegion  *string `gorm:"column:location_region"`
	LocationCountry *string `gorm:"column:location_country"`

	AvatarURL      *string        `gorm:"column:avatar_url"`
	AvatarVariants datatypes.JSON `gorm:"column:avatar_variants"`
	CoverURL       *string        `gorm:"column:cover_url"`
	CoverVariants  datatypes.JSON `gorm:"column:cover_variants"`
	Engagements    datatypes.JSON `gorm:"column:engagement_counts"`
}

func (r *UserRepository) FetchPublicUserProfile(ctx context.Context, username string) (*types.PublicUserProfile, error) {
	var row publicUserProjectionRow
	if err := r.publicUserProfileProjectionQuery(ctx, username).Take(&row).Error; err != nil {
		return nil, err
	}
	result := row.profileView()
	return &result, nil
}

func (r *UserRepository) publicUserProfileProjectionQuery(ctx context.Context, username string) *gorm.DB {
	query := r.db.WithContext(queryContext(ctx)).
		Table("users").
		Select(`
			users.public_id,
			users.user_name,
			users.display_name,
			users.bio,
			'' AS website,
			users.date_of_birth,
			users.privacy_level,
			users.is_online,
			users.is_premium,
			users.created_at,
			users.default_language,
			users.languages,
			users.hobbies,
			users.movies_genres,
			users.tv_shows_genres,
			users.theater_genres,
			users.cinema_genres,
			users.art_interests,
			users.entertainment,
			profile_location.display AS location_display,
			profile_location.city AS location_city,
			profile_location.region AS location_region,
			profile_location.country AS location_country,
			avatar_file.url AS avatar_url,
			avatar_file.variants AS avatar_variants,
			cover_file.url AS cover_url,
			cover_file.variants AS cover_variants,
			jsonb_build_object(
				'follower_count', COALESCE(NULLIF(user_engagement.counts->>'follower_count', ''), '0')::bigint,
				'following_count', COALESCE(NULLIF(user_engagement.counts->>'following_count', ''), '0')::bigint,
				'post_count', COALESCE(NULLIF(user_engagement.counts->>'post_count', ''), '0')::bigint,
				'blocking_count', COALESCE(NULLIF(user_engagement.counts->>'blocking_count', ''), '0')::bigint,
				'blocked_by_count', COALESCE(NULLIF(user_engagement.counts->>'blocked_by_count', ''), '0')::bigint,
				'like_given_count', COALESCE(NULLIF(user_engagement.counts->>'like_given_count', ''), '0')::bigint,
				'like_received_count', COALESCE(NULLIF(user_engagement.counts->>'like_received_count', ''), '0')::bigint,
				'dislike_given_count', COALESCE(NULLIF(user_engagement.counts->>'dislike_given_count', ''), '0')::bigint,
				'dislike_received_count', COALESCE(NULLIF(user_engagement.counts->>'dislike_received_count', ''), '0')::bigint,
				'match_count', COALESCE(NULLIF(user_engagement.counts->>'match_count', ''), '0')::bigint,
				'view_received_count', COALESCE(NULLIF(user_engagement.counts->>'view_received_count', ''), '0')::bigint
			) AS engagement_counts
		`).
		Joins(`
			LEFT JOIN locations AS profile_location
			  ON profile_location.contentable_id = users.id
			 AND profile_location.contentable_type = ?
			 AND profile_location.deleted_at IS NULL
		`, utils.LocationOwnerUser).
		Joins("LEFT JOIN medias AS avatar_media ON avatar_media.id = users.avatar_id").
		Joins("LEFT JOIN file_metadata AS avatar_file ON avatar_file.id = avatar_media.file_id").
		Joins("LEFT JOIN medias AS cover_media ON cover_media.id = users.cover_id").
		Joins("LEFT JOIN file_metadata AS cover_file ON cover_file.id = cover_media.file_id").
		Joins(`
			LEFT JOIN engagements AS user_engagement
			  ON user_engagement.contentable_id = users.id
			 AND user_engagement.contentable_type = ?
		`, models.EngagementContentableTypeUser)

	return publicUserVisibilityScope(query).
		Where("LOWER(users.user_name) = LOWER(?)", strings.TrimSpace(username))
}

func (r *UserRepository) SearchPublicUsers(filters types.Filter) ([]types.PublicUserSummary, error) {
	if filters.Search == nil || strings.TrimSpace(*filters.Search) == "" {
		return []types.PublicUserSummary{}, nil
	}

	var rows []publicUserProjectionRow
	if err := r.publicUserSearchProjectionQuery(filters).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]types.PublicUserSummary, 0, len(rows))
	for index := range rows {
		result = append(result, rows[index].summaryView())
	}
	return result, nil
}

func (r *UserRepository) publicUserSearchProjectionQuery(filters types.Filter) *gorm.DB {
	query := ""
	if filters.Search != nil {
		query = strings.TrimSpace(strings.ToLower(*filters.Search))
	}
	escapedQuery := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(query)
	containsPattern := "%" + escapedQuery + "%"
	prefixPattern := escapedQuery + "%"

	db := r.db
	if filters.Context != nil {
		db = db.WithContext(filters.Context)
	}
	db = db.
		Table("users").
		Select(`
			users.public_id,
			users.user_name,
			users.display_name,
			users.bio,
			users.is_online,
			profile_location.display AS location_display,
			profile_location.city AS location_city,
			profile_location.region AS location_region,
			profile_location.country AS location_country,
			avatar_file.url AS avatar_url,
			avatar_file.variants AS avatar_variants
		`).
		Joins(`
			LEFT JOIN locations AS profile_location
			  ON profile_location.contentable_id = users.id
			 AND profile_location.contentable_type = ?
			 AND profile_location.deleted_at IS NULL
		`, utils.LocationOwnerUser).
		Joins("LEFT JOIN medias AS avatar_media ON avatar_media.id = users.avatar_id").
		Joins("LEFT JOIN file_metadata AS avatar_file ON avatar_file.id = avatar_media.file_id")
	db = publicUserVisibilityScope(db)

	if filters.Domain != nil {
		domain := strings.TrimSpace(string(*filters.Domain))
		if domain != "" && domain != string(models.AllDomains) && domain != string(models.UnknownDomain) {
			db = db.Where("users.domain = ?", domain)
		}
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		limit = constants.MAXIMUM_LIMIT
	}

	return db.
		Where("LOWER(users.user_name) LIKE ? ESCAPE '\\' OR LOWER(users.display_name) LIKE ? ESCAPE '\\'", containsPattern, containsPattern).
		Clauses(clause.OrderBy{Expression: clause.Expr{
			SQL:  "CASE WHEN LOWER(users.user_name) = ? THEN 0 WHEN LOWER(users.user_name) LIKE ? ESCAPE '\\' THEN 1 WHEN LOWER(users.display_name) LIKE ? ESCAPE '\\' THEN 2 ELSE 3 END ASC, LOWER(users.display_name) ASC",
			Vars: []interface{}{query, prefixPattern, prefixPattern},
		}}).
		Limit(limit)
}

// publicUserVisibilityScope centralizes the public discovery policy so every
// query adapter (profile, search, matches) applies the same account filters.
func publicUserVisibilityScope(db *gorm.DB) *gorm.DB {
	return db.
		Where("users.deleted_at IS NULL").
		Where("users.is_bot = ?", false).
		Where("users.user_role NOT IN ?", []constants.UserRole{
			constants.UserRoleBanned,
			constants.UserRoleDeleted,
			constants.UserRolePending,
		}).
		Where("users.privacy_level = ?", constants.PrivacyPublic)
}

func (row publicUserProjectionRow) profileView() types.PublicUserProfile {
	return types.PublicUserProfile{
		ID:              types.SnowflakeID(row.PublicID),
		PublicID:        types.SnowflakeID(row.PublicID),
		UserName:        row.UserName,
		DisplayName:     row.DisplayName,
		Bio:             projectionLocalizedString(row.Bio),
		Website:         row.Website,
		DateOfBirth:     row.DateOfBirth,
		PrivacyLevel:    row.PrivacyLevel,
		IsOnline:        row.IsOnline,
		IsPremium:       row.IsPremium,
		CreatedAt:       row.CreatedAt,
		DefaultLanguage: row.DefaultLanguage,
		Languages:       append([]string(nil), row.Languages...),
		Hobbies:         append([]string(nil), row.Hobbies...),
		MoviesGenres:    append([]string(nil), row.MoviesGenres...),
		TVShowsGenres:   append([]string(nil), row.TVShowsGenres...),
		TheaterGenres:   append([]string(nil), row.TheaterGenres...),
		CinemaGenres:    append([]string(nil), row.CinemaGenres...),
		ArtInterests:    append([]string(nil), row.ArtInterests...),
		Entertainment:   append([]string(nil), row.Entertainment...),
		Location:        row.locationView(),
		Avatar:          projectionMedia(row.AvatarURL, row.AvatarVariants),
		Cover:           projectionMedia(row.CoverURL, row.CoverVariants),
		Engagements: types.PublicUserEngagements{
			Counts: publicUserEngagementCounts(row.Engagements),
		},
	}
}

func (row publicUserProjectionRow) summaryView() types.PublicUserSummary {
	return types.PublicUserSummary{
		ID:          types.SnowflakeID(row.PublicID),
		PublicID:    types.SnowflakeID(row.PublicID),
		UserName:    row.UserName,
		DisplayName: row.DisplayName,
		Bio:         projectionLocalizedString(row.Bio),
		IsOnline:    row.IsOnline,
		Location:    row.locationView(),
		Avatar:      projectionMedia(row.AvatarURL, row.AvatarVariants),
	}
}

func (row publicUserProjectionRow) locationView() *types.PublicUserLocation {
	if row.LocationDisplay == nil && row.LocationCity == nil && row.LocationRegion == nil && row.LocationCountry == nil {
		return nil
	}
	return &types.PublicUserLocation{
		Display: row.LocationDisplay,
		City:    row.LocationCity,
		Region:  row.LocationRegion,
		Country: row.LocationCountry,
	}
}

func projectionLocalizedString(value *utils.LocalizedString) map[string]string {
	if value == nil || len(*value) == 0 {
		return nil
	}
	result := make(map[string]string, len(*value))
	for language, text := range *value {
		result[language] = text
	}
	return result
}

func projectionMedia(url *string, variants datatypes.JSON) *types.PublicUserMedia {
	view := &types.PublicUserMedia{}
	if url != nil {
		view.File.URL = *url
	}
	if len(variants) > 0 && string(variants) != "null" {
		view.File.Variants = append(json.RawMessage(nil), variants...)
	}
	if view.File.URL == "" && len(view.File.Variants) == 0 {
		return nil
	}
	return view
}

func publicUserEngagementCounts(raw datatypes.JSON) types.PublicUserEngagementCounts {
	var source map[string]json.RawMessage
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &source)
	}
	return types.PublicUserEngagementCounts{
		FollowerCount:        projectedCount(source["follower_count"]),
		FollowingCount:       projectedCount(source["following_count"]),
		PostCount:            projectedCount(source["post_count"]),
		BlockingCount:        projectedCount(source["blocking_count"]),
		BlockedByCount:       projectedCount(source["blocked_by_count"]),
		LikeGivenCount:       projectedCount(source["like_given_count"]),
		LikeReceivedCount:    projectedCount(source["like_received_count"]),
		DislikeGivenCount:    projectedCount(source["dislike_given_count"]),
		DislikeReceivedCount: projectedCount(source["dislike_received_count"]),
		MatchCount:           projectedCount(source["match_count"]),
		ViewReceivedCount:    projectedCount(source["view_received_count"]),
	}
}

func projectedCount(raw json.RawMessage) int64 {
	if len(raw) == 0 {
		return 0
	}
	var count int64
	if err := json.Unmarshal(raw, &count); err == nil {
		return count
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		count, _ = strconv.ParseInt(text, 10, 64)
	}
	return count
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
	exists, err := r.ExistsByUsername(input)
	if err != nil || exists {
		return exists, err
	}
	return r.ExistsByEmail(input)
}

func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	return activeUserIdentityExists(activeUsernameIdentityQuery(r.db, username))
}

func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	if strings.TrimSpace(email) == "" {
		return false, nil
	}
	return activeUserIdentityExists(activeEmailIdentityQuery(r.db, email))
}

func activeUserIdentityExists(query *gorm.DB) (bool, error) {
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func activeUsernameIdentityQuery(db *gorm.DB, username string) *gorm.DB {
	return activeUserIdentityQuery(db).
		Where("LOWER(user_name) = LOWER(?)", strings.TrimSpace(username))
}

func activeEmailIdentityQuery(db *gorm.DB, email string) *gorm.DB {
	return activeUserIdentityQuery(db).
		Where("LOWER(email) = LOWER(?)", strings.TrimSpace(email))
}

func activeUserIdentityQuery(db *gorm.DB) *gorm.DB {
	return db.
		Table("users").
		Where("deleted_at IS NULL")
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) Report(ctx context.Context, userPublicID int64, kind string, description string, authUser *models.User) error {
	if authUser == nil || authUser.ID == uuid.Nil {
		return ports.ErrReportTargetNotFound
	}

	var reportedUser models.User
	if err := r.db.WithContext(ctx).
		Select("id", "public_id").
		First(&reportedUser, "public_id = ?", userPublicID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.ErrReportTargetNotFound
		}
		return err
	}

	return createReport(ctx, r.db, reportedUser.ID, models.EngagementContentableTypeUser, authUser.ID, kind, description)
}

func (r *UserRepository) UpdateUser(u *models.User) error {
	return r.db.Save(u).Error
}

// UpdateUserProfile persists all profile fields and the optional location in
// one transaction. The application validates the command before this boundary;
// this adapter owns the all-or-nothing database guarantee.
func (r *UserRepository) UpdateUserProfile(ctx context.Context, update ports.UserProfileUpdate) error {
	if update.UserID == uuid.Nil {
		return ports.ErrNotFound
	}

	err := r.db.WithContext(queryContext(ctx)).Transaction(func(tx *gorm.DB) error {
		values := userProfileUpdateValues(update)
		if len(values) > 0 {
			result := tx.Model(&models.User{}).
				Where("id = ? AND deleted_at IS NULL", update.UserID).
				Updates(values)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ports.ErrNotFound
			}
		}

		if update.Location == nil {
			return nil
		}

		location := &utils.Location{
			ID:              uuid.New(),
			ContentableType: utils.LocationOwnerUser,
			ContentableID:   update.UserID,
			CountryCode:     &update.Location.CountryCode,
			Address:         &update.Location.Address,
			City:            &update.Location.City,
			Country:         &update.Location.Country,
			Region:          &update.Location.Region,
			Timezone:        &update.Location.Timezone,
			Display:         &update.Location.Display,
			Latitude:        &update.Location.Latitude,
			Longitude:       &update.Location.Longitude,
			LocationPoint:   utils.NewLocationPoint(update.Location.Latitude, update.Location.Longitude),
		}
		return upsertLocation(tx, location).Error
	})
	return translateUserProfileUpdateError(err)
}

func translateUserProfileUpdateError(err error) error {
	if err == nil {
		return nil
	}
	constraint := ""
	var pgxError *pgconn.PgError
	if errors.As(err, &pgxError) && pgxError.Code == "23505" {
		constraint = pgxError.ConstraintName
	}
	var pqError *pq.Error
	if constraint == "" && errors.As(err, &pqError) && string(pqError.Code) == "23505" {
		constraint = pqError.Constraint
	}
	switch constraint {
	case "uidx_users_active_user_name_ci":
		return domainuser.ErrUsernameAlreadyExists
	case "uidx_users_active_email_ci":
		return domainuser.ErrEmailAlreadyExists
	default:
		return err
	}
}

func userProfileUpdateValues(update ports.UserProfileUpdate) map[string]interface{} {
	values := make(map[string]interface{}, 8)
	if update.UserName != nil {
		values["user_name"] = *update.UserName
	}
	if update.DisplayName != nil {
		values["display_name"] = *update.DisplayName
	}
	if update.Email != nil {
		values["email"] = *update.Email
	}
	if update.PasswordHash != nil {
		values["password"] = *update.PasswordHash
	}
	if update.Website != nil {
		values["website"] = *update.Website
	}
	if update.Bio != nil {
		values["bio"] = utils.LocalizedString(update.Bio)
	}
	if update.DateOfBirth != nil {
		values["date_of_birth"] = *update.DateOfBirth
	}
	if update.PrivacyLevel != nil {
		values["privacy_level"] = *update.PrivacyLevel
	}
	return values
}

func (r *UserRepository) DeleteUser(filters types.Filter) error {
	userID, err := deleteUserID(filters)
	if err != nil {
		return err
	}

	db := r.db
	if filters.Context != nil {
		db = db.WithContext(filters.Context)
	}

	return db.
		Where("id = ?", userID).
		Delete(&models.User{}).Error
}

func deleteUserID(filters types.Filter) (uuid.UUID, error) {
	if filters.AuthUser != nil && filters.AuthUser.ID != uuid.Nil {
		return filters.AuthUser.ID, nil
	}
	if filters.UserUUID != uuid.Nil {
		return filters.UserUUID, nil
	}
	return uuid.Nil, errors.New("missing user uuid for delete")
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

// GetSessionUserByPublicID keeps the authentication hot path deliberately
// small. Authorization needs identity, role and the lightweight profile; it
// must not hydrate the user's complete engagement graph on every HTTP call.
func (r *UserRepository) GetSessionUserByPublicID(ctx context.Context, publicID int64) (*ports.SessionUser, error) {
	var user ports.SessionUser
	if err := r.sessionUserQuery(publicID).WithContext(ctx).
		Take(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) sessionUserQuery(publicID int64) *gorm.DB {
	return r.db.
		Table("users").
		Select(`
			users.id,
			users.public_id,
			users.domain,
			users.user_name,
			users.display_name,
			users.default_language,
			users.preferences_flags,
			users.user_role AS role,
			users.is_bot,
			users.balance,
			EXISTS (
				SELECT 1
				FROM locations
				WHERE locations.contentable_type = ?
				  AND locations.contentable_id = users.id
				  AND locations.deleted_at IS NULL
			) AS has_location
		`, utils.LocationOwnerUser).
		Where("users.public_id = ?", publicID).
		Where("users.deleted_at IS NULL").
		Where("users.is_bot = FALSE").
		Where("users.user_role NOT IN ?", []constants.UserRole{
			constants.UserRoleBanned,
			constants.UserRoleDeleted,
			constants.UserRolePending,
		})
}

func (r *UserRepository) GetUsersStartingWith(letter string, limit int) ([]models.User, error) {
	query := strings.TrimSpace(strings.ToLower(letter))
	return r.SearchUsers(types.Filter{Search: &query, Limit: limit})
}

func (r *UserRepository) SearchUsers(filters types.Filter) ([]models.User, error) {
	var users []models.User
	if filters.Search == nil {
		return users, nil
	}

	query := strings.TrimSpace(strings.ToLower(*filters.Search))
	if query == "" {
		return users, nil
	}

	// This method is also used by the public search action. Keep prefix hits
	// first for autocomplete, while allowing a query to match a later part of
	// a username or display name as users expect from the full search screen.
	escapedQuery := strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(query)
	containsPattern := "%" + escapedQuery + "%"
	prefixPattern := escapedQuery + "%"
	exactPattern := escapedQuery
	limit := filters.Limit
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		limit = constants.MAXIMUM_LIMIT
	}

	db := r.db
	if filters.Domain != nil {
		domain := strings.TrimSpace(string(*filters.Domain))
		if domain != "" && domain != string(models.AllDomains) && domain != string(models.UnknownDomain) {
			db = db.Where("domain = ?", domain)
		}
	}

	err := db.
		Preload("Avatar").
		Preload("Avatar.File").
		Preload("Cover").
		Preload("Cover.File").
		Limit(limit).
		Where("LOWER(user_name) LIKE ? ESCAPE '\\' OR LOWER(display_name) LIKE ? ESCAPE '\\'", containsPattern, containsPattern).
		Clauses(clause.OrderBy{Expression: clause.Expr{
			SQL:  "CASE WHEN LOWER(user_name) = ? THEN 0 WHEN LOWER(user_name) LIKE ? ESCAPE '\\' THEN 1 WHEN LOWER(display_name) LIKE ? ESCAPE '\\' THEN 2 ELSE 3 END ASC, LOWER(display_name) ASC",
			Vars: []interface{}{exactPattern, prefixPattern, prefixPattern},
		}}).
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

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserByUUIDdWithoutRelations(filters types.Filter) (*models.User, error) {
	var u models.User
	err :=
		r.db.WithContext(filters.Context).First(&u, "id = ?", filters.UserUUID).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByNameOrMailWithoutRelations(input string) (*models.User, error) {
	var userObj models.User
	err := r.db.
		Where("LOWER(user_name) = LOWER(?) OR LOWER(email) = LOWER(?) OR LOWER(display_name) = LOWER(?)", input, input, input).
		First(&userObj).Error
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func (r *UserRepository) GetBySubscriptionSourceID(source, externalID string) (*models.User, error) {
	var userObj models.User
	err := r.db.
		Where(`
			CASE jsonb_typeof(subscriptions)
				WHEN 'array' THEN EXISTS (
					SELECT 1
					FROM jsonb_array_elements(subscriptions) AS elem
					WHERE elem->>'source' = ? AND elem->>'id' = ?
				)
				WHEN 'object' THEN subscriptions->>'source' = ? AND subscriptions->>'id' = ?
				ELSE false
			END
		`, source, externalID, source, externalID).
		First(&userObj).Error
	if err != nil {
		return nil, err
	}
	return &userObj, nil
}

func (r *UserRepository) UpsertLocation(location *utils.Location) error {
	return upsertLocation(r.db, location).Error
}

func upsertLocation(db *gorm.DB, location *utils.Location) *gorm.DB {
	if location == nil {
		_ = db.AddError(errors.New("location is required"))
		return db
	}
	if location.ID == uuid.Nil {
		location.ID = uuid.New()
	}

	now := time.Now().UTC()
	location.UpdatedAt = now
	if location.CreatedAt.IsZero() {
		location.CreatedAt = now
	}

	return db.Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{Name: "contentable_type"},
				{Name: "contentable_id"},
			},
			TargetWhere: clause.Where{Exprs: []clause.Expression{
				clause.Expr{SQL: "deleted_at IS NULL"},
			}},
			DoUpdates: clause.AssignmentColumns([]string{
				"country_code",
				"address",
				"city",
				"country",
				"postal",
				"region",
				"postcode",
				"zip_code",
				"province",
				"town",
				"timezone",
				"display",
				"latitude",
				"longitude",
				"location_point",
				"ip_address",
				"updated_at",
			}),
		},
		clause.Returning{Columns: []clause.Column{{Name: "id"}}},
	).Create(location)
}

func (r *UserRepository) UpsertUserPreference(ctx context.Context, userID uuid.UUID, preferenceItemID string, enabled bool) error {
	if userID == uuid.Nil {
		return errors.New("authenticated user is required")
	}
	var pref models.PreferencesData
	if err := r.db.WithContext(queryContext(ctx)).Model(&models.Preferences{}).Select("data").First(&pref).Error; err != nil {
		return err
	}

	allCategories := append(append(pref.Attributes, pref.Interests...), pref.Fantasies...)

	var foundCategory *models.PreferenceCategory
	var foundItem *models.PreferenceItem
	for i, cat := range allCategories {
		for j, item := range cat.Items {
			if item.ID.String() == preferenceItemID {
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
	if foundItem.BitIndex < 0 || foundItem.BitIndex > 1_000_000 {
		return fmt.Errorf("invalid stored preference bit index")
	}

	return r.db.WithContext(queryContext(ctx)).Transaction(func(tx *gorm.DB) error {
		var current models.User
		if err := tx.
			Select("id", "preferences_flags").
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", userID).
			Take(&current).Error; err != nil {
			return err
		}

		encodedFlags, err := updatePreferenceFlags(current.PreferencesFlags, *foundCategory, *foundItem, enabled)
		if err != nil {
			return err
		}

		return tx.Model(&models.User{}).
			Where("id = ?", userID).
			Update("preferences_flags", encodedFlags).Error
	})
}

func updatePreferenceFlags(encoded string, category models.PreferenceCategory, selected models.PreferenceItem, enabled bool) (string, error) {
	var flags big.Int
	if encoded != "" {
		bytes, err := hex.DecodeString(encoded)
		if err != nil {
			return "", err
		}
		flags.SetBytes(bytes)
	}

	if !category.AllowMultiple {
		for _, item := range category.Items {
			if item.BitIndex < 0 || item.BitIndex > 1_000_000 {
				return "", fmt.Errorf("invalid stored preference bit index")
			}
			flags.SetBit(&flags, int(item.BitIndex), 0)
		}
	}
	if selected.BitIndex < 0 || selected.BitIndex > 1_000_000 {
		return "", fmt.Errorf("invalid stored preference bit index")
	}

	value := uint(0)
	if enabled {
		value = 1
	}
	flags.SetBit(&flags, int(selected.BitIndex), value)
	return hex.EncodeToString(flags.Bytes()), nil
}

const nearbyPointSQL = "ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography"
const nearbyDistanceSQL = "nearby_locations.location_point <-> " + nearbyPointSQL

type nearbyUserRow struct {
	PublicID       int64          `gorm:"column:public_id"`
	UserName       string         `gorm:"column:user_name"`
	DisplayName    string         `gorm:"column:display_name"`
	DateOfBirth    *time.Time     `gorm:"column:date_of_birth"`
	IsOnline       bool           `gorm:"column:is_online"`
	IsPremium      bool           `gorm:"column:is_premium"`
	Latitude       *float64       `gorm:"column:latitude"`
	Longitude      *float64       `gorm:"column:longitude"`
	AvatarPublicID *int64         `gorm:"column:avatar_public_id"`
	AvatarURL      *string        `gorm:"column:avatar_url"`
	AvatarVariants datatypes.JSON `gorm:"column:avatar_variants"`
	Distance       float64        `gorm:"column:distance"`
}

func nearbyUsersLimit(limit int) int {
	if limit <= 0 {
		return constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		return constants.MAXIMUM_LIMIT
	}
	return limit
}

func queryContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (r *UserRepository) nearbyUsersBaseQuery(filters types.Filter) *gorm.DB {
	query := r.db.WithContext(queryContext(filters.Context)).
		Table("users").
		Joins("LEFT JOIN medias AS avatar_media ON avatar_media.id = users.avatar_id").
		Joins("LEFT JOIN file_metadata AS avatar_file ON avatar_file.id = avatar_media.file_id").
		Where("users.deleted_at IS NULL").
		Where("users.is_bot = ?", false).
		Where("users.user_role NOT IN ?", []constants.UserRole{
			constants.UserRoleBanned,
			constants.UserRoleDeleted,
			constants.UserRolePending,
		}).
		Where("users.privacy_level = ?", constants.PrivacyPublic).
		Limit(nearbyUsersLimit(filters.Limit))

	if filters.Domain != nil {
		domain := strings.TrimSpace(*filters.Domain)
		if domain != "" && !strings.EqualFold(domain, string(models.AllDomains)) && !strings.EqualFold(domain, string(models.UnknownDomain)) {
			query = query.Where("users.domain = ?", domain)
		}
	}

	if filters.AuthUser != nil {
		query = query.
			Where("users.id <> ?", filters.AuthUser.ID).
			Where(`NOT EXISTS (
				SELECT 1
				FROM engagement_details AS nearby_blocks
				WHERE nearby_blocks.kind IN (?, ?)
				  AND ((nearby_blocks.engager_id = ? AND nearby_blocks.engagee_id = users.id)
				    OR (nearby_blocks.engager_id = users.id AND nearby_blocks.engagee_id = ?))
			)`, models.EngagementKindBlocking, models.EngagementKindBlockedBy, filters.AuthUser.ID, filters.AuthUser.ID)
	}

	return query
}

func nearbyUserProjection(locationAlias string, includeDistance bool) string {
	projection := fmt.Sprintf(`
		users.public_id,
		users.user_name,
		users.display_name,
		users.date_of_birth,
		users.is_online,
		users.is_premium,
		ST_Y(%[1]s.location_point::geometry) AS latitude,
		ST_X(%[1]s.location_point::geometry) AS longitude,
		avatar_media.public_id AS avatar_public_id,
		avatar_file.url AS avatar_url,
		avatar_file.variants AS avatar_variants
	`, locationAlias)
	if includeDistance {
		projection += ", " + nearbyDistanceSQL + " AS distance"
	}
	return projection
}

func (r *UserRepository) nearbyUsersLocationQuery(filters types.Filter, lat, lng float64) (*gorm.DB, error) {
	query := r.nearbyUsersBaseQuery(filters).
		Joins(`JOIN locations AS nearby_locations
			ON nearby_locations.contentable_id = users.id
			AND nearby_locations.contentable_type = 'user'
			AND nearby_locations.deleted_at IS NULL
			AND nearby_locations.location_point IS NOT NULL`).
		Select(nearbyUserProjection("nearby_locations", true), lng, lat)

	if filters.Cursor != nil {
		if filters.Distance == nil {
			return nil, fmt.Errorf("distance cursor is required for location-based pagination")
		}
		query = query.Where(
			fmt.Sprintf("(%s > ?) OR (%s = ? AND users.public_id > ?)", nearbyDistanceSQL, nearbyDistanceSQL),
			lng, lat, *filters.Distance,
			lng, lat, *filters.Distance, *filters.Cursor,
		)
	}

	return query.Clauses(clause.OrderBy{Expression: clause.Expr{
		SQL:  nearbyDistanceSQL + " ASC, users.public_id ASC",
		Vars: []interface{}{lng, lat},
	}}), nil
}

func (r *UserRepository) nearbyUsersWithoutLocationQuery(filters types.Filter) *gorm.DB {
	query := r.nearbyUsersBaseQuery(filters).
		Joins(`LEFT JOIN locations AS nearby_locations
			ON nearby_locations.contentable_id = users.id
			AND nearby_locations.contentable_type = 'user'
			AND nearby_locations.deleted_at IS NULL`).
		Select(nearbyUserProjection("nearby_locations", false)).
		Order("users.public_id ASC")
	if filters.Cursor != nil {
		query = query.Where("users.public_id > ?", *filters.Cursor)
	}
	return query
}

func nearbyUsersFromRows(rows []nearbyUserRow) []types.NearbyUser {
	users := make([]types.NearbyUser, 0, len(rows))
	for _, row := range rows {
		user := types.NearbyUser{
			PublicID:    types.SnowflakeID(row.PublicID),
			UserName:    row.UserName,
			DisplayName: row.DisplayName,
			DateOfBirth: row.DateOfBirth,
			IsOnline:    row.IsOnline,
			IsPremium:   row.IsPremium,
		}
		if row.Latitude != nil && row.Longitude != nil {
			user.Location = &types.NearbyLocation{Latitude: *row.Latitude, Longitude: *row.Longitude}
		}
		if row.AvatarPublicID != nil {
			avatar := &types.NearbyUserMedia{PublicID: types.SnowflakeID(*row.AvatarPublicID)}
			if row.AvatarURL != nil {
				avatar.File.URL = *row.AvatarURL
			}
			if len(row.AvatarVariants) > 0 && string(row.AvatarVariants) != "null" {
				avatar.File.Variants = json.RawMessage(append([]byte(nil), row.AvatarVariants...))
			}
			user.Avatar = avatar
		}
		users = append(users, user)
	}
	return users
}

func (r *UserRepository) authenticatedCoordinates(filters types.Filter) (float64, float64, bool, error) {
	if filters.Latitude != nil && filters.Longitude != nil {
		return *filters.Latitude, *filters.Longitude, true, nil
	}
	if filters.AuthUser == nil {
		return 0, 0, false, nil
	}

	var coordinates struct {
		Latitude  *float64 `gorm:"column:latitude"`
		Longitude *float64 `gorm:"column:longitude"`
	}
	result := r.db.WithContext(queryContext(filters.Context)).
		Table("locations").
		Select("ST_Y(location_point::geometry) AS latitude, ST_X(location_point::geometry) AS longitude").
		Where("contentable_type = ? AND contentable_id = ? AND deleted_at IS NULL AND location_point IS NOT NULL", utils.LocationOwnerUser, filters.AuthUser.ID).
		Limit(1).
		Scan(&coordinates)
	if result.Error != nil {
		return 0, 0, false, result.Error
	}
	if result.RowsAffected == 0 || coordinates.Latitude == nil || coordinates.Longitude == nil {
		return 0, 0, false, nil
	}
	return *coordinates.Latitude, *coordinates.Longitude, true, nil
}

func (r *UserRepository) FetchNearbyUsers(filters types.Filter) ([]types.NearbyUser, *float64, error) {
	lat, lng, useLocation, err := r.authenticatedCoordinates(filters)
	if err != nil {
		return nil, nil, err
	}

	var rows []nearbyUserRow
	if useLocation {
		query, err := r.nearbyUsersLocationQuery(filters, lat, lng)
		if err != nil {
			return nil, nil, err
		}
		if err := query.Scan(&rows).Error; err != nil {
			return nil, nil, err
		}
		users := nearbyUsersFromRows(rows)
		if len(rows) == 0 {
			return users, nil, nil
		}
		lastDistance := rows[len(rows)-1].Distance
		return users, &lastDistance, nil
	}

	if err := r.nearbyUsersWithoutLocationQuery(filters).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	return nearbyUsersFromRows(rows), nil, nil
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

	q := r.liveUsersWithoutLocationQuery(filters)

	if err := q.
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) liveUsersWithoutLocationQuery(filters types.Filter) *gorm.DB {
	q := r.db.Model(&models.User{})

	if filters.IsLive != nil {
		q = q.Where("is_live = ?", *filters.IsLive)
	}

	q = q.Order("public_id ASC").Limit(filters.Limit)

	if filters.Cursor != nil {
		q = q.Where("public_id > ?", *filters.Cursor)
	}

	return q
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

func (r *UserRepository) UpdateLocation(ctx context.Context, userID uuid.UUID, ipStr string) error {
	if userID == uuid.Nil {
		return nil
	}

	loc, err := r.GetLocationFromIP(ipStr)
	if err != nil {
		return err
	}

	loc.ContentableType = utils.LocationOwnerUser
	loc.ContentableID = userID
	return upsertLocation(r.db.WithContext(ctx), loc).Error
}

func (r *UserRepository) AddReferral(ctx context.Context, referrerID uuid.UUID, referredUserID uuid.UUID, rewardAmount decimal.Decimal) (*decimal.Decimal, error) {
	referral, err := domainuser.NewReferral(referrerID, referredUserID, rewardAmount)
	if err != nil {
		return nil, err
	}
	if r == nil || r.db == nil {
		return nil, errors.New("referral repository is not configured")
	}

	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackReferralTransactionLock.Lock()
		defer fallbackReferralTransactionLock.Unlock()
	}

	var newBalance decimal.Decimal
	applied := false
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isPostgres {
			if err := lockViewAggregate(tx, "referral:"+referral.DedupeKey()).Error; err != nil {
				return err
			}
		}

		referrer, err := lockReferralUsers(tx, referral)
		if err != nil {
			return err
		}
		newBalance = referrer.Balance

		exists, err := referralEngagementExists(tx, referral.DedupeKey())
		if err != nil {
			return err
		}
		if exists {
			return nil
		}

		if err := addTipInTransaction(
			tx,
			referral.ReferredID(),
			referral.ReferrerID(),
			referral.RewardAmount(),
			referral.ReferrerID(),
			models.EngagementContentableTypeUser,
			models.EngagementKindReferral,
			withAmountEngagementDedupeKey(referral.DedupeKey()),
		); err != nil {
			return err
		}

		newBalance, err = referral.Credit(referrer.Balance)
		if err != nil {
			return err
		}
		update := tx.Model(&models.User{}).
			Where("id = ?", referral.ReferrerID()).
			Update("balance", newBalance)
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return errReferralUserNotFound
		}
		applied = true
		return nil
	})
	if errors.Is(err, errEngagementDetailAlreadyExists) {
		return r.referralBalance(ctx, referral.ReferrerID())
	}
	if err != nil {
		return nil, err
	}

	if applied {
		r.sendReferralNotificationBestEffort(ctx, referral)
	}
	return &newBalance, nil
}

func lockReferralUsers(tx *gorm.DB, referral domainuser.Referral) (*models.User, error) {
	var users []models.User
	result := referralUsersForBalanceUpdateQuery(tx, referral).
		Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(users) != 2 {
		return nil, errReferralUserNotFound
	}
	for i := range users {
		if users[i].ID == referral.ReferrerID() {
			return &users[i], nil
		}
	}
	return nil, errReferralUserNotFound
}

// NO KEY UPDATE protects balance writes while remaining compatible with the
// KEY SHARE lock PostgreSQL takes when engagement-detail foreign keys are
// checked. A stronger FOR UPDATE here can deadlock against flows that lock an
// engagement aggregate before inserting its user-referencing detail.
func referralUsersForBalanceUpdateQuery(tx *gorm.DB, referral domainuser.Referral) *gorm.DB {
	return tx.
		Select("id", "balance").
		Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).
		Where("id IN ?", []uuid.UUID{referral.ReferrerID(), referral.ReferredID()}).
		Order("id ASC")
}

func referralEngagementExists(tx *gorm.DB, dedupeKey string) (bool, error) {
	var count int64
	err := tx.Model(&models.EngagementDetail{}).
		Where("dedupe_key = ?", dedupeKey).
		Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) referralBalance(ctx context.Context, referrerID uuid.UUID) (*decimal.Decimal, error) {
	var user models.User
	if err := r.db.WithContext(ctx).
		Select("id", "balance").
		First(&user, "id = ?", referrerID).Error; err != nil {
		return nil, err
	}
	return &user.Balance, nil
}

func (r *UserRepository) sendReferralNotificationBestEffort(ctx context.Context, referral domainuser.Referral) {
	if r.notificationRepo == nil {
		return
	}

	senderUser, err := r.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: referral.ReferredID()})
	if err != nil {
		return
	}
	receiverUser, err := r.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: referral.ReferrerID()})
	if err != nil {
		return
	}

	messageTitle := " New Referral Reward"
	messageText := "You received a new referral reward. Click to read."
	payload := notifications.NotificationPayload{Title: messageTitle, Body: messageText}
	if err := r.notificationRepo.SendNotificationToUser(*senderUser, *receiverUser, notifications.NotificationTypeReferral, messageTitle, messageText, payload); err != nil {
		fmt.Printf("Bildirim gönderilemedi user %s: %v\n", referral.ReferrerID(), err)
	}

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

func (r *UserRepository) GetPreferences() (*models.PreferencesData, error) {
	var preferences models.PreferencesData
	if err := r.DB().Model(&models.Preferences{}).Select("data").First(&preferences).Error; err != nil {
		return nil, err
	}
	return &preferences, nil
}
