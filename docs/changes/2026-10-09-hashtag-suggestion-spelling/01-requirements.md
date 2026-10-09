# Requirements: Hashtag Suggestion Spelling

## 1. Initial Request

Hashtag autocomplete and suggestions should display a representative original spelling while searching, ranking, and counting tags case-insensitively. For four posts using `#memademay` and five using `#MeMadeMay`, show one suggestion: `#MeMadeMay · 9 posts`.

The user selected **most-used spelling overall**, rather than preferentially selecting mixed-case spellings. The user rejected adding a time-window explanation to the displayed count.

## 2. Current Codebase Findings

- `appview/internal/postutil/tags.go`: tag extraction and merging lowercase and deduplicate tags; the indexed tag list loses original casing.
- `appview/internal/index/craftsky_post.go`: indexes tags from post-text facets and structured/faceted project metadata through those helpers.
- `appview/internal/api/facet_store.go`: composer hashtag suggestions group tags case-insensitively, count distinct posts from the last 28 days, and rank exact matches before prefix matches before other substring matches, then by count and tag.
- `appview/internal/api/search_hashtag_store.go`: hashtag search and top hashtags also return lowercase tags with distinct-post counts over the last 28 days. Top hashtags are grouped by craft. Existing eligibility and visibility predicates differ between surfaces.
- `appview/internal/api/search_ranking.go` and `search_request.go`: queries, ranking helpers, hashtag paths, and recent-search hashtag payloads are normalised to lowercase.
- `app/lib/shared/rich_text/widgets/facet_autocomplete_editor.dart`: displays the returned spelling and inserts it into the composer when selected; counts use the existing `N posts` label.
- `app/lib/search/pages/`: search suggestions, hashtag results, and top-hashtag chips display the returned tag. The tag page heading uses the tag supplied on navigation.
- Constraints: preserve existing endpoint-specific eligibility, permissions, normalisation, pagination, and ranking. Public source records belong to the PDS; presentation changes must not rewrite them.
- Test commands discovered for later stages: `just test` (host Go suite against the development stack), `just appview-test-unit` (incomplete unit-only path), `just app-test` / `cd app && flutter test`. No tests are created or run in this requirements stage.

## 3. Clarifying Questions And Decisions

### Q1: How should the display spelling be selected?
Answer: The user chose most-used spelling overall.
Decision / implication: The exact spelling used by the most distinct eligible posts wins, including lowercase or all-caps. There is no CamelCase preference.

### Q2: Should the count label explain the existing time window?
Answer: The user disagreed with adding that explanation.
Decision / implication: Keep the existing count label. Preserving the current 28-day calculation is an assumption; changing it to all-time is outside this update.

No blocking questions identified. Remaining defaults are recorded in section 20.

## 4. Candidate Approaches

### Option A: Most-used spelling overall — selected
Summary: Group case variants together; display the exact observed spelling with the highest distinct-post usage.
Pros: Simple, reflects community usage, and uses real spellings.
Cons: Does not guarantee CamelCase or improved readability.
Risks: Ties and changes in usage can change the displayed spelling.

### Option B: Most-used mixed-case spelling — rejected
Summary: Prefer an observed mixed-case spelling, falling back to lowercase.
Pros: More likely to expose word boundaries.
Cons: Can favour a rare spelling over the community majority.
Risks: Unusual capitalisation can win; identifying readable casing adds policy.

### Option C: Curated spellings or inferred capitalisation — not selected
Summary: Maintain preferred spellings or infer word boundaries.
Pros: Can provide consistent presentation for known tags.
Cons: Requires curation or unreliable inference.
Risks: Incorrect segmentation and overrides of community usage.

## 5. Recommended Direction

Use the user-selected most-used spelling overall. Keep a case-insensitive identity for matching, ranking, navigation, and aggregate counts, while retaining observed spellings for display selection. Use the same eligible post set for both the aggregate count and spelling frequencies within each surface.

## 6. Problem / Opportunity

Always returning lowercase removes author-supplied word boundaries and branding. Returning the most-used original spelling represents actual community usage without fragmenting a hashtag into separate case-sensitive results.

## 7. Goals

- G-001: Display the most-used observed spelling for each suggested hashtag.
- G-002: Preserve case-insensitive search, ranking, and aggregate counts.
- G-003: Apply the policy consistently to composer suggestions, search suggestions/results, and top hashtags.

## 8. Non-Goals

