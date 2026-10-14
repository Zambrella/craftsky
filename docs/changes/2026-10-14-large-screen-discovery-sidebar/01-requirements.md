# Requirements: Large-screen discovery sidebar

## 1. Initial Request

Replace the empty right-hand area on large screens with quick links to the active user's preferred crafts, opening Projects with the chosen craft tab selected, and the five most recent CraftSky members.

## 2. Current Codebase Findings

- `app/lib/router/app_shell.dart`: a shared `_DesktopSidebarPlaceholder` currently renders an empty card. It appears only for the desktop form factor.
- `app/lib/theme/form_factor.dart`: desktop means widths above 1,200 logical pixels; device type alone does not determine visibility.
- `app/lib/projects/providers/preferred_project_crafts_provider.dart`: uses the signed-in profile's `crafts` to place preferred crafts first, followed by all remaining supported crafts. It is not a preferred-only list.
- `app/lib/projects/options/project_option_catalogs.dart`: existing catalog supplies supported craft identities and labels; preference groups retain catalog order.
- `app/lib/projects/pages/projects_page.dart`: selects the first tab initially, preserves an explicitly selected craft during preference reordering, and filters project results by the active craft.
- `app/lib/router/router.dart`: `ProjectsRoute` currently has no craft-selection input. Existing profile navigation should be reused for member rows.
- `app/lib/profile/data/profile_api_client.dart`: supports profile reads; no newest-members operation was identified in this client.
- `appview/migrations/000065_account_onboarding_completion.up.sql`: records account onboarding completion with `account_did` and `completed_at`.
- `appview/migrations/000008_craftsky_profiles.up.sql`: profile `created_at` is an indexing timestamp, not proven to be the member's join date.
- API constraints: AppView-mediated reads, authenticated `/v1/` contracts, camelCase JSON, standard error envelope; see `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`.
- Existing test locations: `app/test/projects/pages/projects_page_test.dart`, `app/test/router/app_shell_navigation_menu_test.dart`, and shell account-switcher tests.
- Validation commands for later stages: `just app-test <test paths>` and `just appview-check` if backend behavior changes. No application tests are run in this requirements stage.

## 3. Clarifying Questions And Decisions

### Q1: What should occupy the space?

Answer: The user selected preferred-craft links and the five most recent CraftSky members, then confirmed proceeding to requirements.

Decision / implication: Two compact sections: “Your crafts” above “New to CraftSky.” Project summaries, drafts, and post conversations are excluded.

### Q2: What counts as a new member?

Answer: The preceding proposal used joining CraftSky rather than creating an AT Protocol account. The exact timestamp was not explicitly selected.

Decision / implication: Assume first recorded onboarding completion for this draft (ASM-002); do not substitute a profile indexing date. The user approved this requirements baseline. Coding planning must verify that the stored timestamp supports this definition before member selection is implemented; approval is not evidence of timestamp immutability or legacy coverage.

### Q3: Where should the sidebar appear?

Answer: The request concerns the existing large-screen empty space.

Decision / implication: Retain the shared shell's existing visibility scope and breakpoint (ASM-001), rather than adding a feed-only sidebar or changing responsive navigation.

## 4. Candidate Approaches

### Option A: Preferred crafts and newest members — selected

Summary: Two compact sections with direct navigation.

Pros: Matches the confirmed request; supports project discovery and community visibility with understandable ordering.

Cons: Needs explicit craft navigation and a viewer-aware member listing.

Risks: Join-date semantics and visibility filtering need care.

### Option B: Projects, drafts, or conversation details

Summary: Use the column for personal work or selected-post replies.

Pros: Useful for continuing work or reading discussions.

Cons: Different behavior from the selected direction; conversations require broader interaction changes.

Risks: Scope expansion. Deferred.

## 5. Recommended Direction

Replace the placeholder with preferred-only craft links and up to five eligible newest members. Reuse craft labels, profile preferences, Projects tabs, and profile navigation. Keep selection deterministic and account-aware.

Why: Delivers the two requested shortcuts while preserving the existing chronological browsing experience.

## 6. Problem / Opportunity

The wide layout reserves a visible but empty column. It can provide immediate access to personally relevant projects and introduce people who have recently joined the community.

## 7. Goals

