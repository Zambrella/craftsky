ALTER TABLE instagram_graph_imports
    DROP CONSTRAINT IF EXISTS instagram_graph_imports_state_check,
    ADD COLUMN retain_unmatched BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN retention_expires_at TIMESTAMPTZ,
    ADD COLUMN final_terminal_at TIMESTAMPTZ,
    ADD COLUMN aggregate_purge_at TIMESTAMPTZ;

UPDATE instagram_graph_imports
SET retention_expires_at = created_at + interval '1 year';

ALTER TABLE instagram_graph_imports
    ALTER COLUMN retain_unmatched DROP DEFAULT,
    ADD CONSTRAINT instagram_graph_imports_state_check
        CHECK (state IN ('active', 'membershipInactive', 'expired')),
    ADD CONSTRAINT instagram_graph_imports_retention_check
        CHECK (retain_unmatched OR retention_expires_at IS NULL);

CREATE INDEX instagram_graph_imports_retention_idx
    ON instagram_graph_imports (retention_expires_at, id)
    WHERE retention_expires_at IS NOT NULL;
CREATE INDEX instagram_graph_imports_aggregate_purge_idx
    ON instagram_graph_imports (aggregate_purge_at, id)
    WHERE aggregate_purge_at IS NOT NULL;

ALTER TABLE instagram_graph_handles
    ADD COLUMN retain_until TIMESTAMPTZ;

UPDATE instagram_graph_handles handle
SET retain_until = source.retention_expires_at
FROM instagram_graph_imports source
WHERE source.id = handle.import_id;

CREATE INDEX instagram_graph_handles_retention_idx
    ON instagram_graph_handles (retain_until, id)
    WHERE retain_until IS NOT NULL;
