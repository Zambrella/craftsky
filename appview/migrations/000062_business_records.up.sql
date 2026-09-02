CREATE TABLE craftsky_business_profiles (
    owner_did       TEXT        PRIMARY KEY,
    uri             TEXT        NOT NULL UNIQUE,
    cid             TEXT        NOT NULL,
    raw_record      JSONB       NOT NULL,
    source_revision TEXT        NOT NULL,
    indexed_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE craftsky_business_events (
    uri             TEXT        PRIMARY KEY,
    owner_did       TEXT        NOT NULL,
    rkey            TEXT        NOT NULL,
    cid             TEXT        NOT NULL,
    raw_record      JSONB       NOT NULL,
    source_revision TEXT        NOT NULL,
    starts_at       TIMESTAMPTZ NOT NULL,
    ends_at         TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    status          TEXT,
    indexed_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_did, rkey)
);

CREATE INDEX craftsky_business_events_owner_starts_uri_idx
    ON craftsky_business_events(owner_did, starts_at, uri);

CREATE TABLE craftsky_business_record_tombstones (
    uri             TEXT        PRIMARY KEY,
    owner_did       TEXT        NOT NULL,
    collection      TEXT        NOT NULL CHECK (collection IN (
        'social.craftsky.business.profile',
        'social.craftsky.business.event'
    )),
    source_revision TEXT        NOT NULL,
    deleted_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX craftsky_business_record_tombstones_owner_collection_uri_idx
    ON craftsky_business_record_tombstones(owner_did, collection, uri);

CREATE INDEX craftsky_business_record_tombstones_owner_uri_idx
    ON craftsky_business_record_tombstones(owner_did, uri);
