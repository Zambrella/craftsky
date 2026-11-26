-- NULL identifies an original publication mention. Edit mentions are pending
-- notification work and are removed after evaluation.
ALTER TABLE post_publication_mentions ADD COLUMN edit_activity_at TIMESTAMPTZ;
