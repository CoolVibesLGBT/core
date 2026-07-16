package chat

import (
	"testing"
)

func TestParseExpiresInSeconds(t *testing.T) {
	tests := []struct {
		name    string
		values  []string
		want    int
		wantNil bool
		wantErr bool
	}{
		{name: "missing", values: nil, wantNil: true},
		{name: "empty slice", values: []string{}, wantNil: true},
		{name: "blank", values: []string{""}, wantNil: true},
		{name: "whitespace", values: []string{"  "}, wantNil: true},
		{name: "zero", values: []string{"0"}, wantNil: true},
		{name: "trimmed zero", values: []string{" 0 "}, wantNil: true},
		{name: "minimum", values: []string{"10"}, want: 10},
		{name: "maximum", values: []string{"604800"}, want: 604800},
		{name: "below minimum", values: []string{"9"}, wantErr: true},
		{name: "negative", values: []string{"-10"}, wantErr: true},
		{name: "above maximum", values: []string{"604801"}, wantErr: true},
		{name: "not an integer", values: []string{"tomorrow"}, wantErr: true},
		{name: "fractional", values: []string{"10.5"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExpiresInSeconds(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil value, got %d", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected seconds, got nil")
			}
			if *got != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, *got)
			}
		})
	}
}
