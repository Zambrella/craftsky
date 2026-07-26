CREATE TABLE saved_post_folders (
    id          UUID        NOT NULL,
    owner_did   TEXT        NOT NULL,
    name        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,

    CONSTRAINT saved_post_folders_pkey
        PRIMARY KEY (id),
    CONSTRAINT saved_post_folders_owner_did_fkey
        FOREIGN KEY (owner_did)
        REFERENCES craftsky_profiles(did)
        ON DELETE CASCADE,
    CONSTRAINT saved_post_folders_owner_did_id_key
        UNIQUE (owner_did, id),
    CONSTRAINT saved_post_folders_name_check
        CHECK (char_length(name) BETWEEN 1 AND 100)
);

CREATE INDEX saved_post_folders_owner_name_idx
    ON saved_post_folders (owner_did, lower(name), id);

CREATE TABLE saved_posts (
    owner_did  TEXT        NOT NULL,
    post_uri   TEXT        NOT NULL,
    folder_id  UUID,
    saved_at   TIMESTAMPTZ NOT NULL,

    CONSTRAINT saved_posts_pkey
        PRIMARY KEY (owner_did, post_uri),
    CONSTRAINT saved_posts_owner_did_fkey
        FOREIGN KEY (owner_did)
        REFERENCES craftsky_profiles(did)
        ON DELETE CASCADE,
    CONSTRAINT saved_posts_post_uri_fkey
        FOREIGN KEY (post_uri)
        REFERENCES craftsky_posts(uri)
        ON DELETE CASCADE,
    CONSTRAINT saved_posts_owner_did_folder_id_fkey
        FOREIGN KEY (owner_did, folder_id)
        REFERENCES saved_post_folders(owner_did, id)
        ON DELETE SET NULL (folder_id)
);

CREATE INDEX saved_posts_owner_saved_at_idx
    ON saved_posts (owner_did, saved_at DESC, post_uri DESC);

CREATE INDEX saved_posts_owner_folder_saved_at_idx
    ON saved_posts (owner_did, folder_id, saved_at DESC, post_uri DESC)
    WHERE folder_id IS NOT NULL;

CREATE INDEX saved_posts_owner_unfiled_saved_at_idx
    ON saved_posts (owner_did, saved_at DESC, post_uri DESC)
    WHERE folder_id IS NULL;
