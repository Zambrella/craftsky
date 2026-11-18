-- Freeze presentation with each durable request so an uncertain retry never
-- changes Stripe's idempotency parameters. Existing sessions remain hosted.
ALTER TABLE billing_web_checkouts ADD COLUMN embedded BOOLEAN NOT NULL DEFAULT false;
