package handlers

import (
	services "coolvibes/services/user"
	"fmt"
	"net/http"
)

type PaymentHandler struct {
	service *services.PaymentService
}

func NewPaymentHandler(service *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func HandleStripeThin(s *services.PaymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("STRIPE_THIN_EXECUTED")
		w.WriteHeader(http.StatusOK)

	}
}

func HandleStripeSnapshot(s *services.PaymentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("STRIPE_SNAPSHOT_EXECUTED")
		w.WriteHeader(http.StatusOK)
	}
}
