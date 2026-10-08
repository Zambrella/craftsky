-- Safety operator credentials are verifier-only. Raw bearer tokens must never be
-- persisted; authentication stores only fixed-size SHA-256 digests.

CREATE TABLE safety_operators (
    id              UUID        NOT NULL PRIMARY KEY,
    actor_id        TEXT        NOT NULL UNIQUE,
    role            TEXT        NOT NULL CHECK (role IN ('moderator', 'helper', 'safetyAdministrator')),
    active          BOOLEAN     NOT NULL DEFAULT true,
    deactivated_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    CONSTRAINT safety_operators_values_check CHECK (
        btrim(actor_id) <> '' AND char_length(actor_id) <= 256
        AND updated_at >= created_at
        AND ((active AND deactivated_at IS NULL)
            OR (NOT active AND deactivated_at IS NOT NULL AND deactivated_at >= created_at))
    )
);

CREATE INDEX safety_operators_active_role_idx
    ON safety_operators(role, actor_id) WHERE active;

CREATE TABLE safety_operator_tokens (
    id            UUID        NOT NULL PRIMARY KEY,
    operator_id   UUID        NOT NULL REFERENCES safety_operators(id) ON DELETE RESTRICT,
    token_digest  BYTEA       NOT NULL UNIQUE CHECK (octet_length(token_digest) = 32),
    active        BOOLEAN     NOT NULL DEFAULT true,
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL,
    last_used_at  TIMESTAMPTZ,
    CONSTRAINT safety_operator_tokens_lifecycle_check CHECK (
        expires_at > created_at
        AND (last_used_at IS NULL OR last_used_at >= created_at)
        AND ((active AND revoked_at IS NULL)
            OR (NOT active AND revoked_at IS NOT NULL AND revoked_at >= created_at))
    )
);

CREATE INDEX safety_operator_tokens_active_digest_idx
    ON safety_operator_tokens(token_digest, expires_at) WHERE active;

CREATE TABLE safety_operator_permissions (
    operator_id  UUID        NOT NULL REFERENCES safety_operators(id) ON DELETE RESTRICT,
    permission   TEXT        NOT NULL CHECK (permission IN (
        'incident.readSafe', 'incident.confirm', 'evidence.access',
		'evidence.preserve', 'hold.manage', 'authority.report',
		'disclosure.approve', 'scan.retry', 'workflow.manage'
    )),
    granted_by   TEXT        NOT NULL,
    granted_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ,
    PRIMARY KEY (operator_id, permission),
    CONSTRAINT safety_operator_permissions_values_check CHECK (
        btrim(granted_by) <> '' AND char_length(granted_by) <= 256
        AND (revoked_at IS NULL OR revoked_at >= granted_at)
    )
);

CREATE INDEX safety_operator_permissions_active_idx
    ON safety_operator_permissions(permission, operator_id) WHERE revoked_at IS NULL;

CREATE TABLE safety_operator_assignments (
    id               UUID        NOT NULL PRIMARY KEY,
    incident_id      UUID        NOT NULL REFERENCES safety_incidents(id) ON DELETE CASCADE,
    operator_id      UUID        NOT NULL REFERENCES safety_operators(id) ON DELETE RESTRICT,
    assignment_role  TEXT        NOT NULL CHECK (assignment_role IN ('owner', 'cover', 'helper')),
    assigned_by      TEXT        NOT NULL,
    assigned_at      TIMESTAMPTZ NOT NULL,
    released_at      TIMESTAMPTZ,
    CONSTRAINT safety_operator_assignments_values_check CHECK (
        btrim(assigned_by) <> '' AND char_length(assigned_by) <= 256
        AND (released_at IS NULL OR released_at >= assigned_at)
    )
);

CREATE UNIQUE INDEX safety_operator_assignments_active_idx
    ON safety_operator_assignments(incident_id, operator_id, assignment_role)
    WHERE released_at IS NULL;

CREATE INDEX safety_operator_assignments_operator_idx
    ON safety_operator_assignments(operator_id, assigned_at, id)
    WHERE released_at IS NULL;
