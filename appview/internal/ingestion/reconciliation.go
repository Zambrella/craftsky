package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

var ErrReconciliationSourceChanged = errors.New("Tap source changed during repository reconciliation")

// ReconciledSource is a read-only PDS observation for one source URI. A
// repository handler obtains it with GetRecord/listRecords; this API never
// writes to the PDS.
type ReconciledSource struct {
	URI                 syntax.ATURI
	DID                 syntax.DID
	ExpectedEventID     uint64
	ExpectedFingerprint [32]byte
	Revision            string
	CID                 syntax.CID
	Record              json.RawMessage
	Present             bool
}

// UncertainSources returns a bounded page for a read-only PDS reconciliation
// handler. The handler must finish every returned source before completing its
// leased repository job.
func (store *Store) UncertainSources(ctx context.Context, did syntax.DID, limit int) ([]SourceRecord, error) {
	if did == "" || limit <= 0 || limit > 1000 {
		return nil, errors.New("invalid uncertain source query")
	}
	rows, err := store.pool.Query(ctx, sourceSelect+`
		WHERE did=$1 AND ordering_status='uncertain'
		ORDER BY uri
		LIMIT $2
	`, did, limit)
	if err != nil {
		return nil, fmt.Errorf("list uncertain Tap sources: %w", err)
	}
	defer rows.Close()
	sources := make([]SourceRecord, 0, limit)
	for rows.Next() {
		source, err := sourceRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan uncertain Tap source: %w", err)
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate uncertain Tap sources: %w", err)
	}
	return sources, nil
}

// ReconcileSource resolves one order-uncertain source under the same owner
// fence and lifecycle transition used by live profile ingestion. A newer live
// Tap event wins and forces this repository job to retry from a fresh read.
func (service *Service) ReconcileSource(ctx context.Context, reconciled ReconciledSource) (tap.Outcome, error) {
	if reconciled.URI == "" || reconciled.DID == "" || reconciled.ExpectedEventID == 0 ||
		reconciled.ExpectedFingerprint == ([32]byte{}) || strings.TrimSpace(reconciled.Revision) == "" {
		return tap.Retryable(tap.ReasonProjectionFailure), errors.New("invalid reconciled source")
	}
	if reconciled.Present && (reconciled.CID == "" || len(reconciled.Record) == 0 || !json.Valid(reconciled.Record)) {
		return tap.Retryable(tap.ReasonProjectionFailure), errors.New("invalid authoritative PDS record")
	}
	expected, err := service.store.Source(ctx, reconciled.URI)
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	if expected.DID != reconciled.DID {
		return tap.Retryable(tap.ReasonProjectionFailure), errors.New("reconciled source owner mismatch")
	}
	if expected.SourceEventID != reconciled.ExpectedEventID ||
		!bytes.Equal(expected.SourceFingerprint[:], reconciled.ExpectedFingerprint[:]) {
		return tap.Retryable(tap.ReasonSourceOrderUncertain), ErrReconciliationSourceChanged
	}
	event := tap.Event{
		URI: expected.URI, DID: expected.DID, Collection: expected.Collection, Rkey: expected.Rkey,
		ID: expected.SourceEventID, Rev: strings.TrimSpace(reconciled.Revision), Live: expected.Live,
	}
	if reconciled.Present {
		event.Action = "update"
		event.CID = reconciled.CID
		event.Record = append(json.RawMessage(nil), reconciled.Record...)
	} else {
		event.Action = "delete"
	}
	fingerprint, err := recordFingerprint(event)
	if err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), err
	}
	lifecycle, err := service.lifecycles.EnsureOnboardingOwner(ctx, reconciled.DID)
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	now := service.store.now().UTC().Truncate(time.Microsecond)
	if lifecycle.State == ownerlifecycle.StateTerminal {
		var outcome tap.Outcome
		err := pgx.BeginFunc(ctx, service.store.pool, func(tx pgx.Tx) error {
			var reconcileErr error
			outcome, reconcileErr = service.store.reconcileSourceTx(ctx, tx, expected, event, fingerprint,
				sourceAuthority{Generation: lifecycle.Generation, State: string(lifecycle.State)}, now)
			return reconcileErr
		})
		if err != nil {
			return reconciliationFailure(err)
		}
		return outcome, nil
	}

	if expected.Collection == "social.craftsky.actor.profile" {
		target, transition := profileTransition(lifecycle.State, event.Action)
		if transition {
			var outcome tap.Outcome
			_, err := service.lifecycles.TransitionWith(ctx, ownerlifecycle.TransitionRequest{
				Owner: reconciled.DID, ExpectedGeneration: lifecycle.Generation,
				To: target, Reason: "profilePDSReconciled",
			}, func(ctx context.Context, tx pgx.Tx, before, after ownerlifecycle.Lifecycle) error {
				var reconcileErr error
				outcome, reconcileErr = service.store.reconcileSourceTx(ctx, tx, expected, event, fingerprint,
					sourceAuthority{Generation: after.Generation, State: string(after.State)}, now)
				if reconcileErr != nil {
					return reconcileErr
				}
				if service.profileParticipant != nil {
					return service.profileParticipant(ctx, tx, before, after)
				}
				return nil
			})
			if err != nil {
				return reconciliationFailure(err)
			}
			return outcome, nil
		}
	}

	var outcome tap.Outcome
	err = service.lifecycles.WithNonTerminalOwners(ctx, []syntax.DID{reconciled.DID}, func(ctx context.Context, tx pgx.Tx, existing map[syntax.DID]ownerlifecycle.Lifecycle) error {
		current, exists := existing[reconciled.DID]
		if !exists {
			return ownerlifecycle.ErrOwnerNotActive
		}
		var reconcileErr error
		outcome, reconcileErr = service.store.reconcileSourceTx(ctx, tx, expected, event, fingerprint,
			sourceAuthority{Generation: current.Generation, State: string(current.State)}, now)
		return reconcileErr
	})
	if err != nil {
		return reconciliationFailure(err)
	}
	return outcome, nil
}