- NG-001: Prefer CamelCase, infer word boundaries, or curate preferred spellings.
- NG-002: Change the count window, count-label wording, search ranking, or content eligibility rules.
- NG-003: Rewrite existing post text, PDS records, or lexicons; change structured project-tag syntax.
- NG-004: Resolve a globally preferred spelling for every inline hashtag or direct tag-page link. Existing post text retains its author-written spelling.
- NG-005: Implement code, tests, migrations, infrastructure changes, or commits in this stage.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Author | Types and selects hashtags in the composer | Suggestions that insert the representative spelling |
| Reader | Searches or browses hashtags | One result per case-insensitive tag with a combined count |
| AppView | Indexes public records and supplies suggestions | Original spellings plus a stable case-insensitive identity |

## 10. Current Behavior

The tag index, suggestion responses, and search results normalise tag spellings to lowercase. Case-insensitive matching and distinct-post aggregation already exist. Original casing cannot be recovered from the normalised tag list alone.

## 11. Desired Behavior

With four eligible distinct posts containing `#memademay` and five containing `#MeMadeMay`, return one suggestion displaying `#MeMadeMay` with nine posts. Reverse those frequencies and display `#memademay` with nine posts. Query casing does not affect membership, ordering, or counts. Selecting the suggestion inserts its displayed spelling; manually authored text is preserved.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Present one representative spelling and combined post count per case-insensitive hashtag identity. | Avoid fragmented results while reflecting usage. | Initial request | AC-001, AC-002 |
| FR-001 | Functional | Must | Match and rank tags using the existing case-insensitive identity; query casing shall not alter result membership, order, or counts for the same data and viewer context. | Preserve search behaviour. | Initial request, codebase | AC-003 |
| FR-002 | Functional | Must | Return the winning observed spelling on composer hashtag suggestions, search suggestions, hashtag search results, and top-hashtag items. | Consistent policy across discovery surfaces. | Discovery, ASM-004 | AC-001, AC-002, AC-004 |
| FR-003 | Functional | Must | Selecting a composer suggestion shall insert the returned spelling; navigating from a suggestion/result shall retrieve posts for the case-insensitive identity. | Display casing must remain usable. | Discovery, current UI | AC-005 |
| FR-004 | Functional | Must | Retain observed original tag spellings for indexed posts and update their contribution on post edits, deletes, and reprocessing. | Spelling selection needs accurate source data. | Codebase | AC-006 |
| FR-005 | Functional | Must | Existing eligible indexed posts shall participate in spelling selection once the update is available, using verified public source spellings. Recovery shall preserve public source records and shall not fabricate casing from normalised tag arrays. | Avoid a winner based only on new posts. | Initial scenario, codebase | AC-007 |
| RULE-001 | Business rule | Must | Select the exact observed spelling used by the greatest number of distinct eligible posts, without preferring any casing style. | Implements the selected approach. | User answer | AC-001, AC-002 |
| RULE-002 | Business rule | Must | Count each eligible post once in the aggregate tag count and at most once per exact spelling, regardless of repeated occurrences or metadata locations. A post containing two case variants may contribute once to each variant but only once to the aggregate. | Avoid inflated counts and ambiguous frequency calculations. | Initial request, ASM-002 | AC-008 |
| RULE-003 | Business rule | Must | Resolve equal spelling frequencies deterministically without depending on query casing or row order. Proposed default: ascending case-sensitive Unicode code-point order. | Stable results under ties. | ASM-003 | AC-009 |
| RULE-004 | Business rule | Must | Preserve each surface's existing post eligibility, visibility, craft grouping, 28-day window, and count-label wording. Use its eligible post set for both spelling frequencies and combined count. | Keep this change scoped and prevent excluded content influencing display. | User answer, codebase, ASM-001 | AC-010 |
| NFR-001 | Non-functional | Must | Reprocessing the same source state shall produce the same spelling frequencies and aggregate counts without duplicate contributions. | Preserve idempotent ingestion. | Repository guidance | AC-006 |
| NFR-002 | Non-functional | Must | Preserve existing request limits, cursor behaviour, and ranking tie-breaks based on normalised tag identity rather than display casing. | Avoid presentation changes affecting pagination/order. | Codebase | AC-003, AC-011 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-002, RULE-001 | Given four eligible distinct posts with `memademay` and five different eligible posts with `MeMadeMay`, each applicable surface returns one `MeMadeMay` item with count nine. |
| AC-002 | BR-001, FR-002, RULE-001 | Given five eligible posts with `memademay` and four with `MeMadeMay`, the item displays `memademay` with count nine. Given an all-caps spelling with the highest frequency, that spelling wins. |
| AC-003 | FR-001, NFR-002 | Given unchanged data/viewer/time, queries differing only in casing produce equivalent tag identities, order, counts, and winning spellings; existing exact/prefix/substring precedence is preserved. |
| AC-004 | FR-002 | Given the same eligible post set on different discovery surfaces, all select the same spelling. Craft-specific top hashtags use the eligible set for that craft. |
| AC-005 | FR-003 | Given a `MeMadeMay` suggestion, selecting it inserts `#MeMadeMay` with existing spacing behaviour. Opening it retrieves the same post identities as opening `memademay` with otherwise identical filters; manually entered post text is not recased. |
| AC-006 | FR-004, NFR-001 | Given indexed casing variants, edits replace the old post's contributions, deletes remove them, and repeated processing of unchanged records does not increase counts or change the winner. |
| AC-007 | FR-005 | Given eligible posts indexed before the update, recovery restores their observed spellings and produces the same winner and aggregate count as fresh indexing of the same public source state, without author edits or source-record changes. Repeating recovery does not duplicate contributions. |
| AC-008 | RULE-002 | Given one post containing repeated `MeMadeMay` and `memademay` across eligible tag sources, its aggregate contribution is one and its contribution to each exact spelling is one. |
| AC-009 | RULE-003 | Given tied spelling frequencies, repeated requests and different input row orders select the same winner using the agreed tie-break rule. Under the proposed default, `MeMadeMay` sorts before `memademay`. |
| AC-010 | RULE-004 | Given posts excluded by a surface's existing time, content, visibility, or craft rules, those posts affect neither its winner nor its count. Existing count labels receive no new time-window explanation. |
| AC-011 | NFR-002 | Given paginated hashtag results, changing only source casing without changing tag identities or aggregate counts does not alter normalised result order, duplicate identities across pages, or invalidate a cursor solely because display spelling changed. |

