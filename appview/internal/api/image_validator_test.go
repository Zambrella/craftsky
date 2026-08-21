package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"
	"time"
)

func mustTestImageValidator(t testing.TB) ImageValidator {
	t.Helper()
	validator, err := NewImageValidator(DefaultImageDecodeLimits())
	if err != nil {
		t.Fatalf("construct test image validator: %v", err)
	}
	return validator
}

func TestImageValidatorRejectsOversizedGeometryBeforeFullDecode(t *testing.T) {
	decoder := &recordingImageDecoder{
		config:       image.Config{Width: HardMaxImageWidth + 1, Height: 1},
		configFormat: "jpeg",
	}
	validator, err := newImageValidator(
		DefaultImageDecodeLimits(),
		decoder,
		nil,
		time.Now,
	)
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}

	_, err = validator.Validate(context.Background(), "image/jpeg", []byte("compact-header"))
	if !errors.Is(err, ErrScheduledImageInvalid) {
		t.Fatalf("Validate error = %v, want ErrScheduledImageInvalid", err)
	}
	if decoder.configCalls != 1 {
		t.Fatalf("DecodeConfig calls = %d, want 1", decoder.configCalls)
	}
	if decoder.decodeCalls != 0 {
		t.Fatalf("Decode calls = %d, want 0", decoder.decodeCalls)
	}
}

func TestImageValidatorFullyDecodesAcceptedGeometry(t *testing.T) {
	decoder := &recordingImageDecoder{
		config:       image.Config{Width: 100, Height: 50},
		configFormat: "jpeg",
		decodeErr:    io.ErrUnexpectedEOF,
	}
	validator, err := newImageValidator(
		DefaultImageDecodeLimits(),
		decoder,
		nil,
		time.Now,
	)
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}

	_, err = validator.Validate(context.Background(), "image/jpeg", []byte("truncated-body"))
	if !errors.Is(err, ErrScheduledImageInvalid) {
		t.Fatalf("Validate error = %v, want ErrScheduledImageInvalid", err)
	}
	if decoder.configCalls != 1 {
		t.Fatalf("DecodeConfig calls = %d, want 1", decoder.configCalls)
	}
	if decoder.decodeCalls != 1 {
		t.Fatalf("Decode calls = %d, want 1", decoder.decodeCalls)
	}
}

func TestImageValidatorCapsConcurrentDecoderWork(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	decoder := blockingImageDecoder{started: started, release: release}
	validator, err := newImageValidator(
		DefaultImageDecodeLimits(),
		decoder,
		nil,
		time.Now,
	)
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}

	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, validateErr := validator.Validate(
				context.Background(),
				"image/jpeg",
				[]byte("valid-image"),
			)
			results <- validateErr
		}()
	}

	<-started
	select {
	case <-started:
		close(release)
		t.Fatal("a second decoder entered before the shared permit was released")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)

	for range 2 {
		if validateErr := <-results; validateErr != nil {
			t.Fatalf("Validate error = %v, want nil", validateErr)
		}
	}
}

func TestImageValidatorContainsDecoderPanicAndReleasesPermit(t *testing.T) {
	decoder := &panicOnceImageDecoder{}
	validator, err := newImageValidator(
		DefaultImageDecodeLimits(),
		decoder,
		nil,
		time.Now,
	)
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}

	_, err = validator.Validate(context.Background(), "image/jpeg", []byte("first"))
	if !errors.Is(err, ErrScheduledImageInvalid) {
		t.Fatalf("first Validate error = %v, want ErrScheduledImageInvalid", err)
	}
	_, err = validator.Validate(context.Background(), "image/jpeg", []byte("second"))
	if err != nil {
		t.Fatalf("second Validate error = %v, want nil", err)
	}
}

func TestImageValidatorEmitsContentFreeOperationalResult(t *testing.T) {
	decoder := &recordingImageDecoder{
		config:       image.Config{Width: 1, Height: 1},
		configFormat: "jpeg",
		decoded:      image.NewRGBA(image.Rect(0, 0, 1, 1)),
		decodeFormat: "jpeg",
	}
	observer := &recordingImageValidationObserver{}
	started := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	times := []time.Time{started, started.Add(5 * time.Millisecond)}
	now := func() time.Time {
		value := times[0]
		times = times[1:]
		return value
	}
	validator, err := newImageValidator(
		DefaultImageDecodeLimits(),
		decoder,
		observer,
		now,
	)
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}

	_, err = validator.Validate(
		context.Background(),
		"image/jpeg",
		[]byte("private-image-canary"),
	)
	if err != nil {
		t.Fatalf("Validate error = %v, want nil", err)
	}
	if len(observer.results) != 2 {
		t.Fatalf("observer calls = %d, want 2", len(observer.results))
	}
	if got := observer.results[0]; got != (imageValidationResult{
		result:   "started",
		format:   "unknown",
		duration: 0,
		inFlight: 1,
	}) {
		t.Fatalf("observer start result = %#v", got)
	}
	if got := observer.results[1]; got != (imageValidationResult{
		result:   "success",
		format:   "jpeg",
		duration: 5 * time.Millisecond,
		inFlight: 0,
	}) {
		t.Fatalf("observer completion result = %#v", got)
	}
}

