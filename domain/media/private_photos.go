package media

import (
	"bytes"
	"errors"
	"strings"
)

type PrivatePhotoAccessStatus string

const (
	PrivatePhotoAccessPending  PrivatePhotoAccessStatus = "pending"
	PrivatePhotoAccessApproved PrivatePhotoAccessStatus = "approved"
	PrivatePhotoAccessDenied   PrivatePhotoAccessStatus = "denied"
)

var (
	ErrInvalidPrivatePhotoAccessStatus     = errors.New("invalid private photo access status")
	ErrInvalidPrivatePhotoAccessTransition = errors.New("invalid private photo access transition")
	ErrPrivatePhotoImageRequired           = errors.New("private photos must be image files")
	ErrPrivatePhotoImageDimensions         = errors.New("private photo dimensions are too large")
)

// IsPrivatePhotoImageHeader validates the file's bytes instead of trusting a
// client-supplied MIME header. It intentionally recognizes only image formats
// supported by the media pipeline.
func IsPrivatePhotoImageHeader(header []byte) bool {
	switch {
	case bytes.HasPrefix(header, []byte{0xff, 0xd8, 0xff}): // JPEG
		return true
	case bytes.HasPrefix(header, []byte("\x89PNG\r\n\x1a\n")):
		return true
	case bytes.HasPrefix(header, []byte("GIF87a")), bytes.HasPrefix(header, []byte("GIF89a")):
		return true
	case len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WEBP")):
		return true
	case bytes.HasPrefix(header, []byte("BM")): // BMP
		return true
	case bytes.HasPrefix(header, []byte("II*\x00")), bytes.HasPrefix(header, []byte("MM\x00*")): // TIFF
		return true
	}
	return false
}

func ParsePrivatePhotoAccessStatus(value string) (PrivatePhotoAccessStatus, error) {
	status := PrivatePhotoAccessStatus(strings.ToLower(strings.TrimSpace(value)))
	if !status.IsValid() {
		return "", ErrInvalidPrivatePhotoAccessStatus
	}
	return status, nil
}

func ParsePrivatePhotoAccessDecision(value string) (PrivatePhotoAccessStatus, error) {
	status, err := ParsePrivatePhotoAccessStatus(value)
	if err != nil || (status != PrivatePhotoAccessApproved && status != PrivatePhotoAccessDenied) {
		return "", ErrInvalidPrivatePhotoAccessStatus
	}
	return status, nil
}

func (s PrivatePhotoAccessStatus) IsValid() bool {
	switch s {
	case PrivatePhotoAccessPending, PrivatePhotoAccessApproved, PrivatePhotoAccessDenied:
		return true
	default:
		return false
	}
}

func (s PrivatePhotoAccessStatus) CanView() bool {
	return s == PrivatePhotoAccessApproved
}

// RequestPrivatePhotoAccess is idempotent for pending and approved requests.
// A denied request may be requested again and returns to pending.
func RequestPrivatePhotoAccess(current *PrivatePhotoAccessStatus) (PrivatePhotoAccessStatus, bool) {
	if current == nil || *current == PrivatePhotoAccessDenied {
		return PrivatePhotoAccessPending, true
	}
	return *current, false
}

// RespondToPrivatePhotoAccess applies an owner's approve/deny decision. An
// identical repeated decision is idempotent, while changing a completed
// decision must go through the explicit revoke or request flow.
func RespondToPrivatePhotoAccess(current, decision PrivatePhotoAccessStatus) (PrivatePhotoAccessStatus, bool, error) {
	if _, err := ParsePrivatePhotoAccessDecision(string(decision)); err != nil {
		return "", false, err
	}
	if current == decision {
		return current, false, nil
	}
	if current != PrivatePhotoAccessPending {
		return "", false, ErrInvalidPrivatePhotoAccessTransition
	}
	return decision, true, nil
}

// RevokePrivatePhotoAccess removes viewing permission while retaining the
// request record for audit and for a later explicit re-request.
func RevokePrivatePhotoAccess(current PrivatePhotoAccessStatus) (PrivatePhotoAccessStatus, bool, error) {
	if !current.IsValid() {
		return "", false, ErrInvalidPrivatePhotoAccessStatus
	}
	if current == PrivatePhotoAccessDenied {
		return current, false, nil
	}
	return PrivatePhotoAccessDenied, true, nil
}
