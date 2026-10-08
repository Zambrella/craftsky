DROP TRIGGER IF EXISTS safety_retention_events_append_only ON safety_retention_events;
DROP FUNCTION IF EXISTS reject_safety_retention_event_mutation();
DROP TABLE IF EXISTS safety_retention_attempts;
DROP TABLE IF EXISTS safety_retention_jobs;
