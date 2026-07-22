package handlers

import (
	usecases "core/application/usecases"
	"core/constants"
	"core/utils"

	"github.com/gofiber/fiber/v3"
)

type PaymentHandler struct {
	service *usecases.PaymentService
}

func NewPaymentHandler(service *usecases.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

// /stripe/thin
func HandleStripeThin(s *usecases.PaymentService) fiber.Handler {
	return func(c fiber.Ctx) error {
		return utils.SendError(c, fiber.StatusNotImplemented, constants.ErrMethodNotImplemented)
	}
}

// /stripe/snapshot
func HandleStripeSnapshot(s *usecases.PaymentService) fiber.Handler {
	return func(c fiber.Ctx) error {
		return utils.SendError(c, fiber.StatusNotImplemented, constants.ErrMethodNotImplemented)
	}
}
