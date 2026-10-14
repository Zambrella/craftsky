# Coding Plan: Large-screen discovery sidebar

## 1. Inputs

- Requirements: `01-requirements.md`, user-approved and revised to address review findings.
- Tests: `02-acceptance-tests.md`, including mandatory variants and contract-finalization checklist.
- Document review: `03-document-review.md`, Approved with notes; preserved as the review of the earlier versions.
- Stage authorization: The user requested coding planning after revising both earlier documents. Implementation is a separate next stage.
- Risk: Medium. This plan resolves the source-inspection and interface-design dependencies; automated verification and device checks remain pending.

## 2. Implementation Strategy

Replace the existing shared desktop placeholder with two independently loaded sections. Derive preferred-only links from the active profile and existing craft catalogs. Add a craft query parameter to Projects navigation and update the mounted page when it changes. Add a narrowly scoped AppView member read rather than reusing search ranking or social-graph list semantics.

Reuse ProfileStore, profile API/repository bindings, account-scoped transport and lease guards, current moderation/lifecycle SQL, existing avatar/profile navigation and the profile-edit dialog. Introduce no dependency, lexicon, scheduled job, global toolchain or infrastructure changes. No migration is planned: existing membership metadata is sufficient for the selected semantics; query performance can be inspected locally during implementation before considering an index.

### Membership verification — DR-001 / GAP-001

Read-only source evidence:

| Source | Verified behavior | Design implication |
|---|---|---|
| `appview/migrations/000065_account_onboarding_completion.up.sql` | One completion row per DID, `completed_at NOT NULL DEFAULT now()` | Read this timestamp as first recorded completed onboarding for the retained membership metadata. |
| `appview/internal/api/onboarding_store.go`, `Complete` | Completion insert uses `ON CONFLICT (account_did) DO NOTHING`; no timestamp update on repeat submission | Repeated completion cannot promote an existing retained account to newest. |
| `appview/internal/routes/onboarding_route_test.go` | Existing test asserts repeated completion returns the same nonzero timestamp | Reuse and extend this test path for IT-004; this stage inspected but did not run it. |
| `appview/internal/index/craftsky_profile.go` | Profile upserts update craft/CID/indexing fields, not onboarding completions | Profile editing or reindexing must not change join order. |
| `appview/internal/accountdeletion/private_cleanup.go` | Account-deletion private cleanup deletes onboarding completion | Deleted accounts disappear through lifecycle eligibility and subsequent cleanup; do not recover deleted metadata. |
| `appview/internal/ownerlifecycle/terminal_inventory.go` | Terminal purge inventory includes onboarding completions | Metadata is not a permanent historical ledger across purges; a future new completion after removal has only its new retained timestamp. |
| `appview/migrations/000039_owner_effects_terminal_purge.up.sql` | `appview_owner_is_active` denies departed, deletion-pending, deleting, terminal and unknown owners | Require active membership, not just a leftover profile row. |

Conclusion: Existing metadata supports the approved **first recorded completion** definition while retained. No source change to onboarding or cleanup is needed. Join the completion table to a current CraftSky profile and require an active lifecycle; omit members without completion metadata. Do not infer membership dates from AT Protocol age, profile indexing or profile creation. Departure with retained completion must not promote a member on reactivation; deletion/purge must not preserve or recreate historical private metadata for this feature.

