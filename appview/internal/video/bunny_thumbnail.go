package video

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"math"
	"net/http"
	"regexp"
)

const MaxThumbnailBytes = 1_000_000

var ErrInvalidThumbnail = errors.New("invalid video thumbnail")
var errThumbnailUpload = errors.New("could not set video thumbnail")

var bunnyThumbnailFilename = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}\.(?:jpg|jpeg|png|webp)$`)

func ValidateThumbnail(data []byte) error {
	if len(data) > MaxThumbnailBytes {
		return ErrInvalidThumbnail
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width < 1 || config.Height < 1 || config.Width > 1600 || config.Height > 1600 {
		return ErrInvalidThumbnail
	}
	return nil
}

func prepareThumbnail(data []byte, width, height int) ([]byte, error) {
	if err := ValidateThumbnail(data); err != nil {
		return nil, err
	}
	source, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil || width < 1 || height < 1 {
		return nil, ErrInvalidThumbnail
	}
	bounds := source.Bounds()
	ratio := float64(width) / float64(height)
	cropWidth := max(1, min(bounds.Dx(), int(math.Round(float64(bounds.Dy())*ratio))))
	cropHeight := max(1, min(bounds.Dy(), int(math.Round(float64(bounds.Dx())/ratio))))
	left := bounds.Min.X + (bounds.Dx()-cropWidth)/2
	top := bounds.Min.Y + (bounds.Dy()-cropHeight)/2
	crop := image.Rect(left, top, left+cropWidth, top+cropHeight)
	target := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	draw.Draw(target, target.Bounds(), source, crop.Min, draw.Src)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, target, &jpeg.Options{Quality: 86}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (b *Bunny) uploadThumbnail(ctx context.Context, videoID string, data []byte) error {
	var response struct {
		Success bool `json:"success"`
	}
	err := b.libraryRequest(ctx, http.MethodPost, "/videos/"+videoID+"/thumbnail", bytes.NewReader(data), "application/octet-stream", &response)
	if err != nil || !response.Success {
		return errThumbnailUpload
	}
	return nil
}
