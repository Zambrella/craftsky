CREATE TABLE billing_promotional_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    billing_account_id UUID NOT NULL REFERENCES billing_accounts(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL,
    revenuecat_subscription_id TEXT NOT NULL,
    environment TEXT NOT NULL CHECK (environment IN ('production','sandbox')),
    tier TEXT NOT NULL CHECK (tier IN ('plus','business')),
    gives_access BOOLEAN NOT NULL,
    expires_at TIMESTAMPTZ,
    accepted_generation BIGINT NOT NULL CHECK (accepted_generation >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id,revenuecat_subscription_id)
);

CREATE INDEX billing_promotional_grants_account_environment
    ON billing_promotional_grants(billing_account_id,environment);

-- Keep the same effective-access projection for API reads, scheduled effects,
-- pin mutations and public Business projections. Promotional expiry is a known
-- non-renewing limit; paid period dates remain informational provider metadata.
CREATE FUNCTION appview_effective_subscription_access(
    candidate_environment TEXT,
    as_of TIMESTAMPTZ DEFAULT statement_timestamp()
) RETURNS TABLE (
    owner_did TEXT,
    tier TEXT,
    gives_access BOOLEAN,
    access_ends_at TIMESTAMPTZ
) LANGUAGE SQL STABLE AS $$
    SELECT DISTINCT ON (source.owner_did)
        source.owner_did,source.tier,source.gives_access,source.access_ends_at
    FROM (
        SELECT license.assigned_did AS owner_did,license.tier,
            COALESCE(subscription.gives_access
            AND subscription.app_id IS NOT NULL
            AND subscription.mapped_tier=license.tier
            AND subscription.anomaly='none'
            AND customer.owner_did=license.assigned_did AND customer.state='active',false) AS gives_access,
            subscription.current_period_ends_at AS access_ends_at
        FROM billing_licenses license
        JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
        JOIN billing_accounts customer ON customer.id=subscription.billing_account_id
        WHERE license.assigned_did IS NOT NULL
          AND subscription.environment=candidate_environment
          AND subscription.store <> 'promotional'
        UNION ALL
        SELECT customer.owner_did,grant_access.tier,true,grant_access.expires_at
        FROM billing_promotional_grants grant_access
        JOIN billing_accounts customer ON customer.id=grant_access.billing_account_id
        WHERE customer.state='active' AND customer.owner_did IS NOT NULL
          AND grant_access.environment=candidate_environment AND grant_access.gives_access
          AND (grant_access.expires_at IS NULL OR grant_access.expires_at > as_of)
    ) AS source
    ORDER BY source.owner_did,source.gives_access DESC,
        CASE source.tier WHEN 'business' THEN 2 WHEN 'plus' THEN 1 ELSE 0 END DESC,
        source.access_ends_at DESC NULLS FIRST
$$;
