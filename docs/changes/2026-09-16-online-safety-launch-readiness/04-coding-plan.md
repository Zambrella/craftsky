# Coding Plan: Pre-Launch Online Safety Readiness

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes` for bounded
  coding planning)
- Architecture boundary: index-time scanning is the selected boundary. Approval
  evidence remains required before implementation completion or launch.
- Provider boundary: this plan includes only a protocol-neutral scanner contract,
  deterministic development/test stub, and production readiness rejection. It
  excludes the production IWF adapter and all guessed IWF payloads, credentials,
  status semantics, rate limits, or retention behavior.

## 2. Implementation Strategy

Implement the work as independently testable vertical slices, starting with the
fail-closed image state machine and Tap projection boundary. Reuse the durable Tap
source/projection worker, blocked-dependency wake mechanism, shared moderation
predicates, append-only moderation services, owner-lifecycle inventory, S3-compatible
object-store pattern, route policy catalogue, and Riverpod account-lease fencing.

For image-bearing posts and business records, a new projection gate records the
current source CID and all required blob dependencies in the same transaction before
returning `tap.OutcomeBlocked`. Public queries require the current subject state to
be `clear`; this immediately hides a previously visible revision while its replacement
is pending. Scanner completion wakes the existing projection dependency, which
re-reads the durable current source before projecting. Deletes clean scan dependencies
and run the existing projector without deleting anything from the PDS.

Bluesky profile text does not use the whole-record gate. Avatar and banner candidates
are tracked independently and promoted to the existing serving fields only after a
current-source clear result. Non-clear candidates leave the last cleared value in
place; existing Flutter avatar/banner fallbacks remain the neutral-placeholder path.

Serious-content detections enter a separate private incident subsystem. A match
contains content immediately but creates no report, decision, effect, strike,
suspension, or notice. Authorized human confirmation creates a `systemDetected`
moderation case linked by a safe incident reference and then uses the existing
decision/effect/history/appeal pipeline.

Use staged reversible migrations beginning at `000073`. Do not edit existing
migrations or lexicons. No public PDS schema changes are planned.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Image safety domain | Small internal Go packages with validated constructors | Add closed scan states, protocol-neutral scanner/fetcher contracts, deterministic stub, retry policy, and visibility rules | BR-002, FR-002, FR-003, FR-006 through FR-008, FR-034, NFR-001, NFR-004, NFR-005, RULE-004 | UT-001 through UT-004, UT-014 through UT-016, IT-007, IT-008 |
| Tap projection | Durable source plus transactional projector and blocked dependencies | Decorate only image-bearing projectors; persist dependencies before returning `image_scan_pending`; wake by subject URI | FR-001, FR-004, FR-026, RULE-005 | AT-002, IT-002, IT-003, IT-005 |
| Post visibility | Shared `postVisibleModerationPredicate` reused across read stores | Add current image-safety eligibility to the shared predicate and audit derivatives/caches | FR-004, FR-025, RULE-002 | IT-003, IT-019, REG-002 |
| Business visibility | Transactional event/profile projectors and business store | Gate complete events and image-bearing business extensions; no partial unsafe image response | FR-001, FR-004, FR-025 | AT-002, IT-002, IT-003 |
| Profile images | Profile projector updates serving avatar/banner fields | Track candidates independently and promote only current clear candidates | FR-001, FR-032 | AT-003, UT-017, IT-004 |
| Blob retrieval | Validated federated clients and bounded no-redirect fetch patterns | Fetch candidate bytes from the authoritative PDS, verify CID/type/size, and never hold remote calls inside projection transactions | FR-001, FR-007, NFR-002, NFR-005 | IT-002, IT-006, IT-025 |
| Scan workers/readiness | Leased workers, bounded backoff, safe observability, fail-closed startup checks | Add scanner worker, dead letters, manual retry, safe queue health, and production stub/unconfigured rejection | FR-006 through FR-008, FR-028, NFR-001, NFR-004 | UT-004, UT-014, IT-007, IT-008, IT-021 |
| Restricted incidents | No current equivalent; moderation is private and append-only | Add incident subjects, events, deadlines, assignments, disclosures, reporting references, and safe projections | BR-004, FR-005, FR-009, FR-010, FR-013, FR-021, FR-022 | AT-007, AT-008, AT-013, AT-014, IT-006, IT-010, IT-027, IT-028 |
| Evidence and holds | Private scheduled-media S3 adapter and lifecycle jobs | Add separate restricted object-store config, evidence metadata, reasoned access audit, scoped expiring holds, and deletion | FR-011 through FR-013, FR-029, NFR-002, NFR-006, NFR-007, RULE-009 | AT-009, UT-009, UT-010, UT-019, IT-011 through IT-013, IT-022 |
| Moderation | Cases, reports, append-only events, effects, appeals | Add case origin and safe incident reference; separate allegation/decision/legal/notice mappings; add human-confirmed detection command | FR-009, FR-015, FR-016, FR-020, FR-036, RULE-001 | UT-006 through UT-008, IT-009, IT-014, IT-016, REG-010 |
| Report UI/API | Flat Flutter enum and server allowlist | Add two-step group/allegation model, safe guidance, IP mail routing, and expanded server validation | BR-003, FR-014, FR-017, FR-022, FR-038 | AT-004, UT-006, UT-023, IT-026 |
| External intake | Email appeal link plus correspondence storage; no inbound adapter | Add provider-neutral, metadata-only manual intake and correspondence commands without raw MIME, media, or fabricated DID | FR-018 through FR-020, FR-037, RULE-006 | AT-005, AT-006, UT-018, IT-015, IT-016, IT-024 |
| Age eligibility | Empty onboarding-completion POST and suspension-only policy | Require 16+ declaration/policy version; add separate reversible eligibility state and retained-capability policy | BR-005, FR-023, FR-024, RULE-010 | AT-010, AT-011, UT-012, IT-017, IT-018, REG-008 |
| Video launch gate | Client flag defaults false; live server dependencies remain callable | Make production denial authoritative at UI submit, routes, scheduled publication, and Tap projection | FR-027 | AT-015, UT-022, IT-020, REG-007 |
| Website consent/policies | Static `web/`; consent-aware PostHog loader; only Privacy/Terms routes | Add browser harness, consent network/storage tests, remaining policy routes, and factual publication gate | BR-001, BR-006, FR-030, FR-031 | AT-001, AT-012, AT-017, AT-018, IT-023, UT-021 |
| Lifecycle/privacy | Terminal inventory and component retention workers | Classify every new DID-bearing table, make account deletion hold-aware, enforce controlled-system retention, and leak-test sinks | FR-012, FR-029, NFR-002, RULE-003 | IT-012, IT-022, IT-025, REG-003 |

## 4. Files And Modules

The paths below are implementation targets. Test-first work may split a listed file
when a package becomes clearer, but must not move behavior across the approved
boundaries.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000073_image_safety.{up,down}.sql` | Create | Scan results/jobs/sources, subject requirements/state, profile candidates, checks and indexes | FR-001 through FR-008, FR-026, FR-032, FR-034, NFR-001 | IT-001 through IT-005 |
| `appview/migrations/000074_safety_incidents.{up,down}.sql` | Create | Restricted incidents, subject links, append-only events, assignments, deadlines, authority references | FR-005, FR-009, FR-010, FR-013, FR-021, FR-022 | IT-001, IT-006, IT-010, IT-027, IT-028 |
| `appview/migrations/000075_safety_evidence_holds.{up,down}.sql` | Create | Evidence metadata, reasoned accesses, exports, scoped holds, retention state | FR-011 through FR-013, NFR-002, NFR-006, NFR-007, RULE-009 | IT-001, IT-011 through IT-013, IT-022 |
| `appview/migrations/000076_moderation_safety_taxonomy.{up,down}.sql` | Create | Case origin/reference and additive private reason constraints | FR-009, FR-015, FR-016, FR-036 | IT-001, IT-009, IT-014 |
| `appview/migrations/000077_external_safety_intake.{up,down}.sql` | Create | External provenance, minimized contact reference, subject link, correspondence and outcome | FR-018 through FR-020, FR-037 | IT-001, IT-015, IT-016, IT-024 |
| `appview/migrations/000078_age_eligibility.{up,down}.sql` | Create | Policy acceptance and separate reversible eligibility review/effects/events | FR-023, FR-024, RULE-010 | IT-001, IT-017, IT-018 |
| `appview/migrations/000079_safety_operators.{up,down}.sql` | Create | Operator roles, explicit permissions and token digests | FR-028, NFR-006, RULE-007, RULE-009 | IT-001, IT-013, IT-021 |
| `appview/migrations/000080_safety_retention.{up,down}.sql` | Create | Retention jobs, attempts and append-only completion events | FR-029, NFR-004, NFR-006 | IT-001, IT-022 |
| `appview/internal/db/online_safety_migration_test.go` | Create | Up/down/reapply, constraints, indexes, lifecycle inventory and no public schema assertions | RULE-003 | IT-001, REG-003 |
| `appview/internal/imagesafety/` | Create | State, scanner, stub, extractor, fetcher, store, gate, worker, retry, rescan and health | BR-002, FR-001 through FR-008, FR-026, FR-032 through FR-035 | AT-002, AT-003, UT-001 through UT-005, UT-014 through UT-017, IT-002 through IT-008 |
| `appview/internal/index/transactional_dispatcher.go` and `transactional_projectors.go` | Change | Register/apply image gate around post, business and profile projectors without remote work in projection transaction | FR-001, FR-004, FR-026, RULE-005 | IT-002 through IT-005 |
| `appview/internal/index/craftsky_post.go` | Change | Keep projection behind gate; reject launch video projection without deleting PDS source | FR-001, FR-004, FR-025, FR-027 | IT-002, IT-003, IT-019, IT-020 |
| `appview/internal/index/craftsky_business_event.go` and `craftsky_business_profile.go` | Change | Apply complete-parent/business-extension visibility semantics | FR-001, FR-004, FR-025 | IT-002, IT-003 |
| `appview/internal/index/bluesky_profile.go` | Change | Update text immediately; record/promote independent avatar/banner candidates | FR-001, FR-032 | AT-003, IT-004 |
| `appview/internal/tap/outcome.go` | Change | Add bounded `image_scan_pending` and launch-video-disabled reason codes | FR-006, FR-027, NFR-004 | IT-003, IT-007, IT-020 |
| `appview/internal/api/post_store.go` and stores using `postVisibleModerationPredicate` | Change | Require current image-safe subject state on all post/project queries | FR-025, FR-026 | IT-019, REG-002 |
| `appview/internal/business/store.go`, notifications and derivative stores | Change | Apply event/business visibility and prevent notifications/previews for non-clear subjects | FR-025 | IT-019 |
| `appview/internal/safetyincident/` | Create | Incident service/store, deadlines, reporting, evidence store, authorization, holds and safe read models | BR-004, FR-005, FR-009 through FR-013, FR-021, FR-022, FR-028, FR-029 | AT-007 through AT-009, AT-013, AT-014, AT-016, IT-006, IT-009 through IT-013, IT-021, IT-022, IT-027, IT-028 |
| `appview/internal/retention/` | Create | Retention policy and idempotent worker for AppView-controlled safety rows/objects | FR-011, FR-012, FR-029 | UT-009, UT-019, IT-012, IT-022 |
| `appview/internal/accountdeletion/private_cleanup.go` and `ownerlifecycle/` inventory/cascade files | Change | Preserve only active scoped holds and classify every new DID-bearing table | FR-012, RULE-003 | AT-009, IT-012, REG-003 |
| `appview/internal/moderation/models.go`, `policy.go`, `presentation.go`, `store.go`, `service.go` | Change | Separate mappings, `systemDetected` case origin, human confirmation, safe notice, existing effects/appeals | FR-009, FR-015, FR-016, FR-020, FR-036, RULE-001 | UT-006 through UT-008, IT-009, IT-014, IT-016 |
| `appview/internal/safetyintake/` | Create | Provider-neutral external-email normalization, storage and safe correspondence | FR-018 through FR-020, FR-037 | UT-018, IT-015, IT-016, IT-024 |
| `appview/internal/api/report_request.go` and report handler/store tests | Change | Validate expanded precise allegation identifiers while retaining `reasonType` wire key | FR-014 through FR-016 | UT-006, IT-014, IT-026 |
| `appview/internal/api/safety_admin.go` | Create | Queue/status, incident, assignment, human confirmation, hold, retry and intake handlers | FR-009, FR-018, FR-028, NFR-004, RULE-009 | IT-009, IT-013, IT-015, IT-021 |
| `appview/internal/routes/policy.go`, `dependencies.go`, route registrars | Change | Catalogue new admin/eligibility routes, body/rate/auth policy and retained eligibility capabilities | FR-024, FR-028, RULE-009 | IT-013, IT-018, REG-009 |
| `appview/internal/api/onboarding.go` and onboarding store | Change | Strict camelCase acceptance body and persisted policy version/time | FR-023, RULE-010 | IT-017, REG-008 |
| `appview/internal/eligibility/` and middleware | Create | Owner-safe state, evidence review commands, reversible restriction, retained route enforcement | FR-024, RULE-010 | UT-012, IT-018 |
| `appview/internal/app/config.go`, `deps.go`, `deps_tap.go`, `routes_adapter.go` | Change | Validate/wire scanner, workers, restricted storage, operator auth, retention and video gate | FR-006 through FR-008, FR-027 through FR-029 | IT-007, IT-008, IT-020 through IT-022 |
| `appview/cmd/appview/main.go` | Change | Start/stop new workers and expose readiness without sensitive detail | FR-006, FR-008, NFR-004 | IT-007, IT-008, IT-021 |
| `app/lib/moderation/models/report_reason.dart` and a new group/destination model | Change / Create | Separate groups, precise allegations, guidance and route destination | FR-014, FR-017, FR-022, FR-038 | AT-004, UT-023, IT-026 |
| `app/lib/moderation/widgets/report_subject_sheet.dart` | Change | Two-step accessible report flow and specialist branches | BR-003, FR-014, FR-017, FR-022, FR-038, NFR-003 | AT-004, AT-019, IT-026 |
| `app/lib/onboarding/` models, repository/client, flow provider and guidelines widget | Change / Create | Collect only explicit threshold acceptance and configured policy version | FR-023, RULE-010, NFR-003 | AT-010, AT-019, IT-017, REG-008 |
| `app/lib/auth/`, `app/lib/router/`, and new eligibility page/provider | Change / Create | Account-keyed safe eligibility status and retained-capability navigation | FR-024, RULE-010, NFR-003 | AT-011, AT-019, IT-018 |
| Flutter moderation history/presentation | Change | Add approved safe reasons and eligibility appeal state; never expose incident internals | FR-020, FR-036, NFR-002 | AT-006, IT-009, IT-016, IT-025 |
| Flutter video provider/composers | Change | Force release-disabled UI and recheck gate at submit boundary | FR-027 | AT-015, IT-020, REG-007 |
| `app/test/feed/providers/create_post_provider_test.dart` | Change | Preserve author-only synthetic post behavior and refresh authority | FR-033 | REG-001 |
| `web/package.json`, browser config, and `web/test/analytics-consent.spec.*` | Create | Browser network/storage coverage for consent, denial and DNT | FR-030 | AT-017, IT-023 |
| `web/main.js` | Change | Keep PostHog entirely unloaded before consent and align approved capture behavior | FR-030 | AT-017, IT-023 |
| `web/` policy HTML routes and publication manifest | Create / Change | Publish all five approved policies in one artifact with version/effective metadata | BR-005, BR-006 | AT-012, MAN-010, MAN-014 |
| `scripts/online-safety-readiness` and focused tests | Create | Fail release on open P0, missing evidence, unsupported policy claim, unsafe scanner/video config, or incomplete policy artifact | BR-001, BR-006, FR-008, FR-027 | AT-001, AT-012, UT-021 |
| `justfile` and `scripts/appview-check` | Change | Add focused safety checks to release-equivalent verification without weakening existing gates | G-006, BR-001 | AT-001, IT-001, IT-008 |

