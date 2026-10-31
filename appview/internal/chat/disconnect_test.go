package chat

import (
	"net/http"
	"social.craftsky/appview/internal/auth"
	"testing"
)

// IT-005 / FR-016: disconnect removes ephemeral authority and cancels work,
// retaining minimal resume/operation tombstones rather than replaying history.
func TestDisconnectCancelsWorkAndRetainsResumeAndOperationTombstones(t *testing.T) {
	service, store, info := newServiceFixture(t, true, auth.DefaultScopes(), roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("disconnect called upstream"); return nil, nil }))
	for _, sql := range []string{
		`INSERT INTO chat_sync_state(owner_did,phase,cursor,version,claim_token,claim_expires_at) VALUES($1,'ready','resume-private-cursor',5,gen_random_uuid(),now()+interval '1 minute')`,
		`INSERT INTO chat_notifications(owner_did,id,logical_key,kind,conversation_id,messaging_generation,open_reference) VALUES($1,gen_random_uuid(),'pending-before-disconnect','normal','direct-1',2,gen_random_uuid())`,
		`INSERT INTO chat_operations(owner_did,operation_id,kind,key_version,request_digest,state) VALUES($1,gen_random_uuid(),'send',1,decode(repeat('01',32),'hex'),'uncertain')`,
	} {
		if _, err := store.pool.Exec(t.Context(), sql, info.DID); err != nil {
			t.Fatal(err)
		}
	}
	seedDeliveryFixture(t, store, info)
	if _, err := store.pool.Exec(t.Context(), `INSERT INTO chat_view_activity(owner_did,child_token_hash,oauth_session_id,view,conversation_id,expires_at) VALUES($1,$2,$3,'conversation','direct-1',now()+interval '12 seconds')`, info.DID, info.ChildTokenHash[:], info.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SetConnection(t.Context(), info, false); err != nil {
		t.Fatal(err)
	}
	var activity, operations, pendingDeliveries int
	var cursor, state string
	var claimed, recovering bool
	if err := store.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM chat_view_activity),(SELECT count(*) FROM chat_operations),cursor,claim_token IS NOT NULL,recovery_id IS NOT NULL,(SELECT state FROM chat_notifications LIMIT 1) FROM chat_sync_state WHERE owner_did=$1`, info.DID).Scan(&activity, &operations, &cursor, &claimed, &recovering, &state); err != nil {
		t.Fatal(err)
	}
	if activity != 0 || operations != 1 || cursor != "resume-private-cursor" || claimed || !recovering || state != "cancelled" {
		t.Fatalf("disconnect activity=%d operations=%d claimed=%v recovery=%v notification=%s", activity, operations, claimed, recovering, state)
	}
	if err := store.pool.QueryRow(t.Context(), `SELECT count(*) FROM chat_push_deliveries WHERE status IN('pending','retry','leased')`).Scan(&pendingDeliveries); err != nil || pendingDeliveries != 0 {
		t.Fatalf("disconnect retained pending device deliveries=%d err=%v", pendingDeliveries, err)
	}
}
