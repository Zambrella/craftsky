# TDD Implementation Plan: Large-screen discovery sidebar

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`
- Authorization: user invoked implement-tdd for this folder on 2026-10-14. No commits, push, deployment or migrations enabled.

## Implementation Rules

- Implement only linked requirements, one focused failing behavior at a time.
- Confirm meaningful red, implement minimally, run focused green, refactor only while green.
- Use current worktree and existing account/visibility boundaries.

## Progress

- completed: Read workflow documents.
- completed: Create execution plan.
- completed: Inspect relevant code and tests.
- completed: Final verification and implementation notes.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State | Status |
|---|---|---|---|---|---|
| 1 | AT-003 | BR-001, FR-003 | AC-003 | Missing behavior or baseline regression | completed |
| 2 | UT-001 | BR-001, FR-002 | AC-002 | Missing behavior or baseline regression | completed |
| 3 | AT-002 | BR-001, FR-002 | AC-002 | Missing behavior or baseline regression | completed |
| 4 | UT-002 | FR-003 | AC-003 | Missing behavior or baseline regression | completed |
| 5 | REG-002 | FR-003 | AC-003 | Missing behavior or baseline regression | completed |
| 6 | REG-003 | FR-003 | AC-003 | Missing behavior or baseline regression | completed |
| 7 | AT-004 | FR-004 | AC-004 | Missing behavior or baseline regression | completed |
| 8 | IT-001 | BR-002, RULE-001 | AC-006 | Missing behavior or baseline regression | completed |
| 9 | IT-002 | FR-005, RULE-002 | AC-005, AC-007 | Missing behavior or baseline regression | completed |
| 10 | IT-003 | FR-007, RULE-002 | AC-007, AC-009 | Missing behavior or baseline regression | completed |
| 11 | IT-004 | RULE-001 | AC-006 | Missing behavior or baseline regression | completed |
| 12 | IT-005 | RULE-003 | AC-011 | Missing behavior or baseline regression | completed |
| 13 | IT-006 | RULE-003, FR-005 | AC-005, AC-011 | Missing behavior or baseline regression | completed |
| 14 | AT-005 | BR-002, FR-005 | AC-005 | Missing behavior or baseline regression | completed |
| 15 | AT-006 | FR-006 | AC-008 | Missing behavior or baseline regression | completed |
| 16 | AT-007 | FR-007 | AC-009 | Missing behavior or baseline regression | completed |
| 17 | REG-004 | FR-007 | AC-009 | Missing behavior or baseline regression | completed |
| 18 | AT-001 | FR-001 | AC-001 | Missing behavior or baseline regression | completed |
| 19 | AT-008 | NFR-001 | AC-010 | Missing behavior or baseline regression | completed |
| 20 | REG-001 | FR-001 | AC-001 | Missing behavior or baseline regression | completed |
| 21 | MAN-001 | NFR-001 | AC-010 | Requires device evidence | cancelled |
| 22 | MAN-002 | NFR-001 | AC-010 | Requires device evidence | cancelled |

## Mandatory variants

- AT-003: fresh navigation; mounted page; second craft; same-craft reselection after manual tab changes; same current craft; late reorder; empty project results; unchanged preferences.
- AT-006: each section pending with other ready; each failure/retry; both pending; both failed; successful empty member response.
- AT-007: cached A cleared at switch; A-before-B and A-after-B; late profile/member reads independently; preference save without switch.
- IT-002: counts 0/1/4/5/6; each exclusion class individually; combined newest exclusions ahead of six eligible rows; all excluded. Seed expired and negated moderation outputs as visible control cases.
- IT-003: A/B/A reads, viewer block/mute and blocked-by behavior, next read after mutation, followed eligible account.
- IT-004: completion preserved on repeat/edit/reindex; departed/deletion-pending/deleting/terminal/unknown owners excluded; active reactivation with retained completion remains old; cleanup removes completion and missing metadata remains omitted.
- IT-006: success/empty/error contract, exact key sets, present/absent fields, private canaries, static route dispatch, unauthorized middleware contract and client decoding.

Also cover AT-002 empty/unknown/all-supported; AT-004 real editor save; AT-005 counts, fallbacks/profile unavailable; UT-002 invalid/absent inputs; all shell destinations/breakpoints, short height, enlarged text, keyboard and semantics.

## Implementation Steps

### AT-003 / BR-001, FR-003 / AC-003

- Red: focused real-router test failed because `discovery-craft-knitting` was absent.
- Green: same focused test passed after craft link, optional route query and mounted TabController update.
- Setup corrections: correct notification repository imports/interface; assert craft identity rather than preference-dependent tab index.
- Command: `cd app && flutter test --no-pub test/router/app_shell_discovery_sidebar_test.dart --plain-name 'sidebar craft link selects a mounted Projects tab'`.
- All mandatory variants completed. Generated unrelated hash/mapper churn restored to keep scope bounded.

### UT-001 / FR-002 / AC-002

- Red: requested preferred-only provider was missing.
- Green: four independent fixtures passed for duplicates/unknown, empty, unknown-only and all supported craft values. Preserves AsyncValue states.
- AT-003 variants now green: fresh/mounted, second craft, repeated/same-current selection, manual tab reselection, preference reorder and unchanged preferences; empty repository results.
- Nearby Projects suite passed (10 tests), including preference default and identity-preserving reorder (REG-002/REG-003).

### AT-002, UT-002, REG-002, REG-003

- Preferred-only UI order and bare/absent/malformed/unknown navigation tests passed against the preceding implementation. Mounted valid→invalid requests restore preference-aware default. Existing default/reorder regressions passed.

### AT-004 / FR-004 / AC-004

- Red: Choose your crafts action missing for empty and unsupported-only arrays.
- Green: both cases open real EditProfileDialog, select Knitting, save through FakeProfileRepository and display its sidebar link.
- Harness corrected to use real overlay-aware UserProfile for save tests rather than overriding it with a stale immutable future. No production save flow changes.

### IT-001 / BR-002, RULE-001 / AC-006

- Red: ProfileStore lacked ListRecentMembers.
- Green: real PostgreSQL 16, exact production migrations, reverse insertion order and fixed A–F join-time fixture returned A,B,C,D,E on repeated reads and after F profile edit/reindex.
- Command: `TEST_DATABASE_URL=<worktree-local-compose-url> TEST_DATABASE_REQUIRED=true GOTOOLCHAIN=go1.27.2 go -C appview test ./internal/api -run ^TestRecentMembersOrdering$ -count=1`.

### IT-002 / FR-005, RULE-002 / AC-005, AC-007

- Red: newest viewer/blocked/muted/blocked-by/moderation-hidden candidates consumed slots; all-excluded returned rows.
- Green: parameterized counts 0/1/4/5/6, every exclusion class, combined exclusions ahead of six older eligible members and all-excluded passed on exact migrated schema.
- Expired, negated and warn moderation controls remain visible. Production moderation predicate reused, eligibility before limit.

### IT-003 / FR-007, RULE-002 / AC-007, AC-009

- Passed against preceding SQL: A/B/A reads, opposite block/mute/blocked-by relationships, followed eligible account, next reads after unmute/unblock fixture changes. No additional implementation needed.

### IT-004 / RULE-001 / AC-006

- Passed using existing onboarding Complete with expected lifecycle generation, actual CraftskyProfile update/replay and DatabasePrivateCleanup.Purge.
- Original completion timestamp retained; missing metadata omitted; departure/reactivation stays old; deletion-pending/deleting hidden; cleanup removes completion; reactivation without metadata omitted; terminal hidden. Unknown lifecycle separately covered in IT-002.
- No changes to onboarding, indexer, deletion or lifecycle implementations.

### IT-005 / RULE-003 / AC-011

- Red: recent was dispatched to the dynamic profile route (invalid_identifier), and no route policy existed.
- Green: full mux + standard bearer/device middleware + current-member/read/no-body policy + no-store response passed. Missing device expectation corrected to existing 400/missing_device_id contract.

### IT-006 / RULE-003, FR-005 / AC-005, AC-011

- Red: shared identity hydrators added accountType/customisation beyond the allowlist; client method absent; diagnostic manifest missing new operation.
- Green: full-mux exact key sets, positive public values, optional omissions, handle.invalid cache fallback, private canaries, successful empty and generic handler error passed. Dedicated endpoint skips only identity presentation decorators.
- Error fixture drops only a completion column inside isolated test schema; closing pool would instead fail membership middleware, so was not used as handler evidence.
- Client decoding + existing profile API regressions passed (25 tests).
- Serialized local/Sentry sinks retain operation/native PgError and exactly one issue, suppress private member/token/preference canaries. New diagnostic category is private.

### AT-005 / BR-002, FR-005 / AC-005

- Red: member rows absent for counts 1/4/5.
- Green: response order, existing ProfileAvatar/name fallbacks, every row's typed profile destination and actual compact-profile unavailable error passed.
- Navigation assertions use decoded pushed GoRouterState identity (imperative base route-information state and encoded URI paths are not identity oracles).

### AT-006 / FR-006 / AC-008

- Red: section-local progress/retry and successful empty message absent.
- Green: crafts pending/members ready, members pending/crafts ready, both pending, each failure/retry, both failed with independent recovery and successful zero members passed.
- Riverpod retains failures inside retrying AsyncLoading; whenData discards its cause. Preferred links now preserve source errors explicitly. Member rendering exposes retained failures while retrying.
- Explicit retry is a disabled spinning button; provider observer remains sole issue owner. No new retry policy or telemetry owner introduced.

### Broader backend gate follow-up / IT-005–IT-006

- First complete appview-check passed feature tests and other Go tests but failed exact route inventory: its reviewed expected list omitted GET /v1/profiles/recent.
- Updated that expected registration list; focused inventory test passed. Complete gate rerun passed; see final verification evidence below.

### AT-007 / FR-007 / AC-009

- Cached A cleared immediately; A-before-B, A-after-B, independently late profile/member reads and active preference cache updates passed with real UserProfile and lease-guarded recent reads.
- Red: accepted block/mute left the member row cached.
- Green: existing accepted publication/refresh points invalidate only the matching current active lease. Failed mute publication does not invalidate recent members.
- Settings unmute already forwards guarded publication to confirmExternalMutation; no separate settings implementation change is needed.

### REG-004, REG-001 / FR-007, FR-001 / AC-009, AC-001

- Updated placeholder-only UIP-022 assertions to fixed sidebar width, two themed cards and unchanged divider/breakpoint; observed its old assertion fail first.
- Added the new account member-read fake to account-switcher harness. Shell/navigation/account-switching/block/mute/settings regression suites passed: 84 tests.

### AT-001 / FR-001 / AC-001

- All nine existing destinations passed at widths 600/900/1200/1201/1440. Sidebar absent at/below 1200, with zero member reads; present above with localized section order and normal navigation.

### AT-008 / NFR-001 / AC-010

- All five crafts and five member links are reachable by Tab, keyboard activatable and meaningfully named; last row scrolls into view. 1201x500/1440x900 at 1x/2x passed without overflow.
- Harness corrections: empty page needed a realistic focusable action; route activation/pop resets focus, so traversal and activation are asserted separately. No shell focus modification was needed. Semantics handles disposed before framework verification.
- Constrained loading/error variants in both light and dark themes passed in the final suite.

### Complete backend verification

- `APPVIEW_CHECK_ARTIFACT_DIR=/tmp/craftsky-discovery-appview-check-final just appview-check`: PASSED, all release gates.
- Evidence includes exact production-schema service tests, race/shuffle checks, static analysis, Docker/smoke/health checks and pinned Tap handshake.
- Tap gate records no injected events: pinned v0.1.10 has no safe deterministic injection API. This is its documented gate evidence limit, not feature SQL coverage.

## Final verification and execution notes

- Within approved phase 2, UT-001 was executed before AT-002 to extract the preferred-only public provider while AT-003 was green; the table above reflects that order. No phase or requirement scope changed.
- `cd app && flutter test --no-pub` with the sidebar, preferred links, Projects, profile adapter, navigation, account-switcher, block/mute and settings suites: **160 passed**. Log: `/tmp/discovery-final-flutter-tests.log`.
- `cd app && flutter analyze`: **No issues found** after final source/test changes. Log: `/tmp/discovery-analyze-final.log`.
- Settings unmute member refresh passed using a retained list subscription, matching the mounted settings page. The initial unmounted auto-dispose fixture could not dispatch reverse; no extra settings source change was needed.
- Delayed preference completion now has an explicit route-initialization test: requested Quilting survives reorder, and a later plain Projects request restores preferred Sewing.
- Filtered build_runner unexpectedly removed unrelated generated outputs; all unrelated tracked generated files were restored byte-for-byte from HEAD. Only the new/touched route/provider companions remain in the diff. Repository-wide automatic dart fix was rejected by approval review; targeted edits resolved every reported lint instead.
- `git diff --check`: passed. Source diff reviewed against requirement/test IDs; no migration, dependency, lexicon, infrastructure, onboarding, lifecycle or deletion changes.
- Complete AppView gate passed at `/tmp/craftsky-discovery-appview-check-final`; the final whitespace contract follow-up also passed the complete gate at `/tmp/craftsky-discovery-appview-check-verified`.

### Final IT-006 whitespace loop / RULE-003, FR-005 / AC-005, AC-011

- Red: legal tab/newline-only display name serialized as a fourth item key instead of being omitted. PostgreSQL btrim's default space handling was insufficient for this input.
- Green: handler trims presentation names with Go strings.TrimSpace and omits an empty result; all recent-member route/store/diagnostic tests passed on real Postgres. No persistent profile data is modified.
- Final exact-source complete gate **PASSED** at `/tmp/craftsky-discovery-appview-check-verified` (exit 0; all release gates passed). This supersedes the earlier gate checkpoint.

## Requirement coverage

| Requirement IDs | Passing evidence |
|---|---|
| BR-001, FR-002 | UT-001 / AT-002 distinct supported preferences, catalog order and empty/unknown/all-supported cases |
| FR-003 | AT-003 / UT-002 / REG-002 / REG-003 route input, mounted reselection, delayed reorder, filtered query and default behavior |
| FR-004 | AT-004 real editor selection/save and overlay-driven sidebar update |
| BR-002, FR-005, RULE-001 | IT-001 / IT-002 / IT-004 ordering, count, retained join metadata; AT-005 rows, fallbacks and profile destination/unavailable flow |
| RULE-002 | IT-002 / IT-003 server eligibility before limit; AT-007 accepted local block/mute and settings unmute refresh; IR-001 delayed-block suppression within the existing overlay window and projected refill |
| RULE-003 | IT-005 / IT-006 full mux/auth/policy/field allowlist/client decoding and serialized diagnostic privacy |
| FR-006 | AT-006 independent pending/error/retry, disabled spinning Retry and successful empty members |
| FR-007 | AT-007 cached/late-account isolation and preference updates; REG-004 account-switcher regressions |
| FR-001 | AT-001 every shared-shell destination/breakpoint; REG-001 width/divider/navigation regressions |
| NFR-001 | AT-008 keyboard/semantics/scroll/large text, including constrained pending/errors in light/dark; manual limits below |

## Visual evidence and manual gaps

- MAN-001 is **partially observed, device portion skipped**: inspected synthetic Flutter test renders at 1440x900 in light/dark and 1201x500 at 2x text. Both sections are legible, and the last member is reachable after sidebar scrolling without clipping an action.
- Render artifacts: `/tmp/craftsky-discovery-desktop-light.png`, `/tmp/craftsky-discovery-desktop-dark.png`, `/tmp/craftsky-discovery-short-large.png`, `/tmp/craftsky-discovery-short-large-scrolled.png`.
- These are test-renderer fixtures, not a live account/browser/device. Temporary Outfit/DM Serif families were loaded for legible text; runtime font-fetch warnings and unregistered navigation icon-font placeholders are harness limitations. Existing craft SVGs and avatar fallbacks rendered.
- Collaborative preview initialized and navigated, then lost its automation host; resize/snapshot could not be completed. Temporary visual web harness was stopped. No live-browser visual pass is claimed.
- Actual reported iPad format, platform gestures and live platform font rendering remain **pending** for MAN-001.
- MAN-002 / iPad VoiceOver reading order and activation remain **pending**: no connected iPad/VoiceOver observation was available. Automated keyboard/semantics evidence is not a VoiceOver pass.
- Legacy production completion coverage remains unverified, as approved in ASM-002/CPQ-001; missing reliable metadata is omitted. No production inspection/backfill occurred.

## Stage exit

### IR-001 correction pass — 2026-10-15

User selected Address required changes after `06-implementation-review.md`.

| Progress item | Linked requirements / criteria | Status |
|---|---|---|
| Read workflow and correction inputs; inspect existing overlay boundaries | RULE-002, FR-007 / AC-007, AC-009 | completed |
| AT-007 delayed block projection, repeated stale reads and refill | RULE-002, FR-005, FR-007 / AC-005, AC-007, AC-009 | completed |
| AT-007 failed block, unblock/reblock and account-isolation controls | RULE-002, FR-007 / AC-007, AC-009 | completed |
| Relevant Flutter regressions, analysis and diff checks | Same scope plus REG-004 | completed |
| Record evidence and read back implementation notes | IR-001 | completed |

Correction scope: consume the existing accepted block overlay for the requested account session after member reads complete. The endpoint omits relationship flags, so do not reconcile an accepted block using default `blocking: false`. Retain the existing overlay's bounded reconciliation lifetime and guarded invalidations; add no polling, cache, API fields or backend selection changes. SQL remains the eligibility-before-limit authority. While a response is still stale, suppress its accepted-block target locally; a subsequent projected server response supplies the replacement fifth member. Test that transient suppression and eventual refill explicitly. Amend section 6 of the coding plan to document this consistency bridge.

- Red: new AT-007 stale-response/refill test failed at the first blocked-row assertion (expected none, found member0). Log: `/tmp/discovery-ir001-red.log`.
- Minimal green: after lease/current checks, read the existing controller overlay for each row using the requested session lease; omit only an accepted `true` block. Same focused test passed in `/tmp/discovery-ir001-green.log`. No reconciliation is attempted from omitted flags or bounded-list absence.
- AT-007 expanded checks: repeated stale responses remain suppressed; a projected response replaces the blocked member with the sixth eligible candidate and restores five rows; a later stale response does not undo suppression merely because the previous list omitted the target. Failed block does not hide; failed unblock retains the accepted block; accepted unblock permits the target; reblock hides again. A different account and a new session for the same DID do not inherit the old overlay. All 10 AT-007 cases passed in `/tmp/discovery-ir001-controls.log`.
- Regenerated the recent-members Riverpod companion with full build_runner (successful; `/tmp/discovery-ir001-codegen.log`). Restored 75 unrelated generated files from a pre-generation byte snapshot, preserving all earlier workflow changes. No filtered generation or broad automated fixes were used.
- Final broader Flutter verification: **182 passed**, the original ten feature/regression suites plus shared PDS mutation controller, account-boundary, retry and reconciliation suites. Log: `/tmp/discovery-ir001-regressions.log`.
- Static analysis initially flagged one cascade style issue in the new test. Corrected it locally; final `flutter analyze` reports **No issues found** in `/tmp/discovery-ir001-analyze-final.log`. The affected accepted-state/account-session control passed again after that test-only edit (`/tmp/discovery-ir001-final-control.log`). `git diff --check` passed.
- Correction diff is limited to `recent_members_provider.dart`, its generated companion, two AT-007 widget cases, and sections of `04`/`05` documenting the consistency bridge and evidence. No backend source, API, storage, diagnostics, controller lifetime or mutation-publication behavior changed. Therefore the prior exact-backend-source AppView gate remains applicable and was not rerun.
- Evidence limit: this fixes stale-row display while the existing accepted-block overlay is retained. The shared controller still refreshes at 2 seconds and expires at 30 seconds, or reconciles/resets through existing owners. This pass does not guarantee hiding during arbitrarily prolonged ingestion outages after that window. Temporary local suppression can show fewer than five rows until the next projected read; no polling or client-side refill query is added. Device/VoiceOver and legacy-data gaps above remain unchanged. Section 6 of `04` records this scope explicitly.
- IR-001 implementation correction is complete and ready for another review. `06` remains the historical review input; its verdict has not been rewritten as an approval by the builder. No commits were enabled.

- No commits were enabled; no commit or push was created.
- The initial implementation review produced IR-001. Review of this correction is deferred to the user's next-stage choice. No automatic review/polish/deployment follows this stage.

## Completion Checklist

Styling follow-up (2026-10-15): user requested that Your crafts and New to CraftSky appear without cutout cards. Removed only the shared section's CraftskyCard wrapper, retaining headings, padding, section gap and scrolling. Updated REG-001's existing card assertion to expect no cards and amended the coding plan's sidebar styling. Sidebar/navigation suites: **94 passed** (`/tmp/discovery-plain-sections-tests.log`); Flutter analysis: **No issues found** (`/tmp/discovery-plain-sections-analyze.log`); `git diff --check` passed. Prior card render artifacts describe the earlier styling, not this update. No new live-browser/device visual evidence is claimed.

Further styling (2026-10-15): user requested a border around New to CraftSky and alignment of Your crafts with its links. Added a one-pixel theme borderHair outline with the existing r3 corner radius around only the member section, without card fill/shadow. Set craft buttons' horizontal padding explicitly to the heading's sp4 inset. Final sidebar suite: **39 passed** (`/tmp/discovery-border-alignment-tests.log`); analysis: **No issues found** (`/tmp/discovery-border-alignment-analyze.log`); `git diff --check` passed. The coding plan reflects the bordered-member/unbordered-craft presentation.

- [x] All Must requirements covered by passing automated tests or explicit manual evidence gaps
- [x] All planned automated Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Final exact-source complete backend gate observed
- [x] Final implementation notes read back
- [x] Implementation review explicitly deferred to stage exit
