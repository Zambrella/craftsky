-- Provider-neutral, metadata-only external safety intake and legally sensitive
-- workflow chronology. Raw messages, attachment bytes, credentials, and unrestricted
-- contact details must never be stored in these tables.

CREATE TABLE external_safety_intakes (
    id                    UUID        NOT NULL PRIMARY KEY,
    reference             TEXT        NOT NULL UNIQUE,
    provider_message_ref  TEXT        NOT NULL UNIQUE,
    intake_kind           TEXT        NOT NULL CHECK (intake_kind IN ('allegation', 'complaint', 'incident', 'appeal')),
    urgency               TEXT        NOT NULL CHECK (urgency IN ('routine', 'priority', 'urgent', 'immediate')),
    canonical_subject     TEXT        NOT NULL,
    owner_actor_id        TEXT        NOT NULL,
    contact_reference     TEXT,
    attachment_status     TEXT        NOT NULL CHECK (attachment_status IN ('none', 'rejected', 'quarantined')),
    received_at           TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL,
    CONSTRAINT external_safety_intakes_values_check CHECK (
        btrim(reference) <> '' AND char_length(reference) <= 64
        AND btrim(provider_message_ref) <> '' AND char_length(provider_message_ref) <= 512
        AND btrim(canonical_subject) <> '' AND char_length(canonical_subject) <= 2048
        AND btrim(owner_actor_id) <> '' AND char_length(owner_actor_id) <= 256
        AND (contact_reference IS NULL OR (btrim(contact_reference) <> '' AND char_length(contact_reference) <= 512))
    )
);

CREATE INDEX external_safety_intakes_queue_idx
    ON external_safety_intakes(urgency, received_at, id);

CREATE TABLE external_safety_correspondence (
    id                    UUID        NOT NULL PRIMARY KEY,
    intake_id             UUID        NOT NULL REFERENCES external_safety_intakes(id) ON DELETE CASCADE,
    provider_message_ref  TEXT        NOT NULL UNIQUE,
    contact_reference     TEXT,
    attachment_status     TEXT        NOT NULL CHECK (attachment_status IN ('none', 'rejected', 'quarantined')),
    received_at           TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL,
    CONSTRAINT external_safety_correspondence_values_check CHECK (
        btrim(provider_message_ref) <> '' AND char_length(provider_message_ref) <= 512
        AND (contact_reference IS NULL OR (btrim(contact_reference) <> '' AND char_length(contact_reference) <= 512))
    )
);

CREATE INDEX external_safety_correspondence_intake_idx
    ON external_safety_correspondence(intake_id, received_at, id);

CREATE TABLE external_safety_appeal_links (
    intake_id                    UUID NOT NULL PRIMARY KEY REFERENCES external_safety_intakes(id) ON DELETE CASCADE,
    moderation_case_id           UUID NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    appeal_correspondence_id     UUID NOT NULL UNIQUE REFERENCES moderation_appeal_correspondence(id) ON DELETE RESTRICT,
    verified_owner_did           TEXT NOT NULL,
    linked_by_actor_id           TEXT NOT NULL,
    linked_at                    TIMESTAMPTZ NOT NULL,
    CONSTRAINT external_safety_appeal_links_values_check CHECK (
        btrim(verified_owner_did) <> '' AND btrim(linked_by_actor_id) <> ''
    )
);

CREATE INDEX external_safety_appeal_links_case_idx
    ON external_safety_appeal_links(moderation_case_id, linked_at, intake_id);
CREATE INDEX external_safety_appeal_links_owner_idx
    ON external_safety_appeal_links(verified_owner_did, intake_id);

CREATE TABLE safety_workflows (
    id               UUID        NOT NULL PRIMARY KEY,
    intake_id         UUID        REFERENCES external_safety_intakes(id) ON DELETE RESTRICT,
    workflow_class    TEXT        NOT NULL CHECK (workflow_class IN ('intimateImage', 'credibleThreat', 'authorityRequest')),
    state             TEXT        NOT NULL CHECK (state IN ('open', 'inProgress', 'refused', 'resolved')),
    urgency           TEXT        NOT NULL CHECK (urgency IN ('priority', 'immediate')),
    responsible_actor TEXT        NOT NULL,
    received_at       TIMESTAMPTZ NOT NULL,
    target_seconds    BIGINT,
    deadline_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL,
    CONSTRAINT safety_workflows_deadline_check CHECK (
        (workflow_class = 'intimateImage' AND target_seconds IS NOT NULL AND target_seconds > 0
            AND deadline_at = received_at + make_interval(secs => target_seconds))
        OR
        (workflow_class <> 'intimateImage' AND target_seconds IS NULL AND deadline_at IS NULL)
    ),
    CONSTRAINT safety_workflows_values_check CHECK (
        btrim(responsible_actor) <> '' AND updated_at >= created_at
    )
);

CREATE INDEX safety_workflows_deadline_idx
    ON safety_workflows(deadline_at, id)
    WHERE state IN ('open', 'inProgress') AND deadline_at IS NOT NULL;

CREATE TABLE safety_workflow_events (
    id              UUID        NOT NULL PRIMARY KEY,
    workflow_id     UUID        NOT NULL REFERENCES safety_workflows(id) ON DELETE CASCADE,
    sequence        BIGINT      NOT NULL CHECK (sequence > 0),
    event_type      TEXT        NOT NULL CHECK (event_type IN (
        'received', 'standingRecorded', 'declarationsRecorded', 'judgmentRecorded',
        'sameImageSearched', 'substantiallySameSearched', 'actionRecorded',
        'exceptionRecorded', 'outcomeRecorded', 'threatEscalated', 'preservationRecorded',
        'authorityVerified', 'authorityRejected', 'lawfulProcessApproved',
        'disclosureRecorded', 'postIncidentReviewed'
    )),
    actor_id        TEXT        NOT NULL,
    detail_code     TEXT,
    occurred_at     TIMESTAMPTZ NOT NULL,
    CONSTRAINT safety_workflow_events_values_check CHECK (
        btrim(actor_id) <> ''
        AND (detail_code IS NULL OR (btrim(detail_code) <> '' AND char_length(detail_code) <= 256))
    ),
    UNIQUE (workflow_id, sequence)
);

CREATE FUNCTION reject_safety_workflow_event_mutation()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'safety_workflow_events is append-only';
END;
$$;

CREATE TRIGGER safety_workflow_events_append_only
BEFORE UPDATE OR DELETE OR TRUNCATE ON safety_workflow_events
FOR EACH STATEMENT EXECUTE FUNCTION reject_safety_workflow_event_mutation();

CREATE INDEX safety_workflow_events_chronology_idx
    ON safety_workflow_events(workflow_id, occurred_at, sequence);

CREATE TABLE safety_authority_requests (
    workflow_id                UUID        NOT NULL PRIMARY KEY REFERENCES safety_workflows(id) ON DELETE CASCADE,
    requester_reference        TEXT        NOT NULL,
    independent_verification   TEXT,
    legal_basis_reference      TEXT,
    approved_by_actor          TEXT,
    verification_state         TEXT        NOT NULL CHECK (verification_state IN ('pending', 'verified', 'rejected')),
    disclosed_fields           TEXT[]      NOT NULL DEFAULT '{}',
    reviewed_at                TIMESTAMPTZ,
    CONSTRAINT safety_authority_requests_values_check CHECK (
        btrim(requester_reference) <> '' AND char_length(requester_reference) <= 512
        AND NOT ('credentials' = ANY(disclosed_fields))
        AND NOT ('unrelatedData' = ANY(disclosed_fields))
    )
);
