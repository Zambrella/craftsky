-- Private delivery metadata only. No report text, evidence, recipient or credentials.
CREATE TABLE safety_email_deliveries (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    lease_token UUID,
    lease_until TIMESTAMPTZ,
    sent_at TIMESTAMPTZ
);
CREATE INDEX safety_email_deliveries_pending_idx
    ON safety_email_deliveries(next_attempt_at, created_at) WHERE sent_at IS NULL;
CREATE INDEX safety_email_deliveries_sent_idx ON safety_email_deliveries(sent_at) WHERE sent_at IS NOT NULL;
CREATE TABLE safety_email_sources (
    source_kind TEXT NOT NULL CHECK (source_kind IN ('case','incident','intake','workflow')),
    source_id UUID NOT NULL,
    delivery_id UUID NOT NULL REFERENCES safety_email_deliveries(id),
    PRIMARY KEY (source_kind, source_id)
);
CREATE INDEX safety_email_sources_delivery_idx ON safety_email_sources(delivery_id);
