package api

import (
	"context"
	"encoding/json"
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
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
)

func TestInstagramVerificationHandlersExactWireContract(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:synthetic-alice")
	id := uuid.MustParse("00000000-0000-0000-0000-000000000101")
	expires := time.Date(2026, 7, 19, 12, 10, 0, 0, time.UTC)
	service := &stubInstagramVerificationService{
		created: instagram.CreatedVerification{
			Attempt:   instagram.VerificationAttempt{ID: id, OwnerDID: alice, State: instagram.AttemptPendingDM, ExpiresAt: expires},
			Challenge: "CSKY-2345-6789-ABCD-E",
			DMURL:     "https://www.instagram.com/direct/t/synthetic",
		},
		attempt: &instagram.VerificationAttempt{
			ID:                id,
			OwnerDID:          alice,
			State:             instagram.AttemptPendingConfirmation,
			ExpiresAt:         expires,
			CandidateUsername: "synthetic.candidate",
		},
		confirmation: instagram.ConfirmationResult{
			State: instagram.AttemptConfirmed,
			Account: instagram.AccountView{
				State:        instagram.LinkActive,
				Username:     "synthetic.candidate",
				Discoverable: true,
				VerifiedAt:   time.Date(2026, 7, 19, 12, 2, 0, 0, time.UTC),
			},
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	createReq := authenticatedInstagramRequest(http.MethodPost, "/v1/migrations/instagram/verifications", "{}", alice)
	createRR := httptest.NewRecorder()
	CreateInstagramVerificationHandler(service, logger).ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", createRR.Code, createRR.Body.String())
	}
	var create map[string]any
	if err := json.Unmarshal(createRR.Body.Bytes(), &create); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if create["verificationId"] != id.String() || create["state"] != "pendingDm" || create["challenge"] != "CSKY-2345-6789-ABCD-E" || create["expiresAt"] != expires.Format(time.RFC3339) || create["dmUrl"] != service.created.DMURL {
		t.Fatalf("create response = %#v", create)
	}
	if len(create) != 5 {
		t.Fatalf("create response has unexpected fields: %#v", create)
	}

	getReq := authenticatedInstagramRequest(http.MethodGet, "/v1/migrations/instagram/verifications/"+id.String(), "", alice)
	getReq.SetPathValue("verificationId", id.String())
	getRR := httptest.NewRecorder()
	GetInstagramVerificationHandler(service, logger).ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get status = %d; body=%s", getRR.Code, getRR.Body.String())
	}
	var get map[string]any
	if err := json.Unmarshal(getRR.Body.Bytes(), &get); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if get["candidateUsername"] != "synthetic.candidate" || get["state"] != "pendingConfirmation" {
		t.Fatalf("get response = %#v", get)
	}
	if _, exists := get["challenge"]; exists {
		t.Fatal("status response exposed challenge")
	}

	confirmReq := authenticatedInstagramRequest(http.MethodPost, "/v1/migrations/instagram/verifications/"+id.String()+"/confirm", `{"discoverable":true}`, alice)
	confirmReq.SetPathValue("verificationId", id.String())
	confirmRR := httptest.NewRecorder()
	ConfirmInstagramVerificationHandler(service, logger).ServeHTTP(confirmRR, confirmReq)
	if confirmRR.Code != http.StatusOK {
		t.Fatalf("confirm status = %d; body=%s", confirmRR.Code, confirmRR.Body.String())
	}
	var confirm map[string]any
	if err := json.Unmarshal(confirmRR.Body.Bytes(), &confirm); err != nil {
		t.Fatalf("decode confirm: %v", err)
	}
	if confirm["state"] != "confirmed" {
		t.Fatalf("confirm response = %#v", confirm)
	}
	account, ok := confirm["account"].(map[string]any)
	if !ok || account["username"] != "synthetic.candidate" || account["discoverable"] != true || account["state"] != "active" {
		t.Fatalf("confirm account = %#v", confirm["account"])
	}
	if len(service.confirmCalls) != 1 || !service.confirmCalls[0].discoverable || service.confirmCalls[0].owner != alice {
		t.Fatalf("confirm calls = %+v", service.confirmCalls)
	}
}

