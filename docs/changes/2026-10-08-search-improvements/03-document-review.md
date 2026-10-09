# Document Review: First Search Improvements

## Verdict

Status: Approved with notes

Reviewer: Codex, document review against the user-confirmed scope

Date: 2026-10-09

Risk level: Medium

## Summary

The requirements and acceptance test specification are ready for coding-plan work. Every active Must requirement has acceptance criteria and planned test coverage. The documents consistently preserve separate Posts and Projects results, all meaningful query concepts, declared-English word forms, three curated craft equivalences, conservative single-word correction, reliable-before-typo ordering and existing visibility/API boundaries.

The user-authorized simplifications are preserved: no performance measurement obligations, no literal-before-synonym ranking guarantee and no matching-configuration versioning or cross-deployment pagination guarantee. NFR-003 and AC-013 remain retired. The relevance benchmark checks retrieval and ordering only.

The findings below are non-blocking test-design and engineering clarifications. They do not change product scope. This review approves proceeding to coding planning; it does not authorize implementation, migrations or production actions. No runtime tests were run.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests | The first-failing-test handoff describes an unknown-language “literal-only contrast.” A `sock` query could legitimately retrieve an unknown-language `socks` record through the separately permitted one-word typo path. Absence from all results would therefore test a stronger rule than the requirements establish. | `01-requirements.md` FR-001, FR-003, AC-003; `02-acceptance-tests.md` UT-002, IT-001, section 11 | In the coding plan, distinguish reliable language-normalized eligibility from typo eligibility. Test the normalizer or reliable tier directly, or use a controlled word-form fixture outside the chosen correction boundary. Preserve exact literal retrieval for unknown/non-English records; do not invent a blanket ban on their typo matches. |
| DR-002 | Suggestion | Tests | AT-001 is labelled a Scenario Outline without inline Examples or placeholders; its intended cases live in TD-001. AT-003 has a `pair` Examples column that is not referenced by a placeholder, and its extra-concept exclusion lacks an explicit negative-record setup in the outline. The surrounding prose and IT-002 identify the intended cases, so behavior is covered, but a literal Gherkin translation would be incomplete. | `02-acceptance-tests.md` AT-001, AT-003, IT-002, TD-001 | Before translating these into executable tests, use explicit table-driven cases for query, tab, source wording, useful URI and negative URI. If retaining Gherkin, bind placeholders to Examples and add the missing-concept Given. Keep both directions for all three equivalences and both tabs. |
| DR-003 | Suggestion | Risk | One-edit correction, minimum four-character word length and eight candidates per query word are explicitly proposed engineering defaults. Their selection order and confidence interpretation are not yet finalized. A cap applied before visibility or using an unstable vocabulary order could suppress a permitted visible correction or change pagination on unchanged data. | `01-requirements.md` FR-003, FR-006, NFR-002, RULE-002; `02-acceptance-tests.md` UT-004, UT-007, IT-009, IT-010, GAP-001 | Finalize deterministic confidence, candidate selection and finite bounds in the coding plan. State how eligibility precedes candidate caps, how equal candidates are ordered and how unchanged-rule pagination remains deterministic. Update boundary expectations if proposed defaults change; retain protected codes and at most one corrected word. No performance assessment is required. |
| DR-004 | Suggestion | Tests | TD-001 specifies useful records and designated negatives, but the six distractors per case are a fixture-construction instruction rather than enumerated content. Some combined rows require query-specific negatives, as the document correctly notes for `knit` versus `sock`. A poorly assembled corpus could make top-five assertions trivial or exclude a valid topic match. | `01-requirements.md` BR-001, AC-001, RISK-005; `02-acceptance-tests.md` AT-001, TD-001 and its query table, GAP-004 | Make each benchmark query/tab case independent and explicit when constructing the test corpus: concrete authored fields, declared language, useful/negative URI sets and rationale. Add plausible distractors, keep the corrected-match tier exception separate and do not designate knitting hats negative for `knit`. This needs fixture design, not analytics or collecting user queries. |

## Traceability Review

- Planning to requirements: The initial request and accepted direction are embedded in `01-requirements.md` sections 1–5. There is no separate `00-initial-prompt.md`; this does not create a gap because the scope decisions and user-authorized revisions are recorded in section 3 and the conversation.
- Requirements to acceptance criteria: All 13 active Must IDs have at least one linked acceptance criterion. All 16 active criteria link to active requirements. The numbering gaps reflect retired NFR-003/AC-013, not missing coverage.
- Acceptance criteria to tests: The coverage matrix covers every active criterion. Acceptance scenarios and unit/integration/regression cases reference active requirement and criterion IDs. Automated validation found no dangling test/data/gap IDs among the 44 defined IDs. IT-014 provides additional update/delete convergence coverage beyond the summary matrix's listed tests.
- Stable identifiers: Existing requirement, criterion and test IDs are preserved. Fixture prefixes now use `post-` and `project-` consistently. No reviewed document was rewritten during this stage.

