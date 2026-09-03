package api

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"time"

	_ "golang.org/x/image/webp"
)

const (
	HardMaxImageWidth                    = 4_000
	HardMaxImageHeight                   = 4_000
	HardMaxImagePixels            uint64 = 16_000_000
	HardMaxImageAspectRatio              = 20
	HardMaxConcurrentImageDecodes        = 1
	hardMaxImageAdmissionWait            = time.Second
)

var (
	ErrScheduledImageInvalid = errors.New("scheduled image is invalid")
	ErrImageDecodeSaturated  = errors.New("scheduled image decoder is saturated")
)

// ImageDecodeLimits bounds both one decoded image and aggregate decoder work.
// All fields are required; zero never disables a limit.
type ImageDecodeLimits struct {
	MaxWidth             int
	MaxHeight            int
	MaxPixels            uint64
	MaxAspectRatio       int
	MaxConcurrentDecodes int
	AdmissionWait        time.Duration
}

func DefaultImageDecodeLimits() ImageDecodeLimits {
	return ImageDecodeLimits{
		MaxWidth:             HardMaxImageWidth,
		MaxHeight:            HardMaxImageHeight,
		MaxPixels:            HardMaxImagePixels,
		MaxAspectRatio:       HardMaxImageAspectRatio,
		MaxConcurrentDecodes: HardMaxConcurrentImageDecodes,
		AdmissionWait:        250 * time.Millisecond,
	}
}

func (limits ImageDecodeLimits) Validate() error {
	switch {
	case limits.MaxWidth <= 0 || limits.MaxWidth > HardMaxImageWidth:
		return errors.New("image decode maximum width is invalid")
	case limits.MaxHeight <= 0 || limits.MaxHeight > HardMaxImageHeight:
		return errors.New("image decode maximum height is invalid")
	case limits.MaxPixels == 0 || limits.MaxPixels > HardMaxImagePixels:
		return errors.New("image decode maximum pixels is invalid")
	case limits.MaxAspectRatio <= 0 || limits.MaxAspectRatio > HardMaxImageAspectRatio:
		return errors.New("image decode maximum aspect ratio is invalid")
	case limits.MaxConcurrentDecodes <= 0 || limits.MaxConcurrentDecodes > HardMaxConcurrentImageDecodes:
		return errors.New("image decode concurrency is invalid")
	case limits.AdmissionWait <= 0 || limits.AdmissionWait > hardMaxImageAdmissionWait:
		return errors.New("image decode admission wait is invalid")
	default:
		return nil
	}
}

type ValidatedScheduledImage struct {
	Format string
	Width  int
	Height int
}

type ImageValidator interface {
	Validate(context.Context, string, []byte) (ValidatedScheduledImage, error)
}

type ImageValidationObserver interface {
	ObserveScheduledImageValidation(result, format string, duration time.Duration, inFlight int)
}

type noopImageValidationObserver struct{}

func (noopImageValidationObserver) ObserveScheduledImageValidation(
	string,
	string,
	time.Duration,
	int,
) {
}

type imageDecoder interface {
	DecodeConfig(io.Reader) (image.Config, string, error)
	Decode(io.Reader) (image.Image, string, error)
}

type registeredImageDecoder struct{}

func (registeredImageDecoder) DecodeConfig(reader io.Reader) (image.Config, string, error) {
	return image.DecodeConfig(reader)
}

func (registeredImageDecoder) Decode(reader io.Reader) (image.Image, string, error) {
	return image.Decode(reader)
}

type boundedImageValidator struct {
	limits   ImageDecodeLimits
	decoder  imageDecoder
	observer ImageValidationObserver
	now      func() time.Time
	permits  chan struct{}
	waiters  chan struct{}
}

func NewImageValidator(limits ImageDecodeLimits) (ImageValidator, error) {
	return newImageValidator(limits, registeredImageDecoder{}, nil, time.Now)
}

func NewImageValidatorWithObserver(
	limits ImageDecodeLimits,
	observer ImageValidationObserver,
) (ImageValidator, error) {
	return newImageValidator(limits, registeredImageDecoder{}, observer, time.Now)
}

func newImageValidator(
	limits ImageDecodeLimits,
	decoder imageDecoder,
	observer ImageValidationObserver,
	now func() time.Time,
) (*boundedImageValidator, error) {
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	if decoder == nil {
		return nil, errors.New("image decoder is required")
	}
	if now == nil {
		return nil, errors.New("image validator clock is required")
	}
	if observer == nil {
		observer = noopImageValidationObserver{}
	}
	return &boundedImageValidator{
		limits:   limits,
		decoder:  decoder,
		observer: observer,
		now:      now,
		permits:  make(chan struct{}, limits.MaxConcurrentDecodes),
		waiters:  make(chan struct{}, limits.MaxConcurrentDecodes),
	}, nil
}

