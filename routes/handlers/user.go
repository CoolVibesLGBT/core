package handlers

import (
	"core/constants"
	"core/middleware"
	"core/models"
	services "core/services/user"
	"core/utils"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func HandleRegister(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Multipart form'u al (error kontrolü ile)
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid form data",
			})
		}

		userObj, token, err := s.Register(form.Value)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserExists)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user":  userObj,
			"token": token,
		})
	}
}

func HandleLogin(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Multipart form verisini al
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid form data",
			})
		}

		userObj, token, err := s.Login(form.Value)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid credentials",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user":  userObj,
			"token": token,
		})
	}
}

func HandleFetchUserProfile(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user": userObj,
		})
	}
}

func HandleUploadAvatar(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		fileHeader, err := c.FormFile("avatar")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid avatar file",
			})
		}

		newAvatar, err := s.UpdateAvatar(c.Context(), fileHeader, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrMediaUploadFailed,
			})
		}

		user.AvatarID = &newAvatar.ID
		user.Avatar = newAvatar

		updatedUser, err := s.GetUserByID(user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrMediaUploadFailed,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user": updatedUser,
		})
	}
}

func HandleUploadCover(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Dosyayı al
		fileHeader, err := c.FormFile("cover")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No cover file uploaded",
			})
		}

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// UpdateCover fonksiyonunu çağır
		newCover, err := s.UpdateCover(c.Context(), fileHeader, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrMediaUploadFailed,
			})
		}

		// user içindeki avatar yerine cover alanlarını güncelle
		user.CoverID = &newCover.ID
		user.Cover = newCover

		updatedUser, err := s.GetUserByID(user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrMediaUploadFailed,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user": updatedUser,
		})
	}
}

func HandleUploadStory(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Dosyayı al (multipart form otomatik parse edilir)
		fileHeader, err := c.FormFile("story")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrMediaInvalidFile,
			})
		}

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": constants.ErrUnauthorized,
			})
		}

		newStory, err := s.AddStory(c.Context(), fileHeader, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrMediaUploadFailed,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"story": newStory,
		})
	}
}

func HandleUserInfo(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		userInfo, err := s.GetUserByID(auth_user.ID)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": constants.ErrUnauthorized,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user": userInfo,
		})
	}
}

func HandleSetUserPreferences(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		// Form verisini Fiber'dan al (multipart form parse'a gerek yok, Fiber zaten parse eder)
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

		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"user": userInfo,
		})
	}
}

func HandleFetchStories(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// limit parametresini query'den al, default 20 olsun
		limit := 20
		if l := c.Query("limit"); l != "" {
			if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		if limit > 20 { // maximum 20
			limit = 20
		}

		stories, err := s.GetAllStories(c.Context(), limit)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer) // utils.SendError fiber uyumlu olmalı
		}

		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"stories": stories,
		})
	}
}

func HandleFetchNearbyUsers(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Println("AuthMiddlewareWithoutCheck")

		// Fiber'de form parse işlemi otomatik olarak yapılır, elle ParseForm yok
		limit := 10
		if limitStr := c.FormValue("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		distance := 10
		if distanceStr := c.FormValue("distance"); distanceStr != "" {
			if parsedDistance, err := strconv.Atoi(distanceStr); err == nil && parsedDistance > 0 {
				distance = parsedDistance
			}
		}

		var cursor int64 = 0
		if cursorStr := c.FormValue("cursor"); cursorStr != "" {
			val, err := strconv.ParseInt(cursorStr, 10, 64)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid cursor",
				})
			}
			cursor = val
		}

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			fmt.Println("LOCATIONLESSx")
			// Burada istersen kullanıcı yoksa hata dön veya boş liste dön
			// return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
			// veya devam et
		}

		fmt.Println("distance", distance, cursor)

		users, err := s.FetchNearbyUsers(c.Context(), auth_user, distance, &cursor, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		var nextCursorStr *string
		if len(users) > 0 {
			last := users[len(users)-1]
			str := fmt.Sprintf("%d", last.PublicID)
			nextCursorStr = &str
		} else {
			nextCursorStr = nil
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"users":       users,
			"next_cursor": nextCursorStr,
		})
	}
}

