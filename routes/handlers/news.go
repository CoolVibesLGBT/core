package handlers

import (
	"core/constants"
	"core/middleware"
	services "core/services/user"
	"core/utils"
	"mime/multipart"

	"github.com/gofiber/fiber/v3"
)

type NewsHandler struct {
	service *services.NewsService
}

func NewNewsHandler(service *services.NewsService) *NewsHandler {
	return &NewsHandler{service: service}
}

func HandleCreateNews(s *services.NewsService) fiber.Handler {
	return func(c fiber.Ctx) error {

		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Could not parse multipart form: "+err.Error())
		}

		formParams := form.Value
		images := form.File["images[]"]
		videos := form.File["videos[]"]

		files := append([]*multipart.FileHeader{}, images...)
		files = append(files, videos...)

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendErrorWithMessage(c, fiber.StatusUnauthorized, constants.ErrUnauthorized, "User not authenticated")
		}

		post, err := s.CreateNews(c.Context(), formParams, files, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrNewsCreateFailed, "Failed to create post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, post, "Post created successfully")
	}
}

func HandleFetchNews(s *services.NewsService) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, _ := middleware.GetAuthenticatedUser(c)

		filters, err := ParseFilters(c, authUser)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid filter: "+err.Error())
		}

		postResult, err := s.GetNews(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrNewsCreateFailed, "Failed to create post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"news":   postResult.Posts,
			"cursor": postResult.Cursor,
		}, "News fetched successfully")
	}
}
