package media

import (
	"core/application/ports"
	"io"
	"mime/multipart"
)

type MultipartUploadedFile struct {
	Header *multipart.FileHeader
}

func NewMultipartUploadedFile(header *multipart.FileHeader) ports.UploadedFile {
	if header == nil {
		return nil
	}
	return MultipartUploadedFile{Header: header}
}

func NewMultipartUploadedFiles(headers []*multipart.FileHeader) []ports.UploadedFile {
	files := make([]ports.UploadedFile, 0, len(headers))
	for _, header := range headers {
		if header != nil {
			files = append(files, NewMultipartUploadedFile(header))
		}
	}
	return files
}

func (f MultipartUploadedFile) Filename() string {
	if f.Header == nil {
		return ""
	}
	return f.Header.Filename
}

func (f MultipartUploadedFile) Size() int64 {
	if f.Header == nil {
		return 0
	}
	return f.Header.Size
}

func (f MultipartUploadedFile) ContentType() string {
	if f.Header == nil {
		return ""
	}
	contentType := f.Header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = f.Header.Header.Get("Content-type")
	}
	return contentType
}

func (f MultipartUploadedFile) Open() (io.ReadCloser, error) {
	return f.Header.Open()
}
