package handlers

import (
	"context"
	"core/adapters/inbound/http/middleware"
	"core/application/types"
	usecases "core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
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
			if errors.Is(err, context.DeadlineExceeded) {
				return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
			}
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
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
		if err != nil || userId <= 0 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		targetUserId, err := s.GetUserUUIDByPublicID(userId)
		if err != nil {
			return utils.SendError(c, fiber.StatusNotFound, constants.ErrUserNotFound)
		}

		reactionStr := c.FormValue("reaction")

		isMatched, err := s.RecordView(
			c.Context(),
			auth_user.ID,
			targetUserId,
			reactionStr,
		)

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
			}
			if errors.Is(err, domainuser.ErrMatchTargetUnavailable) {
				return utils.SendError(c, fiber.StatusNotFound, constants.ErrUserNotFound)
			}
			if errors.Is(err, domainuser.ErrInvalidMatchReaction) || errors.Is(err, domainuser.ErrSelfInteraction) {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
			}
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

		cursor, err := parseMatchListCursor(c.FormValue("cursor"))
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

		page, err := s.GetMatchesAfter(
			c.Context(),
			auth_user.ID,
			cursor,
			limit,
		)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
			}
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}

		var nextCursor *string
		if page.Next != nil {
			nextCursor, err = types.NewTimeUUIDCursor(page.Next.OccurredAt, page.Next.DetailID)
			if err != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"users":  page.Users,
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

		cursor, err := parseMatchListCursor(c.FormValue("cursor"))
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

		page, err := s.GetPassesAfter(
			c.Context(),
			auth_user.ID,
			cursor,
			limit,
		)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
			}
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}

		var nextCursor *string
		if page.Next != nil {
			nextCursor, err = types.NewTimeUUIDCursor(page.Next.OccurredAt, page.Next.DetailID)
			if err != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"users":  page.Users,
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

		cursor, err := parseMatchListCursor(c.Query("cursor", ""))
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

		page, err := s.GetLikesAfter(c.Context(), authUser.ID, cursor, limit)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
			}
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}

		var nextCursor *string
		if page.Next != nil {
			nextCursor, err = types.NewTimeUUIDCursor(page.Next.OccurredAt, page.Next.DetailID)
			if err != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"users":  page.Users,
			"cursor": nextCursor,
		})
	}
}

func parseMatchListCursor(raw string) (*types.MatchListCursor, error) {
	if raw == "" {
		return nil, nil
	}
	if values, ok := types.DecodePaginationCursor(raw); ok {
		occurredAt, ok := types.CursorCreatedAt(values)
		if !ok {
			return nil, errors.New("invalid cursor")
		}
		cursor := &types.MatchListCursor{OccurredAt: occurredAt}
		if _, hasID := values[types.CursorKeyUUID]; hasID {
			detailID, valid := types.CursorUUID(values)
			if !valid {
				return nil, errors.New("invalid cursor")
			}
			cursor.DetailID = detailID
		}
		return cursor, nil
	}

	// Accept the old timestamp cursor during the client migration. It cannot
	// disambiguate equal timestamps, but every newly emitted cursor can.
	parsedTime, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		parsedTime, err = time.Parse(time.RFC3339Nano, raw)
	}
	if err != nil {
		return nil, err
	}
	return &types.MatchListCursor{OccurredAt: parsedTime}, nil
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
