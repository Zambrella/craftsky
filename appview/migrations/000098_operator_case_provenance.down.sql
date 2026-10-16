-- Refuse downgrade when real attributed operator cases exist; never erase them.
ALTER TABLE moderation_cases DROP CONSTRAINT moderation_cases_origin_check,
 DROP CONSTRAINT moderation_cases_origin_reference_check,
 ADD CONSTRAINT moderation_cases_origin_check CHECK (origin IN ('userReport','systemDetected')),
 ADD CONSTRAINT moderation_cases_origin_reference_check CHECK ((origin='systemDetected' AND incident_id IS NOT NULL) OR (origin='userReport' AND incident_id IS NULL));
DROP TABLE moderation_case_provenance;
