package handlers

import (
	"fmt"

	usecases "core/application/usecases"

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
		fmt.Println("STRIPE_THIN_EXECUTED")
		return c.SendStatus(fiber.StatusOK)
	}
}

// /stripe/snapshot
func HandleStripeSnapshot(s *usecases.PaymentService) fiber.Handler {
	return func(c fiber.Ctx) error {
		fmt.Println("STRIPE_SNAPSHOT_EXECUTED")
		return c.SendStatus(fiber.StatusOK)
	}
}
