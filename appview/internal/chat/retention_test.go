package chat

import (
	"net/http"
	"social.craftsky/appview/internal/auth"
	"strings"
	"testing"
)

// IT-018 / NFR-002: expire undelivered alerts after 24h, purge terminal
// delivery/routing detail after 30d, and retain minimal lifetime tombstones.
func TestMaintenancePurgesRoutingDetailWithoutRearmingLifetimeState(t *testing.T) {
	service, store, info := newServiceFixture(t, true, auth.DefaultScopes(), roundTripFunc(func(*http.Request) (*http.Response, error) { t.Fatal("retention called provider"); return nil, nil }))
	seedDeliveryFixture(t, store, info)
	for _, statement := range []string{
		`UPDATE chat_push_deliveries SET deadline_at=now()-interval '25 hours' WHERE owner_did=$1`,
		`WITH notification AS(INSERT INTO chat_notifications(owner_did,id,logical_key,kind,conversation_id,message_id,messaging_generation,open_reference,state,created_at,terminal_at) VALUES($1,gen_random_uuid(),'private-routing-canary:old-message','normal','private-routing-canary','old-message',2,gen_random_uuid(),'complete',now()-interval '40 days',now()-interval '31 days') RETURNING owner_did,id) INSERT INTO chat_push_deliveries(owner_did,id,notification_id,account_subscription_id,status,deadline_at,provider_result_class,created_at,terminal_at) SELECT notification.owner_did,gen_random_uuid(),notification.id,subscription.id,'succeeded',now()-interval '39 days','success',now()-interval '40 days',now()-interval '31 days' FROM notification JOIN push_account_subscriptions subscription ON subscription.account_did=notification.owner_did`,
		`INSERT INTO chat_conversation_state(owner_did,conversation_id,status,request_notified) VALUES($1,'retained-request','request',true)`,
		`INSERT INTO chat_message_dedupe(owner_did,conversation_id,message_id,disposition,observed_revision) VALUES($1,'retained-conversation','retained-message','notified','old-revision')`,
		`INSERT INTO chat_operations(owner_did,operation_id,kind,key_version,request_digest,state,created_at) VALUES($1,gen_random_uuid(),'send',1,decode(repeat('01',32),'hex'),'uncertain',now()-interval '1 year')`,
	} {
		if _, err := store.pool.Exec(t.Context(), statement, info.DID); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		if err := service.Maintain(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	var deliveries, expired, operations, dedupe, requests int
	var retained string
	if err := store.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM chat_push_deliveries),(SELECT count(*) FROM chat_push_deliveries WHERE status='expired'),(SELECT count(*) FROM chat_operations),(SELECT count(*) FROM chat_message_dedupe),(SELECT count(*) FROM chat_conversation_state WHERE request_notified),(SELECT string_agg(row_to_json(notification)::text,'') FROM chat_notifications notification)`).Scan(&deliveries, &expired, &operations, &dedupe, &requests, &retained); err != nil {
		t.Fatal(err)
	}
	if deliveries != 1 || expired != 1 || operations != 1 || dedupe != 1 || requests != 1 {
		t.Fatalf("retention deliveries=%d expired=%d operations=%d dedupe=%d requests=%d", deliveries, expired, operations, dedupe, requests)
	}
	if strings.Contains(retained, "private-routing-canary") || strings.Contains(retained, "old-message") {
		t.Fatal("terminal routing detail retained beyond thirty days")
	}
}
