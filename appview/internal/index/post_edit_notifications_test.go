package index_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/postpublication"
	"social.craftsky/appview/internal/tap"
)

// IT-012: coalescing before notification projection cannot add edit recipients.
func TestPostEditBeforeInitialProjectionKeepsOriginalRecipients(t *testing.T) {
	f := newNotificationPipeline(t)
	seedCraftskyMember(t, f.pool, "did:plc:added-by-edit")
	original := notificationEvent(800, "did:plc:actor", "3aaaaaaaaaaa4", mentionExtra("did:plc:recipient"))
	original.Live = true
	f.ingest(original)
	edited := original
	edited.ID++
	edited.Rev = "3aaaaaaaaaaa3"
	edited.CID = "cid-edited"
	edited.Action = "update"
	var root map[string]any
	if err := json.Unmarshal(edited.Record, &root); err != nil {
		t.Fatal(err)
	}
	facets := root["facets"].([]any)
	facet := facets[0].(map[string]any)
	facet["features"] = append(facet["features"].([]any), map[string]any{"$type": "app.bsky.richtext.facet#mention", "did": "did:plc:added-by-edit"})
	edited.Record, _ = json.Marshal(root)
	f.ingest(edited)
	f.drain()
	f.assertCount(`SELECT count(*) FROM craftsky_post_mentions`, 2)
	f.assertCount(`SELECT count(*) FROM notification_events WHERE recipient_did='did:plc:recipient'`, 1)
	f.assertCount(`SELECT count(*) FROM notification_events WHERE recipient_did='did:plc:added-by-edit'`, 0)
	// Replay both versions and the current job. Neither expands the original set.
	if _, err := f.store.IngestRecord(context.Background(), original); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.IngestRecord(context.Background(), edited); err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE tap_projection_jobs SET state='pending',completed_at=NULL WHERE projection_kind='post_notifications'`)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 1)
}

// IT-012: an edited-only snapshot has no proven original reply recipient.
func TestPostEditOnlySnapshotDoesNotInventReplyNotification(t *testing.T) {
	f := newNotificationPipeline(t)
	target := notificationEvent(810, "did:plc:recipient", "3aaaaaaaaaaa2", "")
	target.Live = true
	f.ingest(target)
	f.drain()
	edited := notificationEvent(811, "did:plc:actor", "3aaaaaaaaaaa4", replyExtra(notificationTarget))
	edited.Action = "update"
	edited.Live = true
	f.ingest(edited)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
}

// IT-012: an original recipient cannot gain a new reason after the original
// mention disappears, even if another client changes structural references.
func TestPostEditCannotReplaceOriginalMentionWithNewReplyReason(t *testing.T) {
	f := newNotificationPipeline(t)
	target := notificationEvent(820, "did:plc:recipient", "3aaaaaaaaaaa2", "")
	target.Live = true
	f.ingest(target)
	f.drain()
	original := notificationEvent(821, "did:plc:actor", "3aaaaaaaaaaa4", mentionExtra("did:plc:recipient"))
	original.Live = true
	f.ingest(original)
	edited := notificationEvent(822, "did:plc:actor", "3aaaaaaaaaaa4", replyExtra(notificationTarget))
	edited.Live = true
	edited.Action = "update"
	edited.Rev = "3aaaaaaaaaaa3"
	f.ingest(edited)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
}

// IT-012 / REG-007: backfill alone cannot prove original publication eligibility.
func TestPostBackfillSnapshotDoesNotInferPublicationRecipients(t *testing.T) {
	f := newNotificationPipeline(t)
	target := notificationEvent(830, "did:plc:recipient", "3aaaaaaaaaaa2", "")
	f.ingest(target)
	f.drain()
	historical := notificationEvent(831, "did:plc:actor", "3aaaaaaaaaaa4", replyExtra(notificationTarget)+mentionExtra("did:plc:recipient"))
	historical.Live = false
	f.ingest(historical)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
}

// IT-012: delayed initial notification uses the original publication time even
// if an external client changes the current record's declared creation time.
func TestPostEditPendingNotificationKeepsOriginalActivityTime(t *testing.T) {
	f := newNotificationPipeline(t)
	original := notificationEvent(840, "did:plc:actor", "3aaaaaaaaaaa4", mentionExtra("did:plc:recipient"))
	f.ingest(original)
	edited := original
	edited.ID++
	edited.Action = "update"
	edited.Rev = "3aaaaaaaaaaa3"
	edited.CID = "cid-edited"
	var root map[string]any
	if err := json.Unmarshal(edited.Record, &root); err != nil {
		t.Fatal(err)
	}
	root["createdAt"] = "2026-05-05T12:00:00Z"
	edited.Record, _ = json.Marshal(root)
	f.ingest(edited)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE activity_at='2026-05-04T12:00:00Z' AND first_activity_at=activity_at`, 1)
}

