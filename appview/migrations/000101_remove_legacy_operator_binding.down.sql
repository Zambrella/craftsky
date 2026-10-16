-- Restore schema compatibility only. Retired external bindings are not
-- recoverable, and this does not reactivate credentials or operator access.
ALTER TABLE safety_operators ADD COLUMN retool_user_id BIGINT UNIQUE CHECK (retool_user_id > 0);
