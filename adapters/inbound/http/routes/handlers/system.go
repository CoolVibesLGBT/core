package handlers

import (
	"core/adapters/inbound/http/middleware"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/constants"
	"core/models"
	"core/utils"
	"encoding/json"
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"
)

// TranslationMap map[string]Translation

// CountryResponse tek ülke objesi
type CountryResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type OrientationData struct {
	ID           string            `json:"id"`
	Key          string            `json:"key"`
	Translations map[string]string `json:"translations"`
}

type GroupedAttributes struct {
	Category   string          `json:"category"`
	Attributes json.RawMessage `json:"attributes"` // JSON array olarak döner
}

type SystemHandler struct {
	service *usecases.NotificationsService
}

func NewSystemHandler(service *usecases.NotificationsService) *SystemHandler {
	return &SystemHandler{service: service}
}

func HandleInitialSync(service *usecases.SystemService) fiber.Handler {
	return func(c fiber.Ctx) error {
		initialData, err := service.InitialSync(c.Context())
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrPreferencesFetchFailed)
		}

		return utils.SendSuccess(c, fiber.StatusOK, initialData)
	}
}

func HandleVapidGetKey(service *usecases.SystemService) fiber.Handler {
	return func(c fiber.Ctx) error {
		key, err := service.VapidPublicKey(c.Context())
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrVapidKeyGenerationFailed)
		}

		resp := struct {
			PublicKey string `json:"key"`
		}{
			PublicKey: key,
		}

		return utils.SendSuccess(c, fiber.StatusOK, resp)
	}
}

func HandleVapidSubscribe(service *usecases.SystemService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		// Multipart formu oku (dosyalar ve alanlar)
		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		// subscription JSON stringini al
		subscriptionJsonArr := form.Value["subscriptions"]
		if len(subscriptionJsonArr) == 0 || subscriptionJsonArr[0] == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}
		subscriptionJson := subscriptionJsonArr[0]

		var newSub models.Subscription
		if err := json.Unmarshal([]byte(subscriptionJson), &newSub); err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		if err := service.SaveVapidSubscription(c.Context(), authUser.ID, newSub); err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrVapidSubscriptionFailed)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"message": "Subscription saved",
		})
	}
}

func HandleGetNotifications(s *usecases.NotificationsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		// Multipart form parse etmeye gerek yoksa atla.
		// Eğer parse etmek istersen:
		// form, err := c.MultipartForm()
		// if err != nil { ... }

		notifications, err := s.FetchNotifications(authUser.ID, 1)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrNotificationsFetchFailed)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"notifications": notifications,
		})
	}
}

func HandleFetchPaymentMethods(service *usecases.SystemService) fiber.Handler {
	return func(c fiber.Ctx) error {
		pm, err := service.PaymentMethod(c.Context())
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrPaymentMethodNotFound, "Payment method not found")
			}

			log.Printf("db error fetching payment method: %v", err)
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrPaymentMethodFetchFailed, "Payment method fetch failed")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, pm, "Payment method fetched successfully")
	}
}