## 5. Services, Interfaces, And Data Flow

### 5.1 Image Safety Contracts

```text
type ScanState string
const pending, clear, match, unavailable, error

type ScanKey struct {
  BlobCID       syntax.CID
  ScannerID     string
  PolicyVersion string
  CorpusVersion string
}

type Scanner interface {
  Scan(context.Context, ScanInput) (ScanResult, error)
}

type BlobFetcher interface {
  Fetch(context.Context, BlobSource) (BoundedImage, error)
}

type Store interface {
  ReconcileSubjectTx(context.Context, pgx.Tx, SubjectRevision) (SubjectState, error)
  ClaimJobs(context.Context, ClaimOptions) ([]Job, error)
  CompleteJob(context.Context, Completion) ([]tap.Dependency, error)
  RetryDeadLetter(context.Context, JobID, Actor) error
}

type Projector interface {
  Project(context.Context, pgx.Tx, tap.Event) (tap.Outcome, error)
}

type Gate struct {
  Extractor Extractor
  Store     Store
  Next      Projector
}
```

The local `Projector` interface is structurally satisfied by index projectors and
keeps `imagesafety` from importing `internal/index`, avoiding an import cycle when
the index package constructs the wrappers.

`ScanInput` contains only the immutable CID, validated image metadata, and a bounded
reader/stream. It contains no IWF request fields. The deterministic stub maps explicit
benign fixture CIDs to outcomes; an unknown fixture returns `error`, never `clear`.