// IT-012: removing an original pending reason cancels its outbox work and a
// later re-addition cannot turn it into a fresh publication.
func TestPostEditRemovedMentionSuppressesPendingDelivery(t *testing.T) {
	f := newNotificationPipeline(t)
	f.exec(`INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES('00000000-0000-0000-0000-000000000001','edit-install','ios','edit-token')`)
	f.exec(`INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) VALUES('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','did:plc:recipient','00000000-0000-0000-0000-000000000003')`)
	original := notificationEvent(850, "did:plc:actor", "3aaaaaaaaaaa4", mentionExtra("did:plc:recipient"))
	f.ingest(original)
	f.drain()
	f.assertCount(`SELECT count(*) FROM push_deliveries WHERE status='pending'`, 1)
	edited := notificationEvent(851, "did:plc:actor", "3aaaaaaaaaaa4", "")
	edited.Action = "update"
	edited.Rev = "3aaaaaaaaaaa3"
	f.ingest(edited)
	f.drain()
	f.assertCount(`SELECT count(*) FROM push_deliveries WHERE status='cancelled'`, 1)
	readded := original
	readded.ID = 852
	readded.Action = "update"
	readded.Rev = "3aaaaaaaaaaa4"
	readded.CID = "cid-readded"
	f.ingest(readded)
	f.drain()
	f.assertCount(`SELECT count(*) FROM push_deliveries`, 1)
	f.assertCount(`SELECT count(*) FROM push_deliveries WHERE status='cancelled'`, 1)
	f.assertCount(`SELECT count(*) FROM notification_events WHERE state='retracted'`, 1)
}

// IT-012: an intervening removal retires the pending original reason even if
// both updates are coalesced before any initial notification projection.
func TestPostEditRemovedThenReaddedBeforeProjectionHasNoCatchup(t *testing.T) {
	f := newNotificationPipeline(t)
	original := notificationEvent(860, "did:plc:actor", "3aaaaaaaaaaa4", mentionExtra("did:plc:recipient"))
	f.ingest(original)
	removed := notificationEvent(861, "did:plc:actor", "3aaaaaaaaaaa4", "")
	removed.Action = "update"
	removed.Rev = "3aaaaaaaaaaa3"
	f.ingest(removed)
	readded := original
	readded.ID = 862
	readded.Action = "update"
	readded.Rev = "3aaaaaaaaaaa4"
	readded.CID = "cid-readded"
	f.ingest(readded)
	f.drain()
	f.assertCount(`SELECT count(*) FROM craftsky_post_mentions`, 1)
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
}

// IT-012: another client's remove/re-add of a structural reason cannot revive
// delayed original publication work either.
func TestPostEditRemovedThenReaddedReferenceHasNoCatchup(t *testing.T) {
	for _, kind := range []string{"reply", "quote"} {
		t.Run(kind, func(t *testing.T) {
			f := newNotificationPipeline(t)
			f.ingest(notificationEvent(870, "did:plc:recipient", "3aaaaaaaaaaa2", ""))
			f.drain()
			extra := replyExtra(notificationTarget)
			if kind == "quote" {
				extra = quoteExtra(notificationTarget)
			}
			original := notificationEvent(871, "did:plc:actor", "3aaaaaaaaaaa4", extra)
			f.ingest(original)
			removed := notificationEvent(872, "did:plc:actor", "3aaaaaaaaaaa4", "")
			removed.Action = "update"
			removed.Rev = "3aaaaaaaaaaa3"
			f.ingest(removed)
			readded := original
			readded.ID = 873
			readded.Action = "update"
			readded.Rev = "3aaaaaaaaaaa4"
			readded.CID = "cid-readded"
			f.ingest(readded)
			f.drain()
			f.assertCount(`SELECT count(*) FROM notification_events`, 0)
		})
	}
}

