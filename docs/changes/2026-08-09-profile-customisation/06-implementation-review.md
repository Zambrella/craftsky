# Implementation Review: Public Profile Customisation

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-08-10
Risk level: Medium

## Summary

The implementation establishes the intended AppView-only persistence and mutation boundary, tolerant public wire model, batched response hydration, shared avatar border renderer, local texture catalogue, Settings editor, and account-fenced authoritative save flow. The focused AppView and Flutter suites pass, the full AppView suite was recorded as passing under the race detector, and static analysis is clean.

The change is not ready to merge because two approved behaviors are incomplete and several Must-level test obligations were marked complete without the planned evidence. The actual Craftsky `ChunkyButton` controls do not consume the approved hover/pressed bundle tones and several secondary controls remain on the global paper palette, so the compact/full profile regions are not completely themed as required. Successful save reconciliation also leaves several identity-bearing caches untouched. Query-plan/count, full surface propagation, and Settings lifecycle/accessibility coverage remain below the approved NFR-006 scope.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior | The profile theme defines the approved hover/pressed tones only for standard Material buttons, but profile actions use `ChunkyButton`, whose custom painter derives interaction overlays independently and whose `themeStyleOf` returns `null`. `ChunkyIconButton` and following/edit actions also explicitly use `BrandSwatchTheme.paper3`. As a result, buttons in the custom-themed compact view and above the full-profile tab bar do not consistently use the selected bundle or its approved interaction tones. | FR-010, FR-011, RULE-001; AC-012; `app/lib/profile/widgets/profile_customisation_theme.dart`; `app/lib/theme/chunky_button.dart:45-143`; `app/lib/theme/chunky_icon_button.dart:21-36`; `app/lib/profile/widgets/profile_actions.dart:97-177`; `app/lib/profile/widgets/profile_card.dart:450-567` | Start with a failing widget/style test that exercises rest, hover, focus, pressed, and secondary action states inside `ProfileCustomisationTheme`. Extend the profile theme through the actual Chunky controls so the selected base/foreground/hover/pressed/soft-container bundle governs every required control without leaking below the full-profile tab bar. |
| IR-002 | Important | Behavior | Successful save reconciliation updates the self-profile cache and invalidates timeline, notifications, several searches, saved posts, and owner collections, but it does not invalidate other retained identity-bearing state such as single-post/thread/comment providers, profile pins, search suggestions/recent results, relationship lists, or project-feed collections. A cached surface can therefore continue rendering the old avatar customisation after the authoritative save. | FR-006, FR-008; AC-007, AC-008; EC-009; `app/lib/profile/providers/profile_customisation_provider.dart:97-125`; compare the account-wide identity cache inventory in `app/lib/auth/providers/account_boundary_provider.dart` | Add failing provider tests for representative already-cached post/thread, suggestion, and account-summary surfaces, then centralize or complete the post-save identity invalidation/update set. Preserve the existing initiating-account fence and do not publish draft values optimistically. |
| IR-003 | Important | Tests | The required bounded-query implementation is visible in code, but the planned statement-count and `EXPLAIN`/index-plan test was never added. The current hydrator test proves one fake batch call and deduplicated inputs; it does not prove statement count as page size grows or that real PostgreSQL uses the intended indexed plan. | NFR-001, NFR-006; AC-019, AC-020; IT-005; `appview/internal/api/profile_customisation_hydrator_test.go`; absent planned `profile_customisation_query_plan_test.go` | Add the real-Postgres IT-005 test covering increasing pages, repeated DIDs, one customisation batch statement, and the primary-key/index plan. Record its red/green and broad-suite evidence. |
| IR-004 | Important | Tests | The approved all-surface avatar regression matrix is not asserted. Production call sites pass customisation through posts, notifications, search, profile, summaries, edit preview, navigation, and account switching, but current customisation-specific widget tests exercise only the shared avatar primitive and profile views. This leaves FR-008 propagation regressions unprotected. | BR-002, FR-008, NFR-006; AC-002, AC-008, AC-010, AC-020; AT-002, REG-004; `app/test/profile/widgets/profile_avatar_test.dart`; no customisation assertions in the affected feed/notification/search surface suites | Add a focused surface-matrix test or extend each affected public surface suite to provide a non-default identity and assert that the shared avatar receives/renders its colour and thickness. Include feed/thread/quote, notification, search, relationship/account, summary, edit preview, and navigation/switcher representatives. |
| IR-005 | Important | Tests | Settings tests cover live selection, successful save, and failed retry, but do not cover clean/dirty/reverted/saved Back behavior, branded discard confirmation, pending duplicate activation in the widget, load retry, semantics selection state/focus order, supported large text, or both themes. These were explicit Must-level acceptance and accessibility cases. | FR-005, NFR-003, NFR-006; AC-006, AC-007, AC-018, AC-020; UT-008, UT-011, AT-006, AT-010, REG-007; `app/test/settings/profile_customisation_page_test.dart` | Add the missing Settings lifecycle and structural accessibility widget tests. Keep physical VoiceOver/TalkBack and perceptual checks as the documented manual supplement rather than replacing automated semantics/layout coverage. |
| IR-006 | Important | Traceability | `05-implementation-plan.md` marks all Must coverage complete and records IT-005, UT-011, AT-002/REG-004, and full Settings behavior as green even though the corresponding evidence above is absent or materially narrower. It also has no executed IT-009 observability entry. | `docs/changes/2026-08-09-profile-customisation/05-implementation-plan.md:212-229`; NFR-006; completion checklist | During the correction pass, update the execution log and checklist to name only tests and assertions that actually ran. Add the missing evidence or record an explicit unresolved gap; do not retain umbrella test IDs for narrower suites. |
| IR-007 | Suggestion | Risk | The Should-level bounded mutation/fallback observability design was not implemented or tested. Existing generic HTTP instrumentation remains, but there is no customisation-specific bounded result/error-class signal or retired-key diagnostic. | NFR-005; AC-019; IT-009; absent planned `profile_customisation_observability_test.go` | Either add the bounded, privacy-safe signal and IT-009 test from the plan, or explicitly defer NFR-005 in the workflow documents with rationale. Never label metrics with DID or selected catalogue values. |

