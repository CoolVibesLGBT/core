package handlers

import (
	"fmt"

	services "coolvibes/services/user"

	"github.com/gofiber/fiber/v2"
)

type PaymentHandler struct {
	service *services.PaymentService
}

func NewPaymentHandler(service *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

// /stripe/thin
func HandleStripeThin(s *services.PaymentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Println("STRIPE_THIN_EXECUTED")
		return c.SendStatus(fiber.StatusOK)
	}
}

// /stripe/snapshot
func HandleStripeSnapshot(s *services.PaymentService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fmt.Println("STRIPE_SNAPSHOT_EXECUTED")
		return c.SendStatus(fiber.StatusOK)
	}
}