func TestImageGeometryAllowed(t *testing.T) {
	limits := DefaultImageDecodeLimits()
	maxInt := int(^uint(0) >> 1)
	tests := []struct {
		name          string
		width, height int
		want          bool
	}{
		{name: "small", width: 1, height: 1, want: true},
		{name: "zero width", width: 0, height: 1},
		{name: "zero height", width: 1, height: 0},
		{name: "negative width", width: -1, height: 1},
		{name: "negative height", width: 1, height: -1},
		{name: "exact width", width: HardMaxImageWidth, height: 1_953, want: true},
		{name: "over width", width: HardMaxImageWidth + 1, height: 1_953},
		{name: "exact height", width: 1_953, height: HardMaxImageHeight, want: true},
		{name: "over height", width: 1_953, height: HardMaxImageHeight + 1},
		{name: "exact pixels", width: 4_000, height: 4_000, want: true},
		{name: "over pixels", width: 4_001, height: 4_000},
		{name: "exact wide aspect", width: 20, height: 1, want: true},
		{name: "over wide aspect", width: 21, height: 1},
		{name: "exact tall aspect", width: 1, height: 20, want: true},
		{name: "over tall aspect", width: 1, height: 21},
		{name: "integer overflow input", width: maxInt, height: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := imageGeometryAllowed(test.width, test.height, limits); got != test.want {
				t.Fatalf("imageGeometryAllowed(%d, %d) = %t, want %t", test.width, test.height, got, test.want)
			}
		})
	}
}

func TestImageDecodeLimitsFailClosed(t *testing.T) {
	valid := DefaultImageDecodeLimits()
	tests := []struct {
		name   string
		mutate func(*ImageDecodeLimits)
	}{
		{name: "zero width", mutate: func(value *ImageDecodeLimits) { value.MaxWidth = 0 }},
		{name: "negative width", mutate: func(value *ImageDecodeLimits) { value.MaxWidth = -1 }},
		{name: "width above ceiling", mutate: func(value *ImageDecodeLimits) { value.MaxWidth = HardMaxImageWidth + 1 }},
		{name: "zero height", mutate: func(value *ImageDecodeLimits) { value.MaxHeight = 0 }},
		{name: "height above ceiling", mutate: func(value *ImageDecodeLimits) { value.MaxHeight = HardMaxImageHeight + 1 }},
		{name: "zero pixels", mutate: func(value *ImageDecodeLimits) { value.MaxPixels = 0 }},
		{name: "pixels above ceiling", mutate: func(value *ImageDecodeLimits) { value.MaxPixels = HardMaxImagePixels + 1 }},
		{name: "zero aspect", mutate: func(value *ImageDecodeLimits) { value.MaxAspectRatio = 0 }},
		{name: "aspect above ceiling", mutate: func(value *ImageDecodeLimits) { value.MaxAspectRatio = HardMaxImageAspectRatio + 1 }},
		{name: "zero concurrency", mutate: func(value *ImageDecodeLimits) { value.MaxConcurrentDecodes = 0 }},
		{name: "concurrency above ceiling", mutate: func(value *ImageDecodeLimits) { value.MaxConcurrentDecodes = HardMaxConcurrentImageDecodes + 1 }},
		{name: "zero admission wait", mutate: func(value *ImageDecodeLimits) { value.AdmissionWait = 0 }},
		{name: "negative admission wait", mutate: func(value *ImageDecodeLimits) { value.AdmissionWait = -time.Second }},
		{name: "admission wait above ceiling", mutate: func(value *ImageDecodeLimits) { value.AdmissionWait = hardMaxImageAdmissionWait + time.Nanosecond }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limits := valid
			test.mutate(&limits)
			if _, err := NewImageValidator(limits); err == nil {
				t.Fatal("NewImageValidator error = nil, want fail-closed configuration error")
			}
		})
	}
}

