package handlers

import (
	"coolvibes/constants"
	"coolvibes/helpers"
	"coolvibes/middleware"
	"coolvibes/models"
	"coolvibes/models/payment"
	eventkinds "coolvibes/models/post/payloads"
	services "coolvibes/services/user"
	"coolvibes/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

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
	Status        string                                `json:"status"`
}

type SystemHandler struct {
	service *services.NotificationsService
}

func NewSystemHandler(service *services.NotificationsService) *SystemHandler {
	return &SystemHandler{service: service}
}

func HandleInitialSync(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Tüm fantezileri çek
		var preferences models.PreferencesData
		if err := db.Model(&models.Preferences{}).Select("data").First(&preferences).Error; err != nil {
			http.Error(w, "Failed to fetch preferences data", http.StatusInternalServerError)
			return
		}

		var eventKinds []eventkinds.EventKind
		if err := db.Model(&eventkinds.EventKind{}).Order("display_order ASC").Find(&eventKinds).Error; err != nil {
			http.Error(w, "Failed to fetch preferences data", http.StatusInternalServerError)
			return
		}

		var reportKinds []models.ReportKind
		if err := db.Model(&models.ReportKind{}).Order("display_order ASC").Find(&reportKinds).Error; err != nil {
			http.Error(w, "Failed to fetch preferences data", http.StatusInternalServerError)
			return
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
			http.Error(w, "Failed to get VAPID key", http.StatusInternalServerError)
			return
		}
		// 5. InitialData hazırla
		initialData := InitialData{
			VapidPubicKey: key.PublicKey,
			Preferences:   preferences,
			Countries:     countries,
			Languages:     constants.Languages,
			EventKinds:    eventKinds,
			ReportKinds:   reportKinds,
			Status:        "ok",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(initialData)
	}
}

func HandleVapidGetKey(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		key, err := helpers.CreateVapidKeys(db)
		if err != nil {
			http.Error(w, "Failed to get VAPID key", http.StatusInternalServerError)
			return
		}

		resp := struct {
			PublicKey string `json:"key"`
		}{
			PublicKey: key.PublicKey,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func HandleVapidSubscribe(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		// Gelen subscription'u json olarak oku
		err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, "Failed to parse multipart form")
			return
		}

		// Form field içindeki JSON stringi al
		subscriptionJson := r.FormValue("subscriptions")
		if subscriptionJson == "" {
			utils.SendError(w, http.StatusBadRequest, "subscriptions field is required")
			return
		}

		fmt.Println("GELEN DATA", subscriptionJson)

		var newSub models.Subscription
		if err := json.Unmarshal([]byte(subscriptionJson), &newSub); err != nil {
			utils.SendError(w, http.StatusBadRequest, "Invalid subscription JSON")
			return
		}

		// Kullanıcıyı veritabanından çek
		var user models.User
		if err := db.First(&user, "id = ?", auth_user.ID).Error; err != nil {
			utils.SendError(w, http.StatusInternalServerError, "User not found")
			return
		}

		fmt.Println("AUTH_USER", user.UserName)

		// Var olan subscriptionları çıkar
		var subscriptions []models.Subscription
		if len(user.Subscriptions) > 0 {
			if err := json.Unmarshal(user.Subscriptions, &subscriptions); err != nil {
				// Eğer hata varsa, boş liste olarak başlatabiliriz
				subscriptions = []models.Subscription{}
			}
		}

		// Yeni subscription zaten varsa ekleme (unique endpoint kontrolü)
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

		// Tekrar json'a çevir
		subsJson, err := json.Marshal(subscriptions)
		if err != nil {
			utils.SendError(w, http.StatusInternalServerError, "Could not marshal subscriptions")
			return
		}

		// Güncelle
		user.Subscriptions = subsJson

		if err := db.Save(&user).Error; err != nil {
			utils.SendError(w, http.StatusInternalServerError, "Failed to update subscriptions")
			return
		}

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Subscription saved",
		})
	}
}

func HandleGetNotifications(s *services.NotificationsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			utils.SendError(w, http.StatusUnauthorized, constants.ErrUnauthorized)
			return
		}

		// Gelen subscription'u json olarak oku
		err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
		if err != nil {
			utils.SendError(w, http.StatusBadRequest, "Failed to parse multipart form")
			return
		}

		notifications, err := s.FetchNotifications(auth_user.ID, 1)

		fmt.Println("Notifications", auth_user.ID)

		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"success":       true,
			"notifications": notifications,
		})
	}
}

func HandleFetchPaymentMethods(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var pm payment.PaymentMethod
		if err := db.First(&pm).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "payment method not found",
				})
				return
			}

			// Beklenmeyen DB hatası
			log.Printf("db error fetching payment method: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "internal server error",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(pm); err != nil {
			log.Printf("encode error: %v", err)
		}
	}
}
