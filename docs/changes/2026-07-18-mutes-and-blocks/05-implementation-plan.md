# TDD Implementation Plan: Account Mutes And Blocks

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Approval And Risk

- Implementation was explicitly authorized by invoking `implement-tdd` on 2026-07-19.
- Risk is High because the work changes privacy, membership, moderation, delivery, and public AT Protocol record boundaries.
- The approved guardrails remain binding: private mutes stay in AppView Postgres, public blocks use `app.bsky.graph.block`, Tap is the only writer of `atproto_blocks`, and `craftsky_profiles EXISTS` is the only membership predicate.
- Commits and pushes are not authorized for this stage.

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update a focused failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated after each loop.
- Preserve unrelated worktree changes.
- Do not edit `lexicon/` or generated Craftsky lexicon files.

## Test Order

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 1 | IT-035 | FR-002, FR-004, FR-029, NFR-004 | AC-008, AC-012, AC-042, AC-053 | Fails: migration 000023 and relationship tables do not exist |
| 2 | IT-002 | FR-002 | AC-008, AC-009 | Fails: no owner-scoped mute store exists |
| 3 | UT-001, UT-014, UT-016 | NFR-002, RULE-001–RULE-003, BR-004, FR-028–FR-029 | AC-040, AC-045–AC-047, AC-051–AC-053 | Fails: no shared policy, membership decision, or lifecycle distinction exists |
| 4 | UT-002, IT-001, IT-003–IT-004 | FR-001, BR-003, NFR-001, RULE-003–RULE-004, RULE-007 | AC-005–AC-007, AC-047, AC-050 | Fails: non-members can be hydrated and relationship routes do not exist |
| 5 | UT-009, IT-011, IT-023 | FR-008, FR-026, NFR-003 | AC-016, AC-038, AC-041 | Fails: relationship lists, cursors, and routes do not exist |
| 6 | UT-010, IT-007, IT-036 | BR-002, FR-003–FR-004, FR-006, FR-028 | AC-004, AC-010–AC-012, AC-014, AC-051 | Fails: block collection is not subscribed, dispatched, or indexed |
| 7 | IT-005–IT-006, IT-009, IT-032 | BR-002, FR-003, FR-006, FR-025, RULE-002 | AC-003, AC-010–AC-011, AC-014, AC-036, AC-046 | Fails: no PDS block mutation or confirmed-overlay reconciliation exists |
| 8 | IT-008 | FR-005 | AC-013, AC-059 | Fails: join/rejoin does not request and exercise block backfill |
| 9 | IT-025, REG-005 | FR-005, FR-020, FR-028–FR-029 | AC-013, AC-051–AC-053, AC-059 | Fails: membership loss/rejoin is not uniformly enforced |
| 10 | UT-003–UT-004, IT-010 | FR-007, FR-016 | AC-015, AC-026 | Fails: profile responses have no relationship state or blocked shell |
| 11 | IT-024, AT-015 | BR-004, FR-028 | AC-051 | Fails: user-facing account targets still expose resolvable non-members |
| 12 | IT-021–IT-022, REG-002, REG-006 | FR-020–FR-021, RULE-005, RULE-008 | AC-031–AC-032, AC-048, AC-058, AC-060 | Fails: graph and search surfaces ignore membership/block policy |
| 13 | IT-012–IT-013, IT-029–IT-030 | FR-009, FR-014, NFR-003–NFR-004 | AC-017, AC-023, AC-041–AC-042 | Fails: relationship filtering is absent from read selection and query plans |
| 14 | UT-005–UT-006, IT-014–IT-015, IT-027, IT-033 | FR-010–FR-015, FR-025, FR-030, RULE-006 | AC-018–AC-019, AC-023–AC-025, AC-037, AC-049, AC-056 | Fails: responses cannot express safe mute/block placeholders or references |
| 15 | UT-008, IT-016–IT-017, REG-001, REG-004 | FR-012, FR-017–FR-018, RULE-001, RULE-007 | AC-020, AC-027–AC-028, AC-045, AC-050 | Fails: directed writes ignore blocks and exception rules are absent |
| 16 | IT-020, IT-034, REG-003 | FR-020, FR-031, RULE-003, RULE-005, RULE-008 | AC-031, AC-047–AC-048, AC-057–AC-058 | Fails: restoration and non-destructive policy guarantees are unproved |
| 17 | UT-007, IT-018 | FR-013, FR-019, FR-031 | AC-021, AC-029, AC-057 | Fails: notification ingestion and eligibility ignore relationships |
| 18 | IT-019–IT-020, AT-006 | FR-013, FR-019–FR-020, FR-031, RULE-005 | AC-022, AC-030–AC-031, AC-048, AC-057 | Fails: list/newness/badge/push delivery races ignore relationships |
| 19 | UT-012, IT-031 | BR-003, NFR-001, NFR-006 | AC-006, AC-039, AC-044 | Fails: relationship observability and redaction coverage do not exist |
| 20 | UT-011–UT-013, REG-009 | FR-025, NFR-001, NFR-005 | AC-006, AC-036–AC-037, AC-043 | Fails: no account-keyed relationship provider or localized semantics exist |
| 21 | AT-001–AT-003, AT-009 | BR-001, BR-003, FR-012, FR-016, FR-022–FR-023, RULE-001 | AC-001, AC-005, AC-020, AC-026, AC-033–AC-034, AC-045 | Fails: profile actions and relationship states are absent |
| 22 | AT-004–AT-005, AT-007–AT-008, AT-014 | FR-010–FR-011, FR-014–FR-015, FR-025, RULE-002, RULE-006 | AC-018–AC-019, AC-023–AC-025, AC-037, AC-046, AC-049 | Fails: client models cannot safely collapse, reveal, or discard protected content |
| 23 | AT-010 | FR-027 | AC-054, AC-055 | Fails: shared content menus expose only existing report/delete actions |
| 24 | AT-011–AT-013 | FR-008, FR-024–FR-025, NFR-001, NFR-005 | AC-016, AC-035–AC-037, AC-039, AC-043 | Fails: relationship settings lists and accessibility coverage do not exist |
| 25 | REG-007–REG-008 | BR-001, FR-026, NFR-002 | AC-002, AC-038, AC-040 | Fails: raw API route enforcement inventory does not cover relationships |
| 26 | MAN-001 | NFR-005 | AC-043 | Manual: requires real assistive technology and device layouts |
| 27 | MAN-002 | BR-002 | AC-003, AC-004 | Manual: requires live local PDS/Tap and compatible-client interaction |

