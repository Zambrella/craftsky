package api

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/scheduledposts"
)

func TestScheduledMediaHandlersKeepBytesOwnerPrivate(t *testing.T) {
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000701")
	alice := syntax.DID("did:plc:alice")
	validJPEG := validJPEGBytes(t)
	service := &fakeScheduledMediaService{
		owner: alice,
		media: scheduledposts.PrivateMedia{
			ID: mediaID, OwnerDID: alice,
			ObjectKey: "scheduled-media/private-object-key-canary",
			State:     "ready", MIMEType: "image/jpeg", SizeBytes: int64(len(validJPEG)),
			BlobCID: syntax.CID("bafk-private"),
		},
		body: validJPEG,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("PUT returns only safe media metadata", func(t *testing.T) {
		handler := PutScheduledMediaHandler(service, DefaultMediaLimits(), logger)
		request := httptest.NewRequest(http.MethodPut, "/v1/scheduled-post-media/"+mediaID.String(), bytes.NewReader(service.body))
		request.SetPathValue("mediaId", mediaID.String())
		request.Header.Set("Content-Type", "image/jpeg")
		request = request.WithContext(middleware.WithDID(request.Context(), alice))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("PUT status=%d body=%s", response.Code, response.Body.String())
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode PUT response: %v", err)
		}
		for _, key := range []string{"id", "state", "mimeType", "sizeBytes", "blobCid"} {
			if _, ok := body[key]; !ok {
				t.Errorf("PUT response missing %s: %v", key, body)
			}
		}
		if strings.Contains(response.Body.String(), "object-key-canary") {
			t.Fatal("PUT response exposed the private object key")
		}
	})

	t.Run("GET is private and owner scoped", func(t *testing.T) {
		handler := GetScheduledMediaHandler(service, logger)
		request := httptest.NewRequest(http.MethodGet, "/v1/scheduled-post-media/"+mediaID.String(), nil)
		request.SetPathValue("mediaId", mediaID.String())
		request = request.WithContext(middleware.WithDID(request.Context(), alice))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), service.body) {
			t.Fatalf("GET status/body=%d/%q", response.Code, response.Body.Bytes())
		}
		if response.Header().Get("Cache-Control") != "private, no-store" ||
			response.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("GET privacy headers=%v", response.Header())
		}

		foreign := httptest.NewRequest(http.MethodGet, "/v1/scheduled-post-media/"+mediaID.String(), nil)
		foreign.SetPathValue("mediaId", mediaID.String())
		foreign = foreign.WithContext(middleware.WithDID(foreign.Context(), syntax.DID("did:plc:bob")))
		foreignResponse := httptest.NewRecorder()
		handler.ServeHTTP(foreignResponse, foreign)
		if foreignResponse.Code != http.StatusNotFound {
			t.Fatalf("foreign GET status=%d body=%s", foreignResponse.Code, foreignResponse.Body.String())
		}
		var errorBody envelope.Error
		if err := json.Unmarshal(foreignResponse.Body.Bytes(), &errorBody); err != nil || errorBody.Error != "scheduled_media_not_found" {
			t.Fatalf("foreign GET envelope=%+v err=%v", errorBody, err)
		}
	})

	t.Run("DELETE is owner scoped", func(t *testing.T) {
		handler := DeleteScheduledMediaHandler(service, func() time.Time {
			return time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
		}, logger)
		for attempt := 1; attempt <= 2; attempt++ {
			request := httptest.NewRequest(http.MethodDelete, "/v1/scheduled-post-media/"+mediaID.String(), nil)
			request.SetPathValue("mediaId", mediaID.String())
			request = request.WithContext(middleware.WithDID(request.Context(), alice))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent {
				t.Fatalf("DELETE attempt %d status=%d body=%s", attempt, response.Code, response.Body.String())
			}
		}
		if service.deleteCalls != 2 {
			t.Fatalf("DELETE service calls=%d, want 2", service.deleteCalls)
		}
	})
}

