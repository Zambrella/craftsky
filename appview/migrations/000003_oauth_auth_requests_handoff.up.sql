-- Per Appendix A of docs/superpowers/plans/2026-04-18-appview-oauth-bff.md:
-- indigo's oauth.AuthRequestData drops unknown JSON fields on round-trip,
-- so handoff_mode and loopback_redirect_uri are stored as sibling columns.
--
-- handoff_mode keeps its DEFAULT permanently: indigo's SaveAuthRequestInfo
-- inserts the row with only (state, data); the AppView's /auth/login handler
-- then UPDATEs handoff_mode + loopback_redirect_uri once the row exists.
-- Without a permanent default, that initial INSERT would fail the NOT NULL
-- constraint.

ALTER TABLE oauth_auth_requests
    ADD COLUMN handoff_mode TEXT NOT NULL DEFAULT 'deep_link',
    ADD COLUMN loopback_redirect_uri TEXT;
