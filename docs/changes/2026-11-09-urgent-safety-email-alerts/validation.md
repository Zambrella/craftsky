# Urgent safety email alerts

## Authorized behavior

Douglas Todd selected the Brevo API with both sender and recipient
`moderation@craftsky.social` during the launch-blocker review on 9 November 2026.
The mailbox is intended to alert him away from the office. Email contains a
static prompt and the authenticated console URL, without report content,
identities, categories, case references, attachments or restricted evidence.

## Implementation

- New private delivery/source tables in migration 000121 persist discovery,
  attempts, leases, acceptance and source deduplication across restarts.
- Poll committed urgent signed-in/guest cases, active scanner incidents,
  priority/urgent external intake and urgent safety workflows. No moderation
  action or API route was added. No public schema or Lexicon changed.
- Batch up to 500 fresh sources; limit successful digest delivery to one per
  minute. Retry failures with bounded exponential delay, reclaim expired leases,
  and reuse the opaque delivery UUID for Brevo idempotency.
- Sender uses the fixed Brevo HTTPS endpoint with a ten-second timeout and
  refuses redirects. Only the documented duplicate code is selected from error
  bodies; raw response prose is discarded.
- Server-only `BREVO_API_KEY` uses the existing redacted Secret wrapper.
  `SAFETY_EMAIL_ENABLED` defaults off. Enabling requires the console, a key and
  HTTPS OAuth public origin. The worker participates in normal app shutdown.
- Orphaned source references are removed on polling after source deletion;
  unreferenced completed delivery metadata is removed after 30 days.
- Operational setup, coverage and known limitations are documented in
  [moderation operators](../../operations/moderation-operators.md#urgent-safety-email-alerts-brevo).

## Local Verification

Dedicated PostgreSQL 16 container; production migrations applied unchanged into
isolated test schemas. Provider calls use a local HTTP test server or fake sender.
No live Brevo request, production write or deployment was performed.

- Focused provider and queue tests: pass, including PostgreSQL migration
  round-trip, urgent guest report on an existing routine case, persisted retries,
  stable delivery UUID, duplicate suppression, lease/crash recovery, detected
  incidents, urgent workflows, rate limiting and orphan cleanup.
- Serialized worker diagnostics: retain operation, stage and retry outcome;
  exclude report/provider prose, identities, credentials and recipient canaries.
- Provider boundary: checks from/to and payload allowlist, credential header,
  idempotency key, acceptance/duplicate/error responses and redirect rejection.
- Configuration checks: disabled by default, reject incomplete configuration,
  accept complete production configuration, protect secret formatting.
- Go vet: passed for safetyalerts, internal/app and cmd/appview.
- Final `go test -race ./internal/safetyalerts -count=1`: passed, including all
  16 urgent allegation names through production-schema persistence and discovery.
- `go test ./internal/app ./cmd/appview -count=1`: passed after moving email
  construction into the dedicated dependency helper required by the repository
  architecture check.
- Python online-safety readiness tests: 7 passed. GOV-02 remains cleared by the
  readiness evaluator; overall launch remains blocked by the other controls.
- Full `go test -p 2 ./...`: every package except internal/app passed. Its
  dependency-construction architecture check identified the direct constructors;
  that issue was corrected and the complete internal/app plus cmd/appview rerun
  passed as recorded above. The full multi-package command was not repeated.
- `git diff --check`: passed. No Flutter or public API behavior changed; no
  device verification was required for this server-only addition.

## Outstanding Production Evidence

1. Confirm the Brevo sender/domain and transactional account are enabled.
2. Configure the server secret and enable flag; deploy the exact reviewed tag.
3. Verify one authorized benign urgent report reaches the inbox and triggers the
   intended phone notification; verify console sign-in and resolve the test item.
4. Record actual production queue health and the monitoring/backup arrangement.

Brevo acceptance is not evidence of inbox/phone receipt. Its documented
idempotency window is 30 minutes; ambiguous delivery outside that window may
produce a duplicate. Provider failure can delay later digests. These alerts do
not provide staffed 24-hour coverage or backup for sleep, illness or absence.
