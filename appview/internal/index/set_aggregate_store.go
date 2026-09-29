package index

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/tap"
)

func replaceSetSourceTx(
	ctx context.Context,
	tx pgx.Tx,
	sourceURI syntax.ATURI,
	next *SetSource,
	now time.Time,
) ([]SetAggregateChange, error) {
	previous, err := readSetSourceTx(ctx, tx, sourceURI)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM pds_set_sources WHERE source_uri=$1`, sourceURI); err != nil {
		return nil, fmt.Errorf("delete prior set source: %w", err)
	}
	if next != nil {
		if next.URI != sourceURI {
			return nil, errors.New("set source URI does not match replacement URI")
		}
		if err := insertSetSourceTx(ctx, tx, *next, now); err != nil {
			return nil, err
		}
	}
	scopes := affectedSetScopes(previous, next)
	changes := make([]SetAggregateChange, 0, len(scopes))
	for _, scope := range scopes {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, setScopeLockKey(scope)); err != nil {
			return nil, fmt.Errorf("lock set scope: %w", err)
		}
		change, err := recomputeSetAggregateTx(ctx, tx, scope, now)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func readSetSourceTx(ctx context.Context, tx pgx.Tx, sourceURI syntax.ATURI) (*SetSource, error) {
	var source SetSource
	var subjectDID, subjectURI, subjectCID, reason, dependencyKind, dependencyKey *string
	err := tx.QueryRow(ctx, `
		SELECT source_uri,kind,actor_did,scope_key,subject_did,subject_uri,subject_cid,
		       activity_at,eligible,ineligibility_reason,dependency_kind,dependency_key
		FROM pds_set_sources WHERE source_uri=$1
	`, sourceURI).Scan(
		&source.URI, &source.Scope.Kind, &source.Scope.Actor, &source.Scope.Key,
		&subjectDID, &subjectURI, &subjectCID, &source.ActivityAt, &source.Eligible,
		&reason, &dependencyKind, &dependencyKey,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read prior set source: %w", err)
	}
	if subjectDID != nil {
		source.SubjectDID = syntax.DID(*subjectDID)
	}
	if subjectURI != nil {
		source.SubjectURI = syntax.ATURI(*subjectURI)
	}
	if subjectCID != nil {
		source.SubjectCID = syntax.CID(*subjectCID)
	}
	if reason != nil {
		source.IneligibilityReason = *reason
	}
	if dependencyKind != nil && dependencyKey != nil {
		source.Dependency = tap.Dependency{Kind: *dependencyKind, Key: *dependencyKey}
	}
	return &source, nil
}

func insertSetSourceTx(ctx context.Context, tx pgx.Tx, source SetSource, now time.Time) error {
	var subjectDID, subjectURI, subjectCID, reason, dependencyKind, dependencyKey any
	if source.SubjectDID != "" {
		subjectDID = source.SubjectDID
	}
	if source.SubjectURI != "" {
		subjectURI = source.SubjectURI
	}
	if source.SubjectCID != "" {
		subjectCID = source.SubjectCID
	}
	if source.IneligibilityReason != "" {
		reason = source.IneligibilityReason
	}
	if source.Dependency != (tap.Dependency{}) {
		dependencyKind, dependencyKey = source.Dependency.Kind, source.Dependency.Key
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO pds_set_sources(
			source_uri,kind,actor_did,scope_key,subject_did,subject_uri,subject_cid,
			activity_at,eligible,ineligibility_reason,dependency_kind,dependency_key,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, source.URI, source.Scope.Kind, source.Scope.Actor, source.Scope.Key,
		subjectDID, subjectURI, subjectCID, source.ActivityAt, source.Eligible,
		reason, dependencyKind, dependencyKey, now); err != nil {
		return fmt.Errorf("insert set source: %w", err)
	}
	return nil
}

func recomputeSetAggregateTx(ctx context.Context, tx pgx.Tx, scope SetScope, now time.Time) (SetAggregateChange, error) {
	previous, err := readSetAggregateTx(ctx, tx, scope)
	if err != nil {
		return SetAggregateChange{}, err
	}
	rows, err := tx.Query(ctx, `
		SELECT source_uri,activity_at,subject_did,subject_uri
		FROM pds_set_sources
		WHERE kind=$1 AND actor_did=$2 AND scope_key=$3 AND eligible
		ORDER BY activity_at,source_uri
	`, scope.Kind, scope.Actor, scope.Key)
	if err != nil {
		return SetAggregateChange{}, fmt.Errorf("read eligible set sources: %w", err)
	}
	defer rows.Close()
	var sources []SetSource
	for rows.Next() {
		var source SetSource
		var subjectDID, subjectURI *string
		if err := rows.Scan(&source.URI, &source.ActivityAt, &subjectDID, &subjectURI); err != nil {
			return SetAggregateChange{}, fmt.Errorf("scan eligible set source: %w", err)
		}
		source.Eligible = true
		if subjectDID != nil {
			source.SubjectDID = syntax.DID(*subjectDID)
		}
		if subjectURI != nil {
			source.SubjectURI = syntax.ATURI(*subjectURI)
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return SetAggregateChange{}, fmt.Errorf("iterate eligible set sources: %w", err)
	}
	current, transition := reduceSetAggregate(previous, sources, now)
	if current == nil {
		if _, err := tx.Exec(ctx, `DELETE FROM pds_set_aggregates WHERE kind=$1 AND actor_did=$2 AND scope_key=$3`, scope.Kind, scope.Actor, scope.Key); err != nil {
			return SetAggregateChange{}, fmt.Errorf("delete set aggregate: %w", err)
		}
	} else {
		representative := sources[0]
		var subjectDID, subjectURI any
		if representative.SubjectDID != "" {
			subjectDID = representative.SubjectDID
		}
		if representative.SubjectURI != "" {
			subjectURI = representative.SubjectURI
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO pds_set_aggregates(
				kind,actor_did,scope_key,subject_did,subject_uri,eligible_source_count,
				representative_source_uri,activated_at,updated_at
			) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT(kind,actor_did,scope_key) DO UPDATE SET
				subject_did=EXCLUDED.subject_did,subject_uri=EXCLUDED.subject_uri,
				eligible_source_count=EXCLUDED.eligible_source_count,
				representative_source_uri=EXCLUDED.representative_source_uri,
				activated_at=EXCLUDED.activated_at,updated_at=EXCLUDED.updated_at
		`, scope.Kind, scope.Actor, scope.Key, subjectDID, subjectURI,
			current.EligibleSourceCount, current.RepresentativeURI, current.ActivatedAt, now); err != nil {
			return SetAggregateChange{}, fmt.Errorf("upsert set aggregate: %w", err)
		}
	}
	return SetAggregateChange{Scope: scope, Previous: previous, Current: current, Transition: transition}, nil
}

func readSetAggregateTx(ctx context.Context, tx pgx.Tx, scope SetScope) (*SetAggregate, error) {
	var aggregate SetAggregate
	err := tx.QueryRow(ctx, `
		SELECT eligible_source_count,representative_source_uri,activated_at
		FROM pds_set_aggregates
		WHERE kind=$1 AND actor_did=$2 AND scope_key=$3
	`, scope.Kind, scope.Actor, scope.Key).Scan(
		&aggregate.EligibleSourceCount, &aggregate.RepresentativeURI, &aggregate.ActivatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read set aggregate: %w", err)
	}
	return &aggregate, nil
}

func setScopeLockKey(scope SetScope) string {
	return fmt.Sprintf("%d:%s%d:%s%d:%s", len(scope.Kind), scope.Kind, len(scope.Actor), scope.Actor, len(scope.Key), scope.Key)
}
