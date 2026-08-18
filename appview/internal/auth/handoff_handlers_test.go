package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/ctxkeys"
)

type fakeHandoffCoordinator struct {
	exchangeResult HandoffExchangeResult
	exchangeErr    error
	confirmErr     error
	exchangeCode   string
	exchangeDevice string
	confirmToken   string
	confirmReceipt uuid.UUID
	confirmDevice  string
}

func (fake *fakeHandoffCoordinator) CreateExchange(_ context.Context, _ CallbackAttempt, _ syntax.Handle, _ string) (string, error) {
	return "callback-code", nil
}

func (fake *fakeHandoffCoordinator) Exchange(_ context.Context, code, deviceID string) (HandoffExchangeResult, error) {
	fake.exchangeCode = code
	fake.exchangeDevice = deviceID
	return fake.exchangeResult, fake.exchangeErr
}

func (fake *fakeHandoffCoordinator) Confirm(_ context.Context, token string, receiptID uuid.UUID, deviceID string) error {
	fake.confirmToken = token
	fake.confirmReceipt = receiptID
	fake.confirmDevice = deviceID
	return fake.confirmErr
}

func handoffHandlerRequest(method, target, body, deviceID, token string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	ctx := ctxkeys.WithDeviceID(request.Context(), deviceID)
	return request.WithContext(ctx)
}

func TestHandoffExchangeHandlerReturnsPendingCredentialWithoutEchoingCode(t *testing.T) {
	receiptID := uuid.MustParse("00000000-0000-4000-8000-000000000811")
	confirmBy := time.Date(2026, 8, 14, 12, 5, 0, 0, time.UTC)
	fake := &fakeHandoffCoordinator{exchangeResult: HandoffExchangeResult{
		Token: "pending-bearer", DID: syntax.DID("did:plc:alice"),
		Handle: syntax.Handle("alice.example"), ReceiptID: receiptID, ConfirmBy: confirmBy,
	}}
	handlers := &HTTPHandlers{Handoffs: fake, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	recorder := httptest.NewRecorder()

	handlers.HandoffExchangeHandler().ServeHTTP(recorder, handoffHandlerRequest(
		http.MethodPost, "/v1/auth/handoffs/exchange", `{"code":"browser-secret-code"}`, "installation-a", "",
	))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if fake.exchangeCode != "browser-secret-code" || fake.exchangeDevice != "installation-a" {
		t.Fatalf("exchange arguments = %q %q", fake.exchangeCode, fake.exchangeDevice)
	}
	if strings.Contains(recorder.Body.String(), "browser-secret-code") {
		t.Fatal("response echoed the exchange code")
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["token"] != "pending-bearer" || body["did"] != "did:plc:alice" ||
		body["handle"] != "alice.example" || body["receiptId"] != receiptID.String() {
		t.Fatalf("response=%v", body)
	}
}

func TestHandoffConfirmHandlerUsesPendingBearerAndIsIdempotent(t *testing.T) {
	receiptID := uuid.MustParse("00000000-0000-4000-8000-000000000812")
	fake := &fakeHandoffCoordinator{}
	handlers := &HTTPHandlers{Handoffs: fake, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	recorder := httptest.NewRecorder()

	handlers.HandoffConfirmHandler().ServeHTTP(recorder, handoffHandlerRequest(
		http.MethodPost, "/v1/auth/handoffs/confirm", `{"receiptId":"`+receiptID.String()+`"}`,
		"installation-a", "pending-bearer",
	))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if fake.confirmToken != "pending-bearer" || fake.confirmReceipt != receiptID || fake.confirmDevice != "installation-a" {
		t.Fatalf("confirm arguments = %q %s %q", fake.confirmToken, fake.confirmReceipt, fake.confirmDevice)
	}
}

func TestHandoffHandlersCollapseInvalidSecretsAndClassifyInfrastructure(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid", err: ErrHandoffInvalid, wantStatus: http.StatusBadRequest, wantCode: "invalid_handoff"},
		{name: "database", err: errors.New("database unavailable"), wantStatus: http.StatusServiceUnavailable, wantCode: "handoff_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeHandoffCoordinator{exchangeErr: test.err}
			handlers := &HTTPHandlers{Handoffs: fake, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			recorder := httptest.NewRecorder()
			handlers.HandoffExchangeHandler().ServeHTTP(recorder, handoffHandlerRequest(
				http.MethodPost, "/v1/auth/handoffs/exchange", `{"code":"secret"}`, "installation-a", "",
			))
			if recorder.Code != test.wantStatus || !strings.Contains(recorder.Body.String(), `"error":"`+test.wantCode+`"`) {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}
