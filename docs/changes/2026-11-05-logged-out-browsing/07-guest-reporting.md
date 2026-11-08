# Guest reporting: approved extension, 8 November 2026

The maintainer approved extending the existing report flow to logged-out visitors. This supersedes the original assumption that every guest mutation requires sign-in. Public post (including comment/reply) and profile reporting are the only new anonymous writes. Other account actions retain their sign-in gates. Existing event, profile-link and chat report permissions are unchanged.

Both clients use the shared report form, reason taxonomy, details field, success state, cancellation and failure recovery. Visitors may optionally supply a contact email; it is private to moderators and is displayed as unverified. Copyright/trade mark complaints retain the existing dedicated email path.

## Server and privacy boundaries

- The public read policy resolves the target before accepting a report. Hidden, opted-out, pending and unavailable subjects retain the existing public error behavior and disclose no protected content.
- A missing reporter is stored as SQL `NULL`, without inventing a DID or trusting caller headers. The existing subject lifecycle fence still prevents writes across terminal transitions. Member reporting keeps its existing active-owner generation checks.
- `ReportStore` creates and attaches the report to the existing moderation case in one transaction. Reports remain private AppView data; no PDS record or external report submission is introduced.
- Only moderator case details include the optional email. Accepted responses contain only `reportId` and `status`; account-owner history excludes reporter details, contact email and report text.
- Public report requests have an 8 KiB body limit and a separate process-wide budget of 10/minute, burst 10, with two concurrent requests. This aggregate budget cannot be bypassed by rotating caller-provided device IDs or forwarded headers. It is shared by visitors and is not a per-person allowance.
- The web proxy enforces same-origin requests, validates a closed post/profile report path, bounds the body and forwards no cookies or bearer credentials. Flutter uses the same closed exception to its anonymous transport. Other writes still fail before network dispatch.
- Report routes are classified as private diagnostic operations, even though their targets were publicly visible. Report text and contact email are not diagnostic context.

## Storage and rollout

Migration `000119_guest_reports` makes `moderation_reports.reporter_did` nullable and adds `contact_email`. It performs no data backfill. Apply it before running the updated AppView. Its down migration intentionally fails while anonymous reports remain, rather than deleting reports or manufacturing an identity.

No production deployment or PDS changes are part of this implementation.

## Validation

Coverage includes anonymous storage/case attachment, canonical targets, invalid and spoofed inputs, public visibility failures, credential stripping, rate limiting, private moderator contact display, owner-history exclusions, optional-email validation, guest report entry points, cancellation and retry. Execution results are recorded in the implementation handoff.

Verified locally:

- `flutter test`: 3,963 passed, 44 skipped; `flutter analyze`: no issues.
- Web unit suite: 464 passed; Svelte check, changed-file ESLint/Prettier and build passed.
- Guest reporting/action browser journeys: 24 passed in the preview runtime and 24 passed in the Cloudflare Worker runtime, covering desktop, mobile Chromium, mobile WebKit and tablet layouts.
- Moderation-console case tests: 23 passed; TypeScript check passed.
- AppView API and moderation package suites passed; route suite passed after updating the exact inventory and private diagnostic manifest. Focused public-report, terminal-owner, body/admission-limit, moderator privacy and migration checks passed.
- Local compose AppView rebuilt, `/healthz` returned 200, schema version is 119 with `dirty=false`, and an invalid anonymous report returned 422 without creating a report. Production was not deployed.
- No running Flutter debug/DTD process was available for hot reload; device interaction was not verified.