func TestImageValidatorAcceptsOnlyConsistentRegisteredFormats(t *testing.T) {
	validator, err := NewImageValidator(DefaultImageDecodeLimits())
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}
	validPNG := func() []byte {
		var buffer bytes.Buffer
		if encodeErr := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 1, 1))); encodeErr != nil {
			t.Fatalf("encode PNG fixture: %v", encodeErr)
		}
		return buffer.Bytes()
	}()
	validWebP, err := base64.StdEncoding.DecodeString(
		"UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAgA0JaQAA3AA/vuUAAA=",
	)
	if err != nil {
		t.Fatalf("decode WebP fixture: %v", err)
	}
	tests := []struct {
		name       string
		mimeType   string
		payload    []byte
		wantFormat string
	}{
		{name: "JPEG", mimeType: "image/jpeg", payload: validJPEGBytes(t), wantFormat: "jpeg"},
		{name: "PNG", mimeType: "image/png", payload: validPNG, wantFormat: "png"},
		{name: "WebP", mimeType: "image/webp", payload: validWebP, wantFormat: "webp"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validated, validateErr := validator.Validate(context.Background(), test.mimeType, test.payload)
			if validateErr != nil {
				t.Fatalf("Validate error = %v, want nil", validateErr)
			}
			if validated.Format != test.wantFormat || validated.Width != 1 || validated.Height != 1 {
				t.Fatalf("validated image = %#v", validated)
			}
		})
	}

	if _, err := validator.Validate(context.Background(), "image/jpeg", validPNG); !errors.Is(err, ErrScheduledImageInvalid) {
		t.Fatalf("MIME mismatch error = %v, want ErrScheduledImageInvalid", err)
	}
}

func TestImageValidatorRejectsHeaderAndDecodeDisagreement(t *testing.T) {
	tests := []struct {
		name    string
		decoder *recordingImageDecoder
	}{
		{
			name: "unsupported header format",
			decoder: &recordingImageDecoder{
				config: image.Config{Width: 1, Height: 1}, configFormat: "gif",
			},
		},
		{
			name: "decoded format changed",
			decoder: &recordingImageDecoder{
				config: image.Config{Width: 1, Height: 1}, configFormat: "jpeg",
				decoded: image.NewRGBA(image.Rect(0, 0, 1, 1)), decodeFormat: "png",
			},
		},
		{
			name: "decoded width changed",
			decoder: &recordingImageDecoder{
				config: image.Config{Width: 1, Height: 1}, configFormat: "jpeg",
				decoded: image.NewRGBA(image.Rect(0, 0, 2, 1)), decodeFormat: "jpeg",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator, err := newImageValidator(
				DefaultImageDecodeLimits(), test.decoder, nil, time.Now,
			)
			if err != nil {
				t.Fatalf("construct image validator: %v", err)
			}
			if _, err := validator.Validate(
				context.Background(), "image/jpeg", []byte("image"),
			); !errors.Is(err, ErrScheduledImageInvalid) {
				t.Fatalf("Validate error = %v, want ErrScheduledImageInvalid", err)
			}
		})
	}
}

func TestImageValidatorRejectsUnsupportedFormatWithEmptyDeclaredMIME(t *testing.T) {
	decoder := &recordingImageDecoder{
		config:       image.Config{Width: 1, Height: 1},
		configFormat: "gif",
		decoded:      image.NewRGBA(image.Rect(0, 0, 1, 1)),
		decodeFormat: "gif",
	}
	validator, err := newImageValidator(
		DefaultImageDecodeLimits(), decoder, nil, time.Now,
	)
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}

	if _, err := validator.Validate(
		context.Background(), "", []byte("unsupported-image"),
	); !errors.Is(err, ErrScheduledImageInvalid) {
		t.Fatalf("Validate error = %v, want ErrScheduledImageInvalid", err)
	}
	if decoder.decodeCalls != 0 {
		t.Fatalf("Decode calls = %d, want 0", decoder.decodeCalls)
	}
}

