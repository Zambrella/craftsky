# TDD Implementation Plan: Instagram DM Ownership Verification And Automatic Following

## Inputs

- Requirements: `01-requirements.md`
- Acceptance tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes
- Coding plan: `04-coding-plan.md`
- Original design: `design-plan.md`; superseded where `01`/`02` are more exact
- Automatic-follow revision approval: explicitly confirmed by the user on
  2026-07-27 after approving requirements, acceptance tests, document review,
  and the rewritten coding plan

## Implementation Rules

- Do not implement behavior without a linked requirement and acceptance test.
- Add or change one focused test, observe its meaningful failure, implement the
  minimum behavior, rerun green, and refactor only while green.
- Keep parallel work in disjoint files and independently green packages.
- Use only wholly synthetic/redacted Instagram inputs in source and tests.
- Never contact Meta from automated tests; use fakes or `httptest.Server`.
- Keep private graph data in AppView Postgres. A verified member's import is the
  approved informed authorization for exact eligible automatic ordinary
  follows; no per-match acceptance surface remains.
- Require current membership and the complete fail-closed eligibility policy at
  every boundary enumerated by the requirements.
- Keep fixed-account Flutter operations fenced by `ActiveAccountLease` after
  every await.
- Do not edit lexicons, commit, push, or enable a production integration.
- Keep red/green commands and meaningful evidence current in the execution log.

## Ordered Slices

| Step | Test IDs | Requirements / criteria | Initial expectation |
|---:|---|---|---|
| 1 | UT-001 | FR-002; AC-003, AC-004 | Challenge package absent; focused Go test fails to compile |
| 2 | UT-008, UT-016, IT-013 | FR-001, FR-003, FR-027; AC-001, AC-002, AC-040 | Config/limit/integration wiring absent |
| 3 | IT-001, UT-002 | FR-004–FR-010, FR-012, FR-015; AC-005–AC-010 | Schema and state machines absent |
| 4 | IT-020 | FR-030; AC-048 | Shared current-member boundary absent |
| 5 | IT-021, TD-011 | FR-011–FR-026; AC-011–AC-038 | Shared wire corpus absent |
| 6 | IT-002, UT-003, UT-004, IT-003 | FR-004–FR-009; AC-005–AC-014 | Verification routes/webhook absent |
| 7 | UT-007, IT-004 | FR-003, FR-007–FR-010; AC-013–AC-016 | Durable worker/Meta adapter absent |
| 8 | UT-006, IT-005, IT-006 | FR-008–FR-011, FR-014, FR-030; AC-017–AC-022 | Link/conflict/eligibility services absent |
| 9 | UT-005, IT-007, IT-008 | FR-012–FR-018; AC-023–AC-029 | Import/match/reconciliation absent |
| 10 | IT-009 | FR-017, FR-018; AC-030, AC-031 | Stable-rkey follow acceptance absent |
| 11 | IT-010 | FR-028; AC-041, AC-042 | Retention/export primitives absent |
| 12 | UT-013, UT-014, IT-011, IT-012 | FR-019–FR-022; AC-032–AC-038 | Social/system union and coalescing absent |
| 13 | IT-018, IT-019 | FR-028, FR-029; AC-043, AC-044 | Operator and worker-control paths absent |
| 14 | UT-009, UT-010, IT-014 | FR-023, FR-024; AC-039–AC-043 | Flutter parser/API/repository absent |
| 15 | UT-011, IT-015, IT-016 | FR-023–FR-026, FR-030; AC-044–AC-047 | Flutter providers/routes/page absent |
| 16 | UT-012, IT-017 | FR-019, FR-025; AC-036–AC-038 | Actorless Flutter notification absent |
| 17 | UT-015, REG-001–REG-012, TD-001–TD-012 | NFR/security/privacy requirements; AC-039–AC-049 | Full verification and privacy review pending |
| 18 | IT-002, IT-022, REG-012 | FR-003, FR-024, NFR-008; AC-042, AC-049 | Owner-current lookup and secure resumable display snapshot absent |

The exact requirement/test matrix in `02-acceptance-tests.md` remains
authoritative if this condensed table omits a secondary linkage.

## Implementation Steps And Evidence

### Step 1 — Challenge contract

- Add `appview/internal/instagram/challenge_test.go` first.
- Prove the 30-symbol alphabet, 13 random symbols, canonical grouping, exact
  whole-message grammar, outer whitespace/ASCII case normalization only,
  injected entropy errors, HMAC digest/equality, and redacted string behavior.
- Red command: `cd appview && go test ./internal/instagram -run TestChallenge -count=1`
- Implement only `challenge.go` and the minimum supporting types.

### Step 2 — Configuration and shared limiting

- Add disabled/local/full/partial-production config tests before fields/parsing.
- Add a transactional Postgres limiter test before persistence implementation.
- Prove trusted peer versus forwarded IP behavior before accepting that header.
- Wire disabled/fake modes without any Meta provider construction/call.

### Step 3 — Schema and state machines

- Add migration inspection/round-trip tests before migrations `000023`/`000024`.
- Assert checks, uniqueness, indexes, absence of membership cascades, sensitive
  fields, support-source multiplicity, deterministic follow operations, and
  system/social union migration.
- Add pure transition table tests before state helpers.

### Step 4 — Current membership

- Add route-policy/middleware and worker-transition failures first.
- Prove reversible membership inactivation/rejoin separately from terminal
  identity purge; reactivation is always explicit and does not extend consent.

### Step 5 — Shared wire corpus

- Add wholly synthetic fixtures and Go validation tests first.
- Include all states, success/error envelopes, cursors, privacy-preserving
  DELETE behavior, conflicts/unavailability, and type-discriminated actorless
  notifications.
- Flutter tests later consume the exact same files.

### Steps 6–13 — AppView vertical behavior

For each route/service slice, start at the narrowest domain/store test, then the
real handler/mux, then nearby regression packages. Provider calls remain behind
fakes. Store transitions around provider calls are separate transactions with
revalidation. Update this runbook with each meaningful red/green result.

### Steps 14–16 — Flutter behavior

Start with the pure parser and wire models, then fixed-account repository tests,
then controllers, page/widgets, and finally notification variant/open flow.
Regenerate Riverpod/go_router/localization output only after hand-written tests
and source are green. Every asynchronous test covers account switch or
switch-away/back fencing where relevant.

### Step 17 — Review and verification

- Run focused Go/Flutter tests during each slice.
- Run migration up/down and real-Postgres integration tests.
- Run `go test ./...`, focused `-race`, `go vet ./...`, full Flutter tests,
  formatting, generated-code drift, analyzer, and `git diff --check`.
- Compare analyzer output with the recorded one-info baseline.
- Use the implementation-review skill, remediate actionable findings test-first,
  re-review, and rerun all affected/full gates.
- Record manual/live Meta, export-shape, safety-adapter, device, accessibility,
  and production privacy checks as pending release gates rather than passing.

## Execution Log