The cache key includes opaque scanner, policy, and corpus versions. Time alone never
invalidates a clear result. An authorized targeted rescan creates a new current scan
requirement and returns affected subjects to non-visible state until it completes.

### 5.2 Image Persistence Shape

Migration `000073` creates private tables with closed constraints:

- `image_scan_results`: cache key, state, safe provider-neutral reference,
  completion time, current version, and restricted integrity metadata reference.
- `image_scan_jobs`: queued/leased/dead-letter state, lease token/expiry, bounded
  attempts, next attempt, and safe error category.
- `image_blob_sources`: CID, source DID/URI/CID, declared MIME/size, and observed time.
- `image_subject_requirements`: subject URI, subject kind, current source CID, image
  slot, blob CID, and scan-result key.
- `image_subject_states`: one current source revision and visibility state per gated
  subject, used by shared read predicates and replay checks.
- `profile_image_candidates`: DID, avatar/banner slot, current source CID, candidate
  CID/MIME, and result key. Existing profile avatar/banner columns remain the
  last-cleared serving values.
- `image_scan_events`: append-only, bounded operational transitions with no bytes,
  raw provider payload, contact data, or unrestricted error text.

All image-bearing post/business projectors write a subject-state row, including a
clear state for revisions with no images. This avoids treating a missing state row as
implicit clearance. Existing development data may be reset; no production backfill
compatibility is required.

