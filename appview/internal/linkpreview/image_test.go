package linkpreview

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

// UT-008: thumbnail validation derives format and dimensions from fully
// decoded bytes and enforces every byte and geometry limit.
func TestValidateThumbnail(t *testing.T) {
	t.Parallel()

	jpegBytes := encodeImage(t, "jpeg", image.NewRGBA(image.Rect(0, 0, 2, 3)))
	pngBytes := encodeImage(t, "png", image.NewRGBA(image.Rect(0, 0, 3, 2)))
	webpBytes, err := base64.StdEncoding.DecodeString("UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAgA0JaQAA3AA/vuUAAA=")
	if err != nil {
		t.Fatalf("decode WebP fixture: %v", err)
	}

	valid := []struct {
		name       string
		bytes      []byte
		wantMIME   string
		wantWidth  int
		wantHeight int
	}{
		{name: "JPEG", bytes: jpegBytes, wantMIME: "image/jpeg", wantWidth: 2, wantHeight: 3},
		{name: "PNG despite spoofed response MIME", bytes: pngBytes, wantMIME: "image/png", wantWidth: 3, wantHeight: 2},
		{name: "WebP", bytes: webpBytes, wantMIME: "image/webp", wantWidth: 1, wantHeight: 1},
	}
	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			thumbnail, err := ValidateThumbnail(tt.bytes)
			if err != nil {
				t.Fatalf("ValidateThumbnail(): %v", err)
			}
			if thumbnail.MIMEType != tt.wantMIME || thumbnail.Width != tt.wantWidth || thumbnail.Height != tt.wantHeight {
				t.Fatalf("thumbnail = %#v, want MIME %q dimensions %dx%d", thumbnail, tt.wantMIME, tt.wantWidth, tt.wantHeight)
			}
		})
	}

	gifBytes := encodeImage(t, "gif", image.NewRGBA(image.Rect(0, 0, 1, 1)))
	tooWide := encodeImage(t, "png", image.NewGray(image.Rect(0, 0, maxThumbnailDimension+1, 1)))
	tooManyPixels := encodeImage(t, "png", image.NewGray(image.Rect(0, 0, 4001, 4000)))
	invalid := []struct {
		name  string
		bytes []byte
	}{
		{name: "corrupt", bytes: pngBytes[:len(pngBytes)/2]},
		{name: "GIF", bytes: gifBytes},
		{name: "AVIF", bytes: []byte("\x00\x00\x00\x18ftypavif")},
		{name: "SVG", bytes: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{name: "over byte limit", bytes: append(append([]byte(nil), pngBytes...), bytes.Repeat([]byte{0}, maxThumbnailBytes)...)},
		{name: "over side limit", bytes: tooWide},
		{name: "over area limit", bytes: tooManyPixels},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ValidateThumbnail(tt.bytes); err == nil {
				t.Fatal("ValidateThumbnail() error = nil, want rejection")
			}
		})
	}
}

func encodeImage(t *testing.T, format string, source image.Image) []byte {
	t.Helper()
	var output bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&output, source, nil)
	case "png":
		err = png.Encode(&output, source)
	case "gif":
		err = gif.Encode(&output, source, nil)
	default:
		t.Fatalf("unsupported fixture format %q", format)
	}
	if err != nil {
		t.Fatalf("encode %s fixture: %v", format, err)
	}
	return output.Bytes()
}
