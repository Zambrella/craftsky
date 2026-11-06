package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/guestvisibility"
)

// IT-005 / FR-009, RULE-004 / AC-009: unchanged authoritative records still
// require transactional local visibility acceptance before success is replayable.
func TestGuestVisibilityUnchangedRecordFinalizesAtomically(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	service, pds, request := postEditCommandFixture(t, &now)
	request.OperationKind = "profile.guest_visibility"
	failed := errors.New("injected local visibility failure")
	fail := true
	calls := 0
	request.FinalizeObserved = func(ctx context.Context, tx pgx.Tx, command PreparedCommand, record AuthoritativeRecord) error {
		calls++
		if record.ObservationRevision == "" || record.ObservationCID == "" {
			t.Fatal("unchanged record accepted without stable observation evidence")
		}
		if _, err := guestvisibility.ApplyTx(ctx, tx, request.Owner, guestvisibility.Hidden, guestvisibility.Evidence{Revision: record.ObservationRevision, CID: record.CID, Kind: guestvisibility.Observation}); err != nil {
			return err
		}
		if fail {
			return failed
		}
		return nil
	}
	if _, err := service.Put(context.Background(), request); !errors.Is(err, failed) {
		t.Fatalf("local finalization failure was accepted: %v", err)
	}
	var rows int
	if err := service.store.pool.QueryRow(context.Background(), `SELECT count(*) FROM bluesky_profiles WHERE did=$1`, request.Owner).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("failed acceptance leaked local state: %d %v", rows, err)
	}
	fail = false
	result, err := service.Put(context.Background(), request)
	if err != nil || result.State != CommandAccepted || calls != 2 || pds.applyCalls != 0 {
		t.Fatalf("retry acceptance=%+v err=%v callbacks=%d writes=%d", result, err, calls, pds.applyCalls)
	}
	_, err = service.Put(context.Background(), request)
	if err != nil || calls != 2 {
		t.Fatalf("terminal replay reran finalization: calls=%d err=%v", calls, err)
	}
}

// IT-005 / FR-009, RULE-004 / AC-009, AC-010.
func TestGuestVisibilityOwnerSaveFencesDelayedSource(t *testing.T) {
	now := time.Date(2026, 11, 6, 13, 0, 0, 0, time.UTC)
	service, pds, base := postEditCommandFixture(t, &now)
	delete(pds.records, base.URI)
	uri := syntax.ATURI("at://" + base.Owner.String() + "/app.bsky.actor.profile/self")
	value := map[string]any{"$type": "app.bsky.actor.profile", "displayName": "Original", "future": "keep"}
	cid, err := postEditTestRecordCID(value)
	if err != nil {
		t.Fatal(err)
	}
	pds.records[uri] = setCommandRecord{cid: cid, value: value}
	setting, err := service.ReadGuestVisibility(context.Background(), base.Owner, base.OwnerGeneration, base.SessionID)
	if err != nil || setting.HideFromLoggedOut || setting.ExpectedCID != cid {
		t.Fatalf("read=%+v err=%v", setting, err)
	}
	result, err := service.PutGuestVisibility(context.Background(), GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, ExpectedCID: cid, HideFromLoggedOut: true, OperationKey: base.OperationKey, Rejected: setCommandRejectedResult})
	if err != nil || result.State != CommandAccepted {
		t.Fatalf("save=%+v err=%v", result, err)
	}
	var saved GuestVisibilitySetting
	if err := json.Unmarshal(result.ResponseBody, &saved); err != nil || !saved.HideFromLoggedOut {
		t.Fatalf("accepted body=%s err=%v", result.ResponseBody, err)
	}
	var hidden *bool
	var hold *string
	if err := service.store.pool.QueryRow(context.Background(), `SELECT guest_visibility_hidden,guest_visibility_hold_key::text FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hidden, &hold); err != nil || hidden == nil || !*hidden || hold != nil {
		t.Fatalf("save acknowledged without local fence: hidden=%v hold=%v err=%v", hidden, hold, err)
	}
	if err := pgx.BeginFunc(context.Background(), service.store.pool, func(tx pgx.Tx) error {
		_, err := guestvisibility.ApplyTx(context.Background(), tx, base.Owner, guestvisibility.Public, guestvisibility.Evidence{Revision: "3aaaaaaaaaaa2", CID: cid, Kind: guestvisibility.Source})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.store.pool.QueryRow(context.Background(), `SELECT guest_visibility_hidden FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hidden); err != nil || hidden == nil || !*hidden {
		t.Fatalf("delayed source undid acknowledged opt-out: %v %v", hidden, err)
	}
	current := pds.records[uri].value.(map[string]any)
	if current["displayName"] != "Original" || current["future"] != "keep" || pds.applyCalls != 1 {
		t.Fatalf("unrelated profile state changed: %+v writes=%d", current, pds.applyCalls)
	}
}

