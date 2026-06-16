package handlers

import (
	usecases "core/application/usecases"
	"core/constants"
	"core/middleware"
	"core/types"
	"core/utils"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

type MatchHandler struct {
	service *usecases.MatchesService
}

func NewMatchHandler(service *usecases.MatchesService) *MatchHandler {
	return &MatchHandler{service: service}
}

func HandleGetUnseenUsers(s *usecases.MatchesService) fiber.Handler {
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

func HandleRecordView(s *usecases.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		// form-data / x-www-form-urlencoded otomatik parse ediliyor
		userIdStr := c.FormValue("public_id")
		if userIdStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrInvalidInput)
		}

		targetUserId, err := s.GetUserUUIDByPublicID(userId)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrInvalidInput)
		}

		reactionStr := c.FormValue("reaction")

		isMatched, err := s.RecordView(
			c.Context(),
			auth_user.ID,
			targetUserId,
			types.ReactionType(reactionStr),
		)

		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"matched":     isMatched,
			"target_user": userIdStr,
		})
	}
}

func HandleGetMatchesAfter(s *usecases.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		cursor, err := parseTimeCursor(c.FormValue("cursor"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "invalid cursor format",
			})
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}

		var nextCursor *string
		if len(matches) > 0 {
			lastMatch := matches[len(matches)-1]
			nextCursor, err = types.NewTimeCursor(lastMatch.CreatedAt)
			if err != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"users":  matches,
			"cursor": nextCursor,
		})
	}
}

func HandleGetPassesAfter(s *usecases.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		cursor, err := parseTimeCursor(c.FormValue("cursor"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}

		var nextCursor *string
		if len(passes) > 0 {
			last := passes[len(passes)-1]
			nextCursor, err = types.NewTimeCursor(last.CreatedAt)
			if err != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"users":  passes,
			"cursor": nextCursor,
		})
	}
}

func HandleGetLikesAfter(s *usecases.MatchesService) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		cursor, err := parseTimeCursor(c.Query("cursor", ""))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}

		var nextCursor *string
		if len(likes) > 0 {
			lastMatch := likes[len(likes)-1]
			nextCursor, err = types.NewTimeCursor(lastMatch.CreatedAt)
			if err != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"users":  likes,
			"cursor": nextCursor,
		})
	}
}

func parseTimeCursor(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if values, ok := types.DecodePaginationCursor(raw); ok {
		if createdAt, ok := types.CursorCreatedAt(values); ok {
			return &createdAt, nil
		}
		return nil, errors.New("invalid cursor")
	}
	parsedTime, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		parsedTime, err = time.Parse(time.RFC3339Nano, raw)
	}
	if err != nil {
		return nil, err
	}
	return &parsedTime, nil
}

func encodeTimeCursorPair(prev *time.Time, next *time.Time) (*string, *string, error) {
	var prevCursor *string
	var nextCursor *string
	var err error

	if prev != nil {
		prevCursor, err = types.NewTimeCursor(*prev)
		if err != nil {
			return nil, nil, err
		}
	}
	if next != nil {
		nextCursor, err = types.NewTimeCursor(*next)
		if err != nil {
			return nil, nil, err
		}
	}
	return prevCursor, nextCursor, nil
}
