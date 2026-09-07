package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

func TestPDSSessionExpiredEnvelopeRedactsJoinedMigrationError(t *testing.T) {
	const secret = `Bearer access-canary refresh-canary dpop-canary {"accessToken":"session-json-canary"}`
	recorder := httptest.NewRecorder()
	writePDSError(
		recorder, http.StatusBadGateway, "pds_failed", "PDS request failed", "request-migration-17",
		errors.Join(auth.ErrPDSSessionExpired, errors.New(secret)),
	)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), secret) || strings.Contains(recorder.Body.String(), "access-canary") {
		t.Fatalf("migration secret escaped into API error: %s", recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode API error: %v", err)
	}
	if len(body) != 3 || body["error"] != "pds_session_expired" || body["requestId"] != "request-migration-17" {
		t.Fatalf("migration API envelope=%#v", body)
	}
}
