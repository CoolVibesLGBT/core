package news

import (
	"core/application/ports"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

type diskUploadedFile struct {
	path        string
	filename    string
	size        int64
	contentType string
}

func newDiskUploadedFiles(paths []string) ([]ports.UploadedFile, error) {
	files := make([]ports.UploadedFile, 0, len(paths))
	for _, path := range paths {
		file, err := newDiskUploadedFile(path)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func newDiskUploadedFile(path string) (_ diskUploadedFile, retErr error) {
	file, err := os.Open(path)
	if err != nil {
		return diskUploadedFile{}, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()

	stat, err := file.Stat()
	if err != nil {
		return diskUploadedFile{}, err
	}
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		buffer := make([]byte, 512)
		read, readErr := file.Read(buffer)
		if readErr != nil && readErr != io.EOF {
			return diskUploadedFile{}, readErr
		}
		contentType = http.DetectContentType(buffer[:read])
	}

	return diskUploadedFile{
		path:        path,
		filename:    filepath.Base(path),
		size:        stat.Size(),
		contentType: contentType,
	}, nil
}

func (f diskUploadedFile) Filename() string    { return f.filename }
func (f diskUploadedFile) Size() int64         { return f.size }
func (f diskUploadedFile) ContentType() string { return f.contentType }
func (f diskUploadedFile) Open() (io.ReadCloser, error) {
	return os.Open(f.path)
}
