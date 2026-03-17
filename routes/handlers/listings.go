package handlers

import (
	"core/constants"
	"core/middleware"
	services "core/services/user"
	"core/utils"
	"mime/multipart"

	"github.com/gofiber/fiber/v3"
)

type ClassifiedsHandler struct {
	service *services.ClassifiedService
}

func NewClassifiedsHandler(service *services.ClassifiedService) *ClassifiedsHandler {
	return &ClassifiedsHandler{service: service}
}

func HandleCreateClassified(s *services.ClassifiedService) fiber.Handler {
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

		post, err := s.CreateClassified(c.Context(), formParams, files, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrNewsCreateFailed, "Failed to create post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, post, "Post created successfully")
	}
}

func HandleFetchJobOffers(s *services.ClassifiedService) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, _ := middleware.GetAuthenticatedUser(c)

		filters, err := ParseFilters(c, authUser)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid filter: "+err.Error())
		}

		postResult, err := s.GetJobOffers(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrNewsCreateFailed, "Failed to create post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"posts":  postResult.Posts,
			"cursor": postResult.Cursor,
		}, "Jobs fetched successfully")
	}
}

func HandleFetchJobSearches(s *services.ClassifiedService) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, _ := middleware.GetAuthenticatedUser(c)

		filters, err := ParseFilters(c, authUser)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "Invalid filter: "+err.Error())
		}

		postResult, err := s.GetJobSearches(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrNewsCreateFailed, "Failed to create post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"posts":  postResult.Posts,
			"cursor": postResult.Cursor,
		}, "Job searches fetched successfully")
	}
}

func HandleGetClassified(s *services.ClassifiedService) fiber.Handler {
	return func(c fiber.Ctx) error {

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		post, err := s.GetClassified(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrInvalidInput, "failed to get post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, post, "Classified fetched successfully")
	}
}
