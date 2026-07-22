package helpers

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/chai2010/webp"
)

func TestResizeLandscapeKeepAspectUsesBlurredBackgroundAtExactSize(t *testing.T) {
	srcPath := writeTestPNG(t, 400, 200)
	dstPath := filepath.Join(t.TempDir(), "landscape.webp")

	if err := ResizeLandscapeKeepAspect(srcPath, dstPath, 100, 100); err != nil {
		t.Fatalf("ResizeLandscapeKeepAspect() error = %v", err)
	}

	img := openWebP(t, dstPath)
	if got := img.Bounds().Dx(); got != 100 {
		t.Fatalf("width = %d, want 100", got)
	}
	if got := img.Bounds().Dy(); got != 100 {
		t.Fatalf("height = %d, want 100", got)
	}

	topLeft := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
	if topLeft.A == 0 {
		t.Fatal("top-left pixel should come from blurred background, not transparent padding")
	}
}

func TestResizeSquareKeepAspectUsesBlurredBackgroundAtExactSize(t *testing.T) {
	srcPath := writeTestPNG(t, 400, 200)
	dstPath := filepath.Join(t.TempDir(), "square.webp")

	if err := ResizeSquareKeepAspect(srcPath, dstPath, 100, 100); err != nil {
		t.Fatalf("ResizeSquareKeepAspect() error = %v", err)
	}

	img := openWebP(t, dstPath)
	if got := img.Bounds().Dx(); got != 100 {
		t.Fatalf("width = %d, want 100", got)
	}
	if got := img.Bounds().Dy(); got != 100 {
		t.Fatalf("height = %d, want 100", got)
	}

	topLeft := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
	if topLeft.A == 0 {
		t.Fatal("top-left pixel should come from blurred background, not transparent padding")
	}

	center := color.NRGBAModel.Convert(img.At(50, 50)).(color.NRGBA)
	if center.A == 0 {
		t.Fatal("center pixel should contain image content")
	}
}

func TestResizeSquareCropProducesSquareOutput(t *testing.T) {
	srcPath := writeTestPNG(t, 400, 200)
	dstPath := filepath.Join(t.TempDir(), "avatar.webp")

	if err := ResizeSquareCrop(srcPath, dstPath, 100, 100); err != nil {
		t.Fatalf("ResizeSquareCrop() error = %v", err)
	}

	img := openWebP(t, dstPath)
	if got := img.Bounds().Dx(); got != 100 {
		t.Fatalf("width = %d, want 100", got)
	}
	if got := img.Bounds().Dy(); got != 100 {
		t.Fatalf("height = %d, want 100", got)
	}
}

func TestLoadImageWithOrientationStreamsSparseUpload(t *testing.T) {
	path := writeTestPNG(t, 2, 2)
	const sparseSize = int64(512 << 20)
	if err := os.Truncate(path, sparseSize); err != nil {
		t.Fatalf("extend sparse image fixture: %v", err)
	}

	img, err := LoadImageWithOrientation(path)
	if err != nil {
		t.Fatalf("LoadImageWithOrientation(sparse upload) error = %v", err)
	}
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Fatalf("decoded bounds = %v, want 2x2", img.Bounds())
	}
}

func writeTestPNG(t *testing.T, width, height int) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "source.png")
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.NRGBA{
				R: uint8((x * 255) / max(width-1, 1)),
				G: uint8((y * 255) / max(height-1, 1)),
				B: 128,
				A: 255,
			})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create() error = %v", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			t.Fatalf("Close() error = %v", cerr)
		}
	}()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	return path
}

func openWebP(t *testing.T, path string) image.Image {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			t.Fatalf("Close() error = %v", cerr)
		}
	}()

	img, err := webp.Decode(f)
	if err != nil {
		t.Fatalf("webp.Decode() error = %v", err)
	}

	return img
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
