-- Links are independently reportable and removable from AppView presentation.
-- Moderation never deletes their PDS records.
DO $$
DECLARE item RECORD;
BEGIN
 FOR item IN
  SELECT conrelid::regclass AS relation, conname FROM pg_constraint
  WHERE conrelid IN ('moderation_reports'::regclass,'moderation_outputs'::regclass,'moderation_cases'::regclass)
    AND contype='c' AND pg_get_constraintdef(oid) LIKE '%subject_type%'
 LOOP
  EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I',item.relation,item.conname);
 END LOOP;
END $$;

ALTER TABLE moderation_reports ADD CONSTRAINT moderation_reports_subject_type_check
 CHECK(subject_type IN ('post','account','event','profileLink'));
ALTER TABLE moderation_outputs ADD CONSTRAINT moderation_outputs_subject_type_check
 CHECK(subject_type IN ('post','account','event','profileLink'));
ALTER TABLE moderation_cases ADD CONSTRAINT moderation_cases_subject_type_check
 CHECK(subject_type IN ('post','account','event','profileLink'));

ALTER TABLE moderation_reports ADD CONSTRAINT moderation_reports_subject_shape_check CHECK (
 (subject_type='account' AND subject_collection IS NULL AND subject_rkey IS NULL AND subject_uri IS NULL AND subject_cid_snapshot IS NULL)
 OR (subject_type='post' AND subject_collection IS NOT NULL AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
 OR (subject_type='event' AND subject_collection='social.craftsky.business.event' AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL AND subject_cid_snapshot IS NOT NULL)
 OR (subject_type='profileLink' AND subject_collection='app.bsky.actor.link' AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL AND subject_cid_snapshot IS NOT NULL)
);
ALTER TABLE moderation_outputs ADD CONSTRAINT moderation_outputs_subject_shape_check CHECK (
 (subject_type='account' AND subject_collection IS NULL AND subject_rkey IS NULL AND subject_uri IS NULL)
 OR (subject_type='post' AND subject_collection IS NOT NULL AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
 OR (subject_type='event' AND subject_collection='social.craftsky.business.event' AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
 OR (subject_type='profileLink' AND subject_collection='app.bsky.actor.link' AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
);
ALTER TABLE moderation_cases ADD CONSTRAINT moderation_cases_subject_shape_check CHECK (
 (subject_type='account' AND subject_collection IS NULL AND subject_rkey IS NULL AND subject_uri IS NULL)
 OR (subject_type='post' AND subject_collection IS NOT NULL AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
 OR (subject_type='event' AND subject_collection='social.craftsky.business.event' AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
 OR (subject_type='profileLink' AND subject_collection='app.bsky.actor.link' AND subject_rkey IS NOT NULL AND subject_uri IS NOT NULL)
);
