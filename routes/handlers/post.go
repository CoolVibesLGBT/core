package handlers

import (
	"core/middleware"
	services "core/services/user"
	"fmt"
	"math"
	"mime/multipart"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
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
	return func(c *fiber.Ctx) error {

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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		post, err := s.CreatePost(c.Context(), formParams, files, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create post: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(post)
	}
}

func HandleVote(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		choiceIdStr := c.FormValue("choice_id")
		weightStr := c.FormValue("weight")
		rankStr := c.FormValue("rank")

		choiceId, err := uuid.Parse(choiceIdStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid_choice_id",
			})
		}

		weight := 1
		if weightStr != "" {
			weight, err = strconv.Atoi(weightStr)
			if err != nil || weight <= 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid_weight",
				})
			}
		}

		rank := 0
		if rankStr != "" {
			rank, err = strconv.Atoi(rankStr)
			if err != nil || rank < 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid_rank",
				})
			}
		}

		err = s.Vote(c.Context(), choiceId, weight, rank, user.ID)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": err == nil,
			"error":   err,
		})
	}
}

func HandlePostDelete(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
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

		if err := s.Delete(c.Context(), postId, user); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to delete post: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandlePostLike(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// authenticated user
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// read form value
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

		// do like
		err = s.Like(c.Context(), postId, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to like post: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandlePostBanana(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// authenticated user
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// form param: post_id
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

		// call service
		err = s.Banana(c.Context(), postId, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to banana post: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandlePostDislike(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// get authenticated user
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		// read post_id from form
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

		// call service
		err = s.Dislike(c.Context(), postId, user)
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
	return func(c *fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
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

		err = s.Bookmark(c.Context(), postId, user)
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
	return func(c *fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
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
	return func(c *fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		fmt.Println("CODER", user.ID)

		// Eğer response dönmek istersen örnek:
		return c.SendStatus(fiber.StatusOK)
	}
}

func HandlePostTip(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
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
	return func(c *fiber.Ctx) error {
		idStr := c.Query("id")
		if idStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "missing post id",
			})
		}

		postId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid post id",
			})
		}

		post, err := s.GetPostByPublicID(postId)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get post: " + err.Error(),
			})
		}

		return c.JSON(post)
	}
}

func HandleGetByPublicID(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		idStr := c.Query("id")
		if idStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "missing post id",
			})
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid id",
			})
		}

		// Örnek debug print (fmt.Printf kullanımı düzeltildi)
		fmt.Printf("%v\n", id)

		// Burada 12 sabit değeri kullanılmaz, muhtemelen id kullanılacak
		post, err := s.GetPostByPublicID(id)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get post: " + err.Error(),
			})
		}

		return c.JSON(post)
	}
}

func HandleTimeline(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {
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
	return func(c *fiber.Ctx) error {

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
	return func(c *fiber.Ctx) error {
		userIdStr := c.FormValue("user_id")
		if userIdStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id is required",
			})
		}

		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			// utils.SendError yerine Fiber JSON
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid user_id",
			})
		}

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		posts, err := s.GetPostsByUserID(userId, filters)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get posts: " + err.Error(),
			})
		}

		return c.JSON(posts)
	}
}

func HandleGetRepliesByUser(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIdStr := c.FormValue("user_id")
		if userIdStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id is required",
			})
		}

		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid user_id",
			})
		}

		limit := 10 // default değer
		if limitStr := c.FormValue("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		cursor := int64(math.MaxInt64)
		if cursorStr := c.FormValue("cursor"); cursorStr != "" {
			val, err := strconv.ParseInt(cursorStr, 10, 64)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid cursor",
				})
			}
			cursor = val
		}

		posts, err := s.GetUserPostReplies(userId, limit, &cursor)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to get posts: " + err.Error(),
			})
		}

		return c.JSON(posts)
	}
}

func HandleGetAllMediasByUser(s *services.PostService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIdStr := c.FormValue("user_id")
		if userIdStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "user_id is required",
			})
		}

		userId, err := strconv.ParseInt(userIdStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid user_id",
			})
		}

		limit := 10 // default değer
		if limitStr := c.FormValue("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				limit = parsedLimit
			}
		}

		cursor := int64(math.MaxInt64)
		if cursorStr := c.FormValue("cursor"); cursorStr != "" {
			val, err := strconv.ParseInt(cursorStr, 10, 64)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid cursor",
				})
			}
			cursor = val
		}

		medias, nextCursor, err := s.GetUserMedias(userId, limit, &cursor)
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
	return func(c *fiber.Ctx) error {
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
	return func(c *fiber.Ctx) error {

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
