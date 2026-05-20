package api

import (
	"io"
	"strconv"
	"strings"
)

var allowedImageUploadMIMETypes = map[string]struct{}{
	"image/jpeg": {},
	"image/png":  {},
	"image/webp": {},
}

// ImageBlobUploadRequest is the validated shape used by the upload handler.
type ImageBlobUploadRequest struct {
	ContentType string
	SizeBytes   int64
}

// ValidateImageBlobUpload checks MIME and (when provided) size bounds.
func ValidateImageBlobUpload(req ImageBlobUploadRequest) error {
	return ValidateImageBlobUploadWithLimits(req, DefaultMediaLimits())
}

// ValidateImageBlobUploadWithLimits checks MIME and deployment-specific size bounds.
func ValidateImageBlobUploadWithLimits(req ImageBlobUploadRequest, limits MediaLimits) error {
	limits = normalizeMediaLimits(limits)
	fields := map[string]string{}
	contentType := canonicalContentType(req.ContentType)
	if _, ok := allowedImageUploadMIMETypes[contentType]; !ok {
		fields["contentType"] = "must be one of image/jpeg, image/png, image/webp"
	}
	if req.SizeBytes > 0 && req.SizeBytes > limits.MaxImageUploadBytes {
		fields["size"] = "must be <= " + formatInt64(limits.MaxImageUploadBytes) + " bytes"
	}
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

// DecodeImageBlobUpload reads and bounds an upload request body.
// It reads at most 15MB + 1 byte so oversized uploads can be detected
// deterministically and rejected before downstream forwarding.
func DecodeImageBlobUpload(contentType string, body io.Reader) (ImageBlobUploadRequest, []byte, error) {
	return DecodeImageBlobUploadWithLimits(contentType, body, DefaultMediaLimits())
}

// DecodeImageBlobUploadWithLimits reads and bounds an upload request body.
func DecodeImageBlobUploadWithLimits(contentType string, body io.Reader, limits MediaLimits) (ImageBlobUploadRequest, []byte, error) {
	limits = normalizeMediaLimits(limits)
	payload, err := io.ReadAll(io.LimitReader(body, limits.MaxImageUploadBytes+1))
	if err != nil {
		return ImageBlobUploadRequest{}, nil, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	req := ImageBlobUploadRequest{
		ContentType: canonicalContentType(contentType),
		SizeBytes:   int64(len(payload)),
	}
	if err := ValidateImageBlobUploadWithLimits(req, limits); err != nil {
		return req, nil, err
	}
	return req, payload, nil
}

func formatInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}

func canonicalContentType(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.Index(v, ";"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
