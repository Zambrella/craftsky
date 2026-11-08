-- Reports remain private and use the existing case intake. A NULL reporter
-- identifies a visitor; never manufacture a DID for an anonymous reporter.
ALTER TABLE moderation_reports ALTER COLUMN reporter_did DROP NOT NULL;
ALTER TABLE moderation_reports ADD COLUMN contact_email TEXT
    CHECK (contact_email IS NULL OR char_length(contact_email) <= 254);