func TestGuestVisibilityWriteSurvivesLocalAcceptanceFailure(t *testing.T) {
	now := time.Date(2026, 11, 6, 14, 0, 0, 0, time.UTC)
	service, pds, base := postEditCommandFixture(t, &now)
	uri := guestProfileURI(base.Owner)
	value := map[string]any{"$type": "app.bsky.actor.profile", "displayName": "Original"}
	cid, err := postEditTestRecordCID(value)
	if err != nil {
		t.Fatal(err)
	}
	pds.records[uri] = setCommandRecord{cid: cid, value: value}
	if _, err := service.ReadGuestVisibility(context.Background(), base.Owner, base.OwnerGeneration, base.SessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.store.pool.Exec(context.Background(), `CREATE FUNCTION fail_visibility_acceptance() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.guest_visibility_hidden IS TRUE THEN RAISE EXCEPTION 'injected visibility failure'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER fail_visibility_acceptance BEFORE UPDATE OF guest_visibility_hidden ON bluesky_profiles FOR EACH ROW EXECUTE FUNCTION fail_visibility_acceptance()`); err != nil {
		t.Fatal(err)
	}
	input := GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, ExpectedCID: cid, HideFromLoggedOut: true, OperationKey: base.OperationKey, Rejected: setCommandRejectedResult}
	if result, err := service.PutGuestVisibility(context.Background(), input); err == nil && result.State == CommandAccepted {
		t.Fatalf("local failure was acknowledged: %+v", result)
	}
	var hold *string
	if err := service.store.pool.QueryRow(context.Background(), `SELECT guest_visibility_hold_key::text FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hold); err != nil || hold == nil || pds.applyCalls != 1 {
		t.Fatalf("uncertain write lost hold: hold=%v writes=%d err=%v", hold, pds.applyCalls, err)
	}
	if _, err := service.store.pool.Exec(context.Background(), `DROP TRIGGER fail_visibility_acceptance ON bluesky_profiles`); err != nil {
		t.Fatal(err)
	}
	reopened, readErr := service.ReadGuestVisibility(context.Background(), base.Owner, base.OwnerGeneration, base.SessionID)
	if readErr != nil || reopened.Pending == nil || reopened.Pending.OperationKey != input.OperationKey || reopened.Pending.ExpectedCID != cid || !reopened.Pending.HideFromLoggedOut {
		t.Fatalf("pending owner operation not recoverable after reopen: %+v %v", reopened, readErr)
	}
	input.OperationKey = reopened.Pending.OperationKey
	input.ExpectedCID = reopened.Pending.ExpectedCID
	input.HideFromLoggedOut = reopened.Pending.HideFromLoggedOut
	result, err := service.PutGuestVisibility(context.Background(), input)
	if err != nil || result.State != CommandAccepted || pds.applyCalls != 1 {
		t.Fatalf("same-key recovery=%+v err=%v writes=%d", result, err, pds.applyCalls)
	}
	if err := service.store.pool.QueryRow(context.Background(), `SELECT guest_visibility_hold_key::text FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hold); err != nil || hold != nil {
		t.Fatalf("recovery did not clear its hold: %v %v", hold, err)
	}
}

