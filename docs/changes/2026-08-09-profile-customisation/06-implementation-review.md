# Implementation Review: Public Profile Customisation

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-08-10
Risk level: Medium

## Summary

The implementation is ready to merge or hand off. It satisfies the approved AppView-only persistence and mutation boundary, complete additive public identity contract, strict current mutation catalogue, tolerant per-field defaults, bounded response hydration, authoritative account-fenced save lifecycle, shared inside-avatar border rendering, bundled local textures, scoped profile colour themes, and Settings editing flow.

The correction pass resolves the six blocking findings from the first review. The actual custom-painted Chunky controls now consume the complete selected colour bundle; post-save invalidation has one auditable identity-cache inventory; real-Postgres query-count and index-plan evidence exists; non-default customisation is asserted across the affected Flutter surfaces; Settings lifecycle and structural accessibility coverage is materially complete; and `05-implementation-plan.md` now distinguishes executed evidence from deferred work.

The remaining notes are non-blocking. NFR-005 is a Should-level custom observability enhancement and is explicitly deferred while generic HTTP instrumentation remains. MAN-001 through MAN-003 still require representative device/assistive-technology and perceptual QA. The broad Flutter run also continues to report the same two pre-existing router-harness failures in untouched `auth_complete_page_test.dart`; no profile-customisation test fails.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-007 | Suggestion | Risk | Customisation-specific bounded mutation/fallback telemetry remains deferred. Existing generic HTTP instrumentation remains and the implementation introduces no DID, catalogue choice, asset filename, or URL metric labels. | NFR-005; AC-019; IT-009; `05-implementation-plan.md` Steps 22–26 and execution log | Non-blocking follow-up: add a bounded operation/result/error-class signal and IT-009 coverage before production-readiness work if feature-specific diagnostics are wanted. |

## Requirement And Test Traceability

- Requirements implemented: All Must BR-001 through BR-005, FR-001 through FR-013, NFR-001 through NFR-004 and NFR-006, and RULE-001 through RULE-005 are represented in implementation and automated evidence. The Should-level NFR-005 signal is explicitly deferred.
- Tests implemented: Catalogue/default and strict request tests; migration up/down/up and cascade; store defaults/upsert/retry/isolation/fallback; authenticated route and boundary behavior; typed response defaults and one-batch hydration; real-Postgres query count/index plan; tolerant Flutter models and repository mapping; authoritative/failed/stale-account save behavior; centralized retained-identity invalidation; exact avatar geometry and fallback rendering; non-default affected-surface propagation; fixed local textures and theme boundaries; real Chunky interaction states; and Settings loading/save/failure/Back/pending/semantics/focus/text-scale/theme behavior.
- Unplanned behavior: None identified. The JSON response middleware is broader than the DTO-specific hydrator sketched in the coding plan, but remains one deduplicated indexed batch lookup per successful JSON response, decorates only retained DID-and-handle identities, and preserves covered blocked/unavailable/actor-free shells.
- Remaining gaps: No Must-level implementation or automated-test gaps identified. IR-007 is a non-blocking Should-level deferment. MAN-001 through MAN-003 remain manual release/device QA supplements.

## Test Evidence

- Commands reviewed and rerun during this review:
  - `flutter test test/profile/widgets/profile_customisation_controls_test.dart test/profile/widgets/profile_card_test.dart test/profile/providers/profile_identity_cache_invalidator_test.dart test/profile/providers/profile_customisation_provider_test.dart test/feed/widgets/post_card_test.dart test/notifications/notifications_page_test.dart test/search/search_page_test.dart test/shared/widgets/post_summary_test.dart test/profile/widgets/edit_profile_banner_avatar_test.dart test/router/app_shell_account_switcher_test.dart test/settings/profile_customisation_page_test.dart`
  - `go test ./internal/api ./internal/routes -run 'ProfileCustomisation|IdentityCustomisation' -count=1`
  - `dart analyze`
  - `git diff --check dca78f94..HEAD`
- Passing evidence: The focused Flutter correction matrix passes 128 tests. The focused AppView API/routes command passes. Static analysis reports no issues and the committed implementation diff passes whitespace checks. The implementation record also reports the full AppView `just test` race suite passing and the real-Postgres migration/store/hydration/query-plan cases passing.
- Failing or skipped tests: The recorded full Flutter run passes 1,409 tests and reproduces only two pre-existing untouched `auth_complete_page_test.dart` failures caused by a `MaterialApp` harness without `GoRouter`; neither the production page nor that test changed. IT-009 is intentionally not claimed as executed. MAN-001 through MAN-003 have not been performed and remain manual supplements.

## Risk Review

- Risk level: Medium.
- Risk notes: The change is cross-cutting because public identity data and avatar presentation appear throughout the app. The highest-risk boundaries are protected by owner-DID persistence, strict authenticated full replacement, one set-based hydrator, tolerant defaults, initiating-account fencing, centralized cache invalidation, shared avatar geometry, local-only assets, and scoped theme tests. No Lexicon, PDS, Tap, blob, billing, or destructive-data behavior was added.
- Approval notes: All blocking implementation-review findings are resolved. The two unrelated Flutter harness failures and the deferred Should/manual work do not prevent implementation approval, but should remain visible in merge and release handoff notes.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The required visual behavior and structural accessibility checks are implemented and automated. A small polish/device-QA pass could still improve confidence across the six palettes, six textures, responsive profile presentations, and physical assistive-technology behavior without changing product rules.
- Suggested polish notes: Review all palette/texture combinations at representative compact/full widths, confirm focus/disabled states on physical targets, and complete VoiceOver/TalkBack, hardware-keyboard, maximum-text-scale, dark-theme, and colour-vision checks. Treat any behavior, data, API, or validation change discovered there as a new TDD finding rather than polish.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: None for the approved implementation. If IR-007 is later taken up, begin with a focused recorder test for bounded mutation result/error classes and retired-key fallback diagnostics that rejects identifiers and selected values as labels.
- Verification to rerun: Before merge, retain the existing full AppView result, rerun the full Flutter suite if the unrelated router harness changes, and keep the two known failures explicitly separated from feature failures. Complete MAN-001 through MAN-003 on representative targets before a production release.
