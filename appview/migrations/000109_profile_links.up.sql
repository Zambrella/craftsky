ALTER TABLE bluesky_profiles ADD COLUMN links JSONB NOT NULL DEFAULT '[]';

CREATE TABLE bluesky_profile_links (
 uri TEXT PRIMARY KEY,
 did TEXT NOT NULL,
 cid TEXT NOT NULL,
 record JSONB NOT NULL,
 indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX bluesky_profile_links_owner ON bluesky_profile_links(did,uri);
