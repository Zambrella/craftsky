package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api/envelope"
)

var (
	ErrSavedPostNotFound       = errors.New("saved post: not found")
	ErrSavedPostFolderNotFound = errors.New("saved post folder: not found")
	ErrInvalidSavedPostCursor  = envelope.ErrInvalidCursor
)

type SavedPostStoreOptions struct {
	Now func() time.Time
}

type SavedPostStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewSavedPostStore(pool *pgxpool.Pool, options ...SavedPostStoreOptions) *SavedPostStore {
	now := time.Now
	if len(options) > 0 && options[0].Now != nil {
		now = options[0].Now
	}
	return &SavedPostStore{pool: pool, now: now}
}

func (s *SavedPostStore) Save(ctx context.Context, owner syntax.DID, postURI syntax.ATURI, assignment FolderAssignment) (SaveMutationResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SaveMutationResult{}, fmt.Errorf("saved post save begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, owner, postURI); err != nil {
		return SaveMutationResult{}, fmt.Errorf("saved post save lock: %w", err)
	}

	desiredFolderID, desiredFolderValue, err := lockOwnedFolderAssignment(ctx, tx, owner, assignment)
	if err != nil {
		return SaveMutationResult{}, err
	}

	existing, err := readSavedPostState(ctx, tx, owner, postURI)
	switch {
	case err == nil:
		if !assignment.Present {
			if err := tx.Commit(ctx); err != nil {
				return SaveMutationResult{}, fmt.Errorf("saved post save commit: %w", err)
			}
			return SaveMutationResult{State: existing}, nil
		}
		changed := !sameOptionalString(existing.FolderID, desiredFolderID)
		if changed {
			if err := tx.QueryRow(ctx, `
				UPDATE saved_posts
				SET folder_id = $3
				WHERE owner_did = $1 AND post_uri = $2
				RETURNING saved_at, folder_id::text
			`, owner, postURI, desiredFolderValue).Scan(&existing.SavedAt, &existing.FolderID); err != nil {
				return SaveMutationResult{}, fmt.Errorf("saved post save update: %w", err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return SaveMutationResult{}, fmt.Errorf("saved post save commit: %w", err)
		}
		return SaveMutationResult{State: existing, Changed: changed}, nil
	case !errors.Is(err, ErrSavedPostNotFound):
		return SaveMutationResult{}, err
	}

	savedAt := savedAtForMutation(nil, s.now().UTC())
	created := SavedPostState{SavedAt: savedAt}
	if err := tx.QueryRow(ctx, `
		INSERT INTO saved_posts (owner_did, post_uri, folder_id, saved_at)
		VALUES ($1, $2, $3, $4)
		RETURNING saved_at, folder_id::text
	`, owner, postURI, desiredFolderValue, savedAt).Scan(&created.SavedAt, &created.FolderID); err != nil {
		return SaveMutationResult{}, fmt.Errorf("saved post save insert: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return SaveMutationResult{}, fmt.Errorf("saved post save commit: %w", err)
	}
	return SaveMutationResult{State: created, Created: true}, nil
}

func (s *SavedPostStore) Unsave(ctx context.Context, owner syntax.DID, postURI syntax.ATURI) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("saved post unsave begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, owner, postURI); err != nil {
		return fmt.Errorf("saved post unsave lock: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM saved_posts
		WHERE owner_did = $1 AND post_uri = $2
	`, owner, postURI); err != nil {
		return fmt.Errorf("saved post unsave: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("saved post unsave commit: %w", err)
	}
	return nil
}

func (s *SavedPostStore) ReadState(ctx context.Context, owner syntax.DID, postURI syntax.ATURI) (SavedPostState, error) {
	return readSavedPostState(ctx, s.pool, owner, postURI)
}

func (s *SavedPostStore) CreateFolder(ctx context.Context, owner syntax.DID, name string) (SavedPostFolder, error) {
	normalized, err := NormalizeSavedPostFolderName(name)
	if err != nil {
		return SavedPostFolder{}, err
	}
	now := s.now().UTC()
	var folder SavedPostFolder
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO saved_post_folders (id, owner_did, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		RETURNING id::text, name, created_at, updated_at
	`, uuid.New(), owner, normalized, now).Scan(
		&folder.ID,
		&folder.Name,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	); err != nil {
		return SavedPostFolder{}, fmt.Errorf("saved post folder create: %w", err)
	}
	return folder, nil
}

func (s *SavedPostStore) RenameFolder(ctx context.Context, owner syntax.DID, folderID, name string) (SavedPostFolder, error) {
	parsed, err := uuid.Parse(folderID)
	if err != nil {
		return SavedPostFolder{}, ErrSavedPostFolderNotFound
	}
	normalized, err := NormalizeSavedPostFolderName(name)
	if err != nil {
		return SavedPostFolder{}, err
	}
	now := s.now().UTC()
	var folder SavedPostFolder
	err = s.pool.QueryRow(ctx, `
		UPDATE saved_post_folders
		SET name = $3, updated_at = $4
		WHERE owner_did = $1 AND id = $2
		RETURNING id::text, name, created_at, updated_at
	`, owner, parsed, normalized, now).Scan(
		&folder.ID,
		&folder.Name,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SavedPostFolder{}, ErrSavedPostFolderNotFound
	}
	if err != nil {
		return SavedPostFolder{}, fmt.Errorf("saved post folder rename: %w", err)
	}
	return folder, nil
}

func (s *SavedPostStore) DeleteFolder(ctx context.Context, owner syntax.DID, folderID string, mode SavedPostFolderDeleteMode) error {
	parsed, err := uuid.Parse(folderID)
	if err != nil {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("saved post folder delete begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if mode == SavedPostFolderRemoveSaves {
		if _, err := tx.Exec(ctx, `
			DELETE FROM saved_posts
			WHERE owner_did = $1 AND folder_id = $2
		`, owner, parsed); err != nil {
			return fmt.Errorf("saved post folder saves delete: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM saved_post_folders
		WHERE owner_did = $1 AND id = $2
	`, owner, parsed); err != nil {
		return fmt.Errorf("saved post folder delete: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("saved post folder delete commit: %w", err)
	}
	return nil
}

func (s *SavedPostStore) ListFolders(ctx context.Context, owner syntax.DID, limit int, cursor string) ([]SavedPostFolder, string, error) {
	decoded, err := DecodeSavedPostFolderCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	var foldedName any
	var folderID any
	if cursor != "" {
		parsed, err := uuid.Parse(decoded.FolderID)
		if err != nil {
			return nil, "", envelope.ErrInvalidCursor
		}
		foldedName = decoded.FoldedName
		folderID = parsed
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, created_at, updated_at
		FROM saved_post_folders
		WHERE owner_did = $1
		  AND ($2::text IS NULL OR (lower(name), id) > ($2::text, $3::uuid))
		ORDER BY lower(name) ASC, id ASC
		LIMIT $4
	`, owner, foldedName, folderID, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("saved post folder list: %w", err)
	}
	defer rows.Close()
	folders := make([]SavedPostFolder, 0, limit+1)
	for rows.Next() {
		var folder SavedPostFolder
		if err := rows.Scan(&folder.ID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, "", fmt.Errorf("saved post folder list scan: %w", err)
		}
		folders = append(folders, folder)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("saved post folder list iter: %w", err)
	}
	if len(folders) <= limit {
		return folders, "", nil
	}
	folders = folders[:limit]
	last := folders[len(folders)-1]
	next, err := EncodeSavedPostFolderCursor(strings.ToLower(last.Name), last.ID)
	if err != nil {
		return nil, "", fmt.Errorf("saved post folder cursor: %w", err)
	}
	return folders, next, nil
}

func (s *SavedPostStore) ListSavedRefs(ctx context.Context, owner syntax.DID, filter SavedPostListFilter) ([]SavedPostRef, string, error) {
	if !validSavedPostScope(filter.Scope) || !validSavedPostSort(filter.Sort) || filter.Limit < 1 || filter.Limit > 100 {
		return nil, "", &FieldError{Code: "validation_failed", Fields: map[string]string{"_": "invalid saved-post list filter"}}
	}
	if (filter.Scope == SavedPostScopeFolder) != (filter.FolderID != "") {
		return nil, "", &FieldError{Code: "validation_failed", Fields: map[string]string{"folderId": "does not match list scope"}}
	}

	var folderValue any
	if filter.Scope == SavedPostScopeFolder {
		parsed, err := uuid.Parse(filter.FolderID)
		if err != nil {
			return nil, "", ErrSavedPostFolderNotFound
		}
		var exists bool
		if err := s.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM saved_post_folders
				WHERE owner_did = $1 AND id = $2
			)
		`, owner, parsed).Scan(&exists); err != nil {
			return nil, "", fmt.Errorf("saved post folder scope: %w", err)
		}
		if !exists {
			return nil, "", ErrSavedPostFolderNotFound
		}
		folderValue = parsed
	}

	decoded, err := DecodeSavedPostCursor(filter.Cursor, filter.Scope, filter.FolderID, filter.Sort)
	if err != nil {
		return nil, "", ErrInvalidSavedPostCursor
	}
	var cursorSavedAt any
	var cursorURI any
	if filter.Cursor != "" {
		cursorSavedAt = decoded.SavedAt
		cursorURI = decoded.URI
	}
	query := savedPostListQuery(filter.Scope, filter.Sort)
	rows, err := s.pool.Query(ctx, query,
		owner,
		string(filter.Scope),
		folderValue,
		cursorSavedAt,
		cursorURI,
		filter.Limit+1,
	)
	if err != nil {
		return nil, "", fmt.Errorf("saved post list: %w", err)
	}
	defer rows.Close()
	refs := make([]SavedPostRef, 0, filter.Limit+1)
	for rows.Next() {
		var ref SavedPostRef
		var postURI string
		if err := rows.Scan(&postURI, &ref.SavedAt, &ref.FolderID); err != nil {
			return nil, "", fmt.Errorf("saved post list scan: %w", err)
		}
		ref.PostURI = syntax.ATURI(postURI)
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("saved post list iter: %w", err)
	}
	if len(refs) <= filter.Limit {
		return refs, "", nil
	}
	refs = refs[:filter.Limit]
	last := refs[len(refs)-1]
	next, err := EncodeSavedPostCursor(filter.Scope, filter.FolderID, filter.Sort, last.SavedAt, last.PostURI.String())
	if err != nil {
		return nil, "", fmt.Errorf("saved post list cursor: %w", err)
	}
	return refs, next, nil
}

func savedPostListQuery(_ SavedPostScope, sort SavedPostSort) string {
	direction := "DESC"
	comparator := "<"
	if sort == SavedPostSortOldest {
		direction = "ASC"
		comparator = ">"
	}
	return fmt.Sprintf(`
		SELECT post_uri, saved_at, folder_id::text
		FROM saved_posts
		WHERE owner_did = $1
		  AND (
			$2::text = 'all'
			OR ($2::text = 'folder' AND folder_id = $3::uuid)
			OR ($2::text = 'unfiled' AND folder_id IS NULL)
		  )
		  AND ($4::timestamptz IS NULL OR (saved_at, post_uri) %s ($4::timestamptz, $5::text))
		ORDER BY saved_at %s, post_uri %s
		LIMIT $6
	`, comparator, direction, direction)
}

type savedPostStateQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readSavedPostState(ctx context.Context, q savedPostStateQuerier, owner syntax.DID, postURI syntax.ATURI) (SavedPostState, error) {
	var state SavedPostState
	err := q.QueryRow(ctx, `
		SELECT saved_at, folder_id::text
		FROM saved_posts
		WHERE owner_did = $1 AND post_uri = $2
	`, owner, postURI).Scan(&state.SavedAt, &state.FolderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return SavedPostState{}, ErrSavedPostNotFound
	}
	if err != nil {
		return SavedPostState{}, fmt.Errorf("saved post read state: %w", err)
	}
	return state, nil
}

func lockOwnedFolderAssignment(ctx context.Context, tx pgx.Tx, owner syntax.DID, assignment FolderAssignment) (*string, any, error) {
	if !assignment.Present || assignment.ID == nil {
		return nil, nil, nil
	}
	parsed, err := uuid.Parse(*assignment.ID)
	if err != nil {
		return nil, nil, ErrSavedPostFolderNotFound
	}
	var canonical string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM saved_post_folders
		WHERE owner_did = $1 AND id = $2
		FOR KEY SHARE
	`, owner, parsed).Scan(&canonical); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrSavedPostFolderNotFound
	} else if err != nil {
		return nil, nil, fmt.Errorf("saved post folder ownership: %w", err)
	}
	return &canonical, parsed, nil
}

func sameOptionalString(first, second *string) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}
	return *first == *second
}
