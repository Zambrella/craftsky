package linkpreview

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

const (
	maxThumbnailBytes     = 1_000_000
	maxThumbnailDimension = 8_192
	maxThumbnailPixels    = 16_000_000
)

var ErrInvalidThumbnail = errors.New("invalid link preview thumbnail")

type Thumbnail struct {
	Bytes    []byte
	MIMEType string
	Width    int
	Height   int
}

func ValidateThumbnail(input []byte) (Thumbnail, error) {
	if len(input) == 0 || len(input) > maxThumbnailBytes {
		return Thumbnail{}, ErrInvalidThumbnail
	}
	configuration, format, err := image.DecodeConfig(bytes.NewReader(input))
	mimeType := thumbnailMIME(format)
	if err != nil || mimeType == "" || !thumbnailGeometryAllowed(configuration.Width, configuration.Height) {
		return Thumbnail{}, ErrInvalidThumbnail
	}
	decoded, decodedFormat, err := image.Decode(bytes.NewReader(input))
	if err != nil || decoded == nil || decodedFormat != format {
		return Thumbnail{}, ErrInvalidThumbnail
	}
	width, height := decoded.Bounds().Dx(), decoded.Bounds().Dy()
	if width != configuration.Width || height != configuration.Height || !thumbnailGeometryAllowed(width, height) {
		return Thumbnail{}, ErrInvalidThumbnail
	}
	return Thumbnail{
		Bytes:    bytes.Clone(input),
		MIMEType: mimeType,
		Width:    width,
		Height:   height,
	}, nil
}

func thumbnailMIME(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return ""
	}
}

func thumbnailGeometryAllowed(width, height int) bool {
	return width > 0 && height > 0 &&
		width <= maxThumbnailDimension && height <= maxThumbnailDimension &&
		uint64(width) <= uint64(maxThumbnailPixels)/uint64(height)
}
