package helpers

import (
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"

	"github.com/boxes-ltd/imaging"
	"github.com/chai2010/webp"
	"github.com/rwcarlsen/goexif/exif"
)

const blurBackgroundSigma = 18.0

const (
	backgroundDimOpacity           = 0.18
	foregroundShadowOpacity        = 0.26
	foregroundShadowOffsetY        = 10
	foregroundShadowAmbientPadding = 18
	foregroundShadowAmbientAlpha   = 120
	foregroundShadowAmbientSigma   = 9.0
	foregroundShadowDiffusePadding = 40
	foregroundShadowDiffuseAlpha   = 86
	foregroundShadowDiffuseSigma   = 20.0
	foregroundSharpenSigma         = 0.6
	maxForegroundUpscale           = 1.15
	backgroundSaturationDrop       = -18
	backgroundContrastDrop         = -6
)

func LoadImageWithOrientation(srcPath string) (image.Image, error) {
	source, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	img, decodeErr := imaging.Decode(source)
	closeErr := source.Close()
	if decodeErr != nil {
		return nil, decodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}

	// EXIF parsing needs its own pass. Reopen the file instead of retaining the
	// complete compressed upload in memory alongside the decoded pixel buffer.
	orientationSource, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	oriented, orientationErr := fixOrientation(img, orientationSource)
	closeErr = orientationSource.Close()
	if orientationErr != nil {
		return nil, orientationErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return oriented, nil
}

func saveWEBP(dstPath string, img image.Image) (err error) {
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

	if err := webp.Encode(f, img, &webp.Options{
		Lossless: true,
		Quality:  100,
	}); err != nil {
		return err
	}

	return nil
}

func resizeToFit(img image.Image, maxWidth, maxHeight int, maxScale float64) image.Image {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	if srcWidth <= 0 || srcHeight <= 0 || maxWidth <= 0 || maxHeight <= 0 {
		return img
	}

	scale := math.Min(
		float64(maxWidth)/float64(srcWidth),
		float64(maxHeight)/float64(srcHeight),
	)
	if maxScale <= 0 {
		maxScale = 1
	}
	if scale > maxScale {
		scale = maxScale
	}
	if scale <= 0 {
		return img
	}

	dstWidth := int(math.Round(float64(srcWidth) * scale))
	dstHeight := int(math.Round(float64(srcHeight) * scale))
	if dstWidth < 1 {
		dstWidth = 1
	}
	if dstHeight < 1 {
		dstHeight = 1
	}

	if dstWidth == srcWidth && dstHeight == srcHeight {
		return img
	}

	return imaging.Resize(img, dstWidth, dstHeight, imaging.Lanczos)
}

func resizeDownOnly(img image.Image, maxWidth, maxHeight int) image.Image {
	return resizeToFit(img, maxWidth, maxHeight, 1)
}

func centeredCropRect(img image.Image, width, height int) image.Rectangle {
	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()
	targetRatio := float64(width) / float64(height)
	currentRatio := float64(srcWidth) / float64(srcHeight)

	if currentRatio > targetRatio {
		cropWidth := int(math.Round(float64(srcHeight) * targetRatio))
		x0 := bounds.Min.X + (srcWidth-cropWidth)/2
		return image.Rect(x0, bounds.Min.Y, x0+cropWidth, bounds.Max.Y)
	}

	cropHeight := int(math.Round(float64(srcWidth) / targetRatio))
	y0 := bounds.Min.Y + (srcHeight-cropHeight)/2
	return image.Rect(bounds.Min.X, y0, bounds.Max.X, y0+cropHeight)
}

func buildSoftShadow(width, height, padding int, maxAlpha uint8) *image.NRGBA {
	shadow := image.NewNRGBA(image.Rect(0, 0, width+padding*2, height+padding*2))
	if padding <= 0 || maxAlpha == 0 {
		return shadow
	}

	left := padding
	top := padding
	right := left + width - 1
	bottom := top + height - 1
	feather := float64(padding)

	for y := 0; y < shadow.Bounds().Dy(); y++ {
		for x := 0; x < shadow.Bounds().Dx(); x++ {
			dx := 0
			switch {
			case x < left:
				dx = left - x
			case x > right:
				dx = x - right
			}

			dy := 0
			switch {
			case y < top:
				dy = top - y
			case y > bottom:
				dy = y - bottom
			}

			distance := math.Hypot(float64(dx), float64(dy))
			if distance >= feather {
				continue
			}

			t := 1 - (distance / feather)
			t = t * t * (3 - 2*t)

			shadow.SetNRGBA(x, y, color.NRGBA{
				R: 0,
				G: 0,
				B: 0,
				A: uint8(float64(maxAlpha) * t),
			})
		}
	}

	return shadow
}

func composeBlurredBackground(img image.Image, width, height int, foregroundMaxScale float64) image.Image {
	background := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
	background = imaging.Blur(background, blurBackgroundSigma)
	background = imaging.AdjustSaturation(background, backgroundSaturationDrop)
	background = imaging.AdjustContrast(background, backgroundContrastDrop)
	background = imaging.Overlay(background, imaging.New(width, height, color.NRGBA{0, 0, 0, 255}), image.Point{}, backgroundDimOpacity)

	foreground := resizeToFit(img, width, height, foregroundMaxScale)
	foreground = imaging.Sharpen(foreground, foregroundSharpenSigma)
	offset := image.Pt(
		(width-foreground.Bounds().Dx())/2,
		(height-foreground.Bounds().Dy())/2,
	)

	diffuseShadow := buildSoftShadow(
		foreground.Bounds().Dx(),
		foreground.Bounds().Dy(),
		foregroundShadowDiffusePadding,
		foregroundShadowDiffuseAlpha,
	)
	diffuseShadow = imaging.Blur(diffuseShadow, foregroundShadowDiffuseSigma)

	background = imaging.Overlay(
		background,
		diffuseShadow,
		image.Pt(offset.X-foregroundShadowDiffusePadding, offset.Y-foregroundShadowDiffusePadding+foregroundShadowOffsetY+2),
		foregroundShadowOpacity*0.6,
	)

	ambientShadow := buildSoftShadow(
		foreground.Bounds().Dx(),
		foreground.Bounds().Dy(),
		foregroundShadowAmbientPadding,
		foregroundShadowAmbientAlpha,
	)
	ambientShadow = imaging.Blur(ambientShadow, foregroundShadowAmbientSigma)

	background = imaging.Overlay(
		background,
		ambientShadow,
		image.Pt(offset.X-foregroundShadowAmbientPadding, offset.Y-foregroundShadowAmbientPadding+foregroundShadowOffsetY),
		foregroundShadowOpacity,
	)

	return imaging.Paste(background, foreground, offset)
}

// ResizeSquareCrop merkezi crop yapar, sonra sadece downscale eder.
func ResizeSquareCrop(srcPath, dstPath string, width, height int) error {
	img, err := LoadImageWithOrientation(srcPath)
	if err != nil {
		return err
	}

	cropped := imaging.Crop(img, centeredCropRect(img, width, height))
	resized := resizeDownOnly(cropped, width, height)
	return saveWEBP(dstPath, resized)
}

// ResizeSquareKeepAspect resmi crop etmeden blur arka plan üstünde kare kutuya yerleştirir.
func ResizeSquareKeepAspect(srcPath, dstPath string, width, height int) error {
	img, err := LoadImageWithOrientation(srcPath)
	if err != nil {
		return err
	}

	return saveWEBP(dstPath, composeBlurredBackground(img, width, height, maxForegroundUpscale))
}

// EXIF orientasyonu uygular.
func fixOrientation(img image.Image, source io.Reader) (image.Image, error) {
	exifData, err := exif.Decode(source)
	if err != nil {
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

// ResizeLandscapeKeepAspect hedef boyutu blur arka planla doldurur, resmi crop etmeden ortalar.
func ResizeLandscapeKeepAspect(srcPath, dstPath string, width, height int) error {
	img, err := LoadImageWithOrientation(srcPath)
	if err != nil {
		return err
	}

	return saveWEBP(dstPath, composeBlurredBackground(img, width, height, maxForegroundUpscale))
}

// ResizePortraitKeepAspect landscape versiyonuyla aynı fit davranışını kullanır.
func ResizePortraitKeepAspect(srcPath, dstPath string, width, height int) error {
	return ResizeLandscapeKeepAspect(srcPath, dstPath, width, height)
}

// ResizePortraitKeepAspectCropCenter hedef orana göre ortadan crop eder, sonra downscale eder.
func ResizePortraitKeepAspectCropCenter(srcPath, dstPath string, width, height int) error {
	img, err := LoadImageWithOrientation(srcPath)
	if err != nil {
		return err
	}

	cropped := imaging.Crop(img, centeredCropRect(img, width, height))
	resized := resizeDownOnly(cropped, width, height)
	return saveWEBP(dstPath, resized)
}