## Implementation Steps

### Step 1: IT-035

- Write failing test: `appview/internal/db/mutes_blocks_migration_test.go`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db -run TestMutesBlocksMigration -count=1` from `appview/`.
- Confirmed failure: Yes — the focused test failed because `000023_mutes_blocks.up.sql` did not exist.
- Implement: Minimum reversible `000023_mutes_blocks` migration only.
- Refactor: Only while green.
- Notes: Added only `actor_mutes` and `atproto_blocks`; owner deletion cascades private mutes, subject deletion does not, public block rows have no membership foreign key, duplicate external block pairs are allowed, URI and owner/rkey identities are unique, and up/down/up passes.

### Step 2: IT-002

- Write failing test: owner-scoped, immediate, unique, idempotent mute store behavior.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/relationships -run TestStoreMuteIsOwnerScopedImmediateAndIdempotent -count=1`.
- Confirmed failure: Yes — `NewStore` and `Store` were undefined.
- Implement: Minimum `relationships.Store` mute CRUD.
- Refactor: Only while green.
- Notes: Added typed-DID `Mute`, `Unmute`, and `IsMuted` methods. Repeated mutations converge, reads are immediate and owner-scoped, and errors omit the private target pair.

### Step 3: UT-001, UT-014, UT-016

- UT-001 red: relationship policy types and `Decide` were undefined.
- UT-001 green: added one closed decision table with moderation over symmetric block over one-way mute precedence.
- UT-014 red: the canonical membership requirement and not-found sentinel were undefined.
- UT-014 green: added a profile-row-only membership boundary; absent identities share `ErrProfileNotFound`, while lookup failures remain distinct.
- UT-016 red: lifecycle event, role, and decision types were undefined.
- UT-016 green: only permanent owner membership removal selects owned-mute deletion; subject/session/device/account events retain rows.
- Regression command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db ./internal/relationships -count=1` passed.

### Step 4: UT-002, IT-001, IT-003, IT-004

- UT-002 red: target resolver and stable identifier/self errors were undefined.
- UT-002 green: DIDs parse directly, handles resolve once, self stops before membership, and only current profile-row members proceed.
- IT-001 red: all four relationship mutation handlers were undefined.
- IT-001 green: handlers derive the owner from auth context, map invalid/self/non-member consistently, pass the session only to public block operations, and emit camelCase state/record fields.
- IT-003 red: the store had no viewer-scoped combined relationship state read.
- IT-003 green: private mute state is owner-only; public block directions use duplicate-safe `EXISTS` queries.
- IT-004 red: the store had no caller-owned block identity selection.
- IT-004 green: unblock lookup returns only outbound records owned by the caller and never inbound/foreign records.
- Regression command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db ./internal/relationships ./internal/api -count=1` passed.

### Step 5: UT-009, IT-011, IT-023

