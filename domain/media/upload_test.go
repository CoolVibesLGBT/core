package media

import "testing"

func TestUploadedFileValidation(t *testing.T) {
	file, err := NewUploadedFile(" image.jpg ", 10, "image/jpeg")
	if err != nil {
		t.Fatalf("NewUploadedFile() error = %v", err)
	}
	if file.Filename != "image.jpg" {
		t.Fatalf("Filename = %q, want %q", file.Filename, "image.jpg")
	}

	if _, err := NewUploadedFile("", 10, "image/jpeg"); err != ErrEmptyFilename {
		t.Fatalf("empty filename error = %v, want %v", err, ErrEmptyFilename)
	}
	if _, err := NewUploadedFile("image.jpg", 0, "image/jpeg"); err != ErrEmptyFile {
		t.Fatalf("empty file error = %v, want %v", err, ErrEmptyFile)
	}
}
