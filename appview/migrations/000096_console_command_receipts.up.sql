-- Console retries retain only a fingerprint and safe response, never request
-- prose, restricted bytes, object keys or operator credentials.
CREATE TABLE operator_command_receipts (
 id UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
 operation TEXT NOT NULL,
 resource_type TEXT NOT NULL,
 resource_id UUID,
 actor_id TEXT NOT NULL,
 key_hash BYTEA NOT NULL CHECK (octet_length(key_hash)=32),
 request_hash BYTEA NOT NULL CHECK (octet_length(request_hash)=32),
 response_status INTEGER NOT NULL CHECK (response_status BETWEEN 200 AND 299),
 response_body BYTEA NOT NULL CHECK (octet_length(response_body)<=16384),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(actor_id,key_hash)
);
CREATE TRIGGER operator_command_receipts_append_only BEFORE UPDATE OR DELETE OR TRUNCATE ON operator_command_receipts
FOR EACH STATEMENT EXECUTE FUNCTION reject_safety_workflow_event_mutation();
CREATE FUNCTION increment_operator_resource_revision() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.revision := OLD.revision + 1; RETURN NEW; END; $$;
ALTER TABLE safety_workflows ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision>=0);
CREATE TRIGGER safety_workflows_revision BEFORE UPDATE ON safety_workflows FOR EACH ROW EXECUTE FUNCTION increment_operator_resource_revision();
ALTER TABLE safety_incidents ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision>=0);
CREATE TRIGGER safety_incidents_revision BEFORE UPDATE ON safety_incidents FOR EACH ROW EXECUTE FUNCTION increment_operator_resource_revision();
ALTER TABLE safety_evidence ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision>=0);
CREATE TRIGGER safety_evidence_revision BEFORE UPDATE ON safety_evidence FOR EACH ROW EXECUTE FUNCTION increment_operator_resource_revision();
ALTER TABLE safety_legal_holds ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision>=0);
CREATE TRIGGER safety_legal_holds_revision BEFORE UPDATE ON safety_legal_holds FOR EACH ROW EXECUTE FUNCTION increment_operator_resource_revision();
ALTER TABLE external_safety_intakes ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision>=0);
CREATE TRIGGER external_safety_intakes_revision BEFORE UPDATE ON external_safety_intakes FOR EACH ROW EXECUTE FUNCTION increment_operator_resource_revision();
ALTER TABLE image_scan_jobs ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision>=0);
CREATE TRIGGER image_scan_jobs_revision BEFORE UPDATE ON image_scan_jobs FOR EACH ROW EXECUTE FUNCTION increment_operator_resource_revision();

ALTER TABLE safety_authority_information_requests ADD COLUMN revision BIGINT NOT NULL DEFAULT 0 CHECK (revision>=0);
CREATE TRIGGER safety_authority_information_requests_revision BEFORE UPDATE ON safety_authority_information_requests FOR EACH ROW EXECUTE FUNCTION increment_operator_resource_revision();
CREATE TABLE safety_hold_releases (
 hold_id UUID PRIMARY KEY REFERENCES safety_legal_holds(id) ON DELETE RESTRICT,
 actor_id TEXT NOT NULL CHECK (btrim(actor_id)<>''),
 reason TEXT NOT NULL CHECK (btrim(reason)<>'' AND char_length(reason)<=512),
 released_at TIMESTAMPTZ NOT NULL
);
CREATE TRIGGER safety_hold_releases_append_only BEFORE UPDATE OR DELETE OR TRUNCATE ON safety_hold_releases
FOR EACH STATEMENT EXECUTE FUNCTION reject_safety_workflow_event_mutation();

CREATE INDEX operator_command_receipts_created_idx ON operator_command_receipts(created_at DESC,id DESC);
CREATE INDEX safety_incidents_created_idx ON safety_incidents(detected_at DESC,id DESC);
CREATE INDEX safety_workflows_created_idx ON safety_workflows(created_at DESC,id DESC);
CREATE INDEX safety_evidence_created_idx ON safety_evidence(created_at DESC,id DESC);
CREATE INDEX safety_legal_holds_created_idx ON safety_legal_holds(created_at DESC,id DESC);
CREATE INDEX external_safety_intakes_created_idx ON external_safety_intakes(created_at DESC,id DESC);
CREATE INDEX image_scan_results_created_idx ON image_scan_results(created_at DESC,id DESC);
CREATE INDEX safety_workflow_events_page_idx ON safety_workflow_events(workflow_id,occurred_at DESC,id DESC);
CREATE INDEX safety_incident_events_page_idx ON safety_incident_events(incident_id,created_at DESC,id DESC);
CREATE INDEX external_safety_correspondence_page_idx ON external_safety_correspondence(intake_id,created_at DESC,id DESC);
