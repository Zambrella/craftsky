# Web client error handling review

Reviewed 2026-11-05 against `app/lib/shared/errors/app_error_mapper.dart`,
`app_error.dart`, `shared/api/api_exception.dart`, the English localization
catalogue, and `docs/development/logging-error-reporting.md`.

## Shared gaps corrected

- Network fetch and response-read exceptions lost their original causes. The
  transport now retains those causes, including when a keyed write has an
  uncertain outcome. Parsing failures for keyed writes retain their cause too.
  Original operation keys, retry rules, and pending-write classification remain.
- Captured API failures lacked HTTP method and response kind. Both now survive
  the final privacy filter. Unknown stable error codes, status, cause stacks,
  and validated AppView request UUIDs remain available without bodies or URLs.
  Invalid response-header request IDs are rejected at the transport boundary.
  Invalid successful responses retain the available header correlation.
- Social, settings-load, and settings-save captures previously shared the
  `workspace` label. They now have separate bounded labels.
- Shared social/settings and composer API feedback displayed server prose.
  They now use client-owned messages for offline, session expiry, missing content,
  server failures, invalid responses, and other failed actions. English wording
  follows Flutter's generic mapper; rate limiting and pending writes retain
  web-specific actionable copy. Field validation remains inline where available.
- Canceled reads no longer produce generic failure feedback through the shared
  social/settings work controllers. Pending writes still require confirmation.
- Unmapped local `FieldValidationError`s no longer create unexpected-error issues.

## Findings addressed

1. **Consumed failures now have an explicit owner.** Messaging's consuming
   boundary reports the original error with a static action label. Failed polling
   is reported once until a successful cycle; draft/report persistence similarly
   avoids reporting on every keystroke during a continuing storage failure.
   Discovery, push settings, lazy page imports, media editing, draft management,
   notification opening and post actions now report at their consuming boundary.
   Expected browser permission denials and input failures stay out of Sentry.
   Speculative preloads and decorative fallbacks remain silent.
2. **UI copy is selected explicitly.** `UserFacingError` carries reviewed
   client-authored recovery copy; `InputError` and `FieldValidationError` identify
   correctable input. Settings, composer/media, messaging and draft presentation
   use a generic Flutter-aligned fallback for arbitrary dependency exceptions.
   Reviewed archive, image, date, language and local-storage guidance remains
   actionable. Raw account-deletion server messages are no longer displayed.
   These classes do not grant their messages permission to enter diagnostics.
3. **Operation labels identify features and actions.** Social/settings Work
   helpers accept static operation labels, with feature-specific defaults for
   inline actions. Source call sites identify profile reads, follow operations,
   settings loads/saves, composer initialization/save/publish/retry and individual
   post actions (including like versus repost). Original labels survive retries.
   The final sink accepts only the closed operation set; no label is derived
   from a runtime route, resource ID, user text, or private payload.
4. **Worker logging has its own protected boundary.** Unknown SvelteKit errors
   use `handleError`; handled public API, authentication and proxy failures log
   where they are consumed. The hook does not recapture handled HTTP errors.
   Logs preserve selected cause types, deployed Worker frame coordinates,
   operation/stage, status, stable code and valid request correlation. Arbitrary
   prose, cookies, OAuth capabilities, payloads and local filesystem paths are
   excluded. This is server-local logging, independent of browser consent and
   without adding a server Sentry exporter. Sink failures cannot change HTTP
   responses or redirects.
5. **Expected local failures use warning severity.** Both explicit and automatic
   browser local output classify expected failures as “Expected browser failure”
   through `console.warn`; unexpected failures remain `console.error`. Safe
   supporting context remains available even without reporting consent.

## Existing safeguards confirmed in source

Browser reporting is opt-in, with guarded initialization and capture. Sentry
receives selected native cause/stack structure and safe deployment metadata.
Attachments, arbitrary context, private messages, request data, replay, and traces
are excluded. The authenticated shell handles 401 centrally, and the server proxy
removes the expired account session. Field validation can focus invalid controls.
Keyed writes retain their original key when the outcome is uncertain.

## Validation scope

- Complete web unit suite: 342 tests across 55 files.
- Svelte check: zero errors/warnings; production build and diff checks passed.
- Local browser fixtures: eight action-feedback tests and three diagnostic tests
  passed, including real SDK envelope filtering against an intercepted loopback
  endpoint, consent/revocation, retry preservation and private-value canaries.
- Added coverage includes Worker sink serialization and failure isolation,
  public parse-cause ownership, hook ownership, safe UI fallbacks, input
  classification, original messaging causes and polling suppression.

Tests requiring Vite/Playwright servers ran with loopback socket permission.
No Flutter source or API/lexicon contract changed. Nothing was deployed and no
Sentry source maps were uploaded. Live Sentry receipt/symbolication, hosted Worker
log retrieval, device behavior and every feature's browser error state remain
outside this local validation. The build used the documented loopback diagnostic
fixture DSN; this is not a production deployment artifact.
