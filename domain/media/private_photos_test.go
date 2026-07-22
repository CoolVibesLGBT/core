package media

import (
	"errors"
	"testing"
)

func TestPrivatePhotoAccessLifecycle(t *testing.T) {
	status, changed := RequestPrivatePhotoAccess(nil)
	if status != PrivatePhotoAccessPending || !changed {
		t.Fatalf("new request = %q, %v", status, changed)
	}

	status, changed, err := RespondToPrivatePhotoAccess(status, PrivatePhotoAccessApproved)
	if err != nil || status != PrivatePhotoAccessApproved || !changed {
		t.Fatalf("approve = %q, %v, %v", status, changed, err)
	}
	if !status.CanView() {
		t.Fatal("approved request cannot view private photos")
	}

	status, changed, err = RevokePrivatePhotoAccess(status)
	if err != nil || status != PrivatePhotoAccessDenied || !changed {
		t.Fatalf("revoke = %q, %v, %v", status, changed, err)
	}

	status, changed = RequestPrivatePhotoAccess(&status)
	if status != PrivatePhotoAccessPending || !changed {
		t.Fatalf("re-request = %q, %v", status, changed)
	}
}

func TestPrivatePhotoAccessRejectsChangingCompletedDecision(t *testing.T) {
	_, _, err := RespondToPrivatePhotoAccess(PrivatePhotoAccessApproved, PrivatePhotoAccessDenied)
	if !errors.Is(err, ErrInvalidPrivatePhotoAccessTransition) {
		t.Fatalf("completed decision error = %v", err)
	}
	if _, err := ParsePrivatePhotoAccessDecision("pending"); !errors.Is(err, ErrInvalidPrivatePhotoAccessStatus) {
		t.Fatalf("pending decision error = %v", err)
	}
}

func TestPrivatePhotoImageHeaderUsesFileSignature(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		valid  bool
	}{
		{name: "jpeg", header: []byte{0xff, 0xd8, 0xff, 0xe0}, valid: true},
		{name: "png", header: []byte("\x89PNG\r\n\x1a\n"), valid: true},
		{name: "gif", header: []byte("GIF89a"), valid: true},
		{name: "webp", header: []byte("RIFF\x00\x00\x00\x00WEBP"), valid: true},
		{name: "unsupported heic", header: []byte("\x00\x00\x00\x18ftypheic"), valid: false},
		{name: "unsupported avif", header: []byte("\x00\x00\x00\x18ftypmif1avif"), valid: false},
		{name: "spoofed text", header: []byte("this is not an image"), valid: false},
		{name: "truncated riff", header: []byte("RIFF"), valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsPrivatePhotoImageHeader(test.header); got != test.valid {
				t.Fatalf("IsPrivatePhotoImageHeader() = %v, want %v", got, test.valid)
			}
		})
	}
}
