ALTER TABLE moderation_cases DROP CONSTRAINT moderation_cases_origin_check,
 DROP CONSTRAINT moderation_cases_origin_reference_check,
 ADD CONSTRAINT moderation_cases_origin_check CHECK (origin IN ('userReport','systemDetected','operatorCreated','externalIntake')),
 ADD CONSTRAINT moderation_cases_origin_reference_check CHECK ((origin='systemDetected' AND incident_id IS NOT NULL) OR (origin<>'systemDetected' AND incident_id IS NULL));
CREATE TABLE moderation_case_provenance (
 id UUID PRIMARY KEY,
 case_id UUID NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
 intake_id UUID REFERENCES external_safety_intakes(id) ON DELETE RESTRICT,
 actor_id TEXT NOT NULL CHECK (btrim(actor_id)<>''),
 source_system TEXT NOT NULL CHECK (btrim(source_system)<>''),
 reason TEXT NOT NULL CHECK (btrim(reason)<>'' AND char_length(reason)<=512),
 created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX moderation_case_provenance_case_idx ON moderation_case_provenance(case_id,created_at DESC,id DESC);
CREATE INDEX moderation_case_provenance_intake_idx ON moderation_case_provenance(intake_id,case_id) WHERE intake_id IS NOT NULL;