| Step | Status | Red evidence | Green evidence | Notes |
|---:|---|---|---|---|
| 1 | Complete | Package symbols were absent (`NewChallengeCodec`, canonicalizer, digest types) | `go test ./internal/instagram -run TestChallenge -count=1` passed | 10,000 deterministic values; rejection sampling; keyed storage-safe digest |
| 2 | Complete | Instagram config and persistent limiter symbols were absent; a pointer-format test then exposed key bytes | Config/race/vet green; shared Postgres limiter exact-boundary and 100-way race tests green | Trusted request-IP and fail-closed dependency wiring complete |
| 3 | Complete | Migration files and closed state symbols were absent | Real-Postgres `000023`/`000024` schema tests and pure transition matrices passed | Current username ownership is now durably unique |
| 4 | Complete | Current-member middleware/store symbols were absent | Middleware, real membership-store, verification mux, worker inactivation, and lifecycle tests passed | All member-facing routes and workers use the shared boundary |
| 5 | Complete | Shared Go/Dart corpus consumers were absent | Go and Dart consume the same wholly synthetic wire corpus | No user-derived fixture data |
| 6 | Complete | Verification domain/store/handler/route symbols were absent; conflict SQL initially had an ambiguous timestamp parameter | Real-Postgres create/supersede/redeem/confirm/replay/conflict/username-refresh tests plus API/mux wire tests passed | Meta remains disabled without complete configuration |
| 7 | Complete | Durable webhook work and bounded retry behavior were absent | Signed ingress, dedupe, lease recovery, retry, membership, and configured-limit tests passed | Provider calls use fakes or `httptest` only |
| 8 | Complete | Link/conflict policy services were absent | Unique identity claims, fail-closed policy, collision, lifecycle, and restoration-hook tests passed | Production relationship-safety adapter remains an external gate |
| 9 | Complete | Import/matcher/reconciliation services were absent | Exact normalized matching, additive support, retention, future-match, invalidation, and notification-boundary tests passed | Reconciliation is targeted and durable |
| 10 | Complete | Shared explicit-follow writer was absent | Ordinary create and deterministic Instagram `PutRecord` share one service; failure/replay/already-following tests pass | No automatic follow behavior |
| 11 | Complete | Retention/export services were absent | Real-Postgres expiry, export, purge, and batch-bound tests passed | Batch remains at most 500 |
| 12 | Complete | Preference scope, actorless feed scans, post-union social activation, partial-retraction newness, and post-lease preference races failed before implementation | Notification/API/push real-Postgres suites cover coalescing, retraction, delivery, feed, open, and newness | Every declared eligibility stage has a production caller |
| 13 | Complete | Operator commands and redacted result types were absent | Conflict/link/job/retention CLI and operator tests passed | Output uses opaque identifiers only |
| 14 | Complete | Flutter parser/API/repository layers were absent | Parser, JSON-only import, wire, repository, and error-envelope tests passed | ZIP remains intentionally unsupported |
| 15 | Complete | Flutter controllers/routes/page were absent | Provider and widget tests cover fixed-account fencing, verification, import, and suggestions | Active-account lease checks remain after awaits |
| 16 | Complete | Actorless notification model/open behavior was absent | Notification model, row, payload, routing, and Instagram destination tests passed | Existing social variants remain intact |
| 17 | Complete | Cross-stack verification was pending | Full race-enabled Go and all 965 Flutter tests pass; analyzer, format, canary, migration, and diff checks are clean for this change | Implementation verification is complete; formal re-review remains the workflow exit choice; live Meta/export/device/edge checks remain external gates |
| D1 | Complete | A live signed Meta delivery returned `200` but produced no durable webhook work, and the attempt remained `pendingDm`; no reducer-boundary diagnostics existed | `go test ./internal/app ./internal/integrations/instagrammeta` passes; rebuilt AppView confirms the dev flag is enabled | User-authorized temporary capability-spike exception: `INSTAGRAM_UNSAFE_LOG_WEBHOOK_BODIES=true` logs raw signed webhook bodies only in dev and is forcibly disabled in prod; Meta tokens/secrets remain excluded |
| D2 | Complete | Store/API/client symbols for current-attempt reads were absent; the resume test then showed the challenge was not cached; confirmed sign-out did not clear private session state | `go test ./...`, focused Go race, `go vet ./...`, and all 964 Flutter tests pass; focused Dart formatting and `git diff --check` pass; analyzer reports only 13 pre-existing diagnostics outside this slice | Owner-scoped current lookup, DID-scoped secure display snapshot, AppView reconciliation, polling/confirmation resumption, terminal cleanup, and session cleanup are complete; ordinary page disposal/account switching retains the bounded snapshot |
| D3 | Complete | The candidate confirmation page placed static discovery copy before an unselected selector | Focused widget test failed on selector order before implementation, then passed with selector state/copy assertions | The selector now follows the account, defaults to discovery allowed, updates the explanation for both choices, and confirms or cancels through existing state operations |

## Implementation Review Remediation (2026-07-21)

The user explicitly approved addressing the `06-implementation-review.md`
findings before the requested UI polish. Remediation keeps the original
requirements and acceptance tests authoritative and proceeds test-first in this
order:

| Step | Finding | Test IDs | Requirements | Initial expectation |
|---:|---|---|---|---|
| R1 | IR-001 | IT-001, IT-005 | FR-009 | Different IGSIDs can currently claim the same normalized username |
| R2 | IR-002 | IT-010 | FR-028, NFR-003 | A superseded link currently retains plaintext username fields |
| R3 | IR-003 | IT-011, IT-012 | FR-010, FR-018, FR-020, FR-028 | Dependent invalidation is duplicated and incomplete |
| R4 | IR-004 | IT-011, IT-012 | FR-015, FR-020, FR-022 | Delivery, feed, and open do not all re-evaluate eligibility |
| R5 | IR-005 | IT-020 | FR-028, FR-030 | Worker-observed membership loss only rejects the attempt |
| R6 | IR-006 | IT-013 | NFR-002, NFR-004 | Private writes can bypass a missing persistent limiter |
| R7 | IR-007 | IT-006, IT-011 | FR-011, FR-019 | Username refresh and safety-restoration enqueue seams are absent |
| R8 | IR-008 | IT-009 | FR-017 | Ordinary and Instagram follows use separate PDS write services |
| R9 | IR-009 | UT-007, UT-016 | NFR-002 | Validated runtime limit settings are not carried to workers/lifecycle |
| R10 | IR-010 | UT-015 | NFR-003 | Controlled privacy canaries do not cover every prohibited surface |
| R11 | IR-011 | REG-001–REG-011 | NFR-005 | Static gates and this execution log are stale |

For each row: add one focused regression, observe a meaningful failure, make
the minimum implementation change, rerun the focused and neighboring suites,
then record the red/green evidence here. UI polish begins only after R1–R11 are
green.

### Remediation Evidence

| Finding | Status | Green evidence |
|---|---|---|
| IR-001 | Complete | Concurrent different-IGSID/same-username confirmations leave one authoritative current link and one private conflict |
| IR-002 | Complete | Superseded tombstones contain no plaintext IGSID or username immediately after confirmation |
| IR-003 | Complete | Account/import/conflict/supersession transitions invalidate accepting work, follow operations, notification support, leased pushes, and targeted jobs transactionally |
| IR-004 | Complete | Delivery, feed, and open/list callers exercise the shared policy; stale support retracts and unavailable safety data fails closed |
| IR-005 | Complete | Worker-observed departed membership invokes full private-data inactivation before terminal work handling |
| IR-006 | Complete | Missing persistent limiter returns stable `instagram_unavailable` for private writes while local reads/privacy deletes remain available |
| IR-007 | Complete | Same-IGSID DM re-verification refreshes usernames safely; a narrow fake-backed restoration enqueue interface is wired |
| IR-008 | Complete | Ordinary and deterministic Instagram follows use `internal/followwrite.Service` |
| IR-009 | Complete | Tightened lease/retry/processing-age and notification window/count settings alter runtime behavior within fixed maxima |
| IR-010 | Complete | Synthetic Go/Dart canaries cover diagnostics, Sentry/logs/metrics, push, PDS records, URLs, errors, and stringification |
| IR-011 | Complete | Go/Dart formatting, `go test ./...`, changed-package race tests, all 965 Flutter tests, focused Instagram tests, and `git diff --check` pass; `flutter analyze` reports only 13 pre-existing info-level findings outside this change |

