ALTER TABLE bunny_video_jobs DROP CONSTRAINT bunny_video_jobs_state_check;
ALTER TABLE bunny_video_jobs ADD CONSTRAINT bunny_video_jobs_state_check
 CHECK (state IN ('created','ready','failed','deleted','deleting','purging'));
ALTER TABLE bunny_video_jobs
 ADD COLUMN cleanup_requested_at TIMESTAMPTZ,
 ADD COLUMN cleanup_next_attempt_at TIMESTAMPTZ,
 ADD COLUMN cleanup_attempts INTEGER NOT NULL DEFAULT 0 CHECK (cleanup_attempts >= 0);
CREATE INDEX bunny_video_jobs_cleanup_due ON bunny_video_jobs(library_id,cleanup_next_attempt_at,id)
 WHERE cleanup_requested_at IS NOT NULL AND state != 'deleted';
CREATE INDEX craftsky_posts_video_reference ON craftsky_posts(did,(record#>>'{embed,video,ref,$link}'))
 WHERE record#>>'{embed,video,ref,$link}' IS NOT NULL;

-- Share the job-row lock with the deletion claim so a reference committed
-- before that claim is always visible to its final reference check.
CREATE FUNCTION bunny_video_reference_lock() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP = 'INSERT' THEN
  PERFORM id FROM bunny_video_jobs WHERE owner_did=NEW.did
   AND blob_cid=NEW.record#>>'{embed,video,ref,$link}' ORDER BY id FOR UPDATE;
 ELSIF TG_OP = 'DELETE' THEN
  PERFORM id FROM bunny_video_jobs WHERE owner_did=OLD.did
   AND blob_cid=OLD.record#>>'{embed,video,ref,$link}' ORDER BY id FOR UPDATE;
 ELSE
  PERFORM id FROM bunny_video_jobs WHERE
   (owner_did=OLD.did AND blob_cid=OLD.record#>>'{embed,video,ref,$link}') OR
   (owner_did=NEW.did AND blob_cid=NEW.record#>>'{embed,video,ref,$link}') ORDER BY id FOR UPDATE;
 END IF;
 IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER bunny_video_reference_lock BEFORE INSERT OR UPDATE OR DELETE ON craftsky_posts
 FOR EACH ROW EXECUTE FUNCTION bunny_video_reference_lock();

CREATE FUNCTION bunny_video_reference_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP != 'DELETE' THEN
  UPDATE bunny_video_jobs SET cleanup_requested_at=NULL,cleanup_next_attempt_at=NULL
   WHERE owner_did=NEW.did AND blob_cid=NEW.record#>>'{embed,video,ref,$link}' AND state='ready';
 END IF;
 IF TG_OP = 'DELETE' THEN
  UPDATE bunny_video_jobs SET cleanup_requested_at=COALESCE(cleanup_requested_at,now()),cleanup_next_attempt_at=now()
   WHERE owner_did=OLD.did AND blob_cid=OLD.record#>>'{embed,video,ref,$link}' AND state='ready';
 ELSIF TG_OP = 'UPDATE' THEN
  IF OLD.did IS DISTINCT FROM NEW.did OR
   OLD.record#>>'{embed,video,ref,$link}' IS DISTINCT FROM NEW.record#>>'{embed,video,ref,$link}' THEN
   UPDATE bunny_video_jobs SET cleanup_requested_at=COALESCE(cleanup_requested_at,now()),cleanup_next_attempt_at=now()
    WHERE owner_did=OLD.did AND blob_cid=OLD.record#>>'{embed,video,ref,$link}' AND state='ready';
  END IF;
 END IF;
 RETURN NULL;
END $$;
CREATE TRIGGER bunny_video_reference_cleanup AFTER INSERT OR UPDATE OR DELETE ON craftsky_posts
 FOR EACH ROW EXECUTE FUNCTION bunny_video_reference_cleanup();