// IT-012 / REG-007: retiring one original reason does not suppress another
// original reason that still applies to the same recipient.
func TestPostEditRemovingMentionKeepsOriginalReply(t *testing.T) {
	f := newNotificationPipeline(t)
	f.ingest(notificationEvent(880, "did:plc:recipient", "3aaaaaaaaaaa2", ""))
	f.drain()
	original := notificationEvent(881, "did:plc:actor", "3aaaaaaaaaaa4", replyExtra(notificationTarget)+mentionExtra("did:plc:recipient"))
	f.ingest(original)
	edited := notificationEvent(882, "did:plc:actor", "3aaaaaaaaaaa4", replyExtra(notificationTarget))
	edited.Action = "update"
	edited.Rev = "3aaaaaaaaaaa3"
	f.ingest(edited)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE category='reply' AND state='active'`, 1)
	f.assertCount(`SELECT count(*) FROM post_publication_mentions WHERE eligible`, 0)
	f.exec(`UPDATE tap_projection_jobs SET state='pending',completed_at=NULL WHERE projection_kind='post_notifications'`)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 1)
}

func seedAcceptedOriginalPublication(t *testing.T, f *notificationPipeline, original tap.Event) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	commands, err := pdscommands.NewStore(pdscommands.StoreConfig{Pool: f.pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := pdscommands.RequestFingerprint(pdscommands.RequestFingerprintInput{AlgorithmVersion: 1, Owner: original.DID, OperationKind: "post.create", Intent: original.Record})
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := commands.PrepareOrReplay(ctx, pdscommands.PrepareRequest{Owner: original.DID, OwnerGeneration: 1, OperationKind: "post.create", OperationKey: uuid.New(), FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: original.Record, SelectedURI: original.URI, SelectedRkey: original.Rkey, ExpectedOwner: original.DID, ExpectedOwnerGeneration: 1})
	if err != nil {
		t.Fatal(err)
	}
	steps := []pdscommands.DispatchStep{{Action: "create", URI: original.URI, Body: original.Record}}
	dispatchFingerprint, err := pdscommands.DispatchFingerprint(pdscommands.DispatchFingerprintInput{AlgorithmVersion: 1, RequestFingerprint: fingerprint, RepositoryHead: "head", Steps: steps})
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := commands.BeginDispatch(ctx, prepared.ID, pdscommands.ExactPlan{Version: 1, FingerprintVersion: 1, Fingerprint: dispatchFingerprint, RepositoryCID: "head", RemoteDeadline: now.Add(time.Minute), Steps: steps})
	if err != nil {
		t.Fatal(err)
	}
	if err := commands.CompleteDispatchAndCommand(ctx, attempt.ID, pdscommands.DispatchCompletion{Outcome: pdscommands.CommandAccepted, ResultRecords: original.Record}, pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: 201, ResponseBody: json.RawMessage(`{"post":null}`)}); err != nil {
		t.Fatal(err)
	}
}

// IT-012: a proven local create supplies original eligibility when the first
// source snapshot is already an update; the edited snapshot cannot add B.
func TestPostEditFirstSnapshotUsesAcceptedOriginalCreateEvidence(t *testing.T) {
	f := newNotificationPipeline(t)
	seedCraftskyMember(t, f.pool, "did:plc:added-by-edit")
	original := notificationEvent(890, "did:plc:actor", "3aaaaaaaaaaa4", mentionExtra("did:plc:recipient"))
	seedAcceptedOriginalPublication(t, f, original)
	edited := original
	edited.ID++
	edited.Action = "update"
	edited.Rev = "3aaaaaaaaaaa3"
	edited.CID = "cid-edited"
	var root map[string]any
	if err := json.Unmarshal(edited.Record, &root); err != nil {
		t.Fatal(err)
	}
	facet := root["facets"].([]any)[0].(map[string]any)
	facet["features"] = append(facet["features"].([]any), map[string]any{"$type": "app.bsky.richtext.facet#mention", "did": "did:plc:added-by-edit"})
	edited.Record, _ = json.Marshal(root)
	f.ingest(edited)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE recipient_did='did:plc:recipient'`, 1)
	f.assertCount(`SELECT count(*) FROM notification_events WHERE recipient_did='did:plc:added-by-edit'`, 0)
}

