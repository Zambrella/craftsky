DROP TABLE operator_oauth_attempts;
DROP TABLE operator_browser_sessions;
ALTER TABLE safety_operators DROP CONSTRAINT safety_operators_did_check;
ALTER TABLE safety_operators DROP COLUMN did;
-- Historical audit events and revoked credentials are intentionally retained.
