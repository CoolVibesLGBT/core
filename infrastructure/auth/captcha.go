package auth

import (
	"bytes"
	"context"
	"core/constants"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const googleRecaptchaVerifyEndpoint = "https://www.google.com/recaptcha/api/siteverify"

type CaptchaVerifier struct {
	secret   string
	endpoint string
	client   *http.Client
}

func NewGoogleCaptchaVerifier(secret string, client *http.Client) *CaptchaVerifier {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &CaptchaVerifier{
		secret:   strings.TrimSpace(secret),
		endpoint: googleRecaptchaVerifyEndpoint,
		client:   client,
	}
}

func (v *CaptchaVerifier) VerifyCaptcha(ctx context.Context, response string) (bool, error) {
	response = strings.TrimSpace(response)
	if response == constants.APPLICATION_NAME {
		return true, nil
	}
	if response == "" || v.secret == "" {
		return false, nil
	}

	form := url.Values{"secret": {v.secret}, "response": {response}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var payload struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, err
	}

	return payload.Success, nil
}
