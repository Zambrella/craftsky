DROP TRIGGER bunny_video_reference_cleanup ON craftsky_posts;
DROP FUNCTION bunny_video_reference_cleanup();
DROP TRIGGER bunny_video_reference_lock ON craftsky_posts;
DROP FUNCTION bunny_video_reference_lock();
DROP INDEX craftsky_posts_video_reference;
DROP INDEX bunny_video_jobs_cleanup_due;
-- Keep unfinished cleanup recoverable by refusing rollback while it is active.
ALTER TABLE bunny_video_jobs DROP CONSTRAINT bunny_video_jobs_state_check;
ALTER TABLE bunny_video_jobs ADD CONSTRAINT bunny_video_jobs_state_check
 CHECK (state IN ('created','ready','failed','deleted'));
ALTER TABLE bunny_video_jobs DROP COLUMN cleanup_requested_at,
 DROP COLUMN cleanup_next_attempt_at, DROP COLUMN cleanup_attempts;