## Following-Only Import Simplification (2026-07-22)

The user approved removing follower imports rather than retaining unused
private facts. The updated requirements and acceptance tests make imports
following-only across Flutter, the wire contract, AppView, and Postgres.

| Step | Test IDs | Requirements | Expected initial failure | Status |
|---:|---|---|---|---|
| F1 | UT-005, IT-014 | FR-012, FR-014, RULE-006 | Old `{username, direction}` requests are still accepted and models expose direction/follower counts | Complete |
| F2 | IT-001, IT-007 | FR-012, FR-018, RULE-006 | Postgres still stores `direction` and `follower_count` | Complete |
| F3 | UT-009, UT-010 | FR-013, RULE-006, RULE-007 | Flutter parser requires a selected direction and can emit follower entries | Complete |
| F4 | IT-016 | FR-025, NFR-007 | Import UI still offers Accounts that follow me and renders follower counts | Complete |
| F5 | IT-021, REG-011 | FR-012–FR-015, NFR-004 | Shared fixtures and cross-stack contract still contain direction/follower fields | Complete |

Each step follows red-green-refactor with the smallest focused Go or Flutter
test first. The schema change is append-only migration `000025` so existing
development databases and fresh stacks converge on the same following-only
shape.

Red-green evidence:

- F1 red: the legacy-direction API case returned `201`; after removing the
  direction field from the domain and strict wire model, it returns
  `400 invalid_request`. A follower-specific entry field is rejected too.
- F2 red: the migration test failed because `000025` did not exist; it now
  verifies that upgraded and fresh schemas have neither `direction` nor
  `follower_count`, and that legacy follower rows are discarded.
- F3 red: the following-only parser/request tests did not compile while
  direction remained required; all ten focused parser/request tests pass
  after reducing entries to `{username}` and ignoring sibling follower data.
- F4/F5 green: the page test confirms that no follower option is rendered and
  only normalized following usernames are uploaded; Go and Dart consume the
  same following-only corpus successfully.

## Verified-Link Lifetime And Direct Import Simplification (2026-07-23)

The user approved replacing optional 12-month retention with one simple
lifetime: imports are available only after verification and persist until
per-import deletion or Instagram unlink. Discoverability remains selectable
but defaults to public. Future-match in-app notifications are always created;
push delivery remains controlled from the linked Notification Settings page.

| Step | Test IDs | Requirements | Expected initial failure | Status |
|---:|---|---|---|---|
| S1 | IT-001, IT-007, IT-010 | FR-012, FR-018, FR-028 | Wire/storage still contain consent and expiry fields; unverified imports are accepted; unlink retains owner imports | Complete |
| S2 | IT-005, IT-016 | FR-008, FR-024, FR-025 | Import cards render without a verified account; discoverability default needs an explicit regression | Complete |
| S3 | IT-014, IT-016, IT-021 | FR-012, FR-013, FR-025 | Flutter still previews normalized handles and sends retention fields | Complete |
| S4 | IT-011, IT-012, IT-017 | FR-019–FR-022, FR-025 | The UI has no Notification Settings link and disabled push behavior needs explicit future-match coverage | Complete |
| S5 | REG-001–REG-012 | NFR/security/privacy requirements | Broad verification is stale after the contract change | Complete |

Red-green and verification evidence:

- S1 red: the migration shape test failed before `000026`; the new migration
  removes consent/expiry columns and the expired import state. Create now
  requires an active verified link. Unlink deletes owner imports/handles and
  invalidates only unfinished suggestions that lose all support.
- S2/S3 red: the page test previously expected import controls without a link
  and a normalized preview. It now verifies the basic verification prompt,
  absence of all import/suggestion cards, public discovery default, and one-step
  normalized manual import. Go and Dart share the reduced `{sourceType,
  entries}` wire corpus.
- S4: future reconciliation creates the private actorless in-app digest even
  with Instagram-match push disabled. Flutter explains the behavior and links
  directly to Notification Settings.
- S5: `just test` passes the full race-enabled Go suite against real Postgres;
  all 965 Flutter tests pass; focused Instagram tests pass; `git diff --check`
  passes. `flutter analyze` reports only the existing 13 info-level findings.

## Instagram ZIP Export Extension (2026-07-23)

The user approved mobile-only support for selecting either a standalone
`following.json` file or the original Instagram ZIP. Both containers remain
the same local evidence source and cross the existing repository/AppView
boundary only as `sourceType: instagramJson` plus normalized usernames. The
additional document-review stage was explicitly skipped; the updated
requirements, acceptance tests, and coding plan are authoritative.

Implementation rules specific to this extension:

- Do not implement ZIP behavior without `FR-031`, `NFR-010`, and `RULE-011`.
- Pass a native file path, never complete archive bytes, into one background
  isolate on iOS and Android.
- Keep the archive file-backed, inspect bounded central-directory metadata,
  decode only the exact canonical following entry, never extract to disk, and
  close resources on every success/failure path.
- Never copy the approved real Instagram archive into the repository. Generate
  only wholly synthetic temporary ZIPs in automated tests.
- Preserve the existing AppView API, storage, and `instagramJson` source value.

### Test Order

| Step | Test ID | Requirement IDs | Acceptance criteria | Expected initial state |
|---:|---|---|---|---|
| Z1 | UT-017 | FR-013, FR-014, RULE-005, RULE-006 | AC-016–AC-020, AC-051 | The parser accepts only legacy direct values and cannot parse the observed exact `_u/<username>` URL plus agreeing-title record |
| Z2 | UT-018 | FR-013, FR-031, NFR-010, RULE-007 | AC-016–AC-018, AC-052, AC-053 | No file-backed ZIP/isolate parser exists and `archive` is only transitive |
| Z3 | IT-023 | BR-002, FR-013, FR-025, FR-031, NFR-008, NFR-010, RULE-007, RULE-011 | AC-016–AC-019, AC-042, AC-050–AC-054 | The picker/page reads selected JSON bytes on the UI isolate and rejects ZIP |
| Z4 | REG-013 | BR-002, FR-012–FR-014, RULE-007, RULE-011 | AC-016–AC-019, AC-050, AC-054 | Broad JSON/AppView compatibility has not been rerun after adding ZIP |

### Step Z1 — UT-017

- Write failing test: extend
  `app/test/instagram_migration/services/instagram_import_parser_test.dart`
  with exact current-shape successes and title-only, mismatch, fuzzy-host/path,
  ambiguous-list, encoded-separator, and invalid-username rejections.
- Run command:
  `cd app && flutter test test/instagram_migration/services/instagram_import_parser_test.dart`
- Confirmed failure: the observed URL/title record produced no entries because
  the existing parser read only `string_list_data[].value`. The expanded
  canonicality cases then exposed that Dart normalizes an explicit default
  `:443` port unless the original authority/URL is checked.
- Implement: preserve legacy direct values; accept one exact HTTPS
  `www.instagram.com/_u/<username>` URL only when its record title normalizes
  to the same valid username.
- Refactor: only after the focused parser test is green.

### Step Z2 — UT-018

- Write failing test: create
  `app/test/instagram_migration/services/instagram_export_file_parser_test.dart`
  using synthetic temporary JSON/ZIP files.
- Run command:
  `cd app && flutter test test/instagram_migration/services/instagram_export_file_parser_test.dart`
- Confirmed failure: the new focused test failed to compile because the
  file-path parser and isolate boundary did not exist.
- Implement: add the direct `archive` dependency and native isolate parser with
  bounded EOCD/ZIP64 preflight, exact target selection, target-only streaming
  output, size/CRC verification, typed safe failures, and deterministic close.
- Refactor: only after focused JSON/ZIP boundary tests are green.

