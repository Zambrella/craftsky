package ingestion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

var (
	errProfileSourceDidNotWin          = errors.New("profile source did not win current state")
	errProfileSourceChangedDuringRetry = errors.New("profile source changed during fenced retry")
)

type ServiceConfig struct {
	Store               *Store
	Lifecycles          *ownerlifecycle.Store
	ProfileParticipant  ownerlifecycle.TransitionParticipant
	TerminalParticipant ownerlifecycle.TerminalParticipant
	TerminalComponents  []ownerlifecycle.PurgeComponent
}

// Service is the production Tap DurableIngestor. Profile and terminal identity
// events compose durable source/receipt state into the owner lifecycle's
// fenced transaction; ordinary records use Store's source-first transaction.
type Service struct {
	store               *Store
	lifecycles          *ownerlifecycle.Store
	profileParticipant  ownerlifecycle.TransitionParticipant
	terminalParticipant ownerlifecycle.TerminalParticipant
	terminalComponents  []ownerlifecycle.PurgeComponent
}

var _ tap.DurableIngestor = (*Service)(nil)

func NewService(config ServiceConfig) (*Service, error) {
	if config.Store == nil || config.Lifecycles == nil || config.ProfileParticipant == nil ||
		config.TerminalParticipant == nil || len(config.TerminalComponents) == 0 {
		return nil, errors.New("ingestion service requires store, lifecycle store, lifecycle participants, and terminal purge catalogue")
	}
	config.Store.lifecycleAware = true
	return &Service{
		store:               config.Store,
		lifecycles:          config.Lifecycles,
		profileParticipant:  config.ProfileParticipant,
		terminalParticipant: config.TerminalParticipant,
		terminalComponents:  append([]ownerlifecycle.PurgeComponent(nil), config.TerminalComponents...),
	}, nil
}

func (service *Service) IngestRecord(ctx context.Context, event tap.Event) (tap.Outcome, error) {
	if event.Collection != "social.craftsky.actor.profile" {
		if err := validateRecordEvent(event); err != nil {
			return tap.Retryable(tap.ReasonMalformedRecord), err
		}
		lifecycle, err := service.lifecycles.EnsureOnboardingOwner(ctx, event.DID)
		if err != nil {
			return tap.Retryable(tap.ReasonStorageUnavailable), err
		}
		fingerprint, err := recordFingerprint(event)
		if err != nil {
			return tap.Retryable(tap.ReasonMalformedRecord), err
		}
		now := service.store.now().UTC().Truncate(time.Microsecond)
		var outcome tap.Outcome
		if lifecycle.State == ownerlifecycle.StateTerminal {
			err = pgx.BeginFunc(ctx, service.store.pool, func(tx pgx.Tx) error {
				var ingestErr error
				outcome, ingestErr = service.store.ingestRecordTx(ctx, tx, event, fingerprint, now, &sourceAuthority{
					Generation: lifecycle.Generation, State: string(lifecycle.State),
				})
				return ingestErr
			})
		} else {
			err = service.lifecycles.WithNonTerminalOwners(ctx, []syntax.DID{event.DID}, func(ctx context.Context, tx pgx.Tx, existing map[syntax.DID]ownerlifecycle.Lifecycle) error {
				current, exists := existing[event.DID]
				if !exists {
					return ownerlifecycle.ErrOwnerNotActive
				}
				var ingestErr error
				outcome, ingestErr = service.store.ingestRecordTx(ctx, tx, event, fingerprint, now, &sourceAuthority{
					Generation: current.Generation, State: string(current.State),
				})
				return ingestErr
			})
		}
		if err != nil {
			return tap.Retryable(tap.ReasonStorageUnavailable), err
		}
		return outcome, nil
	}
	if err := validateRecordEvent(event); err != nil {
		return tap.Retryable(tap.ReasonMalformedRecord), err
	}
	lifecycle, err := service.lifecycles.EnsureOnboardingOwner(ctx, event.DID)
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	if lifecycle.State == ownerlifecycle.StateTerminal {
		return service.store.ingestTerminalDeniedProfile(ctx, event, lifecycle.Generation)
	}

	target, transition := profileTransition(lifecycle.State, event.Action)
	if transition {
		var outcome tap.Outcome
		_, err := service.lifecycles.TransitionWith(ctx, ownerlifecycle.TransitionRequest{
			Owner: event.DID, ExpectedGeneration: lifecycle.Generation,
			To: target, Reason: profileTransitionReason(event.Action),
		}, func(ctx context.Context, tx pgx.Tx, before, after ownerlifecycle.Lifecycle) error {
			var won bool
			var writeErr error
			outcome, won, writeErr = service.store.ingestProfileTx(ctx, tx, event, after.Generation, service.store.now().UTC().Truncate(time.Microsecond), true)
			if writeErr != nil {
				return writeErr
			}
			if !won {
				return errProfileSourceDidNotWin
			}
			if service.profileParticipant != nil {
				return service.profileParticipant(ctx, tx, before, after)
			}
			return nil
		})
		if errors.Is(err, errProfileSourceDidNotWin) {
			// The lifecycle update rolled back. Persist duplicate/stale/uncertain
			// recognition under the same owner fence without changing authority.
			// Profile source order is part of the lifecycle decision, so even the
			// no-transition outcome must not fall back to an unfenced store write.
			return service.ingestProfileWithoutTransition(ctx, event, true)
		}
		if err != nil {
			return tap.Retryable(tap.ReasonStorageUnavailable), err
		}
		return outcome, nil
	}

	return service.ingestProfileWithoutTransition(ctx, event, false)
}

