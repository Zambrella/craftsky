# Coding Plan: Moderation Flow MVP Without Live Ozone/PDS Report Submission

## 1. Inputs
- Requirements: `docs/changes/2026-05-30-moderation-flow-mvp/01-requirements.md`
- Tests: `docs/changes/2026-05-30-moderation-flow-mvp/02-acceptance-tests.md`
- Document review: `docs/changes/2026-05-30-moderation-flow-mvp/03-document-review.md` — status `Approved with notes`, no blocking gaps.
- Additional codebase references inspected read-only:
  - AppView API routes and stores: `appview/internal/routes/routes.go`, `appview/internal/app/config.go`, `appview/internal/api/{post,post_store,post_response,profile,profile_store,profile_response,timeline_store,notification_store,notifications}.go`
  - Existing migrations: highest discovered migration `000013_profile_social_summary_indexes`; implementer must re-check before creating `000014`.
  - Flutter feed/profile API, repository, Riverpod, widget, model, fake, and localization patterns under `app/lib/feed`, `app/lib/profile`, `app/lib/notifications`, `app/lib/l10n` and matching tests.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`.
- Plan review feedback addressed: user review returned “All looks good to me”; no substantive plan changes requested.

## 2. Implementation Strategy
Build Option A from the requirements as a staged, test-first AppView + Flutter change:

1. Anchor the private data model first with migrations and AppView store tests for reports, forwarding metadata, and moderation outputs.
2. Add report request validation, canonical subject resolution, private persistence, and a placeholder forwarder that prepares but never submits future PDS/Ozone payloads.
3. Add dev-only synthetic moderation ingestion behind config, route registration, and a dedicated token header.
4. Enforce active hide/takedown policy in AppView store/query paths, not Flutter, so alternate clients cannot see suppressed rows.
5. Add warning-only response metadata that contains generic warning intent only, never raw report details or internal moderation reasons.
6. Add Flutter report submission methods, mutation providers, report UI, action-menu entry points, and warning banners using existing Dio repository + Riverpod + localized-copy patterns.

This fits the current codebase because AppView handlers already use small store/resolver/PDS interfaces, Postgres-backed stores use pgx with inline SQL, route registration is centralized in `routes.go`, and Flutter feature code already flows through Dio clients, repositories, `@riverpod` mutation notifiers, generated mappers, fakes, localized strings, and `context.showInfo/showError` messaging.

## 3. Affected Areas
| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView persistence | Numbered SQL migrations in `appview/migrations`; pgx stores in `internal/api` | Add private report table and moderation-output table with supporting indexes, no PDS-visible records | BR-002, FR-007, FR-012, NFR-005, RULE-004 | IT-001, IT-002, IT-003, REG-007 |
| AppView report API | `/v1/*` routes in `routes.go`; handlers in `internal/api`; error envelope helpers | Add post/profile report endpoints with validation, canonical target lookup, self-report rejection, minimal response | FR-001..FR-008, FR-026, FR-027, RULE-001, RULE-005 | AT-001..AT-004, AT-011, IT-004..IT-008, IT-016, IT-020, UT-001..UT-004, UT-008, UT-010 |
| Placeholder forwarding seam | Existing write handlers inject narrow PDS/forwarding interfaces | Add `ReportForwarder` that prepares future payload data in memory and returns safe metadata only; no network submission | BR-003, FR-008, NFR-005, RULE-004 | AT-003, IT-003, IT-005, UT-004, REG-007 |
| Synthetic moderation ingestion | Config validated in `internal/app/config.go`; route registration in `routes.go` | Add dev+flag+token-gated route and trusted-source validation for one synthetic output per request | FR-009..FR-012, FR-023, FR-024, NFR-001, RULE-006 | AT-005, AT-006, AT-009, IT-002, IT-017, IT-018, UT-005, UT-009, UT-011 |
| AppView read enforcement | Store SQL currently returns indexed rows directly | Add SQL-level hidden/taken-down filters and batched/local warning hydration for timeline, profile lists, thread/comment lists, direct post/profile, notifications | BR-004, FR-013..FR-018, FR-025, NFR-003, NFR-004, RULE-003 | AT-007..AT-010, IT-009..IT-015, IT-019, REG-001, REG-003..REG-005 |
| AppView response DTOs | Bare camelCase JSON response structs | Add optional `moderation` metadata to post/profile responses; omit when unmoderated | FR-017, FR-018, FR-022, REG-002 | AT-008, IT-015, UT-007, REG-002, REG-003 |
| Flutter report UX | Dio API clients + repositories + Riverpod mutation notifiers + action widgets | Add report methods, reason/detail models, report sheet/dialog, post/profile report actions, in-flight/retry/success handling | BR-001, FR-019..FR-021, RULE-002 | AT-001, AT-002, AT-012, IT-013, UT-012, UT-014, REG-008 |
| Flutter warning UI | `Post`/`Profile` mappers and shared widgets | Decode optional moderation metadata and render exact generic localized inline warning copy | BR-005, FR-022, NFR-002 | AT-008, UT-013, MAN-002, REG-003 |

## 4. Files And Modules
| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000014_moderation_flow.up.sql` | Create | Create `moderation_reports` and `moderation_outputs` tables and indexes. Re-check migration number before implementation. | FR-007, FR-012, NFR-004 | IT-001, IT-002, IT-019 |
| `appview/migrations/000014_moderation_flow.down.sql` | Create | Drop moderation tables/indexes in reverse order. | FR-007, FR-012 | IT-001, IT-002 |
| `appview/internal/api/report_request.go` / `_test.go` | Create | Decode/validate report request, reason taxonomy, detail normalization, error mapping helpers. | FR-004, FR-027 | UT-001, UT-002 |
| `appview/internal/api/report_store.go` / `_test.go` | Create | Persist private report rows and safe forwarding metadata; allow duplicate reports. | FR-007, RULE-001 | IT-001, IT-003, IT-020 |
| `appview/internal/api/report_forwarder.go` / `_test.go` | Create | Placeholder future PDS/Ozone forwarder seam with no network side effect and no persisted full payload. | FR-008, RULE-004, NFR-005 | UT-004, IT-005, REG-007 |
| `appview/internal/api/report_response.go` / `_test.go` | Create | Minimal accepted report response serialization. | FR-026 | UT-008 |
| `appview/internal/api/report.go` / `_test.go` | Create | `ReportPostHandler`, `ReportProfileHandler`, subject canonicalization, self-report rejection, envelope status codes. | FR-001..FR-006, FR-026, FR-027 | AT-001..AT-004, AT-011, IT-004, IT-006, IT-007, IT-016 |
| `appview/internal/api/moderation_request.go` / `_test.go` | Create | Synthetic moderation request schema/validation, one-output-only and trusted-source checks. | FR-009..FR-012, RULE-006 | UT-009 |
| `appview/internal/api/moderation_store.go` / `_test.go` | Create | Persist moderation outputs and query active output sets/policies. | FR-012, FR-023, FR-024 | IT-002, IT-018 |
| `appview/internal/api/moderation_policy.go` / `_test.go` | Create | Pure policy computation for hide/takedown/warn precedence, negation, expiry, and warning-kind selection. | FR-023, FR-024, RULE-003 | UT-005, UT-006 |
| `appview/internal/api/moderation.go` / `_test.go` | Create | Dev synthetic endpoint handler and token/source validation integration. | FR-009..FR-012, NFR-001 | AT-005, AT-006, IT-017 |
| `appview/internal/app/config.go` / `_test.go` | Change | Add dev moderation config fields/env validation. | FR-009, NFR-001, RULE-006 | UT-011, IT-017 |
| `appview/internal/app/deps.go` | Change | Wire report/moderation stores and placeholder forwarder if using shared deps. Keep handlers dependent on narrow interfaces. | FR-007, FR-008, FR-012 | IT-004, IT-017 |
| `appview/internal/routes/routes.go` / `_test.go` | Change | Register report routes; conditionally register synthetic route only when fully enabled. | FR-001, FR-002, FR-003, FR-009, NFR-001 | IT-008, IT-017, REG-006 |
| `appview/internal/api/post_store.go`, `timeline_store.go`, `notification_store.go` + tests | Change | Add SQL-level moderation filters and warning hydration for posts, timelines, comments/replies, notifications. | FR-013, FR-014, FR-015, FR-025, NFR-003, NFR-004 | IT-009, IT-011, IT-012, IT-014, IT-019, REG-001, REG-005 |
| `appview/internal/api/post.go`, `profile.go` + tests | Change | Return hidden/taken-down direct post/profile as existing not-found-style envelopes; preserve hidden-target eligibility for report handlers. | FR-015, FR-016, RULE-005 | IT-010, IT-012, IT-016, REG-004 |
| `appview/internal/api/post_response.go`, `profile_response.go` + tests | Change | Add optional moderation metadata DTOs without raw reason text. | FR-017, FR-018, FR-022 | UT-007, IT-015, REG-002, REG-003 |
| `app/lib/moderation/models/report_reason.dart` | Create | Shared stable reason taxonomy and localized label mapping helper. | FR-021 | UT-012 |
| `app/lib/moderation/models/report_submission.dart`, `report_result.dart` | Create | Dart wire/request and accepted-response models. | FR-021, FR-026 | IT-013 |
| `app/lib/moderation/models/moderation_metadata.dart` + `.mapper.dart` | Create | Shared optional warning metadata model for `Post` and `Profile`. | FR-022 | UT-013, REG-002 |
| `app/lib/moderation/widgets/report_subject_sheet.dart` | Create | Shared report dialog/sheet for reason, optional details, validation, retry, and in-flight state. | FR-021, RULE-002 | AT-012, UT-012, UT-014 |
| `app/lib/moderation/widgets/moderation_warning_banner.dart` | Create | Shared inline warning banner component with generic localized copy. | FR-022 | UT-013, MAN-002 |
| `app/lib/feed/data/post_api_client.dart`, `post_repository.dart`, `api_post_repository.dart` | Change | Add `reportPost(did, rkey, reasonType, details)` through client/repository. | FR-019, FR-021 | IT-013 |
| `app/lib/profile/data/profile_api_client.dart`, `profile_repository.dart`, `api_profile_repository.dart` | Change | Add `reportProfile(handleOrDid, reasonType, details)` through client/repository. | FR-019, FR-021 | IT-013 |
| `app/lib/feed/providers/report_post_provider.dart` + `.g.dart` | Create | Riverpod mutation notifier for in-flight/retry/success report submission. | FR-021, RULE-002 | UT-014 |
| `app/lib/profile/providers/report_profile_provider.dart` + `.g.dart` | Create | Riverpod mutation notifier for profile reports. | FR-021, RULE-002 | UT-014 |
| `app/lib/feed/models/post.dart` + `.mapper.dart` | Change | Decode optional `moderation` metadata; tolerate absence. | FR-017, FR-022 | UT-013, REG-002 |
| `app/lib/profile/models/profile.dart` + `.mapper.dart` | Change | Decode optional `moderation` metadata; tolerate absence. | FR-018, FR-022 | UT-013, REG-002 |
| `app/lib/feed/widgets/post_card.dart` | Change | Add report menu entry when `onReport` is provided and render post/author warning banner. | FR-019, FR-020, FR-022 | UT-012, UT-013, REG-008 |
| `app/lib/feed/pages/feed_page.dart`, `post_thread_page.dart` | Change | Pass report callbacks only for other users' posts; show report sheet and success/error messages. | FR-019, FR-020, FR-021 | AT-001, AT-012 |
| `app/lib/profile/pages/profile_page.dart`, `widgets/profile_actions.dart`, profile tabs | Change | Add visitor-profile report action, profile warning banner, and no report action on own profile. | FR-019, FR-020, FR-022 | AT-002, AT-012, UT-013 |
| `app/lib/l10n/app_en.arb`, `app/lib/l10n/generated/*` | Change | Add approved reason labels, report UI copy, success/error copy, and exact warning strings. | FR-021, FR-022 | UT-012, UT-013, MAN-002 |
| `app/test/feed/fakes/fake_post_repository.dart`, `app/test/profile/fakes/fake_profile_repository.dart` | Change | Add report callbacks/counters for provider/widget tests. | FR-021, RULE-002 | UT-014, IT-013 |

## 5. Services, Interfaces, And Data Flow

### 5.1 AppView persistence shape
Use AppView-local Postgres tables only. Do not add or change lexicons.

```text
moderation_reports
- id TEXT PRIMARY KEY                         // app-generated UUID string
- reporter_did TEXT NOT NULL
- subject_type TEXT NOT NULL CHECK IN ('post','account')
- subject_did TEXT NOT NULL                   // post author DID or account DID
- subject_collection TEXT                     // post only: social.craftsky.feed.post
- subject_rkey TEXT                           // post only
- subject_uri TEXT                            // post only, canonical AT URI when available
- subject_cid_snapshot TEXT                   // post only, nullable if unavailable
- submitted_handle_snapshot TEXT              // account only, nullable audit/debug snapshot
- reason_type TEXT NOT NULL CHECK approved taxonomy
- details TEXT NULL                           // trimmed; empty/whitespace becomes NULL; max 1000 chars
- device_id TEXT NULL
- forwarding_status TEXT NOT NULL             // 'prepared_not_submitted'
- forwarding_schema_version TEXT NULL         // e.g. 'atproto-create-report-v0'
- forwarding_prepared_at TIMESTAMPTZ NOT NULL
- created_at TIMESTAMPTZ NOT NULL DEFAULT now()

Indexes:
- (reporter_did, created_at DESC)
- (subject_type, subject_did, created_at DESC)
- (subject_uri, created_at DESC) WHERE subject_uri IS NOT NULL
- no uniqueness constraint on reporter/subject/reason
```

```text
moderation_outputs
- id TEXT PRIMARY KEY                         // app-generated UUID string
- source_did TEXT NOT NULL
- subject_type TEXT NOT NULL CHECK IN ('post','account')
- subject_did TEXT NOT NULL                   // account DID or post author DID
- subject_collection TEXT                     // post only
- subject_rkey TEXT                           // post only
- subject_uri TEXT                            // post only, normalized from DID/collection/rkey
- value TEXT NOT NULL CHECK IN ('hide','takedown','warn')
- action TEXT NOT NULL CHECK IN ('apply','negate')
- internal_reason TEXT NULL                   // never returned to Flutter
- expires_at TIMESTAMPTZ NULL
- created_at TIMESTAMPTZ NOT NULL             // event/synthetic timestamp, default now when absent
- indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()

Indexes:
- (subject_type, subject_did, value, indexed_at DESC)
- (subject_uri, value, indexed_at DESC) WHERE subject_uri IS NOT NULL
- (source_did, subject_type, subject_did, value, indexed_at DESC)
- (expires_at) WHERE expires_at IS NOT NULL
```

Guardrails:
- Persist `source_did` only after trusted-source validation.
- Do not persist future full report forwarding payload JSON.
- Do not add uniqueness constraints that reject duplicate reports.

### 5.2 Report endpoint contract

#### `POST /v1/posts/{did}/{rkey}/reports`
- Auth: existing authenticated + `X-Craftsky-Device-Id` middleware.
- Path parsing: `{did}` must parse as `syntax.DID`; `{rkey}` must parse as `syntax.RecordKey`; collection is fixed to `social.craftsky.feed.post`.
- Target lookup: use `PostStore.ResolvePostTarget` or a report-specific equivalent that ignores current moderation visibility and checks indexed existence by DID/rkey.

#### `POST /v1/profiles/{handleOrDid}/reports`
- Auth: existing authenticated + `X-Craftsky-Device-Id` middleware.
- Path parsing: mirror profile handlers by stripping optional leading `@`, accepting DID or handle, and resolving to canonical DID.
- Target lookup: verify the account resolves/exists by profile index or existing profile read path that can bypass visibility filtering for report eligibility.

#### Request body, both report endpoints
```json
{
  "reasonType": "spam",
  "details": "Optional private plain text"
}
```
- Allowed `reasonType`: `harassment`, `hate`, `spam`, `misleading`, `suspected_ai_generated`, `adult_or_graphic`, `impersonation`, `off_topic`, `intellectual_property`, `other`.
- `details` is optional. Trim surrounding whitespace. Treat absent, empty, and whitespace-only as omitted. Reject after trimming if length is greater than 1,000 characters. Do not markdown-render or linkify.
- Reject unknown JSON fields consistently with existing strict request decoders.

#### Success response, both report endpoints
Use `201 Created` because a private AppView report row was created.
```json
{
  "reportId": "<uuid>",
  "status": "accepted"
}
```
The response must not include details, forwarding payload, moderation state, report counts, or existing moderation status.

#### Error mapping
| Case | Status | Error code | Notes |
|---|---:|---|---|
| Malformed JSON | 400 | `malformed_body` | Standard envelope. |
| Unknown field | 400 | `unexpected_field` | Include field map. |
| Invalid DID/rkey/handle syntax | 400 | `invalid_identifier` | Match profile style where applicable. |
| Missing/unsupported reason or details too long | 422 | `validation_failed` | Include `fields.reasonType` / `fields.details`. |
| Self-report post/profile | 422 | `invalid_report_target` | Message: `You cannot report your own post or profile.` |
| Unknown post target | 404 | `post_not_found` | No report row. |
| Unresolvable profile target | 404 or existing resolver failure mapping | `profile_not_found` for known not-found; preserve `identity_unavailable` for resolver outage | No report row. |
| Missing auth/device | Existing middleware status | Existing code | Covered by route tests. |

### 5.3 Placeholder report forwarder

```text
type ReportForwarder interface {
  Prepare(ctx, input) (ForwardingMetadata, error)
}

ForwardingMetadata:
- status: 'prepared_not_submitted'
- schemaVersion: 'atproto-create-report-v0'
- preparedAt: time
```

Flow:
1. Handler decodes and validates request.
2. Handler resolves target to canonical subject snapshot.
3. Handler rejects self-report before persistence.
4. `ReportService` or handler-level coordinator generates `reportId` and calls `ReportForwarder.Prepare` in memory.
5. Store writes `moderation_reports` row with private details and safe forwarding metadata in one operation.
6. Return minimal accepted response.

The forwarder may construct an in-memory future-shaped subject/reason payload for tests, but no PDS/Ozone client dependency should exist in the placeholder and nothing beyond safe metadata is stored.

### 5.4 Synthetic moderation endpoint contract

#### Config and registration
Add config fields in `app.Config`:
```text
EnableDevModeration bool        // APPVIEW_ENABLE_DEV_MODERATION == 'true'
DevModerationToken string       // APPVIEW_DEV_MODERATION_TOKEN
DevLabelerDID string            // CRAFTSKY_DEV_LABELER_DID, dev default 'did:plc:labeler'
TrustedModerationSourceDIDs []string // APPVIEW_TRUSTED_MODERATION_SOURCE_DIDS comma-separated; includes DevLabelerDID in dev
```

Rules:
- Register `POST /v1/dev/moderation/ozone-events` only when `Env == dev`, `EnableDevModeration == true`, and `DevModerationToken` is non-empty.
- If `Env == dev` and `EnableDevModeration == true` but token is empty, `LoadConfig` returns a clear error.
- In prod, clear/ignore dev moderation fields and never register the route.
- The route validates `X-Craftsky-Dev-Moderation-Token` against `DevModerationToken` and intentionally does not use product auth/device middleware.

#### Request body
One object per request; arrays/batches are invalid.

Post output:
```json
{
  "sourceDid": "did:plc:labeler",
  "subject": {
    "type": "post",
    "did": "did:plc:bob",
    "rkey": "3lf2abc"
  },
  "value": "hide",
  "action": "apply",
  "internalReason": "private test reason",
  "expiresAt": "2026-06-01T00:00:00Z"
}
```

Account output:
```json
{
  "subject": {
    "type": "account",
    "did": "did:plc:bob"
  },
  "value": "warn",
  "action": "negate"
}
```

Validation:
- `sourceDid` optional; if omitted, default to `DevLabelerDID`.
- Effective source DID must parse as DID and be in `TrustedModerationSourceDIDs`.
- `subject.type` is `post` or `account`.
- Post subject requires DID + rkey; account subject requires DID; unknown extra subject fields are rejected.
- `value` is `hide`, `takedown`, or `warn`.
- `action` is `apply` or `negate`.
- `expiresAt`, if present, must parse as RFC3339 timestamp.
- `internalReason`, if present, is stored privately and never returned to Flutter.

Success response: use `201 Created` after persistence.
```json
{
  "outputId": "<uuid>",
  "status": "indexed"
}
```

Error mapping:
| Case | Status | Error code |
|---|---:|---|
| Route not registered | 404 | default not found |
| Missing/invalid dev token | 403 | `invalid_dev_moderation_token` |
| Malformed JSON or batch array | 400 | `malformed_body` |
| Invalid shape/value/action/expiry | 422 | `validation_failed` |
| Untrusted source DID | 403 | `untrusted_moderation_source` |

### 5.5 Active moderation policy semantics

Policy inputs are stored outputs for the relevant subject(s): post URI, post author DID, profile DID, notification actor DID, and notification subject post/account.

```text
active apply output if:
- action == 'apply'
- expires_at IS NULL OR expires_at > now()
- there is no later non-expired 'negate' from the same source_did, same subject_type, same subject identity, same value

negate output:
- never enforces by itself
- cancels all prior matching active outputs from the same source only
- does not cancel outputs from other trusted sources

precedence:
- Any active account hide/takedown hides that account and authored posts.
- Any active post hide/takedown hides that post.
- Hide/takedown dominates warn.
- Warn applies only if no active hide/takedown applies.
- If post-level and account-level warn both apply to a post response, emit one warning. Prefer `post` as the warning kind for the post detail/card, otherwise `author`.
```

### 5.6 Response metadata wire shape
Document-review notes require this to be concrete before tests are written.

Post response, warning-only post:
```json
{
  "uri": "at://did:plc:bob/social.craftsky.feed.post/3lf2abc",
  "...": "...",
  "moderation": {
    "warningKind": "post"
  }
}
```

Post response, warning-only author/account:
```json
{
  "...": "...",
  "moderation": {
    "warningKind": "author"
  }
}
```

Profile response, warning-only profile/account:
```json
{
  "did": "did:plc:bob",
  "...": "...",
  "moderation": {
    "warningKind": "profile"
  }
}
```

No warning:
- Omit `moderation` entirely (`omitempty` / nullable absent), so existing clients and Flutter fixtures without the field continue decoding.

Never include:
- raw report details
- raw `internalReason`
- source DID
- moderation output IDs
- report counts
- hide/takedown state for hidden subjects, because those subjects are omitted or returned as not found.

## 6. State, Providers, Controllers, Or DI

### 6.1 AppView DI
- Keep handler constructors narrow and test-injectable.
- Add stores/forwarder either through `app.Deps` or local route construction, but avoid passing the whole `Deps` into handlers.
- Suggested AppView interfaces:

```text
type ReportStore interface {
  CreateReport(ctx, input) (ReportRow, error)
}

type ReportTargetResolver interface {
  ResolvePostReportTarget(ctx, did, rkey) (PostReportTarget, error)
  ResolveAccountReportTarget(ctx, handleOrDID) (AccountReportTarget, error)
}

type ModerationOutputStore interface {
  InsertOutput(ctx, input) (ModerationOutputRow, error)
  ActivePolicyForSubject(ctx, subject) (ModerationPolicy, error)
}
```

Implementation can keep these as concrete methods in `api` package while tests use small fakes for handlers.

### 6.2 Flutter provider graph
Use existing Riverpod code-generation style.

```text
postApiClientProvider
  -> postRepositoryProvider
      -> reportPostProvider (AsyncNotifier<ReportResult?>)

profileApiClientProvider
  -> profileRepositoryProvider
      -> reportProfileProvider (AsyncNotifier<ReportResult?>)

PostCard / feed/profile/thread pages
  -> showReportSubjectSheet
      -> reportPostProvider.submit(...)

ProfileActions / ProfilePage
  -> showReportSubjectSheet
      -> reportProfileProvider.submit(...)
```

Provider behavior:
- `build()` returns `ReportResult?` initialized to `null`.
- `submit(...)` sets `AsyncLoading`, calls repository, then `AsyncData(result)`.
- If already loading, ignore repeated submit calls for that attempt to prevent duplicate requests.
- On error, keep dialog/sheet inputs in local widget state and expose retry via the same submit button.
- After success message is shown, caller may reset provider to `AsyncData(null)`; do not persist reported-state in caches.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### 7.1 Report UI
Create one shared report sheet/dialog used for both post and profile subjects.

Composition sketch:
```text
ReportSubjectSheet
- title: Report post / Report profile
- reason radio/list tiles from approved taxonomy
- optional multiline details TextField
  - max length: 1000
  - plain text only
  - helper text encourages detail for 'other' but does not require it
- submit button disabled when no reason, details too long, or provider loading
- cancel button
- loading indicator in submit button or disabled state
- error text/message on failed submit; input remains unchanged
```

Report action placement:
- `PostCard` gets optional `onReport` and renders `Report post` in the existing overflow/context menu only when non-null.
- Feed, profile tabs, and thread pages pass `onReport` only when authenticated user DID differs from `post.author.did`.
- Visitor profile action set adds a report profile action for non-self profiles. Self profile action set does not expose report.

Success/error feedback:
- On success: `context.showInfo(l10n.reportSubmitSuccess)` with copy equivalent to “Thanks — your report was submitted.”
- On API/network error: keep sheet open, preserve inputs, show retryable localized error.
- Do not add global “already reported” markers.

### 7.2 Warning UI
Add `ModerationWarningBanner` shared widget.

Exact localized copy:
- Post warning: `This post may not follow Craftsky community guidelines.`
- Profile warning: `This profile may not follow Craftsky community guidelines.`
- Author warning on post cards: `This author may not follow Craftsky community guidelines.`

Placement:
- `PostCard`: render one inline banner when `post.moderation?.warningKind` is `post` or `author`. Use post copy for `post`, author copy for `author`.
- Post thread root/comment cards reuse `PostCard`; no separate thread warning implementation should be needed if all cards decode metadata.
- `ProfilePage` / `ProfileMetaSection` area: render profile warning banner for `profile.moderation?.warningKind == profile`.

Accessibility:
- Banner must have readable text and semantics equal to the visible generic string.
- Do not expose raw server-side reason/source in tooltip, semantics, test fixture names, or logs.

### 7.3 Navigation / routes
- No new Flutter routes required.
- No AppView OAuth or public ops routes change.
- Synthetic AppView route is dev-only: `/v1/dev/moderation/ozone-events`.

## 8. Error, Loading, Empty, And Edge States
| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Missing report reason | Disable submit and show accessible validation feedback. | FR-021 | AT-012, UT-012 |
| Details > 1,000 chars | Flutter blocks where possible; AppView rejects with `validation_failed`. | FR-004, FR-021, FR-027 | AT-004, AT-012, UT-002 |
| Whitespace-only details | Trim and store as omitted; Flutter may keep field text until submit. | FR-027 | UT-002 |
| `other` without details | Accept if otherwise valid; helper text can encourage but not require details. | FR-027 | AC-041, UT-002 |
| Self-report attempt | Hide UI action; AppView rejects bypass attempts with `422 invalid_report_target`, no row. | FR-004, FR-020 | AC-026, AC-034, UT-003, IT-006, IT-007 |
| Unknown report target | AppView returns standard not-found/malformed envelope and persists nothing. | FR-005, FR-006 | AC-007, AC-008, IT-006, IT-007 |
| Already hidden but indexed target report | Report handler uses target existence/resolution path that bypasses read visibility; non-self report succeeds. | RULE-005 | AT-011, IT-016 |
| Duplicate reports | No unique constraint; valid reports create separate rows. | RULE-001 | AT-011, IT-020 |
| In-flight report submit | Disable button and provider ignores repeated submit while loading. | RULE-002 | AT-012, UT-014 |
| Report API failure | Keep sheet open, preserve inputs, allow retry, show localized error. | FR-021 | AT-012, UT-014 |
| Successful report | Close or reset sheet and show transient success only; no persisted reported marker. | FR-021 | AC-045, UT-014 |
| Synthetic route disabled/prod | Route not registered; cannot mutate state. | FR-009, NFR-001 | AT-005, IT-017 |
| Synthetic route token missing/invalid | Registered route rejects with `403 invalid_dev_moderation_token`, no mutation. | FR-009, RULE-006 | AC-036, IT-017 |
| Dev moderation enabled without token | Config load/startup fails with clear error. | NFR-001 | AC-037, UT-011 |
| Untrusted synthetic source DID | Reject with `403 untrusted_moderation_source`, no mutation. | RULE-006 | EC-019, UT-009 |
| Batch synthetic payload | Reject array/multiple outputs as invalid request. | FR-011 | EC-018, UT-009 |
| Active hide/takedown post | Omit from lists; direct post returns `404 post_not_found`. | FR-013, FR-015 | AT-007, IT-009, IT-012 |
| Active hide/takedown account | Direct profile returns `404 profile_not_found`; authored posts/notifications omitted. | FR-014, FR-016, FR-025 | AT-007, AT-010, IT-010, IT-014 |
| Warn-only post/account | Content remains visible; optional metadata produces one generic banner. | FR-017, FR-018, FR-022 | AT-008, IT-015, UT-013 |
| Warn + hide/takedown | Hide/takedown wins; no warning response because subject omitted/not found. | RULE-003 | AT-009, IT-012, UT-005 |
| Expired output | Active-policy computation ignores expired output. | FR-024 | AT-009, IT-018 |
| Same-source negate | Later negate cancels prior active matching outputs from same source only. | FR-023 | AT-009, IT-018 |
| Pagination after filtering | Filter in SQL before `LIMIT` and cursor encoding; use `limit+1` where existing queries do. | NFR-004 | AC-014, IT-019, MAN-004 |

## 9. Test Implementation Plan
| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---:|---|---|---|---|
| 1 | IT-001 | `appview/internal/api/report_store_test.go` | Apply new migration; seed reporter, post, profile; insert report rows. | Missing moderation tables/store. |
| 2 | UT-002 | `appview/internal/api/report_request_test.go` | Details omitted/empty/whitespace/1000/1001, `other` without details. | No report decoder/normalizer. |
| 3 | UT-004 | `appview/internal/api/report_forwarder_test.go` | Valid canonical post/profile report with private details. | No placeholder forwarder or metadata contract. |
| 4 | IT-004 / UT-008 | `appview/internal/api/report_test.go`, `report_response_test.go` | Authenticated requests to post/profile report handlers. | Routes/handlers/response absent. |
| 5 | IT-006 / IT-007 / UT-001 / UT-003 / UT-010 | AppView report request/handler tests | Malformed IDs, unknown targets, unsupported reasons, self-report, hidden indexed target. | Validation/canonicalization absent. |
| 6 | IT-008 | `appview/internal/routes/routes_test.go` | Missing auth/device against report endpoints. | Routes not registered or middleware missing. |
| 7 | UT-011 / IT-017 | `appview/internal/app/config_test.go`, `routes_test.go` | Env/flag/token/source config variants. | Config fields and gated route absent. |
| 8 | IT-002 / UT-009 | `appview/internal/api/moderation_store_test.go`, `moderation_request_test.go` | Trusted post/account apply/negate output inserts; untrusted/batch invalid. | Moderation store/request absent. |
| 9 | UT-005 / UT-006 / IT-018 | `moderation_policy_test.go`, `moderation_store_test.go` | Apply/negate/expired/cross-source/warn+hide fixture matrix. | Policy not implemented. |
| 10 | IT-009 / IT-010 / IT-011 / IT-012 / IT-014 | AppView store/handler tests for timeline/profile/thread/direct/notifications | Visible, hidden post, hidden author, hidden actor, hidden subject fixtures. | Read paths leak moderated rows. |
| 11 | IT-019 | `appview/internal/api/*_store_test.go` | Multi-row paginated lists with local moderation outputs. | Enforcement is post-hoc/N+1 or not bounded. |
| 12 | UT-007 / IT-015 / REG-002 / REG-003 | `post_response_test.go`, `profile_response_test.go` | Warn metadata with raw internal reason fixture. | No metadata or raw reason leakage. |
| 13 | IT-013 | `app/test/feed/data/post_api_client_test.dart`, `app/test/profile/data/profile_api_client_test.dart` | Dio mock for report post/profile success/failure. | Client/repository methods absent. |
| 14 | UT-014 | `app/test/feed/providers/report_post_provider_test.dart`, `app/test/profile/providers/report_profile_provider_test.dart` | Slow future, repeated taps, failure, retry success. | Providers absent. |
| 15 | UT-012 | `app/test/feed/widgets/post_card_test.dart`, `app/test/profile/profile_page_test.dart` | Own vs other post/profile action menus and reason validation. | Report actions/UI absent. |
| 16 | UT-013 | `app/test/feed/widgets/post_card_test.dart`, `app/test/profile/profile_page_test.dart` | Warning metadata includes raw reason fixture in test JSON/model. | Warning banner absent or raw text shown. |
| 17 | REG-001..REG-008 | Existing + added regression tests | No moderation outputs and unmoderated fixtures. | Behavior drift from new filters/models/UI. |
| 18 | MAN-001..MAN-004 | Manual local smoke/review | Local stack, Flutter app, synthetic warnings/hides, logs. | Manual acceptance not yet done. |

Focused commands after tests exist:
- From `appview/`: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app`
- From `app/`: `flutter test test/feed test/profile test/notifications`
- From repo root for broader AppView coverage: `just test`

Performance verification for DR-003 / `IT-019`:
- Prefer SQL filtering with `NOT EXISTS`/CTEs and existing `limit+1` pagination before rows are scanned.
- Add tests with several visible and moderated rows where the expected page contains only visible rows and cursor points to the last visible row.
- Instrument store tests with a lightweight pgx wrapper only if practical; otherwise assert implementation uses one store call/local SQL query per list path and no per-row remote/service calls. Manual `MAN-004` can inspect query logs/EXPLAIN if needed.

## 10. Sequencing And Guardrails
- First TDD step: `IT-001` — migration/store persists private post/profile report rows with canonical snapshots, normalized optional details, device ID, timestamps, and safe forwarding metadata.
- Dependencies between work items:
  1. Persistence schema before stores/handlers.
  2. Report request/forwarder/store before report routes.
  3. Dev moderation config before synthetic route registration.
  4. Moderation output store/policy before read-path enforcement.
  5. Response metadata before Flutter model/widget warning tests.
  6. Flutter API/repository methods before mutation providers and report sheet.
- Guardrails:
  - Do not modify `lexicon/`; no ADR needed if lexicons remain untouched.
  - Do not submit reports to PDS/Ozone or add PDS report client calls.
  - Do not persist full prepared forwarding payloads.
  - Do not log private report details, internal reasons, or dev moderation token values in normal logs.
  - Do not implement rate limiting, moderator dashboard, appeals, live Ozone ingestion, or search moderation.
  - Do not filter only in Flutter; enforcement belongs in AppView read/store paths.
  - Do not introduce a uniqueness constraint that blocks duplicate reports.
  - Keep synthetic route impossible in prod and unavailable without explicit flag + token.
  - Preserve camelCase JSON and standard error envelope.
  - Use typed atproto identifiers at boundaries where practical (`syntax.DID`, `syntax.RecordKey`, `syntax.ATURI`) and parse once at the HTTP boundary.
  - Because generated Dart files are involved, run the project’s normal codegen (`build_runner` for mappers/Riverpod and gen-l10n) during implementation, not in this planning stage.
- Out of scope:
  - Live Ozone deployment/WebSocket ingestion.
  - `com.atproto.moderation.createReport` submission.
  - PDS record deletion/mutation as a moderation side effect.
  - Blocks, mutes, appeals, admin report lists, legal/email workflows, push notifications, search moderation, report-spam rate limits, and persisted “already reported” UI state.

## 11. Risks And Open Questions
| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact SQL for active-policy filtering may become verbose across timeline/profile/thread/notification queries. | Risk of inconsistent filtering if copy-pasted. | Create shared SQL helper snippets/constants or small store helper methods in `post_store.go`/`moderation_store.go`; cover every read surface with tests. |
| CPQ-002 | Non-blocking | Profile report target resolution must bypass visibility filtering while direct profile reads enforce hidden account 404. | Hidden-but-indexed profiles might become unreportable if the wrong read path is reused. | Add explicit report-target resolver/store method and `IT-016`; do not call visibility-enforcing `ProfileStore.Read` for eligibility without a bypass option. |
| CPQ-003 | Non-blocking | Existing AppView stores use inline pgx SQL despite AGENTS mentioning sqlc. | Introducing sqlc mid-feature would add tooling scope. | Follow current local pattern for this change; do not add sqlc unless the repo has adopted it before implementation. |
| CPQ-004 | Non-blocking | Flutter generated mapper/provider/localization files can fall out of sync. | Analyzer/test failures unrelated to moderation logic. | TDD builder should run build_runner/gen-l10n after model/provider/l10n edits and commit generated outputs. |
| CPQ-005 | Non-blocking | Warning copy could be accidentally duplicated when post and author warn both apply. | UI clutter or inconsistent copy. | Policy/response builder emits one `warningKind`; tests cover combined warning fixture. |
| CPQ-006 | Non-blocking | Synthetic endpoint source-DID defaults must be safe and dev-only. | Unsafe mutation controls if prod config leaks dev defaults. | Clear dev fields in prod config, register route only with dev+flag+token, and test config variants. |
| CPQ-007 | Non-blocking | Query filtering before `LIMIT` may shorten pages when many rows are hidden if SQL is not structured correctly. | Pagination skips/duplicates or surprising short pages. | Put moderation predicates in SQL `WHERE` before ordering/limit and use existing `limit+1` cursor strategy; cover with `IT-019` and `MAN-004`. |

No blocking implementation questions remain from document review.

## 12. Handoff To TDD Builder
- Coding plan: `docs/changes/2026-05-30-moderation-flow-mvp/04-coding-plan.md`
- TDD execution plan: create/follow `05-implementation-plan.md` if the next-stage workflow requires it; otherwise use this coding plan plus `02-acceptance-tests.md` sequencing.
- Start with test: `IT-001` in `appview/internal/api/report_store_test.go`.
- Focused initial command: from `appview/`, `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestReportStore|TestModeration'` after adding the first failing tests.
- Source of truth: Read `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this `04-coding-plan.md` from disk before coding.
- Notes:
  - Treat document-review notes DR-001, DR-002, and DR-003 as addressed by the concrete endpoint contracts, moderation metadata shape, and performance test strategy above.
  - Preserve privacy boundaries throughout: reports/details/internal reasons stay AppView-private, and hidden/taken-down subjects are omitted or returned as normal not-found responses.
