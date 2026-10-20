package chat

import (
	"bytes"
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// IT-006 / FR-005, FR-006: one stable operation ID can never dispatch
// twice, including an ambiguous first response. SQL stores no private text.
func TestOperationReceiptDoesNotRepeatAmbiguousSend(t *testing.T) {
	_, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), nil)
	ledger, err := NewOperationLedger(store, 1, map[int][]byte{1: bytes.Repeat([]byte{0x64}, 32)})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	text := "synthetic-private-message-canary"
	calls := 0
	run := func(text string) (OperationReceipt, error) {
		var receipt OperationReceipt
		var operationErr error
		err := store.owners.WithExistingAuth(t.Context(), info.DID, func(ctx context.Context, _ ownerlifecycle.Lifecycle) error {
			receipt, operationErr = ledger.Execute(ctx, info.DID, id, "send", map[string]any{"conversationId": "direct-1", "text": text}, func(context.Context) (OperationResult, error) {
				calls++
				return OperationResult{}, errors.New("private-provider-canary")
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return receipt, operationErr
	}
	first, _ := run(text)
	again, err := run(text)
	if first.State != "uncertain" || again.State != "uncertain" || calls != 1 || err != nil {
		t.Fatalf("first=%+v repeated=%+v calls=%d err=%v", first, again, calls, err)
	}
	if _, err := run(text + " changed"); !errors.Is(err, ErrOperationConflict) || calls != 1 {
		t.Fatalf("changed operation dispatched: %d %v", calls, err)
	}
	var persisted string
	if err := store.pool.QueryRow(t.Context(), `SELECT row_to_json(operation)::text FROM chat_operations operation WHERE owner_did=$1 AND operation_id=$2`, info.DID, id).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains([]byte(persisted), []byte(text)) || bytes.Contains([]byte(persisted), []byte("private-provider-canary")) {
		t.Fatal("private operation content persisted")
	}
}

// IT-006 / FR-006: crash before the attempting commit proves no
// transmission, but another request still must never take over the write.
func TestExpiredPreparedOperationIsNotSubmittedAndNeverReplayed(t *testing.T) {
	_, store, info := newServiceFixture(t, true, append([]string{"atproto"}, auth.OptionalChatScopes()...), nil)
	key := bytes.Repeat([]byte{0x64}, 32)
	ledger, err := NewOperationLedger(store, 1, map[int][]byte{1: key})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	payload := map[string]any{"text": "synthetic unsent draft"}
	digest, err := operationDigest(key, info.DID, id, "send", payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_operations(owner_did,operation_id,kind,key_version,request_digest,state,created_at) VALUES($1,$2,'send',1,$3,'prepared',now()-interval '2 minutes')`, info.DID, id, digest); err != nil {
		t.Fatal(err)
	}
	var receipt OperationReceipt
	err = store.owners.WithExistingAuth(t.Context(), info.DID, func(ctx context.Context, _ ownerlifecycle.Lifecycle) error {
		var err error
		receipt, err = ledger.Execute(ctx, info.DID, id, "send", payload, func(context.Context) (OperationResult, error) {
			t.Fatal("orphaned write replayed")
			return OperationResult{}, nil
		})
		return err
	})
	if err != nil || receipt.State != "failed" || receipt.FailureCode != "notSubmitted" {
		t.Fatalf("orphan receipt=%+v err=%v", receipt, err)
	}
}

// IT-006 / IT-004: explicit rate-limit rejection is a definite failed write,
// while repeating the operation identity remains receipt-only.
func TestRateLimitedOperationIsFailedWithoutReplay(t *testing.T) {
	for _, kind := range []string{"send", "report"} {
		t.Run(kind, func(t *testing.T) {
			_, store, info := newServiceFixture(t, true, append(auth.DefaultScopes(), auth.OptionalChatScopes()...), nil)
			ledger, err := NewOperationLedger(store, 1, map[int][]byte{1: bytes.Repeat([]byte{0x64}, 32)})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			transport := NewTransport(&atclient.APIClient{Host: "https://pds.example", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"120"}}, Body: io.NopCloser(strings.NewReader(`{"error":"RateLimitExceeded","message":"private-rate-limit-canary"}`)), Request: r}, nil
			})}})
			id := uuid.New()
			method := syntax.NSID("chat.bsky.convo.sendMessage")
			if kind == "report" {
				method = "com.atproto.moderation.createReport"
			}
			var receipt OperationReceipt
			run := func() error {
				return store.owners.WithExistingAuth(t.Context(), info.DID, func(ctx context.Context, _ ownerlifecycle.Lifecycle) error {
					var callErr error
					receipt, callErr = ledger.Execute(ctx, info.DID, id, kind, map[string]any{"text": "private-operation-canary"}, func(ctx context.Context) (OperationResult, error) {
						var response map[string]any
						return OperationResult{}, transport.Procedure(ctx, method, map[string]any{"text": "private-operation-canary"}, &response)
					})
					return callErr
				})
			}
			_ = run()
			if receipt.State != "failed" || receipt.FailureCode != "rejected" {
				t.Fatalf("rate-limited receipt=%+v", receipt)
			}
			if err := run(); err != nil || receipt.State != "failed" || calls != 1 {
				t.Fatalf("replay calls=%d state=%s err=%v", calls, receipt.State, err)
			}
		})
	}
}
