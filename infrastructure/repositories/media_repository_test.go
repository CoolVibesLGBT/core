package repositories

import (
	domainmedia "core/domain/media"
	"core/helpers"
	"core/models/media"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

type generatedUploadReader struct {
	remaining   int64
	largestRead int
	closeErr    error
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

func (r *generatedUploadReader) Close() error { return r.closeErr }

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
	reader := &generatedUploadReader{remaining: size}
	file := domainmedia.NewUploadedFile("large.bin", size, "application/octet-stream", func() (io.ReadCloser, error) { return reader, nil })
	destination := filepath.Join(t.TempDir(), "nested", "large.bin")

	repository := &MediaRepository{}
	if err := repository.SaveUploadedFile(file, destination); err != nil {
		t.Fatalf("SaveUploadedFile() error = %v", err)
	}
	stat, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat stored upload: %v", err)
	}
	if stat.Size() != size {
		t.Fatalf("stored size = %d, want %d", stat.Size(), size)
	}
	if reader == nil || reader.largestRead <= 0 || int64(reader.largestRead) >= size {
		t.Fatalf("upload was not copied incrementally: reader=%#v", reader)
	}
}

func TestSaveUploadedFileReturnsCloseError(t *testing.T) {
	closeErr := errors.New("source close failed")
	file := domainmedia.NewUploadedFile("file.bin", 5, "application/octet-stream", func() (io.ReadCloser, error) {
		return &generatedUploadReader{remaining: 5, closeErr: closeErr}, nil
	})
	repository := &MediaRepository{}
	if err := repository.SaveUploadedFile(file, filepath.Join(t.TempDir(), "file.bin")); !errors.Is(err, closeErr) {
		t.Fatalf("close failure was swallowed: %v", err)
	}
}

func TestAddMediaCleansUpWhenUploadCloseFails(t *testing.T) {
	t.Chdir(t.TempDir())
	closeErr := errors.New("source close failed")
	file := domainmedia.NewUploadedFile("file.bin", 5, "application/octet-stream", func() (io.ReadCloser, error) {
		return &generatedUploadReader{remaining: 5, closeErr: closeErr}, nil
	})
	pool := &recordingTransactionPool{tx: &recordingTransaction{}}
	database := newRecordingPostCreationDB(t, pool)
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatal(err)
	}
	repository := NewMediaRepository(database, node)
	created, err := repository.AddMedia(uuid.New(), media.OwnerPost, uuid.New(), media.RolePost, file)
	if !errors.Is(err, closeErr) || created != nil {
		t.Fatalf("failed upload was persisted: created=%#v error=%v", created, err)
	}
	if pool.beginCount != 0 {
		t.Fatal("failed upload started persistence")
	}
	if files := storedUploadFiles(t); len(files) != 0 {
		t.Fatalf("failed upload left stored files: %v", files)
	}
}
