package relationships

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists private actor mutes and reads the Tap-owned public block
// projection. Mute errors deliberately omit relationship identifiers.
type Store struct {
	pool *pgxpool.Pool
}

type BlockRecord struct {
	URI        syntax.ATURI
	BlockerDID syntax.DID
	Rkey       syntax.RecordKey
	CID        syntax.CID
	SubjectDID syntax.DID
	CreatedAt  time.Time
}

type ListItem struct {
	SubjectDID syntax.DID
	CreatedAt  time.Time
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Mute idempotently stores an owner-private actor mute.
func (s *Store) Mute(ctx context.Context, owner, subject syntax.DID) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO actor_mutes (owner_did, subject_did)
		VALUES ($1, $2)
		ON CONFLICT (owner_did, subject_did) DO UPDATE
		SET updated_at = now()
	`, owner, subject); err != nil {
		return fmt.Errorf("mute actor: %w", err)
	}
	return nil
}

// MuteAndCancelPendingDeliveries makes the private relationship and its
// delivery consequence effective atomically.
func (s *Store) MuteAndCancelPendingDeliveries(ctx context.Context, owner, subject syntax.DID) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin mute actor: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO actor_mutes (owner_did, subject_did)
		VALUES ($1, $2)
		ON CONFLICT (owner_did, subject_did) DO UPDATE
		SET updated_at = now()
	`, owner, subject); err != nil {
		return 0, fmt.Errorf("mute actor: %w", err)
	}
	cancelTag, err := tx.Exec(ctx, `
		UPDATE push_deliveries delivery
		SET status = 'cancelled', lease_owner = NULL, lease_expires_at = NULL, updated_at = now()
		FROM notification_events event
		WHERE delivery.notification_id = event.id
		  AND delivery.status IN ('pending', 'retry', 'leased')
		  AND event.recipient_did = $1
		  AND event.actor_did = $2
	`, owner, subject)
	if err != nil {
		return 0, fmt.Errorf("cancel protected notification deliveries: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit mute actor: %w", err)
	}
	return cancelTag.RowsAffected(), nil
}

// Unmute idempotently removes only the authenticated owner's actor mute.
func (s *Store) Unmute(ctx context.Context, owner, subject syntax.DID) error {
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM actor_mutes
		WHERE owner_did = $1 AND subject_did = $2
	`, owner, subject); err != nil {
		return fmt.Errorf("unmute actor: %w", err)
	}
	return nil
}

// IsMuted reports the private mute state for exactly one owner and subject.
func (s *Store) IsMuted(ctx context.Context, owner, subject syntax.DID) (bool, error) {
	var muted bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM actor_mutes
			WHERE owner_did = $1 AND subject_did = $2
		)
	`, owner, subject).Scan(&muted); err != nil {
		return false, fmt.Errorf("read actor mute: %w", err)
	}
	return muted, nil
}

// State returns only the requesting viewer's private mute plus the two public
// block directions. Duplicate compatible block records collapse through
// EXISTS and cannot amplify or leak another owner's mute state.
func (s *Store) State(ctx context.Context, viewer, subject syntax.DID) (State, error) {
	var state State
	if err := s.pool.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1 FROM actor_mutes
				WHERE owner_did = $1 AND subject_did = $2
			),
			EXISTS (
				SELECT 1 FROM atproto_blocks
				WHERE blocker_did = $1 AND subject_did = $2
			),
			EXISTS (
				SELECT 1 FROM atproto_blocks
				WHERE blocker_did = $2 AND subject_did = $1
			)
	`, viewer, subject).Scan(&state.Muted, &state.Blocking, &state.BlockedBy); err != nil {
		return State{}, fmt.Errorf("read relationship state: %w", err)
	}
	return state, nil
}

// OwnedBlockRecords returns only caller-owned indexed identities for one
// subject. It never exposes an inbound block as deletable by the caller.
func (s *Store) OwnedBlockRecords(ctx context.Context, blocker, subject syntax.DID) ([]BlockRecord, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT uri, blocker_did, rkey, cid, subject_did, created_at
		FROM atproto_blocks
		WHERE blocker_did = $1 AND subject_did = $2
		ORDER BY uri ASC
	`, blocker, subject)
	if err != nil {
		return nil, fmt.Errorf("list owned block records: %w", err)
	}
	defer rows.Close()

	records := make([]BlockRecord, 0)
	for rows.Next() {
		var record BlockRecord
		if err := rows.Scan(
			&record.URI,
			&record.BlockerDID,
			&record.Rkey,
			&record.CID,
			&record.SubjectDID,
			&record.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan owned block record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owned block records: %w", err)
	}
	return records, nil
}

func (s *Store) ListMutes(
	ctx context.Context,
	owner syntax.DID,
	limit int,
	afterCreated time.Time,
	afterSubject syntax.DID,
) ([]ListItem, bool, error) {
	var after any
	if !afterCreated.IsZero() {
		after = afterCreated
	}
	rows, err := s.pool.Query(ctx, `
		SELECT m.subject_did, m.created_at
		FROM actor_mutes m
		JOIN craftsky_profiles cp ON cp.did = m.subject_did
		WHERE m.owner_did = $1
		  AND ($2::timestamptz IS NULL
		       OR (m.created_at, m.subject_did) < ($2::timestamptz, $3::text))
		ORDER BY m.created_at DESC, m.subject_did DESC
		LIMIT $4
	`, owner, after, afterSubject, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("list owned mutes: %w", err)
	}
	defer rows.Close()

	items, err := scanListItems(rows)
	if err != nil {
		return nil, false, fmt.Errorf("list owned mutes: %w", err)
	}
	return trimListItems(items, limit)
}

func (s *Store) ListBlocks(
	ctx context.Context,
	owner syntax.DID,
	limit int,
	afterCreated time.Time,
	afterSubject syntax.DID,
) ([]ListItem, bool, error) {
	var after any
	if !afterCreated.IsZero() {
		after = afterCreated
	}
	rows, err := s.pool.Query(ctx, `
		WITH owned_subjects AS (
			SELECT DISTINCT ON (b.subject_did)
				b.subject_did, b.created_at
			FROM atproto_blocks b
			WHERE b.blocker_did = $1
			ORDER BY b.subject_did, b.created_at DESC, b.uri DESC
		)
		SELECT owned.subject_did, owned.created_at
		FROM owned_subjects owned
		JOIN craftsky_profiles cp ON cp.did = owned.subject_did
		WHERE ($2::timestamptz IS NULL
		       OR (owned.created_at, owned.subject_did) < ($2::timestamptz, $3::text))
		ORDER BY owned.created_at DESC, owned.subject_did DESC
		LIMIT $4
	`, owner, after, afterSubject, limit+1)
	if err != nil {
		return nil, false, fmt.Errorf("list owned blocks: %w", err)
	}
	defer rows.Close()

	items, err := scanListItems(rows)
	if err != nil {
		return nil, false, fmt.Errorf("list owned blocks: %w", err)
	}
	return trimListItems(items, limit)
}

type listItemRows interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanListItems(rows listItemRows) ([]ListItem, error) {
	items := make([]ListItem, 0)
	for rows.Next() {
		var item ListItem
		if err := rows.Scan(&item.SubjectDID, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan relationship list item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relationship list items: %w", err)
	}
	return items, nil
}

func trimListItems(items []ListItem, limit int) ([]ListItem, bool, error) {
	if len(items) <= limit {
		return items, false, nil
	}
	return items[:limit], true, nil
}
