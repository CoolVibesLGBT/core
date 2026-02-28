package handlers

import (
	"core/constants"
	"core/middleware"
	services "core/services/user"
	"core/types"
	"core/utils"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

type MatchHandler struct {
	service *services.MatchesService
}

func NewMatchHandler(service *services.MatchesService) *MatchHandler {
	return &MatchHandler{service: service}
}

func HandleGetUnseenUsers(s *services.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		users, err := s.GetUnseenUsers(c.Context(), user.ID, 10)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrUserNotFound, "Failed to get unseen users: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"users": users,
		}, "Unseen users fetched successfully")
	}
}

func HandleRecordView(s *services.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		// form-data / x-www-form-urlencoded otomatik parse ediliyor
		userIdStr := c.FormValue("public_id")
		if userIdStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "invalid form data",
			})
		}

		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    constants.ErrInvalidInput,
			})
		}

		targetUserId, err := s.UserRepo().GetUserUUIDByPublicID(userId)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"code":    constants.ErrInvalidInput,
			})
		}

		reactionStr := c.FormValue("reaction")

		isMatched, err := s.RecordView(
			c.Context(),
			auth_user.ID,
			targetUserId,
			types.ReactionType(reactionStr),
		)

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to get unseen users",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"matched":     isMatched,
			"target_user": userIdStr,
		})
	}
}

func HandleGetMatchesAfter(s *services.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		// ---- cursor ----
		cursorStr := c.FormValue("cursor")
		var cursor *time.Time
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"message": "invalid cursor format",
				})
			}
			cursor = &parsedTime
		}

		// ---- limit ----
		limit := 10
		if limitStr := c.FormValue("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		matches, err := s.GetMatchesAfter(
			c.Context(),
			auth_user.ID,
			cursor,
			limit,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to get unseen users",
			})
		}

		// ---- pagination cursor ----
		nextCursor := ""
		if len(matches) > 0 {
			lastMatch := matches[len(matches)-1]
			nextCursor = lastMatch.CreatedAt.Format(time.RFC3339)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"users":  matches,
			"cursor": nextCursor,
		})
	}
}

func HandleGetPassesAfter(s *services.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		// ---- cursor ----
		cursorStr := c.FormValue("cursor")
		var cursor *time.Time
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"message": "invalid cursor format",
				})
			}
			cursor = &parsedTime
		}

		// ---- limit ----
		limit := 10
		if limitStr := c.FormValue("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		passes, err := s.GetPassesAfter(
			c.Context(),
			auth_user.ID,
			cursor,
			limit,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Failed to get unseen users",
			})
		}

		// ---- pagination cursor ----
		nextCursor := ""
		if len(passes) > 0 {
			last := passes[len(passes)-1]
			nextCursor = last.CreatedAt.Format(time.RFC3339)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"users":  passes,
			"cursor": nextCursor,
		})
	}
}

func HandleGetLikesAfter(s *services.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		cursorStr := c.Query("cursor", "")
		var cursor *time.Time
		if cursorStr != "" {
			parsedTime, err := time.Parse(time.RFC3339, cursorStr)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid cursor format",
				})
			}
			cursor = &parsedTime
		}

		limitStr := c.Query("limit", "")
		limit := 10
		if limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		likes, err := s.GetLikesAfter(c.Context(), authUser.ID, cursor, limit)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get unseen users",
			})
		}

		nextCursor := ""
		if len(likes) > 0 {
			lastMatch := likes[len(likes)-1]
			nextCursor = lastMatch.CreatedAt.Format(time.RFC3339)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"users":  likes,
			"cursor": nextCursor,
		})
	}
}