func (service *Service) ingestProfileWithoutTransition(ctx context.Context, event tap.Event, rejectWinner bool) (tap.Outcome, error) {
	var outcome tap.Outcome
	err := service.lifecycles.WithNonTerminalOwners(ctx, []syntax.DID{event.DID}, func(ctx context.Context, tx pgx.Tx, existing map[syntax.DID]ownerlifecycle.Lifecycle) error {
		current, exists := existing[event.DID]
		if !exists || current.State == ownerlifecycle.StateTerminal {
			return ownerlifecycle.ErrTerminalOwner
		}
		var writeErr error
		var won bool
		outcome, won, writeErr = service.store.ingestProfileTx(ctx, tx, event, current.Generation, service.store.now().UTC().Truncate(time.Microsecond), true)
		if writeErr == nil && rejectWinner && won {
			return errProfileSourceChangedDuringRetry
		}
		return writeErr
	})
	if errors.Is(err, ownerlifecycle.ErrTerminalOwner) {
		current, getErr := service.lifecycles.Get(ctx, event.DID)
		if getErr != nil {
			return tap.Retryable(tap.ReasonStorageUnavailable), getErr
		}
		return service.store.ingestTerminalDeniedProfile(ctx, event, current.Generation)
	}
	if err != nil {
		if errors.Is(err, errProfileSourceChangedDuringRetry) {
			return tap.Retryable(tap.ReasonSourceOrderUncertain), err
		}
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	return outcome, nil
}

func profileTransition(state ownerlifecycle.State, action string) (ownerlifecycle.State, bool) {
	if action == "delete" {
		if state == ownerlifecycle.StateActive || state == ownerlifecycle.StateDeletionPending {
			return ownerlifecycle.StateDeparted, true
		}
		return state, false
	}
	switch state {
	case ownerlifecycle.StateDeparted:
		return ownerlifecycle.StateActive, true
	default:
		return state, false
	}
}

func profileTransitionReason(action string) string {
	if action == "delete" {
		return "profileDeleted"
	}
	return "profileActivated"
}

func (service *Service) IngestIdentity(ctx context.Context, event tap.IdentityEvent) (tap.Outcome, error) {
	if event.ID == 0 || event.DID == "" {
		return tap.Retryable(tap.ReasonInvalidIdentity), errors.New("invalid Tap identity event")
	}
	if event.Status != "deleted" {
		return service.store.ingestOrdinaryIdentity(ctx, event)
	}
	fingerprint, err := identityFingerprint(event)
	if err != nil {
		return tap.Retryable(tap.ReasonInvalidIdentity), err
	}
	now := service.store.now().UTC().Truncate(time.Microsecond)
	_, err = service.lifecycles.TerminalizeWith(ctx, ownerlifecycle.TerminalizeRequest{
		Owner: event.DID, Reason: "tapIdentityDeleted", Components: service.terminalComponents,
	}, func(ctx context.Context, tx pgx.Tx, before *ownerlifecycle.Lifecycle, terminal ownerlifecycle.Lifecycle) error {
		if err := insertReceipt(ctx, tx, fingerprint, event.ID, "identity", tap.Applied(), "", tap.ReasonNone, now); err != nil {
			return err
		}
		if service.terminalParticipant != nil {
			return service.terminalParticipant(ctx, tx, before, terminal)
		}
		return nil
	})
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	return tap.Applied(), nil
}

func (service *Service) Quarantine(ctx context.Context, event tap.InvalidEvent) (tap.Outcome, error) {
	return service.store.Quarantine(ctx, event)
}

func (store *Store) ingestOrdinaryIdentity(ctx context.Context, event tap.IdentityEvent) (tap.Outcome, error) {
	fingerprint, err := identityFingerprint(event)
	if err != nil {
		return tap.Retryable(tap.ReasonInvalidIdentity), err
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	outcome := tap.Applied()
	err = pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		return insertReceipt(ctx, tx, fingerprint, event.ID, "identity", outcome, "", tap.ReasonNone, now)
	})
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	return outcome, nil
}

