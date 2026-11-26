ALTER TABLE safety_incidents
    ALTER COLUMN scan_result_id DROP NOT NULL,
    ALTER COLUMN provider_reference DROP NOT NULL,
    ALTER COLUMN integrity_metadata_reference DROP NOT NULL,
    DROP CONSTRAINT safety_incidents_incident_kind_check,
    ADD COLUMN source_case_id UUID UNIQUE,
    ADD CONSTRAINT safety_incidents_origin_check CHECK (
        (incident_kind='imageMatch' AND scan_result_id IS NOT NULL AND source_case_id IS NULL
            AND provider_reference IS NOT NULL AND integrity_metadata_reference IS NOT NULL)
        OR
        (incident_kind='manualEvidence' AND scan_result_id IS NULL AND source_case_id IS NOT NULL
            AND provider_reference IS NULL AND integrity_metadata_reference IS NULL)
    );

-- The opaque source case reference survives normal case cleanup. The subject's
-- owner association is handled by the existing restricted evidence purge.
ALTER TABLE safety_incident_events
    DROP CONSTRAINT safety_incident_events_event_type_check,
    ADD CONSTRAINT safety_incident_events_event_type_check CHECK (event_type IN (
        'detected', 'confirmed', 'caseCreated', 'priorityClassified', 'assigned',
        'initialReportSubmitted', 'supplementSubmitted', 'duplicateRecorded',
        'informationRequestReceived', 'informationRequestResponded', 'resolved',
        'manualEvidencePreserved'
    ));