// IT-007 / FR-008 / AC-008: a verified missing standard profile is not a failed
// lookup or a guessed legacy projection; the owner may create its shared label.
func TestGuestVisibilityOwnerWithNoStandardProfile(t *testing.T) {
	now := time.Date(2026, 11, 6, 15, 0, 0, 0, time.UTC)
	service, pds, base := postEditCommandFixture(t, &now)
	setting, err := service.ReadGuestVisibility(context.Background(), base.Owner, base.OwnerGeneration, base.SessionID)
	if err != nil || setting.HideFromLoggedOut || setting.ExpectedCID != "" {
		t.Fatalf("stable profile absence=%+v err=%v", setting, err)
	}
	result, err := service.PutGuestVisibility(context.Background(), GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, ExpectedCID: "", HideFromLoggedOut: true, OperationKey: base.OperationKey, Rejected: setCommandRejectedResult})
	if err != nil || result.State != CommandAccepted {
		t.Fatalf("create shared preference=%+v err=%v", result, err)
	}
	var got GuestVisibilitySetting
	if err := json.Unmarshal(result.ResponseBody, &got); err != nil || !got.HideFromLoggedOut || got.ExpectedCID == "" {
		t.Fatalf("created preference=%+v err=%v", got, err)
	}
	if pds.applyCalls != 1 {
		t.Fatalf("expected one explicit profile create, got %d", pds.applyCalls)
	}
}

// IT-005 / RULE-004 / AC-009: contradictory authoritative observations fail
// closed even when the owner read itself must return a recoverable conflict.
func TestGuestVisibilityConflictingObservationPersistsUnknown(t *testing.T) {
	now := time.Date(2026, 11, 6, 16, 0, 0, 0, time.UTC)
	service, pds, base := postEditCommandFixture(t, &now)
	uri := guestProfileURI(base.Owner)
	value := map[string]any{"$type": "app.bsky.actor.profile"}
	cid, err := postEditTestRecordCID(value)
	if err != nil {
		t.Fatal(err)
	}
	pds.records[uri] = setCommandRecord{cid: cid, value: value}
	if _, err := service.ReadGuestVisibility(context.Background(), base.Owner, base.OwnerGeneration, base.SessionID); err != nil {
		t.Fatal(err)
	}
	value["labels"] = map[string]any{"$type": "com.atproto.label.defs#selfLabels", "values": []any{map[string]string{"val": "!no-unauthenticated"}}}
	cid, err = postEditTestRecordCID(value)
	if err != nil {
		t.Fatal(err)
	}
	pds.records[uri] = setCommandRecord{cid: cid, value: value}
	if _, err := service.ReadGuestVisibility(context.Background(), base.Owner, base.OwnerGeneration, base.SessionID); !errors.Is(err, ErrRepositoryConflict) {
		t.Fatalf("expected conflicting observation, got %v", err)
	}
	var hidden *bool
	if err := service.store.pool.QueryRow(context.Background(), `SELECT guest_visibility_hidden FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hidden); err != nil || hidden != nil {
		t.Fatalf("contradictory evidence retained public state: hidden=%v err=%v", hidden, err)
	}
}

// IT-005 / FR-009 / AC-009: an overlapping retry prepared before acceptance
// cannot reinstall a fail-closed hold after the accepted command cleared it.
func TestGuestVisibilityOverlappingReplayDoesNotReinstallHold(t *testing.T) {
	now := time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	service, pds, base := postEditCommandFixture(t, &now)
	uri := guestProfileURI(base.Owner)
	value := map[string]any{"$type": "app.bsky.actor.profile"}
	cid, err := postEditTestRecordCID(value)
	if err != nil {
		t.Fatal(err)
	}
	pds.records[uri] = setCommandRecord{cid: cid, value: value}
	paused, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	service.newBoundary = func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
		if calls.Add(1) == 1 {
			close(paused)
			<-release
		}
		return &setCommandBoundary{client: pds}, nil
	}
	input := GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, ExpectedCID: cid, HideFromLoggedOut: true, OperationKey: base.OperationKey, Rejected: setCommandRejectedResult}
	first := make(chan error, 1)
	go func() {
		result, err := service.PutGuestVisibility(context.Background(), input)
		if err == nil && result.State != CommandAccepted {
			err = errors.New("first command not accepted")
		}
		first <- err
	}()
	<-paused
	result, err := service.PutGuestVisibility(context.Background(), input)
	close(release)
	if firstErr := <-first; firstErr != nil {
		t.Fatal(firstErr)
	}
	if err != nil || result.State != CommandAccepted {
		t.Fatalf("overlap=%+v %v", result, err)
	}
	var hold *string
	if err := service.store.pool.QueryRow(context.Background(), `SELECT guest_visibility_hold_key::text FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hold); err != nil || hold != nil {
		t.Fatalf("accepted replay reinstalled hold: %v %v", hold, err)
	}
	if pds.applyCalls != 1 {
		t.Fatalf("duplicate PDS writes: %d", pds.applyCalls)
	}
}

