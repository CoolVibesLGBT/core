package telegrammedia

import (
	"context"
	"core/application/ports"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type Client struct{ http *http.Client }

func NewClient() *Client {
	socket := os.Getenv("TELEGRAM_MEDIA_SOCKET")
	if socket == "" {
		socket = filepath.Join("tools", "telegram-media", ".socket")
	}
	return &Client{http: &http.Client{Timeout: 55 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}}
}

var publicGIF = regexp.MustCompile(`^/static/telegram/gifs/[a-f0-9]{64}\.(mp4|gif)$`)

func (c *Client) PrepareGIF(ctx context.Context, filename string) (retErr error) {
	if !publicGIF.MatchString("/static/telegram/gifs/" + filename) {
		return ports.ErrChatGIFUnavailable
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://telegram-media/prepare/"+filename, nil)
	if err != nil {
		return ports.ErrChatGIFUnavailable
	}
	response, err := c.http.Do(request)
	if err != nil {
		return ports.ErrChatGIFUnavailable
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			retErr = errors.Join(retErr, ports.ErrChatGIFUnavailable, err)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return ports.ErrChatGIFUnavailable
	}
	return nil
}
func (c *Client) SearchGIFs(ctx context.Context, query, offset string) (_ *ports.ChatGIFPage, retErr error) {
	if len(query) > 200 || len(offset) > 1024 {
		return nil, ports.ErrChatGIFUnavailable
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://telegram-media/gifs?"+url.Values{"query": {query}, "offset": {offset}}.Encode(), nil)
	if err != nil {
		return nil, ports.ErrChatGIFUnavailable
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, ports.ErrChatGIFUnavailable
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			retErr = errors.Join(retErr, ports.ErrChatGIFUnavailable, err)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return nil, ports.ErrChatGIFUnavailable
	}
	var page ports.ChatGIFPage
	if json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&page) != nil || len(page.Documents) > 100 || len(page.NextOffset) > 1024 {
		return nil, ports.ErrChatGIFUnavailable
	}
	for _, document := range page.Documents {
		if document.Kind != "gif" || !publicGIF.MatchString(document.URL) || len(document.ID) > 128 || document.Width < 0 || document.Height < 0 || document.Width > 4096 || document.Height > 4096 {
			return nil, ports.ErrChatGIFUnavailable
		}
	}
	return &page, nil
}
