package handlers

import (
	"core/application/ports"
	"io"
	"mime/multipart"
)

type multipartUploadedFile struct {
	header *multipart.FileHeader
}

func (f multipartUploadedFile) Filename() string {
	if f.header == nil {
		return ""
	}
	return f.header.Filename
}

func (f multipartUploadedFile) Size() int64 {
	if f.header == nil {
		return 0
	}
	return f.header.Size
}

func (f multipartUploadedFile) ContentType() string {
	if f.header == nil {
		return ""
	}
	contentType := f.header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = f.header.Header.Get("Content-type")
	}
	return contentType
}

func (f multipartUploadedFile) Open() (io.ReadCloser, error) {
	return f.header.Open()
}

func uploadedFile(header *multipart.FileHeader) ports.UploadedFile {
	if header == nil {
		return nil
	}
	return multipartUploadedFile{header: header}
}

func uploadedFormData(values map[string][]string, groups ...[]*multipart.FileHeader) ports.FormData {
	files := make([]ports.UploadedFile, 0)
	for _, group := range groups {
		for _, header := range group {
			if header != nil {
				files = append(files, uploadedFile(header))
			}
		}
	}
	return ports.FormData{Values: values, Files: files}
}
