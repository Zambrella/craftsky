DROP INDEX IF EXISTS instagram_graph_handles_retention_idx;
DROP INDEX IF EXISTS instagram_graph_imports_aggregate_purge_idx;
DROP INDEX IF EXISTS instagram_graph_imports_retention_idx;

-- Expired legacy imports have already lost their unmatched handles and cannot
-- be truthfully restored. Imports created before the verified-link gate also
-- cannot survive without a current verified link.
DELETE FROM instagram_graph_imports import
WHERE import.state = 'expired'
   OR NOT EXISTS (
        SELECT 1
        FROM instagram_account_links link
        WHERE link.owner_did = import.owner_did
          AND link.verified_at IS NOT NULL
          AND link.state IN ('active', 'membershipInactive', 'disputed')
   );

UPDATE instagram_follow_suggestions suggestion
SET state = 'invalidated',
    accepting_since = NULL,
    terminal_at = COALESCE(terminal_at, now()),
    updated_at = now()
WHERE suggestion.state IN ('pending', 'accepting')
  AND NOT EXISTS (
      SELECT 1
      FROM instagram_suggestion_sources source
      JOIN instagram_graph_imports import
        ON import.id = source.import_id
       AND import.owner_did = suggestion.importer_did
       AND import.state = 'active'
      WHERE source.suggestion_id = suggestion.id
  );

UPDATE pds_follow_operations operation
SET status = 'failed',
    last_error_code = 'legacyImportRemoved',
    updated_at = now()
FROM instagram_follow_suggestions suggestion
WHERE suggestion.id = operation.suggestion_id
  AND suggestion.state = 'invalidated'
  AND operation.status IN ('pending', 'writing', 'failed');

UPDATE instagram_reconciliation_jobs job
SET status = 'ignored',
    terminal_at = COALESCE(terminal_at, now()),
    lease_token = NULL,
    lease_expires_at = NULL,
    updated_at = now()
WHERE job.import_id IS NOT NULL
  AND job.status IN ('queued', 'processing', 'retryable')
  AND NOT EXISTS (
      SELECT 1
      FROM instagram_graph_imports import
      WHERE import.id = job.import_id
  );

ALTER TABLE instagram_graph_imports
    DROP CONSTRAINT IF EXISTS instagram_graph_imports_retention_check,
    DROP CONSTRAINT IF EXISTS instagram_graph_imports_state_check,
    DROP COLUMN IF EXISTS retain_unmatched,
    DROP COLUMN IF EXISTS retention_expires_at,
    DROP COLUMN IF EXISTS final_terminal_at,
    DROP COLUMN IF EXISTS aggregate_purge_at;

ALTER TABLE instagram_graph_imports
    ADD CONSTRAINT instagram_graph_imports_state_check
    CHECK (state IN ('active', 'membershipInactive'));

ALTER TABLE instagram_graph_handles
    DROP COLUMN IF EXISTS retain_until;
