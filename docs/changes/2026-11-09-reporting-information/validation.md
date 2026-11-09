# Post-report confirmation validation

Owner narrowed scope on 9 November 2026 to brief process/timeline copy after
submission, with no reporter-status UI. Implemented locally on that date.

## Result

Native uses the existing shared `reportSubmitSuccess` localization. Web uses
`features/moderation/report-confirmation.ts` from the post/profile/profile-link/
business-event success paths. Exact matching copy:

> Report received. We’ll review it and take action if needed. We aim to decide
> routine reports within ten UK business days. Urgent safety concerns take priority.

Existing shared notification surfaces allow longer informational messages time
to be read (300ms per word, minimum four seconds, maximum thirty). The short
message timeout remains four seconds. Warnings/errors retain existing behavior;
web hover/focus still pauses dismissal. There is no status screen/history,
lookup endpoint, backend storage/schema change, new contact/preference field,
reporter email automation or changed Bluesky DM report confirmation.

## Checks

| Local check | Result |
|---|---|
| `flutter gen-l10n` | PASS; generated localizations updated. |
| `flutter test --no-pub test/moderation test/shared/messaging` | PASS, 63 tests. Existing report routes, models/providers, guest form and short/sticky messaging coverage. |
| `flutter analyze --no-pub lib/shared/messaging lib/moderation` | PASS, no issues. |
| Web `npm run check` | PASS, zero errors and warnings. |
| Web `npm run build` | PASS; local build only, Sentry maps prepared without upload. |
| Web `npm test -- src/lib/features/social/toasts.test.ts` | PASS, five tests. |
| `npx playwright test --config tests/playwright-guest.config.ts tests/guest-reporting.spec.ts --project chromium-desktop` | PASS, three tests: post and profile success confirmations plus cancellation/failure/retry. Local mock AppView only. |
| `python3 scripts/test_online_safety_readiness.py` | PASS, seven tests. |
| `git diff --check` | PASS. |

DTD discovery found only an unrelated Stash Hub workspace, so no CraftSky app
was available for hot reload; the unrelated app was not connected or reloaded.
Native widget checks do not establish phone/device behavior. The web flow checks
used the local built client with a mock server; no hosted endpoint, live report,
real email, production mutation, deployment or publication was performed.

## Remaining assessment boundary

This is the owner-authorized post-submit copy step. It does not close remaining
pre-submit disclosure, relevant outcome/communication choices or operational
monitoring items in `online-safety/registers/ofcom-control-mapping.md`. The wider
previous implementation proposal was superseded, not approved. None of these
items is being interpreted as requiring a report-status dashboard.

Subsequent change: the owner separately approved optional guide/report/search
support on 9 November 2026. Its added support UI and verification are recorded
in `../2026-11-09-child-safety-support/validation.md`; this earlier copy-only
record remains the history of the narrower initial change.