### 5.3 Tap-To-Visibility Flow

```text
Tap event
  -> existing durable source row and projection job
  -> lifecycle checks in TransactionalDispatcher
  -> image Gate inside worker-owned transaction
       delete: remove requirements/state, then run existing delete projector
       create/update: decode generated record type and reconcile current requirements
       all current results clear: mark subject clear, run existing projector
       otherwise: mark subject blocked, enqueue missing work, return OutcomeBlocked
  -> projection job commits blocked dependency {kind: image_subject_uri, key: AT URI}

Scanner worker
  -> claim with lease
  -> fetch bytes outside projection transaction
  -> validate origin, redirects, limits, MIME/decode and CID
  -> deterministic stub scan
  -> commit result/retry/dead-letter and append event
  -> on terminal result, WakeDependency(image_subject_uri)
  -> projection worker reloads current durable source and reevaluates all blobs
```

The worker never projects the source it scanned directly. Waking the existing job
ensures a stale scan cannot publish an older record revision. A source update with a
new blob first marks the current subject blocked, hiding the old serving row through
the query predicate. A source delete removes public eligibility and avoids unnecessary
future work while retaining only approved incident/evidence metadata.

Post extraction covers generated CraftSky image embeds and Bluesky external-embed
thumbnails. Business extraction covers event/product images. Profile extraction
covers standard Bluesky avatar/banner blobs. Video is not passed to the image scanner.

### 5.4 Visibility Predicates

Add a small SQL predicate/function that requires a matching current
`image_subject_states` row with the same subject URI/source CID and state `clear`.
Compose it into `postVisibleModerationPredicate`, business event reads, notification
eligibility/writes, direct lookups, search, pins, threads, comments/replies, quotes,
reposts, previews, and any cache publication path. Preserve current relationship,
owner-lifecycle, and moderation predicates rather than replacing them.

Profile reads continue returning only the existing last-cleared avatar/banner fields.
Candidate state and URLs never enter member-facing responses.

### 5.5 Detection And Enforcement Flow

```text
scan result match
  -> keep image_subject_states blocked
  -> create/reuse restricted incident by detection key
  -> link every affected subject revision
  -> append detected event
  -> no moderation report/case/effect/notice

authorized safety review
  -> metadata-only incident detail by default
  -> confirm or reject detection through idempotent command
  -> confirmed: create systemDetected moderation case with safe incident reference
  -> existing moderation decision/effect/history/appeal pipeline
  -> owner notice only after human violation decision
```

The moderation case gains `origin` (`userReport` or `systemDetected`) and a nullable
safe incident reference required for `systemDetected`. Add a dedicated store method;
do not reuse `AttachAcceptedReportTx` or create a report row.

User allegation, internal decision reason, legal classification, incident class,
sanction eligibility, and owner-safe presentation remain separate Go types/mappings.
The confirmed-match owner presentation is exactly `Child safety violation` plus the
action/appeal route, with no provider, match, evidence, reporting, or investigative
detail.

### 5.6 Restricted Evidence, Holds, And Retention

