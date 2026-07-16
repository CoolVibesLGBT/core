package media

import (
	"errors"
	"strings"
)

var (
	ErrEmptyFilename = errors.New("uploaded file name cannot be empty")
	ErrEmptyFile     = errors.New("uploaded file cannot be empty")
)

type UploadedFile struct {
	Filename    string
	Size        int64
	ContentType string
}

func NewUploadedFile(filename string, size int64, contentType string) (UploadedFile, error) {
	file := UploadedFile{
		Filename:    strings.TrimSpace(filename),
		Size:        size,
		ContentType: strings.TrimSpace(contentType),
	}
	if file.Filename == "" {
		return UploadedFile{}, ErrEmptyFilename
	}
	if file.Size <= 0 {
		return UploadedFile{}, ErrEmptyFile
	}
	return file, nil
}
