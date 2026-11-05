CREATE TABLE bunny_video_jobs (
 id UUID PRIMARY KEY,
 owner_did TEXT NOT NULL REFERENCES owner_lifecycles(owner_did) ON DELETE RESTRICT,
 owner_generation BIGINT NOT NULL CHECK(owner_generation > 0),
 library_id TEXT NOT NULL,
 video_id UUID UNIQUE,
 imported BOOLEAN NOT NULL DEFAULT false,
 state TEXT NOT NULL DEFAULT 'created' CHECK(state IN ('created','ready','failed','deleted')),
 blob_cid TEXT,
 blob_size BIGINT CHECK(blob_size > 0 AND blob_size <= 300000000),
 lease_until TIMESTAMPTZ NOT NULL DEFAULT '-infinity',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK ((blob_cid IS NULL) = (blob_size IS NULL))
);
CREATE INDEX bunny_video_jobs_owner_created ON bunny_video_jobs(owner_did,created_at);
CREATE INDEX bunny_video_jobs_blob ON bunny_video_jobs(owner_did,blob_cid) WHERE state='ready';
CREATE INDEX bunny_video_jobs_owner_id ON bunny_video_jobs(owner_did,id);
