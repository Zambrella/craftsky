-- Bounded original publication eligibility, without a previous post body.
-- Accepted creates can precede Tap; explicit owner inventory owns cleanup.
CREATE TABLE post_publication_state (
    uri TEXT PRIMARY KEY,
    owner_did TEXT NOT NULL,
    origin_status TEXT NOT NULL CHECK (origin_status IN ('established','unknown','deleted')),
    original_creation_time TIMESTAMPTZ,
    parent_uri TEXT,
    parent_cid TEXT,
    root_uri TEXT,
    root_cid TEXT,
    quoted_uri TEXT,
    quoted_cid TEXT,
    reply_eligible BOOLEAN NOT NULL DEFAULT true,
    quote_eligible BOOLEAN NOT NULL DEFAULT true,
    CHECK ((origin_status='established') = (original_creation_time IS NOT NULL))
);
CREATE TABLE post_publication_mentions (
    post_uri TEXT NOT NULL REFERENCES post_publication_state(uri) ON DELETE CASCADE,
    recipient_did TEXT NOT NULL,
    eligible BOOLEAN NOT NULL DEFAULT true,
    PRIMARY KEY(post_uri,recipient_did)
);
CREATE INDEX post_publication_state_owner_idx ON post_publication_state(owner_did,uri);
CREATE INDEX post_publication_mentions_recipient_idx ON post_publication_mentions(recipient_did,post_uri);
