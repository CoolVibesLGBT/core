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
	"strings"
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
			Status:     models.ReportStatus(requestField(c, "status")),
			Resolution: requestField(c, "resolution"),
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

	postID, err := strconv.ParseInt(requestField(c, "post_id"), 10, 64)
	if err != nil || postID == 0 {
		return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "post_id is required")
	}

	resolution := requestField(c, "resolution")
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
		Status: models.ReportStatusPending,
		Limit:  constants.DEFAULT_LIMIT,
	}
	status := strings.ToLower(strings.TrimSpace(requestField(c, "status")))
	if status == "all" {
		filter.Status = ""
		filter.AllStatuses = true
	} else if status != "" {
		filter.Status = models.ReportStatus(status)
	}

	if limitStr := requestField(c, "limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			return filter, errors.New("invalid limit")
		}
		filter.Limit = limit
	}

	if postIDStr := requestField(c, "post_id"); postIDStr != "" {
		postID, err := strconv.ParseInt(postIDStr, 10, 64)
		if err != nil || postID <= 0 {
			return filter, errors.New("invalid post_id")
		}
		filter.PostPublicID = postID
		filter.ContentableType = models.EngagementContentableTypePost
	}

	if userIDStr := requestField(c, "user_id"); userIDStr != "" {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil || userID <= 0 {
			return filter, errors.New("invalid user_id")
		}
		if filter.PostPublicID != 0 {
			return filter, errors.New("post_id and user_id cannot be combined")
		}
		filter.UserPublicID = userID
		filter.ContentableType = models.EngagementContentableTypeUser
	}

	contentType := strings.ToLower(strings.TrimSpace(requestField(c, "content_type")))
	if contentType == "" {
		contentType = strings.ToLower(strings.TrimSpace(requestField(c, "contentable_type")))
	}
	if contentType != "" {
		parsedType := models.EngagementContentableType(contentType)
		if parsedType != models.EngagementContentableTypePost && parsedType != models.EngagementContentableTypeUser {
			return filter, errors.New("invalid content_type")
		}
		if filter.ContentableType != "" && filter.ContentableType != parsedType {
			return filter, errors.New("content_type conflicts with target filter")
		}
		filter.ContentableType = parsedType
	}

	if reporterIDStr := requestField(c, "reporter_id"); reporterIDStr != "" {
		reporterID, err := strconv.ParseInt(reporterIDStr, 10, 64)
		if err != nil || reporterID <= 0 {
			return filter, errors.New("invalid reporter_id")
		}
		filter.ReporterPublicID = reporterID
	}

	if cursorStr := requestField(c, "cursor"); cursorStr != "" {
		cursor, cursorID, err := parseModerationCursor(cursorStr)
		if err != nil {
			return filter, err
		}
		filter.Cursor = &cursor
		filter.CursorID = cursorID
	}

	return filter, nil
}

func parseModerationCursor(raw string) (time.Time, *uuid.UUID, error) {
	if values, ok := types.DecodePaginationCursor(raw); ok {
		if cursor, ok := types.CursorCreatedAt(values); ok {
			if cursorID, ok := types.CursorUUID(values); ok {
				return cursor, &cursorID, nil
			}
			return cursor, nil, nil
		}
	}
	if cursor, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return cursor, nil, nil
	}
	if cursor, err := time.Parse(time.RFC3339, raw); err == nil {
		return cursor, nil, nil
	}
	return time.Time{}, nil, errors.New("invalid cursor")
}

func parseUUIDField(c fiber.Ctx, names ...string) (uuid.UUID, error) {
	for _, name := range names {
		if value := requestField(c, name); value != "" {
			return uuid.Parse(value)
		}
	}
	return uuid.Nil, errors.New("uuid field is required")
}

func optionalBoolField(c fiber.Ctx, name string) (bool, bool, error) {
	value := requestField(c, name)
	if value == "" {
		return false, false, nil
	}
	parsed, err := strconv.ParseBool(value)
	return parsed, true, err
}

func moderationError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, usecases.ErrModerationForbidden):
		return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
	case errors.Is(err, usecases.ErrInvalidReportStatus),
		errors.Is(err, usecases.ErrInvalidReportType),
		errors.Is(err, usecases.ErrReportIDRequired),
		errors.Is(err, usecases.ErrPostIDRequired),
		errors.Is(err, ports.ErrInvalidModerationAction):
		return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
	case errors.Is(err, ports.ErrReportNotFound), errors.Is(err, ports.ErrReportTargetNotFound):
		return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrInvalidInput, err.Error())
	case errors.Is(err, ports.ErrInvalidReportTransition):
		return utils.SendErrorWithMessage(c, fiber.StatusConflict, constants.ErrInvalidInput, err.Error())
	default:
		return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, err.Error())
	}
}
