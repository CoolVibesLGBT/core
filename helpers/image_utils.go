package helpers

import (
	"image"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

// ResizeImage belirli genişlik/yüksekliğe göre resmi yeniden boyutlandırır
func ResizeImage(inputPath string, outputPath string, width, height int) error {
	img, err := imaging.Open(inputPath)
	if err != nil {
		return err
	}

	// Oran korumalı şekilde yeniden boyutlandır
	resized := imaging.Fit(img, width, height, imaging.Lanczos)

	// Hedef klasör varsa oluştur
	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return err
	}

	// Kaydet
	return imaging.Save(resized, outputPath)
}

// ResizePortraitCrop görüntüyü mobil odaklı (portrait) olarak yeniden boyutlandırır.
// Fazla genişlik varsa crop yapar.
func ResizePortraitCrop(inputPath, outputPath string, targetWidth, targetHeight int) error {
	img, err := imaging.Open(inputPath)
	if err != nil {
		return err
	}

	srcBounds := img.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Kaynak oran ve hedef oran karşılaştırması
	srcRatio := float64(srcWidth) / float64(srcHeight)
	targetRatio := float64(targetWidth) / float64(targetHeight)

	var cropped image.Image

	if srcRatio > targetRatio {
		// Görsel yataysa — ortadan kırp (kenarları kes)
		newWidth := int(float64(srcHeight) * targetRatio)
		offsetX := (srcWidth - newWidth) / 2
		cropRect := image.Rect(offsetX, 0, offsetX+newWidth, srcHeight)
		cropped = imaging.Crop(img, cropRect)
	} else {
		// Görsel dikeyse — üstten alttan kırpma gerekmez
		cropped = img
	}

	// Şimdi hedef boyuta ölçekle
	resized := imaging.Fill(cropped, targetWidth, targetHeight, imaging.Center, imaging.Lanczos)

	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		return err
	}

	return imaging.Save(resized, outputPath)
}
