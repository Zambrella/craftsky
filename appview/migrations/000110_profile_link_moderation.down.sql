-- Existing profile-link cases/reports are operational evidence. Refuse rollback
-- while they exist rather than silently delete that evidence.
ALTER TABLE moderation_reports DROP CONSTRAINT moderation_reports_subject_type_check;
ALTER TABLE moderation_outputs DROP CONSTRAINT moderation_outputs_subject_type_check;
ALTER TABLE moderation_cases DROP CONSTRAINT moderation_cases_subject_type_check;
ALTER TABLE moderation_reports ADD CONSTRAINT moderation_reports_subject_type_check CHECK(subject_type IN ('post','account','event'));
ALTER TABLE moderation_outputs ADD CONSTRAINT moderation_outputs_subject_type_check CHECK(subject_type IN ('post','account','event'));
ALTER TABLE moderation_cases ADD CONSTRAINT moderation_cases_subject_type_check CHECK(subject_type IN ('post','account','event'));
