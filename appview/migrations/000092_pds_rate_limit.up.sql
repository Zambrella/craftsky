ALTER TABLE pds_command_dispatches ADD COLUMN retry_not_before TIMESTAMPTZ;
ALTER TABLE pds_command_dispatches DROP CONSTRAINT pds_command_dispatches_outcome_check;
ALTER TABLE pds_command_dispatches ADD CONSTRAINT pds_command_dispatches_outcome_check
 CHECK (outcome IN ('dispatching','accepted','ambiguous','rejected','invalid_swap','rate_limited'));
ALTER TABLE pds_command_dispatches ADD CONSTRAINT pds_command_dispatches_rate_limit_check
 CHECK ((outcome = 'rate_limited') = (retry_not_before IS NOT NULL));
