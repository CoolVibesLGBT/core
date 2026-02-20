package handlers

import (
	"core/middleware"
	services "core/services/user"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"
)

type PlaceHandler struct {
	service *services.PlaceService
}

func NewPlaceHandler(service *services.PlaceService) *PlaceHandler {
	return &PlaceHandler{service: service}
}

func HandleCreatePlace(s *services.PlaceService) fiber.Handler {
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

		post, err := s.CreatePlace(c.Context(), formParams, files, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create post: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(post)
	}
}

func HandleGetNearByPlaces(s *services.PlaceService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, _ := middleware.GetAuthenticatedUser(c)

		filters, err := ParseFilters(c, authUser)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		places, cursorInfo, err := s.GetNearByPlaces(filters)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": err == nil,
			"error":   err,
			"places":  places,
			"cursor":  cursorInfo,
		})
	}
}
