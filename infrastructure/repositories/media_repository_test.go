package repositories

import (
	"core/models/media"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type generatedUploadReader struct {
	remaining   int64
	largestRead int
}

func (r *generatedUploadReader) Read(buffer []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	if int64(len(buffer)) > r.remaining {
		buffer = buffer[:r.remaining]
	}
	clear(buffer)
	if len(buffer) > r.largestRead {
		r.largestRead = len(buffer)
	}
	r.remaining -= int64(len(buffer))
	return len(buffer), nil
}

func (*generatedUploadReader) Close() error { return nil }

type generatedUploadedFile struct {
	size   int64
	reader *generatedUploadReader
}

func (*generatedUploadedFile) Filename() string    { return "large.bin" }
func (f *generatedUploadedFile) Size() int64       { return f.size }
func (*generatedUploadedFile) ContentType() string { return "application/octet-stream" }
func (f *generatedUploadedFile) Open() (io.ReadCloser, error) {
	f.reader = &generatedUploadReader{remaining: f.size}
	return f.reader, nil
}

func TestShouldProcessAsync(t *testing.T) {
	if !shouldProcessAsync("image/jpeg") {
		t.Fatalf("image/jpeg should be processed asynchronously")
	}
	if !shouldProcessAsync("video/mp4") {
		t.Fatalf("video/mp4 should be processed asynchronously")
	}
	if shouldProcessAsync("application/pdf") {
		t.Fatalf("application/pdf should not be processed asynchronously")
	}
}

func TestInitialFileVariantsForImage(t *testing.T) {
	variants := initialFileVariants("image/jpeg", "./static/uploads/a.jpg", ".jpg", 1234)
	if variants == nil || variants.Image == nil || variants.Image.Original == nil {
		t.Fatalf("expected initial image variants, got %#v", variants)
	}

	if variants.Image.Original.URL != "/static/uploads/a.jpg" {
		t.Fatalf("unexpected original url: %q", variants.Image.Original.URL)
	}
	if variants.Image.Original.Format != "jpg" {
		t.Fatalf("unexpected original format: %q", variants.Image.Original.Format)
	}
	if variants.Image.Original.Size != 1234 {
		t.Fatalf("unexpected original size: %d", variants.Image.Original.Size)
	}
}

func TestInitialFileVariantsForNonImage(t *testing.T) {
	if variants := initialFileVariants("video/mp4", "./static/uploads/a.mp4", ".mp4", 1234); variants != nil {
		t.Fatalf("expected nil variants for non-image upload, got %#v", variants)
	}
}

func TestProtectedMediaRolesArePrivate(t *testing.T) {
	for _, role := range []media.MediaRole{media.RoleChatImage, media.RoleChatMedia, media.RoleChatVideo, media.RolePrivatePhoto} {
		if isPublicMediaRole(role) {
			t.Fatalf("protected role %q was marked public", role)
		}
	}
	if !isPublicMediaRole(media.RolePost) {
		t.Fatal("regular post media should be public")
	}
}

func TestSaveUploadedFileStreamsWithoutApplicationByteCap(t *testing.T) {
	const size = int64(8 << 20)
	upload := &generatedUploadedFile{size: size}
	destination := filepath.Join(t.TempDir(), "nested", "large.bin")

	repository := &MediaRepository{}
	if err := repository.SaveUploadedFile(upload, destination); err != nil {
		t.Fatalf("SaveUploadedFile() error = %v", err)
	}
	stat, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat stored upload: %v", err)
	}
	if stat.Size() != size {
		t.Fatalf("stored size = %d, want %d", stat.Size(), size)
	}
	if upload.reader == nil || upload.reader.largestRead <= 0 || int64(upload.reader.largestRead) >= size {
		t.Fatalf("upload was not copied incrementally: reader=%#v", upload.reader)
	}
}
