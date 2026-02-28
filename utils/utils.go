package utils

import (
	"core/constants"

	"github.com/gofiber/fiber/v3"
)

type ErrorResponse struct {
	Success bool                `json:"success"`
	Code    constants.ErrorCode `json:"code"`
	Message string              `json:"message"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type SuccessResponseWithMessage struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func SendError(c fiber.Ctx, status int, code constants.ErrorCode) error {
	return c.Status(status).JSON(ErrorResponse{
		Success: false,
		Code:    code,
		Message: code.String(),
	})
}

func SendErrorWithMessage(c fiber.Ctx, status int, code constants.ErrorCode, msg string) error {
	return c.Status(status).JSON(ErrorResponse{
		Success: false,
		Code:    code,
		Message: msg,
	})
}

func SendSuccess(c fiber.Ctx, status int, payload interface{}) error {
	return c.Status(status).JSON(SuccessResponse{
		Success: true,
		Data:    payload,
	})
}

func SendSuccessWithMessage(c fiber.Ctx, status int, payload interface{}, msg string) error {
	return c.Status(status).JSON(SuccessResponseWithMessage{
		Success: true,
		Data:    payload,
		Message: msg,
	})
}

func SendJSON(c fiber.Ctx, status int, payload interface{}) error {
	return c.Status(status).JSON(payload)
}
