package routes

import (
	"compress/gzip"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	httpmiddleware "core/adapters/inbound/http/middleware"
	"core/adapters/inbound/http/router"

	"github.com/gofiber/fiber/v3"
)

type multipartZeroReader struct{}

func (multipartZeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}

func TestRequestBodyHasNoApplicationLimit(t *testing.T) {
	want := int(^uint(0)>>1) - 1
	if got := requestBodyLimit(); got != want {
		t.Fatalf("body limit = %d, want platform maximum %d", got, want)
	}

	routes := NewRouter(Dependencies{})
	config := routes.GetFiber().Config()
	if config.BodyLimit != want {
		t.Fatalf("configured body limit = %d, want %d", config.BodyLimit, want)
	}
	if !config.StreamRequestBody || !config.DisablePreParseMultipartForm {
		t.Fatalf(
			"streaming config = StreamRequestBody:%v DisablePreParseMultipartForm:%v, want both true",
			config.StreamRequestBody,
			config.DisablePreParseMultipartForm,
		)
	}
}

func TestLargeMultipartUploadIsAcceptedAndSpooledToDisk(t *testing.T) {
	spoolDir := t.TempDir()
	t.Setenv("TMPDIR", spoolDir)

	routes := NewRouter(Dependencies{})
	const action = "upload.streaming.test"
	const fileSize = int64(5 << 20) // larger than Fiber's 4 MiB default
	var spooledPath string
	routes.action.Register(action, func(c fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return err
		}
		headers := form.File["file"]
		if len(headers) != 1 || headers[0].Size != fileSize {
			return c.Status(fiber.StatusBadRequest).SendString("unexpected uploaded file")
		}

		file, err := headers[0].Open()
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		diskFile, ok := file.(*os.File)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).SendString("upload was retained in memory")
		}
		spooledPath = diskFile.Name()
		buffer := make([]byte, 1)
		if _, err := io.ReadFull(file, buffer); err != nil {
			return err
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	body, err := os.CreateTemp(t.TempDir(), "multipart-body-*")
	if err != nil {
		t.Fatalf("create multipart body: %v", err)
	}
	defer func() { _ = body.Close() }()
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "large.bin")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := io.CopyN(part, multipartZeroReader{}, fileSize); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	stat, err := body.Stat()
	if err != nil {
		t.Fatalf("stat multipart body: %v", err)
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("rewind multipart body: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/packet", body)
	request.ContentLength = stat.Size()
	request.Header.Set(fiber.HeaderContentType, writer.FormDataContentType())
	request.Header.Set("X-Action", action)
	response, err := routes.GetFiber().Test(request, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != fiber.StatusNoContent {
		encoded, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want 204: %s", response.StatusCode, encoded)
	}
	if spooledPath == "" {
		t.Fatal("multipart parser did not expose a disk-backed upload")
	}
}

func TestHandlePacketPrefersActionHeaderOverFormBody(t *testing.T) {
	actionRouter := router.NewActionRouter()
	actionRouter.Register("header.action", func(c fiber.Ctx) error {
		return c.SendString("header route")
	})
	routes := &Router{action: actionRouter}
	app := fiber.New()
	app.Post("/", routes.handlePacket)

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("action=body.action"))
	request.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationForm)
	request.Header.Set("X-Action", "header.action")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if string(body) != "header route" {
		t.Fatalf("body = %q, want header route", body)
	}
}

func TestTrustedProxySettingsRequireExplicitAllowlist(t *testing.T) {
	enabled, config := trustedProxySettings(" 10.0.0.0/8, 192.0.2.10 ,, ")
	if !enabled || len(config.Proxies) != 2 || config.Proxies[0] != "10.0.0.0/8" || config.Proxies[1] != "192.0.2.10" {
		t.Fatalf("trustedProxySettings() = %v, %#v", enabled, config.Proxies)
	}
	if enabled, config = trustedProxySettings("   "); enabled || len(config.Proxies) != 0 {
		t.Fatalf("empty trusted proxy settings = %v, %#v", enabled, config.Proxies)
	}
}

func TestClientIPIgnoresForwardedHeaderFromUntrustedPeer(t *testing.T) {
	app := fiber.New(fiber.Config{ProxyHeader: fiber.HeaderXForwardedFor})
	app.Get("/", func(c fiber.Ctx) error { return c.SendString(httpmiddleware.GetClientIP(c)) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(fiber.HeaderXForwardedFor, "203.0.113.99")
	response, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if got := string(body); got == "203.0.113.99" {
		t.Fatal("untrusted X-Forwarded-For header spoofed client IP")
	}
}

func TestPacketResponsesAreCompressed(t *testing.T) {
	routes := NewRouter(Dependencies{})
	want := strings.Repeat("nearby-response-", 2_000)
	routes.action.Register("compression.test", func(c fiber.Ctx) error {
		return c.Type("json").SendString(want)
	})

	request := httptest.NewRequest(http.MethodGet, "/packet?action=compression.test", nil)
	request.Header.Set(fiber.HeaderAcceptEncoding, "gzip")
	response, err := routes.GetFiber().Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if got := response.Header.Get(fiber.HeaderContentEncoding); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}

	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read compressed response: %v", err)
	}
	if string(body) != want {
		t.Fatalf("decompressed body length = %d, want %d", len(body), len(want))
	}
}
