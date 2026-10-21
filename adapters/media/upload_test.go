package media

import (
	"bytes"
	domainmedia "core/domain/media"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
)

func TestFromPathsStreamsLargeFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large-image.jpg")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	// Truncate creates a sparse fixture, so this verifies 64-bit upload
	// metadata and lazy opening without allocating or reading a multi-GiB body.
	const size = int64(5)<<30 + 123
	if err := file.Truncate(size); err != nil {
		_ = file.Close()
		t.Fatalf("truncate fixture: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close fixture: %v", err)
	}

	files, err := FromPaths(path)
	if err != nil {
		t.Fatalf("FromPaths() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("file count = %d, want 1", len(files))
	}
	if files[0].Size != size {
		t.Fatalf("Size() = %d, want %d", files[0].Size, size)
	}
	reader, err := files[0].Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = reader.Close() }()
	if _, err := io.CopyN(io.Discard, reader, 1); err != nil {
		t.Fatalf("stream first byte: %v", err)
	}
}

func TestSourcesReopenIndependentStreams(t *testing.T) {
	data := []byte("sample media content")
	path := filepath.Join(t.TempDir(), "image.jpg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	disk, err := FromPath(path)
	if err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="images[]"; filename="image.jpg"`)
	header.Set("Content-Type", "image/jpeg")
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = request.MultipartForm.RemoveAll() }()
	files := FromMultipartFiles([]*multipart.FileHeader{nil}, request.MultipartForm.File["images[]"])
	if len(files) != 1 || FromMultipart(nil) != nil {
		t.Fatal("multipart nil entries were not handled")
	}
	for name, source := range map[string]*domainmedia.UploadedFile{
		"HTTP": files[0], "disk": disk, "memory": FromBytes("image.jpg", "image/jpeg", data),
	} {
		t.Run(name, func(t *testing.T) {
			if err := source.Validate(); err != nil {
				t.Fatal(err)
			}
			if source.Filename != "image.jpg" || source.Size != int64(len(data)) || source.ContentType != "image/jpeg" {
				t.Fatalf("wrong metadata: %#v", source)
			}
			first, err := source.Open()
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.CopyN(io.Discard, first, 1); err != nil {
				t.Fatal(err)
			}
			if err := first.Close(); err != nil {
				t.Fatal(err)
			}
			second, err := source.Open()
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(second)
			if err != nil {
				t.Fatal(err)
			}
			if err := second.Close(); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, data) {
				t.Fatalf("source did not reopen from the start: %q", got)
			}
		})
	}
}
