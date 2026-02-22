package helpers

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"

	"github.com/rwcarlsen/goexif/exif"

	"github.com/boxes-ltd/imaging"
	"github.com/chai2010/webp"
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
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

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

	fitted := imaging.Fill(img, size, size, imaging.Center, imaging.Lanczos)
	dst := imaging.New(size, size, image.Transparent)
	offset := image.Pt((size-fitted.Bounds().Dx())/2, (size-fitted.Bounds().Dy())/2)

	final := imaging.Paste(dst, fitted, offset)

	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return webp.Encode(f, final, &webp.Options{Lossless: true, Quality: 100})
}

// EXIF orientasyonu uygular
func fixOrientation(img image.Image, data []byte) (image.Image, error) {
	exifData, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		// EXIF yoksa, oryantasyon düzeltemezsin
		return img, nil
	}

	orientationTag, err := exifData.Get(exif.Orientation)
	if err != nil {
		return img, nil
	}

	orientation, err := orientationTag.Int(0)
	if err != nil {
		return img, nil
	}

	switch orientation {
	case 2:
		img = imaging.FlipH(img)
	case 3:
		img = imaging.Rotate180(img)
	case 4:
		img = imaging.FlipV(img)
	case 5:
		img = imaging.Transpose(img)
	case 6:
		img = imaging.Rotate270(img)
	case 7:
		img = imaging.Transverse(img)
	case 8:
		img = imaging.Rotate90(img)
	}

	return img, nil
}

func ResizeLandscapeKeepAspect(srcPath, dstPath string, width, height int) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	fmt.Println("WIDTH", width, "HEIGHT", height)

	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	img, err = fixOrientation(img, data)
	if err != nil {
		return err
	}

	final := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	return webp.Encode(f, final, &webp.Options{
		Lossless: true,
		Quality:  100,
	})
}

// ResizePortraitKeepAspect WebP formatında portrait için aspect koruyarak resize eder
func ResizePortraitKeepAspect(srcPath, dstPath string, width, height int) error {
	// Landscape ile aynı, isimlendirme amaçlı ayrı fonksiyon
	return ResizeLandscapeKeepAspect(srcPath, dstPath, width, height)
}

func ResizePortraitKeepAspectCropCenter(srcPath, dstPath string, width, height int) error {
	fmt.Println("ResizePortraitKeepAspectCropCenter")
	img, err := imaging.Open(srcPath)
	if err != nil {
		return err
	}

	// Kare crop için kısa kenar (örneğin ortadan crop yapmak istiyorsan)
	cropWidth := img.Bounds().Dx()
	cropHeight := img.Bounds().Dy()

	// Aspect ratio hedef için
	targetRatio := float64(width) / float64(height)
	currentRatio := float64(cropWidth) / float64(cropHeight)

	var cropRect image.Rectangle

	if currentRatio > targetRatio {
		// Resim geniş, yatay crop yap
		newWidth := int(float64(cropHeight) * targetRatio)
		x0 := (cropWidth - newWidth) / 2
		cropRect = image.Rect(x0, 0, x0+newWidth, cropHeight)
	} else {
		// Resim uzun, dikey crop yap
		newHeight := int(float64(cropWidth) / targetRatio)
		y0 := (cropHeight - newHeight) / 2
		cropRect = image.Rect(0, y0, cropWidth, y0+newHeight)
	}

	cropped := imaging.Crop(img, cropRect)
	resized := imaging.Resize(cropped, width, height, imaging.Lanczos)

	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return webp.Encode(f, resized, &webp.Options{Lossless: true, Quality: 100})
}