Evidence limits: No production database was inspected. The number of legacy accounts missing metadata is unknown. No new fixture was run, and first-ever historical membership across purges is not established. Missing data is handled by the approved omission policy, without backfill. IT-004 verifies these retained-metadata semantics locally during implementation.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Acceptance Criteria | Test IDs |
|---|---|---|---|---|---|
| Desktop shell | Placeholder only above 1,200 logical pixels | Scrollable two-section sidebar on the same routes/breakpoint | FR-001, NFR-001 | AC-001, AC-010 | AT-001, AT-008, REG-001, MAN-001, MAN-002 |
| Preferred crafts | Preferred-first full Projects catalog | Active-profile AsyncValue retaining load/error state; distinct preferred-only links | BR-001, FR-002, FR-004, FR-006, FR-007 | AC-002, AC-004, AC-008, AC-009 | AT-002, AT-004, AT-006, AT-007, UT-001 |
| Projects navigation | Route without craft input; mounted tab state | Optional craft query plus mounted-page update and manual-tab URL synchronization | BR-001, FR-003 | AC-003 | AT-003, UT-002, REG-002, REG-003 |
| Member selection | Profile store and shared moderation predicate | Active/current members ordered by completion, excluding viewer/relationships before limit | BR-002, FR-005, RULE-001, RULE-002 | AC-005–AC-007 | IT-001–IT-004 |
| Member API/client | Existing profile routes, repository and account transport | Authenticated fixed-five read with explicit public allowlist | FR-005, RULE-003 | AC-005, AC-011 | AT-005, IT-005, IT-006 |
| Async isolation | Session registry and ActiveAccountOperation | Lease-keyed member state; independent retries and immediate account-boundary reset | FR-006, FR-007 | AC-008, AC-009 | AT-006, AT-007, IT-003, REG-004 |

## 4. Files And Modules

