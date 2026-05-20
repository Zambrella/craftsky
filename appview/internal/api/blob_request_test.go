package api_test

import (
	"bytes"
	"errors"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestValidateImageBlobUpload_AllowsSupportedMIMETypes(t *testing.T) {
	t.Parallel()
	for _, mime := range []string{"image/jpeg", "image/png", "image/webp"} {
		mime := mime
		t.Run(mime, func(t *testing.T) {
			t.Parallel()
			err := api.ValidateImageBlobUpload(api.ImageBlobUploadRequest{ContentType: mime})
			if err != nil {
				t.Fatalf("ValidateImageBlobUpload(%q): %v", mime, err)
			}
		})
	}
}

func TestValidateImageBlobUpload_RejectsUnsupportedMIMEType(t *testing.T) {
	t.Parallel()
	err := api.ValidateImageBlobUpload(api.ImageBlobUploadRequest{ContentType: "image/gif"})
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if fe.Code != "validation_failed" {
		t.Fatalf("code = %q, want validation_failed", fe.Code)
	}
	if _, ok := fe.Fields["contentType"]; !ok {
		t.Fatalf("fields = %v, want contentType", fe.Fields)
	}
}

func TestValidateImageBlobUpload_RejectsEmptyMIMEType(t *testing.T) {
	t.Parallel()
	err := api.ValidateImageBlobUpload(api.ImageBlobUploadRequest{ContentType: ""})
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if fe.Code != "validation_failed" {
		t.Fatalf("code = %q, want validation_failed", fe.Code)
	}
	if _, ok := fe.Fields["contentType"]; !ok {
		t.Fatalf("fields = %v, want contentType", fe.Fields)
	}
}

func TestDecodeImageBlobUpload_AllowsBodyAt15MBLimit(t *testing.T) {
	t.Parallel()
	body := bytes.Repeat([]byte("a"), int(api.MaxImageUploadBytes))
	req, payload, err := api.DecodeImageBlobUpload("image/jpeg", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("DecodeImageBlobUpload: %v", err)
	}
	if req.SizeBytes != api.MaxImageUploadBytes {
		t.Fatalf("size = %d, want %d", req.SizeBytes, api.MaxImageUploadBytes)
	}
	if len(payload) != int(api.MaxImageUploadBytes) {
		t.Fatalf("payload len = %d, want %d", len(payload), api.MaxImageUploadBytes)
	}
}

func TestDecodeImageBlobUpload_RejectsBodyOver15MBLimit(t *testing.T) {
	t.Parallel()
	body := bytes.Repeat([]byte("a"), int(api.MaxImageUploadBytes+1))
	_, _, err := api.DecodeImageBlobUpload("image/jpeg", bytes.NewReader(body))
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if fe.Code != "validation_failed" {
		t.Fatalf("code = %q, want validation_failed", fe.Code)
	}
	if _, ok := fe.Fields["size"]; !ok {
		t.Fatalf("fields = %v, want size", fe.Fields)
	}
}

func TestDecodeImageBlobUpload_UsesConfiguredSizeLimit(t *testing.T) {
	t.Parallel()
	limits := api.MediaLimits{MaxPostImages: api.DefaultMaxPostImages, MaxImageUploadBytes: 3}
	body := bytes.Repeat([]byte("a"), 4)
	_, _, err := api.DecodeImageBlobUploadWithLimits("image/jpeg", bytes.NewReader(body), limits)
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if _, ok := fe.Fields["size"]; !ok {
		t.Fatalf("fields = %v, want size", fe.Fields)
	}
}
