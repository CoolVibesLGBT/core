package utils

import (
	"coolvibes/constants"

	"github.com/gofiber/fiber/v2"
)

type ErrorResponse struct {
	Success bool                `json:"success"`
	Code    constants.ErrorCode `json:"code"`
	Message string              `json:"message"`
}

func SendError(c *fiber.Ctx, status int, code constants.ErrorCode) error {
	return c.Status(status).JSON(ErrorResponse{
		Success: false,
		Code:    code,
		Message: code.String(),
	})
}

func SendJSON(c *fiber.Ctx, status int, payload interface{}) error {
	return c.Status(status).JSON(payload)
}
