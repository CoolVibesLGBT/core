package ports

import "io"

type UploadedFile interface {
	Filename() string
	Size() int64
	ContentType() string
	Open() (io.ReadCloser, error)
}

type FormData struct {
	Values           map[string][]string
	Files            []UploadedFile
	ExpiresInSeconds *int
	ViewOnce         bool
	ClientID         string
	ImageCount       int
	VideoCount       int
}
