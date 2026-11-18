-- A rollback must not silently discard externally granted access or its audit
-- history. Resolve grants explicitly before reverting this schema.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM billing_promotional_grants) THEN
        RAISE EXCEPTION 'resolve and remove promotional grants before rollback';
    END IF;
END $$;
DROP FUNCTION appview_effective_subscription_access(TEXT,TIMESTAMPTZ);
DROP TABLE billing_promotional_grants;
