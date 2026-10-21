package media

import (
	"io"
	"strings"
)

// UploadedFile is the shared input to the media pipeline, independent of its
// HTTP, disk or memory source. Validate is called before persistence.
type UploadedFile struct {
	Filename    string
	Size        int64
	ContentType string
	// Open supplies a fresh stream for each read without buffering the upload.
	Open func() (io.ReadCloser, error) `json:"-"`
}

func NewUploadedFile(filename string, size int64, contentType string, open func() (io.ReadCloser, error)) *UploadedFile {
	return &UploadedFile{
		Filename:    strings.TrimSpace(filename),
		Size:        size,
		ContentType: strings.TrimSpace(contentType),
		Open:        open,
	}
}

func (file *UploadedFile) Validate() error {
	if file == nil {
		return ErrEmptyFile
	}
	if strings.TrimSpace(file.Filename) == "" {
		return ErrEmptyFilename
	}
	if file.Size <= 0 || file.Open == nil {
		return ErrEmptyFile
	}
	return nil
}
