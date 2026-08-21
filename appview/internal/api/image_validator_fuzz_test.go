package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"runtime"
	"testing"
)

func FuzzImageGeometryAllowed(f *testing.F) {
	for _, seed := range [][2]int{
		{0, 0},
		{1, 1},
		{HardMaxImageWidth, 1_953},
		{4_000, 4_000},
		{HardMaxImageWidth + 1, 1},
		{-1, 1},
		{int(^uint(0) >> 1), 2},
	} {
		f.Add(seed[0], seed[1])
	}

	limits := DefaultImageDecodeLimits()
	f.Fuzz(func(t *testing.T, width, height int) {
		if !imageGeometryAllowed(width, height, limits) {
			return
		}
		if width <= 0 || height <= 0 ||
			width > limits.MaxWidth || height > limits.MaxHeight {
			t.Fatalf("accepted invalid axes: %dx%d", width, height)
		}
		if uint64(width) > limits.MaxPixels/uint64(height) {
			t.Fatalf("accepted over-pixel geometry: %dx%d", width, height)
		}
		longSide, shortSide := width, height
		if height > width {
			longSide, shortSide = height, width
		}
		if longSide > shortSide*limits.MaxAspectRatio {
			t.Fatalf("accepted extreme aspect geometry: %dx%d", width, height)
		}
	})
}

func FuzzImageValidator(f *testing.F) {
	f.Add("image/jpeg", encodeFuzzJPEG())
	f.Add("image/png", encodeFuzzPNG())
	f.Add("image/webp", mustDecodeBase64(f, smallWebPBase64))
	f.Add("image/jpeg", []byte("not-an-image"))
	f.Add("image/gif", []byte("GIF89a"))

	f.Fuzz(func(t *testing.T, declaredMIME string, payload []byte) {
		validator, err := NewImageValidator(DefaultImageDecodeLimits())
		if err != nil {
			t.Fatalf("construct image validator: %v", err)
		}
		validated, err := validator.Validate(
			context.Background(), declaredMIME, payload,
		)
		if err != nil {
			if !errors.Is(err, ErrScheduledImageInvalid) {
				t.Fatalf("unexpected validation error: %v", err)
			}
			return
		}
		if canonicalImageMIME(validated.Format) != canonicalContentType(declaredMIME) {
			t.Fatalf("accepted MIME/format mismatch: %q/%q", declaredMIME, validated.Format)
		}
		if !imageGeometryAllowed(
			validated.Width, validated.Height, DefaultImageDecodeLimits(),
		) {
			t.Fatalf("accepted invalid geometry: %#v", validated)
		}
	})
}

// BenchmarkImageValidatorWorstCase is the reproducible heap/RSS probe for the
// hard 16-megapixel policy boundary. Run each codec independently and wrap the
// command with the target container's peak-RSS tool; for example:
//
//	go test ./internal/api -run '^$' \
//	  -bench '^BenchmarkImageValidatorWorstCase/jpeg$' -benchmem -benchtime=1x
//
// The fixtures are generated before timing so the reported allocations/op are
// decoder allocations. Process RSS is still the release gate because it also
// includes the Go runtime, codec scratch buffers, and the AppView baseline.
func BenchmarkImageValidatorWorstCase(b *testing.B) {
	fixtures := []struct {
		name     string
		mimeType string
		payload  func(testing.TB) []byte
	}{
		{name: "jpeg", mimeType: "image/jpeg", payload: encodeWorstCaseJPEG},
		{name: "png", mimeType: "image/png", payload: encodeWorstCasePNG},
		{
			name: "webp", mimeType: "image/webp",
			payload: func(tb testing.TB) []byte {
				return mustDecodeBase64(tb, worstCaseWebPBase64)
			},
		},
	}

	for _, fixture := range fixtures {
		b.Run(fixture.name, func(b *testing.B) {
			payload := fixture.payload(b)
			validator, err := NewImageValidator(DefaultImageDecodeLimits())
			if err != nil {
				b.Fatalf("construct image validator: %v", err)
			}
			runtime.GC()
			b.ReportAllocs()
			b.ReportMetric(float64(HardMaxImagePixels), "pixels/op")
			b.ResetTimer()
			for range b.N {
				if _, err := validator.Validate(
					context.Background(), fixture.mimeType, payload,
				); err != nil {
					b.Fatalf("validate %s worst-case fixture: %v", fixture.name, err)
				}
			}
		})
	}
}

func encodeFuzzJPEG() []byte {
	var payload bytes.Buffer
	if err := jpeg.Encode(&payload, image.NewGray(image.Rect(0, 0, 1, 1)), nil); err != nil {
		panic(err)
	}
	return payload.Bytes()
}

func encodeFuzzPNG() []byte {
	var payload bytes.Buffer
	if err := png.Encode(&payload, image.NewGray(image.Rect(0, 0, 1, 1))); err != nil {
		panic(err)
	}
	return payload.Bytes()
}

func encodeWorstCaseJPEG(tb testing.TB) []byte {
	tb.Helper()
	var payload bytes.Buffer
	if err := jpeg.Encode(
		&payload,
		image.NewGray(image.Rect(0, 0, 4_000, 4_000)),
		nil,
	); err != nil {
		tb.Fatalf("encode worst-case JPEG: %v", err)
	}
	return payload.Bytes()
}

func encodeWorstCasePNG(tb testing.TB) []byte {
	tb.Helper()
	var payload bytes.Buffer
	if err := png.Encode(
		&payload,
		image.NewGray(image.Rect(0, 0, 4_000, 4_000)),
	); err != nil {
		tb.Fatalf("encode worst-case PNG: %v", err)
	}
	return payload.Bytes()
}

func mustDecodeBase64(tb testing.TB, encoded string) []byte {
	tb.Helper()
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		tb.Fatalf("decode image fixture: %v", err)
	}
	return payload
}

const smallWebPBase64 = "UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAgA0JaQAA3AA/vuUAAA="

// 4000x4000 lossless WebP containing a single solid colour. Its compact
// encoded size deliberately demonstrates why compressed-byte limits cannot
// substitute for decoded-pixel limits.
const worstCaseWebPBase64 = "UklGRqACAABXRUJQVlA4TJQCAAAvn8/nAwfQ//73v/9hLCFBgv+32yMi9UZB2zZM+bPftRTR/wkAkuT//yBERKTk+Y///Pfff//5" +
	"z3///fef//z333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///fff" +
	"f//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///fff" +
	"f//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///fff" +
	"f//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///fff" +
	"f//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///fff" +
	"f//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///fff" +
	"f//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///fff" +
	"f//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///ffff//9999///3333///fff" +
	"f/7bOQA="
