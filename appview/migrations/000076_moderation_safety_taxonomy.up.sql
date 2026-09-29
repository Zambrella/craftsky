ALTER TABLE moderation_cases
    ADD COLUMN origin TEXT NOT NULL DEFAULT 'userReport'
        CHECK (origin IN ('userReport', 'systemDetected')),
    ADD COLUMN incident_id UUID REFERENCES safety_incidents(id) ON DELETE RESTRICT,
    ADD CONSTRAINT moderation_cases_origin_reference_check CHECK (
        (origin='userReport' AND incident_id IS NULL)
        OR (origin='systemDetected' AND incident_id IS NOT NULL)
    );

DROP INDEX moderation_cases_one_open_subject_idx;
CREATE UNIQUE INDEX moderation_cases_one_open_subject_origin_idx
    ON moderation_cases(subject_key,origin) WHERE state='open';
CREATE UNIQUE INDEX moderation_cases_incident_subject_idx
    ON moderation_cases(incident_id,subject_key) WHERE incident_id IS NOT NULL;

ALTER TABLE moderation_decisions
    DROP CONSTRAINT moderation_decisions_reason_check,
    ADD COLUMN legal_classification TEXT CHECK (legal_classification IN (
        'unassessed', 'childSexualExploitation', 'intimateImageAbuse',
        'credibleThreat', 'otherIllegalContent'
    )),
    ADD CONSTRAINT moderation_decisions_reason_check CHECK (reason IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated',
        'adult_or_graphic', 'impersonation', 'off_topic',
        'intellectual_property', 'other', 'child_safety', 'sexual_violation',
        'threat_or_violence', 'self_harm', 'stalking_or_privacy',
        'fraud_or_scam', 'misleading_commercial'
    ));

ALTER TABLE moderation_reports
    DROP CONSTRAINT moderation_reports_reason_type_check,
    ADD CONSTRAINT moderation_reports_reason_type_check CHECK (reason_type IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated',
        'adult_or_graphic', 'impersonation', 'off_topic', 'intellectual_property', 'other',
        'childSexualAbuse', 'childGrooming', 'childEndangerment',
        'nonConsensualIntimateImage', 'sexualContent', 'credibleThreat', 'violence',
        'selfHarm', 'stalking', 'privacyViolation', 'fraudScam',
        'misleadingCommercial', 'offTopic'
    ));

ALTER TABLE safety_incidents
    DROP CONSTRAINT safety_incidents_state_check,
    ADD COLUMN priority TEXT CHECK (priority IN ('immediate', 'urgent', 'standard')),
    ADD COLUMN assigned_to TEXT,
    ADD COLUMN reporting_deadline TIMESTAMPTZ,
    ADD CONSTRAINT safety_incidents_state_check CHECK (state IN (
        'detected', 'confirmed', 'caseCreated', 'reporting', 'resolved', 'rejected', 'closed'
    ));

ALTER TABLE safety_incident_events
    DROP CONSTRAINT safety_incident_events_event_type_check,
    DROP CONSTRAINT safety_incident_events_incident_id_event_type_key,
    ADD COLUMN actor_id TEXT,
    ADD COLUMN source_system TEXT,
    ADD COLUMN replay_id TEXT,
    ADD COLUMN reference_id TEXT,
    ADD CONSTRAINT safety_incident_events_event_type_check CHECK (event_type IN (
        'detected', 'confirmed', 'caseCreated', 'priorityClassified', 'assigned',
        'initialReportSubmitted', 'supplementSubmitted', 'duplicateRecorded',
        'informationRequestReceived', 'informationRequestResponded', 'resolved'
    ));
CREATE UNIQUE INDEX safety_incident_events_replay_idx
    ON safety_incident_events(source_system,replay_id);

CREATE TABLE safety_authority_reports (
    id                  UUID        NOT NULL PRIMARY KEY,
    incident_id         UUID        NOT NULL REFERENCES safety_incidents(id) ON DELETE RESTRICT,
    report_kind         TEXT        NOT NULL CHECK (report_kind IN ('initial', 'supplement', 'duplicate')),
    authority_reference TEXT        NOT NULL,
    duplicate_of_id     UUID        REFERENCES safety_authority_reports(id) ON DELETE RESTRICT,
    submitted_by        TEXT        NOT NULL,
    submitted_at        TIMESTAMPTZ NOT NULL,
    reporting_deadline  TIMESTAMPTZ NOT NULL,
    deadline_met        BOOLEAN     NOT NULL,
    retention_until     TIMESTAMPTZ NOT NULL,
    source_system       TEXT        NOT NULL,
    replay_id           TEXT        NOT NULL,
    UNIQUE (source_system,replay_id),
    CONSTRAINT safety_authority_reports_values_check CHECK (
        btrim(authority_reference) <> '' AND char_length(authority_reference) <= 512
        AND btrim(submitted_by) <> '' AND retention_until > submitted_at
        AND ((report_kind='duplicate' AND duplicate_of_id IS NOT NULL)
            OR (report_kind<>'duplicate' AND duplicate_of_id IS NULL))
    )
);
CREATE INDEX safety_authority_reports_incident_idx
    ON safety_authority_reports(incident_id,submitted_at,id);
CREATE UNIQUE INDEX safety_authority_reports_one_initial_idx
    ON safety_authority_reports(incident_id) WHERE report_kind='initial';

CREATE TABLE safety_authority_information_requests (
    id                  UUID        NOT NULL PRIMARY KEY,
    incident_id         UUID        NOT NULL REFERENCES safety_incidents(id) ON DELETE RESTRICT,
    authority_reference TEXT        NOT NULL,
    state               TEXT        NOT NULL CHECK (state IN ('received', 'responded')),
    due_at              TIMESTAMPTZ NOT NULL,
    received_at         TIMESTAMPTZ NOT NULL,
    responded_at        TIMESTAMPTZ,
    received_by         TEXT        NOT NULL,
    responded_by        TEXT,
    source_system       TEXT        NOT NULL,
    replay_id           TEXT        NOT NULL,
    UNIQUE (source_system,replay_id),
    CONSTRAINT safety_authority_requests_values_check CHECK (
        btrim(authority_reference) <> '' AND char_length(authority_reference) <= 512
        AND due_at >= received_at
        AND ((state='received' AND responded_at IS NULL AND responded_by IS NULL)
            OR (state='responded' AND responded_at IS NOT NULL AND btrim(responded_by) <> ''))
    )
);
CREATE INDEX safety_authority_requests_incident_idx
    ON safety_authority_information_requests(incident_id,received_at,id);

CREATE TABLE safety_incident_resolutions (
    id           UUID        NOT NULL PRIMARY KEY,
    incident_id  UUID        NOT NULL UNIQUE REFERENCES safety_incidents(id) ON DELETE RESTRICT,
    outcome      TEXT        NOT NULL CHECK (outcome IN ('reportedAndClosed', 'noReportRequired', 'duplicateClosed')),
    resolved_by  TEXT        NOT NULL,
    resolved_at  TIMESTAMPTZ NOT NULL,
    CONSTRAINT safety_incident_resolutions_actor_check CHECK (btrim(resolved_by) <> '')
);
