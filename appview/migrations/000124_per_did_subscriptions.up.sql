ALTER TABLE provider_subscriptions ADD COLUMN pending_mapped_tier TEXT
    CHECK (pending_mapped_tier IS NULL OR pending_mapped_tier IN ('plus','business'));

-- A customer belongs to exactly one DID. Never silently transfer an existing
-- paid license from a beneficiary to the payer during this architecture change.
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM billing_licenses license
        JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
        JOIN billing_accounts customer ON customer.id=subscription.billing_account_id
        WHERE license.assigned_did IS NOT NULL
          AND license.assigned_did IS DISTINCT FROM customer.owner_did
    ) THEN
        RAISE EXCEPTION 'per-DID subscription migration requires explicit resolution of cross-account licenses';
    END IF;
    IF EXISTS (
        SELECT billing_account_id FROM provider_subscriptions
        WHERE gives_access AND mapped_tier IS NOT NULL AND anomaly='none'
        GROUP BY billing_account_id,environment HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'per-DID subscription migration requires explicit resolution of concurrent subscriptions';
    END IF;
END $$;

UPDATE billing_licenses SET assigned_did=NULL,assigned_at=NULL,assignable=false;
WITH ranked AS (
    SELECT license.id,customer.owner_did,
           row_number() OVER (PARTITION BY customer.id ORDER BY subscription.gives_access DESC,
               subscription.current_period_ends_at DESC NULLS LAST,subscription.created_at DESC,license.id) AS priority
    FROM billing_licenses license
    JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
    JOIN billing_accounts customer ON customer.id=subscription.billing_account_id
    WHERE customer.state='active'
)
UPDATE billing_licenses license SET assigned_did=ranked.owner_did,assigned_at=now()
FROM ranked WHERE ranked.id=license.id AND ranked.priority=1;

ALTER TABLE billing_licenses ALTER COLUMN assignable SET DEFAULT false;

CREATE FUNCTION enforce_subscription_did_ownership() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.assigned_did IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM provider_subscriptions subscription
        JOIN billing_accounts customer ON customer.id=subscription.billing_account_id
        WHERE subscription.id=NEW.provider_subscription_id
          AND customer.state='active' AND customer.owner_did=NEW.assigned_did
    ) THEN
        RAISE EXCEPTION 'subscription access must belong to the customer DID' USING ERRCODE='23514';
    END IF;
    NEW.assignable := false;
    RETURN NEW;
END $$;
CREATE TRIGGER billing_licenses_enforce_did_ownership
BEFORE INSERT OR UPDATE OF assigned_did,provider_subscription_id,assignable ON billing_licenses
FOR EACH ROW EXECUTE FUNCTION enforce_subscription_did_ownership();

-- Refresh provider truth through the normal generation-fenced queue after the
-- ownership projection changes, including environment/catalog validation.
UPDATE billing_accounts
SET requested_generation=requested_generation+1,
    reconciliation_requested_at=now(),next_attempt_at=now(),updated_at=now()
WHERE state='active';