- G-001: Open project browsing for a preferred craft in one action.
- G-002: Introduce up to five recently joined, eligible CraftSky members.
- G-003: Make the existing sidebar useful without disrupting the main content or compact layouts.

## 8. Non-Goals

- NG-001: Follow buttons in the sidebar, recommendations based on engagement, trending lists, editorial spotlights, or craft-based ranking of members.
- NG-002: New craft preferences, custom preference ordering, or changes to the craft catalog or lexicons.
- NG-003: Sidebar project summaries, drafts, reply panels, pagination, or a member directory.
- NG-004: Production deployment, production data backfill, or changes to responsive breakpoints.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in viewer | Active CraftSky account using a wide layout | Preferred craft shortcuts and safe member discovery |
| Recent member | Eligible account that recently joined CraftSky | Public profile discovery under existing visibility rules |

## 10. Current Behavior

The shared desktop shell shows an empty sidebar card. Projects starts with the first craft tab in preference-aware catalog order. There is no craft-selection input on its route.

## 11. Desired Behavior

The sidebar shows “Your crafts” followed by “New to CraftSky.” Craft links open the corresponding Projects tab, including when Projects is already open. Member rows open profiles. Both sections handle loading, empty, and failed states independently.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Give wide-screen users direct access to projects for their preferred crafts. | Make reserved space useful. | User request | AC-002, AC-003 |
| BR-002 | Business | Must | Introduce the newest eligible CraftSky members. | Support community discovery. | User request | AC-005, AC-006 |
| FR-001 | Functional | Must | Replace the empty card with the two sections, crafts above members, wherever the existing shared desktop sidebar appears. | Preserve current layout scope. | Codebase; ASM-001 | AC-001 |
| FR-002 | Functional | Must | Show each supported craft selected in the active profile once, using existing craft labels and catalog order; omit unselected and unsupported crafts. | Preferred-only shortcuts. | User request; codebase | AC-002 |
| FR-003 | Functional | Must | Activating a craft link shall open Projects with that craft visibly selected and project results filtered to it, including from an already mounted Projects page. | Make the shortcut reliable. | User request | AC-003 |
| FR-004 | Functional | Must | With no supported preferred crafts, offer “Choose your crafts” leading to the existing profile craft editor. | Provide an actionable empty state. | Confirmed preceding proposal | AC-004 |
| FR-005 | Functional | Must | Show up to five distinct eligible members with avatar, display name and handle; use existing profile fallbacks where fields are absent. Activating a row shall open that member's profile. | Recognizable, compact discovery. | User request; preceding proposal | AC-005 |
| FR-006 | Functional | Must | Handle each section's loading, empty and failure states independently; failures shall offer retry without preventing use of the other section or main content. | Keep optional discovery resilient. | Requirements refinement | AC-008 |
| FR-007 | Functional | Must | After active-account changes or preference updates, show the active account's crafts and member eligibility; do not display a previous account's viewer-specific results. | Account correctness. | Existing multi-account shell | AC-009 |
| NFR-001 | Non-functional | Must | Keep all sidebar content reachable without clipping at supported desktop widths, short heights, and enlarged text; support keyboard activation, meaningful accessible names and localized copy using existing app conventions. | Usable tablet/desktop experience. | Screenshot; existing UI conventions | AC-010 |
| RULE-001 | Business rule | Must | Order members by CraftSky join time descending, with deterministic ordering for equal timestamps; never rank by popularity, engagement or AT Protocol account age. | Transparent newest-member meaning. | Preceding proposal; ASM-002 | AC-006 |
| RULE-002 | Business rule | Must | Exclude the viewer, accounts blocked or muted by the viewer, accounts blocking the viewer, and accounts unavailable under existing moderation or lifecycle visibility rules; apply eligibility before taking five. | Preserve viewer safety and visibility. | Preceding proposal; existing visibility policy | AC-007 |
| RULE-003 | Business rule | Must | Use authenticated AppView reads for member discovery, expose only permitted public profile summaries, and conform to the existing API contract for any new operation. | Preserve architecture and privacy. | Repository instructions; API spec | AC-011 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | FR-001 | Given a desktop shell above 1,200 logical pixels, the two sections replace the placeholder on its existing routes; at or below that boundary, the sidebar remains absent and existing navigation remains usable. |
| AC-002 | BR-001, FR-002 | Given selected, duplicate, unselected and unsupported craft values, only distinct supported selections appear, with existing labels in catalog order. |
| AC-003 | BR-001, FR-003 | Given Projects is either unopened or already showing another craft, activating a sidebar craft selects that craft and uses it for the project query; subsequent preference reordering does not override the explicit selection. |
| AC-004 | FR-004 | Given a loaded profile with no supported selected crafts, “Choose your crafts” opens the existing editing flow; saved preferences appear in the sidebar without restarting the app. |
| AC-005 | BR-002, FR-005 | Given at least five eligible members, exactly five distinct profile rows appear; given fewer, all available rows appear. Missing display name or avatar uses existing fallbacks, and activating any row opens the correct profile. |
| AC-006 | BR-002, RULE-001 | Given eligible members with differing join times and a timestamp tie, results are newest first with repeatable tie ordering; profile edits, reindexing and AT Protocol account age do not make an established member newer. |
| AC-007 | RULE-002 | Given a newest-candidate pool containing the viewer, blocked, muted, blocking, moderation-hidden and lifecycle-unavailable accounts, none appear; older eligible members fill available slots up to five. |
| AC-008 | FR-006 | Given either section is loading or fails, its local state is visible while the other section and main page remain usable; retry can recover. A successful empty member response displays a brief empty state rather than an error or blank card. |
| AC-009 | FR-007 | Given an account switch or successful craft-preference update, the sidebar reflects the active viewer and does not show stale viewer-specific data from the prior account. |
| AC-010 | NFR-001 | Given supported desktop dimensions, a short viewport and enlarged text, all sidebar actions remain reachable without overflow; keyboard and assistive technology can identify and activate links, and copy comes from localization resources. |
| AC-011 | RULE-003 | Given member discovery requests, unauthenticated access is rejected under the standard API contract, authenticated responses contain only permitted public summaries and conform to camelCase JSON/error conventions, and private join metadata is not exposed unnecessarily. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Preferences still loading | Show loading rather than prematurely prompting selection. | FR-006 |
| EC-002 | Craft has no projects | Keep the requested tab selected and show the existing Projects empty state. | FR-003 |
| EC-003 | Invalid or unsupported craft navigation input | Use the existing default craft behavior without crashing; do not change preferences. | FR-003 |
| EC-004 | All members excluded or no members exist | Show a brief member-section empty state; retain crafts. | FR-005, FR-006, RULE-002 |
| EC-005 | Member becomes unavailable before row activation | Existing profile unavailable behavior applies. | FR-005, RULE-002 |
| EC-006 | Account switches during an in-flight read | Late prior-account responses must not populate the new viewer's sidebar. | FR-007 |
| EC-007 | Many preferred crafts or long names | All selected supported crafts remain reachable; no arbitrary cap or clipping. | FR-002, NFR-001 |

