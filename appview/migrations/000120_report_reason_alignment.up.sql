-- Match the in-app allegations accepted by ValidateReportRequest. Keep earlier
-- values valid so existing private reports are preserved without rewriting them.
ALTER TABLE moderation_reports
    DROP CONSTRAINT moderation_reports_reason_type_check,
    ADD CONSTRAINT moderation_reports_reason_type_check CHECK (reason_type IN (
        'harassment', 'hate', 'spam', 'misleading', 'suspected_ai_generated',
        'adult_or_graphic', 'impersonation', 'off_topic', 'intellectual_property', 'other',
        'childSexualAbuse', 'childGrooming', 'childEndangerment',
        'nonConsensualIntimateImage', 'sexualContent', 'credibleThreat', 'violence',
        'selfHarm', 'stalking', 'privacyViolation', 'fraudScam',
        'misleadingCommercial', 'offTopic',
        'childSexualExploitation', 'grooming', 'childAbuseMaterial',
        'intimateImageAbuse', 'sexualExploitation', 'adultContent', 'graphicContent',
        'immediateDanger', 'terrorism', 'doxxing', 'blockEvasion', 'privacy',
        'fraud', 'scam', 'phishing', 'counterfeit', 'platformManipulation', 'suspectedAiGenerated'
    ));
