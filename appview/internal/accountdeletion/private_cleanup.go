package accountdeletion

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DatabasePrivateCleanup struct {
	pool *pgxpool.Pool
}

func NewDatabasePrivateCleanup(pool *pgxpool.Pool) *DatabasePrivateCleanup {
	return &DatabasePrivateCleanup{pool: pool}
}

func (*DatabasePrivateCleanup) Name() string { return "databasePrivate" }

func (cleanup *DatabasePrivateCleanup) Purge(ctx context.Context, owner syntax.DID) error {
	if cleanup == nil || cleanup.pool == nil || owner == "" {
		return errors.New("private database cleanup scope is invalid")
	}
	return pgx.BeginFunc(ctx, cleanup.pool, func(tx pgx.Tx) error {
		var boundOAuthSessionID string
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(deletion_oauth_session_id,'')
			FROM account_deletion_operations
			WHERE owner_did=$1 AND state IN ('active','retrying')
			FOR UPDATE
		`, owner).Scan(&boundOAuthSessionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOperationNotFound
			}
			return fmt.Errorf("lock private database cleanup: %w", err)
		}

		statements := []struct {
			name string
			sql  string
		}{
			{"saved posts", `DELETE FROM saved_posts WHERE owner_did=$1`},
			{"saved post folders", `DELETE FROM saved_post_folders WHERE owner_did=$1`},
			{"profile pins", `DELETE FROM profile_pins WHERE owner_did=$1`},
			{"profile customisation", `DELETE FROM profile_customisations WHERE owner_did=$1`},
			{"recent searches", `DELETE FROM craftsky_recent_searches WHERE viewer_did=$1`},
			{"mutes", `DELETE FROM actor_mutes WHERE owner_did=$1`},
			{"language preferences", `DELETE FROM account_language_preferences WHERE account_did=$1`},
			{"notification events", `DELETE FROM notification_events WHERE recipient_did=$1`},
			{"notification preferences", `DELETE FROM notification_preferences WHERE account_did=$1`},
			{"notification seen state", `DELETE FROM notification_seen_state WHERE account_did=$1`},
			{"moderation reports", `DELETE FROM moderation_reports WHERE reporter_did=$1 OR subject_did=$1`},
			{"moderation outputs", `DELETE FROM moderation_outputs WHERE source_did=$1 OR subject_did=$1`},
			{"deletion OAuth requests", `DELETE FROM oauth_auth_requests WHERE account_deletion_owner_did=$1`},
		}
		for _, statement := range statements {
			if _, err := tx.Exec(ctx, statement.sql, owner); err != nil {
				return fmt.Errorf("delete owner %s: %w", statement.name, err)
			}
		}

		if _, err := tx.Exec(ctx, `
			WITH removed AS (
				DELETE FROM push_account_subscriptions
				WHERE account_did=$1
				RETURNING installation_id
			)
			DELETE FROM push_installations installation
			WHERE installation.id IN (SELECT installation_id FROM removed)
			  AND NOT EXISTS (
				SELECT 1 FROM push_account_subscriptions remaining
				WHERE remaining.installation_id=installation.id
				  AND remaining.account_did<>$1
			  )
		`, owner); err != nil {
			return fmt.Errorf("delete owner push subscriptions: %w", err)
		}
		// Authentication rows are deliberately not purged here. The owner
		// lifecycle transition has already revoked ordinary children and queued
		// every non-bound parent for upstream cleanup; finalization does the same
		// for the accepted deletion-only parent.
		_ = boundOAuthSessionID
		return nil
	})
}

type PrivateCleanupComponent interface {
	Name() string
	Purge(context.Context, syntax.DID) error
}

type NamedPrivateCleanup struct {
	name  string
	purge func(context.Context, syntax.DID) error
}

func NewNamedPrivateCleanup(name string, purge func(context.Context, syntax.DID) error) (*NamedPrivateCleanup, error) {
	if name == "" || purge == nil {
		return nil, errors.New("private cleanup component is invalid")
	}
	return &NamedPrivateCleanup{name: name, purge: purge}, nil
}

func (cleanup *NamedPrivateCleanup) Name() string { return cleanup.name }

func (cleanup *NamedPrivateCleanup) Purge(ctx context.Context, owner syntax.DID) error {
	return cleanup.purge(ctx, owner)
}

type PrivateCleaner struct {
	components []PrivateCleanupComponent
}

func NewPrivateCleaner(components []PrivateCleanupComponent) (*PrivateCleaner, error) {
	if len(components) == 0 {
		return nil, errors.New("private cleanup is unavailable")
	}
	seen := make(map[string]struct{}, len(components))
	for _, component := range components {
		if component == nil || component.Name() == "" {
			return nil, errors.New("private cleanup component is invalid")
		}
		if _, exists := seen[component.Name()]; exists {
			return nil, fmt.Errorf("duplicate private cleanup component %q", component.Name())
		}
		seen[component.Name()] = struct{}{}
	}
	return &PrivateCleaner{
		components: append([]PrivateCleanupComponent(nil), components...),
	}, nil
}

func (cleaner *PrivateCleaner) Run(ctx context.Context, owner syntax.DID) error {
	if cleaner == nil || owner == "" {
		return errors.New("private cleanup scope is invalid")
	}
	for _, component := range cleaner.components {
		if err := component.Purge(ctx, owner); err != nil {
			return fmt.Errorf("purge private cleanup component %s: %w", component.Name(), err)
		}
	}
	return nil
}

func (cleaner *PrivateCleaner) ComponentNames() []string {
	if cleaner == nil {
		return nil
	}
	names := make([]string, 0, len(cleaner.components))
	for _, component := range cleaner.components {
		names = append(names, component.Name())
	}
	return names
}
