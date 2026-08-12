DROP TABLE IF EXISTS account_deletion_operations;

ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_account_deletion_metadata_check,
    DROP CONSTRAINT IF EXISTS oauth_auth_requests_purpose_check,
    DROP COLUMN IF EXISTS account_deletion_job_id,
    DROP COLUMN IF EXISTS account_deletion_owner_did,
    DROP COLUMN IF EXISTS device_id,
    DROP COLUMN IF EXISTS purpose;
