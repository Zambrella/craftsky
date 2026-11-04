# ADR 026: Web client preview versions and manual promotion

Date: 2026-11-04
Status: Accepted for the requested preview-first release workflow

## Context

The maintainer needs to review a hosted web client, including sign-in, then make
that exact version live from the Cloudflare dashboard. The existing
`app.craftsky.social` host is the `craftsky-verified-links` Pages project. Pages
preview deployments are not valid production rollback targets. Workers supports
uploading a version separately from promoting it to live traffic.

## Decision

Use the `craftsky-public-app` Worker with static assets in the existing CraftSky
Cloudflare account. Upload candidates with `wrangler versions upload`, assigning
the stable `preview` version alias. Promote the reviewed version ID to 100% in
the dashboard. Do not rebuild at promotion. Keep automatic deployment disabled.
The first public release separately moves `app.craftsky.social` from Pages to
the Worker, preserving the native callback pages and association documents.

Extend ADR 025's production browser callback authority to exactly
`https://preview-craftsky-public-app.hugetoothstudios.workers.dev` in addition to
`https://app.craftsky.social`. The default `WEB_APP_ORIGIN` remains the latter.
Development loopback configurations retain their exact origin isolation.
Unique version URLs, other aliases, other accounts and wildcard workers.dev
origins are not accepted as OAuth callback destinations. The stable preview
alias must remain owned by the same trusted Cloudflare account.

All browser handoff path, intent nonce, device binding, one-time exchange,
confirmation and purpose separation rules remain in force. This applies to
sign-in, registration, chat authorization and deletion reauthentication through
their shared completion validator. OAuth credentials and DPoP keys remain in
AppView; the preview uses the existing host-only browser child-session cookies.

The preview is unlisted, with `noindex, nofollow` on rendered responses. This is
not access control. It uses production AppView and real accounts/data; writes
are real. This is not a staging backend. Changing the alias during an OAuth
flow has the same deployment-consistency considerations as a production release.

## Consequences

Sign-in is tested on the stable preview alias; unique version URLs identify the
exact artifact for inspection and promotion but cannot receive OAuth handoffs.
The preview callback change needs a normal exact-tag AppView release before
hosted preview sign-in works. No database migration or lexicon change is needed.
The first domain cutover is separate from routine version promotion. Workers
rollback restores frontend code/configuration, not backend data or mutations.

References: [Worker deployment management](https://developers.cloudflare.com/workers/versions-and-deployments/deployment-management/),
[version URLs](https://developers.cloudflare.com/workers/versions-and-deployments/version-urls/),
[Pages rollback limits](https://developers.cloudflare.com/pages/configuration/rollbacks/).
