package accountdeletion

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PrivateCleanupCheckpoints interface {
	CleanupStepComplete(context.Context, uuid.UUID, syntax.DID, string) (bool, error)
	CompleteCleanupStep(context.Context, uuid.UUID, syntax.DID, string) error
}

type DatabasePrivateCleanup struct {
	pool *pgxpool.Pool
}

func NewDatabasePrivateCleanup(pool *pgxpool.Pool) *DatabasePrivateCleanup {
	return &DatabasePrivateCleanup{pool: pool}
}

func (*DatabasePrivateCleanup) Name() string { return "databasePrivate" }

func (cleanup *DatabasePrivateCleanup) Purge(ctx context.Context, jobID uuid.UUID, owner syntax.DID) error {
	if cleanup == nil || cleanup.pool == nil || jobID == uuid.Nil || owner == "" {
		return errors.New("private database cleanup scope is invalid")
	}
	return pgx.BeginFunc(ctx, cleanup.pool, func(tx pgx.Tx) error {
		var boundOAuthSessionID string
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(deletion_oauth_session_id,'')
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND state IN ('active','retrying','needsAttention')
			FOR UPDATE
		`, jobID, owner).Scan(&boundOAuthSessionID); err != nil {
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
			{"ordinary CraftSky sessions", `DELETE FROM craftsky_sessions WHERE account_did=$1`},
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
		if _, err := tx.Exec(ctx, `
			DELETE FROM oauth_sessions
			WHERE account_did=$1 AND session_id<>$2
		`, owner, boundOAuthSessionID); err != nil {
			return fmt.Errorf("delete owner unbound OAuth sessions: %w", err)
		}
		return nil
	})
}

type PrivateCleanupComponent interface {
	Name() string
	Purge(context.Context, uuid.UUID, syntax.DID) error
}

type NamedPrivateCleanup struct {
	name  string
	purge func(context.Context, uuid.UUID, syntax.DID) error
}

func NewNamedPrivateCleanup(name string, purge func(context.Context, uuid.UUID, syntax.DID) error) (*NamedPrivateCleanup, error) {
	if name == "" || purge == nil {
		return nil, errors.New("private cleanup component is invalid")
	}
	return &NamedPrivateCleanup{name: name, purge: purge}, nil
}

func (cleanup *NamedPrivateCleanup) Name() string { return cleanup.name }

func (cleanup *NamedPrivateCleanup) Purge(ctx context.Context, jobID uuid.UUID, owner syntax.DID) error {
	return cleanup.purge(ctx, jobID, owner)
}

type PrivateCleaner struct {
	checkpoints PrivateCleanupCheckpoints
	components  []PrivateCleanupComponent
}

func NewPrivateCleaner(checkpoints PrivateCleanupCheckpoints, components []PrivateCleanupComponent) (*PrivateCleaner, error) {
	if checkpoints == nil || len(components) == 0 {
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
		checkpoints: checkpoints,
		components:  append([]PrivateCleanupComponent(nil), components...),
	}, nil
}

func (cleaner *PrivateCleaner) Run(ctx context.Context, jobID uuid.UUID, owner syntax.DID) error {
	if cleaner == nil || cleaner.checkpoints == nil || jobID == uuid.Nil || owner == "" {
		return errors.New("private cleanup scope is invalid")
	}
	for _, component := range cleaner.components {
		complete, err := cleaner.checkpoints.CleanupStepComplete(ctx, jobID, owner, component.Name())
		if err != nil {
			return fmt.Errorf("read private cleanup checkpoint %s: %w", component.Name(), err)
		}
		if complete {
			continue
		}
		if err := component.Purge(ctx, jobID, owner); err != nil {
			return fmt.Errorf("purge private cleanup component %s: %w", component.Name(), err)
		}
		if err := cleaner.checkpoints.CompleteCleanupStep(ctx, jobID, owner, component.Name()); err != nil {
			return fmt.Errorf("checkpoint private cleanup component %s: %w", component.Name(), err)
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