```text
type EvidenceStore interface {
  Put(context.Context, RestrictedObject) (ObjectRef, error)
  Delete(context.Context, ObjectRef) error
  // No public URL or generic Get method.
}

type EvidenceService interface {
  Preserve(context.Context, PreserveCommand) (EvidenceRef, error)
  Access(context.Context, AccessCommand) (AuditedStream, error)
  Export(context.Context, ExportCommand) (MinimizedExport, error)
}
```

The default match path stores metadata only. Byte preservation is a distinct,
permission-checked command requiring an approved procedure reason. The S3-compatible
adapter has separate endpoint/bucket/region/credentials/KMS configuration from
scheduled media and never produces public/CDN/presigned URLs. MinIO proves object
isolation and lifecycle logic; MAN-003 proves actual Frankfurt/IAM/KMS/logging posture.

Holds require basis, explicit incident/evidence scope, approver, creation time, and
mandatory expiry. Invalid, owner-wide, or indefinite holds fail validation. Account
deletion removes unrelated private data and preserves only explicitly covered rows
and objects. The retention worker deletes held data after expiry and writes
non-sensitive completion events.

### 5.7 External Intake

The bounded implementation provides an authenticated admin command, not a mailbox
listener. It accepts provider message reference, received time, canonical CraftSky
URL/AT URI, allegation/complaint/appeal kind, urgency, minimized sender reference,
and attachment handling status. It rejects raw MIME, attachment bytes, fake/member
reporter DID, and unrestricted contact text. Repeated messages append correspondence
to the canonical item instead of creating independent guilt signals.

A future mailbox adapter may call this service only after MAN-006 proves attachment
handling. Its provider protocol is not part of this plan.

### 5.8 Age Eligibility Contracts

`GET /v1/onboarding/status` adds the server-configured
`requiredPolicyVersion`. `POST /v1/onboarding/completion` changes from
`BodyNoBody` to strict JSON:

```text
{
  "meetsMinimumAge": true,
  "policyVersion": "<server-configured current version>"
}
```

Unknown fields, false/missing declaration, and missing/stale policy version fail with
the standard camelCase error envelope. The accepted version and timestamp are stored
against the authenticated DID. No DOB, age band, estimate, or derived age field is
accepted or persisted.

Eligibility is separate from suspension. A private service records specific evidence
references, reviewer, state, reversible restriction, appeal, and append-only events.
`GET /v1/account/eligibility` exposes only owner-safe status and appeal guidance.
Route policy gains an explicit eligibility class/retained allowlist mirroring, but not
overloading, suspension. Server enforcement remains authoritative.

### 5.9 Operator Authorization

Extend the moderator authentication context from source/actor identity to explicit
permissions resolved from active operator/token records. Store only token digests.
Route-level `AccessModerator` remains the first boundary; safety services enforce
permissions such as `incident.readSafe`, `incident.confirm`, `evidence.access`,
`hold.manage`, `authority.report`, `disclosure.approve`, and `scan.retry`.

Every state-changing admin command requires an idempotency key, expected revision
where applicable, actor, reason, source system, and append-only event. Ordinary
moderators receive safe metadata only. Operator bootstrap is an operational CLI or
deployment secret flow that never prints/stores plaintext tokens in logs.

## 6. State, Providers, Controllers, Or DI

### AppView Wiring

Add narrow dependency groups rather than expanding one unstructured constructor:

```text
imageSafetyDependencies
  scanner: deterministic stub or unconfigured
  fetcher: authoritative PDS blob fetcher
  store, gate, worker, health

safetyDependencies
  incidents, evidence, holds, retention
  intake, operator authorization, admin handlers
```

`newTapDependencies` receives the already-built image gate and registers wrapped
projectors. `cmd/appview/main.go` starts scanner/retention/deadline workers using the
same lifecycle and shutdown pattern as existing workers. Route dependencies expose
only handler-facing services, not scanner credentials or evidence object-store
internals.

Configuration adds bounded scanner mode/identity/policy/corpus, worker budgets,
alert thresholds, restricted S3 settings, retention intervals, current policy
version, and production video state. Validated rules are:

- Dev/test may select `stub`; production may not.
- Production remains unready while the future real scanner is unconfigured.
- Stub fixture configuration cannot be interpreted as production clearance.
- Evidence storage configuration is separate from scheduled media.
- Unknown/empty policy identity fails closed.
- Video is always disabled in production for this launch.

### Flutter Provider Graph

Preserve generated Riverpod and account-lease patterns:

```text
sessionRegistryProvider
  -> active account lease
  -> onboardingFlowProvider(lease)
       -> onboarding repository/client
       -> immutable declaration + configured policy version
  -> accountEligibilityProvider(lease)
       -> eligibility repository/client
       -> router redirect / retained-capability shell

report sheet local state
  category -> allegation -> destination/guidance
  inApp -> existing report provider
  intellectualProperty -> moderationMailLauncherProvider
```

Use a `Notifier` only for the local two-step report state if widget state becomes
unwieldy; do not add a global provider for static taxonomy. Keep report mutations in
the existing account-scoped providers and preserve single-flight behavior.

Add the eligibility status to exact-lease initialization so a late response from
account A cannot redirect account B. The client does not infer eligibility from a
generic 403.

Profile widgets need no scanner provider or unsafe candidate model. The AppView
continues returning only the safe selected URL. The synthetic-post providers receive
regression tests but no production change.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Flutter Report Flow

Keep `showPostReportSheet`/`showProfileReportSheet` on the root full-screen route.
Inside `ReportSubjectSheet`:

1. Show broad, understandable groups.
2. Move focus to the precise allegation heading after selection.
3. Show category-specific guidance before submission controls.
4. Submit only destinations marked `inApp` through existing report providers.
5. Route intellectual property to the dedicated copyright/trade-mark email flow.
6. Permit a CraftSky threat report after emergency-services-first guidance.
7. Preserve details limit, loading/error recovery, root navigation, and account
   operation fencing.

