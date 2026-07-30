# Implementation Re-Review: Post And Content Languages

## Verdict

Status: Approved

Reviewer: Codex

Date: 2026-07-30

Risk level: High

## Summary

The correction pass resolves both findings from the 2026-07-29 implementation review. IT-025 now exercises the approved 20,000-row PostgreSQL plan contract with the production visibility predicate, the private preference primary-key lookup, and the actual `craftsky_posts_langs_gin` index. The Flutter catalogue now supplies friendly English names for all 184 selectable tags from the pinned official Bluesky snapshot, while valid external or future tags retain a lossless raw fallback.

The wider implementation remains aligned with the approved architecture: post languages are public PDS record data, account language preferences remain private in the AppView, App language remains device-local and English-only, filtering is server-authoritative and occurs before pagination on all eight in-scope browse/discovery surfaces, deliberate/contextual exceptions remain separate, and no Lexicon change was introduced.

## Findings

None identified.

## Resolved Prior Findings

| ID | Resolution | Evidence |
|---|---|---|
| IR-001 | Resolved | `TestIT025RepresentativeLanguagePlansUsePreferencePKAndGINIndexes` seeds 20,000 posts with 5% exact matches, runs `ANALYZE`, checks `account_language_preferences_pkey`, executes the production ownership-or-overlap predicate, requires `craftsky_posts_langs_gin`, and rejects a post-table sequential scan. A separate review diagnostic with the planner's default sequential-scan setting also selected a bitmap OR over the author index and language GIN index. The database-backed focused and complete AppView suites pass. |
| IR-002 | Resolved | The selectable catalogue contains the exact 184-tag pinned set and a distinct friendly English label for every selectable tag. Flutter and Go pin the same sorted tag fingerprint, Flutter verifies that every selectable value renders a non-code label, both settings and composer selectors search the shared friendly-label function, and unknown stored/external tags preserve their raw value as fallback. |

## Requirement And Test Traceability

- Requirements implemented:
  - Private per-account Primary and Content language preferences, atomic device-locale initialization, public post language propagation, exact tag matching, pre-pagination eligibility, ownership visibility, Saved Posts visibility, deliberate/contextual exceptions, cache invalidation, and composer defaults remain present.
  - Empty Content languages retain show-all behavior; untagged or invalidly tagged posts are shown as-is only on the approved exception surfaces and are otherwise hidden by active filtering.
  - App language is an independent, device-local English-only setting with future languages signposted.
  - No client-supplied browse filter or Lexicon change was introduced.
- Tests implemented:
  - The approved Go, PostgreSQL, Flutter unit, provider, widget, composed-flow, migration, visibility, pagination, exception, and query-plan coverage is present.
  - UT-008 and UT-009 now cover the exact selectable tag snapshot, complete friendly-label availability, and lossless fallback behavior.
  - IT-025 now covers the approved preference and post-language plan contract.
- Unplanned behavior:
  - None identified.
- Remaining release gates:
  - MAN-001, MAN-002, and MAN-003 remain explicit manual release checks.

## Test Evidence

Fresh re-review verification:

- `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/languages ./internal/db ./internal/routes -count=1`
  - Passed.
- `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./... -count=1`
  - Passed for every AppView package.
- Focused Flutter catalogue, selector, preferences, Saved Posts model, router, account-switcher, and composer regression command
  - Passed all 34 tests.
- `dart analyze`
  - Passed with no issues.
- `git diff --check`
  - Passed.
- `git diff --name-only -- lexicon`
  - Returned no paths.
- Rollback-only PostgreSQL review diagnostic using 20,000 representative posts and the planner's default settings
  - Used a bitmap index scan on the language GIN index plus the existing author index, with no sequential scan.

Implementation-pass evidence read back from `05-implementation-plan.md`:

- The broader focused Flutter feature command passed all 143 tests.
- The final complete Flutter run passed 1,138 tests and retained exactly one unrelated existing failure in `instagram_migration_page_test.dart`: the test expects a `Notification settings` `InkWell`, while unchanged production uses a `TextSpan` recognizer.

Failing or skipped checks:

- No feature-scoped automated check is failing.
- The unrelated Instagram widget-test failure remains outside this change.
- MAN-001, MAN-002, and MAN-003 were not executed and remain release gates rather than automated review evidence.

## Risk Review

- Risk level:
  - High, unchanged, because the feature changes private account preferences, record ingestion, and eligibility across multiple paginated surfaces.
- Risk notes:
  - The server remains the authority for content-language policy; clients cannot broaden results with request parameters.
  - Preferences remain private, account-scoped AppView data and are not placed on the public PDS.
  - Language eligibility remains inside SQL before cursor ordering and limits.
  - The automated plan test disables sequential scans for deterministic index-path evidence. The independent default-planner diagnostic confirmed that the representative query naturally selects `craftsky_posts_langs_gin`; MAN-003 still covers the approved 100,000-row latency and buffer thresholds before release.
  - Exact tag preservation and raw fallback prevent valid external metadata from being discarded.
- Approval notes:
  - The implementation is approved against the automated requirements and test design.
  - Complete the three documented manual checks before release.

## UI Polish Recommendation

- Recommendation: Optional
- Reason:
  - The behavior is complete, but a short device pass could compare the bespoke settings and composer selectors with the supplied Bluesky reference and check spacing, keyboard/focus flow, selected-state announcements, and enlarged layouts.
- Suggested polish scope:
  - Keep behavior, persistence, filtering, and catalogue semantics unchanged.
  - Limit changes to visual consistency, accessibility presentation, and copy refinements found during device review.

## Completion Checklist

- [x] All Must requirements covered by tests or documented manual gates
- [x] All planned feature-scoped Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] `05-implementation-plan.md` read back
- [x] Review completed
- [x] Manual device checks retained as explicit release gates
- [x] Automated representative query-plan check passed
- [x] Production-scale performance check retained as the MAN-003 release gate