func (validator *boundedImageValidator) Validate(
	ctx context.Context,
	declaredMIME string,
	payload []byte,
) (validated ValidatedScheduledImage, err error) {
	started := validator.now()
	if err := ctx.Err(); err != nil {
		validator.observeCompletion("cancelled", "unknown", started)
		return ValidatedScheduledImage{}, err
	}
	if err := validator.acquire(ctx); err != nil {
		validator.observeCompletion(imageValidationResultClass(err), "unknown", started)
		return ValidatedScheduledImage{}, err
	}
	validator.emit("started", "unknown", 0, len(validator.permits))
	defer func() {
		result := imageValidationResultClass(err)
		format := "unknown"
		if err == nil {
			result = "success"
			format = safeImageFormat(validated.Format)
		}
		validator.release()
		validator.observeCompletion(result, format, started)
	}()
	return validator.validateWithDecoder(declaredMIME, payload)
}

func (validator *boundedImageValidator) observeCompletion(
	result string,
	format string,
	started time.Time,
) {
	duration := validator.now().Sub(started)
	if duration < 0 {
		duration = 0
	}
	validator.emit(result, format, duration, len(validator.permits))
}

func (validator *boundedImageValidator) emit(
	result string,
	format string,
	duration time.Duration,
	inFlight int,
) {
	defer func() { _ = recover() }()
	validator.observer.ObserveScheduledImageValidation(
		result,
		safeImageFormat(format),
		duration,
		inFlight,
	)
}

func imageValidationResultClass(err error) string {
	switch {
	case err == nil:
		return "success"
	case errors.Is(err, ErrImageDecodeSaturated):
		return "saturated"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "cancelled"
	default:
		return "invalid"
	}
}

func safeImageFormat(format string) string {
	switch format {
	case "jpeg", "png", "webp":
		return format
	default:
		return "unknown"
	}
}

func (validator *boundedImageValidator) validateWithDecoder(
	declaredMIME string,
	payload []byte,
) (validated ValidatedScheduledImage, err error) {
	defer func() {
		if recover() != nil {
			validated = ValidatedScheduledImage{}
			err = ErrScheduledImageInvalid
		}
	}()
	config, format, err := validator.decoder.DecodeConfig(bytes.NewReader(payload))
	configuredMIME := canonicalImageMIME(format)
	if err != nil || configuredMIME == "" ||
		configuredMIME != canonicalContentType(declaredMIME) ||
		!imageGeometryAllowed(config.Width, config.Height, validator.limits) {
		return ValidatedScheduledImage{}, ErrScheduledImageInvalid
	}
	decoded, decodedFormat, err := validator.decoder.Decode(bytes.NewReader(payload))
	decodedMIME := canonicalImageMIME(decodedFormat)
	if err != nil || decoded == nil || decodedFormat != format || decodedMIME == "" ||
		decodedMIME != canonicalContentType(declaredMIME) {
		return ValidatedScheduledImage{}, ErrScheduledImageInvalid
	}
	decodedWidth := decoded.Bounds().Dx()
	decodedHeight := decoded.Bounds().Dy()
	if decodedWidth != config.Width || decodedHeight != config.Height ||
		!imageGeometryAllowed(decodedWidth, decodedHeight, validator.limits) {
		return ValidatedScheduledImage{}, ErrScheduledImageInvalid
	}
	return ValidatedScheduledImage{
		Format: format,
		Width:  config.Width,
		Height: config.Height,
	}, nil
}

func (validator *boundedImageValidator) acquire(ctx context.Context) error {
	select {
	case validator.permits <- struct{}{}:
		return nil
	default:
	}

	select {
	case validator.waiters <- struct{}{}:
		defer func() { <-validator.waiters }()
	default:
		return ErrImageDecodeSaturated
	}

	timer := time.NewTimer(validator.limits.AdmissionWait)
	defer timer.Stop()
	select {
	case validator.permits <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ErrImageDecodeSaturated
	}
}

func (validator *boundedImageValidator) release() {
	<-validator.permits
}

func canonicalImageMIME(format string) string {
	return map[string]string{
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"webp": "image/webp",
	}[format]
}

func imageGeometryAllowed(width, height int, limits ImageDecodeLimits) bool {
	if width <= 0 || height <= 0 || width > limits.MaxWidth || height > limits.MaxHeight {
		return false
	}
	if uint64(width) > limits.MaxPixels/uint64(height) {
		return false
	}
	longSide, shortSide := width, height
	if height > width {
		longSide, shortSide = height, width
	}
	return longSide <= shortSide*limits.MaxAspectRatio
}
