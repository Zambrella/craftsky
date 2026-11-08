-- This intentionally fails if anonymous reports remain; rollback must not
-- delete reports or invent reporter identities to restore the old schema.
ALTER TABLE moderation_reports ALTER COLUMN reporter_did SET NOT NULL;
ALTER TABLE moderation_reports DROP COLUMN contact_email;
