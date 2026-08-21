-- Development OAuth requests are short-lived, non-production artifacts. They
-- cannot be represented by the previous constraint and must not block rollback.
DELETE FROM oauth_auth_requests WHERE handoff_mode = 'dev_scheme';

ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT oauth_auth_requests_handoff_check,
    ADD CONSTRAINT oauth_auth_requests_handoff_check
        CHECK (
            handoff_mode IN ('verified_link', 'loopback')
            AND btrim(device_id) <> ''
            AND (
                (handoff_mode = 'verified_link' AND loopback_redirect_uri IS NULL)
                OR
                (handoff_mode = 'loopback' AND loopback_redirect_uri IS NOT NULL)
            )
        );