### Lifecycle example for AC-006

At a fixed time and with unchanged eligibility, start with four lowercase and five mixed-case posts. Changing one mixed-case post to lowercase produces frequencies 5–4, aggregate nine, and winner `memademay`. Replay leaves those values unchanged. Removing that changed post's tag produces frequencies 4–4 and aggregate eight; under ASM-003, `MeMadeMay` wins the tie. Deleting the already untagged post has no further effect. Deleting another mixed-case contributing post produces frequencies 4–3, aggregate seven, and winner `memademay`. Repeating that delete leaves the result unchanged.

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Only lowercase spelling observed | Display lowercase. | RULE-001 |
| EC-002 | Several mixed-case or all-caps spellings | Highest distinct-post frequency wins; stable tie-break on equality. | RULE-001, RULE-003 |
| EC-003 | Same post contains several spellings | Aggregate once; each exact spelling once. | RULE-002 |
| EC-004 | Winning posts age out, are edited, deleted, or become ineligible | Recalculate from remaining eligible contributions; no permanently pinned spelling. | FR-004, RULE-004 |
| EC-005 | No eligible posts / empty query | Preserve existing empty-result behaviour. | FR-001, RULE-004 |
| EC-006 | Non-ASCII tags or languages without case | Preserve the existing accepted tag syntax and identity normalisation; retain exact observed spelling. No new transliteration or word segmentation. | FR-001, RULE-001 |
| EC-007 | Structured project tags are lowercase by design | They remain valid observed spellings and participate under the same counting rules; their schema is unchanged. | RULE-001, RULE-002 |

## 15. Data / Persistence Impact

- New fields: original spelling information is needed in the AppView projection; storage shape is deferred to implementation planning.
- Changed fields: retain normalised search identity alongside display candidates. Do not infer casing from lowercased tag arrays.
- Recovery required: existing eligible spelling contributions must be recovered. Coding planning must inventory verified source fields for post-text facets, structured project tags, pattern facets, and material facets, and identify any other currently indexed tag source. Normalised tag arrays alone are insufficient evidence of original spelling.
- Completion gate: specify the recovery mechanism, its integration-test target, eligible-source coverage, and how completion is verified. Compare recovered results with fresh indexing of the same source state. If any required source spelling is unavailable, report the affected scope and seek a requirement revision rather than guessing casing or silently covering only new posts.
- Migration required: depends on the storage design. If a migration is selected, verify the actual migration against pre-change data; updated test fixture DDL alone does not establish migration correctness. No migration or recovery operation is authorised by this document.
- Backwards compatibility: shipped-client compatibility is not required by repository policy, but production data and infrastructure controls apply. No PDS record or lexicon change is intended.

