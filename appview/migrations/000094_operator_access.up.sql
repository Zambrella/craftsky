ALTER TABLE safety_operators ADD COLUMN retool_user_id BIGINT UNIQUE CHECK (retool_user_id > 0);

ALTER TABLE safety_operator_permissions DROP CONSTRAINT safety_operator_permissions_permission_check;
ALTER TABLE safety_operator_permissions ADD CONSTRAINT safety_operator_permissions_permission_check CHECK (permission IN (
    'incident.readSafe', 'incident.confirm', 'evidence.access', 'evidence.preserve',
    'hold.manage', 'authority.report', 'disclosure.approve', 'scan.retry', 'workflow.manage',
    'moderation.read', 'moderation.manage', 'safety.read', 'intake.manage',
    'eligibility.review', 'operator.manage'
));

CREATE TABLE safety_operator_events (
    id UUID PRIMARY KEY,
	sequence BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    operator_id UUID NOT NULL REFERENCES safety_operators(id) ON DELETE RESTRICT,
    actor_id TEXT NOT NULL CHECK (btrim(actor_id) <> '' AND char_length(actor_id) <= 256),
    action TEXT NOT NULL CHECK (action IN (
        'bootstrapped', 'created', 'tokenIssued', 'tokensRotated', 'tokenRevoked',
        'deactivated', 'permissionGranted', 'permissionRevoked', 'assignmentGranted', 'assignmentReleased'
    )),
    token_id UUID REFERENCES safety_operator_tokens(id) ON DELETE RESTRICT,
    permission TEXT,
	-- Historical references survive independent incident/assignment retention.
	incident_id UUID,
	assignment_id UUID,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX safety_operator_events_operator_idx ON safety_operator_events(operator_id, created_at, id);

CREATE FUNCTION prevent_operator_event_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'operator events are append-only';
END;
$$;
CREATE TRIGGER safety_operator_events_append_only
    BEFORE UPDATE OR DELETE OR TRUNCATE ON safety_operator_events
    FOR EACH STATEMENT EXECUTE FUNCTION prevent_operator_event_mutation();
