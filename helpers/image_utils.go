package helpers

import (
	"image"
	"os"
	"path/filepath"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// ResizeSquareCrop WebP formatında kare crop + resize yapar
func ResizeSquareCrop(srcPath, dstPath string, width, height int) error {
	img, err := imaging.Open(srcPath)
	if err != nil {
		return err
	}

	// Kare crop için kısa kenar
	var cropSize int
	if img.Bounds().Dx() < img.Bounds().Dy() {
		cropSize = img.Bounds().Dx()
	} else {
		cropSize = img.Bounds().Dy()
	}

	cropped := imaging.CropCenter(img, cropSize, cropSize)
	resized := imaging.Resize(cropped, width, height, imaging.Lanczos)

	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return webp.Encode(f, resized, &webp.Options{Lossless: true, Quality: 100})
}

// ResizeSquareKeepAspect WebP formatında kare kutu içinde aspect koruyarak resize eder
func ResizeSquareKeepAspect(srcPath, dstPath string, width, height int) error {
	size := width
	if height < width {
		size = height
	}

	img, err := imaging.Open(srcPath)
	if err != nil {
		return err
	}

	fitted := imaging.Fit(img, size, size, imaging.Lanczos)
	dst := imaging.New(size, size, image.Transparent)
	offset := image.Pt((size-fitted.Bounds().Dx())/2, (size-fitted.Bounds().Dy())/2)
	// DÜZELTME:
	offset = image.Pt((size-fitted.Bounds().Dx())/2, (size-fitted.Bounds().Dy())/2)

	final := imaging.Paste(dst, fitted, offset)

	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return webp.Encode(f, final, &webp.Options{Lossless: true, Quality: 100})
}

// ResizeLandscapeKeepAspect WebP formatında landscape için aspect koruyarak resize eder
func ResizeLandscapeKeepAspect(srcPath, dstPath string, width, height int) error {
	img, err := imaging.Open(srcPath)
	if err != nil {
		return err
	}

	fitted := imaging.Fit(img, width, height, imaging.Lanczos)
	dst := imaging.New(width, height, image.Transparent)
	offset := image.Pt((width-fitted.Bounds().Dx())/2, (height-fitted.Bounds().Dy())/2)
	final := imaging.Paste(dst, fitted, offset)

	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return webp.Encode(f, final, &webp.Options{Lossless: true, Quality: 100})
}

// ResizePortraitKeepAspect WebP formatında portrait için aspect koruyarak resize eder
func ResizePortraitKeepAspect(srcPath, dstPath string, width, height int) error {
	// Landscape ile aynı, isimlendirme amaçlı ayrı fonksiyon
	return ResizeLandscapeKeepAspect(srcPath, dstPath, width, height)
}
