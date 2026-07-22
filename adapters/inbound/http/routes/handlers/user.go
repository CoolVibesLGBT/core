package handlers

import (
	"context"
	"core/adapters/inbound/http/middleware"
	legacyviews "core/application/legacyviews"
	"core/application/ports"
	"core/application/types"
	usecases "core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"core/utils"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func HandleUserReport(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		userID, err := strconv.ParseInt(requestField(c, "user_id"), 10, 64)
		if err != nil || userID <= 0 {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, usecases.ErrUserIDRequired.Error())
		}
		kind, description := reportFields(c)
		if kind == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, ports.ErrInvalidReportKind.Error())
		}
		if err := validateReportFields(kind, description); err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		err = s.Report(c.Context(), userID, kind, description, authUser)
		switch {
		case err == nil:
			return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "User reported successfully")
		case errors.Is(err, usecases.ErrCannotReportSelf), errors.Is(err, usecases.ErrUserIDRequired), errors.Is(err, ports.ErrInvalidReportKind):
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		case errors.Is(err, ports.ErrReportTargetNotFound):
			return utils.SendError(c, fiber.StatusNotFound, constants.ErrUserNotFound)
		default:
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrDatabaseError)
		}
	}
}

type UserHandler struct {
	service *usecases.UserService
}

func NewUserHandler(service *usecases.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func HandleRegister(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		userObj, token, err := s.RegisterUser(c.Context(), usecases.RegisterInput{
			Name:           multipartValue(form.Value, "name"),
			Nickname:       multipartValue(form.Value, "nickname"),
			Password:       multipartValue(form.Value, "password"),
			Domain:         multipartValue(form.Value, "domain"),
			Email:          multipartValue(form.Value, "email"),
			Captcha:        multipartValue(form.Value, "captcha"),
			RecaptchaToken: multipartValue(form.Value, "recaptchaToken"),
			Referral:       multipartValue(form.Value, "referralCode"),
		})
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusUnauthorized, constants.ErrUserExists, err.Error())
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"user":  usecases.AuthUserResponse(userObj),
			"token": token,
		})
	}
}

func HandleLogin(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		userObj, token, err := s.LoginUser(c.Context(), usecases.LoginInput{
			UserName: multipartValue(form.Value, "nickname"),
			Password: multipartValue(form.Value, "password"),
		})
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrInvalidInput)
		}

		payload := fiber.Map{
			"user":  usecases.AuthUserResponse(userObj),
			"token": token,
		}
		return utils.SendSuccess(c, fiber.StatusOK, payload)
	}
}

func multipartValue(values map[string][]string, key string) string {
	items := values[key]
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func HandleAuthCheck(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		sessionUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		// Authentication middleware deliberately stores a lightweight session
		// projection. Reload the account internally, then cross the HTTP boundary
		// through the allowlisted public-ID projection.
		user, err := s.GetUserByID(sessionUser.ID)
		if err != nil || user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		return utils.SendSuccess(c, fiber.StatusOK, usecases.AuthUserResponse(user))
	}
}

func HandleFetchUserProfile(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		username := c.FormValue("nickname")
		if username == "" {
			username = c.FormValue("username")
		}

		if username == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUsernameRequired)
		}
		userObj, err := s.FetchPublicUserProfileByUsername(c.Context(), username)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUserNotFound)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user": userObj,
		}, "User fetched successfully")
	}
}

func HandleUploadAvatar(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		fileHeader, err := c.FormFile("avatar")
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		newAvatar, err := s.UpdateAvatar(c.Context(), uploadedFile(fileHeader), user)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrMediaUploadFailed)
		}

		user.AvatarID = &newAvatar.ID
		user.Avatar = newAvatar

		updatedUser, err := s.GetUserByID(user.ID)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrMediaUploadFailed)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user": usecases.AuthUserResponse(updatedUser),
		}, "Avatar updated successfully")
	}
}

func HandleUploadCover(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Dosyayı al
		fileHeader, err := c.FormFile("cover")
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		// UpdateCover fonksiyonunu çağır
		newCover, err := s.UpdateCover(c.Context(), uploadedFile(fileHeader), user)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrMediaUploadFailed)
		}

		// user içindeki avatar yerine cover alanlarını güncelle
		user.CoverID = &newCover.ID
		user.Cover = newCover

		updatedUser, err := s.GetUserByID(user.ID)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrMediaUploadFailed)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user": usecases.AuthUserResponse(updatedUser),
		}, "Cover updated successfully")
	}
}

func HandleUploadStory(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrMediaInvalidFile)
		}
		storyFiles := append(form.File["story"], form.File["story[]"]...)
		images := form.File["images[]"]
		videos := form.File["videos[]"]
		if len(storyFiles)+len(images)+len(videos) == 0 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrMediaInvalidFile)
		}

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		newStory, err := s.AddStory(c.Context(), uploadedFormData(form.Value, storyFiles, images, videos), user)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrMediaUploadFailed)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"story": legacyviews.ProjectPublicPost(*newStory),
		}, "Story added successfully")
	}
}

