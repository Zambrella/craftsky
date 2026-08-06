CREATE TABLE profile_pins (
    owner_did   TEXT        NOT NULL,
    slot        TEXT        NOT NULL,
    post_uri    TEXT        NOT NULL,
    state_token UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,

    CONSTRAINT profile_pins_pkey
        PRIMARY KEY (owner_did, slot),
    CONSTRAINT profile_pins_owner_did_fkey
        FOREIGN KEY (owner_did)
        REFERENCES craftsky_profiles(did)
        ON DELETE CASCADE,
    CONSTRAINT profile_pins_post_uri_fkey
        FOREIGN KEY (post_uri)
        REFERENCES craftsky_posts(uri)
        ON DELETE CASCADE,
    CONSTRAINT profile_pins_slot_check
        CHECK (slot IN ('standard', 'project'))
);

CREATE INDEX profile_pins_post_uri_idx
    ON profile_pins (post_uri);
