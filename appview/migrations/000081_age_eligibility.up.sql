CREATE TABLE account_policy_acceptances (
    account_did TEXT PRIMARY KEY,
    policy_version TEXT NOT NULL CHECK (
        btrim(policy_version) = policy_version
        AND policy_version <> ''
        AND char_length(policy_version) <= 128
    ),
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE account_age_eligibility (
    account_did TEXT PRIMARY KEY,
    state TEXT NOT NULL DEFAULT 'eligible' CHECK (state IN ('eligible', 'restricted')),
    evidence_kind TEXT CHECK (evidence_kind IN (
        'self_disclosure',
        'guardian_disclosure',
        'verified_authority',
        'verified_account_record'
    )),
    evidence_reference TEXT,
    reviewer_id TEXT,
    review_reason TEXT,
    appeal_guidance TEXT,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    reviewed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT account_age_eligibility_restriction_check CHECK (
        (state = 'eligible')
        OR
        (state = 'restricted'
            AND evidence_kind IS NOT NULL
            AND evidence_reference IS NOT NULL AND btrim(evidence_reference) <> ''
            AND reviewer_id IS NOT NULL AND btrim(reviewer_id) <> ''
            AND review_reason IS NOT NULL AND btrim(review_reason) <> ''
            AND appeal_guidance IS NOT NULL AND btrim(appeal_guidance) <> ''
            AND reviewed_at IS NOT NULL)
    )
);

CREATE TABLE account_age_eligibility_events (
    id UUID PRIMARY KEY,
    account_did TEXT NOT NULL,
    revision BIGINT NOT NULL CHECK (revision > 0),
    from_state TEXT NOT NULL CHECK (from_state IN ('eligible', 'restricted')),
    to_state TEXT NOT NULL CHECK (to_state IN ('eligible', 'restricted')),
    evidence_kind TEXT,
    evidence_reference TEXT,
    reviewer_id TEXT NOT NULL CHECK (btrim(reviewer_id) <> ''),
    reason TEXT NOT NULL CHECK (btrim(reason) <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (account_did, revision)
);

CREATE INDEX account_age_eligibility_state_idx
    ON account_age_eligibility (state, updated_at, account_did);
CREATE INDEX account_age_eligibility_events_account_idx
    ON account_age_eligibility_events (account_did, created_at, id);
CREATE INDEX account_age_eligibility_events_terminal_idx
    ON account_age_eligibility_events (account_did, id);
