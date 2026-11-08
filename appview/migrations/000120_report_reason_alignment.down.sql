-- Rollback refuses to discard reports using the newly accepted allegations.
-- If any exist, PostgreSQL rejects this change and retains the expanded check.
ALTER TABLE moderation_reports
    DROP CONSTRAINT moderation_reports_reason_type_check,
    ADD CONSTRAINT moderation_reports_reason_type_check CHECK (reason_type IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated',
        'adult_or_graphic', 'impersonation', 'off_topic', 'intellectual_property', 'other',
        'childSexualAbuse', 'childGrooming', 'childEndangerment',
        'nonConsensualIntimateImage', 'sexualContent', 'credibleThreat', 'violence',
        'selfHarm', 'stalking', 'privacyViolation', 'fraudScam',
        'misleadingCommercial', 'offTopic'
    ));
