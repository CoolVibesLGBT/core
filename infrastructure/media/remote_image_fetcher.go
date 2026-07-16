package media

import (
	"bytes"
	"context"
	"core/application/ports"
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

type memoryUploadedFile struct {
	filename    string
	size        int64
	contentType string
	data        []byte
}

func (f memoryUploadedFile) Filename() string {
	return f.filename
}

func (f memoryUploadedFile) Size() int64 {
	return f.size
}

func (f memoryUploadedFile) ContentType() string {
	return f.contentType
}

func (f memoryUploadedFile) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.data)), nil
}

func (f *RemoteImageFetcher) FetchImage(ctx context.Context, imageURL string) (ports.UploadedFile, error) {
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

	return memoryUploadedFile{
		filename:    filename,
		size:        int64(len(data)),
		contentType: mimeType,
		data:        data,
	}, nil
}