- UT-009 red: relationship limit parsing and typed cursor helpers were undefined.
- UT-009 green: limits default to 50, accept 1–100, reject invalid values, and use a type-tagged timestamp/DID cursor with strict decoding.
- IT-011 store red: mute/block list methods were undefined.
- IT-011 store green: owner-scoped current-member selection happens before `LIMIT`; 112 equal-time mutes traverse as 100 + 11 eligible rows, former members stay hidden, and duplicate public block pairs collapse.
- IT-011 handler red: mute/block list handlers were undefined.
- IT-011 handler green: handlers derive owner from auth context, resolve current handles, return owner-relative relationship flags, and emit a cursor only when another eligible page exists.
- IT-023 red: all six relationship route policies were absent.
- IT-023 green: two read and four write routes use authenticated device middleware, no-body policy, and the existing rate classes.
- Regression command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db ./internal/relationships ./internal/api ./internal/app ./internal/routes -count=1` passed.

### Step 6: UT-010, IT-007, IT-036

- UT-010 red: the canonical block indexer constructor was undefined.
- UT-010 green: Indigo `bsky.GraphBlock` records validate DID/time, upsert by URI, retain compatible duplicate pairs, and delete/replay idempotently.
- IT-007 initial state: Green after UT-010 and earlier list filtering; a current-member-owned block targeting an absent DID is retained, hidden from lists, and reappears when the profile row is inserted.
- IT-036 red: block events fell through to `index.NotImplemented` and the Tap filter omitted the collection.
- IT-036 green: `app.bsky.graph.block` is registered once and appears exactly once in `TAP_COLLECTION_FILTERS`; no local block NSID exists.
- Regression command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db ./internal/relationships ./internal/index ./internal/app ./internal/routes -count=1` passed.

### Step 7: IT-005–IT-006, IT-009, IT-032

- IT-005 red: block mutation returned the intentional unavailable stub and the authenticated PDS client had no paginated record-listing capability.
- IT-005 green: block lists caller-owned public records, creates a generated `app.bsky.graph.block` only when none match, waits for PDS confirmation, returns URI/CID/rkey plus the expected state, and never writes the Tap-owned projection synchronously. The observability wrapper preserves the optional listing capability.
- IT-006 red: rapid pre-index unblock returned `block mutation unavailable`.
- IT-006 green: an indexed identity is deleted directly; a missing or ambiguous projection triggers authenticated paginated PDS reconciliation, deterministically deletes every matching caller-owned rkey, accepts record-not-found on retry, and leaves projection changes to ordered Tap events.
- IT-009 server portion: the PDS-confirmed mutation response supplies the expected provisional block state while the projection remains stale; the Flutter confirmed-overlay portion remains scheduled with the client provider work.
- IT-032 initial state: Green after the owned-direction mutation and indexer work; mutual records remain independently owned, removing one leaves the inbound direction effective for both parties, and only the final delete clears symmetric policy.
- Focused commands passed for the PDS adapter, relationship reconciliation, observability wrapper, and ordered/mutual index convergence.

### Step 8: IT-008

- Tap admin client red: the reusable `NewAdminClient`/repository tracker did not exist.
- Tap admin client green: WebSocket/HTTP Tap URLs normalize to one bounded `/repos/add` client; repeated requests are supported and non-success responses fail clearly. The existing CLI now reuses this implementation.
- OAuth retry red: successful profile initialization could not request repository tracking.
- OAuth retry green: every successful initialization requests ordinary Tap tracking best-effort, including returning users and restart retries, without making Tap convergence an authentication readiness state.
- Index wiring red: the production dispatcher had no repository tracker path.
- Index wiring green: genuine profile-row creation requests tracking and retains direct Bluesky profile backfill behavior even when the tracking request fails.
- Full convergence fixture: retained inbound blocks apply immediately once the profile row establishes membership; joining-owned outbound blocks remain absent until resumed Tap delivery, then converge after service recreation/retry. No activation/readiness table exists.

### Step 9: IT-025, REG-005

- Relationship restoration portion initial state: Green from the membership-filtered list queries and retention schema. Subject departure hides mute/block management rows and targeting while preserving the private mute plus both public block directions; re-adding the same DID restores lists and full relationship state.
- Follow, content, search, report, and directed-write membership coverage completed with Steps 11–16 and now shares the same retention/restoration boundary.

### Step 10: UT-003–UT-004, IT-010

- UT-003 red: profile rows/responses had no `muted`, `blocking`, or `blockedBy` state.
- UT-003 green: full profiles and account summaries carry the authenticated viewer's three relationship directions; private mute state is selected only for its owner.
- UT-004 red: a blocked profile serialized the ordinary bio, metrics, mutual/follow state, crafts, activity counts, and banner.
- UT-004 green: either block direction activates a dedicated minimum wire shell containing identity, membership, and accurate relationship annotations only; owned-unblock, reciprocal-block, and report decisions can be derived without exposing ordinary profile content.
- IT-010 green: real-Postgres viewer matrix proves Alice's private mute does not appear for Bob or Carol, mutual public block directions remain accurate, blocked pairs receive the shell, and unrelated Carol receives Bob's full eligible profile.

### Step 11: IT-024, AT-015