### Step Z3 — IT-023

- Write failing test: add the native selection/privacy/account-switch cases in
  `instagram_import_privacy_test.dart` and page-level copy/cancellation cases in
  `instagram_migration_page_test.dart`.
- Run command:
  `cd app && flutter test test/instagram_migration/instagram_import_privacy_test.dart test/instagram_migration/instagram_migration_page_test.dart`
- Confirmed failure: the page test failed to compile because the normalized
  Instagram-export picker/provider did not exist; adding archive errors also
  made the localized error switch intentionally non-exhaustive until the UI
  mapping was implemented.
- Implement: replace the JSON-byte picker boundary with an Instagram-export
  picker returning only a normalized parse result; preserve account-lease
  fencing and the existing repository request.
- Refactor: only after focused integration/widget tests are green.

### Step Z4 — REG-013

- Run existing parser, request-model, golden-wire, repository, AppView
  API/store, privacy, and page tests.
- Assert no `instagramZip` source, raw path/archive field, AppView branch,
  migration, or committed user-derived fixture exists.
- Finish with Dart formatting, analyzer, full Flutter tests, relevant Go tests,
  `git diff --check`, and manual `MAN-005` status.

### ZIP Extension Execution Log

| Step | Status | Red evidence | Green evidence | Notes |
|---:|---|---|---|---|
| Z1 | Complete | Exact URL/title input returned no entries; explicit `:443` was initially accepted after the first implementation | `flutter test test/instagram_migration/services/instagram_import_parser_test.dart` passes 11 tests | Legacy direct values remain supported; exact URL/title evidence is normalized and ambiguous/noncanonical variants are ignored |
| Z2 | Complete | The focused test failed to compile because `InstagramExportFileParser` did not exist; a later regression showed a mismatched unsupported local-header method was accepted from central-directory metadata alone | Both focused parser suites pass; focused Dart analysis reports no errors | Direct JSON, stored/deflated ZIP, exact target, duplicate/missing, encrypted/unsupported, local/central disagreement, truncation/CRC, ZIP64, 20 MiB declared/actual, archive metadata, and 10,000-entry limits are covered |
| Z3 | Complete | Page contract failed to compile without the normalized export picker/provider and new localized archive errors | `flutter test test/instagram_migration/instagram_import_privacy_test.dart test/instagram_migration/instagram_migration_page_test.dart` passes 8 tests; focused Dart analysis reports no errors | Picker accepts JSON/ZIP, native path goes to the isolate, web keeps bounded JSON, result contains only normalized entries, and a late Alice result is discarded after switching to Bob |
| Z4 | Complete | Compatibility was stale after replacing the byte picker and adding ZIP parsing | The full Flutter and AppView test suites pass; the Instagram-focused Flutter suite, changed-file formatting, and `git diff --check` pass; analysis reports only 13 pre-existing info-level findings outside this slice | No AppView file changed and the only `instagramZip` text is a negative privacy assertion |

### ZIP Extension Completion Checklist

- [x] `UT-017`, `UT-018`, `IT-023`, and `REG-013` pass
- [x] File/archive parsing stays off the UI isolate on iOS and Android
- [x] ZIP processing is bounded independently of unrelated media size
- [x] Only the canonical following entry is decoded; nothing is extracted
- [x] JSON and ZIP submit the unchanged `instagramJson` request
- [x] No raw file path/archive bytes/non-target canary crosses the parser boundary
- [x] No real/user-derived archive or data is committed
- [x] Focused and broad verification evidence is recorded
- [x] `MAN-005` host compatibility is complete and its physical-device portion is
  explicitly retained as a release gate

Final verification on 2026-07-23:

- The full Flutter suite passes.
- `go test ./...` passes for the complete AppView module.
- Changed Instagram Dart files pass `dart format
  --output=none --set-exit-if-changed`.
- `flutter analyze` reports only 13 pre-existing info-level findings in unrelated
  auth, notifications, and observability files.
- `git diff --check` passes.
- The approved local Instagram ZIP parses successfully on the host to 88
  following entries and was not copied into the repository.
- Physical iOS/Android picker, responsiveness, and memory observation remains
  the uncompleted portion of `MAN-005`.

### Implementation Review Correction Pass (2026-07-23)

`06-implementation-review.md` requires three corrections before handoff. They
remain within the approved ZIP requirements and tests:

| Step | Finding | Test IDs | Requirements | Expected initial state | Status |
|---:|---|---|---|---|---|
| C1 | IR-012 | UT-018 | FR-031, NFR-010 | `ZipDecoder.decodeStream` runs before independent actual-header and Unix-symlink preflight | Complete |
| C2 | IR-013 | UT-018 | NFR-010 | Exact below/at/above metadata limits and dishonest declared/actual counts are not exercised | Complete |
| C3 | IR-014 | IT-023 | FR-025, FR-031, NFR-008 | Cancellation, disposal, late-error, localized-error, and native selected-path composition tests are absent | Complete |
| C4 | IR-012–IR-014 | REG-013 | FR-013, FR-031, NFR-010, RULE-011 | Broad evidence is stale after correction | Complete |

Correction rules:

- Preflight the bounded central directory from disk before invoking the package
  decoder. Count actual headers, require exactly one canonical target, and
  reject Unix-symlink metadata without reading entry contents.
- Use production hard maxima unchanged. A test-only tightened limit seam may
  exercise below/at/above behavior without creating a 64 MiB/100,000-entry
  fixture.
- Preserve file-backed isolate parsing, normalized-only results, and the
  unchanged `instagramJson` repository/AppView contract.
- Add missing lifecycle/error tests without widening feature behavior.

Correction evidence:

- C1 red: an archive with one valid target plus unrelated Unix-symlink metadata
  parsed successfully because Archive 4.0.9 read the symlink content while
  building its archive model.
- C1 green: a file-backed central-directory scan now counts actual headers,
  requires exactly one canonical target, and rejects Unix symlinks before
  `ZipDecoder.decodeStream`; all 12 focused export-parser tests pass.
- C2 red: the exact-boundary test failed to compile because no hard-max-capped
  tightened-limit parser existed.
- C2 green: the test-only parser constructor can only tighten production
  maxima. A two-entry archive now proves actual entry/directory sizes below,
  at, and above tightened limits, and a dishonest one-entry EOCD cannot hide
  the second actual header. All 13 focused export-parser tests pass.
- C3 characterization: after adding the missing test import, the existing UI
  behavior passed the new cancellation, all safe localized errors, generic
  picker failure, switch-away/back late success/error, and page-disposal cases
  without a production-code change. The privacy integration now enters through
  the native selected-`XFile` path adapter before asserting the normalized
  `instagramJson` request. The two focused files pass 11 tests.
- C4 verification: all 71 Instagram Flutter tests, the full Flutter suite, and
  `go test ./...` for AppView pass. Focused analysis reports no issues; full
  analysis reports only the same 13 unrelated pre-existing info-level findings.
  Changed-file formatting and `git diff --check` pass, no AppView file changed,
  and the approved external ZIP still parses to 88 normalized entries without
  entering the repository.
- `IR-015` was a non-blocking suggestion and was not included in the user's
  required-correction choice; BZip2 behavior remains unchanged for re-review.

## Notification Type-Only Discriminator Simplification (2026-07-26)

The user approved removing the redundant notification `kind` field after
confirming that the current database contract made it exactly derivable from
`type`. `instagramMatch` remains a first-class actorless notification with its
own sealed Flutter subtype and required system payload. Existing social types
retain their actor and AT Protocol source requirements.

Implementation rules specific to this simplification:

