package news

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDiskUploadedFilesStreamsFilesWithoutMultipartMemoryLimit(t *testing.T) {
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

	files, err := newDiskUploadedFiles([]string{path})
	if err != nil {
		t.Fatalf("newDiskUploadedFiles() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("file count = %d, want 1", len(files))
	}
	if files[0].Size() != size {
		t.Fatalf("Size() = %d, want %d", files[0].Size(), size)
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