func TestInstagramVerificationDeleteIsPermanentPrivacyNoOp(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:synthetic-alice")
	service := &stubInstagramVerificationService{}
	handler := DeleteInstagramVerificationHandler(service, slog.Default())
	for _, rawID := range []string{
		"00000000-0000-0000-0000-000000000102",
		"not-an-opaque-uuid",
		"",
	} {
		req := authenticatedInstagramRequest(http.MethodDelete, "/v1/migrations/instagram/verifications/"+rawID, "", alice)
		req.SetPathValue("verificationId", rawID)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent || rr.Body.Len() != 0 {
			t.Errorf("DELETE %q status=%d body=%q, want empty 204", rawID, rr.Code, rr.Body.String())
		}
	}
	if len(service.cancelCalls) != 1 {
		t.Fatalf("cancel calls = %d, want only valid ID", len(service.cancelCalls))
	}
}

func TestInstagramVerificationHandlersRejectInvalidAndMapSafeErrors(t *testing.T) {
	t.Parallel()

	alice := syntax.DID("did:plc:synthetic-alice")
	id := uuid.MustParse("00000000-0000-0000-0000-000000000103")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name    string
		handler func(InstagramVerificationService, *slog.Logger) http.Handler
		method  string
		body    string
		err     error
		status  int
		code    string
	}{
		{name: "create unknown field", handler: CreateInstagramVerificationHandler, method: http.MethodPost, body: `{"unexpected":"synthetic-private-canary"}`, status: 400, code: "invalid_request"},
		{name: "create unavailable", handler: CreateInstagramVerificationHandler, method: http.MethodPost, body: `{}`, err: instagram.ErrVerificationUnavailable, status: 503, code: "instagram_verification_unavailable"},
		{name: "get foreign", handler: GetInstagramVerificationHandler, method: http.MethodGet, err: instagram.ErrInstagramResourceNotFound, status: 404, code: "instagram_verification_not_found"},
		{name: "confirm state conflict", handler: ConfirmInstagramVerificationHandler, method: http.MethodPost, body: `{"discoverable":false}`, err: instagram.ErrInstagramStateTransition, status: 409, code: "instagram_verification_state_conflict"},
		{name: "confirm malformed", handler: ConfirmInstagramVerificationHandler, method: http.MethodPost, body: `{"discoverable":"yes"}`, status: 400, code: "invalid_request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &stubInstagramVerificationService{err: tt.err}
			req := authenticatedInstagramRequest(tt.method, "/v1/migrations/instagram/verifications/"+id.String(), tt.body, alice)
			req.SetPathValue("verificationId", id.String())
			rr := httptest.NewRecorder()
			tt.handler(service, logger).ServeHTTP(rr, req)
			if rr.Code != tt.status {
				t.Fatalf("status = %d, want %d; body=%s", rr.Code, tt.status, rr.Body.String())
			}
			var body envelope.Error
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if body.Error != tt.code {
				t.Fatalf("code = %q, want %q", body.Error, tt.code)
			}
			if strings.Contains(rr.Body.String(), "synthetic-private-canary") {
				t.Fatal("invalid request echoed a private input canary")
			}
		})
	}
}

func authenticatedInstagramRequest(method, target, body string, did syntax.DID) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	return req.WithContext(middleware.WithDID(req.Context(), did))
}

type stubInstagramVerificationService struct {
	created      instagram.CreatedVerification
	attempt      *instagram.VerificationAttempt
	confirmation instagram.ConfirmationResult
	err          error
	cancelCalls  []uuid.UUID
	confirmCalls []struct {
		owner        syntax.DID
		id           uuid.UUID
		discoverable bool
	}
}

func (s *stubInstagramVerificationService) CreateVerification(context.Context, syntax.DID) (instagram.CreatedVerification, error) {
	return s.created, s.err
}

func (s *stubInstagramVerificationService) GetVerification(context.Context, syntax.DID, uuid.UUID) (*instagram.VerificationAttempt, error) {
	return s.attempt, s.err
}

func (s *stubInstagramVerificationService) CancelVerification(_ context.Context, _ syntax.DID, id uuid.UUID) error {
	s.cancelCalls = append(s.cancelCalls, id)
	return s.err
}

func (s *stubInstagramVerificationService) ConfirmVerification(_ context.Context, owner syntax.DID, id uuid.UUID, discoverable bool) (instagram.ConfirmationResult, error) {
	s.confirmCalls = append(s.confirmCalls, struct {
		owner        syntax.DID
		id           uuid.UUID
		discoverable bool
	}{owner: owner, id: id, discoverable: discoverable})
	return s.confirmation, s.err
}

var _ InstagramVerificationService = (*stubInstagramVerificationService)(nil)
