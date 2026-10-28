-- Exact write revisions and read-only recovery bounds have different meanings.
ALTER TABLE pds_command_dispatches
    ADD COLUMN result_commit_revision TEXT COLLATE "C",
    ADD COLUMN observation_cid TEXT,
    ADD COLUMN observation_revision TEXT COLLATE "C",
    ADD CONSTRAINT pds_command_observation_pair CHECK (
        (observation_cid IS NULL) = (observation_revision IS NULL)
    );
