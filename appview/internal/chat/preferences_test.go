package chat

import (
	"net/http"
	"social.craftsky/appview/internal/auth"
	"testing"
)

// IT-011 / FR-014, RULE-005: one owner preference survives disconnect;
// changing it needs no provider write or social-audience preference update.
func TestPushPreferenceIsLocalAccountWideAndPreservedOnDisconnect(t *testing.T) {
	service, store, info := newServiceFixture(t, true, auth.DefaultScopes(), roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("local preference called provider")
		return nil, nil
	}))
	initial, err := service.PushPreference(t.Context(), info)
	if err != nil || !initial.PushEnabled {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	seedDeliveryFixture(t, store, info)
	changed, err := service.SetPushPreference(t.Context(), info, false)
	if err != nil || changed.PushEnabled {
		t.Fatalf("changed=%+v err=%v", changed, err)
	}
	var pending int
	if err := store.pool.QueryRow(t.Context(), `SELECT count(*) FROM chat_push_deliveries WHERE status IN('pending','retry','leased')`).Scan(&pending); err != nil || pending != 0 {
		t.Fatalf("disabled preference retained pending deliveries=%d err=%v", pending, err)
	}
	if _, err := service.SetConnection(t.Context(), info, false); err != nil {
		t.Fatal(err)
	}
	connection, err := store.ConnectionForChild(t.Context(), info)
	if err != nil || connection.Enabled || connection.PushEnabled {
		t.Fatalf("connection=%+v err=%v", connection, err)
	}
	if _, err := store.pool.Exec(t.Context(), `UPDATE craftsky_sessions SET lifecycle_state='revoked',revoked_at=now() WHERE token_hash=$1`, info.ChildTokenHash[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetPushPreference(t.Context(), info, true); err == nil {
		t.Fatal("revoked child changed preference")
	}
}