- Membership inventory red: user-facing profile/report targets still admitted retained indexed content or externally resolvable identities after the profile row disappeared.
- Green: profile, account report, post report, follow, mute, block, search, graph, relationship-list, and directed-interaction targets all require current `craftsky_profiles` membership. Unknown and resolvable non-members share the existing not-found result and no longer trigger identity hydration or PDS work.
- Retention regression: a former member's post stays indexed but cannot become a new report target until the same DID rejoins.

### Step 12: IT-021–IT-022, REG-002, REG-006

- Graph/search red: block directions and membership were not part of follower/following, count, profile search, post search, hashtag, project, or identity-cache selection.
- Green: block filtering is symmetric for authenticated viewers, mute leaves graph relationships visible, nonmembers remain absent, and counts use the same eligibility predicate as rows.
- Regression coverage proves unrelated viewers retain eligible results and private mute state is never added to shared search or graph responses.

### Step 13: IT-012–IT-013, IT-029–IT-030

- Read-selection red: post-filter shaping could return sparse pages when ineligible authors occupied the SQL limit.
- Green: timeline, search, hashtag, project, profile-post, and profile-comment queries apply membership and viewer relationship predicates before ordering and limit. Eligible rows refill pages and cursors remain based on delivered rows.
- Query-plan fixtures exercise the relationship lookup indexes and dense-page behavior with excluded rows ahead of eligible rows.

### Step 14: UT-005–UT-006, IT-014–IT-015, IT-027, IT-033

- Response-shaping red: post and quote DTOs could not safely represent relationship-protected content, and a muted thread branch had no scoped reveal path.
- Green: direct muted roots remain addressable, muted thread branches and quotes use revealable placeholders, blocked content uses non-revealable placeholders, and platform moderation retains precedence.
- Flutter deserialization now creates internal sentinel data only for a fully protected wire payload; protected cards short-circuit before ordinary fields render. Muted branch reveal fetches only the selected parent/descendant branch by URI.

### Step 15: UT-008, IT-016–IT-017, REG-001, REG-004

- Authorization red: follow, like, repost, reply, quote, and mention creation reached record creation across either block direction.
- Green: every directed actor is resolved as a current member and checked symmetrically before PDS work. Cleanup of caller-owned records, profile/content reporting, and reciprocal block creation remain allowed; ownership remains independently enforced.

### Step 16: IT-020, IT-034, REG-003

- Retention red: original post and interaction migrations tied public records to membership deletion.
- Green: public posts, follows, likes, reposts, and blocks remain indexed when membership disappears; private subject relationships remain retained while owner-private mute deletion is the only permanent relationship cleanup.
- Rejoin tests restore eligible relationship state and preserve unrelated engagement contributions without manufacturing activation state.

### Step 17: UT-007, IT-018

- Notification red: activation could persist a relationship-ineligible event and enqueue a push.
- Green: one shared relationship eligibility check runs before notification persistence. Mute creation and block indexing cancel pending, retry, or leased deliveries in the same transaction that makes indexed policy effective.

### Step 18: IT-019–IT-020, AT-006

- Delivery red: stored stale notifications still influenced list/newness and a leased push could race a new relationship.
- Green: list and newness queries recheck eligibility, badge state derives from the filtered count, push claim excludes ineligible rows, and the dispatcher rechecks the exact lease immediately before provider send. Cancelled or succeeded deliveries never become sendable again after unmute/unblock.

### Step 19: UT-012, IT-031

- Observability red: relationship operations had no dedicated bounded telemetry contract.
- Green: mutation, Tap index, and join/backfill paths emit bounded operation/result/duration metrics. PDS block create/delete/list fallback retains its existing bounded wrapper, and notification suppression/cancellation uses safe categorical decisions.
- Redaction tests send identifier-shaped sentinels and prove they normalize to `unknown`; no actor pair, DID, rkey, URI, session, or private mute target is accepted as a metric dimension.

### Step 20: UT-011–UT-013, REG-009

- Client-state red: no account-owned relationship cache existed and a stale pre-Tap response could reverse a PDS-confirmed optimistic block.
- Green: `profileRelationshipProvider(AccountKey, subject)` uses fixed-account Dio, permits one subject/action mutation in flight, applies exact optimistic rollback, fences late completion by account session generation, and preserves a confirmed overlay until indexed state matches.
- Generated Riverpod/mapping/localization artifacts were refreshed from source; localized actions, annotations, confirmations, placeholders, errors, list states, and semantics are covered by Flutter tests.

### Step 21: AT-001–AT-003, AT-009

- Profile UI red: visitor profiles exposed no relationship controls or annotations and blocked profiles rendered ordinary content.
- Green: profile More actions reflect Mute/Unmute and Block/Unblock, public block and restoration confirmations are mandatory, mutations provide localized feedback, and either block direction renders only the minimum identity/action shell.

### Step 22: AT-004–AT-005, AT-007–AT-008, AT-014