- `type`/database `category` is the only explicit discriminator.
- AppView notification JSON and shared fixtures contain no `kind`.
- Postgres contains no notification `kind` column; partial indexes, checks,
  newness, delivery, lifecycle, and retention predicates use `category`.
- Known social types still require actor/source facts and forbid system facts.
- `instagramMatch` still forbids actor/source/reference facts and requires the
  bounded count, group, and coalescing fields; Flutter derives navigation from
  the type without an AppView-owned route field.
- Unknown Flutter types remain inert and receive no identity-bearing
  destination.
- Because the branch merged main migrations with colliding `000023`/`000024`
  versions, renumber the unshipped Instagram migration series after main's
  migrations. The earlier divergent local migration histories cannot be
  upgraded unambiguously, so existing local development volumes must be
  recreated; there are no production users.

### Test Order

| Step | Test IDs | Requirement IDs | Acceptance criteria | Expected initial state | Status |
|---:|---|---|---|---|---|
| T1 | UT-012, IT-021 | FR-020, FR-022 | AC-035–AC-037 | Flutter and the shared corpus still require/emit `kind` | Complete |
| T2 | IT-012, REG-002 | FR-020–FR-022 | AC-034–AC-037 | AppView API/store still expose `NotificationKind` and JSON `kind` | Complete |
| T3 | IT-001, IT-011, REG-010 | FR-019, FR-020, FR-028 | AC-029–AC-031, AC-037 | Postgres and notification lifecycle still store/query `kind`; migration versions collide after the main merge | Complete |
| T4 | REG-001–REG-013 | FR-020–FR-023 | AC-034–AC-038 | Broad verification is stale after the contract and migration change | Complete |

The red-green order is Flutter/shared wire first, AppView API second, then
storage/lifecycle and migration convergence. Each focused test must fail on the
presence or expectation of `kind`, not on an unrelated setup error.

Execution evidence:

- T1 red: the exact actorless Flutter model test failed by entering the social
  decoder after `kind` was removed from its fixture. Green: notification model,
  rendering/provider, and shared-wire tests pass with `type` alone.
- T2 red: the shared Go corpus test rejected AppView's emitted `kind`. Green:
  `NotificationItem`, `NotificationRow`, hydration, and wire fixtures contain
  no kind model or field; focused API and wire tests pass.
- T3 red: the migration regression test found `ADD COLUMN kind` and the old
  kind-derived check. Green: `category` now drives all payload, grouping,
  newness, push, export, purge, and lifecycle predicates; the Instagram
  migrations are uniquely numbered `000025` through `000028`. The real
  Postgres race gate also exposed two isolated fixtures missing the
  main-branch mute/block tables; those fixtures were corrected and pass.
- T4 green: `go test ./... -count=1`, `just test`, `go vet ./...`,
  `flutter analyze --fatal-infos`, full `flutter test`, Dart formatting, Go
  formatting, and `git diff --check` pass. The initially observed Tap reconnect
  timing failure passed five focused repetitions and the subsequent full gates.

## Automatic Following Revision (2026-07-27)

The approved revision removes the member-facing suggestion review flow.
Creating an import authorizes durable AppView automatic follows for current and
future exact eligible matches. One actorful `instagramMatch` notification is
created only after each deterministic PDS follow succeeds. A manual unfollow
suppresses re-follow for the current verification lifetime.

Implementation rules specific to this revision:

- The operation owner DID is the only authority for background OAuth selection.
- Write a failing exact-owner selector test before wiring any background PDS
  call.
- Re-evaluate the shared eligibility policy immediately before every PDS write.
- Keep deterministic operation rkeys and leased idempotent completion.
- `alreadyFollowing` is terminal and creates no automatic-follow notification.
- Retain followed/already-following suppression until verification revocation.
- Remove suggestion API/model/provider/UI surfaces rather than replacing them
  with client-visible worker progress.
- Replace actorless grouped/count notifications with one actorful source-less
  event per successful operation.
- Keep provider push data identity-free and infer Notifications navigation in
  Flutter.
- Preserve verification, webhook, following-only JSON/ZIP parsing, import
  privacy, current-membership, and account-lease behavior.

### Test Order

| Step | Test ID | Requirement IDs | Acceptance criteria | Expected initial state | Status |
|---:|---|---|---|---|---|
| A1 | UT-019 | FR-032 | AC-055 | No exact-owner background OAuth selector exists | Complete |
| A2 | IT-024 | FR-017, FR-032, NFR-005 | AC-055 | No worker rotates only among the operation owner's sessions | Complete |
| A3 | UT-020 | FR-017, FR-019, RULE-012 | AC-025, AC-056 | Existing suggestion/operation transitions do not express verification-lifetime suppression | Complete |
| A4 | IT-025 | FR-017, FR-019, RULE-012 | AC-025, AC-056 | Reconciliation after manual unfollow has no explicit terminal-ledger regression | Complete |
| A5 | IT-001 | FR-017, FR-020, NFR-005 | AC-015, AC-025, AC-029 | Schema still requires actorless grouped match notifications and request-time acceptance states | Complete |
| A6 | IT-008 | FR-015, FR-016 | AC-020–AC-023 | Suggestion routes/services remain public | Complete |
| A7 | IT-009 | FR-015, FR-017, NFR-005 | AC-024, AC-025, AC-029 | No leased worker performs deterministic owner-scoped follows and idempotent completion | Complete |
| A8 | IT-010 | FR-018, FR-028, RULE-012 | AC-026–AC-028, AC-031, AC-044 | Revocation does not delete the completed suppression ledger | Complete |
| A9 | IT-011 | FR-017, FR-019, FR-020 | AC-029–AC-031 | Reconciliation creates review/digest state rather than per-target automatic follow work | Complete |
| A10 | IT-012 | FR-020–FR-022 | AC-034–AC-037 | AppView still persists/serves actorless system payloads | Complete |
| A11 | IT-021 | FR-016, FR-020, FR-022 | AC-023, AC-034–AC-037 | Shared corpus still accepts suggestions and actorless match payloads | Complete |
| A12 | IT-014 | FR-016, FR-025 | AC-023, AC-026 | Flutter repository/API still expose suggestion methods/models | Complete |
| A13 | IT-015 | FR-024–FR-026, NFR-008 | AC-023–AC-026, AC-035, AC-042 | Account-scoped controllers still load suggestion review state | Complete |
| A14 | IT-017 | FR-020–FR-023, FR-026 | AC-034–AC-038, AC-042 | Match is a system row that opens Instagram Settings | Complete |
| A15 | IT-016 | FR-016, FR-023–FR-025 | AC-009, AC-023, AC-034, AC-038 | Old terminology/default/color/suggestion/revoke layout remains | Complete |
| A16 | IT-023 | FR-025, FR-031 | AC-038, AC-050–AC-054 | Export privacy is implemented but default/input copy needs regression coverage | Complete |
| A17 | AT-001–AT-009, REG-001–REG-014 | All linked Must requirements | AC-001–AC-056 | Broad verification is stale after the revision | Complete |

### Step A1 — UT-019

- Write failing test:
  `appview/internal/auth/background_session_selector_test.go`.
- Run command:
  `cd appview && go test ./internal/auth -run TestBackgroundSessionSelector -count=1`.
- Confirmed failure: the focused package failed to compile because
  `NewBackgroundSessionSelector` and `ErrNoUsableBackgroundSession` did not
  exist.
- Implement: exact-DID active-session selection, deterministic activity order,
  retryable absence, and narrow `(DID, sessionID)` invalidation.
- Green evidence:
  `go test ./internal/auth -run TestBackgroundSessionSelector -count=1` and
  `go test ./internal/auth -count=1` pass.

### Step A2 — IT-024

- Write failing worker integration using Alice's multiple sessions plus a newer
  Bob session before adding the background worker.