// IT-005 / FR-009, RULE-004 / AC-009: conflict rejection must retain the
// currently observed shared preference before releasing the anonymous hold.
func TestGuestVisibilityRejectedSaveRetainsCurrentAuthority(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		t.Run(map[bool]string{false: "observed", true: "observation unavailable"}[unavailable], func(t *testing.T) {
			now := time.Date(2026, 11, 6, 18, 0, 0, 0, time.UTC)
			service, pds, base := postEditCommandFixture(t, &now)
			ctx := context.Background()
			uri := guestProfileURI(base.Owner)
			value := map[string]any{"$type": "app.bsky.actor.profile"}
			oldCID, err := postEditTestRecordCID(value)
			if err != nil {
				t.Fatal(err)
			}
			pds.records[uri] = setCommandRecord{cid: oldCID, value: value}
			if _, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID); err != nil {
				t.Fatal(err)
			}
			value["labels"] = map[string]any{"$type": "com.atproto.label.defs#selfLabels", "values": []any{map[string]string{"val": "!no-unauthenticated"}}}
			newCID, err := postEditTestRecordCID(value)
			if err != nil {
				t.Fatal(err)
			}
			pds.records[uri] = setCommandRecord{cid: newCID, value: value}
			pds.revision = syntax.NewTIDFromInteger(pds.revision.Integer() + 10)
			pds.head, err = postEditTestRecordCID(map[string]any{"head": 99})
			if err != nil {
				t.Fatal(err)
			}
			boundary := service.newBoundary
			calls := 0
			if unavailable {
				service.newBoundary = func(ctx context.Context, owner syntax.DID, session string) (auth.ActiveEffectPDSBoundary, error) {
					calls++
					if calls > 1 {
						return nil, ErrDispatchUnavailable
					}
					return boundary(ctx, owner, session)
				}
			}
			result, saveErr := service.PutGuestVisibility(ctx, GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, ExpectedCID: oldCID, HideFromLoggedOut: false, OperationKey: base.OperationKey, Rejected: setCommandRejectedResult})
			if !unavailable && (saveErr != nil || result.State != CommandRejected) {
				t.Fatalf("conflicting save=%+v %v", result, saveErr)
			}
			var hidden *bool
			var hold *string
			if err := service.store.pool.QueryRow(ctx, `SELECT guest_visibility_hidden,guest_visibility_hold_key::text FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hidden, &hold); err != nil {
				t.Fatal(err)
			}
			if unavailable {
				if hold == nil {
					t.Fatal("rejected save released hold without current authority")
				}
				service.newBoundary = boundary
				setting, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID)
				if err != nil || !setting.HideFromLoggedOut || setting.Pending != nil {
					t.Fatalf("owner reload did not recover rejected hold: %+v %v", setting, err)
				}
			} else if hidden == nil || !*hidden || hold != nil {
				t.Fatalf("rejection left stale public projection: hidden=%v hold=%v", hidden, hold)
			}
			if pds.applyCalls != 0 {
				t.Fatal("conflict wrote to PDS")
			}
		})
	}
}

// IT-007 / FR-008, RULE-004 / AC-008, AC-010: unsupported authoritative
// records are an error and persist unknown; they never imply a public default.
func TestGuestVisibilityUnsupportedOwnerRecordFailsClosed(t *testing.T) {
	for _, record := range []map[string]any{
		{"$type": "app.bsky.actor.profile", "labels": map[string]any{"$type": "future.labels", "values": []any{}}},
		{"$type": "future.profile"},
	} {
		t.Run(fmt.Sprint(record["$type"], record["labels"]), func(t *testing.T) {
			now := time.Date(2026, 11, 6, 19, 0, 0, 0, time.UTC)
			service, pds, base := postEditCommandFixture(t, &now)
			ctx := context.Background()
			uri := guestProfileURI(base.Owner)
			initial := map[string]any{"$type": "app.bsky.actor.profile"}
			cid, err := postEditTestRecordCID(initial)
			if err != nil {
				t.Fatal(err)
			}
			pds.records[uri] = setCommandRecord{cid: cid, value: initial}
			if _, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID); err != nil {
				t.Fatal(err)
			}
			cid, err = postEditTestRecordCID(record)
			if err != nil {
				t.Fatal(err)
			}
			pds.records[uri] = setCommandRecord{cid: cid, value: record}
			pds.revision = syntax.NewTIDFromInteger(pds.revision.Integer() + 1)
			pds.head, err = postEditTestRecordCID(map[string]any{"head": 77})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID); !errors.Is(err, ErrGuestVisibilityUnknown) {
				t.Fatalf("unsupported profile was treated as known public: %v", err)
			}
			var hidden *bool
			if err := service.store.pool.QueryRow(ctx, `SELECT guest_visibility_hidden FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hidden); err != nil || hidden != nil {
				t.Fatalf("unsupported authority retained public state: hidden=%v err=%v", hidden, err)
			}
		})
	}
}

