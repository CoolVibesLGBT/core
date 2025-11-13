package handlers

import (
	"coolvibes/constants"
	"coolvibes/middleware"
	services "coolvibes/services/user"
	"coolvibes/utils"
	"fmt"
	"net/http"
	"strconv"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func HandleRegister(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		form := r.MultipartForm.Value
		userObj, token, err := s.Register(form)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrUserExists)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user":  userObj,
			"token": token,
		})
	}
}

func HandleLogin(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		form := r.MultipartForm.Value

		userObj, token, err := s.Login(form)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user":  userObj,
			"token": token,
		})
	}
}

func HandleFetchUserProfile(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		nicknames, ok := r.MultipartForm.Value["nickname"]
		if !ok || len(nicknames) == 0 {
			utils.SendError(w, http.StatusBadRequest, "nickname is required")
			return
		}
		nickname := nicknames[0]

		// Service çağrısı
		userObj, err := s.FetchUserProfileByNickname(nickname)
		if err != nil {
			utils.SendError(w, http.StatusNotFound, "user not found")
			return
		}
		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user": userObj,
		})
	}
}

func HandleUploadAvatar(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(5 * 1024 * 1024 * 1024)
		if err != nil {
			http.Error(w, "Could not parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		file, _, err := r.FormFile("avatar")
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}
		defer file.Close()

		fileHeader := r.MultipartForm.File["avatar"][0]

		newAvatar, err := s.UpdateAvatar(r.Context(), fileHeader, user)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, constants.ErrMediaUploadFailed)
			return
		}
		user.AvatarID = &newAvatar.ID
		user.Avatar = newAvatar

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user": user,
		})

	}
}

func HandleUploadCover(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(5 * 1024 * 1024 * 1024)
		if err != nil {
			http.Error(w, "Could not parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		file, _, err := r.FormFile("cover")
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}
		defer file.Close()

		fileHeader := r.MultipartForm.File["cover"][0]

		newCover, err := s.UpdateCover(r.Context(), fileHeader, user)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, constants.ErrMediaUploadFailed)
			return
		}
		user.AvatarID = &newCover.ID
		user.Avatar = newCover

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user": user,
		})

	}
}

func HandleUploadStory(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(5 * 1024 * 1024 * 1024)
		if err != nil {
			http.Error(w, "Could not parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		file, _, err := r.FormFile("story")
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrMediaInvalidFile)
			return
		}
		defer file.Close()

		fileHeader := r.MultipartForm.File["story"][0]

		newStory, err := s.AddStory(r.Context(), fileHeader, user)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, constants.ErrMediaUploadFailed)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"story": newStory,
		})

	}
}

func HandleUserInfo(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		userInfo, err := s.GetUserByID(auth_user.ID)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user": userInfo,
		})
	}
}

func HandleSetUserPreferences(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		// Form verisini parse et
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		preferenceItemId := r.FormValue("id")
		if len(preferenceItemId) == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		bitIndex := r.FormValue("bit_index")
		if len(bitIndex) == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		enabledStr := r.FormValue("enabled")
		enabled, err := strconv.ParseBool(enabledStr)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		fmt.Println("BITINDEX", bitIndex, "Enabled", enabled)

		err = s.UpsertUserPreference(r.Context(), *auth_user, preferenceItemId, bitIndex, enabled)
		if err != nil {
			fmt.Println("ERROR", err)
			utils.SendError(w, http.StatusInternalServerError, constants.ErrUnknown)
			return
		}

		userInfo, err := s.GetUserByID(auth_user.ID)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user": userInfo,
		})
	}
}

func HandleFetchStories(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// limit parametresini query'den al, default 20 olsun
		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsedLimit, err := strconv.Atoi(l); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		if limit > 20 { //maximum 20
			limit = 20
		}

		stories, err := s.GetAllStories(r.Context(), limit)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, constants.ErrInternalServer) // kendi error yapınıza göre ayarla
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"stories": stories,
		})
	}
}

func HandleFetchNearbyUsers(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("AuthMiddlewareWithoutCheck")

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		// form’dan limit değerini al
		limitStr := r.FormValue("limit") // hem application/x-www-form-urlencoded hem multipart/form-data destekler
		limit := 10                      // default değer
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		distance := 10                         // kilometre
		distanceStr := r.FormValue("distance") // hem application/x-www-form-urlencoded hem multipart/form-data destekler
		if distanceStr != "" {
			if parsedDistance, err := strconv.Atoi(distanceStr); err == nil && parsedDistance > 0 {
				distance = parsedDistance
			}
		}

		cursorStr := r.FormValue("cursor")
		var cursor int64 = 0

		if cursorStr != "" {
			val, err := strconv.ParseInt(cursorStr, 10, 64)
			if err != nil {
				http.Error(w, "invalid cursor", http.StatusBadRequest)
				return
			}
			cursor = val
		}

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			fmt.Println("LOCATIONLESSx")
			//utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			//returnx
		}

		fmt.Println("distance", distance, cursor)

		users, err := s.FetchNearbyUsers(r.Context(), auth_user, distance, &cursor, limit)
		if err != nil {
			utils.SendJSON(w, http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}

		nextCursor := func() *int64 {
			if len(users) > 0 {
				last := users[len(users)-1]
				return &last.PublicID
			}
			return nil
		}()

		var nextCursorStr *string
		if nextCursor != nil {
			str := fmt.Sprintf("%d", *nextCursor)
			nextCursorStr = &str
		} else {
			nextCursorStr = nil // JSON'da null olarak serialize edilir
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"users":       users,
			"next_cursor": nextCursorStr,
		})

	}
}

