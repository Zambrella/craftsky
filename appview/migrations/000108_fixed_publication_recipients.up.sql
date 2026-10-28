-- Editing does not withdraw an original publication recipient (ADR 022).
ALTER TABLE post_publication_state DROP COLUMN reply_eligible, DROP COLUMN quote_eligible;
ALTER TABLE post_publication_mentions DROP COLUMN eligible;
