DROP TABLE safety_incident_resolutions;
DROP TABLE safety_authority_information_requests;
DROP INDEX safety_authority_reports_one_initial_idx;
DROP TABLE safety_authority_reports;

DROP INDEX safety_incident_events_replay_idx;
ALTER TABLE safety_incident_events
    DROP CONSTRAINT safety_incident_events_event_type_check,
    DROP COLUMN reference_id,
    DROP COLUMN replay_id,
    DROP COLUMN source_system,
    DROP COLUMN actor_id,
    ADD CONSTRAINT safety_incident_events_event_type_check CHECK (event_type IN ('detected')),
    ADD CONSTRAINT safety_incident_events_incident_id_event_type_key UNIQUE (incident_id,event_type);

ALTER TABLE safety_incidents
    DROP CONSTRAINT safety_incidents_state_check,
    DROP COLUMN reporting_deadline,
    DROP COLUMN assigned_to,
    DROP COLUMN priority,
    ADD CONSTRAINT safety_incidents_state_check CHECK (state IN ('detected','confirmed','rejected','closed'));

ALTER TABLE moderation_reports
    DROP CONSTRAINT moderation_reports_reason_type_check,
    ADD CONSTRAINT moderation_reports_reason_type_check CHECK (reason_type IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated',
        'adult_or_graphic', 'impersonation', 'off_topic', 'intellectual_property', 'other'
    ));

ALTER TABLE moderation_decisions
    DROP CONSTRAINT moderation_decisions_reason_check,
    DROP COLUMN legal_classification,
    ADD CONSTRAINT moderation_decisions_reason_check CHECK (reason IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated',
        'adult_or_graphic', 'impersonation', 'off_topic', 'intellectual_property', 'other'
    ));

DROP INDEX moderation_cases_incident_subject_idx;
DROP INDEX moderation_cases_one_open_subject_origin_idx;
CREATE UNIQUE INDEX moderation_cases_one_open_subject_idx
    ON moderation_cases(subject_key) WHERE state='open';
ALTER TABLE moderation_cases
    DROP CONSTRAINT moderation_cases_origin_reference_check,
    DROP COLUMN incident_id,
    DROP COLUMN origin;
