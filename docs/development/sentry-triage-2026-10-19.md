# Sentry open-issue review — 19 October 2026

Used the authenticated `sentry` CLI to query `craftsky/` with `is:unresolved lastSeen:>=2026-01-01`, limit 100. The fresh final query returned 13 issues with `hasMore: false`. Inspected the latest events for the eight new issues, recurrence dates for the five older issues, all three APP-VIEW-D event summaries, correlated logs, and current source. Counts below are a snapshot.

## Fixes in this checkout

- [FLUTTER-APP-18](https://craftsky.sentry.io/issues/154287764/) (one event, 17 October): `UserComments.build` resumes after disposal and reads optimistic post overlays, even for an empty page. Capture account authority before dispatch and require it before overlay reads. Apply the same check to pagination.
- [FLUTTER-APP-19](https://craftsky.sentry.io/issues/154295332/) (one event, 17 October): `UserProjects.build` reads post overlays after disposal. Guard initial loading, pagination, and invalid-cursor restart, including before dispatching the restart.
- [FLUTTER-APP-1A](https://craftsky.sentry.io/issues/154443752/) (one event, 18 October): `PostCommentSection.build` reads like overlays after disposal. Guard the completion. Also guard the existing muted-parent reveal continuation before it reads the repository and dispatches a second request.
- The same missing completion guard exists in single-post, project-browse and authored-repost readers. Guard these callers too, including browse/repost pagination and repost cursor restart. Reuse the established `ActiveAccountOperation` and `ApiCanceled` behavior; overlay helpers continue to require a live ref.
- [APP-VIEW-E](https://craftsky.sentry.io/issues/154419198/) (one event, 18 October): a trusted registration `canceled` result is captured as unexpected. The real callback service selects this code for `access_denied`. Mark only this typed cancellation as `result=expected` at the auth diagnostic boundary. Preserve the original error chain and callback page; provider-unavailable and incomplete-registration failures remain actionable.

## Open issues requiring more evidence

- [FLUTTER-APP-1B](https://craftsky.sentry.io/issues/154597593/) (one event) and [FLUTTER-APP-1C](https://craftsky.sentry.io/issues/154597596/) (two events): Instagram account GETs returned HTTP 502 on 19 October, in app 1.7.2+9. The client retained the Dio/API causes but no AppView request ID or stable server error code. Trace-correlated logs contain only the client provider failures. Do not infer a PDS, database, or client defect from the status alone.
- [FLUTTER-APP-1D](https://craftsky.sentry.io/issues/154597620/) (one event): authored-project loading returned HTTP 502 in the same client session. It has the same missing server evidence. Its timing alone does not establish a shared cause with Instagram loading.
- [APP-VIEW-D](https://craftsky.sentry.io/issues/154418202/) (three events, latest 18 October): login start failed at `authorization_server` in AppView 1.0.16. The latest retained exception chain contains generic string/wrapper errors without a specific provider reason or useful stack. Its trace has no retained logs. No supported behavioral fix can be established from these events.

The three 502 groups need server request/log evidence from their request times or a fresh reproduction with the current client/server. APP-VIEW-D needs the authorization-server discovery/PAR cause. No production configuration change or speculative retry/suppression was made. Recent backend ERROR log summaries were also queried; the CLI's detail lookup returned `Log '…' not found in craftsky/app-view` for a listed ID, including a fresh retry, so those summaries do not establish additional causes.

## Older issues with existing remedies

The following issues each still have one event and no newer recurrence. Checked the current source against the [16 October review](sentry-triage-2026-10-16.md):

- FLUTTER-APP-Z, last seen 13 October: fonts are already bundled instead of downloaded at runtime.
- FLUTTER-APP-Y, last seen 13 October: active identity already checks ref lifetime after awaited work.
- FLUTTER-APP-10, last seen 15 October: invalid handles are already classified as expected validation.
- FLUTTER-APP-X, last seen 13 October: subscription lease expiry already uses typed account cancellation with a lifetime guard.
- FLUTTER-APP-W, last seen 13 October: timeline already checks captured authority before applying block/post/interaction overlays.

These findings do not verify which fixes are installed on any particular device or prove production recovery.

## Validation and delivery boundary

- Six new lifecycle tests hold repository requests pending, dispose their provider, then complete the requests. All six fail against the original source with `UnmountedRefException`; all pass with the fixes restored.
- Final focused Flutter provider suites: 47 passed. Includes the new lifecycle cases and existing single-post, thread, author-post, repost, timeline, project-feed, project-list and project-provider tests.
- `flutter analyze --no-pub`: no issues found.
- `go test -race ./internal/auth ./internal/observability -count=1`: passed. PostgreSQL-dependent tests use their existing skip behavior; this is not the complete AppView release gate.
- The new actual HTTP/diagnostic/Sentry-boundary regression checks cancellation suppression, continued capture of both actionable registration failure codes, unchanged callback pages, retained typed diagnostics/correlation, and protected-value canaries in serialized local/Sentry output. It was rerun separately after adding a private provider-error canary.
- `git diff --check`: passed.

Changes remain local and uncommitted. No deployment, production mutation, Sentry resolution/archive, device run, or production recovery verification was performed. Raw event/log downloads and test transcripts are under `/tmp/craftsky-sentry-*`, outside the repository.


## Follow-up: evidence for recurrences

The four undiagnosed groups above now have targeted diagnostic coverage in this checkout:

- Login start wraps each dependency boundary in a typed diagnostic without changing the original cause or authorization behavior. Both local and Sentry output retain the actual identity/discovery/metadata/authorization/persistence stage. Reviewed OAuth provider error names and numeric upstream statuses survive the existing privacy filters. Tests exercise the real coordinator, pinned Indigo client, HTTP handler and serialized sinks with local fake provider responses.
- Instagram account reads previously reduced unexpected store failures to a cause-free log. They now report the original typed cause through the request diagnostic owner. Database failures retain SQLSTATE and request correlation. Error response status and envelope remain unchanged.
- Project reads already retained original causes for most stages. Relationship failures now identify `project.author.list` and `relationship` instead of a shared handler name. Regressions verify both database failures and the HTTP 502 identity-resolution path retain one issue, their stage and the request ID.
- Every AppView response passing through Logging gains the generated UUID as `X-Request-ID`; allowed web origins can read it through CORS. Flutter uses a single validated UUIDv4 header only when the envelope has no valid request ID, preserving envelope precedence. It also records a closed response-shape label (`appViewError`, `json`, `nonJson`, `empty`). Tests verify these fields reach serialized Sentry events while response bodies, cookies, malformed/multiple IDs and arbitrary response-shape values stay excluded.

These changes require an AppView deployment and an updated Flutter build before they improve live evidence. A proxy failure that never reaches AppView cannot contain an AppView cause or request ID; the client will preserve its HTTP status and response-shape evidence without inventing server correlation. No production fault was injected and no production telemetry gates were changed.


Follow-up validation completed: shared Flutter API/error/observability suites passed 168 tests with one existing skip; `flutter analyze --no-pub` reported no issues. The complete `GOFLAGS='-p=1' ./scripts/appview-check` gate passed with 6,223 integration cases and 6,223 race cases, zero test skips, plus module/codegen/vet/Staticcheck, source/binary vulnerability checks, Docker builds, migration-failure checks, media-memory checks and service smoke checks. The real login regression also captures Indigo's process logger through the production diagnostic filter, verifying provider descriptions remain excluded. The fixture resides in the existing approved auth diagnostic test file; the Sentry import policy is unchanged. Transcript: `/tmp/craftsky-recurrence-gate-final.log`; artifacts: `/tmp/craftsky-recurrence-gate-final/`. Pinned Tap handshake passed; deterministic event injection remains unavailable in Tap v0.1.10. `git diff --check` passed. No device run, deployment or live recurrence delivery is implied by these local checks.