// IT-005 / FR-009 / AC-009: terminal command retention must not orphan a
// fail-closed hold while the owner still needs an authoritative recovery read.
func TestGuestVisibilityHoldSurvivesCommandCompactionUntilOwnerRecovery(t *testing.T) {
	now := time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	service, _, base := postEditCommandFixture(t, &now)
	ctx := context.Background()
	result, err := service.PutGuestVisibility(ctx, GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, HideFromLoggedOut: true, OperationKey: base.OperationKey, Rejected: setCommandRejectedResult})
	if err != nil || result.State != CommandAccepted {
		t.Fatalf("fixture save=%+v %v", result, err)
	}
	if _, err := service.store.pool.Exec(ctx, `UPDATE bluesky_profiles SET guest_visibility_hold_key=(SELECT id FROM pds_commands WHERE owner_did=$1 AND operation_kind='profile.guest_visibility' AND operation_key=$2) WHERE did=$1`, base.Owner, base.OperationKey); err != nil {
		t.Fatal(err)
	}
	now = now.Add(ReplayRetention + time.Minute)
	if count, err := service.store.CompactExpired(ctx, 10); err != nil || count != 0 {
		t.Fatalf("compaction orphaned visibility recovery: count=%d err=%v", count, err)
	}
	setting, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID)
	if err != nil || !setting.HideFromLoggedOut {
		t.Fatalf("recovery=%+v %v", setting, err)
	}
	if count, err := service.store.CompactExpired(ctx, 10); err != nil || count != 1 {
		t.Fatalf("resolved command not compacted: count=%d err=%v", count, err)
	}
}