func HandleFollow(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		form := r.MultipartForm.Value
		followerID := auth_user.PublicID
		followeeIDStr := form["followee_id"][0]
		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		status, err := s.Follow(r.Context(), followerID, followeeID)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"message": "User followed successfully",
			"status":  status,
		})
	}
}

func HandleUnfollow(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		form := r.MultipartForm.Value
		followerID := auth_user.PublicID
		followeeIDStr := form["followee_id"][0]
		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		status, err := s.Unfollow(r.Context(), followerID, followeeID)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"message": "User unfollowed successfully",
			"status":  status,
		})

	}
}

func HandleToggleFollow(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		followeeIDStr := r.FormValue("followee_id")
		followerID := auth_user.PublicID
		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		if followeeID == 0 || followerID == 0 || followeeID == followerID {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		fmt.Println("FOLLOWER,FOLLOWEE", followerID, followeeID)
		status, err := s.ToggleFollow(r.Context(), followerID, followeeID)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		var message string
		if status {
			message = "User unfollowed successfully"
		} else {
			message = "User followed successfully"

		}

		utils.SendJSON(w, http.StatusOK, map[string]string{
			"message": message,
		})
	}
}

func HandleGetUsersStartingWith(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		searchStr := r.FormValue("query")
		limit := 15

		users, err := s.GetUsersStartingWith(searchStr, limit)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"users": users,
		})
	}
}

func HandleUpdateUserProfile(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}
		form := r.MultipartForm.Value

		// Formdan kullanıcı bilgilerini al

		// Örnek: Kullanıcıyı güncelle
		user, err := s.UpdateUserProfile(*auth_user, form)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, "failed to update user profile")
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user":    user,
			"success": true,
		})
	}
}

func HandleFetchUserEngagements(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		//engagement_type

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"user":    nil,
			"success": true,
		})
	}
}

func HandleUserLike(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		engagement_type := r.FormValue("engagement_type")
		userIdStr := r.FormValue("user_id")

		authUserId := auth_user.PublicID
		requestUserId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		fmt.Println("engagement_type", engagement_type)
		fmt.Println("authUserId", authUserId)
		fmt.Println("requestUserId", requestUserId)

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"message": "User liked successfully",
			"status":  true,
		})
	}
}

func HandleUserDislike(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		likeeIdStr := r.FormValue("likee_id")
		likerId := auth_user.PublicID
		likeeId, err := strconv.ParseInt(likeeIdStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		if likeeId == 0 || likerId == 0 || likeeId == likerId {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		_, status, err := s.Dislike(r.Context(), *auth_user, likerId, likeeId)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"message": "User disliked successfully",
			"status":  status,
		})

	}
}

func HandleUserToggleLikeDislike(s *services.UserService, isLike bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		likeeIdStr := r.FormValue("likee_id")
		likerId := auth_user.PublicID
		likeeId, err := strconv.ParseInt(likeeIdStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		if likerId == 0 || likeeId == 0 || likeeId == likerId {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		fmt.Println("FOLLOWER,FOLLOWEE", likerId, likeeId)
		_, status, err := s.ToggleLike(r.Context(), *auth_user, likerId, likeeId, isLike)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		var message string
		if status {
			message = "User unfollowed successfully"
		} else {
			message = "User followed successfully"

		}

		utils.SendJSON(w, http.StatusOK, map[string]string{
			"message": message,
		})
	}
}

func HandleUserBlock(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		blockedIdStr := r.FormValue("blocked_id")
		blockerId := auth_user.PublicID
		blockedId, err := strconv.ParseInt(blockedIdStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		if blockerId == 0 || blockedId == 0 || blockerId == blockedId {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}
		// Block işlemi
		status, err := s.Block(r.Context(), *auth_user, blockerId, blockedId)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"message": "User blocked successfully",
			"status":  status,
		})
	}
}

func HandleUserUnblock(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		blockedIdStr := r.FormValue("blocked_id")
		blockerId := auth_user.PublicID
		blockedId, err := strconv.ParseInt(blockedIdStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		if blockerId == 0 || blockedId == 0 || blockerId == blockedId {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		status, err := s.Unblock(r.Context(), *auth_user, blockerId, blockedId)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"message": "User unblocked successfully",
			"status":  status,
		})

	}
}

func HandleUserToggleBlock(s *services.UserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		blockedIdStr := r.FormValue("blocked_id")
		blockerId := auth_user.PublicID
		blockedId, err := strconv.ParseInt(blockedIdStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		if blockerId == 0 || blockedId == 0 || blockerId == blockedId {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		fmt.Println("BLOCK,", blockerId, blockedId)
		status, err := s.ToggleBlock(r.Context(), *auth_user, blockerId, blockedId)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		var message string
		if status {
			message = "User blocked successfully"
		} else {
			message = "User unblocked successfully"

		}

		utils.SendJSON(w, http.StatusOK, map[string]string{
			"message": message,
		})
	}
}
