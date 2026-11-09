DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM moderation_decisions WHERE reason = 'illegal_content') THEN
        RAISE EXCEPTION 'cannot remove illegal-content decision support while such decisions exist';
    END IF;
END $$;
ALTER TABLE moderation_decisions
    DROP CONSTRAINT moderation_decisions_illegal_content_check,
    DROP CONSTRAINT moderation_decisions_reason_check,
    DROP COLUMN illegal_content_rationale,
    ADD CONSTRAINT moderation_decisions_reason_check CHECK (reason IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated', 'adult_or_graphic', 'impersonation', 'off_topic', 'intellectual_property', 'other', 'child_safety', 'sexual_violation', 'threat_or_violence', 'self_harm', 'stalking_or_privacy', 'fraud_or_scam', 'misleading_commercial'
    ));