- Client rendering red: protected roots, quote states, branch reveal, and account-boundary cache eviction had no durable Flutter representation.
- Green: muted and blocked placeholders are distinct, only muted placeholders reveal, direct muted roots remain visible, account transitions invalidate protected cached state, and stale server payloads cannot repopulate content hidden by a confirmed overlay.

### Step 23: AT-010

- Content-menu red: shared post-shaped menus exposed only report/delete actions and loaded top-level rows remained visible until server refetch.
- Green: every non-self `PostCard` offers current-state author mute/block actions with the same confirmations and feedback as profiles. Feed, search, project, and profile-list cards disappear as soon as optimistic mute/block state begins while direct/thread cards retain the required protected shape.

### Step 24: AT-011–AT-013

- Settings red: no relationship list navigation, pagination, retry, empty state, or row reversal existed.
- Green: Settings exposes Muted accounts and Blocked accounts with cursor-driven pagination, localized loading/empty/error/retry/load-more states, profile navigation, immediate successful row removal, and mandatory unblock confirmation.

### Step 25: REG-007–REG-008

- Route-inventory red: the six relationship endpoints were absent from the policy registry.
- Green: authenticated no-body policies and rate classes cover list mutes, list blocks, mute, unmute, block, and unblock. Existing protected profile/post/notification routes continue to enforce server policy for clients without new Flutter behavior.

### Step 26: MAN-001

- Check: Real VoiceOver/TalkBack/keyboard navigation on phone and large layout in a non-default locale.
- Status: Not run in this workspace. The automated environment has no attached assistive-technology session or physical/emulated phone and large-layout device. Localization and semantics are covered by widget/static tests, but real device verification remains a manual follow-up.

### Step 27: MAN-002

- Check: Local PDS/Tap block create/delete from Craftsky and a compatible client or repository tool.
- Status: Not run in this workspace. No interactive compatible-client session was available. PDS create/list/delete, held/resumed Tap ordering, dispatcher registration, backfill, rapid unblock, mutual blocks, and projection ownership are covered by integration fakes plus real-Postgres tests.

## Execution Notes

- 2026-07-19: Loaded all four authoritative workflow documents. Document review is Approved with notes and has no blocking findings.
- 2026-07-19: User explicitly invoked `implement-tdd`; this satisfies the source-implementation approval gate recorded in the workflow documents.
- 2026-07-19: Existing modifications to `01-requirements.md` through `04-coding-plan.md` predate implementation and must be preserved.
- 2026-07-19: IT-035 red was confirmed on the absent migration; the focused test passed after adding the two-table reversible schema.
- 2026-07-19: IT-002 red was confirmed on the absent store; the focused owner-isolation/idempotency test passed with the minimum mute CRUD implementation.
- 2026-07-19: UT-001, UT-014, and UT-016 each completed a separate red-green loop; the combined foundation packages pass.
- 2026-07-19: UT-002 and IT-001/IT-003/IT-004 completed separate red-green loops; target, privacy, ownership, database, and API package regressions pass.
- 2026-07-19: UT-009, IT-011, and IT-023 completed separate red-green loops; list pagination and six-route policy wiring pass across all nearby packages.
- 2026-07-19: UT-010 and IT-036 completed red-green loops; IT-007 was already green from those minimal dependencies and is recorded without forcing an artificial failure.
- 2026-07-19: IT-005 and IT-006 completed red-green loops; the server half of IT-009 and already-enabled IT-032 behavior pass without synchronous projection ownership crossing into the API layer.
- 2026-07-19: IT-008 completed separate Tap-client, OAuth-retry, dispatcher-wiring, and convergence loops. IT-025/REG-005 relationship retention/restoration passes; broader public-record coverage remains linked to later surface groups.
- 2026-07-19: UT-003, UT-004, and IT-010 completed profile response and store privacy loops; the full profile test set passes.
- 2026-07-19: Steps 11–18 completed membership inventory, read selection/shaping, directed authorization, record retention, notification, newness, and push-race loops across the AppView packages.
- 2026-07-19: Step 19 added identifier-free relationship mutation/index/backfill telemetry and redaction coverage while retaining the bounded PDS and notification signals.
- 2026-07-19: Steps 20–25 completed account-keyed Flutter state, profile/content/settings UI, protected payload rendering, immediate loaded-row removal, localization, generated artifacts, and route inventory.
- 2026-07-19: A final former-member post-report regression, settings unblock confirmation test, and pending-request top-level removal widget test closed the last contract-level gaps.
- 2026-07-19: Final gates passed: `just test` (all Go packages with `-race`), `just app-test` (920 tests), `just app-analyze` (no issues), and `git diff --check`.
- 2026-07-19: MAN-001 and MAN-002 remain explicitly recorded manual follow-ups because this workspace had neither an assistive-device session nor an interactive compatible-client session.

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented manual gaps
- [x] All planned automated Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Generated artifacts updated from source where required
- [x] Full `just test` passing
- [x] Full `just app-test` passing (932 tests)
- [x] Full `just app-analyze` passing
- [x] `git diff --check` passing
- [x] Manual checks completed or explicitly recorded as unavailable in this environment
- [x] Documentation updated and read back
- [x] Review completed: `06-implementation-review.md` required IR-007 after the first correction pass; Step 36 records that correction pending independent re-review