func HandleUserInfo(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		userInfo, err := s.GetUserByID(auth_user.ID)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user": usecases.AuthUserResponse(userInfo),
		}, "User info fetched successfully")
	}
}

func HandleSetUserPreferences(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		preferenceItemId := c.FormValue("id")
		if len(preferenceItemId) == 0 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		enabledStr := c.FormValue("enabled")
		enabled, err := strconv.ParseBool(enabledStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		err = s.UpsertUserPreference(c.Context(), auth_user.ID, preferenceItemId, enabled)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrUnknown)
		}

		userInfo, err := s.GetUserByID(auth_user.ID)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user": usecases.AuthUserResponse(userInfo),
		}, "User preferences updated successfully")
	}
}

func HandleFetchStories(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		stories, err := s.GetAllStories(filters)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"stories": stories.Posts,
			"cursor":  stories.Cursor,
		}, "Stories fetched successfully")
	}
}

func HandleFetchNearbyUsers(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, _ := middleware.GetAuthenticatedUser(c)

		filters, err := ParseFilters(c, auth_user)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		users, lastDistance, err := s.FetchNearbyUsers(filters)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
			}
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrDatabaseError)
		}

		var cursorObj *types.Cursor
		if len(users) > 0 {
			last := users[len(users)-1]
			nextCursor, err := types.NewPublicIDDistanceCursor(int64(last.PublicID), lastDistance)
			if err != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
			cursorObj = &types.Cursor{
				Next:     nextCursor,
				Distance: lastDistance,
			}
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, map[string]interface{}{
			"users":  users,
			"cursor": cursorObj,
		}, "Users fetched successfully")
	}
}

func HandleFollow(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		// Form verisini Fiber'de c.FormValue ile alıyoruz
		followeeIDStr := c.FormValue("followee_id")
		if followeeIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		followerID := auth_user.PublicID
		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid followee ID")
		}

		status, err := s.Follow(c.Context(), followerID, followeeID)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, "User followed successfully")
	}
}

func HandleUnfollow(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		followeeIDStr := c.FormValue("followee_id")
		if followeeIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		followerID := auth_user.PublicID
		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid followee ID")
		}

		status, err := s.Unfollow(c.Context(), followerID, followeeID)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, "User unfollowed successfully")
	}
}

func HandleToggleFollow(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		followeeIDStr := c.FormValue("followee_id")
		if followeeIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		followerID := auth_user.PublicID
		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid followee ID")
		}

		fmt.Println("FOLLOWER,FOLLOWEE", followerID, followeeID)

		status, err := s.ToggleFollow(c.Context(), followerID, followeeID)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
		}

		message := "User unfollowed successfully"
		if status {
			message = "User followed successfully"
		}

		user, err := s.GetUserByID(auth_user.ID)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
			"user":   usecases.AuthUserResponse(user),
		}, message)
	}
}

func HandleGetUsersStartingWith(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		searchStr := strings.TrimSpace(getParam(c, "query"))
		limit := 15
		if searchStr == "" {
			return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
				"users": []types.PublicUserSummary{},
			}, "Users fetched successfully")
		}

		users, err := s.GetPublicUsersStartingWith(searchStr, limit)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"users": users,
		}, "Users fetched successfully")
	}
}

func HandleUpdateUserProfile(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)

		}

		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "invalid form data")
		}

		user, err := s.UpdateUserProfile(c.Context(), *auth_user, usecases.UpdateUserProfileInput{
			UserName:                multipartValue(form.Value, "username"),
			Password:                multipartValue(form.Value, "password"),
			CurrentPassword:         multipartValue(form.Value, "current_password"),
			NewPassword:             multipartValue(form.Value, "new_password"),
			NewPasswordConfirmation: multipartValue(form.Value, "new_password_confirmation"),
			Email:                   multipartValue(form.Value, "email"),
			DisplayName:             multipartValue(form.Value, "displayname"),
			Bio:                     multipartValue(form.Value, "bio"),
			Website:                 multipartValue(form.Value, "website"),
			DateOfBirth:             multipartValue(form.Value, "date_of_birth"),
			PrivacyLevel:            multipartValue(form.Value, "privacy_level"),
			LocationContentableType: multipartValue(form.Value, "location[contentable_type]"),
			LocationCountryCode:     multipartValue(form.Value, "location[country_code]"),
			LocationAddress:         multipartValue(form.Value, "location[address]"),
			LocationCity:            multipartValue(form.Value, "location[city]"),
			LocationCountry:         multipartValue(form.Value, "location[country]"),
			LocationRegion:          multipartValue(form.Value, "location[region]"),
			LocationTimezone:        multipartValue(form.Value, "location[timezone]"),
			LocationDisplay:         multipartValue(form.Value, "location[display]"),
			LocationLatitude:        multipartValue(form.Value, "location[latitude]"),
			LocationLongitude:       multipartValue(form.Value, "location[longitude]"),
		})
		if err != nil {
			status, code, safeMessage := profileUpdateHTTPError(err)
			if safeMessage != "" {
				return utils.SendErrorWithMessage(c, status, code, safeMessage)
			}
			return utils.SendError(c, status, code)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user":    usecases.AuthUserResponse(user),
			"success": true,
		}, "User profile updated successfully")
	}
}