func validJPEGBytes(t *testing.T) []byte {
	t.Helper()
	value := image.NewRGBA(image.Rect(0, 0, 1, 1))
	value.Set(0, 0, color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff})
	var buffer bytes.Buffer
	if err := jpeg.Encode(&buffer, value, nil); err != nil {
		t.Fatalf("encode valid JPEG fixture: %v", err)
	}
	return buffer.Bytes()
}

func TestScheduledMediaPutRejectsInvalidImageBodies(t *testing.T) {
	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000702")
	alice := syntax.DID("did:plc:alice")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cases := []struct {
		name        string
		contentType string
		body        []byte
		limits      MediaLimits
	}{
		{name: "empty", contentType: "image/jpeg"},
		{name: "malformed image", contentType: "image/jpeg", body: []byte("not-an-image")},
		{name: "MIME mismatch", contentType: "image/jpeg", body: []byte("\x89PNG\r\n\x1a\n")},
		{name: "truncated JPEG", contentType: "image/jpeg", body: []byte{0xff, 0xd8, 0xff, 0xdb}},
		{name: "truncated PNG", contentType: "image/png", body: []byte("\x89PNG\r\n\x1a\n")},
		{name: "truncated WebP", contentType: "image/webp", body: []byte("RIFF\x00\x00\x00\x00WEBP")},
		{
			name: "configured oversize", contentType: "image/jpeg",
			body:   []byte{0xff, 0xd8, 0xff, 0xdb},
			limits: MediaLimits{MaxPostImages: DefaultMaxPostImages, MaxImageUploadBytes: 3},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			service := &fakeScheduledMediaService{owner: alice}
			handler := PutScheduledMediaHandler(service, testCase.limits, logger)
			request := httptest.NewRequest(
				http.MethodPut,
				"/v1/scheduled-post-media/"+mediaID.String(),
				bytes.NewReader(testCase.body),
			)
			request.SetPathValue("mediaId", mediaID.String())
			request.Header.Set("Content-Type", testCase.contentType)
			request = request.WithContext(middleware.WithDID(request.Context(), alice))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("PUT status=%d body=%s", response.Code, response.Body.String())
			}
			var errorBody envelope.Error
			if err := json.Unmarshal(response.Body.Bytes(), &errorBody); err != nil ||
				errorBody.Error != "scheduled_media_invalid" {
				t.Fatalf("PUT envelope=%+v err=%v", errorBody, err)
			}
			if service.putCalls != 0 {
				t.Fatalf("service Put calls=%d, want 0", service.putCalls)
			}
		})
	}
}

type fakeScheduledMediaService struct {
	owner       syntax.DID
	media       scheduledposts.PrivateMedia
	body        []byte
	putCalls    int
	deleteCalls int
}

func (s *fakeScheduledMediaService) Put(_ context.Context, params scheduledposts.PutPrivateMediaParams) (scheduledposts.PrivateMedia, error) {
	s.putCalls++
	if params.OwnerDID != s.owner {
		return scheduledposts.PrivateMedia{}, scheduledposts.ErrScheduledMediaNotFound
	}
	return s.media, nil
}

func (s *fakeScheduledMediaService) Open(_ context.Context, owner syntax.DID, _ uuid.UUID) (scheduledposts.OpenedPrivateMedia, error) {
	if owner != s.owner {
		return scheduledposts.OpenedPrivateMedia{}, scheduledposts.ErrScheduledMediaNotFound
	}
	return scheduledposts.OpenedPrivateMedia{
		MIMEType: s.media.MIMEType, SizeBytes: int64(len(s.body)),
		Body: io.NopCloser(bytes.NewReader(s.body)),
	}, nil
}

func (s *fakeScheduledMediaService) Delete(_ context.Context, owner syntax.DID, _ uuid.UUID, _ time.Time) error {
	s.deleteCalls++
	if owner != s.owner {
		return scheduledposts.ErrScheduledMediaNotFound
	}
	return nil
}
