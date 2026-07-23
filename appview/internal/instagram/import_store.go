package instagram

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInstagramImportInactive      = errors.New("Instagram import inactive")
	ErrInvalidInstagramImportCursor = errors.New("invalid Instagram import cursor")
)

type ImportSourceType string

const (
	ImportSourceManual        ImportSourceType = "manual"
	ImportSourceInstagramJSON ImportSourceType = "instagramJson"
)

func (s ImportSourceType) Valid() bool {
	return s == ImportSourceManual || s == ImportSourceInstagramJSON
}

type GraphImport struct {
	ID             uuid.UUID
	OwnerDID       syntax.DID
	State          InstagramImportState
	SourceType     ImportSourceType
	FollowingCount int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (GraphImport) String() string {
	return "Instagram graph import [REDACTED]"
}
func (i GraphImport) GoString() string { return i.String() }

type ImportCounts struct {
	Following int `json:"following"`
}

type CreateImportResult struct {
	Import                 GraphImport
	Counts                 ImportCounts
	InitialSuggestionCount int
}

type CreateImportParams struct {
	ID         uuid.UUID
	OwnerDID   syntax.DID
	SourceType ImportSourceType
	Entries    []ImportEntry
	Now        time.Time
}

type UpdateImportParams struct {
	Reactivate *bool
	Now        time.Time
}

type ImportCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type ImportStore struct {
	pool *pgxpool.Pool
}

func NewImportStore(pool *pgxpool.Pool) *ImportStore {
	return &ImportStore{pool: pool}
}

func (s *ImportStore) CreateImport(ctx context.Context, params CreateImportParams) (CreateImportResult, error) {
	return s.createImport(ctx, params)
}

func (s *ImportStore) CreateImportForMatching(ctx context.Context, params CreateImportParams) (CreateImportResult, error) {
	return s.createImport(ctx, params)
}

func (s *ImportStore) createImport(ctx context.Context, params CreateImportParams) (CreateImportResult, error) {
	if s == nil || s.pool == nil {
		return CreateImportResult{}, errors.New("Instagram import store is unavailable")
	}
	if params.ID == uuid.Nil || params.OwnerDID == "" || !params.SourceType.Valid() || params.Now.IsZero() {
		return CreateImportResult{}, errors.New("invalid Instagram import parameters")
	}
	entries, err := NormalizeImportEntries(params.Entries)
	if err != nil {
		return CreateImportResult{}, err
	}
	if len(entries) == 0 {
		return CreateImportResult{}, ErrInvalidInstagramUsername
	}
	counts := ImportCounts{Following: len(entries)}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CreateImportResult{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 2))`, params.OwnerDID); err != nil {
		return CreateImportResult{}, err
	}
	var verified bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM instagram_account_links
			WHERE owner_did = $1 AND state = 'active' AND verified_at IS NOT NULL
		)
	`, params.OwnerDID).Scan(&verified); err != nil {
		return CreateImportResult{}, err
	}
	if !verified {
		return CreateImportResult{}, ErrInstagramVerificationRequired
	}
	graphImport, err := scanGraphImport(tx.QueryRow(ctx, `
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, following_count,
			created_at, updated_at
		) VALUES ($1, $2, 'active', $3, $4, $5, $5)
		RETURNING `+graphImportColumns,
		params.ID, params.OwnerDID, params.SourceType, counts.Following, params.Now))
	if err != nil {
		return CreateImportResult{}, err
	}
	for _, entry := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO instagram_graph_handles (
				import_id, username_normalized, matched, created_at
			) VALUES ($1, $2, false, $3)
		`, params.ID, entry.Username, params.Now); err != nil {
			return CreateImportResult{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return CreateImportResult{}, err
	}
	return CreateImportResult{Import: graphImport, Counts: counts}, nil
}

func (s *ImportStore) FinalizeImportMatching(ctx context.Context, owner syntax.DID, id uuid.UUID, now time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram import store is unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `
		SELECT true
		FROM instagram_graph_imports
		WHERE id = $1 AND owner_did = $2
		FOR UPDATE
	`, id, owner).Scan(new(bool)); errors.Is(err, pgx.ErrNoRows) {
		return ErrInstagramResourceNotFound
	} else if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE instagram_graph_imports SET updated_at = $2 WHERE id = $1`, id, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ImportStore) GetImport(ctx context.Context, owner syntax.DID, id uuid.UUID, now time.Time) (GraphImport, error) {
	if s == nil || s.pool == nil {
		return GraphImport{}, errors.New("Instagram import store is unavailable")
	}
	graphImport, err := scanGraphImport(s.pool.QueryRow(ctx, `
		SELECT `+graphImportColumns+`
		FROM instagram_graph_imports
		WHERE id = $1 AND owner_did = $2
	`, id, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		return GraphImport{}, ErrInstagramResourceNotFound
	}
	return graphImport, err
}

func (s *ImportStore) ListImports(ctx context.Context, owner syntax.DID, limit int, after *ImportCursor, now time.Time) ([]GraphImport, *ImportCursor, error) {
	if s == nil || s.pool == nil {
		return nil, nil, errors.New("Instagram import store is unavailable")
	}
	if owner == "" || limit < 1 || now.IsZero() {
		return nil, nil, errors.New("invalid Instagram import list parameters")
	}
	if after != nil {
		if after.ID == uuid.Nil || after.CreatedAt.IsZero() {
			return nil, nil, ErrInvalidInstagramImportCursor
		}
		var present bool
		err := s.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM instagram_graph_imports
				WHERE owner_did = $1 AND id = $2 AND created_at = $3
			)
		`, owner, after.ID, after.CreatedAt).Scan(&present)
		if err != nil {
			return nil, nil, err
		}
		if !present {
			return nil, nil, ErrInvalidInstagramImportCursor
		}
	}

	query := `
		SELECT ` + graphImportColumns + `
		FROM instagram_graph_imports
		WHERE owner_did = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2`
	args := []any{owner, limit + 1}
	if after != nil {
		query = `
			SELECT ` + graphImportColumns + `
			FROM instagram_graph_imports
			WHERE owner_did = $1 AND (created_at, id) < ($3, $4)
			ORDER BY created_at DESC, id DESC
			LIMIT $2`
		args = append(args, after.CreatedAt, after.ID)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	items := make([]GraphImport, 0, limit+1)
	for rows.Next() {
		item, err := scanGraphImport(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if len(items) <= limit {
		return items, nil, nil
	}
	items = items[:limit]
	last := items[len(items)-1]
	return items, &ImportCursor{CreatedAt: last.CreatedAt, ID: last.ID}, nil
}

func (s *ImportStore) UpdateImport(ctx context.Context, owner syntax.DID, id uuid.UUID, params UpdateImportParams) (GraphImport, error) {
	if s == nil || s.pool == nil {
		return GraphImport{}, errors.New("Instagram import store is unavailable")
	}
	if params.Reactivate == nil || !*params.Reactivate {
		return GraphImport{}, errors.New("Instagram import update is empty")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return GraphImport{}, err
	}
	defer tx.Rollback(ctx)
	graphImport, err := scanGraphImport(tx.QueryRow(ctx, `
		SELECT `+graphImportColumns+`
		FROM instagram_graph_imports
		WHERE id = $1 AND owner_did = $2
		FOR UPDATE
	`, id, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		return GraphImport{}, ErrInstagramResourceNotFound
	}
	if err != nil {
		return GraphImport{}, err
	}
	reactivated := false
	if params.Reactivate != nil && *params.Reactivate {
		if graphImport.State != ImportMembershipInactive {
			return GraphImport{}, ErrInstagramImportInactive
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_graph_imports
			SET state = 'active', membership_inactive_at = NULL, updated_at = $2
			WHERE id = $1
		`, id, params.Now); err != nil {
			return GraphImport{}, err
		}
		graphImport.State = ImportActive
		reactivated = true
	}
	if reactivated {
		if _, err := tx.Exec(ctx, `
			INSERT INTO instagram_reconciliation_jobs (
				id, owner_did, import_id, reason, status, next_attempt_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, 'instagramImportReactivated', 'queued', $4, $4, $4)
		`, uuid.New(), owner, id, params.Now); err != nil {
			return GraphImport{}, fmt.Errorf("queue reactivated Instagram import reconciliation: %w", err)
		}
	}
	updated, err := scanGraphImport(tx.QueryRow(ctx, `
		SELECT `+graphImportColumns+` FROM instagram_graph_imports WHERE id = $1
	`, id))
	if err != nil {
		return GraphImport{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return GraphImport{}, err
	}
	return updated, nil
}

