-- Restricted machine-detection incidents contain safe references only. Suspected
-- image bytes and raw provider payloads must not be stored in these tables.

CREATE TABLE safety_incidents (
    id                            UUID        NOT NULL PRIMARY KEY,
    scan_result_id                UUID        NOT NULL UNIQUE
        REFERENCES image_scan_results(id) ON DELETE RESTRICT,
    incident_kind                 TEXT        NOT NULL DEFAULT 'imageMatch'
        CHECK (incident_kind = 'imageMatch'),
    state                         TEXT        NOT NULL DEFAULT 'detected'
        CHECK (state IN ('detected', 'confirmed', 'rejected', 'closed')),
    provider_reference            TEXT        NOT NULL,
    integrity_metadata_reference  TEXT        NOT NULL,
    detected_at                   TIMESTAMPTZ NOT NULL,
    updated_at                    TIMESTAMPTZ NOT NULL,
    CONSTRAINT safety_incidents_reference_check CHECK (
        btrim(provider_reference) <> '' AND char_length(provider_reference) <= 512
        AND btrim(integrity_metadata_reference) <> ''
        AND char_length(integrity_metadata_reference) <= 512
    ),
    CONSTRAINT safety_incidents_timestamp_check CHECK (updated_at >= detected_at)
);

CREATE INDEX safety_incidents_queue_idx
    ON safety_incidents(state, detected_at, id);

CREATE TABLE safety_incident_subjects (
    incident_id   UUID        NOT NULL REFERENCES safety_incidents(id) ON DELETE CASCADE,
    owner_did     TEXT,
    subject_uri   TEXT        NOT NULL,
    source_cid    TEXT        NOT NULL,
    subject_kind  TEXT        NOT NULL,
    image_slot    TEXT        NOT NULL,
    linked_at     TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (incident_id, subject_uri, image_slot),
    CONSTRAINT safety_incident_subjects_values_check CHECK (
        btrim(owner_did) <> '' AND btrim(subject_uri) <> '' AND btrim(source_cid) <> ''
        AND btrim(subject_kind) <> '' AND btrim(image_slot) <> ''
    )
);

CREATE INDEX safety_incident_subjects_subject_idx
    ON safety_incident_subjects(subject_uri, incident_id);
CREATE INDEX safety_incident_subjects_owner_idx
    ON safety_incident_subjects(owner_did, incident_id, subject_uri, image_slot);

CREATE TABLE safety_incident_events (
    id           UUID        NOT NULL PRIMARY KEY,
    incident_id  UUID        NOT NULL REFERENCES safety_incidents(id) ON DELETE CASCADE,
    event_type   TEXT        NOT NULL CHECK (event_type IN ('detected')),
    created_at   TIMESTAMPTZ NOT NULL,
    UNIQUE (incident_id, event_type)
);

CREATE INDEX safety_incident_events_incident_created_idx
    ON safety_incident_events(incident_id, created_at, id);

CREATE FUNCTION reject_safety_incident_event_mutation()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'safety_incident_events is append-only';
END;
$$;

CREATE TRIGGER safety_incident_events_append_only
BEFORE UPDATE OR DELETE OR TRUNCATE ON safety_incident_events
FOR EACH STATEMENT EXECUTE FUNCTION reject_safety_incident_event_mutation();
