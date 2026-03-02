package handlers

import (
	"core/constants"
	"core/middleware"
	"core/models"
	services "core/services/user"
	"core/types"
	"core/utils"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func HandleRegister(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		userObj, token, err := s.Register(c.Context(), form.Value)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserExists)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"user":  userObj,
			"token": token,
		})
	}
}

func HandleLogin(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		userObj, token, err := s.Login(c.Context(), form.Value)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrInvalidInput)
		}

		payload := fiber.Map{
			"user":  userObj,
			"token": token,
		}
		return utils.SendSuccess(c, fiber.StatusOK, payload)
	}
}

func HandleAuthCheck(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}
		return utils.SendSuccess(c, fiber.StatusOK, user)
	}
}

func HandleFetchUserProfile(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		username := c.FormValue("nickname")
		if username == "" {
			username = c.FormValue("username")
		}

		if username == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUsernameRequired)
		}
		userObj, err := s.FetchUserProfileByUsername(username)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUserNotFound)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user": userObj,
		}, "User fetched successfully")
	}
}

func HandleUploadAvatar(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		fileHeader, err := c.FormFile("avatar")
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		newAvatar, err := s.UpdateAvatar(c.Context(), fileHeader, user)
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
			"user": updatedUser,
		}, "Avatar updated successfully")
	}
}

func HandleUploadCover(s *services.UserService) fiber.Handler {
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
		newCover, err := s.UpdateCover(c.Context(), fileHeader, user)
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
			"user": updatedUser,
		}, "Cover updated successfully")
	}
}

func HandleUploadStory(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Dosyayı al (multipart form otomatik parse edilir)
		fileHeader, err := c.FormFile("story")
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrMediaInvalidFile)
		}

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		newStory, err := s.AddStory(c.Context(), fileHeader, user)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrMediaUploadFailed)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"story": newStory,
		}, "Story added successfully")
	}
}

func HandleUserInfo(s *services.UserService) fiber.Handler {
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
			"user": userInfo,
		}, "User info fetched successfully")
	}
}

func HandleSetUserPreferences(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		preferenceItemId := c.FormValue("id")
		if len(preferenceItemId) == 0 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		bitIndex := c.FormValue("bit_index")
		if len(bitIndex) == 0 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		enabledStr := c.FormValue("enabled")
		enabled, err := strconv.ParseBool(enabledStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		fmt.Println("BITINDEX", bitIndex, "Enabled", enabled)

		err = s.UpsertUserPreference(c.Context(), *auth_user, preferenceItemId, bitIndex, enabled)
		if err != nil {
			fmt.Println("ERROR", err)
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrUnknown)
		}

		userInfo, err := s.GetUserByID(auth_user.ID)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user": userInfo,
		}, "User preferences updated successfully")
	}
}

func HandleFetchStories(s *services.UserService) fiber.Handler {
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
			"stories": stories,
		}, "Stories fetched successfully")
	}
}

func HandleFetchNearbyUsers(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			fmt.Println("LOCATIONLESSx")
		}

		filters, err := ParseFilters(c, auth_user)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		users, err := s.FetchNearbyUsers(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
		}

		var nextCursorStr *string
		if len(users) > 0 {
			last := users[len(users)-1]
			str := fmt.Sprintf("%d", last.PublicID)
			nextCursorStr = &str
		} else {
			nextCursorStr = nil
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, map[string]interface{}{
			"users":  users,
			"cursor": nextCursorStr,
		}, "Users fetched successfully")
	}
}

func HandleFollow(s *services.UserService) fiber.Handler {
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

func HandleUnfollow(s *services.UserService) fiber.Handler {
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

func HandleToggleFollow(s *services.UserService) fiber.Handler {
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
			"user":   user,
		}, message)
	}
}

func HandleGetUsersStartingWith(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		searchStr := c.FormValue("query")
		limit := 15

		users, err := s.GetUsersStartingWith(searchStr, limit)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"users": users,
		}, "Users fetched successfully")
	}
}

func HandleUpdateUserProfile(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)

		}

		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "invalid form data")
		}

		formValues := form.Value // map[string][]string olarak direkt kullan

		user, err := s.UpdateUserProfile(c.Context(), *auth_user, formValues)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"user":    user,
			"success": true,
		}, "User profile updated successfully")
	}
}

func HandleFetchUserEngagements(s *services.UserService) fiber.Handler {
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

		var cursor *time.Time
		cursorStr := c.FormValue("cursor")
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid cursor format. Use RFC3339 format.")
			}
			cursor = &parsedTime
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

		engageeUser, err := s.UserRepository().GetUserByPublicIdWithoutRelations(types.Filter{Context: c.Context(), UserID: engageeId})
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUserNotFound)
		}

		var engagementKind models.EngagementKind
		switch engagement_type {
		case "followings":
			engagementKind = models.EngagementKindFollowing
		case "followers":
			engagementKind = models.EngagementKindFollower
		default:
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidEngagementKind, "Invalid engagement kind")
		}

		engagements, nextCursor, err := s.FetchUserEngagements(c.Context(), auth_user, engageeUser.ID, models.EngagementContentableTypeUser, engagementKind, cursor, limit)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrEngagementNotFound, "Engagement not found")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"engagements": engagements,
			"cursor": fiber.Map{
				"prev": cursor,
				"next": nextCursor,
			},
		}, "Engagements fetched successfully")
	}
}

func HandleUserLike(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		// Fiber'da ParseForm yok, doğrudan c.FormValue kullan
		engagement_type := c.FormValue("engagement_type")
		userIdStr := c.FormValue("user_id")

		authUserId := auth_user.PublicID
		requestUserId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		//todo: not implemented

		fmt.Println("engagement_type", engagement_type)
		fmt.Println("authUserId", authUserId)
		fmt.Println("requestUserId", requestUserId)

		/*utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"message": "User liked successfully",
			"status":  true,
		})*/
		return utils.SendError(c, fiber.StatusBadRequest, constants.ErrMethodNotImplemented)

	}
}

func HandleUserDislike(s *services.UserService) fiber.Handler {
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

func HandleUserToggleLikeDislike(s *services.UserService, isLike bool) fiber.Handler {
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

func HandleUserBlock(s *services.UserService) fiber.Handler {
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

func HandleUserUnblock(s *services.UserService) fiber.Handler {
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

func HandleUserToggleBlock(s *services.UserService) fiber.Handler {
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

		fmt.Println("BLOCK,", blockerId, blockedId)
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

func HandleUserNotifications(s *services.UserService) fiber.Handler {
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

		var cursor *time.Time
		cursorStr := c.FormValue("cursor")
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				return utils.SendError(c, fiber.StatusBadRequest, "Invalid cursor format. Use RFC3339 format.")
			}
			cursor = &parsedTime
		}

		notifications, nextCursor, err := s.FetchUserNotifications(c.Context(), auth_user, cursor, limit)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to fetch notifications")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"notifications": notifications,
			"cursor": fiber.Map{
				"prev": cursor,
				"next": nextCursor,
			},
		}, "Notifications fetched successfully")
	}
}

func HandleUserCheckIn(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		if auth_user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		checkInKind := c.Params("check_in")

		fmt.Println("CheckIn", checkInKind)

		err := s.CheckIn(c.Context())

		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to check in")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{}, "Check in successful")
	}
}

func HandleUserDelete(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		if auth_user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, auth_user)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to parse filters")
		}
		delUserError := s.DeleteUser(filters)
		if delUserError != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to delete user")
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{}, "User deleted successfully")
	}
}
