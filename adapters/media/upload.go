package media

import (
	"bytes"
	domainmedia "core/domain/media"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// FromMultipart adapts an HTTP upload without opening or buffering its contents.
func FromMultipart(header *multipart.FileHeader) *domainmedia.UploadedFile {
	if header == nil {
		return nil
	}
	return domainmedia.NewUploadedFile(header.Filename, header.Size, header.Header.Get("Content-Type"), func() (io.ReadCloser, error) {
		return header.Open()
	})
}

func FromMultipartFiles(groups ...[]*multipart.FileHeader) []*domainmedia.UploadedFile {
	files := make([]*domainmedia.UploadedFile, 0)
	for _, group := range groups {
		for _, header := range group {
			if header != nil {
				files = append(files, FromMultipart(header))
			}
		}
	}
	return files
}

// FromPath reads metadata and opens a fresh stream each time the media pipeline
// needs the file. File contents are never buffered in memory.
func FromPath(path string) (_ *domainmedia.UploadedFile, retErr error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		buffer := make([]byte, 512)
		read, readErr := file.Read(buffer)
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}
		contentType = http.DetectContentType(buffer[:read])
	}
	return domainmedia.NewUploadedFile(filepath.Base(path), stat.Size(), contentType, func() (io.ReadCloser, error) {
		return os.Open(path)
	}), nil
}

func FromPaths(paths ...string) ([]*domainmedia.UploadedFile, error) {
	files := make([]*domainmedia.UploadedFile, 0, len(paths))
	for _, path := range paths {
		file, err := FromPath(path)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func FromBytes(filename, contentType string, data []byte) *domainmedia.UploadedFile {
	return domainmedia.NewUploadedFile(filename, int64(len(data)), contentType, func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	})
}
