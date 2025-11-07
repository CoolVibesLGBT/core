package handlers

import (
	"coolvibes/constants"
	"coolvibes/middleware"
	"coolvibes/models/user/payloads"
	services "coolvibes/services/user"
	"coolvibes/utils"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
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

		newAvatar, err := s.UpdateAvatar(fileHeader, user)
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

		newCover, err := s.UpdateCover(fileHeader, user)
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

		newStory, err := s.AddStory(fileHeader, user)
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

func HandleSetUserAttribute(s *services.UserService) http.HandlerFunc {
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

		form := r.MultipartForm.Value
		attrIDs, exists := form["attribute_id"]
		if !exists || len(attrIDs) == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		var notes *string
		if noteVals, ok := form["notes"]; ok && len(noteVals) > 0 {
			notes = &noteVals[0]
		}

		// Tek attribute_id al
		attributeID, err := uuid.Parse(attrIDs[0])
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		attribute, err := s.GetAttribute(attributeID)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)

		}

		attr := &payloads.UserAttribute{
			CategoryType: attribute.Category,
			UserID:       auth_user.ID,
			AttributeID:  attributeID,
			Notes:        notes,
		}

		err = s.UpsertUserAttribute(attr)
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

func HandleSetUserInterests(s *services.UserService) http.HandlerFunc {
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

		form := r.MultipartForm.Value
		attrIDs, exists := form["interest_id"]
		if !exists || len(attrIDs) == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		var notes *string
		if noteVals, ok := form["notes"]; ok && len(noteVals) > 0 {
			notes = &noteVals[0]
		}

		// Tek attribute_id al
		interestId, err := uuid.Parse(attrIDs[0])
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		interest, err := s.GetInterestItem(interestId)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)

		}

		userInterest := &payloads.UserInterest{

			UserID:         auth_user.ID,
			InterestItemID: interest.ID,
			Notes:          notes,
		}

		err = s.UpsertUserInterest(userInterest)
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

func HandleSetUserFantasies(s *services.UserService) http.HandlerFunc {
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

		form := r.MultipartForm.Value
		fantasyIdRaw, exists := form["fantasy_id"]
		if !exists || len(fantasyIdRaw) == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		// Tek attribute_id al
		fantasyId, err := uuid.Parse(fantasyIdRaw[0])
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		fantasyInfo, err := s.GetFantasy(fantasyId)
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)

		}

		fantasy := &payloads.UserFantasy{
			FantasyID: fantasyInfo.ID,
			UserID:    auth_user.ID,
		}

		err = s.UpsertUserFantasy(fantasy)
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

func HandleSetUserSexualIdentities(s *services.UserService) http.HandlerFunc {
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

		form := r.MultipartForm.Value

		genderIDs, hasGender := form["gender_identity_id"]
		sexualIDs, hasSexual := form["sexual_orientation_id"]
		sexRoleIDs, hasRole := form["sexual_role_id"]

		// Üçü de boşsa hata döndür
		if (!hasGender || len(genderIDs) == 0) &&
			(!hasSexual || len(sexualIDs) == 0) &&
			(!hasRole || len(sexRoleIDs) == 0) {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		//

		fmt.Println("SEX ROLE IDS", sexRoleIDs)

		// En az birisi dolu olmalı
		if len(genderIDs) == 0 && len(sexualIDs) == 0 && len(sexRoleIDs) == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		err := s.UpsertUserSexualIdentify(auth_user.ID, genderIDs, sexualIDs, sexRoleIDs)
		if err != nil {

			utils.SendError(w, http.StatusInternalServerError, constants.ErrInvalidInput)
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

		stories, err := s.GetAllStories(limit)
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

		users, err := s.FetchNearbyUsers(auth_user, distance, &cursor, limit)
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

		if followerID == 0 || followeeID == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		if err := s.Follow(followerID, followeeID); err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]string{
			"message": "User followed successfully",
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

		if followerID == 0 || followeeID == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		if err := s.Unfollow(followerID, followeeID); err != nil {
			utils.SendError(w, http.StatusBadRequest, constants.ErrDatabaseError)
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]string{
			"message": "User unfollowed successfully",
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

		form := r.MultipartForm.Value
		followerID := auth_user.PublicID
		followeeIDStr := form["followee_id"][0]
		followeeID, err := strconv.ParseInt(followeeIDStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		if followerID == 0 || followeeID == 0 {
			utils.SendError(w, http.StatusBadRequest, constants.ErrInvalidInput)
			return
		}

		fmt.Println("FOLLOWER,FOLLOWEE", followerID, followeeID)
		status, err := s.ToggleFollow(followerID, followeeID)
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
