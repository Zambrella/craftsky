CREATE TABLE revenuecat_events (
    event_id           TEXT         PRIMARY KEY,
    event_type         VARCHAR(128) NOT NULL,
    billing_account_id UUID         REFERENCES billing_accounts(id) ON DELETE SET NULL,
    event_occurred_at  TIMESTAMPTZ,
    accepted_at        TIMESTAMPTZ  NOT NULL,
    outcome            TEXT         NOT NULL,

    CONSTRAINT revenuecat_events_outcome_check
        CHECK (outcome IN ('reconciliation_queued', 'ignored_unmapped', 'ignored_closed'))
);

CREATE INDEX revenuecat_events_account_accepted_idx
    ON revenuecat_events (billing_account_id, accepted_at DESC)
    WHERE billing_account_id IS NOT NULL;
