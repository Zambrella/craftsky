# Development OAuth Scheme Requirements

## Context

CraftSky production OAuth completion uses verified HTTPS App/Universal Links. Local Flutter development needs a callback path that does not depend on deployed domain-association files. The previously removed token-bearing custom scheme must not return.

## Requirements

### DEV-OAUTH-001 — Explicit development-only server mode

AppView may accept `handoffMode: dev_scheme` only when running with `APPVIEW_ENV=dev` and `APPVIEW_ENABLE_DEV_OAUTH_SCHEME=true`. Production configuration must reject the flag at startup. The callback destination is fixed by the server and is never client supplied.

### DEV-OAUTH-002 — Code-only callback URLs

Development login completion must use `craftsky-dev:///auth/complete?code=...`. Development account-deletion reauthentication must use `craftsky-dev:///account-deletion/reauth-complete?job-id=...&proof=...`. No CraftSky session bearer may appear in either URL.

### DEV-OAUTH-003 — Debug-only Flutter opt-in

Flutter may request `dev_scheme` only when both `kDebugMode` and the compile-time define `CRAFTSKY_DEV_OAUTH_SCHEME=true` are present. All other builds request `verified_link`.

### DEV-OAUTH-004 — Release artifacts remain scheme-free

Android registers `craftsky-dev` only in the debug manifest. iOS registers it only through the Debug build configuration. Profile and Release artifacts contain no custom OAuth scheme and retain the verified HTTPS configuration.

## Acceptance Criteria

- AC-001: Disabled development AppView rejects `dev_scheme` without starting OAuth.
- AC-002: Enabled development AppView persists and completes `dev_scheme` login.
- AC-003: Production startup rejects `APPVIEW_ENABLE_DEV_OAUTH_SCHEME=true`.
- AC-004: Login and deletion callbacks contain only their approved short-lived parameters.
- AC-005: Debug opt-in sends `dev_scheme`; debug without opt-in and all release/profile builds send `verified_link`.
- AC-006: Debug platform configuration registers both exact callback paths; release configuration registers neither custom-scheme path.

