-- Restricted evidence metadata is separate from both incident metadata and ordinary
-- media. Object bytes live only behind the dedicated EvidenceStore boundary.

CREATE TABLE safety_evidence (
    id                    UUID        NOT NULL PRIMARY KEY,
    incident_id           UUID        NOT NULL REFERENCES safety_incidents(id) ON DELETE RESTRICT,
    object_key            TEXT        NOT NULL UNIQUE,
    integrity_sha256      BYTEA       NOT NULL CHECK (octet_length(integrity_sha256) = 32),
    content_type          TEXT        NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png', 'image/gif', 'image/webp')),
    byte_size             BIGINT      NOT NULL CHECK (byte_size > 0),
    preservation_reason   TEXT        NOT NULL,
    created_by            TEXT        NOT NULL,
    retention_expires_at  TIMESTAMPTZ NOT NULL,
    deleted_at            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL,
    CHECK (btrim(object_key) <> '' AND char_length(object_key) <= 512),
    CHECK (btrim(preservation_reason) <> '' AND char_length(preservation_reason) <= 512),
    CHECK (btrim(created_by) <> '' AND char_length(created_by) <= 256),
    CHECK (retention_expires_at > created_at),
    CHECK (deleted_at IS NULL OR deleted_at >= created_at)
);

CREATE INDEX safety_evidence_incident_idx ON safety_evidence(incident_id, created_at, id);
CREATE INDEX safety_evidence_retention_idx ON safety_evidence(retention_expires_at, id) WHERE deleted_at IS NULL;

CREATE TABLE safety_evidence_accesses (
    id           UUID        NOT NULL PRIMARY KEY,
    evidence_id  UUID        NOT NULL REFERENCES safety_evidence(id) ON DELETE RESTRICT,
    actor_id     TEXT        NOT NULL,
    action       TEXT        NOT NULL CHECK (action IN ('access', 'export', 'delete')),
    reason       TEXT        NOT NULL,
    allowed      BOOLEAN     NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    CHECK (btrim(actor_id) <> '' AND char_length(actor_id) <= 256),
    CHECK (btrim(reason) <> '' AND char_length(reason) <= 512)
);

CREATE INDEX safety_evidence_accesses_evidence_created_idx
    ON safety_evidence_accesses(evidence_id, created_at, id);

CREATE TABLE safety_evidence_exports (
    id             UUID        NOT NULL PRIMARY KEY,
    evidence_id    UUID        NOT NULL REFERENCES safety_evidence(id) ON DELETE RESTRICT,
    actor_id       TEXT        NOT NULL,
    reason         TEXT        NOT NULL,
    manifest       JSONB       NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL,
    CHECK (jsonb_typeof(manifest) = 'object'),
    CHECK (btrim(actor_id) <> '' AND btrim(reason) <> '')
);

CREATE FUNCTION reject_safety_evidence_audit_mutation()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'safety evidence audit tables are append-only';
END;
$$;

CREATE TRIGGER safety_evidence_accesses_append_only
BEFORE UPDATE OR DELETE OR TRUNCATE ON safety_evidence_accesses
FOR EACH STATEMENT EXECUTE FUNCTION reject_safety_evidence_audit_mutation();

CREATE TRIGGER safety_evidence_exports_append_only
BEFORE UPDATE OR DELETE OR TRUNCATE ON safety_evidence_exports
FOR EACH STATEMENT EXECUTE FUNCTION reject_safety_evidence_audit_mutation();

CREATE TABLE safety_legal_holds (
    id           UUID        NOT NULL PRIMARY KEY,
    incident_id  UUID        NOT NULL REFERENCES safety_incidents(id) ON DELETE RESTRICT,
    basis        TEXT        NOT NULL,
    approved_by  TEXT        NOT NULL,
    created_by   TEXT        NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    released_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL,
    CHECK (btrim(basis) <> '' AND char_length(basis) <= 512),
    CHECK (btrim(approved_by) <> '' AND btrim(created_by) <> ''),
    CHECK (expires_at > created_at),
    CHECK (released_at IS NULL OR released_at >= created_at)
);

CREATE INDEX safety_legal_holds_active_idx
    ON safety_legal_holds(incident_id, expires_at, id) WHERE released_at IS NULL;

CREATE TABLE safety_legal_hold_evidence (
    hold_id      UUID        NOT NULL REFERENCES safety_legal_holds(id) ON DELETE CASCADE,
    evidence_id  UUID        NOT NULL REFERENCES safety_evidence(id) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (hold_id, evidence_id)
);

CREATE INDEX safety_legal_hold_evidence_evidence_idx
    ON safety_legal_hold_evidence(evidence_id, hold_id);

CREATE TABLE safety_retention_events (
    id             UUID        NOT NULL PRIMARY KEY,
    data_class     TEXT        NOT NULL,
    item_reference UUID        NOT NULL,
    outcome        TEXT        NOT NULL CHECK (outcome IN ('deleted', 'held', 'live', 'failed')),
    occurred_at    TIMESTAMPTZ NOT NULL,
    UNIQUE (data_class, item_reference, outcome)
);

CREATE INDEX safety_retention_events_occurred_idx
    ON safety_retention_events(occurred_at, id);