- Run command:
  `cd appview && go test ./internal/instagram ./internal/auth -run 'TestAutomaticFollowWorker.*Session|TestBackgroundSessionSelector' -count=1`.
- Confirmed failure: the worker integration did not compile because automatic
  operation, worker, and options types did not exist.
- Implement only the selector/worker seam required to prove owner isolation and
  retry behavior.
- Green evidence:
  `go test ./internal/instagram ./internal/auth -run
  'TestAutomaticFollowWorker.*Session|TestBackgroundSessionSelector' -count=1`
  passes. The real selector rotates from Alice's rejected newest session to
  Alice's older valid session, deletes only the rejected composite key, never
  observes Bob's newer session, and retries without a PDS call when Alice has
  no usable session. Neighboring Instagram/auth/followwrite packages pass.

### Step A3 — UT-020

- Write failing test:
  `appview/internal/instagram/automatic_follow_state_test.go`.
- Run command:
  `cd appview && go test ./internal/instagram -run TestAutomaticFollowState
  -count=1`.
- Confirmed failure: the package failed to compile because the automatic-follow
  state type, state constants, transition validator, and suppression predicate
  did not exist.
- Implement: explicit `pending`, `writing`, `followed`, `alreadyFollowing`, and
  `invalidated` states. Retry returns `writing` to `pending`; successful and
  already-following results are terminal; invalidated work can be reconsidered
  through a new pending transition.
- Green evidence:
  `go test ./internal/instagram -run TestAutomaticFollowState -count=1` and
  `go test ./internal/instagram ./internal/auth ./internal/followwrite -count=1`
  pass.

### Step A4 — IT-025

- Write failing Postgres integration:
  `TestAutomaticFollowLedgerSuppressesReconciliationUntilVerificationRevocation`.
- Run command:
  `cd appview && go test ./internal/instagram -run
  TestAutomaticFollowLedgerSuppressesReconciliationUntilVerificationRevocation
  -count=1`.
- Confirmed failure: the test did not compile because no private
  automatic-follow store, reconciliation parameters/result, or
  verification-ledger deletion operation existed.
- Implement: a transactionally locked private pair ledger, additive import
  support, stable operation creation, terminal followed/already-following
  suppression, invalidated-work revival, and account-scoped ledger deletion.
- Green evidence: the focused Postgres test passes after simulating a successful
  follow, manual unfollow, repeated/same/different-import triggers, revocation,
  and fresh authorization. Neighboring Instagram/auth/followwrite packages
  pass.

### Step A5 — IT-001

- Extend the migration integration test with worker-state/lease indexes and the
  actorful, source-less notification constraint.
- Confirmed failure: migration 000030 lacked the lease-shape constraint,
  owner/target and notification/operation uniqueness, grouping-table removal,
  system-column removal, and actorful payload rules.
- Implement: migrate the private ledger/operation states, add claim and expired
  lease indexes plus lease-shape enforcement, remove actorless grouping storage,
  and require one actorful source-less `instagramMatch` event per operation.
- Green evidence:
  `TEST_DATABASE_URL=... go test ./internal/db -run
  TestInstagramAutomaticFollowMigrationEnforcesWorkerAndActorfulNotificationShape
  -count=1` passes against the running local Postgres container.

### Step A6 — IT-008

- Change the route-policy test to reject every
  `/v1/migrations/instagram/suggestions` policy.
- Confirmed failure: all three list/accept/dismiss policies were still
  registered.
- Implement: unregister the routes and policies, remove the dependency/service
  construction, and delete the HTTP handlers and member-facing suggestion
  service/tests. Preserve verification/account/import routes.
- Green evidence: the focused route-policy test passes; route, app, API, and
  Instagram packages compile; a repository scan finds no removed suggestion
  route, dependency, or service symbol.

### Step A7 — IT-009

- Add a real-Postgres leased-store test before implementing claim/completion.
- Confirmed failure: `AutomaticFollowStore` had no claim/completion API and
  there was no stale-lease error.
- Implement: bounded `FOR UPDATE SKIP LOCKED` claims, expired-lease recovery,
  opaque lease tokens, lease-fenced retry/invalidate/completion, stable rkeys,
  and terminal ledger synchronization.
- Green evidence: the focused real-Postgres tests prove one claimant, stale
  completion rejection, deterministic replay suppression, and the worker's
  complete path through the real leased store. Neighboring
  Instagram/auth/followwrite packages pass.
- Crash-recovery red: after a deterministic PDS write succeeded but before
  database completion, retry classified the target as `alreadyFollowing` and
  therefore omitted the required notification.
- Crash-recovery green: `followwrite.Service` now verifies only the operation's
  deterministic follow record on replay. A matching record completes the
  operation as `followed` and creates exactly one notification; an unrelated
  pre-existing follow remains `alreadyFollowing` and creates none.

### Step A8 — IT-010

- Add a migrated-schema revocation test with a completed operation and an
  actorful historical notification.
- Confirmed failure: imports were deleted but the completed private ledger and
  operation remained.
- Implement: revocation transactionally deletes the owner's imports, private
  pair ledger, and operation rows. It leaves the successful PDS follow alone
  and does not retract the historical actorful notification.
- Green evidence: revised and legacy idempotent revocation tests pass against
  local Postgres, including keyed cooldown tombstone/privacy assertions.

### Step A9 — IT-011

- Add migrated-schema tests for both initial-import and future targeted
  reconciliation.
- Confirmed failures: the reconciliation options had no private
  automatic-follow store, and no automatic-follow matcher existed.
- Implement: initial and future matching share the same policy checks and
  transactionally deduplicate private owner/target operations with stable
  rkeys. Reconciliation no longer creates notifications or review items before
  a successful PDS write. Production dependency wiring uses these producers.
- Green evidence: duplicate initial and future triggers create one pending
  operation and zero `instagramMatch` events in real Postgres.

### Step A10 — IT-012

- Replace the grouped-notification tests with actorful per-operation tests
  before changing production code.
- Confirmed failures: activation had no actor/operation fields and the service,
  API, dispatcher, and payload still depended on system counts/coalescing.
- Implement: successful worker completion creates one actorful source-less event
  in the same transaction; API hydration returns the actor/current follow
  state; push fan-out is immediate and its provider payload contains only type,
  opaque account binding, and notification ID. Preference disable still
  cancels unsent delivery; history is not retracted.
- Green evidence: notification service, worker completion, API/store, social
  notification regression, dispatcher, and bounded identity-free payload tests
  pass against local Postgres.

### Steps A11–A16

For each row, add or revise one focused test, record the meaningful failure,
make the minimum production change, rerun the focused test and neighboring
package tests, then update this section before advancing. The detailed targets,
data flow, migration policy, provider graph, and guardrails in
`04-coding-plan.md` are authoritative.

### Step A11 — IT-021

- Change the shared corpus first to remove suggestion routes/responses and
  replace the grouped actorless match with an actorful, source-less item.
- Confirmed failures: the Go public response test found
  `initialSuggestionCount` and actorless-system assumptions; the Dart corpus
  consumer rejected the smaller import response, looked up removed suggestion
  fixtures, and decoded the match as `GenericSystemNotification`.
- Implement: remove the public initial-suggestion count, update the Go
  notification contract, remove suggestion fixtures/page/delete contracts,
  and decode the actorful match in Dart without source-record identifiers.
- Green evidence:
  `go test ./internal/api -run 'TestInstagramWireCorpus|TestInstagramImportHandlers' -count=1`
  and
  `flutter test test/instagram_migration/data/instagram_wire_contract_test.dart`
  pass against the same synthetic corpus.

### Step A12 — IT-014

- Change the Flutter API-client/repository contract tests before removing the
  public suggestion methods and models.