## 15. Data / Persistence Impact

- New fields: None required by the user-facing behavior; reuse profile crafts and verified membership metadata where suitable.
- Changed fields: None intended.
- Migration required: Not established. Planning must verify onboarding timestamp semantics and query/index needs. No migration or production backfill is authorized by this document.
- Membership verification deliverable (DR-001): Coding planning must document the onboarding write and repeat-completion behavior, relevant account lifecycle transitions, and how accounts without completion metadata are treated, with source references. A local fixture can verify lifecycle semantics; claims about actual legacy coverage require separate evidence. No production inspection or backfill is required by this planning task. If existing metadata cannot represent first completed onboarding, record the discrepancy and resolve the design before implementing member selection. Do not substitute indexing dates.
- Backwards compatibility: Production data must be preserved. Craft links must not modify saved preferences. Older members without trustworthy join metadata must not be assigned fabricated recent dates.

## 16. UI / API / CLI Impact

- UI: Shared desktop sidebar, craft-selecting Projects navigation, existing profile and craft-edit flows.
- API: Viewer-aware newest-member read capability is needed. Coding planning must specify reuse versus extension, method/path and auth policy, an explicit public-summary field allowlist and null/fallback behavior, and deterministic ordering including the tie-breaker (DR-002). Define expected serialized success/error examples and expected tied-member identities for test design. These interface choices must preserve RULE-001 through RULE-003 and the existing API architecture spec.
- CLI: None identified.
- Background jobs: None expected.

