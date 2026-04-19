-- Drop in reverse dependency order: craftsky_sessions has FK → oauth_sessions.
-- oauth_auth_requests has no dependents; drop it alongside for a clean rollback.
DROP TABLE IF EXISTS craftsky_sessions;
DROP TABLE IF EXISTS oauth_auth_requests;
DROP TABLE IF EXISTS oauth_sessions;
