package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/tap"
)

var ErrReconciliationSourceChanged = errors.New("tap source changed during repository reconciliation")

// ReconciledSource is a read-only PDS observation for one source URI. A
// repository handler obtains it with GetRecord/listRecords; this API never
// writes to the PDS.
type ReconciledSource struct {
	URI                 syntax.ATURI
	DID                 syntax.DID
	ExpectedEventID     uint64
	ExpectedFingerprint [32]byte
	Revision            syntax.TID
	CID                 syntax.CID
	Record              json.RawMessage
	Present             bool
}

// RepositoryReconciliationSources returns a bounded page for a read-only PDS
// reconciliation handler. It includes sources whose retained content ordering
// is uncertain and sources whose projection requested an authoritative
// repository read (for example after an owner-generation change). The handler
// must finish every returned source before completing its leased repository
// job.
func (store *Store) RepositoryReconciliationSources(ctx context.Context, did syntax.DID, limit int) ([]SourceRecord, error) {
	if did == "" || limit <= 0 || limit > 1000 {
		return nil, errors.New("invalid repository reconciliation source query")
	}
	rows, err := store.pool.Query(ctx, sourceSelect+`
		WHERE did=$1 AND (
			ordering_status='uncertain'
			OR EXISTS (
				SELECT 1
				FROM tap_projection_jobs AS job
				WHERE job.source_uri=tap_source_records.uri
				  AND job.source_event_id=tap_source_records.source_event_id
				  AND job.state='blocked'
				  AND job.dependency_kind='repository_did'
				  AND job.dependency_key=tap_source_records.did
			)
		)
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
		reconciled.ExpectedFingerprint == ([32]byte{}) {
		return tap.Retryable(tap.ReasonProjectionFailure), errors.New("invalid reconciled source")
	}
	if _, err := syntax.ParseTID(reconciled.Revision.String()); err != nil {
		return tap.Retryable(tap.ReasonProjectionFailure), errors.New("invalid reconciled source revision")
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
		ID: expected.SourceEventID, Rev: reconciled.Revision, Live: expected.Live,
	}
	if reconciled.Present {
		event.Action = "update"
		event.CID = reconciled.CID
		event.Record = append(json.RawMessage(nil), reconciled.Record...)
	} else {
		event.Action = "delete"
	}
	fingerprint, err := repositorySourceFingerprint(event)
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
		err := service.lifecycles.WithOwnerStates(ctx, []syntax.DID{reconciled.DID}, func(
			ctx context.Context,
			tx pgx.Tx,
			states map[syntax.DID]ownerlifecycle.Lifecycle,
		) error {
			current, exists := states[reconciled.DID]
			if !exists {
				return ownerlifecycle.ErrOwnerNotActive
			}
			var reconcileErr error
			outcome, reconcileErr = service.store.reconcileSourceTx(
				ctx, tx, expected, event, fingerprint,
				sourceAuthority{
					Lifecycle: current, Authoritative: true,
					LockedOperationID: expected.EffectOperationID,
				},
				now,
			)
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
				outcome, reconcileErr = service.store.reconcileSourceTx(
					ctx, tx, expected, event, fingerprint,
					sourceAuthority{
						Lifecycle: after, Authoritative: true,
						LockedOperationID: expected.EffectOperationID,
					},
					now,
				)
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
		outcome, reconcileErr = service.store.reconcileSourceTx(
			ctx, tx, expected, event, fingerprint,
			sourceAuthority{
				Lifecycle: current, Authoritative: true,
				LockedOperationID: expected.EffectOperationID,
			},
			now,
		)
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
	if errors.Is(err, ownerlifecycle.ErrEffectSourceAmbiguous) {
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
	orderingStatus := "authoritative"
	var projectionGeneration any
	independentBusiness := isIndependentBusinessCollection(event.Collection)
	if !independentBusiness {
		projectionGeneration = authority.Lifecycle.Generation
	}
	var effectOperationID any
	state := "pending"
	var dependencyKind, dependencyKey, completedAt any
	if authority.Lifecycle.State == ownerlifecycle.StateTerminal {
		outcome = tap.PermanentInvalid(tap.ReasonOwnerTerminal)
		disposition = "denied_terminal"
		state = "permanent_denied"
		completedAt = now
	} else if event.Action != "delete" && authority.Lifecycle.State != ownerlifecycle.StateActive &&
		(authority.Lifecycle.State != ownerlifecycle.StateDeparted || !independentBusiness) {
		outcome = tap.Blocked(tap.ReasonOwnerDeparted, tap.Dependency{Kind: "member_did", Key: event.DID.String()})
		disposition = "blocked_departed"
		state = "blocked"
		dependencyKind = outcome.Dependency.Kind
		dependencyKey = outcome.Dependency.Key
	}
	if event.Action != "delete" && !independentBusiness {
		recordContentFingerprint, err := pdseffects.RecordContentFingerprint(
			event.DID, event.Collection, event.Rkey, event.Record,
		)
		if err != nil {
			return tap.Outcome{}, err
		}
		resolution, err := ownerlifecycle.ResolvePDSRecordSourceTx(
			ctx,
			tx,
			authority.Lifecycle,
			ownerlifecycle.PDSRecordSourceObservation{
				Owner: event.DID, URI: event.URI, CID: event.CID,
				RecordFingerprint: recordContentFingerprint,
				LockedOperationID: authority.LockedOperationID,
				Authoritative:     true,
			},
			now,
		)
		if err != nil {
			return tap.Outcome{}, err
		}
		if resolution.Match == ownerlifecycle.EffectSourceAmbiguous {
			return tap.Outcome{}, ownerlifecycle.ErrEffectSourceAmbiguous
		}
		if resolution.Match == ownerlifecycle.EffectSourceMatched {
			effectOperationID = resolution.Attempt.OperationID
			switch resolution.Attempt.ProjectionDisposition {
			case ownerlifecycle.ProjectionEligibleCurrent:
				outcome = tap.Applied()
				disposition = "eligible"
				state = "pending"
				dependencyKind, dependencyKey, completedAt = nil, nil, nil
			case ownerlifecycle.ProjectionHiddenNonActive:
				outcome = tap.Blocked(
					tap.ReasonOwnerDeparted,
					tap.Dependency{Kind: "member_did", Key: event.DID.String()},
				)
				disposition = "blocked_departed"
				state = "blocked"
				dependencyKind, dependencyKey = outcome.Dependency.Kind, outcome.Dependency.Key
			case ownerlifecycle.ProjectionDeniedTerminal:
				outcome = tap.PermanentInvalid(tap.ReasonOwnerTerminal)
				disposition = "denied_terminal"
				state = "permanent_denied"
				completedAt = now
			case ownerlifecycle.ProjectionNotApplicable:
				outcome = tap.PermanentInvalid(tap.ReasonStaleSource)
				disposition = "not_accepted"
				state = "permanent_denied"
				completedAt = now
			default:
				return tap.Outcome{}, ownerlifecycle.ErrEffectSourceAmbiguous
			}
		}
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
		    ordering_status=$8,projection_disposition=$9,
		    projection_generation=$10,effect_operation_id=$11,updated_at=$12
		WHERE uri=$1
	`, event.URI, fingerprint[:], event.Rev, cid, event.Action, record, recordBytes,
		orderingStatus, disposition, projectionGeneration, effectOperationID, now)
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