All strings originate in ARB files; regenerate localization output rather than
editing generated files.

### Onboarding And Eligibility

Add a full-sentence checkbox to the guidelines step for explicit 16+ confirmation.
Finish remains disabled or shows inline recoverable validation until checked. The
screen announces checked/error/busy states and remains scrollable at 200% text.

Add an owner-safe eligibility page/shell reached from router state, not generic HTTP
failure. It presents explanation and appeal guidance while exposing only the retained
routes for sign-out, reports, appeals/standing, privacy actions, content removal,
account deletion, blocks, and mutes. Do not reuse suspension copy or collect age.

### Moderation Presentation

Extend safe reason parsing and history presentation with forward-compatible unknown
handling. Owner surfaces may show `Child safety violation`, action, effective time,
and appeal route only after adjudication. Incident IDs, provider references, hashes,
evidence, authority-report status, and legal/internal classifications remain absent.

### Video

Force the release provider false regardless of Dart defines, hide affordances, and
recheck eligibility at both post/project submit boundaries so restored/injected video
cannot bypass UI hiding. AppView denies authorization, limits/publication operations
and keeps externally created video records out of launch serving projections.

### Admin HTTP Routes

Add private `/v1/admin/safety/*` routes through the route catalogue and existing
moderator middleware. Initial routes are status/queue, incident safe detail,
assignment, human confirmation/case creation, hold management, scan retry, external
intake/correspondence, and reasoned evidence access. Bodies are strict camelCase;
lists use opaque cursors; errors use `{error, message, requestId}`.

No public unauthenticated reporting API or public admin UI is added.

### Public Website And Policies

Add a small Playwright browser harness scoped to `web/`. Test every public HTML route
with clean storage, denial, grant, and DNT. No PostHog script/request/storage may
exist before valid consent; grant may load only the approved host.