// IT-005 / FR-008, FR-009 / AC-009, AC-014: a later unrelated PDS edit
// must not strand local-acceptance or response-loss recovery.
func TestGuestVisibilityRecoversAfterExternalEdit(t *testing.T) {
	for _, hidden := range []bool{true, false} {
		for _, failure := range []string{"local acceptance", "response loss", "local acceptance/reversed", "response loss/reversed"} {
			t.Run(fmt.Sprintf("hidden=%t/%s", hidden, failure), func(t *testing.T) {
				reversed := strings.HasSuffix(failure, "/reversed")
				localFailure := strings.HasPrefix(failure, "local acceptance")
				ctx := context.Background()
				now := time.Date(2026, 11, 6, 18, 0, 0, 0, time.UTC)
				service, pds, base := postEditCommandFixture(t, &now)
				uri := guestProfileURI(base.Owner)
				raw, err := guestvisibility.SetPreference(json.RawMessage(`{"$type":"app.bsky.actor.profile","displayName":"Original","extension":9007199254740993}`), !hidden)
				if err != nil {
					t.Fatal(err)
				}
				cid, err := postEditTestRecordCID(raw)
				if err != nil {
					t.Fatal(err)
				}
				pds.records[uri] = setCommandRecord{cid: cid, value: json.RawMessage(raw)}
				dispatchHead, dispatchRevision := pds.head, pds.revision
				if _, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID); err != nil {
					t.Fatal(err)
				}
				if localFailure {
					_, err := service.store.pool.Exec(ctx, fmt.Sprintf(`CREATE FUNCTION fail_visibility_acceptance() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.guest_visibility_hidden IS %t THEN RAISE EXCEPTION 'injected visibility failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER fail_visibility_acceptance BEFORE UPDATE OF guest_visibility_hidden ON bluesky_profiles FOR EACH ROW EXECUTE FUNCTION fail_visibility_acceptance()`, hidden))
					if err != nil {
						t.Fatal(err)
					}
				} else {
					pds.loseNextResponse = true
				}
				input := GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, ExpectedCID: cid, HideFromLoggedOut: hidden, OperationKey: base.OperationKey, Rejected: setCommandRejectedResult}
				if result, err := service.PutGuestVisibility(ctx, input); err == nil && result.State == CommandAccepted {
					t.Fatal("uncertain operation acknowledged")
				}
				if localFailure {
					if _, err := service.store.pool.Exec(ctx, `DROP TRIGGER fail_visibility_acceptance ON bluesky_profiles`); err != nil {
						t.Fatal(err)
					}
				}
				// Preserve every profile field and label while another app changes the name.
				current, err := json.Marshal(pds.records[uri].value)
				if err != nil {
					t.Fatal(err)
				}
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(current, &fields); err != nil {
					t.Fatal(err)
				}
				fields["displayName"] = json.RawMessage(`"Changed externally"`)
				current, err = json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				if reversed {
					current, err = guestvisibility.SetPreference(current, !hidden)
					if err != nil {
						t.Fatal(err)
					}
				}
				changedCID, err := postEditTestRecordCID(json.RawMessage(current))
				if err != nil {
					t.Fatal(err)
				}
				pds.records[uri] = setCommandRecord{cid: changedCID, value: json.RawMessage(current)}
				pds.head = changedCID
				pds.revision = syntax.NewTIDFromInteger(pds.revision.Integer() + 1)
				reopened, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID)
				if err != nil || reopened.Pending == nil {
					t.Fatalf("pending operation lost: %+v %v", reopened, err)
				}
				observedHead, observedRevision := pds.head, pds.revision
				for _, evidence := range []struct {
					head     syntax.CID
					revision syntax.TID
				}{
					{dispatchHead, observedRevision}, {observedHead, dispatchRevision},
				} {
					pds.head, pds.revision = evidence.head, evidence.revision
					if result, err := service.PutGuestVisibility(ctx, input); err == nil || result.State == CommandAccepted || result.State == CommandRejected {
						t.Fatalf("unsafe observation resolved operation: %+v %v", result, err)
					}
					var hold *string
					if err := service.store.pool.QueryRow(ctx, `SELECT guest_visibility_hold_key::text FROM bluesky_profiles WHERE did=$1`, base.Owner).Scan(&hold); err != nil || hold == nil {
						t.Fatalf("unsafe evidence released hold: %v %v", hold, err)
					}
				}
				pds.head, pds.revision = observedHead, observedRevision
				wantState := CommandAccepted
				if reversed {
					wantState = CommandRejected
				}
				for retry := 0; retry < 2; retry++ {
					result, err := service.PutGuestVisibility(ctx, input)
					if err != nil || result.State != wantState || pds.applyCalls != 1 {
						t.Fatalf("recovery=%+v err=%v writes=%d", result, err, pds.applyCalls)
					}
					if reversed {
						if result.HTTPStatus != 409 {
							t.Fatalf("conflict status=%d", result.HTTPStatus)
						}
						continue
					}
					var saved GuestVisibilitySetting
					if err := json.Unmarshal(result.ResponseBody, &saved); err != nil || saved.ExpectedCID != changedCID || saved.HideFromLoggedOut != hidden {
						t.Fatalf("current state not acknowledged: %+v %v", saved, err)
					}
				}
				reopened, err = service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID)
				if err != nil || reopened.Pending != nil || reopened.ExpectedCID != changedCID {
					t.Fatalf("unresolved reopen: %+v %v", reopened, err)
				}
				// A subsequent deliberate change remains possible and preserves that edit.
				input.ExpectedCID, input.OperationKey, input.HideFromLoggedOut = changedCID, uuid.New(), !reopened.HideFromLoggedOut
				if result, err := service.PutGuestVisibility(ctx, input); err != nil || result.State != CommandAccepted {
					t.Fatalf("subsequent change=%+v %v", result, err)
				}
				current, _ = json.Marshal(pds.records[uri].value)
				if err := json.Unmarshal(current, &fields); err != nil {
					t.Fatal(err)
				}
				if string(fields["displayName"]) != `"Changed externally"` || string(fields["extension"]) != "9007199254740993" {
					t.Fatalf("external fields changed: %s", current)
				}
			})
		}
	}
}