func HandleFollow(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		// Form verisini Fiber'de c.FormValue ile alıyoruz
		followeeIDStr := c.FormValue("followee_id")
		if followeeIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		followerID := auth_user.PublicID
		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		status, err := s.Follow(c.Context(), followerID, followeeID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrDatabaseError,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "User followed successfully",
			"status":  status,
		})
	}
}

func HandleUnfollow(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		followeeIDStr := c.FormValue("followee_id")
		if followeeIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		followerID := auth_user.PublicID
		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		status, err := s.Unfollow(c.Context(), followerID, followeeID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrDatabaseError,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "User unfollowed successfully",
			"status":  status,
		})
	}
}

func HandleToggleFollow(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		followeeIDStr := c.FormValue("followee_id")
		if followeeIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		followerID := auth_user.PublicID
		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		fmt.Println("FOLLOWER,FOLLOWEE", followerID, followeeID)

		status, err := s.ToggleFollow(c.Context(), followerID, followeeID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrDatabaseError,
			})
		}

		message := "User unfollowed successfully"
		if status {
			message = "User followed successfully"
		}

		user, err := s.GetUserByID(auth_user.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrDatabaseError,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": message,
			"status":  status,
			"user":    user,
		})
	}
}

func HandleGetUsersStartingWith(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		searchStr := c.FormValue("query")
		limit := 15

		users, err := s.GetUsersStartingWith(searchStr, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": constants.ErrDatabaseError,
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"users": users,
		})
	}
}

func HandleUpdateUserProfile(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)

		}

		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid form data",
			})
		}

		formValues := form.Value // map[string][]string olarak direkt kullan

		user, err := s.UpdateUserProfile(*auth_user, formValues)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to update user profile",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"user":    user,
			"success": true,
		})
	}
}

func HandleFetchUserEngagements(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		engagement_type := c.FormValue("engagement_type")
		engageeIdStr := c.FormValue("user_id")
		engageeId, err := strconv.ParseInt(engageeIdStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidInput,
			})
		}

		var cursor *time.Time
		cursorStr := c.FormValue("cursor")
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid cursor format. Use RFC3339 format.",
				})
			}
			cursor = &parsedTime
		}

		limit := 100 // default limit
		limitStr := c.FormValue("limit")
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil || parsedLimit <= 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid limit value",
				})
			}
			limit = parsedLimit
		}

		engageeUser, err := s.UserRepository().GetUserByPublicIdWithoutRelations(engageeId)
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
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": constants.ErrInvalidEngagementKind,
			})
		}

		engagements, nextCursor, err := s.FetchUserEngagements(c.Context(), auth_user, engageeUser.ID, models.EngagementContentableTypeUser, engagementKind, cursor, limit)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrEngagementNotFound)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"engagements": engagements,
			"next_cursor": nextCursor,
			"prev_cursor": cursor,
			"success":     true,
		})
	}
}

func HandleUserLike(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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
	return func(c *fiber.Ctx) error {
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

		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"message": "User disliked successfully",
			"status":  status,
		})
	}
}

func HandleUserToggleLikeDislike(s *services.UserService, isLike bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		return utils.SendJSON(c, fiber.StatusOK, map[string]string{
			"message": message,
		})
	}
}

func HandleUserBlock(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"message": "User blocked successfully",
			"status":  status,
		})
	}
}

func HandleUserUnblock(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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

		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"message": "User unblocked successfully",
			"status":  status,
		})
	}
}

func HandleUserToggleBlock(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {

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

		return utils.SendJSON(c, fiber.StatusOK, map[string]string{
			"message": message,
		})
	}
}

func HandleUserNotifications(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {

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

		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"notifications": notifications,
			"prev_cursor":   cursor,
			"next_cursor":   nextCursor,
			"success":       true,
		})
	}
}

func HandleUserCheckIn(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {

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
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"success": true,
		})
	}
}

func HandleUserDelete(s *services.UserService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		if auth_user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, auth_user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		delUserError := s.DeleteUser(filters)
		if delUserError != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": delUserError.Error(),
			})
		}
		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"success": true,
		})
	}
}