func profileUpdateHTTPError(err error) (status int, code constants.ErrorCode, safeMessage string) {
	switch {
	case errors.Is(err, usecases.ErrUsernameAlreadyExists):
		return fiber.StatusConflict, constants.ErrUsernameTaken, err.Error()
	case errors.Is(err, usecases.ErrEmailAlreadyExists):
		return fiber.StatusConflict, constants.ErrInvalidInput, err.Error()
	case errors.Is(err, usecases.ErrInvalidCurrentPassword):
		return fiber.StatusUnauthorized, constants.ErrInvalidPassword, ""
	case errors.Is(err, ports.ErrNotFound):
		return fiber.StatusNotFound, constants.ErrUserNotFound, ""
	case errors.Is(err, context.DeadlineExceeded):
		return fiber.StatusGatewayTimeout, constants.ErrRequestTimeout, ""
	case isProfileValidationError(err):
		return fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error()
	default:
		return fiber.StatusInternalServerError, constants.ErrDatabaseError, ""
	}
}

func isProfileValidationError(err error) bool {
	for _, validationError := range []error{
		usecases.ErrLegacyPasswordField,
		usecases.ErrLocationCoordinates,
		usecases.ErrInvalidLocationOwner,
		domainuser.ErrInvalidBirthDate,
		domainuser.ErrFutureBirthDate,
		domainuser.ErrInvalidEmail,
		domainuser.ErrInvalidWebsite,
		domainuser.ErrInvalidPrivacyLevel,
		domainuser.ErrInvalidLatitude,
		domainuser.ErrInvalidLongitude,
		domainuser.ErrCurrentPasswordRequired,
		domainuser.ErrPasswordConfirmationMismatch,
	} {
		if errors.Is(err, validationError) {
			return true
		}
	}
	return false
}

func HandleFetchUserEngagements(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		engagement_type := c.FormValue("engagement_type")
		engageeIdStr := c.FormValue("user_id")
		engageeId, err := strconv.ParseInt(engageeIdStr, 10, 64)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid engagee ID")
		}

		cursor, err := parseTimeCursor(c.FormValue("cursor"))
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid cursor format.")
		}

		limit := 100 // default limit
		limitStr := c.FormValue("limit")
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil || parsedLimit <= 0 {
				return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid limit value")
			}
			limit = parsedLimit
		}

		engageeUser, err := s.GetUserByPublicID(c.Context(), engageeId)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUserNotFound)
		}

		engagementKind := models.EngagementKind(engagement_type)
		if !engagementKind.IsValid() {
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusBadRequest,
				constants.ErrInvalidEngagementKind,
				constants.ErrInvalidEngagementKind.String(),
			)
		}

		engagements, nextCursor, err := s.FetchUserEngagements(c.Context(), auth_user, engageeUser.ID, models.EngagementContentableTypeUser, engagementKind, cursor, limit)
		if err != nil {
			if errors.Is(err, usecases.ErrPrivateViewEngagements) {
				return utils.SendErrorWithMessage(c, fiber.StatusForbidden, constants.ErrUnauthorized, err.Error())
			}
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrEngagementNotFound, "Engagement not found")
		}
		prevCursorToken, nextCursorToken, err := encodeTimeCursorPair(cursor, nextCursor)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInternalServer, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"engagements": engagements,
			"cursor": fiber.Map{
				"prev": prevCursorToken,
				"next": nextCursorToken,
			},
		}, "Engagements fetched successfully")
	}
}

func HandleUserViewProfile(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		publicID, err := strconv.ParseInt(c.FormValue("public_id"), 10, 64)
		if err != nil || publicID <= 0 {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid profile public ID")
		}

		counted, err := s.ViewProfile(c.Context(), authUser, publicID)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrUserNotFound, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{"counted": counted}, "Profile viewed successfully")
	}
}

