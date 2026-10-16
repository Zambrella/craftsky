-- An operator must explicitly review any loss of retained appeal explanations.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM moderation_appeals WHERE resolution_rationale IS NOT NULL) THEN
        RAISE EXCEPTION 'refusing to discard retained appeal rationales';
    END IF;
END $$;
ALTER TABLE moderation_appeals DROP COLUMN resolution_rationale;