// IT-005 / FR-008–009 / AC-009: creating the standard profile uses the same
// recoverable owner setting contract as updating one.
func TestGuestVisibilityRecoversCreatedProfile(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 6, 19, 0, 0, 0, time.UTC)
	service, pds, base := postEditCommandFixture(t, &now)
	setting, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID)
	if err != nil || setting.ExpectedCID != "" {
		t.Fatalf("verified absence=%+v %v", setting, err)
	}
	input := GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, OperationKey: base.OperationKey, HideFromLoggedOut: true, Rejected: setCommandRejectedResult}
	pds.loseNextResponse = true
	if result, err := service.PutGuestVisibility(ctx, input); err != nil || result.State != CommandAmbiguous {
		t.Fatalf("lost create response=%+v %v", result, err)
	}
	if result, err := service.PutGuestVisibility(ctx, input); err != nil || result.State != CommandAccepted || pds.applyCalls != 1 {
		t.Fatalf("create recovery=%+v %v writes=%d", result, err, pds.applyCalls)
	}
	setting, err = service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID)
	if err != nil || setting.Pending != nil || !setting.HideFromLoggedOut {
		t.Fatalf("reopened create=%+v %v", setting, err)
	}
}

// IT-005 / FR-009 / AC-009: deletion in another app is a later authoritative
// state, not a reason to keep an old visibility write pending forever.
func TestGuestVisibilityRecoveryAfterExternalProfileDeletion(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	service, pds, base := postEditCommandFixture(t, &now)
	uri := guestProfileURI(base.Owner)
	value := json.RawMessage(`{"$type":"app.bsky.actor.profile"}`)
	cid, err := postEditTestRecordCID(value)
	if err != nil {
		t.Fatal(err)
	}
	pds.records[uri] = setCommandRecord{cid: cid, value: value}
	if _, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID); err != nil {
		t.Fatal(err)
	}
	input := GuestVisibilityPutRequest{Owner: base.Owner, OwnerGeneration: base.OwnerGeneration, SessionID: base.SessionID, OperationKey: base.OperationKey, ExpectedCID: cid, HideFromLoggedOut: true, Rejected: setCommandRejectedResult}
	pds.loseNextResponse = true
	if result, err := service.PutGuestVisibility(ctx, input); err != nil || result.State != CommandAmbiguous {
		t.Fatalf("lost response=%+v %v", result, err)
	}
	delete(pds.records, uri)
	pds.head, err = postEditTestRecordCID(map[string]any{"deleted": uri})
	if err != nil {
		t.Fatal(err)
	}
	pds.revision = syntax.NewTIDFromInteger(pds.revision.Integer() + 1)
	if result, err := service.PutGuestVisibility(ctx, input); err != nil || result.State != CommandRejected || result.HTTPStatus != 409 {
		t.Fatalf("deleted-profile recovery=%+v %v", result, err)
	}
	setting, err := service.ReadGuestVisibility(ctx, base.Owner, base.OwnerGeneration, base.SessionID)
	if err != nil || setting.Pending != nil || setting.HideFromLoggedOut || setting.ExpectedCID != "" {
		t.Fatalf("unresolved absence=%+v %v", setting, err)
	}
	if pds.applyCalls != 1 {
		t.Fatal("recovery recreated deleted profile")
	}
}
