-- Queue mutations can acquire these tables in another order. Release partial
-- acquisitions before waiting to avoid holding a lock a worker needs.
DO $$
BEGIN
    FOR attempt IN 1..100 LOOP
        BEGIN
            IF to_regclass('chat_push_deliveries') IS NOT NULL THEN
                LOCK TABLE push_account_subscriptions, push_deliveries, chat_push_deliveries
                    IN ACCESS EXCLUSIVE MODE NOWAIT;
            ELSE
                LOCK TABLE push_account_subscriptions, push_deliveries
                    IN ACCESS EXCLUSIVE MODE NOWAIT;
            END IF;
            RETURN;
        EXCEPTION WHEN lock_not_available THEN
            IF attempt = 100 THEN
                RAISE;
            END IF;
        END;
        PERFORM pg_sleep(0.1);
    END LOOP;
END;
$$;

ALTER TABLE push_account_subscriptions
    ADD COLUMN activation_generation BIGINT NOT NULL DEFAULT 1;

-- Null identifies deliveries queued by an earlier AppView binary. Those
-- deliveries continue to use the timestamp fence until they settle.
ALTER TABLE push_deliveries ADD COLUMN subscription_activation_generation BIGINT;
ALTER TABLE IF EXISTS chat_push_deliveries ADD COLUMN subscription_activation_generation BIGINT;

CREATE FUNCTION advance_push_subscription_activation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT OLD.active THEN
        NEW.activation_generation := OLD.activation_generation + 1;
    END IF;
    -- Pre-audit registration binaries cleared this value on every renewal.
    NEW.deactivated_at := OLD.deactivated_at;
    RETURN NEW;
END;
$$;

CREATE TRIGGER push_subscription_activation_trigger
    BEFORE UPDATE OF active ON push_account_subscriptions
    FOR EACH ROW
    WHEN (NEW.active)
    EXECUTE FUNCTION advance_push_subscription_activation();
