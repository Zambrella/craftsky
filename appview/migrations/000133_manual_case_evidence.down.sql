-- Preserve live evidence: restoring the match-only schema is safe only before
-- any manual incident has been recorded.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM safety_incidents WHERE incident_kind='manualEvidence') THEN
        RAISE EXCEPTION 'cannot roll back while manual evidence incident records exist';
    END IF;
END;
$$;

ALTER TABLE safety_incident_events
    DROP CONSTRAINT safety_incident_events_event_type_check,
    ADD CONSTRAINT safety_incident_events_event_type_check CHECK (event_type IN (
        'detected', 'confirmed', 'caseCreated', 'priorityClassified', 'assigned',
        'initialReportSubmitted', 'supplementSubmitted', 'duplicateRecorded',
        'informationRequestReceived', 'informationRequestResponded', 'resolved'
    ));

ALTER TABLE safety_incidents
    DROP CONSTRAINT safety_incidents_origin_check,
    DROP COLUMN source_case_id,
    ALTER COLUMN scan_result_id SET NOT NULL,
    ALTER COLUMN provider_reference SET NOT NULL,
    ALTER COLUMN integrity_metadata_reference SET NOT NULL,
    ADD CONSTRAINT safety_incidents_incident_kind_check CHECK (incident_kind='imageMatch');
