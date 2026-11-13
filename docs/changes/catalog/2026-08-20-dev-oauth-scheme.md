---
{
  "id": "2026-08-20-dev-oauth-scheme",
  "title": "Debug-only development OAuth callback scheme",
  "source": "docs/changes/2026-08-20-dev-oauth-scheme",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "auth",
    "development",
    "verified-links"
  ],
  "keywords": [
    "craftsky-dev",
    "CRAFTSKY_DEV_OAUTH_SCHEME",
    "debug manifest",
    "release plist",
    "verified HTTPS"
  ],
  "summary": "The recorded change adds an explicit development-only OAuth custom-scheme fallback while preserving verified HTTPS links for normal operation.",
  "currentDocs": [
    "app/README.md",
    "verified-links/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/auth",
    "app/android",
    "app/ios",
    "appview/internal/auth"
  ],
  "evidence": [
    "docs/changes/2026-08-20-dev-oauth-scheme/05-implementation-plan.md#final-verification-evidence"
  ],
  "supersededBy": [],
  "followUps": [
    "Physical/provider OAuth round trips remain distinct from compiled manifest/plist checks."
  ]
}
---

# Debug-only development OAuth callback scheme

## Outcome and decisions

The recorded change adds an explicit development-only OAuth custom-scheme fallback while preserving verified HTTPS links for normal operation. The fallback requires both client and AppView opt-in and is restricted to debug artifacts. Release Android manifests and iOS URL registrations must exclude craftsky-dev. It does not move PDS OAuth credentials onto the device or broaden the production handoff contract.

## Evidence and limits

Execution records default and opt-in Flutter tests, auth regressions, backend race/static checks and built Android/iOS artifact inspection. Debug contained the fallback while release did not. These artifact checks do not prove actual provider/device handoff behavior; the completion checklist leaves formal re-review unrecorded.

Historical evidence: [05-implementation-plan.md — Final Verification Evidence](../../../docs/changes/2026-08-20-dev-oauth-scheme/05-implementation-plan.md#final-verification-evidence).

## Current entry points

[app/README.md](../../../app/README.md); [verified-links/README.md](../../../verified-links/README.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
