package handlers

import (
	usecases "core/application/usecases"
	"core/constants"
	"core/middleware"
	"core/utils"
	"mime/multipart"

	"github.com/gofiber/fiber/v3"
)

type PlaceHandler struct {
	service *usecases.PlaceService
}

func NewPlaceHandler(service *usecases.PlaceService) *PlaceHandler {
	return &PlaceHandler{service: service}
}

func HandleCreatePlace(s *usecases.PlaceService) fiber.Handler {
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

		post, err := s.CreatePlace(c.Context(), formParams, files, user)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrPlaceCreateFailed, "Failed to create post: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, post, "Post created successfully")
	}
}

func HandleGetNearByPlaces(s *usecases.PlaceService) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, _ := middleware.GetAuthenticatedUser(c)

		filters, err := ParseFilters(c, authUser)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		places, cursorInfo, err := s.GetNearByPlaces(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrPlaceNotFound, "Failed to get places: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"places": places,
			"cursor": cursorInfo,
		}, "Places fetched successfully")
	}
}

func HandleGetPlaceCategories(s *usecases.PlaceService) fiber.Handler {
	return func(c fiber.Ctx) error {
		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		categories, err := s.GetPlacesCategories(filters)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrPlaceNotFound, "Failed to get categories: "+err.Error())
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{
			"categories": categories,
		}, "Categories fetched successfully")
	}
}
