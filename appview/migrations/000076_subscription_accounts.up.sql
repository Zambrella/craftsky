CREATE TABLE billing_accounts (
    id                         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_did                  TEXT        UNIQUE,
    revenuecat_app_user_id     UUID        NOT NULL UNIQUE,
    state                      TEXT        NOT NULL DEFAULT 'active',
    closed_at                  TIMESTAMPTZ,
    requested_generation       BIGINT      NOT NULL DEFAULT 0,
    claimed_generation         BIGINT,
    reconciled_generation      BIGINT      NOT NULL DEFAULT 0,
    reconciliation_requested_at TIMESTAMPTZ,
    reconciled_at              TIMESTAMPTZ,
    reconciliation_attempts    INTEGER     NOT NULL DEFAULT 0,
    next_attempt_at            TIMESTAMPTZ,
    lease_token                UUID,
    lease_expires_at           TIMESTAMPTZ,
    deletion_requested_generation BIGINT,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT billing_accounts_state_check
        CHECK (state IN ('active', 'closed')),
    CONSTRAINT billing_accounts_lifecycle_check
        CHECK (
            (state = 'active' AND owner_did IS NOT NULL AND closed_at IS NULL)
            OR
            (state = 'closed' AND owner_did IS NULL AND closed_at IS NOT NULL)
        ),
    CONSTRAINT billing_accounts_generation_check
        CHECK (
            requested_generation >= 0
            AND reconciled_generation >= 0
            AND reconciled_generation <= requested_generation
            AND (claimed_generation IS NULL OR claimed_generation >= 0)
            AND reconciliation_attempts >= 0
        ),
    CONSTRAINT billing_accounts_lease_check
        CHECK (
            (claimed_generation IS NULL AND lease_token IS NULL AND lease_expires_at IS NULL)
            OR
            (claimed_generation IS NOT NULL AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL)
        )
);

CREATE TABLE provider_subscriptions (
    id                             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    billing_account_id             UUID        NOT NULL REFERENCES billing_accounts(id) ON DELETE CASCADE,
    project_id                     TEXT        NOT NULL,
    revenuecat_subscription_id     TEXT        NOT NULL,
    product_id                     TEXT        NOT NULL,
    pending_product_id             TEXT,
    app_id                         TEXT,
    store                          TEXT        NOT NULL,
    environment                    TEXT        NOT NULL,
    status                         TEXT        NOT NULL,
    gives_access                   BOOLEAN     NOT NULL,
    pending_payment                BOOLEAN     NOT NULL DEFAULT false,
    auto_renewal_status            TEXT,
    starts_at                      TIMESTAMPTZ,
    current_period_starts_at       TIMESTAMPTZ,
    current_period_ends_at         TIMESTAMPTZ,
    ends_at                        TIMESTAMPTZ,
    mapped_tier                    TEXT,
    accepted_generation            BIGINT      NOT NULL,
    anomaly                        TEXT        NOT NULL DEFAULT 'none',
    created_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT provider_subscriptions_provider_identity_key
        UNIQUE (project_id, revenuecat_subscription_id),
    CONSTRAINT provider_subscriptions_tier_check
        CHECK (mapped_tier IS NULL OR mapped_tier IN ('plus', 'business')),
    CONSTRAINT provider_subscriptions_environment_check
        CHECK (environment IN ('production', 'sandbox')),
    CONSTRAINT provider_subscriptions_generation_check
        CHECK (accepted_generation >= 0),
    CONSTRAINT provider_subscriptions_anomaly_check
        CHECK (anomaly IN ('none', 'same_tier_duplicate', 'cross_tier_conflict', 'unsupported'))
);

CREATE INDEX provider_subscriptions_account_idx
    ON provider_subscriptions (billing_account_id, accepted_generation DESC, id);

CREATE TABLE billing_licenses (
    id                         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_subscription_id   UUID        NOT NULL UNIQUE REFERENCES provider_subscriptions(id) ON DELETE CASCADE,
    tier                       TEXT        NOT NULL,
    assigned_did               TEXT,
    assigned_at                TIMESTAMPTZ,
    last_target_change_at      TIMESTAMPTZ,
    assignable                 BOOLEAN     NOT NULL DEFAULT true,
    anomaly                    TEXT        NOT NULL DEFAULT 'none',
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT billing_licenses_tier_check
        CHECK (tier IN ('plus', 'business')),
    CONSTRAINT billing_licenses_assignment_check
        CHECK (
            (assigned_did IS NULL AND assigned_at IS NULL)
            OR
            (assigned_did IS NOT NULL AND assigned_at IS NOT NULL)
        ),
    CONSTRAINT billing_licenses_anomaly_check
        CHECK (anomaly IN ('none', 'same_tier_duplicate', 'cross_tier_conflict'))
);

CREATE UNIQUE INDEX billing_licenses_one_assignment_per_did_idx
    ON billing_licenses (assigned_did)
    WHERE assigned_did IS NOT NULL;

CREATE INDEX billing_licenses_access_idx
    ON billing_licenses (assigned_did, provider_subscription_id)
    WHERE assigned_did IS NOT NULL;

CREATE INDEX billing_licenses_terminal_purge_idx
    ON billing_licenses (assigned_did, id)
    WHERE assigned_did IS NOT NULL;

CREATE INDEX billing_accounts_terminal_purge_idx
    ON billing_accounts (owner_did, id)
    WHERE owner_did IS NOT NULL;

CREATE INDEX billing_accounts_reconciliation_due_idx
    ON billing_accounts (next_attempt_at, reconciliation_requested_at, id)
    WHERE state = 'active' AND requested_generation > reconciled_generation;
