package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	"core/models"
	eventkinds "core/models/post/payloads"
	"encoding/json"

	"github.com/google/uuid"
)

type CountryResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
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

type SystemService struct {
	repo ports.SystemRepository
}

func NewSystemService(repo ports.SystemRepository) *SystemService {
	return &SystemService{repo: repo}
}

func (s *SystemService) InitialSync(ctx context.Context) (InitialData, error) {
	preferences, err := s.repo.GetPreferences(ctx)
	if err != nil {
		return InitialData{}, err
	}

	eventKinds, err := s.repo.GetEventKinds(ctx)
	if err != nil {
		return InitialData{}, err
	}

	reportKinds, err := s.repo.GetReportKinds(ctx)
	if err != nil {
		return InitialData{}, err
	}

	vapidPublicKey, err := s.repo.GetVapidPublicKey(ctx)
	if err != nil {
		return InitialData{}, err
	}

	return InitialData{
		VapidPubicKey: vapidPublicKey,
		Preferences:   preferences,
		Countries: map[string]CountryResponse{
			"TR": {Code: "TR", Name: "Turkey"},
			"US": {Code: "US", Name: "United States"},
		},
		Languages:   constants.Languages,
		EventKinds:  eventKinds,
		ReportKinds: reportKinds,
		CheckInTags: models.GetAllCheckInTagTypes(),
	}, nil
}

func (s *SystemService) VapidPublicKey(ctx context.Context) (string, error) {
	return s.repo.GetVapidPublicKey(ctx)
}

func (s *SystemService) SaveVapidSubscription(ctx context.Context, userID uuid.UUID, subscription models.Subscription) error {
	return s.repo.SaveVapidSubscription(ctx, userID, subscription)
}

func (s *SystemService) PaymentMethod(ctx context.Context) (*types.PublicPaymentMethod, error) {
	method, err := s.repo.GetPaymentMethod(ctx)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return nil, ports.ErrNotFound
	}
	return &types.PublicPaymentMethod{
		Kind:               string(method.DefaultPaymentKind),
		IBANDetails:        clonePaymentJSON(method.IBANDetails),
		IsIBANEnabled:      method.IsIBANEnabled,
		CryptoDetails:      clonePaymentJSON(method.CryptoDetails),
		IsCryptoEnabled:    method.IsCryptoEnabled,
		GooglePayDetails:   clonePaymentJSON(method.GooglePayDetails),
		IsGooglePayEnabled: method.IsGooglePayEnabled,
		Packages:           clonePaymentJSON(method.Packages),
	}, nil
}

func clonePaymentJSON(value []byte) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}