Create all five policy HTML routes in one static artifact and a machine-checkable
publication manifest containing route, source document, version, effective date, and
approved evidence references. The repository gate validates completeness and stale
placeholders; actual Cloudflare atomic deployment/rollback remains external evidence.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| `pending` | Subject hidden; one current job; safe pending metrics only | FR-003, FR-004 | UT-001, UT-002, IT-003 |
| `clear` | Reproject current source; parent visible only when every requirement is current/clear | FR-003, FR-004 | UT-001, UT-002, IT-003 |
| `match` | Keep hidden; create/reuse restricted incident; no report/sanction/notice | FR-005, RULE-001 | AT-008, IT-006 |
| `unavailable` or `error` | Keep hidden; bounded retry; dead-letter after exhaustion; manual retry by permission | FR-006, NFR-004 | UT-004, IT-007 |
| Unknown scan state/fixture | Reject or record error; never default to clear | FR-003, FR-007 | UT-001, UT-015 |
| Shared blob | Reuse current result; retain separate subject visibility/incident links | FR-002, FR-005 | UT-003, UT-005 |
| Policy/corpus change | New requirement only when approved policy requests it; no silent stale reuse | FR-002, FR-034 | UT-003, UT-016 |
| Worker restart/lease loss | Idempotent reclaim; no duplicate result/event/effect | NFR-001 | IT-005, IT-007 |
| Source changes during scan | Old revision remains hidden; wake reloads current durable source | FR-026 | IT-005 |
| Source delete during scan | Remove serving eligibility/dependencies; cancel avoidable job; retain approved incident data only | FR-011, FR-026 | IT-005, IT-012 |
| New profile image non-clear | Keep last clear slot or neutral placeholder; update safe text fields | FR-032 | AT-003, IT-004 |
| Stub/unconfigured production scanner | Startup or readiness fails; no result can become production-clear | FR-008, RULE-004 | UT-014, IT-008 |
| Restricted object-store failure | No partial metadata claims; retry safely; never fall back to scheduled/public storage | FR-011, NFR-007 | IT-011 |
| Unauthorized evidence/hold/report request | Deny before object access/state mutation; write safe denied-access audit/alert | RULE-009 | UT-010, IT-013 |
| Expired hold | Retention worker deletes covered row/object and records safe completion | FR-011, FR-012 | AT-009, IT-012, IT-022 |
| External email with media status | Reject raw bytes; require provider reference and safe handling status | FR-037 | AT-005, IT-024 |
| Duplicate external correspondence | Append to canonical item; do not increase guilt/sanction state | FR-018, RULE-001 | IT-015, REG-006 |
| Missing/false age declaration | Keep onboarding incomplete with field error; store no DOB/band | FR-023, RULE-010 | AT-010, IT-017 |
| Eligibility-restricted account | Deny ordinary product actions while retained safety/account actions work | FR-024 | AT-011, IT-018 |
| Stale account response in Flutter | Lease check discards result; no cross-account redirect/state | FR-024, NFR-003 | IT-018 and Flutter account-isolation tests |
| Video attempt in production | UI blocks and server returns standard unavailable error; no serving projection | FR-027 | AT-015, IT-020 |
| No analytics consent or DNT | Do not load script, send requests, or write PostHog identifiers | FR-030 | AT-017, IT-023 |
| Open P0 or unsupported policy claim | Readiness/publication check fails with non-sensitive evidence location | BR-001, BR-006 | AT-001, AT-012, UT-021 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `appview/internal/imagesafety/state_test.go` | Table of five states plus unknown | No closed scan-state type or visibility rule exists |
| 2 | UT-002, UT-003 | `imagesafety/visibility_test.go`, `cache_test.go` | Multi-blob and versioned CID tables | No all-clear rule or versioned cache key exists |
| 3 | UT-004, UT-014 through UT-016 | `imagesafety/retry_test.go`, `config_test.go`, `scanner_test.go`, `rescan_test.go` | Fake clock and deterministic benign fixtures | No bounded retry, stub guard, neutral scanner, or rescan policy exists |
| 4 | IT-001 | `internal/db/online_safety_migration_test.go` | Disposable PostgreSQL migration cycle | Migration 73 and later private schema do not exist |
| 5 | AT-002, IT-003 | `internal/index/image_scan_acceptance_test.go` | Post with mixed scan results | Existing projector exposes image post without subject scan state |
| 6 | AT-003, UT-017, IT-002, IT-004 | Existing index suites plus image extraction/profile-policy tests | Generated post/business/profile/external-embed fixtures | In-scope blobs do not enter one scan workflow; profiles replace immediately |
| 7 | IT-005, REG-001 | Tap replay and Flutter synthetic-post suites | Stable/changed CID, delete, restart and local-cache fixtures | Replay/current-revision safety is not connected; regression assertion absent |
| 8 | IT-007, IT-008, IT-021 | Scanner worker/config/admin status tests | Failing stub, fake clock, production config | No retries, dead letters, readiness rejection, or safe queue status exists |
| 9 | UT-005, IT-006, AT-008 | `safetyincident/detection*_test.go` | Synthetic match and shared blob | Match cannot create restricted incident; auto-sanction invariant unproved |
| 10 | UT-010, IT-011, IT-013 | Evidence/authorization tests with MinIO | Role fixtures and restricted canaries | Separate object store, reasoned audit and permissions do not exist |
| 11 | UT-009, UT-019, IT-012, IT-022, AT-009 | Retention/hold/deletion tests | No/active/expired/out-of-scope holds | Account deletion and retention ignore holds |
| 12 | UT-006 through UT-008, IT-009, IT-014, REG-010 | Moderation suites | Every mapping, signal-only input and confirmed incident | No system-detected origin or separated child-safety presentation exists |
| 13 | AT-007, IT-010 | CSEA workflow service tests | Synthetic metadata-only incident | Reporting/supplement/reference lifecycle does not exist |
| 14 | AT-013, UT-011, IT-027 | Intimate-image workflow tests | Configurable target and complaint fixtures | Required fields/deadline service do not exist; unconfigured target must fail closed |
| 15 | AT-014, IT-028 | Threat/authority workflow tests | Synthetic genuine/fraudulent requests | Verification, legal approval and minimized export workflow does not exist |
| 16 | UT-006, UT-023, IT-026, AT-004 | Flutter/server report suites | Every group/reason/destination/guidance | Current flat ten-reason flow and allowlist fail expected taxonomy |
| 17 | UT-018, IT-015, IT-024, AT-005 | `safetyintake` tests | Metadata-only and media-status email fixtures | External provenance/correspondence storage does not exist |
| 18 | IT-016, AT-006 | Existing appeal suites plus intake link | Owner/wrong-owner and correspondence fixtures | External intake cannot bind to existing appeal chronology |
| 19 | IT-017, AT-010, REG-008 | AppView onboarding and Flutter widget/repository tests | Strict declaration/version body | Current endpoint accepts no body and stores no policy acceptance |
| 20 | UT-012, IT-018, AT-011 | Eligibility service/middleware/router tests | Allowed/prohibited evidence and route matrix | No separate reversible eligibility state or retained policy exists |
| 21 | UT-022, IT-020, AT-015, REG-007 | App/server video gate tests | Production config plus injected/restored video | Client define and live server routes can still enable publication |
| 22 | IT-019, REG-002, REG-003, REG-005, REG-006 | Cross-package visibility/privacy regression suite | Every surface, suspension, duplicate signal and PDS spy | Some derivatives may expose blocked content or private state |
| 23 | UT-013, UT-020, IT-025, AT-016 | Redaction/priority units, sink canaries and admin queue tests | Restricted canaries across logs/errors/push/APIs | Leak/priority/cover assertions do not exist |
| 24 | IT-023, AT-017 | `web/` browser suite | Clean, denied, granted and DNT contexts | No real browser/network test harness exists |
| 25 | UT-021, AT-001, AT-012, REG-004, REG-009 | Readiness/policy/route catalogue tests | Open/closed P0 and policy manifests | No online-safety release gate or complete policy artifact exists |
| 26 | AT-018, AT-019 and MAN-001 through MAN-014 | Evidence pack/manual gates | Approved synthetic exercises | External operational evidence remains absent; these do not block bounded code iterations |

Focused red-green commands begin with:

```text
cd appview && go test ./internal/imagesafety -run TestScanState
```

Subsequent AppView integration slices use `just test` with the Compose PostgreSQL and
MinIO stack. Flutter slices use `just app-test <path>` and `just app-analyze`.
Browser commands are added with the `web/` harness. Final code evidence uses
`just appview-check`, the full Flutter suite/analyzer, and the online-safety readiness
gate.

## 10. Sequencing And Guardrails

- First TDD step: write `UT-001` for the closed scan vocabulary and all visibility
  decisions; implement only enough `imagesafety.State` behavior to pass it.
- Dependencies between work items: migrations and state model precede Tap gating;
  gating precedes incidents; incidents precede evidence/moderation confirmation;
  server wire contracts precede Flutter clients; hold schema precedes account-deletion
  changes; operator permissions precede restricted admin routes.
