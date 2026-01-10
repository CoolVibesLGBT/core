package handlers

import (
	"coolvibes/constants"
	"coolvibes/helpers"
	"coolvibes/middleware"
	"coolvibes/models"
	"coolvibes/models/payment"
	eventkinds "coolvibes/models/post/payloads"
	services "coolvibes/services/user"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
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

// InitialData dönecek ana struct
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
	return func(c *fiber.Ctx) error {
		// 1. Tüm fantezileri çek
		var preferences models.PreferencesData
		if err := db.Model(&models.Preferences{}).Select("data").First(&preferences).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch preferences data",
			})
		}

		var eventKinds []eventkinds.EventKind
		if err := db.Model(&eventkinds.EventKind{}).Order("display_order ASC").Find(&eventKinds).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch event kinds",
			})
		}

		var reportKinds []models.ReportKind
		if err := db.Model(&models.ReportKind{}).Order("display_order ASC").Find(&reportKinds).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch report kinds",
			})
		}

		// 3. Ülkeleri çek
		// Örneğin countries tablosu veya sabit listeden
		countries := map[string]CountryResponse{
			"TR": {Code: "TR", Name: "Turkey"},
			"US": {Code: "US", Name: "United States"},
			// dilediğin kadar ekle
		}

		key, err := helpers.CreateVapidKeys(db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get VAPID key",
			})
		}
		// 5. InitialData hazırla
		initialData := InitialData{
			VapidPubicKey: key.PublicKey,
			Preferences:   preferences,
			Countries:     countries,
			Languages:     constants.Languages,
			EventKinds:    eventKinds,
			ReportKinds:   reportKinds,
			CheckInTags:   models.GetAllCheckInTagTypes(),
			Status:        "ok",
		}

		return c.Status(fiber.StatusOK).JSON(initialData)
	}
}

func HandleVapidGetKey(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key, err := helpers.CreateVapidKeys(db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get VAPID key",
			})
		}

		resp := struct {
			PublicKey string `json:"key"`
		}{
			PublicKey: key.PublicKey,
		}

		return c.JSON(resp)
	}
}

func HandleVapidSubscribe(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": constants.ErrUnauthorized,
			})
		}

		// Multipart formu oku (dosyalar ve alanlar)
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Failed to parse multipart form: " + err.Error(),
			})
		}

		// subscription JSON stringini al
		subscriptionJsonArr := form.Value["subscriptions"]
		if len(subscriptionJsonArr) == 0 || subscriptionJsonArr[0] == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "subscriptions field is required",
			})
		}
		subscriptionJson := subscriptionJsonArr[0]

		var newSub models.Subscription
		if err := json.Unmarshal([]byte(subscriptionJson), &newSub); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid subscription JSON",
			})
		}

		// Kullanıcıyı veritabanından çek
		var user models.User
		if err := db.First(&user, "id = ?", authUser.ID).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		// Mevcut subscriptionları ayıkla
		var subscriptions []models.Subscription
		if len(user.Subscriptions) > 0 {
			if err := json.Unmarshal(user.Subscriptions, &subscriptions); err != nil {
				subscriptions = []models.Subscription{}
			}
		}

		// Yeni abonelik mevcutsa ekleme
		exists := false
		for _, sub := range subscriptions {
			if sub.Endpoint == newSub.Endpoint {
				exists = true
				break
			}
		}
		if !exists {
			subscriptions = append(subscriptions, newSub)
		}

		subsJson, err := json.Marshal(subscriptions)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not marshal subscriptions",
			})
		}
		user.Subscriptions = subsJson

		if err := db.Save(&user).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update subscriptions",
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": true,
			"message": "Subscription saved",
		})
	}
}

func HandleGetNotifications(s *services.NotificationsService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": constants.ErrUnauthorized,
			})
		}

		// Multipart form parse etmeye gerek yoksa atla.
		// Eğer parse etmek istersen:
		// form, err := c.MultipartForm()
		// if err != nil { ... }

		notifications, err := s.FetchNotifications(authUser.ID, 1)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch notifications",
			})
		}

		fmt.Println("Notifications", authUser.ID)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success":       true,
			"notifications": notifications,
		})
	}
}

func HandleFetchPaymentMethods(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var pm payment.PaymentMethod
		if err := db.First(&pm).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "payment method not found",
				})
			}

			// Beklenmeyen DB hatası
			log.Printf("db error fetching payment method: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal server error",
			})
		}

		return c.Status(fiber.StatusOK).JSON(pm)
	}
}
