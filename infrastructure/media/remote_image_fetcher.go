package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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

func (f *RemoteImageFetcher) FetchImage(ctx context.Context, imageURL string) (*multipart.FileHeader, error) {
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

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	header.Set("Content-Type", mimeType)

	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(data) + 1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read form: %w", err)
	}

	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		return nil, fmt.Errorf("file header is empty")
	}
	return fileHeaders[0], nil
}
