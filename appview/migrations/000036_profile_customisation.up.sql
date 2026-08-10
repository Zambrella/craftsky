CREATE TABLE profile_customisations (
    owner_did          TEXT        NOT NULL,
    colour             TEXT        NOT NULL,
    profile_border     TEXT        NOT NULL,
    profile_background TEXT        NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT profile_customisations_pkey
        PRIMARY KEY (owner_did),
    CONSTRAINT profile_customisations_owner_did_fkey
        FOREIGN KEY (owner_did)
        REFERENCES craftsky_profiles(did)
        ON DELETE CASCADE
);