Paths marked Create are proposed targets, not files created during planning.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/router/app_shell.dart` | Change | Replace placeholder; preserve width, divider and breakpoint | FR-001, NFR-001 | AT-001, AT-008, REG-001 |
| `app/lib/router/widgets/desktop_discovery_sidebar.dart` | Create | Section composition and member/craft actions | FR-001, FR-004–FR-007, NFR-001 | AT-001, AT-004–AT-008 |
| `app/lib/projects/providers/preferred_craft_links_provider.dart` | Create | Preferred-only async derivation from active profile | FR-002, FR-006, FR-007 | UT-001, AT-002, AT-006, AT-007 |
| `app/lib/router/router.dart`, generated `router.g.dart` | Change / regenerate | Optional `craft` query parameter passed to ProjectsPage | FR-003 | AT-003, UT-002 |
| `app/lib/projects/pages/projects_page.dart` | Change | Apply valid input in init/update; synchronize explicit manual tab selection | FR-003 | AT-003, UT-002, REG-002, REG-003 |
| `app/lib/profile/data/profile_repository.dart`, `api_profile_repository.dart`, `profile_api_client.dart` | Change | `fetchRecentMembers` / `getRecentMembers` read, decoding public summaries | FR-005, RULE-003 | AT-005, IT-006 |
| `app/lib/profile/providers/recent_members_provider.dart`, generated companion | Create / generate | Auto-dispose member read keyed by ActiveAccountLease | FR-006, FR-007 | AT-006, AT-007 |
| `app/lib/profile/providers/toggle_block_profile_provider.dart`, `profile_relationship_provider.dart`, `app/lib/settings/providers/relationship_list_provider.dart` | Change minimally | Invalidate current-lease member read after accepted local block/mute changes; reuse guarded publication | RULE-002, FR-007 | AT-005, AT-007, IT-003 |
| `app/lib/profile/models/profile_account_summary.dart`, `profile/widgets/profile_account_list_tile.dart` | Reuse | Existing typed identity, fallback, avatar and accessible profile link | FR-005, NFR-001 | AT-005, AT-008 |
| `app/lib/l10n/app_en.arb` and generated localization outputs | Change / regenerate | Section headings, empty/error copy; reuse localized craft labels/retry text | FR-001, FR-004, FR-006, NFR-001 | AT-001, AT-004, AT-006, AT-008 |
| `appview/internal/api/recent_members_store.go` | Create | Method on existing ProfileStore with explicit eligible-member SQL | BR-002, FR-005, RULE-001, RULE-002 | IT-001–IT-004 |
| `appview/internal/api/recent_members.go` | Create | Narrow read interface, response DTO, handler and standard error handling | RULE-003, FR-005 | IT-005, IT-006 |
| `appview/internal/routes/routes_profile_notification.go`, `catalogue.go` and applicable route policy/manifest declarations | Change | Register member read with current-member/read/no-body policy | RULE-003 | IT-005, IT-006 |
| `app/test/router/app_shell_discovery_sidebar_test.dart` | Create | Real router/widget acceptance and async cases | FR-001–FR-007, NFR-001 | AT-001–AT-008 |
| `app/test/projects/providers/preferred_craft_links_test.dart` | Create | Derivation tests, using existing provider test conventions | FR-002 | UT-001 |
| `app/test/projects/pages/projects_page_test.dart` | Change | Craft validation/update/default/reorder cases | FR-003 | UT-002, REG-002, REG-003 |
| Existing shell/navigation/account-switcher tests and profile repository fakes | Change as needed | Replace placeholder assertions, add repository method fake and preserve regressions | FR-001, FR-007 | REG-001, REG-004 |
| `app/test/profile/data/recent_members_api_client_test.dart` | Create | Serialized response decoding through a fake Dio adapter | RULE-003, FR-005 | IT-006 |
| `appview/internal/api/recent_members_store_test.go` | Create | Real Postgres selection/visibility/time fixtures | BR-002, FR-005, RULE-001, RULE-002 | IT-001–IT-004 |
| `appview/internal/routes/recent_members_route_test.go` | Create | Full mux access/schema/private canary tests | RULE-003 | IT-005, IT-006 |
| Existing onboarding route tests | Reuse / extend if needed | Verify preserved first completion, no business-flow changes | RULE-001 | IT-004 |

Storage SQL follows the existing parameterized pgx ProfileStore pattern and shared SQL predicates; no ORM or new query framework is introduced. The reference document describes sqlc as the intended stack, but this checkout's profile query owner is the store itself and no sqlc configuration/query source was found in inspection. Do not invent a codegen layer as part of this feature.

## 5. Services, Interfaces, And Data Flow

### Member API contract — DR-002 / GAP-002

- Operation: `GET /v1/profiles/recent`. Static registration must coexist with `/v1/profiles/{handleOrDid}` and `/v1/profiles/me`; route tests verify dispatch.
- Auth: Standard session bearer plus device header, `AccessCurrentMember`, `RateClassRead`, `BodyNoBody`, normal request/error middleware. Viewer comes from authenticated context, never a caller-supplied DID.
- Count: Fixed maximum five; no caller-selectable limit, filter, cursor or pagination. This is a bounded sidebar summary, not a browse directory. It deliberately has no next-page contract; future paginated discovery would use the standard opaque-cursor convention.
- Success: HTTP 200, `{"items": [...]}` with an array even when empty. Response uses `Cache-Control: no-store` to avoid intermediary viewer-specific reuse.
- Allowed item keys: `did`, `handle`, `isCraftskyProfile`, optional `displayName`, optional `avatar`. No join timestamp, biography, preferences, follow counts, relationship flags, OAuth data or moderation/private metadata. Use a dedicated Go DTO so expanding the full ProfileAccountSummary cannot accidentally broaden this response.
- `did` and `handle` use typed syntax wrappers in Go and existing typed parsing in Flutter. Use identity-cache presentation as in profile search; a missing cache handle falls back to the existing `handle.invalid` unavailable identity, rather than dropping eligible members after limiting or doing per-member network resolution.
- `isCraftskyProfile` is always true. Missing/blank display names and missing avatars are omitted on the wire; the client uses existing handle/avatar fallbacks. Synthesize public avatar URLs through the existing blob helper. Decode into existing Flutter ProfileAccountSummary defaults (`crafts: []`, relationship flags false, default customisation); no new client model is needed.
- Database/serialization failures use the existing internal-error envelope. Authentication/access failures come from normal middleware with its existing status/code, not custom sidebar exceptions. No partial-success list on a failed database read.

Illustrative wire examples:

```json
{"items":[{"did":"did:plc:aaaaaaaaaaaaaaaaaaaaaaaa","handle":"alice.craftsky.social","isCraftskyProfile":true,"displayName":"Alice"}]}
```

```json
{"items":[]}
```

```json
{"error":"internal_error","message":"Unable to load recent members","requestId":"example-request-id"}
```

The error example fixes the new handler's generic 500 message; use the existing envelope helper for requestId and standard code. IT-006 asserts exact key sets (allowing only the specified optional fields), positive public values and absence of private fixture canaries on success/error paths. Flutter adapter tests assert decoding of both present and omitted optional fields.

### Selection and ordering

Extend ProfileStore with a read accepting typed viewer DID. Query current CraftSky profiles joined to onboarding completions, left-joined to Bluesky profile presentation and identity cache. Require `appview_owner_is_active(cp.did)` and reuse `profileVisibleModerationPredicate` with its expected `cp` alias. Exclude the viewer, active mute relationship and block aggregates in either direction. Keep terminal-source handling consistent with existing relationship queries. Do not reuse search's exact-handle bypass, followed-rank or relevance ordering. Do not filter by follow status.

Apply all eligibility in SQL before `ORDER BY completion.completed_at DESC, cp.did COLLATE "C" ASC LIMIT 5`. The C collation makes the DID tie-break independent of database locale. Do not load five then hide some in Flutter. One SQL statement supplies an eligibility/ordering snapshot; the handler only maps public presentation fields.

Independent IT-001 cutoff oracle: four synthetic eligible members A–D have distinct times newer than time T; E and F share T, with E's DID bytewise before F. Expected five identities are A, B, C, D, E. Seed valid DIDs in reverse insertion order; F stays omitted across repeated reads and unrelated profile/indexing changes. Excluded candidates all newer than A must not consume slots.

Partial interfaces only:

```text
Go: ProfileStore.ListRecentMembers(ctx, viewer syntax.DID) ([]RecentMemberRow, error)
Go: RecentMembersReader.ListRecentMembers(ctx, viewer syntax.DID)
Dart: ProfileRepository.fetchRecentMembers() -> Future<List<ProfileAccountSummary>>
Dart: ProfileApiClient.getRecentMembers() -> Future<List<ProfileAccountSummary>>
```

No changes to PDS records, profile-indexer behavior, onboarding persistence or cleanup workers. Query-plan inspection uses local representative fixtures if needed; any demonstrated index need becomes a reviewed amendment rather than an unplanned migration.

## 6. State, Providers, Controllers, Or DI

Use the session registry's `activeLease`, including activation/session generations, as the identity boundary. The sidebar must be keyed or composed by that lease so old content cannot be retained via AsyncValue refresh behavior after switching.

```text
sessionRegistryProvider -> activeLease
  -> userProfileProvider(activeLease.session.account.did)
     -> preferredCraftLinksProvider(activeLease): AsyncValue<List<ProjectOption>>
  -> accountProfileRepositoryProvider(activeLease)
     -> accountDioProvider(activeLease.session.account)
     -> recentMembersProvider(activeLease): FutureProvider (auto-dispose)