## Coverage Review

- Must requirements covered: BR-001; FR-001 through FR-006; NFR-001 and NFR-002; RULE-001 through RULE-004. Coverage is designed, not implemented or verified at runtime.
- Retrieval and precision: Word forms, filler handling, exact names, non-English/unknown metadata, bidirectional equivalences, multiword concept alternatives, cross-field concepts, missing-concept negatives, protected codes and one-versus-two-word corrections have concrete tests.
- Ranking and pagination: Controlled primary/supporting-field comparisons, tier precedence, duplicate tags, popularity neutrality, descending time/URI ties, dual-qualified records, small pages, tier transitions and wrong-query/tab cursors are covered. Tests do not add a literal-versus-synonym priority or continuity across data/rule changes.
- Security and data boundaries: Reliable/synonym/typo visibility paths, hidden-candidate crowding, real lifecycle predicates, private-record and quote boundaries, actual serialized diagnostic sink canaries, authentication/hydration, no extra history writes and no external search/PDS mutations have verification paths.
- Existing records: IT-007 addresses pre-existing-row upgrade/backfill behavior; IT-014 addresses update, replay and deletion convergence. The schema-dependent harness is properly deferred to coding planning under GAP-002.
- Missing or weak coverage: No uncovered active Must requirement or acceptance criterion. DR-001 through DR-004 identify clarifications to apply during planning/test construction. Proposed defaults and exact fixture/migration setup must not be mistaken for settled implementation details.
- Manual-only coverage: None. Practical Go/PostgreSQL and Flutter automation targets are identified. Relevance fixture review is document/test-data review, not a new manual release gate.
- Scope exclusions checked: Performance metrics, latency/query-plan measurement, load testing, numerical release gates, semantic search, new controls, replies, merged tabs and configuration versioning are absent as test obligations. Existing instrumentation remains unchanged.

## Risk And Approval Review

- Risk level: Medium. Matching precision, visibility eligibility and mixed-tier pagination are the main risks. No higher-risk product or production scope was introduced by test design.
- Review requirement: The recommended document-review stage has been completed. The user requested this stage; the next stage still requires their selection under the workflow exit gate.
- Approval notes: Product decisions were confirmed through grilling and later explicit simplification. This review accepts that revised scope and does not revive retired performance requirements. Numerical confidence, weights, candidate bounds and local projection strategy are engineering decisions for the coding plan.
- Remaining risks: GAP-001/GAP-002 require engineering resolution; GAP-003 requires migrated-schema lifecycle evidence; GAP-004 concerns fixture realism; GAP-005 distinguishes designed tests from runtime evidence. These are not blocking product questions.
- Production controls: No production migration, deployment or infrastructure action is approved. Any later production operation remains subject to repository change controls.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Inspect the existing search/store/cursor/indexing seams and design the smallest reliable matching increment. Start with IT-001: a declared-English record using `socks` is retrievable with `sock` in each tab. Apply DR-001 when contrasting non-English/unknown metadata so the test distinguishes language normalization from typo retrieval.
- Blocking issues: None for coding planning. Resolve the proposed correction policy and exact migration/backfill harness in the plan before implementing their tests; apply the other findings when turning the specification into concrete cases.
- Test order: The suggested sequence in `02-acceptance-tests.md` section 11 is coherent. Visibility constraints must be present in every new retrieval increment rather than added after matching is complete. Complete ranking and cursor work together before claiming mixed-tier pagination coverage.
- Validation performed: Read both documents together; checked active requirement/criterion coverage and defined ID references; reviewed scope, edge cases, fixture expectations, automation levels and stage handoff. No runtime or performance tests were performed.

## Notes For Next Stage

- Produce `04-coding-plan.md` using `write-coding-plan` only after the user selects that stage.
- Carry forward DR-001 through DR-004 and existing GAP-001 through GAP-005. Resolve engineering choices explicitly without asking the user to reapprove settled product intent.
- Keep matching in AppView's PostgreSQL projection. Choose an index/extension strategy and explain old-row convergence; no external search/model service, lexicon or published-record mutation is needed.
- Read the architecture reference and API contract before architectural or route decisions. Preserve existing diagnostic privacy rules and user-visible query wording.
- Prefer observable retrieval/order and serialized-sink assertions. Pure matching logic can use unit tests; SQL-owned behavior needs real PostgreSQL evidence rather than duplicating SQL in a test-only algorithm.
- Keep candidate limits deterministic and visibility-aware. Retain complete duplicate-free pagination only for unchanged data/rules; do not introduce configuration-version continuity as a requirement.
- Carry forward the relevance corpus and corrected-result exception. Do not introduce performance measurement or release gates under the heading of benchmark validation.
- Plan future commands using the repository test harness; PostgreSQL skips are missing evidence. No source changes, executable tests, migrations, dependencies or commits were created in this review stage.
