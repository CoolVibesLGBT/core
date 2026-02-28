package handlers

import (
	"core/constants"
	"core/middleware"
	services "core/services/user"
	"core/utils"
	"mime/multipart"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PostHandler struct {
	service *services.PostService
}

func NewPostHandler(service *services.PostService) *PostHandler {
	return &PostHandler{service: service}
}

func HandleCreate(s *services.PostService) fiber.Handler {
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

		files := append([]*multipart.FileHeader{}, images...)
		files = append(files, videos...)

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		post, err := s.CreatePost(c.Context(), formParams, files, user)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrPostCreateFailed)
		}

		return utils.SendSuccess(c, fiber.StatusCreated, post)
	}
}

func HandleVote(s *services.PostService) fiber.Handler {
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

func HandlePostDelete(s *services.PostService) fiber.Handler {
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

func HandlePostLike(s *services.PostService) fiber.Handler {
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

func HandlePostBanana(s *services.PostService) fiber.Handler {
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

func HandlePostDislike(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}
		filters, err := ParseFilters(c, user)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		err = s.Dislike(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to dislike post: " + err.Error(),
			})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandlePostBookmark(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, user)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		err = s.Bookmark(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to bookmark post: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandlePostReport(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		postIdStr := c.FormValue("post_id")
		if postIdStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "post_id is required",
			})
		}

		postId, err := strconv.ParseInt(postIdStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid post_id: " + err.Error(),
			})
		}

		reason := c.FormValue("reason")
		description := c.FormValue("description")

		err = s.Report(c.Context(), postId, reason, description, user)
		if err != nil {
			// constants.ErrInvalidInput örnek hata, istersen kendi hata mesajını yazabilirsin
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid input",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandlePostView(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		//todo:without middleware
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, user)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		err = s.View(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandlePostTip(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		postIdStr := c.FormValue("post_id")
		if postIdStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "post_id is required",
			})
		}

		postId, err := strconv.ParseInt(postIdStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid post_id: " + err.Error(),
			})
		}

		amountStr := c.FormValue("amount")
		if amountStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "amount is required",
			})
		}

		amount, err := decimal.NewFromString(amountStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid amount: " + err.Error(),
			})
		}

		balance, err := s.Tip(c.Context(), postId, user, amount)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to tip post: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"balance": balance,
		})
	}
}

func HandleGetByID(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		post, err := s.GetPostByPublicID(filters.PostID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get post: " + err.Error(),
			})
		}

		return c.JSON(post)
	}
}

func HandleGetByPublicID(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		post, err := s.GetPostByPublicID(filters.PostID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(post)
	}
}

func HandleTimeline(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		result, err := s.GetTimeline(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get timeline: " + err.Error(),
			})
		}
		return c.JSON(result)
	}
}

func HandleTimelineVibes(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		result, err := s.GetTimelineVibes(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get timeline: " + err.Error(),
			})
		}
		return c.JSON(result)
	}
}

func HandleGetPostsByUser(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		posts, err := s.GetPostsByUserID(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get posts: " + err.Error(),
			})
		}

		return c.JSON(posts)
	}
}

func HandleGetRepliesByUser(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		posts, err := s.GetUserPostReplies(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get posts: " + err.Error(),
			})
		}

		return c.JSON(posts)
	}
}

func HandleGetAllMediasByUser(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		medias, nextCursor, err := s.GetUserMedias(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get medias: " + err.Error(),
			})
		}

		var nextCursorStr string
		if nextCursor != nil {
			nextCursorStr = strconv.FormatInt(*nextCursor, 10)
		} else {
			nextCursorStr = "0"
		}

		response := fiber.Map{
			"medias":      medias,
			"next_cursor": nextCursorStr,
			"has_more":    nextCursor != nil,
		}

		return c.JSON(response)
	}
}

func HandleGetAllLikesByUser(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		idStr := c.FormValue("id")
		if idStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "missing post id",
			})
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid uuid",
			})
		}

		post, err := s.GetPostByID(id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get post: " + err.Error(),
			})
		}

		return c.JSON(post)
	}
}

func HandleGetTrends(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		hashtags, err := s.GetRecentHashtags(filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get trends: " + err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success":     true,
			"trends":      hashtags,
			"last_update": time.Now(),
		})
	}
}

func HandleGetCategories(s *services.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {

		//

		categories, err := s.GetPillarsWithClusters(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get trends: " + err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"success":     true,
			"categories":  categories,
			"last_update": time.Now(),
		})
	}
}
