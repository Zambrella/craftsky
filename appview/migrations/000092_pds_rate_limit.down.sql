-- A rate limit is known not to have applied, just like a repository swap refusal.
UPDATE pds_command_dispatches SET outcome='invalid_swap',retry_not_before=NULL WHERE outcome='rate_limited';
ALTER TABLE pds_command_dispatches DROP CONSTRAINT pds_command_dispatches_rate_limit_check;
ALTER TABLE pds_command_dispatches DROP CONSTRAINT pds_command_dispatches_outcome_check;
ALTER TABLE pds_command_dispatches ADD CONSTRAINT pds_command_dispatches_outcome_check
 CHECK (outcome IN ('dispatching','accepted','ambiguous','rejected','invalid_swap'));
ALTER TABLE pds_command_dispatches DROP COLUMN retry_not_before;
