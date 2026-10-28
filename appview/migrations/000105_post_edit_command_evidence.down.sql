ALTER TABLE pds_command_dispatches
    DROP CONSTRAINT pds_command_observation_pair,
    DROP COLUMN observation_revision,
    DROP COLUMN observation_cid,
    DROP COLUMN result_commit_revision;
