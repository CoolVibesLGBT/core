package handlers

import (
	usecases "core/application/usecases"
	"core/constants"
	"core/middleware"
	"core/models/post"
	"core/utils"

	"github.com/gofiber/fiber/v3"
)

func HandleUserCheckIn(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		if auth_user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

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

		post, err := s.CheckIn(c.Context(), uploadedFormData(formParams, images, videos), user, post.PostKindCheckIn)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedtoCheckIn)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, post, "Check in successful")
	}
}

func HandleFetchCheckIns(s *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		if auth_user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		filters, err := ParseFilters(c, auth_user)
		filters.PostKind = post.PostKindCheckIn
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)

		}

		posts, err := s.FetchCheckIns(filters)

		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to check in")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, posts, "Checks fetched successful")
	}
}