## Review Correction Pass

The user selected **Address required changes** on 2026-07-19 after the implementation review. That selection authorizes the focused high-risk privacy, migration, delivery, and observability corrections below. The existing approved requirements and acceptance tests remain authoritative; the correction pass adds missing evidence rather than changing product scope.

| Step | Finding | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|---|
| 28 | IR-001 | IT-027 | FR-030, NFR-002 | AC-040, AC-056 | Fails because third-party reply/reference hydration checks only viewer-to-author blocks. |
| 29 | IR-002 | IT-006 | FR-003 | AC-011 | Fails when one record is indexed and another matching PDS record is not. |
| 30 | IR-003 | IT-019 | FR-002, FR-013 | AC-008, AC-022 | Fails because mute upsert and pending-delivery cancellation are separate commits. |
| 31 | IR-004 | AT-003 | FR-007, FR-012, FR-027 | AC-015, AC-020, AC-055 | Fails because a fresh direct muted post has no author viewer state or annotation. |
| 32 | IR-004 | AT-006, IT-009 | FR-025 | AC-036, AC-037 | Fails because loaded notification/repost actor surfaces do not observe pending relationship state and confirmed overlays never schedule bounded reconciliation. |
| 33 | IR-005 | IT-035 | FR-020, NFR-004 | AC-031, AC-042 | Fails because migration 000023 does not remove version-22 public-record membership cascades. |
| 34 | IR-006 | IT-031 | NFR-001, NFR-006 | AC-039, AC-044 | Fails because denial, cancellation, and lag/failure paths do not emit the required bounded relationship telemetry. |
| 35 | IR-006 | IT-029–IT-030, UT-013, REG-007–REG-008 | NFR-002–NFR-005 | AC-040–AC-043 | Fails because dense-pagination, query-plan, relationship accessibility, and raw protected-route evidence is absent or not explicitly mapped. |
| 36 | IR-007 | UT-005, IT-029 | FR-010, NFR-003 | AC-018, AC-041 | Fails when a muted reply parent falls on an earlier page or outside a focused-reply window and its unmuted descendant is selected independently. |

### Step 28: IR-001 / IT-027

- Write failing test: Real-Postgres Carol-view thread/reference fixtures covering a blocked parent/child author pair before attribution hydration.
- Run command: Focused `appview/internal/api` test.
- Confirmed failure: Yes — Carol received both Alice's reply to blocked Bob and its descendant.
- Implement: Added a shared indexed parent-author/child-author block predicate to root-comment and recursive branch selection; protected descendants inherit the parent decision.
- Run command: `go test ./internal/api -run TestPostStore_ListCommentBranchReplies_HidesThirdPartyBlockedParentChildEdge -count=1` passed against real Postgres.
- Refactor: Reused three set-based predicates for reply-parent, mention, and quote participant pairs. Recursive branch protection propagates to descendants, and notification metadata uses one event-level reference decision before any reference join hydrates payload.
- Notes: Three meaningful red-green loops now cover Carol viewing a blocked reply edge, a blocked mention edge, and a retained notification whose reference graph must become content-free. The combined focused IT-027 set passes against real Postgres.

### Step 29: IR-002 / IT-006

- Write failing test: One indexed Alice-to-Bob block plus one PDS-only matching block.
- Run command: Focused `appview/internal/relationships` test.
- Confirmed failure: Yes — with one indexed block plus one PDS-only duplicate, unblock made zero PDS list calls and deleted only the indexed rkey.
- Implement: Seed deletion candidates from every indexed caller-owned record, always list the canonical caller PDS, union every matching rkey, sort once, and delete idempotently.
- Run command: The new mixed indexed/PDS-only regression and the existing rapid-unblock pagination/retry regression pass together in `appview/internal/relationships`.
- Refactor: One deterministic set now covers local projection lag in either direction without synchronously mutating the projection.
- Notes: The PDS listing capability is now mandatory for unblock, matching the complete-record reconciliation contract.

### Step 30: IR-003 / IT-019

- Write failing test: Force delivery cancellation to fail and assert the mute row is rolled back.
- Run command: Focused `appview/internal/relationships` test.
- Confirmed failure: Yes — a trigger-forced cancellation failure left Alice's mute row committed while the endpoint returned failure and the delivery remained pending.
- Implement: Added one store transaction that upserts the mute, cancels pending/retry/leased deliveries for the exact owner/actor direction, and commits only after both succeed.
- Run command: The failure rollback test and the existing owner-scoped/idempotent mute regression pass together in `appview/internal/relationships`.
- Refactor: Removed the obsolete standalone cancellation helper after the service switched to the atomic boundary.
- Notes: The dispatcher's existing claim and final lease/relationship checks remain independent race protection.

