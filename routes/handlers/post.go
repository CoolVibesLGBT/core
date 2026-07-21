package handlers

import (
	"core/application/ports"
	usecases "core/application/usecases"
	"core/constants"
	"core/middleware"
	postmodel "core/models/post"
	postpayloads "core/models/post/payloads"
	"core/types"
	"core/utils"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PostHandler struct {
	service *usecases.PostService
}

func NewPostHandler(service *usecases.PostService) *PostHandler {
	return &PostHandler{service: service}
}

func HandleCreate(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Could not parse multipart form: " + err.Error(),
			})
		}

		formParams := form.Value
		images := form.File["images[]"]
		videos := form.File["videos[]"]

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		postKind := postmodel.PostKindStatus

		if kind := c.FormValue("kind"); kind != "" && postmodel.IsValidPostKind(kind) {
			postKind = postmodel.PostKind(kind)
		}

		post, err := s.CreatePost(c.Context(), uploadedFormData(formParams, images, videos), user, postKind)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrPostCreateFailed)
		}

		return utils.SendSuccess(c, fiber.StatusCreated, post)
	}
}

func HandleVote(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		choiceIdStr := c.FormValue("choice_id")
		weightStr := c.FormValue("weight")
		rankStr := c.FormValue("rank")

		choiceId, err := uuid.Parse(choiceIdStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrChoiceIDInvalid)
		}

		weight := 1
		if weightStr != "" {
			weight, err = strconv.Atoi(weightStr)
			if err != nil || weight <= 0 {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrWeightInvalid)
			}
		}

		rank := 0
		if rankStr != "" {
			rank, err = strconv.Atoi(rankStr)
			if err != nil || rank < 0 {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrRankInvalid)
			}
		}

		err = s.Vote(c.Context(), choiceId, weight, rank, user.ID)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrVoteFailed)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"message": "Vote registered successfully",
		})
	}
}

func HandlePostEventRSVP(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		postID, err := strconv.ParseInt(c.FormValue("post_id"), 10, 64)
		if err != nil || postID <= 0 {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "invalid post_id")
		}

		rawStatus := strings.ToLower(strings.TrimSpace(c.FormValue("status")))
		var status *postpayloads.EventAttendanceStatus
		if rawStatus != "" && rawStatus != "clear" && rawStatus != "none" {
			parsed, valid := postpayloads.ParseEventAttendanceStatus(rawStatus)
			if !valid || string(parsed) != rawStatus {
				return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "status must be going, not_going, maybe, or clear")
			}
			status = &parsed
		}

		result, err := s.SetEventRSVP(c.Context(), postID, authUser, status)
		if err != nil {
			switch {
			case errors.Is(err, postpayloads.ErrEventNotFound):
				return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrPostNotFound, err.Error())
			case errors.Is(err, postpayloads.ErrEventClosed), errors.Is(err, postpayloads.ErrEventAtCapacity):
				return utils.SendErrorWithMessage(c, fiber.StatusConflict, constants.ErrInvalidAction, err.Error())
			default:
				return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInternalServer, err.Error())
			}
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Event RSVP updated successfully")
	}
}

func HandlePostDelete(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		if err := s.Delete(filters); err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrPostDeleteFailed)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Post deleted successfully")
	}
}

func HandlePostLike(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		// authenticated user
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		// do like
		err = s.Like(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Post liked successfully")
	}
}

func HandlePostBanana(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		filters, err := ParseFilters(c, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		err = s.Banana(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, err.Error())
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Post banana successfully")
	}
}

func HandlePostDislike(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		filters, err := ParseFilters(c, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		err = s.Dislike(filters)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDislikePost)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"success": true,
		})
	}
}

func HandlePostBookmark(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		err = s.Bookmark(filters)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToBookmarkPost)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Post bookmarked successfully")
	}
}

func HandlePostReport(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		postIdStr := requestField(c, "post_id")
		if postIdStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrPostNotFound)
		}

		postId, err := strconv.ParseInt(postIdStr, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrPostNotFound)
		}

		kind, description := reportFields(c)
		if kind == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, ports.ErrInvalidReportKind.Error())
		}
		if err := validateReportFields(kind, description); err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		err = s.Report(c.Context(), postId, kind, description, user)
		if err != nil {
			switch {
			case errors.Is(err, ports.ErrInvalidReportKind):
				return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
			case errors.Is(err, ports.ErrReportTargetNotFound):
				return utils.SendError(c, fiber.StatusNotFound, constants.ErrPostNotFound)
			default:
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrDatabaseError)
			}
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Post reported successfully")
	}
}

