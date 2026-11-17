-- Account identity and provider history are retained. Reassignment must only be
-- re-enabled together with a deliberate rollback of the application policy.
ALTER TABLE billing_licenses ALTER COLUMN assignable SET DEFAULT true;
DROP TRIGGER IF EXISTS billing_licenses_enforce_did_ownership ON billing_licenses;
DROP FUNCTION IF EXISTS enforce_subscription_did_ownership();

ALTER TABLE provider_subscriptions DROP COLUMN IF EXISTS pending_mapped_tier;
