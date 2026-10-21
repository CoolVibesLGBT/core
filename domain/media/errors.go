package media

import "errors"

var (
	ErrEmptyFilename                       = errors.New("uploaded file name cannot be empty")
	ErrEmptyFile                           = errors.New("uploaded file cannot be empty")
	ErrInvalidPrivatePhotoAccessStatus     = errors.New("invalid private photo access status")
	ErrInvalidPrivatePhotoAccessTransition = errors.New("invalid private photo access transition")
	ErrPrivatePhotoImageRequired           = errors.New("private photos must be image files")
	ErrPrivatePhotoImageDimensions         = errors.New("private photo dimensions are too large")
)
