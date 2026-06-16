package handlers

import (
	"core/application/ports"
	usecases "core/application/usecases"
	"core/constants"
	"core/middleware"
	"core/models"
	"core/types"
	"core/utils"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func HandleModerationFetchReports(s *usecases.ModerationService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		filter, err := parseModerationReportFilter(c)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		result, err := s.FetchReports(c.Context(), authUser, filter)
		if err != nil {
			return moderationError(c, err)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Reports fetched successfully")
	}
}

func HandleModerationResolveReport(s *usecases.ModerationService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		reportID, err := parseUUIDField(c, "report_id", "id")
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		input := ports.ModerationResolveInput{
			ReportID:   reportID,
			Status:     models.ReportStatus(getField(c, "status")),
			Resolution: getField(c, "resolution"),
		}

		if publishPost, ok, err := optionalBoolField(c, "publish_post"); err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		} else if ok {
			input.PublishPost = &publishPost
		}

		if hidePost, ok, err := optionalBoolField(c, "hide_post"); err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		} else if ok && hidePost {
			publishPost := false
			input.PublishPost = &publishPost
		}

		report, err := s.ResolveReport(c.Context(), authUser, input)
		if err != nil {
			return moderationError(c, err)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"report": report,
		}, "Report resolved successfully")
	}
}

func HandleModerationHidePost(s *usecases.ModerationService) fiber.Handler {
	return func(c fiber.Ctx) error {
		return handleModerationPostVisibility(c, s, false, "Post hidden successfully")
	}
}

func HandleModerationUnhidePost(s *usecases.ModerationService) fiber.Handler {
	return func(c fiber.Ctx) error {
		return handleModerationPostVisibility(c, s, true, "Post restored successfully")
	}
}

func handleModerationPostVisibility(c fiber.Ctx, s *usecases.ModerationService, published bool, message string) error {
	authUser, ok := middleware.GetAuthenticatedUser(c)
	if !ok {
		return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
	}

	postID, err := strconv.ParseInt(getField(c, "post_id"), 10, 64)
	if err != nil || postID == 0 {
		return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "post_id is required")
	}

	resolution := getField(c, "resolution")
	var post any
	if published {
		post, err = s.UnhidePost(c.Context(), authUser, postID, resolution)
	} else {
		post, err = s.HidePost(c.Context(), authUser, postID, resolution)
	}
	if err != nil {
		return moderationError(c, err)
	}

	return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
		"post": post,
	}, message)
}

func parseModerationReportFilter(c fiber.Ctx) (ports.ModerationReportFilter, error) {
	filter := ports.ModerationReportFilter{
		Status: models.ReportStatus(getField(c, "status")),
		Limit:  constants.DEFAULT_LIMIT,
	}

	if limitStr := getField(c, "limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			return filter, errors.New("invalid limit")
		}
		filter.Limit = limit
	}

	if postIDStr := getField(c, "post_id"); postIDStr != "" {
		postID, err := strconv.ParseInt(postIDStr, 10, 64)
		if err != nil {
			return filter, errors.New("invalid post_id")
		}
		filter.PostPublicID = postID
	}

	if reporterIDStr := getField(c, "reporter_id"); reporterIDStr != "" {
		reporterID, err := strconv.ParseInt(reporterIDStr, 10, 64)
		if err != nil {
			return filter, errors.New("invalid reporter_id")
		}
		filter.ReporterPublicID = reporterID
	}

	if cursorStr := getField(c, "cursor"); cursorStr != "" {
		cursor, err := parseModerationCursor(cursorStr)
		if err != nil {
			return filter, err
		}
		filter.Cursor = &cursor
	}

	return filter, nil
}

func parseModerationCursor(raw string) (time.Time, error) {
	if values, ok := types.DecodePaginationCursor(raw); ok {
		if cursor, ok := types.CursorCreatedAt(values); ok {
			return cursor, nil
		}
	}
	if cursor, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return cursor, nil
	}
	if cursor, err := time.Parse(time.RFC3339, raw); err == nil {
		return cursor, nil
	}
	return time.Time{}, errors.New("invalid cursor")
}

func parseUUIDField(c fiber.Ctx, names ...string) (uuid.UUID, error) {
	for _, name := range names {
		if value := getField(c, name); value != "" {
			return uuid.Parse(value)
		}
	}
	return uuid.Nil, errors.New("uuid field is required")
}

func optionalBoolField(c fiber.Ctx, name string) (bool, bool, error) {
	value := getField(c, name)
	if value == "" {
		return false, false, nil
	}
	parsed, err := strconv.ParseBool(value)
	return parsed, true, err
}

func getField(c fiber.Ctx, name string) string {
	if value := c.FormValue(name); value != "" {
		return value
	}
	return c.Query(name)
}

func moderationError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usecases.ErrModerationForbidden):
		return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
	case errors.Is(err, usecases.ErrInvalidReportStatus),
		errors.Is(err, usecases.ErrReportIDRequired),
		errors.Is(err, usecases.ErrPostIDRequired):
		return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
	default:
		return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
	}
}
