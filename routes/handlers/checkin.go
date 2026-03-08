package handlers

import (
	"core/constants"
	"core/middleware"
	services "core/services/user"
	"core/utils"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func HandleUserCheckIn(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		if auth_user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		checkInKind := c.Params("check_in")

		fmt.Println("CheckIn", checkInKind)

		err := s.CheckIn(c.Context())

		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to check in")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{}, "Check in successful")
	}
}

func HandleFetchCheckIns(s *services.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {

		auth_user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		if auth_user == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		checkInKind := c.Params("check_in")

		fmt.Println("CheckIn", checkInKind)

		err := s.CheckIn(c.Context())

		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to check in")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, fiber.Map{}, "Check in successful")
	}
}
