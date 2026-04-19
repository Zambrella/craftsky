ALTER TABLE oauth_auth_requests
    DROP COLUMN handoff_mode,
    DROP COLUMN loopback_redirect_uri;