- Confirmed failure: the tests still expected list, accept, and dismiss
  requests and imported the member-facing `InstagramSuggestion` model.
- Implement: remove those methods, the suggestion model, and every associated
  wire state while preserving verification and import contracts.
- Green evidence: API-client, model-redaction, model-state, and shared-wire
  tests pass; a repository scan finds no Flutter suggestion API/model surface.

### Step A13 — IT-015

- Remove suggestion-provider expectations from the fixed-account provider tests
  before deleting the provider and account-boundary invalidation hook.
- Confirmed failure: the provider graph still loaded and mutated suggestion
  review state for each account.
- Implement: delete the suggestion provider/generated output and retain only
  fixed-account verification/import controllers with their existing
  post-await `ActiveAccountLease` checks.
- Green evidence: the Instagram provider suite and account-boundary suite pass,
  including switch-away/back, late-result, disposal, and verification-resume
  coverage.

### Step A14 — IT-017

- Replace the actorless settings-destination expectations with an actorful
  notification-row test before changing the Dart notification subtype.
- Confirmed failure: `instagramMatch` decoded as a generic system notification,
  rendered no actor/profile, and opened Instagram Settings.
- Implement: decode an actorful source-less match, render the normal profile
  affordance and shared follow/unfollow control, open the actor profile from the
  in-app row, and infer Notifications for the identity-free push payload.
- Green evidence: actorful row/profile tests, destination inference, open-flow,
  notification model, and notifications-provider tests pass.

### Step A15 — IT-016

- Update the page test first for verified terminology, success semantics,
  removed suggestion card, and the destructive action's final position.
- Confirmed failure: the page still described the account as linked, used the
  old discovery color, rendered People You May Know, and placed revocation
  above the import history.
- Implement: use verified terminology, the themed semantic success color for
  discoverability, remove People You May Know, and place the red
  icon-and-label revoke action at the bottom behind its confirmation dialog.
- Green evidence: the focused widget tests prove the bold verified handle,
  semantic discovery switch, absence of suggestion UI, bottom revoke placement,
  error styling, and cancel/confirm dialog behavior.

### Step A16 — IT-023

- Make the page test expect Instagram Export before changing the selector
  default and explanatory copy.
- Confirmed failure: manual entry was selected initially.
- Implement: select Instagram Export by default while preserving native
  path-to-isolate parsing, normalized-only repository input, and the manual
  entry alternative.
- Green evidence: the export/page/privacy tests pass and prove that JSON and ZIP
  still submit only normalized usernames using `sourceType: instagramJson`.

### Step A17 — AT-001–AT-009, REG-001–REG-014

- Full verification on 2026-07-27:
  - real-Postgres `go test ./... -count=1` passes;
  - focused `go test -race` passes for Instagram, auth, follow writing,
    notifications, API, and relationship packages;
  - `go vet ./...` passes;
  - all 1,096 Flutter tests pass;
  - Dart MCP analysis and `flutter analyze` report no issues;
  - `dart format`, `gofmt`, generated-code regeneration, and
    `git diff --check` are clean.
- Automated tests remain synthetic and make no Meta calls. The live Meta,
  physical-device, production safety-adapter, trusted-edge, accessibility, and
  final production privacy checks remain external release gates.

### Automatic Following Completion Checklist

- [x] UT-019 and IT-024 prove exact-owner background session selection
- [x] UT-020 and IT-025 prove verification-lifetime manual-unfollow suppression
- [x] Automatic worker writes one deterministic follow or `alreadyFollowing`
- [x] Exactly one actorful notification is created after each successful write
- [x] Push remains identity-free and account-correct
- [x] Suggestion routes/models/providers/widgets are removed
- [x] Instagram Export is default and People You May Know is absent
- [x] Verified terminology, semantic moss discovery, and bottom revoke dialog pass
- [x] Shared Go/Dart corpus matches the actorful contract
- [x] Focused and broad Go/Flutter verification passes
- [x] External MAN-001–MAN-005 release gates remain explicitly reported

## Implementation Review Correction Pass (2026-07-27)

The user selected **Address required changes** after the fresh implementation
review. This pass addresses `IR-016`–`IR-020` without changing the approved
product behavior or adding migrations.

### Correction Test Order

| Step | Test ID | Requirement IDs | Acceptance criteria | Expected initial state |
|---:|---|---|---|---|
| R1 | IT-017 | FR-026, NFR-008 | AC-025, AC-042 | A notification Follow/Following action can use the globally active account instead of the captured notification owner |
| R2 | TD-012 | FR-017, FR-032, NFR-005 | AC-025, AC-055 | Automatic-follow batch/retry boundaries do not enforce 20/100, five attempts, or capped exponential backoff |
| R3 | IT-020 | FR-028, FR-030 | AC-048 | Reconciliation and automatic-follow workers do not inactivate a departed owner |
| R4 | IT-011 | FR-019 | AC-029, AC-056 | The restoration enqueuer is not called by successful safety-restoring production mutations |
| R5 | IT-009 | FR-015, FR-017, FR-030, NFR-005 | AC-024, AC-025, AC-048 | Final-policy, failure, crash, and concurrent-worker coverage is incomplete |

### Correction Execution Log

#### R1 — IT-017

- Write failing test: render an `instagramMatch` row owned by retained Bob
  while Alice is active, tap Unfollow, and assert Alice's repository is not
  called. Add a late Bob failure after switching to Alice and assert it cannot
  roll back UI or emit an error under Alice.
- Run command:
  `flutter test test/notifications/instagram_match_notification_test.dart
  test/notifications/notifications_page_test.dart`
- Confirmed failure: the retained-owner test observed one call through Alice's
  global repository.
- Implement: pass the captured notification-owner lease into the shared
  Follow/Following control; use the account-specific repository and recheck the
  active lease before repository use, before mutation, after the await, and
  before any error/final UI work.
- Green evidence: 17 focused Instagram-match and shared notification-row widget
  tests pass, including inactive-owner rejection and late-result fencing.

#### R2 — TD-012

- Write failing test: assert the production loop uses batch default 20 and
  rejects 101; assert the store schedules claim attempt three at four seconds;
  exercise exact/above lease/backoff/provider-attempt hard maxima and the
  default/capped exponential retry function.
- Run command: focused `go test` for the AppView command loop and Instagram
  automatic-follow tests, with `TEST_DATABASE_URL` pointed at this worktree's
  compose Postgres for store behavior.
- Confirmed failure: the loop passed 100 instead of 20 and accepted 101; a real
  leased attempt-three operation was scheduled at one second instead of four;
  configurable store/worker limit types were absent.
- Implement: centralize automatic-follow batch 20/100, 60-second lease,
  five-provider-attempt, one-second initial backoff, and five-minute capped
  backoff limits. The store now applies claim-attempt exponential retry timing,
  rejects runtime values above hard maxima, and receives the validated AppView
  configuration; the worker receives the configured provider-attempt limit.
- Green evidence: focused command-loop, pure-boundary, real-Postgres store,
  exact-owner session, and AppView dependency tests pass.

#### R3 — IT-020

- Write failing test: make the automatic worker claim a departed owner's
  operation and assert full inactivation occurs before any PDS call; process a
  real reconciliation job whose target has left `craftsky_profiles` and assert
  its link becomes `membershipInactive`, the job is ignored, and no operation
  is created.
- Run command: focused automatic-worker unit tests plus real-Postgres
  reconciliation integration tests.
- Confirmed failure: neither worker options type accepted membership lookup or
  inactivation dependencies.
