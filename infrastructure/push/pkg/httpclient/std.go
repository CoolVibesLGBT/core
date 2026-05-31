package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type StdHttpClient struct {
	client *http.Client
}

func StdHttp(client *http.Client) *StdHttpClient {
	if client != nil {
		return &StdHttpClient{client: client}
	}

	return &StdHttpClient{
		client: &http.Client{},
	}
}

func (f *StdHttpClient) RequestDelivery(endpoint string, headers *Headers, body *bytes.Buffer) (int, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Length", strconv.Itoa(body.Len()))
	req.Header.Set("Authorization", headers.Authorization)
	req.Header.Set("TTL", strconv.FormatInt(int64(headers.TTL/time.Second), 10))

	if headers.Urgency != "" {
		req.Header.Set("Urgency", headers.Urgency)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}

	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(resp.Body)

	return resp.StatusCode, nil
}
