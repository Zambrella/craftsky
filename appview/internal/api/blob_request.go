package api

import (
	"io"
	"strings"
)

const MaxImageUploadBytes int64 = 15 * 1024 * 1024

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
	fields := map[string]string{}
	contentType := canonicalContentType(req.ContentType)
	if _, ok := allowedImageUploadMIMETypes[contentType]; !ok {
		fields["contentType"] = "must be one of image/jpeg, image/png, image/webp"
	}
	if req.SizeBytes > 0 && req.SizeBytes > MaxImageUploadBytes {
		fields["size"] = "must be <= 15728640 bytes"
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
	payload, err := io.ReadAll(io.LimitReader(body, MaxImageUploadBytes+1))
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
	if err := ValidateImageBlobUpload(req); err != nil {
		return req, nil, err
	}
	return req, payload, nil
}

func canonicalContentType(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.Index(v, ";"); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}
