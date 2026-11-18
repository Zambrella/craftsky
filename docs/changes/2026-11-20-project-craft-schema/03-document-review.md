# Project craft schema document review

## Verdict

| Review field | Result |
|---|---|
| Status | **Approved with notes** |
| Reviewer | Codex, with independent coverage and compatibility checks |
| Date | 2026-11-20 |
| Risk level | High |

## Summary

The [requirements](01-requirements.md) reflect the agreed craft taxonomy, both release phases, legacy size compatibility and the current-only scheduling scope. The revised [acceptance test specification](02-acceptance-tests.md) links all 20 Must requirements and 18 criteria to planned tests. The recheck confirmed that the revised cases address all five original findings.

## Findings

The table records findings from the original review. Their resolutions follow.

| ID | Severity | Area | Finding | References | Required action |
|---|---|---|---|---|---|
| DR-001 | Important | Legacy size compatibility | IT-002 asks for one scalar compatibility read. A passing public post response can hide a missing scalar in the owner-only editor seed, which both legacy editors use to build their change or clear requests. | FR-010, AC-015, IT-002, MAN-001. [Public post response](../../../appview/internal/api/post_response.go) · [Owner edit seed](../../../appview/internal/api/post_edit_seed.go) · [Editable record projection](../../../appview/internal/api/post_edit_editable.go) | In IT-002, assert the singleton in both the public project response and the owner edit seed. Generate or replay representative legacy Flutter and web change and clear requests from that seed. Verify the PDS candidate stores the canonical representation and reads leave the source record intact. Pair the rejected legacy multi-size change with a successful updated-client list change or clear in REG-001. |
| DR-002 | Important | Published detail edits | IT-001 verifies writes from create. IT-003 checks URI and untouched data after an edit, and client scenarios can pass using fixture responses. None of these asserts that the actual AppView edit applied the requested nested or new-variant field. The published merge has a separate allowlist by details variant. | FR-004, FR-005, FR-006, FR-009, AC-004, AC-006, AC-007, AC-010, IT-001, IT-003. [AppView detail edit merge](../../../appview/internal/api/post_edit_merge.go) | Extend real AppView edit coverage with a representative Phase 1 nested selection and a Phase 2 detail variant. Verify the changed values in the captured PDS write and indexed read. For the nested case, clear one dependent selection and assert unrelated groups and extension fields survive. Keep the existing owner and conflict checks. |
| DR-003 | Important | Machine Knitting authoring | AT-006 starts with an already saved machine-knit cardigan. It can pass when the detail view and filter work but neither composer can set the method or save both gauge values. | FR-007, AC-008, AT-006, TD-005 | In each supported client, select Machine alone and with Hand, confirm the applicable fields appear, enter different machine spacing and measured gauge values, and save and reopen with the Cardigan subtype intact. Retain the Machine filter and missing-method controls. |
| DR-004 | Important | Existing Crochet classifications | AC-004 requires Granny square and Amigurumi to remain selectable and readable as item subtypes. AT-003 exercises the Granny square technique on a blanket, and REG-003 exercises the existing Granny square subtype in search. Neither checks both subtype picker entries and their saved display. | FR-004, AC-004, AT-003, REG-003, TD-004, Q16 | Add a small picker and readback regression for both Crochet item subtypes. Assert that Granny square remains available as a technique and that the initial Techniques picker excludes Amigurumi. Keep the blanket and keyword cases. |
| DR-005 | Important | Primary craft validation | IT-006 proves valid primary and additional craft discovery. A faulty validator could also accept recognized Spinning details on a Knitting project that lists Spinning as an additional craft. The existing ADR requires authoring to reject a recognized variant that mismatches the primary craft. | FR-006, RULE-001, AC-011, IT-006. [Project craft ADR](../../adr/020-project-evolution-search-and-multiple-crafts.md) | Pair the accepted Spinning-primary fixture with a rejected Knitting-primary, Spinning-additional command carrying Spinning details. Assert rejection before the PDS write. Retain the round-trip coverage for valid unknown extensions. |

| Finding | Verified resolution in the acceptance test specification |
|---|---|
| DR-001 | IT-002 and MAN-001 check the public response and owner edit seed. IT-002 exercises legacy patches and REG-001 checks accepted updated-client edits to the same multi-size shape. |
| DR-002 | IT-003 verifies Phase 1 grouped edits and a Phase 2 detail edit in the captured PDS candidate and indexed read, with preservation and authorization checks. |
| DR-003 | AT-006 exercises independent Hand and Machine selection, distinct gauge values, save and reopen, and continued filtering. The later coding plan review approved the multi-select and the revised scenario covers it. |
| DR-004 | AT-003 exercises picker selection and readback for Granny square and Amigurumi and checks their technique distinction. |
| DR-005 | IT-006 rejects a mismatched recognized Spinning detail variant on a Knitting project before the PDS write. |

## Traceability review

- The initial request and both review rounds are represented, including shared design tags, plural size strings, grouped techniques and styles, six new primary crafts and their member-facing entry points. The prelaunch scheduling decision applies to schedule compatibility. Existing public records and legacy local drafts remain in scope.
- All 20 Must requirements link to valid criteria. All 18 criteria link to at least one test in the 26-case matrix. Test definitions and the matrix point to each other. DR-001 through DR-005 concern what the tests actually assert.
- Data, client and API impact, protected write and visibility boundaries, existing observability, release order and non-goals have explicit sections. No product question needs reopening to resolve these review findings.

## Coverage review

- The levels are appropriate. Unit tests cover normalization, catalog rules and payload shapes. Fixture-backed Flutter and Svelte scenarios cover first-party behaviour. AppView integration targets cover PDS writes, database reads, search and scheduled publication.
- The revised scenarios cover the five findings with representative payloads.
- MAN-001 is justified for the actual legacy builds selected for rollout. MAN-002 records deployed version order. Local fixtures establish request and response behaviour. The rollout checks establish real build compatibility and AppView readiness.
- GAP-001 remains a schema-design input. The approved limits can drive the boundary tests once chosen. GAP-002 and GAP-003 accurately separate local test evidence from rollout evidence.

## Risk and approval review

- **High risk** because the change extends public records, merges edits against existing records and coordinates AppView and two clients. Mistakes in compatibility could hide or discard recorded sizes.
- The maintainer agreed the requirements scope and the scheduling exclusion, requested the test-specification revisions, and approved moving to coding planning. Implementation still requires explicit approval.
- Both phases can use the existing PostgreSQL schema and read existing public records in place. Any performance-driven projection or index should be justified separately in the coding plan.

## Coding plan readiness

- **Ready for coding planning** Yes. The revised test specification addresses DR-001 through DR-005 and the requirements provide the product rules. The remaining test gaps concern field bounds and rollout verification.
- **Recommended first implementation test after review** IT-002 remains the right starting point. Exercise a legacy singleton edit from the owner edit seed and a destructive scalar attempt against a multi-size record. Inspect the persisted public record and the applicable read responses.

## Notes for the next stage

- Keep Phase 1 independently runnable and rerun its coverage after Phase 2. Use approved aliases and craft IDs consistently across the test fixtures.
- Use current scheduling fixtures for save, reopen, publish and retry. Keep legacy fixtures for immediate publication, published edits and local draft recovery.
- The implementation plan should name the schema and catalog artifacts, existing edit and search boundaries, and the rollout order. Keep compatibility assertions focused on observable requests, responses and stored records.
