CREATE TABLE account_onboarding_completions (
    account_did  TEXT PRIMARY KEY,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