// IT-012: late original proof may establish unknown eligibility, but cannot
// erase a removal observed while that proof was unavailable.
func TestPostEditLateOriginalProofRespectsObservedApplicability(t *testing.T) {
	for _, removed := range []bool{false, true} {
		name := "kept"
		if removed {
			name = "removed_then_readded"
		}
		t.Run(name, func(t *testing.T) {
			f := newNotificationPipeline(t)
			original := notificationEvent(895, "did:plc:actor", "3aaaaaaaaaaa4", mentionExtra("did:plc:recipient"))
			snapshot := original
			snapshot.Action = "update"
			snapshot.Rev = "3aaaaaaaaaaa3"
			snapshot.ID = 896
			f.ingest(snapshot)
			f.drain()
			f.assertCount(`SELECT count(*) FROM notification_events`, 0)
			if removed {
				absent := notificationEvent(897, "did:plc:actor", "3aaaaaaaaaaa4", "")
				absent.Action = "update"
				absent.Rev = "3aaaaaaaaaaa4"
				f.ingest(absent)
				restored := snapshot
				restored.ID = 898
				restored.Rev = "3aaaaaaaaaaa5"
				f.ingest(restored)
			}
			seedAcceptedOriginalPublication(t, f, original)
			snapshot.ID = 899
			snapshot.Rev = "3aaaaaaaaaaa6"
			f.ingest(snapshot)
			f.drain()
			want := 1
			if removed {
				want = 0
			}
			f.assertCount(`SELECT count(*) FROM notification_events WHERE recipient_did='did:plc:recipient'`, want)
		})
	}
}

// IT-012: late accepted-create seeding must wake already-completed unknown
// notification work without waiting for another source update.
func TestPostCreateProofWakesWaitingOriginalPublication(t *testing.T) {
	f := newNotificationPipeline(t)
	original := notificationEvent(900, "did:plc:actor", "3aaaaaaaaaaa4", mentionExtra("did:plc:recipient"))
	snapshot := original
	snapshot.Action = "update"
	snapshot.Rev = "3aaaaaaaaaaa3"
	f.ingest(snapshot)
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events`, 0)
	seed := func() {
		t.Helper()
		if err := pgx.BeginFunc(context.Background(), f.pool, func(tx pgx.Tx) error {
			return postpublication.SeedAcceptedCreate(context.Background(), tx, original.URI, original.Record)
		}); err != nil {
			t.Fatal(err)
		}
	}
	seed()
	f.drain()
	f.assertCount(`SELECT count(*) FROM notification_events WHERE recipient_did='did:plc:recipient'`, 1)
	var revision int64
	if err := f.pool.QueryRow(context.Background(), `SELECT newness_revision FROM notification_events`).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	seed()
	f.drain()
	var replayRevision int64
	if err := f.pool.QueryRow(context.Background(), `SELECT newness_revision FROM notification_events`).Scan(&replayRevision); err != nil {
		t.Fatal(err)
	}
	if replayRevision != revision {
		t.Fatalf("repeated seed changed original activity: %d -> %d", revision, replayRevision)
	}
	f.assertCount(`SELECT count(*) FROM notification_events`, 1)
}