func reconciliationFailure(err error) (tap.Outcome, error) {
	if errors.Is(err, ErrReconciliationSourceChanged) {
		return tap.Retryable(tap.ReasonSourceOrderUncertain), err
	}
	return tap.Retryable(tap.ReasonStorageUnavailable), err
}

func (store *Store) reconcileSourceTx(
	ctx context.Context,
	tx pgx.Tx,
	expected SourceRecord,
	event tap.Event,
	fingerprint [32]byte,
	authority sourceAuthority,
	now time.Time,
) (tap.Outcome, error) {
	current, err := sourceTx(ctx, tx, expected.URI)
	if err != nil {
		return tap.Outcome{}, err
	}
	if current.SourceEventID != expected.SourceEventID || !bytes.Equal(current.SourceFingerprint[:], expected.SourceFingerprint[:]) {
		return tap.Outcome{}, ErrReconciliationSourceChanged
	}
	outcome := tap.Applied()
	disposition := "eligible"
	state := "pending"
	var dependencyKind, dependencyKey, completedAt any
	if authority.State == string(ownerlifecycle.StateTerminal) {
		outcome = tap.PermanentInvalid(tap.ReasonOwnerTerminal)
		disposition = "denied_terminal"
		state = "permanent_denied"
		completedAt = now
	} else if event.Action != "delete" && authority.State != string(ownerlifecycle.StateActive) {
		outcome = tap.Blocked(tap.ReasonOwnerDeparted, tap.Dependency{Kind: "member_did", Key: event.DID.String()})
		disposition = "blocked_departed"
		state = "blocked"
		dependencyKind = outcome.Dependency.Kind
		dependencyKey = outcome.Dependency.Key
	}
	var cid, record any
	recordBytes := 0
	if event.Action != "delete" {
		cid = event.CID
		record = event.Record
		recordBytes = len(event.Record)
	}
	sourceResult, err := tx.Exec(ctx, `
		UPDATE tap_source_records
		SET source_fingerprint=$2,revision=$3,cid=$4,action=$5,record=$6,record_bytes=$7,
		    ordering_status='authoritative',projection_disposition=$8,
		    owner_generation=$9,updated_at=$10
		WHERE uri=$1
	`, event.URI, fingerprint[:], event.Rev, cid, event.Action, record, recordBytes,
		disposition, authority.Generation, now)
	if err != nil {
		return tap.Outcome{}, fmt.Errorf("install reconciled Tap source: %w", err)
	}
	if sourceResult.RowsAffected() != 1 {
		return tap.Outcome{}, ErrReconciliationSourceChanged
	}
	jobResult, err := tx.Exec(ctx, `
		UPDATE tap_projection_jobs
		SET state=$2,dependency_kind=$3,dependency_key=$4,attempts=0,
		    next_attempt_at=$5,lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
		    last_reason_code=$6,completed_at=$7,updated_at=$5
		WHERE source_uri=$1 AND source_event_id=$8
	`, event.URI, state, dependencyKind, dependencyKey, now,
		nullableString(string(outcome.Reason)), completedAt, event.ID)
	if err != nil {
		return tap.Outcome{}, fmt.Errorf("wake reconciled Tap projection: %w", err)
	}
	if jobResult.RowsAffected() != 1 {
		return tap.Outcome{}, ErrReconciliationSourceChanged
	}
	return outcome, nil
}
