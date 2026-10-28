ALTER TABLE post_publication_state ADD COLUMN reply_eligible BOOLEAN NOT NULL DEFAULT true, ADD COLUMN quote_eligible BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE post_publication_mentions ADD COLUMN eligible BOOLEAN NOT NULL DEFAULT true;