## Requirement And Test Traceability

- Requirements implemented: AppView ownership and lifecycle; authenticated self-only full replacement; strict catalogue validation; additive nested public customisation; tolerant per-field defaults; one deduplicated batch hydration seam; moderation-shell preservation; local fixed catalogues/assets; explicit-save editor using `AsyncValue`; initiating-account save fencing; shared inside-avatar geometry; compact/full header textures; and the full-profile tab-bar theme boundary.
- Tests implemented: catalogue/default and strict request tests; migration up/down/up and cascade; store default/upsert/retry/isolation/fallback; handler and route auth/device/current-member behavior; typed response defaults and batch hydration; Flutter tolerant models/API mapping; authoritative/failed/late-account save provider behavior; shared avatar width/fallback rendering; local background mapping/rendering; core profile theme boundary; and basic Settings save/failure behavior.
- Unplanned behavior: None identified outside implementation techniques. The JSON response middleware is broader than the DTO-specific hydrator originally described, but it remains bounded to one deduplicated batch lookup and preserves stripped/actorless shells in the covered cases.
- Remaining gaps: IR-001 through IR-006 are required before approval. IR-007 is a non-blocking Should-level decision. MAN-001 through MAN-003 remain manual release/device QA supplements.

## Test Evidence

- Commands reviewed:
  - `go test ./internal/api ./internal/routes -run ProfileCustomisation -count=1`
  - `flutter test test/profile/models/profile_customisation_test.dart test/profile/providers/profile_customisation_provider_test.dart test/profile/widgets/profile_avatar_test.dart test/profile/widgets/profile_header_background_test.dart test/settings/profile_customisation_page_test.dart test/profile/profile_page_test.dart test/profile/widgets/profile_card_test.dart`
  - Recorded implementation evidence: focused real-Postgres profile-customisation tests, `just test`, `dart analyze`, broad `flutter test`, and `git diff --check`.
- Passing evidence: Both focused review commands pass. The implementation record reports the full AppView race suite passing, `dart analyze` with no issues, and 1,398 Flutter tests passing.
- Failing or skipped tests: The broad Flutter run retains two pre-existing `auth_complete_page_test.dart` router-harness failures in untouched code. IT-005 query-plan/count, IT-009 observability, UT-011 accessibility, the complete REG-004 surface matrix, and MAN-001 through MAN-003 were not run as designed.

## Risk Review

- Risk level: Medium.
- Risk notes: Storage, authorization, strict input validation, and AppView/PDS separation are well isolated. Remaining risk is concentrated in visible theme fidelity, stale client identity caches after save, and insufficient regression evidence across a broad shared-avatar contract. No Lexicon, PDS, Tap, blob, billing, or destructive data change was introduced.
- Approval notes: Do not merge or hand off as complete until IR-001 through IR-006 are addressed and the relevant focused and broad verification is rerun. The two unrelated auth test failures may remain documented if reconfirmed unchanged.

## UI Polish Recommendation

- Recommendation: Optional.
- Reason: Required behavior and coverage corrections come first. Once they pass, a bounded polish/device-QA pass would be useful because the feature introduces six palettes, six textures, responsive profile boundaries, and new accessible choice controls.
- Suggested polish notes: Review all palette/texture combinations at representative compact/full sizes, verify the selected/disabled/focus states of choice chips and Chunky controls, and complete the planned VoiceOver/TalkBack, hardware-keyboard, large-text, dark-theme, and colour-vision checks without changing product behavior.

## Handoff Back To TDD Builder

- Required fixes: IR-001 through IR-006.
- Suggested next failing test: Add a compact-profile widget test using a non-default bundle that inspects an actual primary `ChunkyButton`, a secondary `ChunkyIconButton`, and their hover/pressed states; assert the approved base/foreground/hover/pressed/soft-container values and confirm the full-profile tab bar remains under the app theme.
- Verification to rerun:
  - New focused theme/control, cache invalidation, query-plan, surface-matrix, and Settings lifecycle/accessibility tests.
  - Existing profile customisation AppView and Flutter focused suites.
  - Real-Postgres migration/store/hydration/query-plan suites.
  - `just test`.
  - Full `flutter test` with the two unrelated auth harness failures accounted for explicitly.
  - `dart analyze`.
  - `git diff --check`.
