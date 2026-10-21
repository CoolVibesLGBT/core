package media

import (
	"context"
	upload "core/adapters/media"
	domainmedia "core/domain/media"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type RemoteImageFetcher struct {
	client *http.Client
}

func NewRemoteImageFetcher(client *http.Client) *RemoteImageFetcher {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &RemoteImageFetcher{client: client}
}

func (f *RemoteImageFetcher) FetchImage(ctx context.Context, imageURL string) (*domainmedia.UploadedFile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create image request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: status code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image body: %w", err)
	}

	ext := filepath.Ext(imageURL)
	if ext == "" {
		ext = ".jpg"
	}
	if indexQuery := strings.Index(ext, "?"); indexQuery != -1 {
		ext = ext[:indexQuery]
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(data)
	}

	return upload.FromBytes(filename, mimeType, data), nil
}