## 16. UI / API / CLI Impact

- UI: use the selected spelling in existing discovery rows/chips and composer insertion. Keep current count wording. Headings reached from those selections inherit the returned spelling; no global heading lookup is required.
- API: hashtag responses must convey the winning spelling while retaining case-insensitive lookup semantics. Exact field design is deferred; coding planning must identify response fields carrying display spelling and normalised identity, plus their API and client test targets. Trace spelling through existing response builders, ranking helpers, client mapping, insertion, and navigation to prevent accidental lowercasing. Existing camelCase JSON, error, limit, and cursor contracts apply.
- CLI: None identified beyond a possible projection-recovery operation to be designed later.
- Background jobs: ingestion/reprocessing must maintain spelling contributions; no additional recurring job is required by these requirements.

## 17. Security / Privacy / Permissions

- Authentication and authorization: preserve existing endpoint policies and surface-specific visibility rules.
- Sensitive data: candidates come only from public hashtag sources already eligible for that surface; drafts and other private content do not contribute.
- Abuse cases: repeated occurrences within a post do not increase its weight. Many distinct posts can influence the winning spelling; new anti-spam or editorial policy is outside scope.
- No source record rewrites or new access to PDS credentials is needed.

## 18. Observability

New events, logs, metrics, and alerts: None identified. Existing diagnostic conventions remain mandatory if implementation changes diagnostics. Recovery completion and correctness need verification during rollout, without logging private payloads.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Existing normalised arrays have lost casing | Historical winner is wrong or recovery is incomplete. | Verify recoverable public source data and plan recovery before enabling the feature. |
| RISK-002 | Additional variant aggregation increases query cost | Autocomplete latency rises. | Assess representative query/index plans during coding planning; preserve bounded result limits. No numeric latency threshold has been agreed, so do not claim a performance pass or invent one. |
| RISK-003 | Counts overlap when one post uses several variants | Summing variant counts inflates the total. | Aggregate distinct post identities separately from per-spelling frequencies. |
| RISK-004 | Popularity changes the displayed spelling | Labels may change over time and may remain lowercase. | Accept this consequence of the selected policy; use a deterministic tie-break and stable normalised identities. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Retain the current 28-day window for both spelling selection and aggregate counts; rejecting an explanatory label does not request all-time counts. | A different window changes aggregation and recovery scope. |
| ASM-002 | Usage means distinct eligible posts, with each post allowed to contribute once to each exact spelling it contains. | A one-vote-per-post policy would need a rule for choosing that post's spelling. |
| ASM-003 | Ties use ascending case-sensitive Unicode code-point order, without a casing preference. | Substitute the agreed deterministic tie-break before test design is finalised. |
| ASM-004 | Scope includes composer suggestions, search suggestions/results, and top hashtags; each retains its own eligible post set. | Narrower scope reduces affected surfaces; a global winner would require separate population rules. |

## 21. Open Questions

Blocking: None identified.

Non-blocking: Confirm or revise ASM-001 through ASM-004 during review; they are explicit defaults rather than additional confirmed product decisions.

## 22. Review Status

Status: Draft

Risk level: Medium, due to casing recovery and aggregation correctness.

Review recommended: Yes; user may skip review.

Reviewer: Codex document review recorded in `03-document-review.md`; subsequent revisions await review.

Date: 2026-10-09

Notes: Revised following DR-001–DR-005 in `03-document-review.md`. Added recovery and migration verification gates, explicit lifecycle outcomes, API-seam handoff, and query-cost follow-up. Most-used spelling and unchanged count-label wording are confirmed; ASM-001–ASM-004 remain defaults rather than additional confirmed decisions. No implementation, migrations, or commits are included.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`.
- Next test specification: `02-acceptance-tests.md`.
- Must-cover requirement IDs: BR-001; FR-001–FR-005; RULE-001–RULE-004; NFR-001–NFR-002.
- Suggested test levels: unit coverage for winner/tie/deduplication rules; PostgreSQL integration coverage for visibility, lifecycle, recovery, counts, and pagination; API/Flutter coverage for spelling preservation, selection, navigation, and unchanged labels.
- Blocking product questions: None. Carry ASM-001–ASM-004 into later planning unless revised. Before implementation/rollout, resolve source recovery feasibility, API/storage seams, and any migration-backed verification; do not silently weaken a Must requirement.
