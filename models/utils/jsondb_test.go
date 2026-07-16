package utils

import (
	"context"
	"testing"
)

func TestLocalizedStringGormValueReturnsJSONString(t *testing.T) {
	expr := (LocalizedString{"en": "hello"}).GormValue(context.Background(), nil)
	if expr.SQL != "?" {
		t.Fatalf("GormValue() SQL = %q, want %q", expr.SQL, "?")
	}

	if len(expr.Vars) != 1 || expr.Vars[0] != `{"en":"hello"}` {
		t.Fatalf("GormValue() vars = %#v", expr.Vars)
	}
}

func TestLocalizedStringScanAcceptsString(t *testing.T) {
	var value LocalizedString
	if err := (&value).Scan(`{"en":"hello"}`); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if got := value["en"]; got != "hello" {
		t.Fatalf("Scan() value = %q, want %q", got, "hello")
	}
}

func TestFileVariantsGormValueReturnsJSONString(t *testing.T) {
	width := 240
	height := 240

	expr := (FileVariants{
		Image: &ImageVariants{
			Thumbnail: &VariantInfo{
				URL:    "thumb.webp",
				Width:  &width,
				Height: &height,
				Format: "webp",
			},
		},
	}).GormValue(context.Background(), nil)
	if expr.SQL != "?" {
		t.Fatalf("GormValue() SQL = %q, want %q", expr.SQL, "?")
	}
	if len(expr.Vars) != 1 {
		t.Fatalf("GormValue() vars = %#v", expr.Vars)
	}
}

func TestFileVariantsScanAcceptsString(t *testing.T) {
	var variants FileVariants
	if err := (&variants).Scan(`{"image":{"thumbnail":{"url":"thumb.webp"}}}`); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if variants.Image == nil || variants.Image.Thumbnail == nil || variants.Image.Thumbnail.URL != "thumb.webp" {
		t.Fatalf("Scan() decoded unexpected variants: %#v", variants)
	}
}
