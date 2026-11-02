# Optional browser diagnostics

Set the public build-time `PUBLIC_SENTRY_DSN` to configure a Sentry web project.
The control is hidden when the DSN is absent. No Sentry project, DSN, server gate,
retention rule or deployment is created or changed by this implementation.

`DiagnosticsHost.svelte` loads the official pinned `@sentry/browser` SDK only
after the device-wide `craftsky.diagnostics-consent` preference is explicitly
enabled. Settings and onboarding use `setConsent()` to update the host immediately.
Revoking consent clears the active SDK client and closes it. Event and transport
gates independently check current consent, so queued unsent envelopes are dropped
after revocation. A request already sent before revocation cannot be recalled.

Only GlobalHandlers, bounded LinkedErrors, and Dedupe integrations are installed.
SDK 11 uses `dataCollection` instead of the removed `sendDefaultPii` option; every
private collection category is disabled. There are no breadcrumbs, fetch/XHR
capture, replay, session tracking, console capture or performance integrations.

The final event selector reconstructs only native error type, a static generic
message and code-frame coordinates. Published bundle filenames and development
source paths are retained without origin/query/fragment. Other filenames are
redacted. No function names, source context, frame locals, request or viewer
information, raw exception prose, trace context or custom fields survive. The
transport repeats selection and removes all non-event items, including scope
attachments added after the event hook. Expected API, cancellation and common
offline failures are suppressed.

Tests capture actual SDK-serialized envelopes using a local fake fetch. Synthetic
protected-value canaries cover cause messages, query capabilities, local paths,
users, cookies, request bodies, breadcrumbs, extras, tags, contexts and attachments,
while asserting native cause types and stack coordinates survive. Lifecycle tests
cover off-by-default behavior, revocation and re-enabling.

This verifies local serialization and lifecycle. Sentry ingestion, release source
maps and production symbolication require a configured deployment and separate
verification; no synthetic failures are sent to production.