```

- Preferred links: A synchronous Provider family maps the active profile's AsyncValue without collapsing error/loading to `[]`. Membership of `Profile.crafts` uses existing bare craft tags matched against ProjectOption catalog token suffixes. Deduplicate selections, retain supported catalog order, and obtain localized labels from the existing craft label mapping. Do not alter `preferredProjectCraftsProvider`, which still orders all Projects tabs.
- Member read: Generated `@riverpod` FutureProvider family keyed by ActiveAccountLease, watching the existing accountProfileRepository family. Capture ActiveAccountOperation for that lease before dependency acquisition and guard the completed read before returning. Check `ref.mounted` after awaits; preserve genuine causes/stacks rather than treating disposal misuse as cancellation.
- Initial member load runs independently of profile loading. Only mount/read these sidebar providers above the existing desktop boundary. No timer, global keepAlive cache, persisted member list or polling.
- Retry invalidates just the failing profile/member provider for the current lease. Ignore previous values during an account-boundary transition; profile saves already update UserProfile cache/record overlays and therefore rederive craft links.
- Successful local block/mute changes should invalidate the current lease's member read using existing mutation-publication boundaries, so a locally hidden member is removed promptly. Extend `_invalidateBlockReads` in `toggle_block_profile_provider.dart` and the guarded mute/unmute publication paths in `profile_relationship_provider.dart` / settings `relationship_list_provider.dart`; invalidate only after the operation's account guard accepts publication. Include a widget variant that locally blocks/mutes a listed member and observes the refreshed eligible list. There is no promise of server-push freshness for externally changed relationships; every new server read reapplies eligibility.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Sidebar

Keep the existing sidebar width/padding and divider in AppShell; replace the square placeholder with `DesktopDiscoverySidebar`. Use one scrollable column with separate CraftskyCard sections: preferred crafts first, recent member rows second. Avoid fixed card heights or unbounded nested lists. Retain navigation and main-content width constraints.

Craft rows use existing craft icon/label styles and normal focusable buttons/links. Empty preferences offer “Choose your crafts” calling `showEditProfileDialog(context)` under the existing responsive modal and lease/unsaved-work rules. This existing editor includes the craft picker; no new editor or account route is added.

Member rows reuse ProfileAccountListTile and its ProfileAvatar fallback. Its current order is handle then display name; retain it unless a small layout adaptation is needed for available width. Do not expose craft icons from member preferences; decoded crafts are empty in this contract. Use existing `showUserProfileCard`/UserProfileRoute flow rather than a custom destination. No trailing follow action.

Add localized section/empty/error text to the app's localization source and regenerate outputs with the existing project toolchain. Reuse retry and craft labels where available. Add accessible names/focus order and a scroll controller/scrollbar only if consistent with shell conventions.

### Projects route and mounted state

- Extend `ProjectsRoute` with optional `String? craft`, generating `/projects?craft=knitting`; leave a plain `/projects` navigation valid. Sidebar links use supported bare catalog tokens.
- Resolve input once against supported ProjectOption identities at the navigation/page boundary. ProjectsPage receives a validated catalog selection or absent input. Invalid/unknown/absent input uses the existing preferred-first default without changing preferences.
- Initialize TabController index from a valid requested craft; set `_hasSelectedCraft` so subsequent preference arrival does not override explicit route selection.
- Implement `didUpdateWidget` handling of a changed requested craft by selecting its current index while keeping the mounted page and existing filters/sort. Route changes must not require destroying the whole page or clearing unrelated user state.
- Keep the route query synchronized when the user explicitly changes a tab, using typed route replacement to avoid a history entry per tab change. Do not synchronize during programmatic selection or passive preference reorder; reuse a small suppression flag to avoid loops.
- Synchronization is necessary because clicking a sidebar link equal to an old query would otherwise be ignored after a manual tab change. A different current tab implies a different current query; sidebar activation then delivers a changed route request. Re-clicking the current tab is harmless and leaves its query correct.
- Reset to default only when receiving a new navigation request with absent/invalid craft; preserve current selection during catalog reordering by identity, not index. Test browser/route updates and already mounted selection using the real router.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Either section pending | Local skeleton/progress only; other section/main navigation usable | FR-006 | AT-006 |
| Profile pending | No premature Choose your crafts prompt | FR-006 | AT-004, AT-006 |
| Either read failed | Local localized error + retry, current-lease target only | FR-006, FR-007 | AT-006, AT-007 |
| Loaded craft list empty/unsupported-only | Existing profile editor action | FR-002, FR-004 | AT-002, AT-004 |
| No eligible members | HTTP 200 items array empty; brief member empty state | FR-005, FR-006 | AT-006, IT-002 |
| Account changes mid-read | Clear displayed previous viewer results; new lease loads independently; late old work cannot publish | FR-007 | AT-007, IT-003, REG-004 |
| Requested craft has no projects | Preserve selected tab and use existing Projects empty state | FR-003 | AT-003 |
| Bad craft URL token | Default selection, no crash or preference write | FR-003 | UT-002 |
| Member disappears after listing | Existing profile-unavailable presentation | FR-005, RULE-002 | AT-005 |
| Many crafts, large text, short height | Entire column scrolls; every action reachable; long labels follow existing wrapping/ellipsis conventions | NFR-001 | AT-008, MAN-001, MAN-002 |

Diagnostic changes must follow `docs/development/logging-error-reporting.md`: use supported owners, typed causes/stacks and permitted request correlation; no private payloads or credential/lease logging. Keep provider errors owned by the existing observer path rather than reporting again in the sidebar. If new handler diagnostics are added, include serialized sink positive-field/private-canary verification alongside IT-006 under the existing ID, without introducing telemetry infrastructure.

## 9. Test Implementation Plan

Every test inherits its requirement/AC mappings from `02-acceptance-tests.md`; the affected-area table above maps implementation to ACs. All scenario variants in that document are mandatory, not optional prose.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | AT-003 | New shell discovery widget suite + existing Projects suite | Real router, signed-in profile and recording FakeProjectRepository; Projects already mounted on another craft | No link/route-selection support; query remains wrong |
| 2 | AT-002, UT-001, UT-002, REG-002, REG-003 | Preferred-link provider / Projects suites | TD-001; invalid URL inputs and delayed preferences | No preferred-only provider/input handling; existing regressions may already pass |
| 3 | AT-004 | Shell suite with real existing edit flow and fake profile save | Empty/unknown-only selections; save supported craft | No empty sidebar action or live link derivation |
| 4 | IT-001, IT-002, IT-003 | New recent-member store suite | Real Postgres, TD-002/TD-003; cutoff tie E/F; active/hidden lifecycle states; A/B relationships | No eligible-member read method |
| 5 | IT-004 | Recent-member store and onboarding route suite | TD-005; repeat completion/edit/reindex/departure/reactivation/purge cleanup | Member-order assertion absent; existing completion idempotence may pass |
| 6 | IT-005, IT-006 | Recent-member route and Flutter API adapter suites | Real mux; auth/device fixtures; allowed-key schema and private canaries; omitted fields | Missing route/policy/read/decoding |
| 7 | AT-005, AT-006 | Shell suite | Counts 0/1/4/5, name/avatar fallbacks, controlled futures and retries | No member rows/independent states |
| 8 | AT-007, REG-004 | Shell suite and account-switcher regression suite | Disjoint A/B preferences/member rows; both late-response orders | Missing lease-specific isolation/invalidation |
| 9 | AT-001, AT-008, REG-001 | Shell suite and existing navigation regression suite | Widths 600/900/1200/1201/1440; short height; 1x/2x text; keyboard/semantics | Placeholder remains; new actions not reachable |
| 10 | MAN-001, MAN-002 | Reported iPad format and desktop/device review | TD-006; supported themes, VoiceOver | Requires completed implementation; record observations, not assumed passes |

Mandatory named variant groups to carry into `05-implementation-plan.md`:

- AT-003: fresh navigation; mounted page; second craft; same-craft reselection after manual tab changes; same current craft; late reorder; empty project results; unchanged preferences.
- AT-006: each section pending with other ready; each failure/retry; both pending; both failed; successful empty member response.
- AT-007: cached A cleared at switch; A-before-B and A-after-B; late profile/member reads independently; preference save without switch.
- IT-002: counts 0/1/4/5/6; each exclusion class individually; combined newest exclusions ahead of six eligible rows; all excluded. Seed expired and negated moderation outputs as visible control cases.
- IT-003: A/B/A reads, viewer block/mute and blocked-by behavior, next read after mutation, followed eligible account.
- IT-004: completion preserved on repeat/edit/reindex; departed/deletion-pending/deleting/terminal/unknown owners excluded; active reactivation with retained completion remains old; cleanup removes completion and missing metadata remains omitted.
- IT-006: success/empty/error contract, exact key sets, present/absent fields, private canaries, static route dispatch, unauthorized middleware contract and client decoding.

Use existing valid identifier factories, testdb migration helpers, profile fixtures and deterministic pump helpers. Tests assert handwritten expected sets/order and recorded query identity; do not compute expectations using the production selection helper. Do not use a fake server list to prove SQL exclusions. No external PDS/Render calls are needed.

## 10. Sequencing And Guardrails

- First TDD step: Write AT-003 for an already mounted Projects page, run it and capture the real failing assertion. Make only enough craft-link/navigation behavior to pass; then add variants and refactor.
- Next implement preference derivation/editor behavior, then backend query/contracts, then member presentation/retry/account isolation, followed by layout/regression/device checks.
- Regenerate route/Riverpod/localization code only in the implementation stage, using the checkout's toolchain (`dart run build_runner build --delete-conflicting-outputs` and `flutter gen-l10n` from `app/` where required). Generated outputs belong with their source changes.
- Preserve existing AppView API version, error envelope, auth headers, typed identity boundaries, visibility SQL and deletion protections. No PDS record writes are introduced by member discovery or craft navigation; the existing profile editor retains its normal mediated write flow.
- Member selection and response mapping must not use private join metadata as display copy or return it to the client. Do not log active lease/session contents or capability data.
- No implementation, tests, dependency changes, migration, commit, push or deployment has occurred in this planning stage.
- No broad shell redesign, polling, ranking, follow buttons, new preference ordering, lexicon or infrastructure work.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking evidence limit | Legacy production metadata coverage unknown | Some members may be omitted | Approved missing-metadata policy; no production query/backfill in this scope. Source semantics verified; local tests pending. |
| CPQ-002 | Non-blocking implementation risk | Cached AsyncValue or same-URL navigation could retain incorrect state | Stale account rows or wrong craft | Lease-keyed composition, guarded reads and explicit tab/URL synchronization; AT-003/AT-007 required before completion. |
| CPQ-003 | Non-blocking performance risk | New query has no established production workload evidence | Potential slow selection at scale | Start without migration; inspect local representative query plan if needed. Propose any index separately with evidence. |
| CPQ-004 | Non-blocking platform verification | Real iPad layout/VoiceOver cannot be proven by source inspection | Device-specific UX defects | Automated layout/semantics plus MAN-001/MAN-002; report manual checks pending until observed. |

Blocking design questions: None. DR-001 source semantics and DR-002 operation/schema/tie-break decisions are specified here; DR-003 was corrected in the earlier documents; DR-004 has explicit test sequencing and mandatory variants. Approval to implement remains the workflow exit gate.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md` (Draft for user approval).
- TDD execution plan: `05-implementation-plan.md`.
- Start with: AT-003, mounted-page craft selection and recorded project query.
- Focused first command after the test exists: `just app-test test/router/app_shell_discovery_sidebar_test.dart --plain-name "sidebar craft link selects a mounted Projects tab"`.
- Focused subsequent commands: `just app-test test/projects/pages/projects_page_test.dart`; `just app-test test/projects/providers/preferred_craft_links_test.dart test/profile/data/recent_members_api_client_test.dart`; shell/account-switcher regression tests; `just app-analyze`.
- Backend validation: compose services via `just dev`, host tests via `just test`, required complete gate `just appview-check`. Unit-only mode intentionally skips real Postgres/MinIO tests and is not evidence for IT-001–IT-006. Run focused Go tests using the repository's local test-service environment; do not fabricate credentials or use production database URLs.
- Preserve requirements/AC/test IDs, independent fixture oracles and all mandatory variants. Carry manual evidence limits into the final implementation report.
- No commits are enabled for this workflow; leave this artifact uncommitted until requested.
