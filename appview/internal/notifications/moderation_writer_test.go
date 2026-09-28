package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

const moderationWriterTestDDL = `
CREATE TABLE notification_events (
 id UUID PRIMARY KEY, recipient_did TEXT NOT NULL, actor_did TEXT,
 category TEXT NOT NULL, subject_key TEXT NOT NULL, source_uri TEXT, source_cid TEXT,
 source_rkey TEXT, eligibility_scope TEXT NOT NULL, recipient_followed_actor BOOLEAN NOT NULL,
 push_enabled_snapshot BOOLEAN NOT NULL, state TEXT NOT NULL, first_activity_at TIMESTAMPTZ NOT NULL,
 activity_at TIMESTAMPTZ NOT NULL, indexed_at TIMESTAMPTZ NOT NULL,
 initial_push_evaluated_at TIMESTAMPTZ NOT NULL, moderation_case_reference TEXT,
 moderation_event_id UUID
);
CREATE UNIQUE INDEX notification_events_moderation_event_unique
 ON notification_events(moderation_event_id) WHERE category='moderation';
CREATE TABLE notification_preferences (
 account_did TEXT NOT NULL, category TEXT NOT NULL, scope TEXT NOT NULL,
 push_enabled BOOLEAN NOT NULL, PRIMARY KEY(account_did,category)
);
CREATE TABLE push_installations (
 id UUID PRIMARY KEY, device_id TEXT NOT NULL, platform TEXT NOT NULL,
 fcm_token TEXT NOT NULL, active BOOLEAN NOT NULL
);
CREATE TABLE push_account_subscriptions (
 id UUID PRIMARY KEY, installation_id UUID NOT NULL REFERENCES push_installations(id),
 account_did TEXT NOT NULL, routing_id UUID NOT NULL, active BOOLEAN NOT NULL
);
CREATE TABLE push_deliveries (
 id UUID PRIMARY KEY, notification_id UUID NOT NULL REFERENCES notification_events(id),
 account_subscription_id UUID NOT NULL REFERENCES push_account_subscriptions(id), status TEXT NOT NULL,
 next_attempt_at TIMESTAMPTZ NOT NULL, deadline_at TIMESTAMPTZ NOT NULL,
 UNIQUE(notification_id,account_subscription_id)
);`

func TestModerationWriterQueuesDeliveryWhenPreferenceIsDisabledUntilDispatchRecheck(t *testing.T) {
	pool := testdb.WithSchema(t, moderationWriterTestDDL)
	ctx := context.Background()
	installationID, subscriptionID := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO push_installations(id,device_id,platform,fcm_token,active) VALUES($1,'device','ios','token',true)`, installationID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id,active) VALUES($1,$2,'did:plc:owner',$3,true)`, subscriptionID, installationID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	caseID, eventOne, eventTwo := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	writer := ModerationWriter{}
	insert := func(eventID uuid.UUID) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err := writer.InsertModerationIntentTx(ctx, tx, ModerationIntent{RecipientDID: syntax.DID("did:plc:owner"), CaseID: caseID, CaseReference: "MOD-" + caseID.String(), EventID: eventID, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	insert(eventOne)
	insert(eventOne)
	if _, err := pool.Exec(ctx, `INSERT INTO notification_preferences(account_did,category,scope,push_enabled) VALUES('did:plc:owner','moderation','everyone',false)`); err != nil {
		t.Fatal(err)
	}
	insert(eventTwo)

	var events, deliveries int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events WHERE moderation_case_reference=$1`, "MOD-"+caseID.String()).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM push_deliveries`).Scan(&deliveries); err != nil {
		t.Fatal(err)
	}
	if events != 2 || deliveries != 2 {
		t.Fatalf("events/deliveries = %d/%d, want 2/2", events, deliveries)
	}
}