### Step 31: IR-004 / AT-003

- Write failing test: Decode a post-author viewer relationship and render a fresh direct muted root with an annotation and Unmute action.
- Run command: Focused Flutter model/widget tests.
- Confirmed failure: Yes — the Dart model had no viewer fields, the direct muted card rendered no annotation, its menu offered Mute, and the AppView author object omitted all relationship flags.
- Implement: Added nullable post-author viewer flags to preserve known-versus-absent state, emitted the authenticated direct-post relationship on the AppView author object, seeded the account-scoped relationship provider from that wire state, and rendered a visible muted annotation while retaining direct content.
- Run command: `flutter test test/feed/models/post_test.dart`; focused `PostCard` AT-003 widget test; `go test ./internal/api -run 'TestGetPost(IncludesAuthorViewerRelationshipState|_HappyPath)' -count=1` — all passed.
- Refactor: Centralized AppView author assignment in `ApplyPostAuthorViewerState`; the widget derives one initialized relationship value for both immediate rendering and provider seeding.
- Notes: Legacy and signed-out payloads can omit the nullable fields; authenticated false values remain distinguishable and prevent an empty cache seed from overwriting a true muted state.

### Step 32: IR-004 / AT-006, IT-009

- Write failing test: Pending relationship changes immediately remove loaded notification/repost actor surfaces; confirmed overlays schedule bounded reconciliation without silently reversing state.
- Run command: Focused Flutter provider/page/widget tests.
- Confirmed failure: Yes — notification actors could not decode viewer state, loaded notification and repost rows ignored the pending overlay, the badge count stayed stale, profile mutations used a handle-keyed cache separate from DID-keyed feed state, and no follow-up reconciliation was scheduled.
- Implement: Added known viewer fields to notification and repost actors; canonicalized profile relationship providers on target DID; seeded and watched actor state in notification rows and both post/reposter state in cards; added synchronous loaded timeline/notification/badge suppression at mutation start; and scheduled a cancelable two-second reconciliation refresh with a categorical diagnostic while keeping the confirmed overlay authoritative.
- Run command: Focused red-green model, provider, widget, profile, and AppView handler tests; then a combined nine-file Flutter correction set and the four lifecycle-heavy widget/provider files — all passed. Focused AppView notification/repost actor tests also passed.
- Refactor: Cache suppression lives on the owning timeline, notification-page, and badge notifiers; the relationship controller coordinates them without copying list internals. The reconciliation scheduler is injectable and owns a cancellation handle, canceled on Tap agreement or provider disposal.
- Notes: A combined widget run initially exposed a pending-timer teardown defect. The scheduler contract was tightened to return cancellation, and the full affected widget/provider set then passed with no leaked timers. A final reconciliation regression then proved the account-scoped notification page and badge providers were not invalidated; both account-family providers are now refreshed at mutation completion and at the bounded follow-up.

### Step 33: IR-005 / IT-035

- Write failing test: Faithful version-22 public tables retain post/like/repost rows after applying migration 000023 and deleting membership.
- Run command: Focused `appview/internal/db` test.
- Confirmed failure: Yes — the first focused command skipped because no database URL was present; rerunning through `just test` executed real PostgreSQL and left one post instead of two because the version-22 author-membership cascade remained active.
- Implement: Migration 000023 now drops the version-22 `did` membership foreign keys from public posts, likes, and reposts while retaining subject-post integrity. Down restores those constraints as `NOT VALID`, so rollback preserves already-retained public records while enforcing membership for future writes.
- Run command: `just test` — the race-enabled canonical Go suite passed, including the faithful version-22 upgrade, down, and second-up cycle against compose PostgreSQL.
- Refactor: The up statements are table/constraint tolerant for both fresh schemas and true version-22 upgrades; the down guards each table and existing constraint independently.
- Notes: The skipped focused run is not counted as evidence. Only the canonical real-Postgres red and green runs are recorded.

### Step 34: IR-006 / IT-031

- Write failing test: Production-path denial, index failure/lag, notification suppression, and push cancellation emit only bounded relationship telemetry.
- Run command: Focused AppView package tests.
- Confirmed failure: Yes — the observer exposed only operation/result, a real blocked like emitted no relationship metric, malformed/lagged block events exposed only a generic completion result, relationship-suppressed notification activation emitted no dedicated signal, mute/block delivery cancellation emitted none, and mutation failures lacked store/PDS stages.
- Implement: Extended the relationship metric with bounded stage and error-class allowlists; emitted directed-operation denial outcomes, block decode/validate/store outcomes plus creation-to-index lag, relationship notification suppression, atomic mute and block-index delivery cancellation counts, and store/PDS mutation failure stages.
- Run command: Each production-path integration was driven red then green through `just test`; the final race-enabled canonical Go suite passed all packages against compose PostgreSQL.
- Refactor: Runtime boundaries use optional detailed-observer interfaces so narrow test doubles and older local adapters retain the original operation/result contract. All metric inputs are closed categorical values; actors, targets, pairs, URIs, rkeys, CIDs, and provider error strings never cross the observer boundary.
- Notes: Block-index lag is the non-negative duration from the canonical block record's `createdAt` to AppView indexing. Cancellation reports only `none` or `some`, never a count or relationship identity.

