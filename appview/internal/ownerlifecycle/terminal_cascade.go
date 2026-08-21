package ownerlifecycle

import (
	"context"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// terminalCascadePolicies documents every DID-bearing parent whose migrated
// foreign keys can cascade deletes or SET NULL updates. "drain" parents have
// unbounded children that this file mutates first in the same row budget.
// "dependency" parents wait for fixed child-ledger components. "fixed" has a
// schema-enforced one-to-one child, so deleting N parents affects at most N
// additional rows. The catalogue integration test rejects an unclassified new
// cascading parent.
var terminalCascadePolicies = map[string]string{
	"account_deletion_operations":       "dependency",
	"craftsky_posts":                    "drain",
	"craftsky_profiles":                 "dependency",
	"craftsky_sessions":                 "fixed",
	"instagram_account_links":           "drain",
	"instagram_automatic_follow_ledger": "drain",
	"instagram_graph_imports":           "drain",
	"instagram_private_suggestions":     "drain",
	"instagram_reconciliation_jobs":     "drain",
	"instagram_verification_attempts":   "drain",
	"notification_events":               "drain",
	"oauth_handoff_exchanges":           "fixed",
	"oauth_sessions":                    "dependency",
	"push_account_subscriptions":        "drain",
	"saved_post_folders":                "dependency",
	"scheduled_posts":                   "dependency",
	"tap_source_records":                "drain",
}

// lockTerminalCascadeParentsTx locks the exact parent rows a purge batch may
// delete. Foreign-key inserts take a conflicting key-share lock, so no new
// cascading child can commit between the final drain and the parent delete.
// CTIDs are safe identifiers for the remainder of this transaction because
// the rows remain locked and are not updated before deletion.
func lockTerminalCascadeParentsTx(
	ctx context.Context,
	tx pgx.Tx,
	entry TerminalDIDEntry,
	owner syntax.DID,
	limit int,
) ([]pgtype.TID, error) {
	if _, classified := terminalCascadePolicies[entry.Table]; !classified {
		return nil, nil
	}
	order := make([]string, 0, len(entry.KeyColumns))
	for _, column := range entry.KeyColumns {
		order = append(order, quoteIdentifier(column))
	}
	query := "SELECT ctid FROM " + quoteIdentifier(entry.Table) +
		" WHERE " + quoteIdentifier(entry.Column) + "=$1 ORDER BY " +
		strings.Join(order, ",") + " LIMIT $2 FOR UPDATE SKIP LOCKED"
	rows, err := tx.Query(ctx, query, owner, limit)
	if err != nil {
		return nil, fmt.Errorf("lock terminal cascade parents %s/%s: %w", entry.Component, entry.Role, err)
	}
	defer rows.Close()
	targets := make([]pgtype.TID, 0, limit)
	for rows.Next() {
		var target pgtype.TID
		if err := rows.Scan(&target); err != nil {
			return nil, fmt.Errorf("scan terminal cascade parent %s/%s: %w", entry.Component, entry.Role, err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate terminal cascade parents %s/%s: %w", entry.Component, entry.Role, err)
	}
	return targets, nil
}

func deleteLockedTerminalRoleBatchTx(
	ctx context.Context,
	tx pgx.Tx,
	entry TerminalDIDEntry,
	targets []pgtype.TID,
) (int64, error) {
	if len(targets) == 0 {
		return 0, nil
	}
	query := "DELETE FROM " + quoteIdentifier(entry.Table) +
		" WHERE ctid=ANY($1::tid[])"
	result, err := tx.Exec(ctx, query, targets)
	if err != nil {
		return 0, fmt.Errorf("delete locked terminal role %s/%s: %w", entry.Component, entry.Role, err)
	}
	return result.RowsAffected(), nil
}

// drainTerminalCascadeBatchTx removes at most limit dependent rows before a
// parent delete. It returns after the first non-empty dependent class, so one
// ProcessClaim never turns a parent batch into a sum of several row budgets.
func drainTerminalCascadeBatchTx(
	ctx context.Context,
	tx pgx.Tx,
	entry TerminalDIDEntry,
	owner syntax.DID,
	parents []pgtype.TID,
	limit int,
) (int64, error) {
	if len(parents) == 0 {
		return 0, nil
	}
	statements := terminalCascadeDrainSQL(entry)
	for index, statement := range statements {
		result, err := tx.Exec(ctx, statement, owner, limit, parents)
		if err != nil {
			return 0, fmt.Errorf(
				"drain terminal cascade %s/%s dependency %d: %w",
				entry.Component, entry.Role, index, err,
			)
		}
		if result.RowsAffected() > 0 {
			return result.RowsAffected(), nil
		}
	}
	return 0, nil
}

func terminalCascadeDrainSQL(entry TerminalDIDEntry) []string {
	role := quoteIdentifier(entry.Column)
	switch entry.Table {
	case "craftsky_posts":
		return []string{
			deletePostDependentSQL("craftsky_likes", "subject_uri", "uri"),
			deletePostDependentSQL("craftsky_reposts", "subject_uri", "uri"),
			deletePostDependentSQL("craftsky_post_mentions", "post_uri", "post_uri,mentioned_did"),
			deletePostDependentSQL("craftsky_project_posts", "uri", "uri"),
			deletePostDependentSQL("saved_posts", "post_uri", "owner_did,post_uri"),
			deletePostDependentSQL("profile_pins", "post_uri", "owner_did,slot"),
		}
	case "notification_events":
		return []string{`
			WITH target AS (
				SELECT child.id
				FROM push_deliveries AS child
				JOIN notification_events AS parent ON parent.id=child.notification_id
				WHERE parent.` + role + `=$1
				  AND parent.ctid=ANY($3::tid[])
				ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
			)
			DELETE FROM push_deliveries AS child
			USING target WHERE child.id=target.id
		`}
	case "push_account_subscriptions":
		return []string{`
			WITH target AS (
				SELECT child.id
				FROM push_deliveries AS child
				JOIN push_account_subscriptions AS parent
				  ON parent.id=child.account_subscription_id
				WHERE parent.account_did=$1
				  AND parent.ctid=ANY($3::tid[])
				ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
			)
			DELETE FROM push_deliveries AS child
			USING target WHERE child.id=target.id
		`}
	case "instagram_account_links":
		return []string{
			deletePrivateSuggestionSourceForLinkSQL,
			deletePrivateSuggestionForLinkSQL,
			deleteIdentityClaimForLinkSQL,
			nullLinkConflictReferenceSQL("existing_link_id"),
			nullLinkConflictReferenceSQL("claimant_link_id"),
		}
	case "instagram_graph_imports":
		return []string{
			deleteSourceForImportSQL("instagram_private_suggestion_sources", "suggestion_id"),
			deleteSourceForImportSQL("instagram_automatic_follow_sources", "automatic_follow_id"),
			deleteGraphHandleForImportSQL,
		}
	case "instagram_automatic_follow_ledger":
		return []string{`
			WITH target AS (
				SELECT child.automatic_follow_id,child.import_id
				FROM instagram_automatic_follow_sources AS child
				JOIN instagram_automatic_follow_ledger AS parent
				  ON parent.id=child.automatic_follow_id
				WHERE parent.` + role + `=$1
				  AND parent.ctid=ANY($3::tid[])
				ORDER BY child.automatic_follow_id,child.import_id
				LIMIT $2 FOR UPDATE OF child NOWAIT
			)
			DELETE FROM instagram_automatic_follow_sources AS child USING target
			WHERE child.automatic_follow_id=target.automatic_follow_id
			  AND child.import_id=target.import_id
		`}
	case "instagram_private_suggestions":
		return []string{`
			WITH target AS (
				SELECT child.suggestion_id,child.import_id
				FROM instagram_private_suggestion_sources AS child
				JOIN instagram_private_suggestions AS parent
				  ON parent.id=child.suggestion_id
				WHERE parent.` + role + `=$1
				  AND parent.ctid=ANY($3::tid[])
				ORDER BY child.suggestion_id,child.import_id
				LIMIT $2 FOR UPDATE OF child NOWAIT
			)
			DELETE FROM instagram_private_suggestion_sources AS child USING target
			WHERE child.suggestion_id=target.suggestion_id
			  AND child.import_id=target.import_id
		`}
	case "instagram_verification_attempts":
		return []string{
			`
				WITH target AS (
					SELECT child.id
					FROM instagram_webhook_work AS child
					JOIN instagram_verification_attempts AS parent
					  ON parent.id=child.verification_attempt_id
					WHERE parent.owner_did=$1
					  AND parent.ctid=ANY($3::tid[])
					ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
				)
				DELETE FROM instagram_webhook_work AS child USING target
				WHERE child.id=target.id
			`,
			nullVerificationConflictReferenceSQL,
		}
	case "instagram_reconciliation_jobs":
		return []string{`
			WITH target AS (
				SELECT child.moderation_output_id
				FROM moderation_restoration_outbox AS child
				JOIN instagram_reconciliation_jobs AS parent
				  ON parent.id=child.reconciliation_job_id
				WHERE parent.` + role + `=$1
				  AND parent.ctid=ANY($3::tid[])
				ORDER BY child.moderation_output_id
				LIMIT $2 FOR UPDATE OF child NOWAIT
			)
			UPDATE moderation_restoration_outbox AS child
			SET reconciliation_job_id=NULL
			FROM target WHERE child.moderation_output_id=target.moderation_output_id
		`}
	case "tap_source_records":
		return []string{`
			WITH target AS (
				SELECT child.id
				FROM tap_projection_jobs AS child
				JOIN tap_source_records AS parent ON parent.uri=child.source_uri
				WHERE parent.did=$1
				  AND parent.ctid=ANY($3::tid[])
				ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
			)
			DELETE FROM tap_projection_jobs AS child USING target
			WHERE child.id=target.id
		`}
	default:
		return nil
	}
}

func deletePostDependentSQL(table, postColumn, orderBy string) string {
	return `
		WITH target AS (
			SELECT child.ctid
			FROM ` + quoteIdentifier(table) + ` AS child
			JOIN craftsky_posts AS parent
			  ON parent.uri=child.` + quoteIdentifier(postColumn) + `
			WHERE parent.did=$1
			  AND parent.ctid=ANY($3::tid[])
			ORDER BY ` + qualifyOrder("child", orderBy) + `
			LIMIT $2 FOR UPDATE OF child NOWAIT
		)
		DELETE FROM ` + quoteIdentifier(table) + ` AS child
		USING target WHERE child.ctid=target.ctid
	`
}

func qualifyOrder(alias, columns string) string {
	// Callers pass only package-owned constant column lists. Split explicitly
	// so every identifier is still quoted independently.
	result := ""
	start := 0
	for index := 0; index <= len(columns); index++ {
		if index != len(columns) && columns[index] != ',' {
			continue
		}
		if result != "" {
			result += ","
		}
		result += alias + "." + quoteIdentifier(columns[start:index])
		start = index + 1
	}
	return result
}

const deletePrivateSuggestionSourceForLinkSQL = `
	WITH target AS (
		SELECT child.suggestion_id,child.import_id
		FROM instagram_private_suggestion_sources AS child
		JOIN instagram_private_suggestions AS suggestion ON suggestion.id=child.suggestion_id
		JOIN instagram_account_links AS parent ON parent.id=suggestion.evidence_link_id
		WHERE parent.owner_did=$1
		  AND parent.ctid=ANY($3::tid[])
		ORDER BY child.suggestion_id,child.import_id
		LIMIT $2 FOR UPDATE OF child NOWAIT
	)
	DELETE FROM instagram_private_suggestion_sources AS child USING target
	WHERE child.suggestion_id=target.suggestion_id AND child.import_id=target.import_id
`

const deletePrivateSuggestionForLinkSQL = `
	WITH target AS (
		SELECT child.id
		FROM instagram_private_suggestions AS child
		JOIN instagram_account_links AS parent ON parent.id=child.evidence_link_id
		WHERE parent.owner_did=$1
		  AND parent.ctid=ANY($3::tid[])
		ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
	)
	DELETE FROM instagram_private_suggestions AS child USING target WHERE child.id=target.id
`

const deleteIdentityClaimForLinkSQL = `
	WITH target AS (
		SELECT child.id
		FROM instagram_identity_claims AS child
		JOIN instagram_account_links AS parent ON parent.id=child.link_id
		WHERE parent.owner_did=$1
		  AND parent.ctid=ANY($3::tid[])
		ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
	)
	DELETE FROM instagram_identity_claims AS child USING target WHERE child.id=target.id
`

func nullLinkConflictReferenceSQL(column string) string {
	quoted := quoteIdentifier(column)
	return `
		WITH target AS (
			SELECT child.id
			FROM instagram_link_conflicts AS child
			JOIN instagram_account_links AS parent ON parent.id=child.` + quoted + `
			WHERE parent.owner_did=$1
			  AND parent.ctid=ANY($3::tid[])
			ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
		)
		UPDATE instagram_link_conflicts AS child SET ` + quoted + `=NULL
		FROM target WHERE child.id=target.id
	`
}

func deleteSourceForImportSQL(table, sourceIDColumn string) string {
	return `
		WITH target AS (
			SELECT child.ctid
			FROM ` + quoteIdentifier(table) + ` AS child
			JOIN instagram_graph_imports AS parent ON parent.id=child.import_id
	WHERE parent.owner_did=$1
	  AND parent.ctid=ANY($3::tid[])
			ORDER BY child.import_id,child.` + quoteIdentifier(sourceIDColumn) + `
			LIMIT $2 FOR UPDATE OF child NOWAIT
		)
		DELETE FROM ` + quoteIdentifier(table) + ` AS child
		USING target WHERE child.ctid=target.ctid
	`
}

const deleteGraphHandleForImportSQL = `
	WITH target AS (
		SELECT child.id
		FROM instagram_graph_handles AS child
		JOIN instagram_graph_imports AS parent ON parent.id=child.import_id
			WHERE parent.owner_did=$1
			  AND parent.ctid=ANY($3::tid[])
		ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
	)
	DELETE FROM instagram_graph_handles AS child USING target WHERE child.id=target.id
`

const nullVerificationConflictReferenceSQL = `
	WITH target AS (
		SELECT child.id
		FROM instagram_link_conflicts AS child
		JOIN instagram_verification_attempts AS parent ON parent.id=child.claimant_attempt_id
		WHERE parent.owner_did=$1
		  AND parent.ctid=ANY($3::tid[])
		ORDER BY child.id LIMIT $2 FOR UPDATE OF child NOWAIT
	)
	UPDATE instagram_link_conflicts AS child SET claimant_attempt_id=NULL
	FROM target WHERE child.id=target.id
`
