-- Never lose idempotency/recovery records for potentially payable sessions.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM billing_web_checkouts) THEN
  RAISE EXCEPTION 'resolve and remove Stripe checkout operations before rollback';
 END IF;
END $$;
DROP TRIGGER billing_web_checkout_pending ON billing_web_checkouts;
DROP FUNCTION appview_sync_web_checkout_pending();
ALTER TABLE billing_accounts DROP COLUMN web_checkout_pending;
DROP TABLE billing_web_checkouts;