## 17. Security / Privacy / Permissions

- Authentication: Use the active signed-in account for discovery.
- Authorization: Apply existing profile visibility, relationship exclusions, moderation and owner-lifecycle rules on the server before limiting results.
- Sensitive data: Return public profile presentation data only; no OAuth credentials, private preferences of other members, or unnecessary onboarding metadata.
- Abuse cases: Hidden accounts must not reappear through this alternative discovery surface. Existing API rate protections apply.

## 18. Observability

- Events: No new analytics events required.
- Logs: Use existing supported error-reporting paths and `docs/development/logging-error-reporting.md` for any diagnostic changes; preserve permitted causes and avoid private payloads.
- Metrics: None newly required.
- Alerts: None newly required.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Onboarding completion may omit existing members or fail to represent immutable first joining. | Incorrect newest-member ordering. | Verify timestamp lifecycle during coding planning and document legacy-coverage evidence limits; omit missing trustworthy metadata; never use reindexing as joining. |
| RISK-002 | Discovery bypasses existing visibility or limits before filtering. | Hidden profiles leak or fewer than five eligible members appear. | Server-side eligibility followed by ordering/limit; integration tests with excluded candidates. |
| RISK-003 | Projects branch remains mounted and ignores navigation selection. | Shortcut opens the wrong tab. | Cover fresh navigation and repeated selection on an existing page. |
| RISK-004 | Viewer-specific results survive account switching. | Wrong preferences or relationship-filtered results appear. | Verify account-scoped state and late-response handling. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Sidebar remains on all existing shared-shell routes above 1,200 logical pixels. | Visibility requirements must be revised if it should be feed-only. |
| ASM-002 | First recorded completed onboarding defines joining CraftSky; accounts without trustworthy completion metadata are omitted pending a separate decision. | Membership definition/data coverage needs revision before implementation. |
| ASM-003 | Five means newest eligible members, excluding the viewer and visibility exclusions; already-followed members remain eligible. | Eligibility requirements must be revised if discovery should mean unfollowed members only. |
| ASM-004 | No live updates or timed polling are required; ordinary load, account changes and explicit retry suffice for this first version. | Refresh requirements need expansion if continuous freshness is desired. |

## 21. Open Questions

- Blocking for coding planning: None. ASM-001 through ASM-004 are the user-approved baseline.
- Implementation dependency (DR-001): Verify and document onboarding timestamp semantics and missing-metadata treatment during coding planning, as specified in section 15. If verification reveals incompatible semantics, resolve that discrepancy before implementing member selection. Legacy production coverage remains unverified unless separately evidenced; missing trustworthy metadata is omitted under ASM-002.
- Interface-design dependency (DR-002): Finalize the member read contract, response allowlist and deterministic tie-breaker in the coding plan, as specified in section 16.
- Any proposed production backfill remains a separate reviewed decision.

## 22. Review Status

Status: Approved baseline; revised to address document-review notes

Risk level: Medium

Review outcome: Approved with notes in `03-document-review.md`; risk remains Medium.

Reviewer: User approval (“LGTM”) followed by Codex document review.

Date: 2026-10-14

Notes: The user approved the requirements before test design and requested this revision of both earlier documents after review. DR-001 and DR-002 are now explicit planning deliverables, not completed verifications. DR-003 is addressed by correcting this status. DR-004 is addressed by the mandatory variant checklist in `02-acceptance-tests.md`. No application behavior, stable IDs or scope was changed.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001–BR-002, FR-001–FR-007, NFR-001, RULE-001–RULE-003.
- Suggested test levels: craft selection/order and eligibility unit tests; sidebar, navigation, mounted-page selection and account-switch widget tests; server integration/API tests for ordering, exclusions, auth and response contracts; manual wide-screen, short-viewport, keyboard and enlarged-text checks.
- Required test-design detail (DR-004): Carry every scenario variant into the implementation checklist, especially mounted-page reselection, both account-response orders, independent section states, and excluded-candidate cardinality. Do not implement only headline scenarios.
- Blocking open questions: None for coding planning. Membership verification and interface decisions above must be completed before the member data path is implemented.
