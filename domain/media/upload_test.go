package media

import (
	"io"
	"strings"
	"testing"
)

func TestUploadedFileValidation(t *testing.T) {
	open := func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("image")), nil }
	file := NewUploadedFile(" image.jpg ", 10, " image/jpeg ", open)
	if err := file.Validate(); err != nil {
		t.Fatalf("valid upload rejected: %v", err)
	}
	if file.Filename != "image.jpg" || file.ContentType != "image/jpeg" {
		t.Fatalf("metadata was not normalized: %#v", file)
	}
	for name, test := range map[string]struct {
		file *UploadedFile
		want error
	}{
		"missing file":   {nil, ErrEmptyFile},
		"empty filename": {NewUploadedFile("", 10, "image/jpeg", open), ErrEmptyFilename},
		"empty file":     {NewUploadedFile("image.jpg", 0, "image/jpeg", open), ErrEmptyFile},
		"missing stream": {NewUploadedFile("image.jpg", 10, "image/jpeg", nil), ErrEmptyFile},
	} {
		t.Run(name, func(t *testing.T) {
			if err := test.file.Validate(); err != test.want {
				t.Fatalf("Validate() = %v, want %v", err, test.want)
			}
		})
	}
}

func TestUploadedFileMetadataHasNoByteSizeCap(t *testing.T) {
	const reportedSize = int64(8)<<30 + 17
	file := NewUploadedFile("archive.bin", reportedSize, "application/octet-stream", func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("")), nil })
	if err := file.Validate(); err != nil {
		t.Fatalf("large file rejected: %v", err)
	}
	if file.Size != reportedSize {
		t.Fatalf("Size = %d, want %d", file.Size, reportedSize)
	}
}
