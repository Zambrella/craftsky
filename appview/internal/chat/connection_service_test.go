package chat

import (
	"errors"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

// IT-005 / FR-001, FR-016: disconnect remains possible after optional
// permission loss; re-enable needs current full consent and service access.
func TestConnectionToggleRequiresCurrentChildAndOptionalAuthority(t *testing.T) {
	for _, test := range []struct {
		name                               string
		initial, enable, capable, disabled bool
		wantErr                            bool
	}{
		{"disconnect after grant loss", true, false, false, false, false},
		{"enable capable", false, true, true, false, false},
		{"enable requires consent", false, true, false, false, true},
		{"restricted stays disconnected", false, true, true, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			scopes := auth.DefaultScopes()
			if test.capable {
				scopes = append(scopes, auth.OptionalChatScopes()...)
			}
			calls := 0
			service, store, info := newServiceFixture(t, test.initial, scopes, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				body := `{"chatDisabled":false}`
				if test.disabled {
					body = `{"chatDisabled":true}`
				}
				if r.URL.Path != "/xrpc/chat.bsky.actor.getStatus" {
					t.Fatal("connection changed shared policy or history")
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			}))
			_, err := service.SetConnection(t.Context(), info, test.enable)
			if (err != nil) != test.wantErr {
				t.Fatalf("connection error=%v", err)
			}
			connection, loadErr := store.Connection(t.Context(), info.DID)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if connection.Enabled != (test.enable && !test.wantErr) {
				t.Fatal("incorrect durable enablement")
			}
			if !test.enable && calls != 0 {
				t.Fatal("disconnect contacted provider")
			}
			if test.enable && !test.capable && !errors.Is(err, auth.ErrPDSPermissionsRequired) {
				t.Fatal("missing optional permission not classified")
			}
		})
	}
}

// IT-005 / FR-016, FR-013: reauthorization resumes saved progress promptly
// without clearing lifetime notification policy or starting a fresh baseline.
func TestReconnectResumesPausedCursorWithoutRebaselining(t *testing.T) {
	service, store, info := newServiceFixture(t, false, append([]string{"atproto"}, auth.OptionalChatScopes()...), roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/xrpc/chat.bsky.actor.getStatus" {
			t.Fatal("reconnect fetched or replaced history")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"chatDisabled":false}`)), Request: r}, nil
	}))
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_sync_state(owner_did,phase,cursor,safe_state,next_due_at,failure_count) VALUES($1,'ready','retained-cursor','authorizationNeeded',now()+interval '5 minutes',3)`, info.DID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetConnection(t.Context(), info, true); err != nil {
		t.Fatal(err)
	}
	var resumed, recovering bool
	var phase, cursor string
	if err := store.pool.QueryRow(t.Context(), `SELECT next_due_at<=now(),failure_count=0 AND recovery_id IS NOT NULL,phase,cursor FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&resumed, &recovering, &phase, &cursor); err != nil {
		t.Fatal(err)
	}
	if !resumed || !recovering || phase != "ready" || cursor != "retained-cursor" {
		t.Fatalf("resumed=%v recovery=%v phase=%s", resumed, recovering, phase)
	}
}