func HandleUserLike(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		likeeIdStr := c.FormValue("likee_id")
		if likeeIdStr == "" {
			likeeIdStr = c.FormValue("user_id")
		}
		if likeeIdStr == "" {
			likeeIdStr = c.Query("likee_id")
		}
		if likeeIdStr == "" {
			likeeIdStr = c.Query("user_id")
		}

		likerId := auth_user.PublicID
		likeeId, err := strconv.ParseInt(likeeIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		if likerId == 0 || likeeId == 0 || likeeId == likerId {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		_, status, err := s.Like(c.Context(), *auth_user, likerId, likeeId)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrDatabaseError)
		}

		message := "User liked successfully"
		if !status {
			message = "User unliked successfully"
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, message)
	}
}

func HandleUserDislike(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		// Fiber'da ParseForm yok, c.FormValue kullanılır
		likeeIdStr := c.FormValue("likee_id")
		likerId := auth_user.PublicID
		likeeId, err := strconv.ParseInt(likeeIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		if likeeId == 0 || likerId == 0 || likeeId == likerId {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		_, status, err := s.Dislike(c.Context(), *auth_user, likerId, likeeId)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrDatabaseError)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, "User disliked successfully")
	}
}

func HandleUserToggleLikeDislike(s *usecases.UserService, isLike bool) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		likeeIdStr := c.FormValue("likee_id")
		likerId := auth_user.PublicID
		likeeId, err := strconv.ParseInt(likeeIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		if likerId == 0 || likeeId == 0 || likeeId == likerId {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		fmt.Println("LIKER, LIKEE", likerId, likeeId)
		_, status, err := s.ToggleLike(c.Context(), *auth_user, likerId, likeeId, isLike)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrDatabaseError)
		}

		var message string
		if status {
			if isLike {
				message = "User liked successfully"
			} else {
				message = "User disliked successfully"
			}
		} else {
			if isLike {
				message = "User unliked successfully"
			} else {
				message = "User undisliked successfully"
			}
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, message)
	}
}

func HandleUserBlock(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		blockedIdStr := c.FormValue("blocked_id")
		blockerId := auth_user.PublicID
		blockedId, err := strconv.ParseInt(blockedIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		if blockerId == 0 || blockedId == 0 || blockerId == blockedId {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		status, err := s.Block(c.Context(), *auth_user, blockerId, blockedId)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrDatabaseError)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, "User blocked successfully")
	}
}

func HandleUserUnblock(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		blockedIdStr := c.FormValue("blocked_id")
		blockerId := auth_user.PublicID
		blockedId, err := strconv.ParseInt(blockedIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		if blockerId == 0 || blockedId == 0 || blockerId == blockedId {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		status, err := s.Unblock(c.Context(), *auth_user, blockerId, blockedId)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrDatabaseError)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, "User unblocked successfully")
	}
}

func HandleUserToggleBlock(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		blockedIdStr := c.FormValue("blocked_id")
		blockerId := auth_user.PublicID
		blockedId, err := strconv.ParseInt(blockedIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		if blockerId == 0 || blockedId == 0 || blockerId == blockedId {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		status, err := s.ToggleBlock(c.Context(), *auth_user, blockerId, blockedId)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrDatabaseError)
		}

		var message string
		if status {
			message = "User blocked successfully"
		} else {
			message = "User unblocked successfully"
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, message)
	}
}

func HandleUserToggleSubscribe(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		subscribedIdStr := c.FormValue("subscribed_id")
		subscriberId := auth_user.PublicID
		subscribedId, err := strconv.ParseInt(subscribedIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		if subscriberId == 0 || subscribedId == 0 || subscriberId == subscribedId {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		status, err := s.ToggleSubscribe(c.Context(), *auth_user, subscriberId, subscribedId)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrDatabaseError)
		}

		var message string
		if status {
			message = "User subscribed successfully"
		} else {
			message = "User unsubscribed successfully"
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"status": status,
		}, message)
	}
}

func HandleUserNotifications(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		limit := 100 // default limit
		limitStr := c.FormValue("limit")
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil || parsedLimit <= 0 {
				return utils.SendError(c, fiber.StatusBadRequest, "Invalid limit value")
			}
			limit = parsedLimit
		}

		cursor, err := parseTimeCursor(c.FormValue("cursor"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "Invalid cursor format.")
		}

		notifications, nextCursor, err := s.FetchUserNotifications(c.Context(), auth_user, cursor, limit)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to fetch notifications")
		}
		prevCursorToken, nextCursorToken, err := encodeTimeCursorPair(cursor, nextCursor)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to encode cursor")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"notifications": notifications,
			"cursor": fiber.Map{
				"prev": prevCursorToken,
				"next": nextCursorToken,
			},
		}, "Notifications fetched successfully")
	}
}

func HandleUserDelete(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		if auth_user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters := types.Filter{
			AuthUser: actorFromUser(auth_user),
			Context:  c.Context(),
			UserUUID: auth_user.ID,
		}
		delUserError := s.DeleteUser(filters)
		if delUserError != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to delete user")
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{}, "User deleted successfully")
	}
}
