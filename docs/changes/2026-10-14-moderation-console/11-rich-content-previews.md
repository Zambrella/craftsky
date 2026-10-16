# Rich content previews

Date: 16 October 2026

Follow-up: [context, business and version completion](12-preview-context-business-versions.md)
extends this implementation with quoted/reply cards, business records and explicit
case/current version handling.

The case workspace's **Content preview** now displays the indexed subject rather
than a text-only excerpt:

- Accounts: display name, DID, bio, published avatar/banner and crafts.
- Posts: author, timestamp, text, image gallery with alt text and enlargement,
  project/pattern/material details, external link cards and video.
- Events: title, summary, image, start/end dates, venue, status and website links.

Images fail to an explicit unavailable state. The enlarged image dialog closes
with Escape and restores focus. Video is loaded on request, uses ordinary player
controls and never autoplays. Native HLS is preferred; the pinned, lazy-loaded
`hls.js` player supports browsers needing MediaSource playback. Author-provided
video descriptions are displayed. Consumer-authenticated caption download routes
are not included in the admin preview.

The existing admin preview route still requires `moderation.read` and sends
`no-store`. It returns media only for the case's indexed record version. Changed
records, blocked/missing image projections and terminal owners are unavailable;
incident-linked subjects remain restricted. Published profile image columns keep
their existing safety masking. No remote PDS read, raw record exposure, evidence
fetch, permission change or schema migration was added.

The console's media policy allows the existing image CDN and configured video
origins. Scripts remain same-origin, arbitrary frame embedding remains disabled,
and referrers are suppressed. Restricted evidence retains its separate audited
access and in-memory blob viewer. External destinations use validated URLs and
open with `noopener noreferrer`.

## Verification

- Red-to-green HTTP regressions for captured post media and account/event cards;
  rendered workspace regressions for image enlargement, cards and video loading.
- All 47 frontend tests pass. TypeScript and the Vite build pass, including the
  Docker build's frontend checks. The HLS bundle is loaded only when needed;
  Vite reports its size advisory for this separate chunk.
- Complete `internal/routes` and `internal/operatorconsole` race suites pass.
  Focused checks also pass after the final review, including project/link
  hydration, changed/blocked subjects, restricted incidents, helper denial,
  terminal owners and the configured media policy.
- The OpenAPI contract was updated; its JSON parses and all 1,054 references
  resolve. `git diff --check` passes.
- The rebuilt local console loads an actual HTTP preview with two benign textile
  JPEGs. Both decode in the browser, with no horizontal overflow at the measured
  desktop viewport. Image enlargement, Escape and focus restoration work.
  The account fixture displays its bio and crafts.
- Native HLS and forced MediaSource/HLS.js both decode a locally generated
  three-second green video. For that player check only, the synthetic case's
  browser response temporarily pointed to a same-origin test playlist. The
  browser was then fully reloaded to remove that override and the temporary
  development-media files were removed. This verifies the player and console
  policy, not live Bluesky CDN availability.
- Screenshot saving failed in the collaborative preview. Desktop checks use DOM,
  decoded-image and playback evidence. Mobile visual acceptance remains pending
  as previously requested. The full AppView release gate was not repeated for
  this follow-up; the previous cleanup gate is recorded in [report 10](10-prototype-cleanup.md).

Local logs are under `/private/tmp/craftsky-rich-preview-*`, including
`focused.log`, `final-focused.log` and `final-build.log`.
Production deployment/configuration, commits and pushes are outside this change.

## Local review

The current development database has benign post/account fixtures. To add them
to another migrated development database:

```sh
./scripts/compose-dev exec -T postgres psql -v ON_ERROR_STOP=1 \
  -U craftsky -d craftsky_dev < tools/moderation-console/seed-previews.sql
```

Open `/admin/#/cases/MOD-65000000-0000-4000-8000-000000000001` for the photo post,
or `/admin/#/cases/MOD-65000000-0000-4000-8000-000000000002` for the account.
The seed preserves existing decisions, operator access and prior walkthroughs.