func HandlePostView(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		//todo:without middleware
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		counted, err := s.View(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{"counted": counted}, "Post viewed successfully")
	}
}

func HandlePostTip(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		postIdStr := c.FormValue("post_id")
		if postIdStr == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "post_id is required")
		}

		postId, err := strconv.ParseInt(postIdStr, 10, 64)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "invalid post_id: "+err.Error())
		}

		amountStr := c.FormValue("amount")
		if amountStr == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "amount is required")
		}

		amount, err := decimal.NewFromString(amountStr)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "invalid amount: "+err.Error())
		}

		balance, err := s.Tip(c.Context(), postId, user, amount)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidAction, "failed to tip post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"balance": balance,
		}, "Post tipped successfully")
	}
}

func HandleFetchPost(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		var post *postmodel.Post

		switch {
		case filters.PostID > 0:
			post, err = s.GetPostByPublicID(filters.PostID)

		case filters.PostUUID != uuid.Nil:
			post, err = s.GetPostByID(filters.PostUUID)

		case filters.Slug != nil && *filters.Slug != "":
			post, err = s.GetPostBySlug(filters)

		default:
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusBadRequest,
				constants.ErrPostNotFound,
				"no valid identifier provided",
			)
		}

		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrPostNotFound, "post not found")
			}
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusInternalServerError,
				constants.ErrPostNotFound,
				"failed to get post: "+err.Error(),
			)
		}

		if post == nil {
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusNotFound,
				constants.ErrPostNotFound,
				"post not found",
			)
		}
		if post.PostKind == postmodel.PostKindMessage || (post.ContentableType != nil && *post.ContentableType == string(postmodel.PostKindChat)) {
			// Chat messages are only available through authenticated chat actions;
			// the public post endpoint must never bypass recipient sanitization.
			return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrPostNotFound, "post not found")
		}

		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrPostNotFound, "failed to get post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, post, "Post fetched successfully")
	}
}

func HandleGetByID(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		post, err := s.GetPostByPublicID(filters.PostID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				return utils.SendError(c, fiber.StatusNotFound, constants.ErrPostNotFound)
			}
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, post, "Post fetched successfully")
	}
}

func HandleGetBySlug(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		post, err := s.GetPostBySlug(filters)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				return utils.SendError(c, fiber.StatusNotFound, constants.ErrPostNotFound)
			}
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, post, "Post fetched successfully")
	}
}

func HandleGetByPublicID(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		post, err := s.GetPostByPublicID(filters.PostID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				return utils.SendError(c, fiber.StatusNotFound, constants.ErrPostNotFound)
			}
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get post: "+err.Error())
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, post, "Post fetched successfully")
	}
}

func HandleTimeline(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		result, err := s.GetTimeline(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get timeline: "+err.Error())
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Timeline fetched successfully")
	}
}

func HandleSearchPost(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		result, err := s.SearchPost(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to search: "+err.Error())
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Successfully")
	}
}

func HandleTimelineVibes(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		result, err := s.GetTimelineVibes(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get timeline: "+err.Error())
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Timeline fetched successfully")
	}
}

func HandleGetPostsByUser(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		posts, err := s.GetPostsByUserID(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get posts: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, posts, "Posts fetched successfully")
	}
}

func HandleGetRepliesByUser(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		posts, err := s.GetUserPostReplies(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get posts: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, posts, "Posts fetched successfully")
	}
}

func HandleGetAllMediasByUser(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		medias, nextCursor, err := s.GetUserMedias(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get medias: "+err.Error())
		}

		var nextCursorStr *string
		if nextCursor != nil {
			nextCursorStr, err = types.NewPublicIDCursor(*nextCursor)
			if err != nil {
				return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to encode cursor: "+err.Error())
			}
		}

		response := fiber.Map{
			"medias":   medias,
			"cursor":   nextCursorStr,
			"has_more": nextCursor != nil,
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, response, "Medias fetched successfully")
	}
}

func HandleGetAllLikesByUser(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		idStr := c.FormValue("id")
		if idStr == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "missing post id")
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "invalid uuid")
		}

		post, err := s.GetPostByID(id)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, post, "Post fetched successfully")
	}
}

func HandleGetTrends(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		hashtags, err := s.GetRecentHashtags(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get trends: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"trends":      hashtags,
			"last_update": time.Now(),
		}, "Trends fetched successfully")
	}
}

func HandleGetCategories(s *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		categories, err := s.GetPillarsWithClusters(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get trends: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"categories":  categories,
			"last_update": time.Now(),
		}, "Categories fetched successfully")
	}
}
