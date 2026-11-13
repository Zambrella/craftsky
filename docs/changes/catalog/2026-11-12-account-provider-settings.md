---
{
  "id": "2026-11-12-account-provider-settings",
  "title": "Account hosting and provider-management settings",
  "source": "docs/changes/2026-11-12-account-provider-settings.md",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "web-client",
    "appview"
  ],
  "topics": [
    "accounts",
    "settings",
    "federation"
  ],
  "keywords": [
    "hostingProvider",
    "pdsUrl",
    "accountManagementUrl",
    "passwordChangeUrl",
    "shared AT Protocol account"
  ],
  "summary": "Added safe provider-management links and account identifiers while distinguishing CraftSky deletion from deletion of the wider AT Protocol account.",
  "currentDocs": [
    "docs/architecture.md",
    "docs/operations/oauth-permissions.md"
  ],
  "code": [
    "appview/internal/api/account_hosting.go",
    "app/lib/settings",
    "web-client/src/lib"
  ],
  "evidence": [
    "docs/changes/2026-11-12-account-provider-settings.md#account-provider-settings"
  ],
  "supersededBy": [],
  "followUps": [
    "The single-file record contains no complete validation/deployment transcript; consult current checks before release assertions."
  ]
}
---

# Account hosting and provider-management settings

Mobile and web account settings display handle, DID and hosting provider, with copy actions and external provider-management links. The CraftSky deletion section explicitly concerns the CraftSky account rather than deleting the shared AT Protocol identity or provider account.

The additive whoami fields come from the server-held identity directory. Provider selection uses the resolved PDS endpoint rather than the handle: reviewed Bluesky hosts receive Bluesky account/password links, while independent HTTPS origins use the reference `/account` convention with copy explaining provider variation. Endpoints with credentials, paths, queries or fragments do not become hosting links.

Returned details must match the active DID; native account leases and web component recreation fence account-switch completions. Missing hosting metadata leaves ordinary account details and CraftSky deletion usable. This single-file record documents the implemented contract but does not itself provide a complete test/deployment transcript. No new OAuth permission, credential handoff, migration or Lexicon is required.

Evidence: [2026-11-12-account-provider-settings.md](../2026-11-12-account-provider-settings.md#account-provider-settings). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
