# Optional browser error reporting

Configure the dedicated `craftsky/web-client` Sentry project using public build-time
`PUBLIC_SENTRY_DSN` and `PUBLIC_SENTRY_ENVIRONMENT` (`development`, `preview` or
`production`). The public DSN is configuration, not an OAuth or upload credential.
Never put Sentry auth tokens in public environment variables.

Reporting is off until the device-wide `craftsky.diagnostics-consent` preference
is explicitly enabled in onboarding or Settings → Privacy → Error reporting.
Account switching does not enable reporting. The client startup hook installs
protected local diagnostics before hydration and initializes the pinned official
`@sentry/browser` SDK only with consent. The root host listens for preference
changes, including changes in another tab. Revocation detaches/closes the client;
both event and transport gates drop unsent envelopes after consent is revoked.
An already sent request cannot be recalled.

The SDK owns unhandled errors/rejections. SvelteKit's client error hook owns
unexpected framework failures. The social/settings Work helpers, startup boundary
and feature consuming boundaries call `reportFailure` with the original error.
Pass a static action label from `DiagnosticOperation`; do not derive labels from
paths, account IDs, private resources or user text. Work helpers carry their
labels through retries. Repeated messaging polling/storage failures report once
until recovery, rather than once per poll or keystroke.
Feature code never imports Sentry. Reporter failures cannot affect normal UI,
responses or retry behavior. Expected cancellation, offline, API 4xx and pending
mutation outcomes are suppressed from Sentry; malformed responses and API 5xx
failures remain actionable. Their supporting local diagnostic output is protected
independently of Sentry and consent. Expected failures use warning-level local
output; unexpected failures use error-level output. Correctable `InputError`
(including field validation) and browser permission denials are expected.

Use `userErrorMessage` for presentation. Only `UserFacingError` and its input
subclasses may expose reviewed client-authored copy; ordinary exceptions use the
Flutter-aligned generic fallback. Never construct these wrappers from dependency
or server prose. Wrappers should retain an original cause when one exists.

Worker logging uses `lib/server/diagnostics.ts`, separately from the browser SDK.
The unknown-error hook and consuming public/auth/proxy boundaries own these logs;
handled HTTP responses are not logged again by the hook. Only selected cause
kinds, Worker coordinates and bounded operational metadata are emitted.

Only GlobalHandlers, bounded LinkedErrors and Dedupe integrations are installed.
SDK 11 uses `dataCollection` rather than `sendDefaultPii`; private collection
categories are disabled. No fetch/XHR instrumentation, replay, session tracking,
console export, breadcrumbs, logs, metrics or performance tracing is enabled.

The final event and transport filters retain native error/cause types, generic
messages, handled/cause structure, application-code stack coordinates, release,
environment, SDK version and source-map debug IDs. Consumed failures select only
static operation labels, bounded stable API error codes, HTTP status and an
AppView-generated UUIDv4 request ID. Envelope request IDs take precedence, with
`X-Request-ID` as fallback. No route, query, user identity, recipient, input,
request/response body, server message, field validation prose, attachment, frame
locals, source context or arbitrary custom field reaches the wire. Preview code
paths are allowed only on the current origin. The transport filters again after
SDK processing so late scope attachments cannot bypass selection.

## Source maps and releases

`npm run build` stamps debug IDs using the official pinned Sentry Vite plugin,
without remote upload, release creation or plugin telemetry. It retains the exact
browser bundles and maps in ignored `.sentry-artifacts/`, then removes maps from
both public output directories. `release.json` records the matching
release label (`craftsky-web@<AppView version>+<git commit>`, or an explicit
`SENTRY_RELEASE` override).

Before deploying that exact build, run `npm run sentry:upload`. The script uses
the authenticated Sentry CLI with explicit `craftsky/web-client` targeting and
`--no-rewrite`; it uploads the retained files without modifying deployed bundles.
Do not rebuild between upload and deployment. Keep the corresponding private
artifact if production debugging will be done later. Upload is intentionally a
separate action; a normal build cannot send source code remotely.

This is browser reporting. AppView keeps its existing server diagnostics; this
change does not install telemetry in the web BFF/Worker, whose private OAuth and
session routes require a separately scoped server integration.

## Verification

Unit tests inspect actual SDK-serialized envelopes with protected-value canaries,
positive cause/stack/debug-ID/correlation fields, opt-in/revocation and SDK failure
containment. For real browser checks without hosted Sentry traffic:

```sh
PUBLIC_SENTRY_DSN=http://public@127.0.0.1:19846/1 PUBLIC_SENTRY_ENVIRONMENT=test npm run build
WEB_CLIENT_DIAGNOSTICS=1 npm run test:e2e:app -- tests/authenticated-diagnostics.spec.ts
WEB_CLIENT_DIAGNOSTICS=1 WEB_CLIENT_WORKER=1 npm run test:e2e:app -- tests/authenticated-diagnostics.spec.ts
```

The browser tests intercept the loopback envelope endpoint, exercise actual
Settings consent/reload/revocation, capture a consumed AppView failure and verify
that expected failures and export outages preserve normal recovery behavior.
Local envelopes and matching debug IDs do not prove hosted ingestion or production
symbolication. Do not inject synthetic failures into production to validate this.
