ALTER TABLE pds_follow_operations
    RENAME CONSTRAINT pds_follow_operations_automatic_follow_id_key
        TO pds_follow_operations_suggestion_id_key;

ALTER TABLE pds_follow_operations
    RENAME COLUMN automatic_follow_id TO suggestion_id;

ALTER INDEX instagram_automatic_follow_sources_import_idx
    RENAME TO instagram_suggestion_sources_import_idx;

ALTER TABLE instagram_automatic_follow_sources
    RENAME CONSTRAINT instagram_automatic_follow_sources_pkey
        TO instagram_suggestion_sources_pkey;
ALTER TABLE instagram_automatic_follow_sources
    RENAME CONSTRAINT instagram_automatic_follow_sources_automatic_follow_id_fkey
        TO instagram_suggestion_sources_suggestion_id_fkey;
ALTER TABLE instagram_automatic_follow_sources
    RENAME CONSTRAINT instagram_automatic_follow_sources_import_id_fkey
        TO instagram_suggestion_sources_import_id_fkey;

ALTER TABLE instagram_automatic_follow_sources
    RENAME COLUMN automatic_follow_id TO suggestion_id;

ALTER TABLE instagram_automatic_follow_sources
    RENAME TO instagram_suggestion_sources;

ALTER INDEX instagram_automatic_follow_ledger_owner_page_idx
    RENAME TO instagram_follow_suggestions_owner_page_idx;
ALTER INDEX instagram_automatic_follow_ledger_target_idx
    RENAME TO instagram_follow_suggestions_target_idx;
ALTER INDEX instagram_automatic_follow_ledger_terminal_retention_idx
    RENAME TO instagram_follow_suggestions_terminal_retention_idx;

ALTER TABLE instagram_automatic_follow_ledger
    RENAME CONSTRAINT instagram_automatic_follow_ledger_pkey
        TO instagram_follow_suggestions_pkey;
ALTER TABLE instagram_automatic_follow_ledger
    RENAME CONSTRAINT instagram_automatic_follow_ledger_state_check
        TO instagram_follow_suggestions_state_check;
ALTER TABLE instagram_automatic_follow_ledger
    RENAME CONSTRAINT instagram_automatic_follow_ledger_reason_check
        TO instagram_follow_suggestions_reason_check;
ALTER TABLE instagram_automatic_follow_ledger
    RENAME CONSTRAINT instagram_automatic_follow_ledger_not_self_check
        TO instagram_follow_suggestions_not_self_check;
ALTER TABLE instagram_automatic_follow_ledger
    RENAME CONSTRAINT instagram_automatic_follow_ledger_pair_reason_key
        TO instagram_follow_suggestions_importer_did_target_did_reason_key;

ALTER TABLE instagram_automatic_follow_ledger
    RENAME TO instagram_follow_suggestions;
