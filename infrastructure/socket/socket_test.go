package socket

import (
	"errors"
	"testing"
)

func TestShouldSuppressSocketError(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		err  error
		want bool
	}{
		{
			name: "suppress ping timeout",
			msg:  "failed to get ping writer",
			err:  errors.New("write: timeout"),
			want: true,
		},
		{
			name: "suppress closed ping writer",
			msg:  "failed to close ping writer",
			err:  errors.New("use of closed network connection"),
			want: true,
		},
		{
			name: "keep unrelated error",
			msg:  "failed to get ping writer",
			err:  errors.New("permission denied"),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldSuppressSocketError(tc.msg, tc.err); got != tc.want {
				t.Fatalf("shouldSuppressSocketError(%q, %v) = %v, want %v", tc.msg, tc.err, got, tc.want)
			}
		})
	}
}
