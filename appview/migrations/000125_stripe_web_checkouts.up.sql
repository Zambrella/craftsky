-- A durable operation must exist before Stripe can create a payable session.
-- Checkout URLs are capabilities: persist only the provider session ID and
-- frozen creation arguments, and fetch the URL when the owner resumes.
CREATE TABLE billing_web_checkouts (
 billing_account_id UUID NOT NULL REFERENCES billing_accounts(id) ON DELETE CASCADE,
 environment TEXT NOT NULL CHECK(environment IN ('production','sandbox')),
 operation_id UUID NOT NULL UNIQUE,
 tier TEXT NOT NULL CHECK(tier IN ('plus','business')),
 stripe_account_id TEXT NOT NULL,
 price_id TEXT NOT NULL,
 success_url TEXT NOT NULL,
 cancel_url TEXT NOT NULL,
 session_id TEXT,
 terminal_reconciliation_generation BIGINT CHECK(terminal_reconciliation_generation > 0),
 created_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(billing_account_id,environment)
);

ALTER TABLE billing_accounts ADD COLUMN web_checkout_pending BOOLEAN NOT NULL DEFAULT false;
CREATE FUNCTION appview_sync_web_checkout_pending() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE account UUID;
BEGIN
 account := CASE WHEN TG_OP='DELETE' THEN OLD.billing_account_id ELSE NEW.billing_account_id END;
 UPDATE billing_accounts SET web_checkout_pending=EXISTS(SELECT 1 FROM billing_web_checkouts WHERE billing_account_id=account) WHERE id=account;
 RETURN NULL;
END $$;
CREATE TRIGGER billing_web_checkout_pending AFTER INSERT OR DELETE ON billing_web_checkouts
 FOR EACH ROW EXECUTE FUNCTION appview_sync_web_checkout_pending();
