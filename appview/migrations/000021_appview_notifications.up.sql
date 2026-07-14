CREATE TABLE notification_events (
    id                         UUID        NOT NULL PRIMARY KEY,
    recipient_did              TEXT        NOT NULL,
    actor_did                  TEXT        NOT NULL,
    category                   TEXT        NOT NULL CHECK (category IN ('like', 'follow', 'reply', 'mention', 'quote', 'repost', 'everythingElse')),
    subject_key                TEXT        NOT NULL,
    source_uri                 TEXT        NOT NULL,
    source_cid                 TEXT        NOT NULL,
    source_rkey                TEXT        NOT NULL,
    subject_uri                TEXT,
    subject_cid                TEXT,
    parent_uri                 TEXT,
    parent_cid                 TEXT,
    root_uri                   TEXT,
    root_cid                   TEXT,
    quoted_uri                 TEXT,
    quoted_cid                 TEXT,
    eligibility_scope          TEXT        NOT NULL CHECK (eligibility_scope IN ('everyone', 'peopleIFollow')),
    recipient_followed_actor   BOOLEAN     NOT NULL,
    push_enabled_snapshot      BOOLEAN     NOT NULL,
    state                      TEXT        NOT NULL CHECK (state IN ('active', 'retracted')),
    first_activity_at          TIMESTAMPTZ NOT NULL,
    activity_at                TIMESTAMPTZ NOT NULL,
    indexed_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    initial_push_evaluated_at  TIMESTAMPTZ NOT NULL,
    retracted_at               TIMESTAMPTZ,
    retraction_reason          TEXT,
    CONSTRAINT notification_events_recipient_actor_category_subject_key_key
        UNIQUE (recipient_did, actor_did, category, subject_key)
);

CREATE INDEX notification_events_active_feed_idx
    ON notification_events (recipient_did, activity_at DESC, id DESC)
    WHERE state = 'active';
CREATE INDEX notification_events_source_uri_idx ON notification_events (source_uri);
CREATE INDEX notification_events_subject_uri_idx ON notification_events (subject_uri) WHERE subject_uri IS NOT NULL;
CREATE INDEX notification_events_parent_uri_idx ON notification_events (parent_uri) WHERE parent_uri IS NOT NULL;
CREATE INDEX notification_events_root_uri_idx ON notification_events (root_uri) WHERE root_uri IS NOT NULL;
CREATE INDEX notification_events_quoted_uri_idx ON notification_events (quoted_uri) WHERE quoted_uri IS NOT NULL;
CREATE INDEX notification_events_actor_did_idx ON notification_events (actor_did);

CREATE TABLE notification_preferences (
    account_did   TEXT        NOT NULL,
    category      TEXT        NOT NULL CHECK (category IN ('like', 'follow', 'reply', 'mention', 'quote', 'repost', 'everythingElse')),
    scope         TEXT        NOT NULL CHECK (scope IN ('everyone', 'peopleIFollow')),
    push_enabled  BOOLEAN     NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_did, category)
);

CREATE TABLE push_installations (
    id              UUID        NOT NULL PRIMARY KEY,
    device_id       TEXT        NOT NULL UNIQUE,
    platform        TEXT        NOT NULL CHECK (platform IN ('ios', 'android')),
    fcm_token       TEXT        NOT NULL,
    active          BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX push_installations_active_token_unique
    ON push_installations (fcm_token) WHERE active;

CREATE TABLE push_account_subscriptions (
    id              UUID        NOT NULL PRIMARY KEY,
    installation_id UUID        NOT NULL REFERENCES push_installations(id) ON DELETE CASCADE,
    account_did     TEXT        NOT NULL,
    routing_id      UUID        NOT NULL UNIQUE,
    active          BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at  TIMESTAMPTZ,
    UNIQUE (installation_id, account_did)
);

CREATE INDEX push_account_subscriptions_active_account_idx
    ON push_account_subscriptions (account_did, id) WHERE active;
CREATE INDEX push_account_subscriptions_active_installation_idx
    ON push_account_subscriptions (installation_id, id) WHERE active;

CREATE TABLE push_deliveries (
    id                       UUID        NOT NULL PRIMARY KEY,
    notification_id          UUID        NOT NULL REFERENCES notification_events(id) ON DELETE CASCADE,
    account_subscription_id  UUID        NOT NULL REFERENCES push_account_subscriptions(id) ON DELETE CASCADE,
    status                   TEXT        NOT NULL CHECK (status IN ('pending', 'leased', 'retry', 'succeeded', 'permanent_failure', 'expired', 'cancelled')),
    attempts                 INTEGER     NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at          TIMESTAMPTZ NOT NULL,
    deadline_at              TIMESTAMPTZ NOT NULL,
    lease_owner              TEXT,
    lease_expires_at         TIMESTAMPTZ,
    provider_result_class    TEXT,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at                  TIMESTAMPTZ,
    CONSTRAINT push_deliveries_notification_id_account_subscription_id_key
        UNIQUE (notification_id, account_subscription_id)
);

CREATE INDEX push_deliveries_claim_idx
    ON push_deliveries (next_attempt_at, id) WHERE status IN ('pending', 'retry');
CREATE INDEX push_deliveries_expired_lease_idx
    ON push_deliveries (lease_expires_at, id) WHERE status = 'leased';
CREATE INDEX push_deliveries_queue_age_idx
    ON push_deliveries (created_at, id) WHERE status IN ('pending', 'retry', 'leased');
