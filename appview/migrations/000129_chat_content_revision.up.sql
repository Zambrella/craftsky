ALTER TABLE chat_sync_state
    ADD COLUMN content_revision BIGINT NOT NULL DEFAULT 0 CHECK (content_revision >= 0);
