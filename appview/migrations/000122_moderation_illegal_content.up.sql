ALTER TABLE moderation_decisions
    DROP CONSTRAINT moderation_decisions_reason_check,
    ADD COLUMN illegal_content_rationale TEXT,
    ADD CONSTRAINT moderation_decisions_reason_check CHECK (reason IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated', 'adult_or_graphic', 'impersonation', 'off_topic', 'intellectual_property', 'other', 'child_safety', 'sexual_violation', 'threat_or_violence', 'self_harm', 'stalking_or_privacy', 'fraud_or_scam', 'misleading_commercial', 'illegal_content'
    )),
    ADD CONSTRAINT moderation_decisions_illegal_content_check CHECK (
        (reason = 'illegal_content'
         AND legal_classification IS NOT DISTINCT FROM 'otherIllegalContent'
         AND COALESCE(btrim(internal_evidence_notes), '') <> ''
         AND COALESCE(btrim(user_safe_detail), '') <> ''
         AND COALESCE(btrim(illegal_content_rationale), '') <> '')
        OR (reason IS DISTINCT FROM 'illegal_content' AND illegal_content_rationale IS NULL)
    );
