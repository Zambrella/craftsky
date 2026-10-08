package imagesafety

import (
	"bytes"
	"context"
	"image"
	"io"
)

// DevelopmentScanner validates image format for local development.
type DevelopmentScanner struct{}

// ManualModerationScanner allows valid images to be served pending ordinary
// manual moderation. StateClear is a visibility result, not an automated
// content-safety verdict.
type ManualModerationScanner struct{}

func (DevelopmentScanner) Scan(ctx context.Context, input ScanInput) (ScanResult, error) {
	return ManualModerationScanner{}.Scan(ctx, input)
}

func (ManualModerationScanner) Scan(_ context.Context, input ScanInput) (ScanResult, error) {
	body, err := io.ReadAll(input.Content)
	if err != nil {
		return ScanResult{State: StateError}, err
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || canonicalImageFormat(format) != input.MIMEType {
		return ScanResult{State: StateError}, ErrBlobUnavailable
	}
	return ScanResult{State: StateClear}, nil
}
