DROP TABLE IF EXISTS account_deletion_cleanup_artifacts;
DROP TABLE IF EXISTS account_deletion_cleanup_steps;
DROP TABLE IF EXISTS account_deletion_index_receipts;
DROP TABLE IF EXISTS account_deletion_expected_records;
DROP TABLE IF EXISTS account_deletion_recovery_credentials;
DROP TABLE IF EXISTS account_deletion_status_credentials;
DROP TABLE IF EXISTS account_deletion_operations;
DROP TABLE IF EXISTS account_deletion_audits;

ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_account_deletion_metadata_check,
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_purpose_check,
    DROP COLUMN IF EXISTS account_deletion_job_id,
    DROP COLUMN IF EXISTS account_deletion_owner_did,
    DROP COLUMN IF EXISTS device_id,
    DROP COLUMN IF EXISTS purpose;