- Implement: both workers now check importer and target membership before
  policy/external transitions, recheck a membership-denied policy decision,
  invoke the shared private-data inactivator for the departed DID, and fail
  closed into retry behavior when lookup/inactivation is unavailable.
  Production dependency wiring supplies the shared membership store and
  private-data service to both workers.
- Green evidence: automatic departed-owner tests and the real-Postgres
  departed-target reconciliation test pass; focused Instagram and dependency
  packages remain green.

#### R4 — IT-011

- Write failing test: invoke successful relationship unmute/unblock mutations,
  an account-level moderation safety negate, and expiry of an account-level
  moderation safety output through their production services. Assert each path
  enqueues the corresponding bounded reconciliation work rather than requiring
  a direct trigger call.
- Run command: focused real-Postgres tests across `internal/relationships`,
  `internal/api`, `internal/instagram`, and production dependency construction.
- Confirmed failure: the production relationship and moderation services had no
  restoration dependency, while retention did not scan newly expired
  account-level safety outputs; only direct `ReconciliationTrigger` calls were
  covered.
- Implement: inject one production `ReconciliationTrigger` into relationship
  mutations, moderation output persistence, and retention. Successful
  unmute/unblock enqueues both relationship directions; an eligible
  hide/takedown negate enqueues target-wide work; retention transactionally
  deduplicates newly expired hide/takedown outputs and enqueues target-wide
  work. All paths retain bounded batches and safe retry/error propagation.
- Green evidence: focused relationship restoration, moderation-negate,
  moderation-expiry, retention invocation, trigger, and dependency tests pass
  against the worktree Postgres.

#### R5 — IT-009

- Write failing test: exercise final block-both-directions, importer-mute,
  hide/takedown, departed-member, transient PDS failure, five simultaneous
  store claimants, externally pre-followed completion, and restart after an
  expired post-PDS lease. Inspect durable operation and notification rows.
- Run command: focused worker tests plus real-Postgres concurrent claim,
  already-following, and crash-recovery integration tests.
- Confirmed failure: the final-policy test initially failed to compile because
  the worker fixture could not record invalidation or its safe reason. The
  broader missing scenarios had no durable-row assertions, so the prior green
  suite could not establish `IT-009` as written.
- Implement: extend the test fixture to observe invalidation and add the full
  final-policy/failure/race/restart matrix. No new worker behavior was required:
  permanent final-policy exclusions invalidate before session/PDS work,
  transient PDS failures release for retry, `SKIP LOCKED` yields one claim,
  externally pre-followed work emits no match notification, and expired-lease
  recovery completes the deterministic write with one actorful notification.
- Green evidence: the focused `IT-009` matrix passes against the worktree
  Postgres, including five concurrent claimants and exactly-one durable
  crash-recovery notification.

### Correction Broad Verification

- Real-Postgres `go test ./... -count=1` passes across every AppView package.
- Focused real-Postgres `go test -race` passes for Instagram, auth,
  follow-writing, notifications, API, and relationship packages.
- `go vet ./...`, `gofmt -l .`, and `git diff --check` are clean.
- All 1,096 Flutter tests pass.
- Dart MCP analysis and `flutter analyze` report no issues after removing the
  two new unnecessary-null-check infos; `dart fix --dry-run` reports nothing to
  fix.
- `dart format lib test` changes no files, and build-runner regeneration
  completes without a generated delta.
- Automated verification remains synthetic. The existing live Meta,
  physical-device, production safety-adapter, trusted-edge, accessibility, and
  final production privacy gates remain external.

## Second Implementation Review Correction Pass (2026-07-27)

The user selected **Address required changes** after the second implementation
review. This pass addresses only required finding `IR-022`; the non-blocking
`IR-021`, `IR-023`, and `IR-024` suggestions remain deferred.

### Correction Test Order

| Step | Test ID | Requirement IDs | Acceptance criteria | Expected initial state |
|---:|---|---|---|---|
| S1 | IT-020 | FR-028, FR-030 | AC-048 | A departed import-job owner with zero matching candidates is never checked and its import remains active |

### Correction Execution Log

#### S1 — IT-020

- Write failing real-Postgres test: queue an import-scoped reconciliation job
  for a departed owner whose active retained handle has no matching verified
  link. Assert the shared inactivator changes the import to
  `membershipInactive`, the job terminates as ignored, and no PDS follow
  operation is created.
- Run command:
  `go test -v ./internal/instagram -run
  '^TestReconciliationWorkerInactivatesDepartedOwnerWithoutCandidates$'
  -count=1` with `TEST_DATABASE_URL` pointed at this worktree's Compose
  Postgres.
- Confirmed failure: the job became `ignored` and created no operation, but the
  departed owner's import remained `active` because candidate loading returned
  zero rows before any membership check.
- Implement: check the reconciliation job owner through the shared membership
  boundary before candidate loading. Confirmed departure invokes the shared
  private-data inactivator and stops processing; lookup or inactivation errors
  return through the existing bounded retry path. Candidate importer/target
  checks now reuse the same helper.
- Green evidence: the zero-candidate departed-owner test passes. The retained
  existing-candidate departed-target control also passes, and explicit
  membership-lookup and membership-inactivation failures each persist the job
  as `retryable` with attempt one, no lease, and the expected one-second next
  attempt.

### Second Correction Verification

- Focused real-Postgres departed-owner, departed-target, lookup-failure, and
  inactivation-failure reconciliation tests pass.
- Real-Postgres `go test ./internal/instagram -count=1` and
  `go test ./... -count=1` pass.
- Real-Postgres `go test -race ./internal/instagram -count=1` passes.
- `go vet ./...`, focused `gofmt -l`, and `git diff --check` are clean.
- No Flutter or API contract code changed in this correction. The 17 focused
  notification tests and all 1,096 Flutter tests still pass, and
  `flutter analyze` reports no issues.

## Post-Review Automatic-Follow Storage Rename (2026-07-27)

- Added reversible migration `000031` to rename the private pair ledger to
  `instagram_automatic_follow_ledger` and its import-support join table to
  `instagram_automatic_follow_sources`.
- Renamed both source and PDS-operation references from `suggestion_id` to
  `automatic_follow_id`, together with related constraints and indexes.
- Updated all current AppView SQL, real-Postgres fixtures, schema assertions,
  private export, retention, revocation, reconciliation, retry, and
  notification paths. Historical migrations retain their original names so
  ordered upgrades and rollback remain truthful.
- Green evidence: focused migration forward/rollback tests and the affected
  Instagram, notification, push, and API package suites pass against the
  worktree Postgres.

## Completion Checklist

- [x] Every Must requirement is implemented or recorded as an external release gate
- [x] All planned automated Must tests pass
- [x] AppView disabled/local/fake/full configuration paths are covered
- [x] No automated test contacts Meta
- [x] No real/user-derived private fixture is committed
- [x] Every Instagram API/worker transition enforces current membership
- [x] Eligibility is shared and fails closed at every required boundary
- [x] Membership loss and terminal account deletion remain distinct
- [x] Flutter operations remain fixed-account and redacted
- [x] Existing social notification and follow behavior remains green
- [x] Full Go and Flutter verification is complete
- [ ] Final implementation re-review is complete
- [x] Live Meta/export/safety/device/accessibility gates are explicitly reported

## Known External Gates

- Meta app credentials, webhook subscription, live unrelated-sender DM,
  profile lookup, token/permission, and reply behavior
- Approved observation of current Accounts Center JSON export variants
- Production block/mute safety-data adapter
- Deployed trusted-edge/replica limit behavior
- Physical-device push/open/file-picker/accessibility validation
- Final security/privacy/operator-access review before production enablement
- Remove the temporary raw-webhook diagnostic and unset
  `INSTAGRAM_UNSAFE_LOG_WEBHOOK_BODIES` after the live Meta capability spike.