func TestImageValidatorBoundsWaitersAndObservesCancellation(t *testing.T) {
	started := make(chan struct{}, 4)
	release := make(chan struct{})
	validator, err := newImageValidator(
		DefaultImageDecodeLimits(),
		blockingImageDecoder{started: started, release: release},
		nil,
		time.Now,
	)
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}

	firstResult := make(chan error, 1)
	go func() {
		_, validateErr := validator.Validate(
			context.Background(), "image/jpeg", []byte("first"),
		)
		firstResult <- validateErr
	}()
	<-started

	waitingContext, cancelWaiting := context.WithCancel(context.Background())
	secondResult := make(chan error, 1)
	go func() {
		_, validateErr := validator.Validate(
			waitingContext, "image/jpeg", []byte("second"),
		)
		secondResult <- validateErr
	}()
	deadline := time.Now().Add(time.Second)
	for len(validator.waiters) != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(validator.waiters) != 1 {
		close(release)
		t.Fatal("second validation did not enter the bounded waiter slot")
	}

	if _, err := validator.Validate(
		context.Background(), "image/jpeg", []byte("third"),
	); !errors.Is(err, ErrImageDecodeSaturated) {
		close(release)
		t.Fatalf("third Validate error = %v, want ErrImageDecodeSaturated", err)
	}
	cancelWaiting()
	if err := <-secondResult; !errors.Is(err, context.Canceled) {
		close(release)
		t.Fatalf("cancelled Validate error = %v, want context.Canceled", err)
	}

	close(release)
	if err := <-firstResult; err != nil {
		t.Fatalf("first Validate error = %v, want nil", err)
	}
	if _, err := validator.Validate(
		context.Background(), "image/jpeg", []byte("after-cancellation"),
	); err != nil {
		t.Fatalf("Validate after cancellation error = %v, want nil", err)
	}
}

func TestImageValidatorAdmissionWaitIsBounded(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	limits := DefaultImageDecodeLimits()
	limits.AdmissionWait = 5 * time.Millisecond
	validator, err := newImageValidator(
		limits,
		blockingImageDecoder{started: started, release: release},
		nil,
		time.Now,
	)
	if err != nil {
		t.Fatalf("construct image validator: %v", err)
	}

	firstResult := make(chan error, 1)
	go func() {
		_, validateErr := validator.Validate(
			context.Background(), "image/jpeg", []byte("first"),
		)
		firstResult <- validateErr
	}()
	<-started
	if _, err := validator.Validate(
		context.Background(), "image/jpeg", []byte("waiting"),
	); !errors.Is(err, ErrImageDecodeSaturated) {
		close(release)
		t.Fatalf("waiting Validate error = %v, want ErrImageDecodeSaturated", err)
	}
	close(release)
	if err := <-firstResult; err != nil {
		t.Fatalf("first Validate error = %v, want nil", err)
	}
}

type recordingImageDecoder struct {
	config       image.Config
	configFormat string
	configErr    error
	decoded      image.Image
	decodeFormat string
	decodeErr    error
	configCalls  int
	decodeCalls  int
}

func (decoder *recordingImageDecoder) DecodeConfig(io.Reader) (image.Config, string, error) {
	decoder.configCalls++
	return decoder.config, decoder.configFormat, decoder.configErr
}

func (decoder *recordingImageDecoder) Decode(io.Reader) (image.Image, string, error) {
	decoder.decodeCalls++
	return decoder.decoded, decoder.decodeFormat, decoder.decodeErr
}

type blockingImageDecoder struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (decoder blockingImageDecoder) DecodeConfig(io.Reader) (image.Config, string, error) {
	decoder.started <- struct{}{}
	<-decoder.release
	return image.Config{Width: 1, Height: 1}, "jpeg", nil
}

func (blockingImageDecoder) Decode(io.Reader) (image.Image, string, error) {
	value := image.NewRGBA(image.Rect(0, 0, 1, 1))
	value.Set(0, 0, color.White)
	return value, "jpeg", nil
}

type panicOnceImageDecoder struct {
	panicked bool
}

func (decoder *panicOnceImageDecoder) DecodeConfig(io.Reader) (image.Config, string, error) {
	if !decoder.panicked {
		decoder.panicked = true
		panic("decoder detail must not escape")
	}
	return image.Config{Width: 1, Height: 1}, "jpeg", nil
}

func (*panicOnceImageDecoder) Decode(io.Reader) (image.Image, string, error) {
	return image.NewRGBA(image.Rect(0, 0, 1, 1)), "jpeg", nil
}

type imageValidationResult struct {
	result   string
	format   string
	duration time.Duration
	inFlight int
}

type recordingImageValidationObserver struct {
	results []imageValidationResult
}

func (observer *recordingImageValidationObserver) ObserveScheduledImageValidation(
	result string,
	format string,
	duration time.Duration,
	inFlight int,
) {
	observer.results = append(observer.results, imageValidationResult{
		result: result, format: format, duration: duration, inFlight: inFlight,
	})
}
