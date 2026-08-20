ALTER TABLE oauth_auth_requests
    DROP CONSTRAINT oauth_auth_requests_handoff_check,
    ADD CONSTRAINT oauth_auth_requests_handoff_check
        CHECK (
            handoff_mode IN ('verified_link', 'loopback', 'dev_scheme')
            AND btrim(device_id) <> ''
            AND (
                (handoff_mode IN ('verified_link', 'dev_scheme') AND loopback_redirect_uri IS NULL)
                OR
                (handoff_mode = 'loopback' AND loopback_redirect_uri IS NOT NULL)
            )
        );