func (s *ImportStore) DeleteImport(ctx context.Context, owner syntax.DID, id uuid.UUID, now time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram import store is unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, `DELETE FROM instagram_graph_imports WHERE id = $1 AND owner_did = $2`, id, owner)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return tx.Commit(ctx)
	}
	suggestionIDs, err := invalidateUnsupportedSuggestions(ctx, tx, owner, now)
	if err != nil {
		return err
	}
	if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "importDeleted", now); err != nil {
		return err
	}
	if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "import_deleted", now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status='ignored', terminal_at=COALESCE(terminal_at,$2),
		    lease_token=NULL, lease_expires_at=NULL, updated_at=$2
		WHERE import_id=$1 AND status IN ('queued','processing','retryable')
	`, id, now); err != nil {
		return fmt.Errorf("cancel deleted Instagram import reconciliation: %w", err)
	}
	return tx.Commit(ctx)
}

func invalidateUnsupportedSuggestions(ctx context.Context, tx pgx.Tx, owner syntax.DID, now time.Time) ([]uuid.UUID, error) {
	return queryUUIDs(ctx, tx, `
		UPDATE instagram_follow_suggestions suggestion
		SET state = 'invalidated', accepting_since = NULL,
		    terminal_at = COALESCE(terminal_at, $2), updated_at = $2
		WHERE suggestion.importer_did = $1
		  AND suggestion.state IN ('pending', 'accepting')
		  AND NOT EXISTS (
			SELECT 1
			FROM instagram_suggestion_sources source
			JOIN instagram_graph_imports source_import
			  ON source_import.id = source.import_id
			 AND source_import.owner_did = suggestion.importer_did
			 AND source_import.state = 'active'
			WHERE source.suggestion_id = suggestion.id
		  )
		RETURNING suggestion.id
	`, owner, now)
}

const graphImportColumns = `
	id, owner_did, state, source_type, following_count,
	created_at, updated_at`

type graphImportRow interface {
	Scan(dest ...any) error
}

func scanGraphImport(row graphImportRow) (GraphImport, error) {
	var graphImport GraphImport
	var owner string
	if err := row.Scan(
		&graphImport.ID,
		&owner,
		&graphImport.State,
		&graphImport.SourceType,
		&graphImport.FollowingCount,
		&graphImport.CreatedAt,
		&graphImport.UpdatedAt,
	); err != nil {
		return GraphImport{}, err
	}
	if !graphImport.State.Valid() || !graphImport.SourceType.Valid() {
		return GraphImport{}, fmt.Errorf("%w: graph import", ErrInvalidInstagramState)
	}
	graphImport.OwnerDID = syntax.DID(owner)
	return graphImport, nil
}
