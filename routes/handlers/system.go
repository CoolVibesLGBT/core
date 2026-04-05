package handlers

import (
	"core/constants"
	"core/helpers"
	"core/middleware"
	"core/models"
	"core/models/payment"
	eventkinds "core/models/post/payloads"
	services "core/services/user"
	"core/utils"
	"encoding/json"
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
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

type InitialData struct {
	VapidPubicKey string                                `json:"vapid_public_key"`
	Preferences   models.PreferencesData                `json:"preferences"`
	EventKinds    []eventkinds.EventKind                `json:"event_kinds"`
	ReportKinds   []models.ReportKind                   `json:"report_kinds"`
	Countries     map[string]CountryResponse            `json:"countries"`
	Languages     map[string]constants.LanguageResponse `json:"languages"`
	CheckInTags   []models.CheckInTag                   `json:"checkin_tag_types"`
	Status        string                                `json:"status"`
}

type SystemHandler struct {
	service *services.NotificationsService
}

func NewSystemHandler(service *services.NotificationsService) *SystemHandler {
	return &SystemHandler{service: service}
}

func HandleInitialSync(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		var preferences models.PreferencesData
		if err := db.Model(&models.Preferences{}).Select("data").First(&preferences).Error; err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrPreferencesFetchFailed)
		}

		var eventKinds []eventkinds.EventKind
		if err := db.Model(&eventkinds.EventKind{}).Order("display_order ASC").Find(&eventKinds).Error; err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrEventKindsFetchFailed)
		}

		var reportKinds []models.ReportKind
		if err := db.Model(&models.ReportKind{}).Order("display_order ASC").Find(&reportKinds).Error; err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrReportKindsFetchFailed)
		}

		countries := map[string]CountryResponse{
			"TR": {Code: "TR", Name: "Turkey"},
			"US": {Code: "US", Name: "United States"},
		}

		key, err := helpers.CreateVapidKeys(db)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrVapidKeyGenerationFailed)
		}
		initialData := InitialData{
			VapidPubicKey: key.PublicKey,
			Preferences:   preferences,
			Countries:     countries,
			Languages:     constants.Languages,
			EventKinds:    eventKinds,
			ReportKinds:   reportKinds,
			CheckInTags:   models.GetAllCheckInTagTypes(),
		}

		return utils.SendSuccess(c, fiber.StatusOK, initialData)
	}
}

func HandleVapidGetKey(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		key, err := helpers.CreateVapidKeys(db)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrVapidKeyGenerationFailed)
		}

		resp := struct {
			PublicKey string `json:"key"`
		}{
			PublicKey: key.PublicKey,
		}

		return utils.SendSuccess(c, fiber.StatusOK, resp)
	}
}

func HandleVapidSubscribe(db *gorm.DB) fiber.Handler {
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

		var user models.User
		if err := db.First(&user, "id = ?", authUser.ID).Error; err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrUserNotFound)
		}

		items, err := helpers.DecodeJSONItems(user.Subscriptions)
		if err != nil {
			items = nil
		}
		exists := false
		for _, item := range items {
			var sub models.Subscription
			if err := json.Unmarshal(item, &sub); err != nil {
				continue
			}
			if sub.Endpoint == newSub.Endpoint {
				exists = true
				break
			}
		}
		if !exists {
			rawNewSub, err := json.Marshal(newSub)
			if err != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrVapidSubscriptionFailed)
			}

			items = append(items, rawNewSub)
		}

		subsJson, err := json.Marshal(items)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrVapidSubscriptionFailed)
		}
		if err := db.Model(&models.User{}).
			Where("id = ?", authUser.ID).
			Update("subscriptions", datatypes.JSON(subsJson)).Error; err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrVapidSubscriptionFailed)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"message": "Subscription saved",
		})
	}
}

func HandleGetNotifications(s *services.NotificationsService) fiber.Handler {
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

func HandleFetchPaymentMethods(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		var pm payment.PaymentMethod
		if err := db.First(&pm).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrPaymentMethodNotFound, "Payment method not found")
			}

			log.Printf("db error fetching payment method: %v", err)
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrPaymentMethodFetchFailed, "Payment method fetch failed")
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, pm, "Payment method fetched successfully")
	}
}
