package constants

const (
	MIMEImagePrefix = "image/"
	MIMEVideoPrefix = "video/"

	MIMEImageJPEG   = "image/jpeg"
	MIMEImageJPG    = "image/jpg"
	MIMEImagePNG    = "image/png"
	MIMEImageGIF    = "image/gif"
	MIMEImageWebP   = "image/webp"
	MIMEImageBMP    = "image/bmp"
	MIMEImageTIFF   = "image/tiff"
	MIMEImageAVIF   = "image/avif"
	MIMEOctetStream = "application/octet-stream"
)

const (
	ImageFormatJPEG = "jpeg"
	ImageFormatPNG  = "png"
	ImageFormatGIF  = "gif"
	ImageFormatWebP = "webp"
	ImageFormatBMP  = "bmp"
	ImageFormatTIFF = "tiff"
	ImageFormatAVIF = "avif"

	ImageExtensionJPEG = ".jpg"
	ImageExtensionPNG  = ".png"
	ImageExtensionGIF  = ".gif"
	ImageExtensionWebP = ".webp"
	ImageExtensionBMP  = ".bmp"
	ImageExtensionTIFF = ".tiff"
	ImageExtensionAVIF = ".avif"
)

const (
	ImageMinDimension                = 1
	GroupAvatarMaxBytes              = 10 << 20
	GroupAvatarHeaderReadLimit       = 1 << 20
	GroupAvatarMaxDimension          = 8192
	GroupAvatarMaxPixels       int64 = 32_000_000
	PrivatePhotoMaxDimension         = 12000
	PrivatePhotoMaxPixels      int64 = 50_000_000
)
