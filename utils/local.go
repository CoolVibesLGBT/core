package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
)

func DetectContentType(filename string) string {
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		// Fallback
		return "application/octet-stream"
	}
	return mimeType
}

func DetectContentTypeFromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buffer := make([]byte, 512)
	_, err = f.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

func DetectFormField(contentType string) string {
	if strings.HasPrefix(contentType, "image/") {
		return "images[]"
	}
	if strings.HasPrefix(contentType, "video/") {
		return "videos[]"
	}
	return "files[]"
}

func FilesFromDiskEx(paths []string) ([]*multipart.FileHeader, error) {
	var files []*multipart.FileHeader

	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			return nil, err
		}

		filename := filepath.Base(path)
		contentType := DetectContentType(filename)
		fieldName := DetectFormField(contentType)

		header := &multipart.FileHeader{
			Filename: filename,
			Size:     stat.Size(),
			Header:   make(textproto.MIMEHeader),
		}

		header.Header.Set(
			"Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, filename),
		)
		header.Header.Set("Content-Type", contentType)

		files = append(files, header)
	}

	return files, nil
}

func FilesFromDiskExternal(paths []string) ([]*multipart.FileHeader, error) {
	var files []*multipart.FileHeader

	// Yeni buffer, içine multipart form verisini yazacağız
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, path := range paths {
		// Dosyayı aç
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		// Dosya adı
		filename := filepath.Base(path)

		// Multipart formda "file" alanı oluştur, istediğin isim olabilir
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil, err
		}

		// Dosyayı multipart alanına yaz
		_, err = io.Copy(part, file)
		if err != nil {
			return nil, err
		}
	}

	// Writer'ı kapat, bitiş boundary eklenir
	err := writer.Close()
	if err != nil {
		return nil, err
	}

	// Şimdi body.Bytes() içinde gerçek multipart verisi var
	// Multipart reader ile okuyup FileHeader'ları alacağız

	reader := multipart.NewReader(bytes.NewReader(body.Bytes()), writer.Boundary())

	form, err := reader.ReadForm(10 << 20) // 10 MB max memory
	if err != nil {
		return nil, err
	}

	// form.File map[string][]*multipart.FileHeader var
	for _, fhs := range form.File {
		for _, fh := range fhs {
			files = append(files, fh)
		}
	}

	return files, nil
}

func FilesFromDisk(paths []string) ([]*multipart.FileHeader, error) {
	var files []*multipart.FileHeader

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		filename := filepath.Base(path)

		// Dosyanın mime tipini dosya içeriğinden tespit et
		contentType, err := DetectContentTypeFromFile(path)
		if err != nil {
			contentType = "application/octet-stream"
		}

		// Content-Disposition ve Content-Type headerlarını manuel oluştur
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
		h.Set("Content-Type", contentType)

		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(part, file)
		if err != nil {
			return nil, err
		}
	}

	err := writer.Close()
	if err != nil {
		return nil, err
	}

	reader := multipart.NewReader(bytes.NewReader(body.Bytes()), writer.Boundary())

	form, err := reader.ReadForm(10 << 20) // 10MB max memory
	if err != nil {
		return nil, err
	}

	for _, fhs := range form.File {
		for _, fh := range fhs {
			files = append(files, fh)
		}
	}

	return files, nil
}
