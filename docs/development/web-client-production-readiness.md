# Web client production readiness — 4 November 2026

Decision: **do not promote this candidate yet**. Review target: `web-client/`,
starting at commit `e581f73d3fa5`, with the local fixes described below.

## Release blocker

The existing `authenticated-navigation-parity.spec.ts` scenario “hashtag Back
returns to the originating feed or post and restores paginated feed position”
fails on both the production preview and Cloudflare Worker artifact. Load the
second timeline page, scroll to the sixteenth post, open its hashtag, then use
Back. All twenty posts return, but scroll position is 0 instead of 3197 in the
fixture viewport. The first restoration assertion fails.

Reproduce from `web-client/` after `npm run build`:

```sh
npm run test:e2e:app -- --grep 'hashtag Back' --repeat-each=3
WEB_CLIENT_WORKER=1 npm run test:e2e:app -- --grep 'hashtag Back'
```

Three isolated repetitions failed on preview, and the full Worker suite also
reproduced the failure. Temporary tracing showed the popstate navigation entering
SvelteKit's completion code but its awaited UI-settlement promise not resolving
before the assertion deadline. Consequently AppShell's `afterNavigate` callback
and `PageHistory.arrive` did not run for that return. This locates the stalled
path; the underlying cause is still unconfirmed. All temporary application and
framework tracing was removed. No speculative navigation workaround was retained.

## Fixes from this review

- Shared toast actions now use the theme's ink color. The previous dark warning
  Retry text had a 3.04:1 contrast ratio against the warning background, below
  the 4.5:1 threshold. Added explicit light/dark failure-toast accessibility tests;
  both pass on preview and Worker.
- The reduced-motion like test could outlive the 400 ms reaction while making
  separate browser assertions. Both motion variants passed three isolated runs
  before any test change. The test now controls time with Playwright's clock;
  pixel stability/movement and removal assertions remain. Both variants pass
  after the change on preview and Worker. Application animation behavior is
  unchanged.
- Updated the README's dependency-audit findings.

## Validation

- Svelte check: 0 errors, 0 warnings, including after the changes.
- Unit tests: 316 passed across 51 files.
- Production build: passed; verified zero `.map` files in either public output
  directory. Private debug-ID source maps were prepared without upload.
- Public browser tests: 9 passed on preview and 9 passed on Worker.
- Initial full authenticated preview run: 310 passed, 5 configuration skips,
  3 failures (toast contrast, animation timing, and the navigation blocker).
  Focused recheck after changes: the four toast/animation cases passed; navigation
  remains failing. The complete preview suite has not been rerun after the fixes.
- Full authenticated Worker run after the fixes: 314 passed, 5 configuration
  skips, 1 failure (the navigation blocker).
- Deployment-wrapper tests: 7 passed. Development-launcher tests: 5 passed.
- Dependency audit: zero production-dependency advisories. Full audit: four low
  entries through build-time deepmerge, four moderate entries through Sentry's
  glob/minimatch build chain, zero high/critical. The latest Sentry plugin still
  requires the affected glob major. Patched glob/minimatch versions require major
  dependency changes; no forced override or toolchain downgrade was applied.
  See [glob advisory](https://github.com/advisories/GHSA-3g3m-2fjr-6pw4) and
  [minimatch advisory](https://github.com/advisories/GHSA-g3hv-75wg-m249).

## Hosted checks and launch requirements

Read-only smoke checks passed for the hosted preview's welcome/sign-in pages,
both native callbacks and both association documents, including required privacy,
cache and indexing headers. This checks the existing hosted preview, not that it
contains the current local fixes. Production AppView `/healthz` returned HTTP 200
with status `ok`. Public `https://app.craftsky.social/sign-in` returned HTTP 403
from this machine, so public application availability is not verified.

Before public promotion:

1. Resolve the navigation blocker and pass the complete gate with
   `just web-client-check`.
2. Upload a clean, reviewed candidate with `just web-client-preview`; verify real
   hosted sign-in against the required AppView browser callback support. This
   review did not authenticate against or mutate production accounts.
3. Upload private Sentry maps for that exact candidate if reporting is enabled.
   Browser push is optional and requires its public Firebase/VAPID configuration
   and actual delivery verification if enabled. Subscription and video-upload
   controls default off in the checked-in Worker configuration.
4. Follow the [release runbook](../operations/releases.md#web-client-preview-then-release)
   for manual promotion, first domain cutover, public smoke checks, sign-in,
   publication/sign-out checks and native app-link verification.

No deployment, promotion, DNS change, source-map upload or production write was
performed. Chromium fixture coverage does not prove Safari/Firefox behavior,
physical assistive technology, live push/video processing or hosted OAuth.