func identityFingerprint(event tap.IdentityEvent) ([32]byte, error) {
	encoded, err := json.Marshal(struct {
		ID       uint64 `json:"id"`
		DID      string `json:"did"`
		Handle   string `json:"handle,omitempty"`
		IsActive bool   `json:"isActive"`
		Status   string `json:"status"`
	}{event.ID, event.DID.String(), event.Handle, event.IsActive, event.Status})
	if err != nil {
		return [32]byte{}, fmt.Errorf("fingerprint Tap identity: %w", err)
	}
	return sha256.Sum256(encoded), nil
}

func (store *Store) ingestProfileTx(
	ctx context.Context,
	tx pgx.Tx,
	event tap.Event,
	generation int64,
	now time.Time,
	enqueueAddRepo bool,
) (tap.Outcome, bool, error) {
	fingerprint, err := recordFingerprint(event)
	if err != nil {
		return tap.Outcome{}, false, err
	}
	var currentID int64
	var currentFingerprint []byte
	readErr := tx.QueryRow(ctx, `
		SELECT source_event_id,source_fingerprint
		FROM tap_source_records WHERE uri=$1 FOR UPDATE
	`, event.URI).Scan(&currentID, &currentFingerprint)
	switch {
	case readErr == nil && currentID > int64(event.ID):
		outcome := tap.Applied()
		return outcome, false, insertReceipt(ctx, tx, fingerprint, event.ID, "record", outcome, event.URI, tap.ReasonStaleSource, now)
	case readErr == nil && currentID == int64(event.ID) && bytes.Equal(currentFingerprint, fingerprint[:]):
		outcome, err := readCurrentJobOutcome(ctx, tx, event.URI)
		if err != nil {
			return tap.Outcome{}, false, err
		}
		return outcome, false, insertReceipt(ctx, tx, fingerprint, event.ID, "record", outcome, event.URI, outcome.Reason, now)
	case readErr == nil && currentID == int64(event.ID):
		outcome, err := markSourceConflictTx(ctx, tx, event, fingerprint, now)
		return outcome, false, err
	case readErr != nil && !errors.Is(readErr, pgx.ErrNoRows):
		return tap.Outcome{}, false, fmt.Errorf("read current profile Tap source: %w", readErr)
	}

	record := any(nil)
	cid := any(nil)
	recordBytes := 0
	if event.Action != "delete" {
		record = event.Record
		cid = event.CID
		recordBytes = len(event.Record)
	} else if event.CID != "" {
		cid = event.CID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tap_source_records(
			uri,did,collection,rkey,source_event_id,source_fingerprint,
			revision,cid,action,record,record_bytes,live,ordering_status,
			projection_disposition,owner_generation,observed_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'authoritative','eligible',$13,$14,$14)
		ON CONFLICT(uri) DO UPDATE SET
			did=EXCLUDED.did,collection=EXCLUDED.collection,rkey=EXCLUDED.rkey,
			source_event_id=EXCLUDED.source_event_id,
			source_fingerprint=EXCLUDED.source_fingerprint,revision=EXCLUDED.revision,
			cid=EXCLUDED.cid,action=EXCLUDED.action,record=EXCLUDED.record,
			record_bytes=EXCLUDED.record_bytes,
			live=EXCLUDED.live,ordering_status=EXCLUDED.ordering_status,
			projection_disposition=EXCLUDED.projection_disposition,
			owner_generation=EXCLUDED.owner_generation,
			observed_at=EXCLUDED.observed_at,updated_at=EXCLUDED.updated_at
	`, event.URI, event.DID, event.Collection, event.Rkey, event.ID, fingerprint[:],
		event.Rev, cid, event.Action, record, recordBytes, event.Live, generation, now); err != nil {
		return tap.Outcome{}, false, fmt.Errorf("upsert profile Tap source: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tap_projection_jobs(
			source_uri,projection_kind,source_event_id,state,next_attempt_at,created_at,updated_at
		) VALUES($1,$2,$3,'pending',$4,$4,$4)
		ON CONFLICT(source_uri,projection_kind) DO UPDATE SET
			source_event_id=EXCLUDED.source_event_id,state='pending',
			dependency_kind=NULL,dependency_key=NULL,attempts=0,
			next_attempt_at=EXCLUDED.next_attempt_at,
			lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
			last_reason_code=NULL,completed_at=NULL,updated_at=EXCLUDED.updated_at
	`, event.URI, projectionKind(event.Collection), event.ID, now); err != nil {
		return tap.Outcome{}, false, fmt.Errorf("upsert profile Tap projection job: %w", err)
	}
	if enqueueAddRepo && event.Action != "delete" {
		if err := enqueueRepositoryJob(ctx, tx, event.DID, string(RepositoryJobTapAddRepo), now); err != nil {
			return tap.Outcome{}, false, err
		}
	}
	outcome := tap.Applied()
	if err := insertReceipt(ctx, tx, fingerprint, event.ID, "record", outcome, event.URI, tap.ReasonNone, now); err != nil {
		return tap.Outcome{}, false, err
	}
	return outcome, true, nil
}

func (store *Store) ingestTerminalDeniedProfile(ctx context.Context, event tap.Event, generation int64) (tap.Outcome, error) {
	now := store.now().UTC().Truncate(time.Microsecond)
	outcome := tap.Applied()
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		_, _, err := store.ingestProfileTx(ctx, tx, event, generation, now, false)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE tap_source_records
			SET projection_disposition='denied_terminal',owner_generation=$2,updated_at=$3
			WHERE uri=$1
		`, event.URI, generation, now); err != nil {
			return fmt.Errorf("deny terminal profile source: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE tap_projection_jobs
			SET state='permanent_denied',dependency_kind=NULL,dependency_key=NULL,
			    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
			    last_reason_code=$2,completed_at=COALESCE(completed_at,$3),updated_at=$3
			WHERE source_uri=$1
		`, event.URI, tap.ReasonOwnerTerminal, now); err != nil {
			return fmt.Errorf("deny terminal profile projection: %w", err)
		}
		return nil
	})
	if err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), err
	}
	return outcome, nil
}
