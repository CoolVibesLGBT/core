package handlers

import (
	"bifrost/constants"
	"bifrost/middleware"
	services "bifrost/services/user"
	"bifrost/types"
	"bifrost/utils"
	"net/http"
	"strconv"
	"time"
)

type MatchHandler struct {
	service *services.MatchesService
}

func NewMatchHandler(service *services.MatchesService) *MatchHandler {
	return &MatchHandler{service: service}
}

func HandleGetUnseenUsers(s *services.MatchesService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}
		users, err := s.GetUnseenUsers(user.ID, 10)
		if err != nil {
			http.Error(w, "Failed to get unseen users", http.StatusInternalServerError)
			return
		}
		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"users": users,
		})
	}
}

func HandleRecordView(s *services.MatchesService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		userIdStr := r.FormValue("public_id")
		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		targetUserId, err := s.UserRepo().GetUserUUIDByPublicID(userId)
		if err != nil {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrInvalidInput)
			return
		}

		reactionStr := r.FormValue("reaction") //like dislike vs.

		isMatched, err := s.RecordView(auth_user.ID, targetUserId, types.ReactionType(reactionStr))
		if err != nil {
			http.Error(w, "Failed to get unseen users", http.StatusInternalServerError)
			return
		}
		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"matched":     isMatched,
			"target_user": userIdStr,
		})
	}
}

func HandleGetMatchesAfter(s *services.MatchesService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		cursorStr := r.FormValue("cursor")
		var cursor *time.Time
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				http.Error(w, "invalid cursor format", http.StatusBadRequest)
				return
			}
			cursor = &parsedTime
		}

		limitStr := r.FormValue("limit")
		limit := 10 // default değer
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		matches, err := s.GetMatchesAfter(auth_user.ID, cursor, int(limit))
		if err != nil {
			http.Error(w, "Failed to get unseen users", http.StatusInternalServerError)
			return
		}

		nextCursor := ""
		if len(matches) > 0 {
			lastMatch := matches[len(matches)-1]
			nextCursor = lastMatch.CreatedAt.Format(time.RFC3339)
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"users":  matches,
			"cursor": nextCursor,
		})
	}
}

func HandleGetPassesAfter(s *services.MatchesService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		cursorStr := r.FormValue("cursor")
		var cursor *time.Time
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				http.Error(w, "invalid cursor format", http.StatusBadRequest)
				return
			}
			cursor = &parsedTime
		}

		limitStr := r.FormValue("limit")
		limit := 10 // default değer
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		passes, err := s.GetPassesAfter(auth_user.ID, cursor, int(limit))
		if err != nil {
			http.Error(w, "Failed to get unseen users", http.StatusInternalServerError)
			return
		}

		nextCursor := ""
		if len(passes) > 0 {
			lastMatch := passes[len(passes)-1]
			nextCursor = lastMatch.CreatedAt.Format(time.RFC3339)
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"users":  passes,
			"cursor": nextCursor,
		})
	}
}

func HandleGetLikesAfter(s *services.MatchesService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		cursorStr := r.FormValue("cursor")
		var cursor *time.Time
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				http.Error(w, "invalid cursor format", http.StatusBadRequest)
				return
			}
			cursor = &parsedTime
		}

		limitStr := r.FormValue("limit")
		limit := 10 // default değer
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		likes, err := s.GetLikesAfter(auth_user.ID, cursor, int(limit))
		if err != nil {
			http.Error(w, "Failed to get unseen users", http.StatusInternalServerError)
			return
		}

		nextCursor := ""
		if len(likes) > 0 {
			lastMatch := likes[len(likes)-1]
			nextCursor = lastMatch.CreatedAt.Format(time.RFC3339)
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"users":  likes,
			"cursor": nextCursor,
		})
	}
}