- Use one migration per coherent rollback boundary. Run each migration up/down/up and
  update lifecycle inventory tests with the same slice.
- Keep remote blob fetching and scanning outside projection transactions. Only state
  reconciliation and projector mutation occur under the worker-owned transaction.
- Parse typed atproto identifiers at HTTP/Tap boundaries and pass `syntax.*` types
  internally. Do not introduce plain strings for semantic DIDs, AT URIs, NSIDs, or
  record keys.
- Use generated lexicon-derived record types for extraction. No hand-rolled record
  structs and no lexicon changes.
- A scan signal never calls moderation effect code. Only the explicit authorized
  human-confirmation path may create a `systemDetected` case.
- Never store or log image bytes, raw provider payloads, scanner credentials,
  perceptual hashes, contact details, authority credentials, or internal judgments
  in ordinary logs/analytics/Sentry/push/user APIs.
- Never use real illegal imagery in development or tests. Use benign deterministic
  fixtures and canary strings only.
- Never delete a source PDS record for scanning, moderation, retention, or legal
  hold behavior.
- Do not add automatic periodic rescanning. Targeted rescan requires explicit
  authorization and audit.
- Do not expose profile candidate URLs or scan states to Flutter.
- Do not use report volume, automated matches, appearance, writing, interests,
  location, or inferred age as a violation/eligibility decision.
- Do not add Ozone, a public reporting API, video scanning, facial-age estimation,
  or production IWF request code.
- Production readiness remains false until a separately reviewed real scanner exists;
  bounded implementation passing tests is not launch approval.
- Public policy text changes only after corresponding controls and evidence pass;
  no placeholder operator details or unsupported present-tense claims may publish.
- Out of scope: production IWF adapter, mailbox-provider automation, Cloudflare
  deployment configuration, actual AWS IAM/KMS provisioning, worldwide legal advice,
  regulator registration, staffing appointments/training, and manual/tabletop signoff.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking for plan; blocking completion/launch | Documentary reference for index-time containment approval is pending. | Conditions could require returning to requirements/test design. | Accountable owner supplies reviewer/date/reference/conditions; do not close DR-001/GAP-002 before verification. |
| CPQ-002 | Blocking production adapter/completion/launch | IWF contract, eligibility and benign test vectors are unavailable. | Production scanner cannot be designed or accepted. | Exclude adapter entirely; re-enter document review when approved documentation arrives. |
| CPQ-003 | Blocking affected production workflow/launch | Intimate-image response target is unapproved. | No production deadline value can be configured or accepted. | Implement required configuration with no permissive default; obtain specialist value before enabling workflow. |
| CPQ-004 | Non-blocking for server contract; blocking onboarding release | Authoritative policy-version value/format is not yet published. | Client/server acceptance cannot use a final production value. | Make the server current version explicit configuration, return it from onboarding status, and have Flutter round-trip that value; set it only after atomic policy approval. |
| CPQ-005 | Blocking restricted operations/launch | Safety administrator/helpers and exact assignments are not named/trained. | Production permissions and coverage cannot be accepted. | Implement roles/permissions generically; provision only after approved authorization matrix and training. |
| CPQ-006 | Blocking external-email launch route | Mailbox provider safe attachment behavior is unknown. | Provider automation could preview/forward unsafe media. | Limit code to metadata-only manual intake; select provider and pass MAN-006 before adapter work. |
| CPQ-007 | Blocking evidence deployment/launch | AWS Frankfurt bucket/IAM/KMS/logging/lifecycle are not provisioned or audited. | MinIO tests cannot establish production isolation. | Keep separate strict config; complete MAN-003 before production evidence bytes are permitted. |
| CPQ-008 | Non-blocking for repository work; blocking policy publication | Cloudflare deployment/rollback is external and no repository CI config exists. | Atomic publication cannot be proven solely by tests. | Build one complete static artifact/manifest; record external deployment evidence in P0 gate. |
| CPQ-009 | Blocking AC-022/launch | Retention periods and deletion evidence for email, Sentry, PostHog, backups and local drafts remain external. | Unified retention cannot pass end to end. | Implement AppView-controlled classes; assign external owners/mechanisms before claiming AC-022. |
| CPQ-010 | Blocking launch | Operator/controller details, DPO position and worldwide jurisdiction review remain unresolved. | Policies and territorial controls cannot publish accurately. | Keep release gate failed until approved details/evidence replace placeholders. |
| CPQ-011 | Non-blocking | Existing development image rows have no scan state. | Strict predicate would hide/reset local data after migration. | Because there are no production users, reset development data or migrate every existing supported subject to blocked; do not grandfather as clear. |
| CPQ-012 | Non-blocking | The current website analytics events differ from an older landing-page design. | Broader analytics behavior may be inconsistent although pre-consent requirement is clear. | Scope this plan to AC-030; separately approve event inventory before changing post-consent collection beyond what the requirement demands. |

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-001` in
  `appview/internal/imagesafety/state_test.go`.
- Focused command:
  `cd appview && go test ./internal/imagesafety -run TestScanState`.
- First vertical slice: migration `000073`, scan states/cache/retry/config stub, Tap
  subject gating, and `IT-003` fail-closed post visibility.
- Provider constraint: deterministic stub only; no production IWF adapter or guessed
  IWF contract.
- Completion constraint: passing bounded implementation does not close DR-001 through
  DR-008, GAP-001 through GAP-012, or authorize public launch.
- Stage verification: focused Go tests during red-green-refactor, `just test` for
  PostgreSQL/MinIO integration, `just appview-check`, targeted then full Flutter tests,
  `just app-analyze`, browser consent tests, and the new readiness gate.
