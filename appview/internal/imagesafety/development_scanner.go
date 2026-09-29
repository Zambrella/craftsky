package imagesafety

import (
	"bytes"
	"context"
	"image"
	"io"
)

// DevelopmentScanner is a deterministic local-only scanner. The validated
// configuration prevents this implementation from marking production content
// clear.
type DevelopmentScanner struct{}

func (DevelopmentScanner) Scan(_ context.Context, input ScanInput) (ScanResult, error) {
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
