-- Per Appendix A of docs/superpowers/plans/2026-04-18-appview-oauth-bff.md:
-- indigo's oauth.AuthRequestData drops unknown JSON fields on round-trip,
-- so handoff_mode and loopback_redirect_uri are stored as sibling columns.

ALTER TABLE oauth_auth_requests
    ADD COLUMN handoff_mode TEXT NOT NULL DEFAULT 'deep_link',
    ADD COLUMN loopback_redirect_uri TEXT;
-- Drop the transient default so new rows must supply a value explicitly.
ALTER TABLE oauth_auth_requests ALTER COLUMN handoff_mode DROP DEFAULT;
