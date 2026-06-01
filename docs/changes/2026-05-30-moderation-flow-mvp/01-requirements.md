# Requirements: Moderation Flow MVP Without Live Ozone/PDS Report Submission

## 1. Initial Request

Plan a moderation flow that omits live Ozone operation and omits submitting reports to a PDS via the AppView for now, while still allowing the Flutter app to submit reports to the AppView. The AppView should have a placeholder forwarding seam that prepares the future PDS/Ozone report submission but stops before the final network step. The AppView should also have an Ozone-output ingestion seam that can index moderation outputs, with a dev/test path for synthesizing Ozone-like outputs programmatically.

Confirmed expansion: profile/account reports are in scope, and indexed hide/takedown decisions should be enforced in read APIs by filtering posts and authors from feeds/profile reads/direct post fetches. Warning outputs should remain visible with generic localized UI copy.

## 2. Current Codebase Findings

- Relevant files:
  - `appview/internal/routes/routes.go` registers `/v1/*` routes and composes authenticated + device-id middleware for product endpoints.
  - `appview/internal/api/post.go`, `post_store.go`, and `post_response.go` own most post read/write handlers, store methods, and response shapes.
  - `appview/internal/api/profile.go` and `profile_store.go` own profile read/update handlers and profile lookup patterns.
  - `app/lib/feed/widgets/post_card.dart` owns shared post row/menu rendering and currently supports delete but not report menu items.
  - `app/lib/feed/data/post_api_client.dart`, `post_repository.dart`, and `app/test/feed/fakes/fake_post_repository.dart` show the Flutter API/repository/fake pattern.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` defines `/v1/`, auth/device headers, error envelope, and status-code conventions.
- Existing patterns:
  - Authenticated write endpoints require `Authorization` and `X-Craftsky-Device-Id`.
  - AppView stores private-by-intent data in Postgres and keeps PDS tokens server-side.
  - Handlers use AppView stores plus small dependency interfaces so tests can inject fakes.
  - Flutter surfaces API calls through Dio clients, repositories, Riverpod providers, localized UI copy, and `AppMessenger` for user feedback.
- Current behavior:
  - There is no report endpoint, moderation report persistence, moderation label store, synthetic moderation endpoint, or read-time moderation enforcement.
  - Post/profile feeds and direct reads serve indexed records without hide/takedown filtering.
  - Flutter post/profile UI has no report action and no moderation warning display.
- Constraints discovered:
  - Reports and moderation notes are private AppView data; they must not be written to public PDS records in this MVP.
  - Lexicon changes are out of scope.
  - Synthetic moderation endpoints must not be available in production.
  - Existing untracked file `docs/design/ozone-lifecycle.html` is unrelated to this requirements stage and should not be staged with this document.
- Test/build commands discovered:
  - AppView focused tests generally run from `appview/` with `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`.
  - Flutter focused tests run from `app/` with `flutter test <paths>`; analyzer baseline has had unrelated/profile warnings in recent work.

## 3. Clarifying Questions And Decisions

### Q1: Should indexed synthetic/Ozone outputs be enforced or only stored?

Answer: Enforce hide/warn.

Decision / implication: The MVP shall apply hide/takedown decisions to read APIs and shall expose warning metadata for visible subjects.

### Q2: Are account/profile reports in scope?

Answer: Yes.

Decision / implication: The MVP shall support both post reports and profile/account reports.

### Q3: How should the synthetic moderation endpoint be gated?

Answer: Use both `APPVIEW_ENV=dev` and an explicit config flag.

Decision / implication: The route must not be registered unless the process is in dev mode and the explicit flag is enabled.

### Q4: What should direct hidden post fetches return?

Answer: `404 post_not_found`.

Decision / implication: The API should not reveal whether a missing post is absent or hidden/taken down in this MVP.

### Q5: Should duplicate reports be prevented server-side?

Answer: Allow reports from multiple people; prevent accidental double-submit client-side.

Decision / implication: The database must not use uniqueness constraints that block legitimate multiple reports of the same subject.

### Q6: What is the report detail max length?

Answer: 1,000 characters.

Decision / implication: Request validation and Flutter UI must enforce a 1,000-character maximum for optional report details.

### Q7: Should warning labels display raw reason text to users?

Answer: No; use generic localized copy in Flutter and keep raw reason server-side.

Decision / implication: Response metadata may include labels, but Flutter user-facing copy must not display raw report/label reason text in this MVP.

### Q8: How should hidden/taken-down profiles behave?

Answer: Use the same 404 behavior as hidden direct post fetches.

Decision / implication: `GET /v1/profiles/{handleOrDid}` should return a not-found-style response for hidden/taken-down accounts.

### Q9: Where should account-level warning render?

Answer: On the profile page and post cards by that author.

Decision / implication: Account-level warning metadata must be available to profile responses and post responses for authored content.

### Q10: What should the dev synthetic source DID be?

Answer: Configurable `CRAFTSKY_DEV_LABELER_DID`, with a safe local default only in dev.

Decision / implication: The synthetic endpoint should accept or default the source DID only within dev/test constraints.

### Q11: Should server-side direct self-report requests be rejected?

Answer: Yes.

Decision / implication: AppView should reject post/profile report submissions where the authenticated reporter DID is the reported post author or reported account, even though Flutter also hides self-report actions.

### Q12: How should notifications referencing hidden/taken-down subjects or actors behave?

Answer: Omit them.

Decision / implication: Notification list APIs should filter out notifications whose subject or actor is hidden/taken down by active moderation output, rather than returning unavailable placeholder notifications.

### Q13: Should the placeholder forwarder persist the prepared future payload?

Answer: No; persist minimal forwarding status/metadata and recompute future payloads later.

Decision / implication: The MVP should not store a full speculative PDS/Ozone forwarding payload. It should persist safe metadata such as report ID, subject identity, `prepared_not_submitted` status, timestamps, and an optional payload/schema version marker. Report reason/details remain in the private report row, and live forwarding can recompute the payload when implemented.

### Q14: What exact generic warning copy should Flutter use?

Answer: Use neutral localized copy: post warning "This post may not follow Craftsky community guidelines."; profile warning "This profile may not follow Craftsky community guidelines."; author warning on post cards "This author may not follow Craftsky community guidelines."

Decision / implication: Warning copy must avoid implying confirmed wrongdoing and must not include raw report or label reasons.

### Q15: Should warned content be behind a click-through interstitial?

Answer: No; warned content remains visible with an inline warning banner.

Decision / implication: `warn` is not a soft-hide in this MVP. Flutter should display inline warning UI while preserving normal content visibility.

### Q16: What report reason taxonomy should the MVP support?

Answer: `harassment`, `hate`, `spam`, `misleading`, `suspected_ai_generated`, `adult_or_graphic`, `impersonation`, `off_topic`, `intellectual_property`, and `other`.

Decision / implication: Flutter may localize labels, but AppView receives stable snake_case `reasonType` values from this allowlist.

### Q17: What error should direct self-report attempts return?

Answer: Return a standard validation error envelope with `error: "invalid_report_target"` and a user-safe message such as "You cannot report your own post or profile."

Decision / implication: Own-post and own-profile report attempts should share the same error code and should not persist a report row.

### Q18: Should the synthetic moderation endpoint accept batches?

Answer: No; accept one moderation output per request in MVP.

Decision / implication: The dev endpoint maps one request to one stored moderation output, simplifying validation and tests. Batch ingestion can be added later for live Ozone or automod pipelines.

### Q19: What is the account-level hide/takedown scope?

Answer: Suppress the account profile, authored posts, and notifications involving that actor. Do not attempt to suppress every social graph side effect unless it already appears in a current user-facing read surface.

Decision / implication: Account-level hide/takedown means direct profile 404, authored post direct reads 404, authored posts omitted from lists, and notifications where the hidden account is actor or subject omitted. Follows/likes/reposts are deferred unless surfaced through current APIs.

### Q20: What moderation precedence applies when multiple active outputs exist?

Answer: Hide/takedown wins over warn. Account-level hide/takedown hides the account and authored posts. Subject-specific post hide/takedown hides that post. Warn applies only when no active hide/takedown controls the subject. Combined post-level and account-level warn may show one generic warning rather than duplicate banners.

Decision / implication: Enforcement must compute a deterministic policy before shaping API responses.

### Q21: What are synthetic/Ozone-like negation semantics?

Answer: A `negate` cancels all prior active outputs with the same `sourceDid`, subject type, subject identity, and value. It does not require matching reason text and does not cancel outputs from other sources.

Decision / implication: The active-policy computation should treat matching prior outputs from the same source as inactive after a negation.

### Q22: How should multiple labeler/source DIDs be handled?

Answer: Ingest only trusted configured source DIDs, then enforce active outputs from any trusted stored source.

Decision / implication: The synthetic endpoint defaults to `CRAFTSKY_DEV_LABELER_DID`; any explicit `sourceDid` must be allowlisted by dev/test configuration. Future Ozone source allowlisting can reuse the same gate.

### Q23: How should profile report subject identity be stored?

Answer: Store the resolved DID as the canonical subject identity and optionally store the submitted handle as audit/debug snapshot metadata.

Decision / implication: Account moderation targets stable DIDs, not mutable handles.

### Q24: How should post report subject identity be stored?

Answer: Store canonical post identity with author DID, collection/NSID, rkey, AT URI where available, and the current indexed CID snapshot when available.

Decision / implication: Enforcement targets the record path/URI, while the CID snapshot preserves audit context for the exact indexed version reported.

### Q25: Can users report posts that already have active hide/takedown outputs?

Answer: Yes, if the post exists in the AppView index. Normal Flutter UI usually will not expose hidden content, but stale UI/shared-link/race/test flows may still submit reports.

Decision / implication: Report eligibility and read visibility are separate. Self-report rejection still applies.

### Q26: Can users report profiles/accounts that already have active hide/takedown outputs?

Answer: Yes, if the account resolves to a DID / exists in the profile index.

Decision / implication: Profile report eligibility and profile read visibility are separate. Self-report rejection still applies.

### Q27: What feedback should Flutter show after successful report submission?

Answer: Show a transient confirmation, such as "Thanks — your report was submitted." Do not globally mark the item/profile as reported in persisted UI state.

Decision / implication: The in-flight disabled submit state prevents accidental duplicate taps, but the MVP does not introduce cross-refresh/cross-device reported-state tracking.

### Q28: How should optional report details be normalized?

Answer: Treat details as plain free text; trim surrounding whitespace; treat empty/whitespace-only as omitted; enforce the 1,000-character maximum; do not render markdown or linkify details in MVP surfaces.

Decision / implication: Details remain private plain text and are never displayed to end users.

### Q29: Does `other` require detail text?

Answer: No; `other` may be submitted without details, though Flutter can encourage useful detail copy.

Decision / implication: Validation requires a reason type but not optional details for any reason type.

### Q30: What response shape should accepted report endpoints return?

Answer: Return a minimal JSON body with `reportId` and `status: "accepted"`.

Decision / implication: Responses must not include private details, prepared forwarding payload, moderation decision, other-report counts, or whether the subject is already moderated.

### Q31: What route/auth model should the synthetic endpoint use?

Answer: Use `POST /v1/dev/moderation/ozone-events`, registered only in dev with the explicit enable flag, and require a dedicated dev moderation token header such as `X-Craftsky-Dev-Moderation-Token`. Do not require product auth or device middleware for this local/dev synthetic route.

Decision / implication: Normal user auth is neither required nor sufficient to mutate synthetic moderation state. The route should still follow `/v1/` API conventions and rely on dev-only registration plus the dedicated dev token.

### Q32: How is the dev moderation token configured?

Answer: `APPVIEW_ENV=dev`, `APPVIEW_ENABLE_DEV_MODERATION=true`, and non-empty `APPVIEW_DEV_MODERATION_TOKEN` are required for the synthetic route. If the enable flag is true but the token is missing, AppView should fail startup with a clear error.

Decision / implication: A half-enabled synthetic moderation route is treated as unsafe configuration rather than silently operating without token protection.

## 4. Candidate Approaches

### Option A: Local report queue + synthetic labels + enforcement

Summary: Build local report intake for posts and accounts, local synthetic moderation-output ingestion, and minimal AppView enforcement now, with interfaces for future PDS/Ozone integration.

Pros:

- Gives users a report path quickly.
- Exercises AppView enforcement decisions before running Ozone.
- Keeps private reports in Postgres, aligned with architecture rules.
- Makes future Ozone integration additive.
- Synthetic endpoint makes automated and local testing practical.
- Supports both content and account/profile safety reports from the start.

Cons:

- Creates local moderation tables before live Ozone exists.
- Requires touching multiple read surfaces.
- Requires careful gating for synthetic endpoints.
- Larger than post-only reporting.

Risks: High; affects private safety data, migrations, content visibility, and user-facing reports.

### Option B: Report queue only, no label enforcement yet

Summary: Add Flutter report submission and AppView report persistence for posts/profiles, but do not index or enforce Ozone-like outputs.

Pros:

- Smaller first slice.
- No risk of accidentally hiding content.
- Straightforward user-facing report MVP.

Cons:

- Does not test the action side of moderation.
- Ozone/label enforcement remains unproven.
- Synthetic endpoint is less useful.

Risks: Medium; private reports and UI, but fewer read-path risks.

### Option C: Full Ozone-compatible event ingestion now

Summary: Define ingestion close to real Ozone label stream semantics and build the synthetic endpoint as a local adapter.

Pros:

- Future Ozone integration may be smoother.
- Less rework if Ozone event shapes are stable and mapped accurately.

Cons:

- Larger and easier to overfit to external implementation details.
- Requires more Ozone-specific decisions while intentionally not running Ozone.
- More complex tests.

Risks: High; external integration assumptions plus content/account enforcement.

## 5. Recommended Direction

Recommended approach: Option A, with strict boundaries.

Why: It matches the requested simplification while still proving the hard parts: report UX, private report persistence, synthetic Ozone-like output indexing, account/post visibility enforcement, and safe future integration seams. It avoids live Ozone and PDS report submission while ensuring test design can cover the future-facing moderation lifecycle.

## 6. Problem / Opportunity

Craftsky has social posting, profiles, timelines, threads, follows, and notifications, but no user-facing safety intake path or AppView enforcement mechanism for moderator decisions. Before inviting broader testers, users need a way to report harmful, spammy, misleading, suspected-AI, off-topic, adult/graphic, or impersonating content/accounts. The AppView also needs a safe local mechanism to simulate future Ozone outputs and prove how hide/takedown/warn decisions affect product reads.

## 7. Goals

- G-001: Let signed-in users report posts and profiles/accounts from the Flutter app.
- G-002: Persist report submissions privately in AppView Postgres.
- G-003: Introduce a placeholder report-forwarding seam that prepares future PDS/Ozone report payloads but does not submit them.
- G-004: Index synthetic Ozone-like moderation outputs through the same internal seam future Ozone ingestion will use.
- G-005: Apply hide/takedown outputs to AppView read APIs using 404/filtering behavior.
- G-006: Surface warn outputs as generic user-facing Flutter warnings without exposing raw report text.
- G-007: Keep all new dev/test synthetic controls impossible to use in production.

## 8. Non-Goals

- NG-001: Do not run or deploy Ozone in this slice.
- NG-002: Do not submit reports to a PDS or Ozone with `com.atproto.moderation.createReport`.
- NG-003: Do not implement live Ozone WebSocket label ingestion.
- NG-004: Do not implement blocks, mutes, appeals, moderator dashboard, email, legal workflow, or Ozone UI.
- NG-005: Do not add or change atproto lexicons.
- NG-006: Do not delete or mutate user PDS records as part of moderation.
- NG-007: Do not display raw report details or raw synthetic-label reasons to end users.
- NG-008: Do not implement search moderation; search is not yet present.
- NG-009: Do not create an internal admin report list unless a later scope explicitly asks for it.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in user | Craftsky user viewing posts/profiles. | Report posts/profiles and get clear confirmation or retry feedback. |
| Reported author | User whose content/account is reported or labeled. | No public leakage of private report text; no PDS deletion by Craftsky. |
| Craftsky AppView | Go backend serving product APIs. | Persist reports, prepare future forwarding payloads, index moderation outputs, and enforce visibility. |
| Flutter app | Mobile client. | Submit reports, prevent accidental double-submit, and render generic moderation warnings. |
| Future Ozone integration | Later moderation labeler/service. | Reuse forwarding and ingestion seams without changing product API contracts unnecessarily. |
| Test/design agents | Workflow participants after requirements. | Clear acceptance criteria for report intake, synthetic outputs, and enforcement. |

## 10. Current Behavior

The AppView has no report API, no moderation report table, no moderation label/output table, no synthetic moderation endpoint, and no read-time filtering based on moderation decisions. Flutter post/profile UI has no report action and no warning rendering. Post/profile/timeline/thread/notification reads return indexed content regardless of future moderation status.

## 11. Desired Behavior

After the change, signed-in users can report posts and profiles/accounts. AppView stores those reports privately and prepares but does not submit or fully persist a future atproto/Ozone report payload. Dev/test callers with a configured dev moderation token can synthesize one Ozone-like moderation output per request when explicitly enabled in dev. AppView indexes trusted-source outputs and enforces hide/takedown by filtering posts/authors/notifications from read APIs and returning 404 for direct hidden post/profile fetches. Warn outputs keep subjects visible but add moderation metadata so Flutter can show generic localized inline warnings.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky shall provide a user-facing report flow for posts and profiles/accounts. | Users need a safety intake path before broader testing. | Prompt / User feedback | AC-001, AC-002, AC-021 |
| BR-002 | Business | Must | Craftsky shall keep report submissions private in AppView Postgres and shall not write report records to user PDS repositories in this MVP. | Reports are private-by-intent and PDS data is public. | AGENTS.md / Reference doc / Prompt | AC-003, AC-004, AC-022 |
| BR-003 | Business | Must | Craftsky shall preserve future Ozone/PDS integration seams without performing live Ozone or PDS report submission. | Allows staged implementation without blocking on Ozone. | Prompt | AC-005, AC-006, AC-035 |
| BR-004 | Business | Must | Craftsky shall enforce indexed hide/takedown moderation outputs in product read APIs. | Ozone-like actions must affect reach once indexed. | User answer | AC-012, AC-013, AC-014, AC-015, AC-016, AC-033, AC-040 |
| BR-005 | Business | Must | Craftsky shall support warning moderation outputs without exposing raw report or label reason text to users. | Users need context, but raw reasons may be sensitive. | User answer | AC-017, AC-018, AC-030, AC-039 |
| FR-001 | Functional | Must | The AppView shall expose `POST /v1/posts/{did}/{rkey}/reports` for authenticated post reports. | Post reporting is core moderation intake. | Recommended direction | AC-001, AC-003, AC-007 |
| FR-002 | Functional | Must | The AppView shall expose `POST /v1/profiles/{handleOrDid}/reports` for authenticated profile/account reports. | Profile/account reporting is in scope. | User feedback | AC-002, AC-003, AC-008 |
| FR-003 | Functional | Must | Report endpoints shall require the existing authenticated + device-id middleware stack. | Matches API conventions for product write endpoints. | Codebase / API spec | AC-009 |
| FR-004 | Functional | Must | Report request validation shall reject malformed JSON, invalid path identifiers, unsupported reason types outside the approved taxonomy, report details longer than 1,000 characters, and direct self-report attempts. | Prevents bad data, oversized private text, and appeal/self-report complexity. | User answer / API spec | AC-010, AC-011, AC-034, AC-041 |
| FR-005 | Functional | Must | AppView shall validate that a reported post exists in the AppView index before storing a post report, regardless of whether that post is currently visible after moderation filtering. | Prevents reports for unknown post targets while allowing stale/race reports against already moderated posts. | Codebase / Discovery / Grill review | AC-007, AC-043 |
| FR-006 | Functional | Must | AppView shall resolve and validate a reported profile/account target before storing an account report, regardless of whether that account is currently visible after moderation filtering. | Prevents malformed or unresolvable account targets while allowing stale/race reports against already moderated accounts. | User feedback / Codebase / Grill review | AC-008, AC-044 |
| FR-007 | Functional | Must | AppView shall persist report submissions with reporter DID, canonical subject identity, reason type, normalized optional details, optional device ID, timestamps, forwarding status, and safe audit snapshots. | Provides a private audit/intake queue and future forwarding state. | Recommended direction / Grill review | AC-003, AC-004, AC-042 |
| FR-008 | Functional | Must | AppView shall call a placeholder report-forwarding interface that prepares future atproto/Ozone payload data but does not perform the final PDS network submission or persist the full prepared payload. | Keeps future seam while honoring no-live-PDS and data-minimization constraints. | Prompt / Grill review | AC-005, AC-006, AC-035 |
| FR-009 | Functional | Must | The AppView shall expose `POST /v1/dev/moderation/ozone-events` only when `APPVIEW_ENV=dev`, an explicit enable flag, and a non-empty dev moderation token are configured; accepted requests shall require the dev moderation token header and shall not require product auth/device middleware. | Synthetic controls must not be available in prod or without the explicit dev token. | User answer / Grill review | AC-019, AC-020, AC-036, AC-037 |
| FR-010 | Functional | Must | Synthetic moderation output ingestion shall support `post` and `account` subject types. | Tests must cover content and author/profile decisions. | User feedback | AC-019, AC-023 |
| FR-011 | Functional | Must | Synthetic moderation output ingestion shall support `hide`, `takedown`, and `warn` values with `apply` and `negate` actions, one moderation output per request. | Covers minimal enforcement and reversal semantics with simple validation. | Recommended direction / Grill review | AC-019, AC-024, AC-038 |
| FR-012 | Functional | Must | AppView shall persist moderation outputs with trusted source DID, subject identity, value, negation, optional expiry, optional internal reason, created timestamp, and indexed timestamp. | Supports enforcement, future Ozone ingestion, and tests. | Recommended direction / Grill review | AC-023, AC-024, AC-038 |
| FR-013 | Functional | Must | AppView read APIs shall omit hidden/taken-down posts from list surfaces. | Hide/takedown changes reach. | User answer | AC-012, AC-014 |
| FR-014 | Functional | Must | AppView read APIs shall omit posts by hidden/taken-down authors from list surfaces. | Account-level moderation changes reach. | User feedback | AC-013, AC-014 |
| FR-015 | Functional | Must | Direct `GET /v1/posts/{did}/{rkey}` for hidden/taken-down posts or posts by hidden/taken-down authors shall return `404 post_not_found`. | Avoids revealing moderation-vs-absence details. | User answer | AC-015 |
| FR-016 | Functional | Must | `GET /v1/profiles/{handleOrDid}` for hidden/taken-down accounts shall return a not-found-style 404 response. | Keeps profile enforcement consistent with post enforcement. | User answer | AC-016 |
| FR-017 | Functional | Must | Warn outputs shall be represented in post/profile response metadata without hiding the subject. | Lets Flutter render warning UI. | User answer | AC-017, AC-039 |
| FR-018 | Functional | Must | Account-level warn outputs shall be available on the profile page and post cards by that author. | Confirmed warning location. | User answer | AC-018, AC-039 |
| FR-019 | Functional | Must | Flutter shall expose report actions for other users' posts and visitor profiles. | Completes the user-facing report flow. | Recommended direction / User feedback | AC-001, AC-002, AC-025 |
| FR-020 | Functional | Must | Flutter shall not show report actions for the signed-in user's own posts or own profile in this MVP. | Avoids appeal/self-report complexity. | Recommended direction | AC-026 |
| FR-021 | Functional | Must | Flutter shall present localized approved reason options, optional detail entry, disabled in-flight submit state, transient success confirmation, and retryable error handling. | Ensures usable report UX without adding persisted reported-state UI. | Recommended direction / Grill review | AC-027, AC-028, AC-029, AC-045 |
| FR-022 | Functional | Must | Flutter shall render exact generic localized inline warning copy for warned posts/profiles/authors and must not render raw server-side reason text. | Protects sensitive moderation context. | User answer / Grill review | AC-017, AC-018, AC-030, AC-039 |
| FR-023 | Functional | Should | AppView shall support negated moderation outputs cancelling all prior matching active outputs with the same source DID, subject type, subject identity, and value. | Needed for reversible synthetic/Ozone decisions. | Recommended direction / Grill review | AC-024, AC-038 |
| FR-024 | Functional | Should | AppView shall treat expired moderation outputs as inactive. | Future-compatible label semantics. | Recommended direction | AC-024 |
| FR-025 | Functional | Must | Notification list APIs shall omit notifications whose subject or actor is hidden/taken down by active moderation output. | Prevents moderated subjects and actors from leaking through notifications. | User feedback | AC-033 |
| FR-026 | Functional | Must | Accepted report endpoints shall return only `reportId` and `status: "accepted"`. | Keeps privacy boundaries clear and avoids implying immediate moderation action. | Grill review | AC-046 |
| FR-027 | Functional | Must | Report detail handling shall trim surrounding whitespace, treat empty/whitespace-only text as omitted, store plain text only, and not require details for any reason type including `other`. | Reduces friction and avoids unsafe rich-text behavior. | Grill review | AC-011, AC-041 |
| NFR-001 | Non-functional | Must | The synthetic moderation endpoint shall be impossible to register in production configuration or without required dev token configuration. | Prevents production abuse. | User answer / Risk review / Grill review | AC-020, AC-037 |
| NFR-002 | Non-functional | Must | Report details and synthetic/internal reasons shall not be surfaced as end-user copy. | Protects reporter/moderator privacy. | User answer | AC-030 |
| NFR-003 | Non-functional | Should | Moderation enforcement should be applied in store/query paths rather than only in Flutter. | Prevents hidden content from leaking across clients. | Architecture | AC-012, AC-013, AC-014, AC-015, AC-016, AC-033 |
| NFR-004 | Non-functional | Should | Moderation lookups should be efficient enough for paginated timeline/profile/thread/notification reads without N+1 queries per row. | Read surfaces are latency-sensitive. | Codebase / Risk review | AC-031 |
| NFR-005 | Non-functional | Must | Placeholder forwarding shall minimize private-data duplication by not storing full prepared forwarding payloads. | Reduces leakage and future-shape coupling risk. | Grill review | AC-035 |
| RULE-001 | Business rule | Must | Reports may be submitted by multiple users for the same subject; the server shall not reject legitimate reports solely because another report already exists. | Multiple reports are a signal. | User answer | AC-032 |
| RULE-002 | Business rule | Must | Flutter shall prevent accidental double-submit during one report submission attempt. | Avoids accidental duplicates without blocking legitimate reports. | User answer | AC-029 |
| RULE-003 | Business rule | Must | Hide and takedown outputs have the same user-visible enforcement behavior in this MVP and dominate warn outputs. | Keeps the first enforcement policy simple. | Recommended direction / Grill review | AC-012, AC-013, AC-015, AC-016, AC-040 |
| RULE-004 | Business rule | Must | Craftsky shall never delete or mutate a user's PDS records as part of this MVP's report or moderation-output flows. | AppView controls reach, not PDS data. | AGENTS.md / Prompt | AC-022 |
| RULE-005 | Business rule | Must | Report eligibility is based on indexed target existence/resolution rather than current read visibility, except self-report attempts remain invalid. | Allows reports from stale UI/race/shared-link flows. | Grill review | AC-043, AC-044 |
| RULE-006 | Business rule | Must | Only configured trusted moderation source DIDs may be ingested, and active outputs from any trusted stored source may affect enforcement. | Prevents arbitrary labeler spoofing while supporting future Ozone allowlisting. | Grill review | AC-023, AC-036 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-019 | Given a signed-in user views another user's post, when they submit a valid report, then Flutter calls `POST /v1/posts/{did}/{rkey}/reports` and the user sees a success confirmation. |
| AC-002 | BR-001, FR-002, FR-019 | Given a signed-in user views another user's profile, when they submit a valid report, then Flutter calls `POST /v1/profiles/{handleOrDid}/reports` and the user sees a success confirmation. |
| AC-003 | BR-002, FR-001, FR-002, FR-007 | Given a valid post or profile report request, when AppView accepts it, then a private `moderation_reports` row is persisted with reporter, subject, reason, timestamp, and forwarding status. |
| AC-004 | BR-002, FR-007 | Given a persisted report with optional details, when AppView returns the user-facing response, then the response contains only report ID/status and does not include private detail text. |
| AC-005 | BR-003, FR-008 | Given a valid report, when the placeholder forwarder is invoked, then it prepares future atproto/Ozone subject/reason payload data and records a non-submitted status. |
| AC-006 | BR-003, FR-008, RULE-004 | Given a report is accepted, when the request completes, then no PDS network submission or PDS record write has occurred. |
| AC-007 | FR-001, FR-005 | Given an unknown or malformed post target, when a report is submitted, then AppView rejects it with the standard error envelope and does not persist a report. |
| AC-008 | FR-002, FR-006 | Given an unresolvable or malformed profile/account target, when a report is submitted, then AppView rejects it with the standard error envelope and does not persist a report. |
| AC-009 | FR-003 | Given a report endpoint request lacks valid auth or device ID, when it reaches routing middleware, then it is rejected according to existing auth/device error-envelope behavior. |
| AC-010 | FR-004 | Given malformed JSON or unsupported `reasonType`, when a report is submitted, then AppView returns a 400/422-style standard error envelope and no report row is created. |
| AC-011 | FR-004, FR-027 | Given optional report details exceed 1,000 characters, when a report is submitted, then AppView rejects it and Flutter prevents submission where possible. |
| AC-012 | BR-004, FR-013, NFR-003, RULE-003 | Given a post has an active `hide` or `takedown` output, when timeline/profile/thread list APIs are requested, then that post is omitted. |
| AC-013 | BR-004, FR-014, NFR-003, RULE-003 | Given an account has an active `hide` or `takedown` output, when list APIs include posts by that account, then those posts are omitted. |
| AC-014 | BR-004, FR-013, FR-014 | Given list APIs return paginated results after moderation filtering, when hidden rows are filtered out, then the API does not expose hidden rows and pagination remains deterministic. |
| AC-015 | BR-004, FR-015, RULE-003 | Given a direct post target is hidden/taken down or authored by a hidden/taken-down account, when `GET /v1/posts/{did}/{rkey}` is requested, then AppView returns `404 post_not_found`. |
| AC-016 | BR-004, FR-016, RULE-003 | Given an account has an active `hide` or `takedown` output, when `GET /v1/profiles/{handleOrDid}` is requested, then AppView returns a not-found-style 404 response. |
| AC-017 | BR-005, FR-017, FR-022 | Given a post has an active `warn` output and no active hide/takedown output, when the post is returned, then the response includes moderation metadata and Flutter renders generic localized warning copy. |
| AC-018 | BR-005, FR-018, FR-022 | Given an account has an active `warn` output, when its profile or authored post card is rendered, then Flutter shows generic localized warning copy without raw reason text. |
| AC-019 | FR-009, FR-010, FR-011 | Given AppView runs in dev with the explicit moderation dev flag enabled, when a valid synthetic moderation request is submitted, then AppView ingests and persists the output. |
| AC-020 | FR-009, NFR-001 | Given AppView runs in prod or dev without the explicit flag, when the synthetic moderation route is requested, then it is unavailable/not registered and cannot mutate moderation state. |
| AC-021 | BR-001, FR-004 | Given report intake exists for posts and profiles, when users encounter harassment, hate, spam, misleading content, suspected AI-generated content, adult/graphic content, impersonation, off-topic content, intellectual-property issues, or other reportable content/account behavior, then they can choose a matching approved reason or `other`. |
| AC-022 | BR-002, RULE-004 | Given any report or synthetic moderation output flow executes, when storage/network effects are inspected, then only AppView-local moderation tables are changed and PDS records are untouched. |
| AC-023 | FR-010, FR-012 | Given a synthetic post or account moderation output is accepted, when the moderation store is queried, then it contains source DID, subject identity, value, timestamps, and optional internal reason. |
| AC-024 | FR-011, FR-023, FR-024 | Given apply, negate, and expired outputs exist for the same subject/value/source, when active policy is computed, then negated or expired outputs do not enforce visibility. |
| AC-025 | FR-019 | Given Flutter renders another user's post/profile, when the overflow/action UI opens, then `Report post` or `Report profile` is available. |
| AC-026 | FR-020 | Given Flutter renders the signed-in user's own post/profile, when the overflow/action UI opens, then report actions are absent. |
| AC-027 | FR-021 | Given the report dialog/sheet opens, when no reason is selected or details exceed the max length, then submission is blocked with accessible localized feedback. |
| AC-028 | FR-021 | Given report submission fails, when Flutter receives an API/network error, then the report dialog/sheet preserves input and offers retry. |
| AC-029 | FR-021, RULE-002 | Given a report is submitting, when the user taps submit repeatedly, then only one request is sent for that submission attempt. |
| AC-030 | FR-022, NFR-002 | Given moderation metadata includes internal reason text, when Flutter renders warnings, then user-visible copy is generic/localized and raw reason text is not displayed. |
| AC-031 | NFR-004 | Given a paginated read returns multiple posts/profiles, when moderation policy is applied, then implementation avoids per-row remote calls and uses local indexed state. |
| AC-032 | RULE-001 | Given different users report the same post/profile or the same user reports a subject again later, when requests are valid, then the server does not reject them solely due to an existing report row. |
| AC-033 | BR-004, FR-025, NFR-003 | Given a notification references a hidden/taken-down subject or actor, when notification list APIs are requested, then that notification is omitted. |
| AC-034 | FR-004 | Given a user directly submits a report request for their own post or own profile/account, when AppView validates the request, then AppView rejects it with the standard error envelope and does not persist a report. |
| AC-035 | BR-003, FR-008, NFR-005 | Given a report is accepted and the placeholder forwarder prepares future payload data, when persisted forwarding state is inspected, then the full prepared forwarding payload is not stored; only safe status/metadata such as non-submitted status, timestamps, and optional schema marker are stored. |
| AC-036 | FR-009, RULE-006 | Given the synthetic endpoint is registered, when a request omits or provides an invalid `X-Craftsky-Dev-Moderation-Token`, then AppView rejects the request and does not mutate moderation state; product auth/device headers are not required for this dev route. |
| AC-037 | FR-009, NFR-001 | Given `APPVIEW_ENABLE_DEV_MODERATION=true` and `APPVIEW_ENV=dev` but `APPVIEW_DEV_MODERATION_TOKEN` is empty, when AppView starts, then startup fails with a clear configuration error. |
| AC-038 | FR-011, FR-012, FR-023 | Given a `negate` output is ingested, when active policy is computed, then all prior active outputs from the same source DID with the same subject type, subject identity, and value are inactive, while outputs from other trusted sources remain active. |
| AC-039 | BR-005, FR-017, FR-018, FR-022 | Given a warned post/profile/author is rendered without active hide/takedown, when Flutter displays moderation UI, then content remains visible with one inline localized warning using the approved generic copy and no raw reason text. |
| AC-040 | BR-004, RULE-003 | Given both warn and hide/takedown outputs could apply to a subject, when AppView computes effective moderation policy, then hide/takedown behavior wins and the subject is filtered or returned as 404 rather than warned. |
| AC-041 | FR-004, FR-027 | Given a report request uses an approved reason type including `other`, when optional details are omitted, empty, or whitespace-only, then AppView accepts otherwise valid requests and stores details as omitted; unsupported reason types are rejected. |
| AC-042 | FR-007 | Given a post report is accepted, when the persisted row is inspected, then it contains canonical author DID, collection/NSID, rkey, AT URI where available, and current indexed CID snapshot where available; given a profile report is accepted, then it stores canonical subject DID and may store the submitted handle snapshot. |
| AC-043 | FR-005, RULE-005 | Given a reported post exists in the index but currently has active hide/takedown moderation, when another non-author user submits a valid report, then AppView accepts and persists the report. |
| AC-044 | FR-006, RULE-005 | Given a reported account resolves to a DID / exists in the profile index but currently has active hide/takedown moderation, when another user submits a valid report, then AppView accepts and persists the report. |
| AC-045 | FR-021 | Given a report submission succeeds, when Flutter receives the accepted response, then it shows a transient success confirmation and does not persistently mark the item/profile as reported. |
| AC-046 | FR-026 | Given AppView accepts a report, when it returns the response body, then it includes only `reportId` and `status: "accepted"` and excludes private details, forwarding payload, moderation decision, other-report counts, and existing moderation state. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Report target post is malformed or not indexed. | Reject with standard error envelope and no persisted report. | FR-004, FR-005 |
| EC-002 | Report target profile handle cannot resolve. | Reject with standard error envelope and no persisted report. | FR-006 |
| EC-003 | Report details exceed 1,000 characters. | Reject server-side; prevent or show validation feedback client-side. | FR-004, FR-021 |
| EC-004 | User submits multiple reports for same subject over time. | Accept valid reports; do not block due to existing rows. | RULE-001 |
| EC-005 | User double-taps submit while request is in flight. | Flutter sends only one request for that attempt. | RULE-002 |
| EC-006 | Synthetic endpoint enabled flag is missing. | Route is not available and cannot mutate state. | FR-009, NFR-001 |
| EC-007 | Synthetic hide and warn both exist for a subject. | Hide/takedown dominates; subject is filtered or 404 rather than warned. | FR-013, FR-017, RULE-003 |
| EC-008 | Negated synthetic output follows an applied output. | Matching prior output is no longer active. | FR-023 |
| EC-009 | Synthetic output expires. | Expired output is not enforced. | FR-024 |
| EC-010 | Account-level hide applies to an author with otherwise visible posts. | Author profile direct read returns 404 and their posts are omitted from enforced list/direct read surfaces. | FR-014, FR-016 |
| EC-011 | Account-level warn applies to an author. | Profile page and authored post cards show generic warning copy. | FR-018, FR-022 |
| EC-012 | Notification references hidden/taken-down post or actor. | Notification is omitted from notification list API results. | FR-025 |
| EC-013 | Direct self-report attempt bypasses Flutter UI. | AppView rejects with `invalid_report_target` and no persisted report. | FR-004 |
| EC-014 | Report reason is `other` with no details. | Accept otherwise valid report and store details as omitted. | FR-027 |
| EC-015 | Report details are whitespace only. | Trim and store details as omitted. | FR-027 |
| EC-016 | Report target is already hidden/taken down but exists in the index. | Accept otherwise valid reports from non-author users. | FR-005, FR-006, RULE-005 |
| EC-017 | Synthetic endpoint flag is enabled but token config is missing. | AppView fails startup with a clear configuration error. | FR-009, NFR-001 |
| EC-018 | Synthetic endpoint receives more than one output in a request. | Reject as invalid request for MVP. | FR-011 |
| EC-019 | Synthetic output specifies an untrusted source DID. | Reject and do not persist or enforce the output. | FR-012, RULE-006 |
| EC-020 | Post-level and account-level warn both apply. | Render at most one generic inline warning for the post card/profile surface. | FR-022, RULE-003 |

## 15. Data / Persistence Impact

- New fields/tables:
  - `moderation_reports` for private report intake.
  - `moderation_labels` (or equivalent) for indexed synthetic/Ozone-like moderation outputs.
  - Report rows should store canonical subject identity: profile/account reports use resolved DID plus optional submitted-handle snapshot; post reports use author DID, collection/NSID, rkey, AT URI where available, and current indexed CID snapshot where available.
  - Report rows should store normalized plain-text optional details after trimming, with empty/whitespace-only text treated as omitted.
  - Forwarding state should store safe status/metadata only, not the full prepared future PDS/Ozone payload.
  - Moderation output rows should store only trusted source DIDs accepted by configuration.
- Changed fields:
  - Post/profile response shapes may gain nullable/omitted `moderation` metadata.
- Migration required:
  - Yes. Add new AppView migrations after current highest migration (`000013` at discovery time; implementer must verify current highest number before writing migrations).
- Backwards compatibility:
  - Existing clients should tolerate omitted `moderation`; Flutter in this repo will be updated to decode/render it.
  - Existing post/profile reads should behave unchanged for subjects without active moderation outputs.
  - Hide/takedown behavior intentionally changes read results for subjects with active moderation outputs.

## 16. UI / API / CLI Impact

- UI:
  - Add report post/profile actions for other users' content/accounts.
  - Add report reason/detail UI with approved localized reason labels, optional plain-text details, in-flight submit disabling, transient success confirmation, validation, and retry/error states.
  - Do not persistently mark content/accounts as reported after a successful report in MVP.
  - Add inline generic warning rendering for warned posts/profiles/authors while keeping content visible.
  - Approved warning copy: post "This post may not follow Craftsky community guidelines."; profile "This profile may not follow Craftsky community guidelines."; author warning on post cards "This author may not follow Craftsky community guidelines."
- API:
  - Add `POST /v1/posts/{did}/{rkey}/reports`.
  - Add `POST /v1/profiles/{handleOrDid}/reports`.
  - Report request `reasonType` allowlist: `harassment`, `hate`, `spam`, `misleading`, `suspected_ai_generated`, `adult_or_graphic`, `impersonation`, `off_topic`, `intellectual_property`, `other`.
  - Accepted report response shape: `{ "reportId": "...", "status": "accepted" }` only.
  - Self-report validation error should use `error: "invalid_report_target"` with a user-safe message such as "You cannot report your own post or profile."
  - Add dev-only + flag/token-gated synthetic moderation endpoint `POST /v1/dev/moderation/ozone-events`.
  - Synthetic endpoint accepts one output per request, with fields equivalent to subject type, subject identity, value, action, optional reason, optional expiry, and trusted/default source DID.
  - Add nullable/omitted moderation metadata to post/profile responses.
  - Apply hide/takedown filtering to relevant read APIs.
- CLI:
  - None required for MVP.
- Background jobs:
  - No live Ozone/PDS jobs in MVP.
  - Internal ingestion service should be reusable by future background Ozone label ingestion.

## 17. Security / Privacy / Permissions

- Authentication:
  - Report endpoints require existing authenticated + device-id middleware.
  - Synthetic endpoint must be dev-only, explicit-flag-gated, and protected by `X-Craftsky-Dev-Moderation-Token` or equivalent dedicated dev token header; it intentionally does not require product auth/device middleware.
  - If `APPVIEW_ENABLE_DEV_MODERATION=true` without non-empty `APPVIEW_DEV_MODERATION_TOKEN`, AppView should fail startup with a clear configuration error.
- Authorization:
  - Users may report other users' posts/profiles.
  - AppView rejects direct self-report requests for the authenticated user's own post/profile.
  - Flutter should not offer report actions for own posts/profiles in this MVP.
  - Normal user auth/device headers are not required for the synthetic route and are not enough to mutate synthetic moderation state without the dev token.
  - Synthetic moderation ingestion must reject untrusted source DIDs.
- Sensitive data:
  - Report details and internal synthetic reasons are private AppView data.
  - Raw reasons must not be displayed in Flutter warning UI.
  - Reports must not be written to user PDS repositories.
  - Full prepared forwarding payloads must not be persisted in MVP.
- Abuse cases:
  - Synthetic endpoint in prod would be dangerous; prevent registration outside dev+flag and require token protection in dev.
  - Repeated reports may be spammy but are allowed in MVP; rate limiting is future work.
  - Hide/takedown filters must not leak hidden content through alternate read surfaces.

## 18. Observability

- Events:
  - No analytics event taxonomy required in MVP.
- Logs:
  - Log report acceptance/failure with request ID, reporter DID, subject type, and subject identity; do not log full report details at high verbosity unless explicitly safe.
  - Log placeholder forwarding status and synthetic moderation ingestion with request ID; do not log dev moderation token values or private report details.
  - Log when a direct post/profile read is suppressed due to moderation, without exposing private report details.
- Metrics:
  - Could add counters for accepted reports, rejected reports, synthetic outputs ingested, and moderated reads filtered, but metrics are not required unless existing infrastructure supports them.
- Alerts:
  - None required for MVP.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Private report details leak through API, logs, or UI. | Reporter privacy and trust harm. | Keep report details out of user responses and Flutter warning copy; review logging. |
| RISK-002 | Synthetic endpoint is reachable in production or reachable without dev token protection. | Attackers or mistakes could hide/warn content. | Require dev environment, explicit config flag, non-empty dev token, trusted source DID allowlist, and route/auth tests. |
| RISK-003 | Hide/takedown filtering is inconsistent across read surfaces. | Hidden content may leak through thread/profile/notification paths. | Acceptance tests must cover all affected surfaces. |
| RISK-004 | Pagination behaves poorly when hidden rows are filtered. | Skips/duplicates or short pages. | Apply filtering in query/store layer and test cursor behavior. |
| RISK-005 | Account-level moderation accidentally hides too much or too little. | User-visible availability errors or leaked posts. | Define account policy narrowly and test profile plus authored posts. |
| RISK-006 | Placeholder forwarder stores too much future payload data. | Sensitive report details retained unnecessarily. | Do not persist the full prepared payload; store only status/metadata and recompute payloads later. |
| RISK-007 | Future live Ozone shape differs from synthetic model. | Integration rework. | Keep ingestion service abstract and limit synthetic values to minimal policy semantics. |
| RISK-008 | Warning labels are interpreted as confirmed wrongdoing. | User trust/reputation harm. | Use generic, neutral localized copy; hide raw reasons. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Reports and moderation-output state should live in AppView Postgres for this MVP. | Persistence design would need to move to Ozone/PDS sooner. |
| ASM-002 | `hide` and `takedown` can share the same user-visible behavior initially. | Requirements/tests would need separate policy matrices. |
| ASM-003 | 404 is acceptable for direct hidden/taken-down posts and profiles. | API contract and Flutter error handling would need different unavailable states. |
| ASM-004 | Generic warning copy is sufficient for MVP. | Flutter/API may need richer label definitions and user preference controls. |
| ASM-005 | Live Ozone ingestion can reuse the synthetic ingestion seam later. | Ingestion abstraction may require refactor when real Ozone lands. |
| ASM-006 | Search moderation is not needed because search is not implemented yet. | If search lands concurrently, moderation policy must be added there too. |
| ASM-007 | A dedicated dev moderation token is acceptable friction for local/test synthetic moderation. | Synthetic endpoint auth might need an alternate local testing helper. |
| ASM-008 | CID snapshots are available or can be nullable for indexed post report audit. | Tests should allow absent CID only where the current store truly lacks it. |

## 21. Open Questions

- None identified for this requirements stage.

## 22. Review Status

Status: Approved direction; requirements draft with annotation and grilling updates
Risk level: High
Review recommended: Required
Reviewer: User plan review via planning gate
Date: 2026-05-30
Notes: The plan was approved with changes and then grilled. Confirmed decisions include profile/account reports, canonical subject storage with post CID snapshots, approved reason taxonomy, optional plain-text details, minimal report response, transient Flutter success feedback, no persisted full forwarding payload, dev synthetic route with env+flag+token gating, trusted source DID ingestion, one synthetic output per request, deterministic moderation precedence, same-source negation semantics, hide/takedown notification omission, server-side self-report rejection, and exact generic inline warning copy.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-05-30-moderation-flow-mvp/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: `BR-001` through `BR-005`
  - Functional: `FR-001` through `FR-027`
  - Non-functional: `NFR-001` through `NFR-005`
  - Rules: `RULE-001` through `RULE-006`
- Suggested test levels:
  - AppView store/migration tests for reports and moderation outputs.
  - AppView handler/route tests for report endpoints and synthetic endpoint gating.
  - AppView read-path tests for timeline, profile posts/comments, direct post, profile, thread/comment, and notification enforcement.
  - Flutter API/repository/provider tests for report submission.
  - Flutter widget tests for report actions, dialog validation, submit/retry states, and warning rendering.
  - Regression tests proving ordinary unmoderated content behaves unchanged.
- Blocking open questions: None identified for test design.