### Step 35: IR-006 / IT-029–IT-030, UT-013, REG-007–REG-008

- Write failing test: Query plans use relationship indexes, all relationship controls expose localized semantics, and raw protected routes enforce policy without Flutter filtering.
- Run command: Focused Go and Flutter tests.
- Confirmed failure: Yes for the missing evidence boundaries. The first real-Postgres query-plan run showed the shared API fixture omitted both production block-pair indexes and PostgreSQL selected the owner/rkey uniqueness index. The localization completeness artifact failed because no localized destructive-action hint existed, and the profile menu semantics test found an empty hint. The new dense three-page relationship pagination test passed first run, confirming already-implemented SQL filtering rather than requiring a behavior change.
- Implement: Added `relationship_query_plan_test.go` and aligned the shared relationship fixture with migration 000023; added `relationship_pagination_test.go` with protected rows before, between, and after three full eligible pages; added `mute_block_l10n_test.dart`, a localized destructive-action hint, explicit button/enabled semantics for context-menu rows, and destructive semantics for profile/post block actions. Extended the six-route policy test to assert the existing `{error,message,requestId}` camelCase envelope and reject `request_id`.
- Run command: `just test`; focused localization, profile-action, and relationship-provider Flutter tests; `just app-test`; `just app-analyze`; `git diff --check` — all passed. The canonical Flutter suite completed 932 tests.
- Refactor: One optional semantic hint now serves compact and wide CraftSky context menus while every row exposes an explicit label, button role, and enabled state. Production relationship indexes remain defined only by migration 000023; test DDL now mirrors them instead of inventing a second index contract.
- Notes: IT-029 is directly evidenced by `TestDenseRelationshipFilteredTimelineFillsThreeOpaquePages`, with `TestSearchStoreRelationshipFiltersBeforeHashtagPagination`, `TestStoreRelationshipListsAreOwnerScopedEligibleStableAndDeduplicated`, and the existing timeline/search seek-cursor tests covering the other paginated shapes. IT-030 is directly evidenced by `TestRelationshipFilteringQueryPlanUsesBidirectionalIndexes`; the set-based selection tests prove relationship checks are not issued per item. UT-013 is evidenced by `mute_block_l10n_test.dart`, `ProfileActions` semantics, protected-card semantics, relationship-list widget states, and generated localization files. REG-007 is evidenced by `TestRelationshipRoutesUseAuthenticatedNoBodyPolicies`, relationship list camelCase/cursor tests, and the shared envelope/cursor suites. REG-008's raw-server inventory maps direct Go handler/store tests for profile shells, timeline/search/thread/reference filtering, notification suppression, and directed follow/like/repost/reply/quote/mention denial; none depends on Flutter-side filtering.

### Step 36: IR-007 / UT-005, IT-029

- Write failing test: Real-Postgres ordinary-page and focused-window reply fixtures where muted Bob is an ancestor of an unmuted descendant outside the current response window.
- Run command: The first focused `go test` compiled but skipped because neither database URL was exported; repository-root `just test` then ran the regression against compose PostgreSQL.
- Confirmed failure: Yes — page 1 returned muted Bob, but page 2 returned Carol's `hidden-descendant` instead of the later eligible sibling. The focused-window regression separately returned the ten newest descendants while their muted ancestor was outside the slice.
- Implement: Added one indexed viewer-mute predicate and carried the first `muted_ancestor_uri` through ordinary, focused, and continuation recursive queries. A muted row remains eligible as the content-free placeholder source; every descendant is excluded before `LIMIT`. A focus inside the collapsed subtree resolves to that ancestor, and `hasMore` ignores the hidden descendants.
- Run command: Both new real-Postgres regressions passed together; the broader AppView `ListCommentBranchReplies|ListCommentReplies|GetPostComments` suite passed; focused Flutter placeholder and explicit-reveal tests passed; `just test`, `just app-test`, `just app-analyze`, and `git diff --check` all passed. The canonical Flutter suite remains 932 tests.
- Refactor: Ordinary pagination, focused-window selection, and continuation detection now share the same `muted_ancestor_uri` invariant instead of rebuilding page-local ancestry. Existing block/reference propagation remains stricter and unchanged.
- Notes: The ordinary server path keeps one content-free muted placeholder while excluding descendants before pagination. The existing explicit Flutter direct-load reveal remains the only path that hydrates the branch for the current view, and its UT-005 tests remain green. MAN-001 and MAN-002 remain the previously documented external manual gaps.
