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

DROP TRIGGER push_subscription_activation_trigger ON push_account_subscriptions;
DROP FUNCTION advance_push_subscription_activation();

ALTER TABLE IF EXISTS chat_push_deliveries DROP COLUMN subscription_activation_generation;
ALTER TABLE push_deliveries DROP COLUMN subscription_activation_generation;
ALTER TABLE push_account_subscriptions DROP COLUMN activation_generation;
