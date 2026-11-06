# Implementation Review: Logged-out browsing

## Verdict

Status: **Approved with notes**

Reviewer: Codex, final correction review.

Date: 2026-11-06.

Risk level: **High**.

## Summary

All five implementation findings are resolved. The latest correction makes each accepted public profile loader result own the shared ProfileView lifetime. Fresh identity, content, unavailable embeds and continuation state now replace the previous snapshot even when DID/tab/cursor values are unchanged. Existing destruction guards dispose old work and per-component tab caches. Initial SSR content remains usable without duplicate initial reads.

The original independent reproduction now passes both handle-to-DID navigation and default-to-explicit Posts navigation: the new safe response and visible profile agree, the excluded quote is absent, and cold reload remains correct. The permanent tests additionally verify the updated body and continuation cursor. No further blocking implementation defect was identified.

This approves the reviewed implementation for code handoff with the platform/manual verification notes below. It does not establish full physical-device acceptance, Android runtime behavior, live-provider interoperability, remote CI or deployment readiness. Those evidence limits remain explicit.

Only this review artifact was updated in this stage. Source, tests, implementation notes, dependencies, migrations and generated files were unchanged. No commit, push, deployment, backfill or live account operation was performed.

## Findings

No blocking implementation findings remain. The following non-blocking verification note remains open.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-006 | Suggestion | Tests / Verification | Physical assistive-technology/share/media inspection and Android runtime execution remain unverified. Existing automated semantics, browser projects and prior iOS simulator fixtures do not establish those platform results. Optional live-PDS/Bluesky interoperation also remains unexecuted. | NFR-003, FR-003, FR-006, FR-008–009; AC-005, AC-008–010, AC-015; MAN-001–003; `05-implementation-plan.md` verification gaps. | Complete the remaining applicable device/manual checks before claiming full platform acceptance and record actual results. Keep Android unverified until the documented Device-service issue is resolved. Run optional live interoperation only with explicitly authorized disposable account access. No source correction is prescribed by this note. |

### Implementation finding disposition

| ID | Status | Correction and evidence |
|---|---|---|
| IR-001 | Resolved | Different-subject canonical profile/post navigation replaces the prior subject. Existing disposal guards and tests cover Back/Forward and delayed previous-profile pagination. Current desktop detail checks pass. |
| IR-002 | Resolved | Public loader data initializes shared profile/post rendering. Initial browser read failure retains SSR content; cursor/history state and no-duplicate-initial-read cases pass. |
| IR-003 | Resolved | Feature-scoped visibility recovery uses stable authoritative observation and an advanced immutable-dispatch head/revision boundary. It preserves unrelated fields, resolves an opposing preference as conflict, and atomically persists visibility with matching hold release. Generic addressed-command defaults remain strict. Recovery, create/deletion, invalid-evidence and race-enabled checks passed in the documented correction stage; focused recovery was independently rerun in the preceding review. |
| IR-004 | Resolved | Public serialization preserves safe `externalImport` provenance. DTO coverage and the Flutter public adapter/shared-card test protect the existing label in guest and member contexts. |
| IR-005 | Resolved | `web-client/src/routes/profiles/[handleOrDid]/+page.svelte:52` now keys the guest ProfileView by accepted loader data. This replaces the once-consumed initial snapshot and disposes its old requests/caches on each new result. Both original independent same-profile reproductions and the permanent identity/body/exclusion/cursor tests pass. |

The additional C5 progressive sign-in return-field fix remains verified: unrelated query parameters are removed while supported craft filters/sort, search and content state survive. The unrelated immediate-image-upload formatter diffs noted in the first review were removed.

## Requirement And Test Traceability

Reviewed inputs: the approved requirements, acceptance specification, DR-001–004 document-review handoff, coding plan, implementation/correction evidence in `05-implementation-plan.md`, current tracked/untracked changes and previous implementation findings. This final pass focuses on IR-005; the unchanged backend/native conclusions retain the evidence boundaries from earlier reviews.

| Requirement group | Disposition |
|---|---|
| BR-001, FR-001–003 | Guest craft feeds, submitted search and public reading are implemented. Recent users, autocomplete, top hashtags and advanced guest search controls remain excluded; craft project feed filters/sort remain. Shared public presentation includes the corrected provenance and data lifetime. |
| FR-004–007, RULE-001 | Guest modal/banner, hidden account controls, protected routes, safe auth returns and account-boundary behavior remain implemented. The latest route change is confined to the guest profile branch. |
| BR-002, FR-008–009, RULE-004 | Shared Bluesky preference, durable owner recovery and revision-fenced projection retain their reviewed behavior. No-backfill remains explicit: unknown legacy rows stay hidden until an ordinary authoritative event or explicit owner operation establishes visibility. |
| FR-010–011, RULE-002–003, NFR-004 | Server eligibility, explicit public DTOs and metadata remain shared. Fresh accepted profile responses now update visible content, including excluded nested presentation. ASM-009 still permits ordinary ingestion convergence and does not promise push-driven live erasure. |
| NFR-001–002 | Shared components, explicit guest contracts and bounded anonymous requests remain. Keying by loader result preserves existing component disposal instead of adding another fetch, polling mechanism or parallel UI. |
| NFR-003 | Automated semantics/responsive coverage is present; physical/native and Android acceptance remains qualified by IR-006. AC-015 is not claimed fully verified on every platform. |

IR-005 is linked to IT-010, AT-004/009–010 and REG-003–004, with FR-003, FR-010–011, NFR-001/004 and RULE-002. Its tests first verify the safe fresh payload, then the visible identity/body/quote state and the new pagination cursor. The synthetic fixture is isolated to its own profile and reset explicitly; it introduces no application API or data migration.

No new custom Lexicon, OAuth scope, client-held PDS credential, public endpoint, migration, background refresh, historical replay, startup population or guest-read repair was introduced by the latest correction. No unplanned product behavior was identified.

## Test Evidence

### Independently executed in this review

| Check | Result | Boundary |
|---|---|---|
| Original isolated reproduction: `node /tmp/guest-rereview-repro.mjs` | Both same-profile transitions and the cold-reload control passed | Fresh safe data produced updated headings/body and zero excluded quote matches. Local Chromium/mock/preview only; the harness closed its processes. |
| `npx playwright test guest-details.spec.ts --config tests/playwright-guest.config.ts --project chromium-desktop` | **9 passed**, 5.8s | Includes IR-005 plus original subject navigation, SSR failure retention, cursor/history and delayed old-response cases. |
| `npm run test:e2e` | **10 passed**, 5.1s | Public no-JS readability, metadata/canonical canaries, progressive links/form and accessibility regressions. |
| Scope/hash and diff checks | Only this artifact changed; `git diff --check` clean | Source/test/implementation-note hashes were captured before review. |

The original reproduction observed **Updated Alice** in the document title and visible heading for both handle-to-DID and default-to-explicit Posts navigation. Each fresh response omitted the hidden-quote canary, and the rendered page also contained zero matches. Cold reload retained the correct safe state. This directly closes the previous response/UI mismatch.

### Implementation-stage evidence reviewed, not rerun wholesale

| Gate | Recorded result |
|---|---|
| Latest full guest matrix | **84 passed per runtime**, preview 35.3s / Worker 36.7s; desktop Chromium, phone Chromium, phone WebKit and tablet Chromium |
| Latest public no-JS | **10 passed per runtime**, preview 4.9s / Worker 5.3s |
| Latest member social/detail/profile | **43 passed per runtime**, preview 20.4s / Worker 22.8s |
| Web static analysis / units / build | Zero errors/warnings; **374 tests in 58 files** passed; build passed; private source maps prepared without upload |
| Earlier affected Go suites / recovery race checks | Passed as recorded in `05`; unchanged in the IR-005 correction |
| Earlier Flutter card/guest-transport/Privacy / analysis | **115 passed**, no analysis issues; unchanged in the IR-005 correction |

The TDD notes identify meaningful red failures before the source change, focused green results, broader gates and corrected fixture/readiness issues. Initial failures are not counted as passing evidence. Browser runners were serial to preserve local Miniflare isolation. No relevant failing automated test remains unexplained.

### Remaining evidence limits

- MAN-001: physical VoiceOver/TalkBack and human screen-reader/enlarged-text validation were not performed.
- MAN-002: physical native gallery/video, clipboard/share cancellation, keyboard and safe-area inspection remain unverified beyond existing automated/simulator coverage.
- Android: no device/emulator pass is claimed. The earlier Device service reported “Android emulator video needs working ffmpeg with the libx264 encoder”; this review did not alter the host runtime or bypass the service.
- MAN-003: optional real-PDS/Bluesky interoperability was not run. Local protocol/database fixtures do not prove live OAuth/provider behavior or cached preview eviction.
- No remote CI, production deployment, physical release build, external crawler refresh or third-party cache revocation is established here.

## Risk Review

- **Risk level:** High, retained because anonymous presentation and a shared profile preference are involved.
- **Implementation assessment:** No blocking defect remains in the reviewed scope. The final change applies accepted safe loader data while retaining the existing initial-state and request-disposal mechanisms.
- **Controls retained:** Explicit anonymous routes/DTOs, credential-free public reads, independent member authorization, conservative moderation/lifecycle/image eligibility, durable matching holds, authoritative revision evidence, no-store responses and no-backfill defaults.
- **Approval boundary:** Approved for implementation/code handoff with IR-006 and the stated evidence limits. This is not deployment authorization, full device certification, or permission to mutate a live account. No commits or publication were requested.

## UI Polish Recommendation

- **Recommendation:** Optional.
- **Reason:** No cosmetic blocker was confirmed. The remaining large-text/safe-area and assistive-technology inspection may identify small visual refinements, but no speculative styling change is required by this review.
- **Suggested polish notes:** Inspect modal/banner spacing and long Privacy copy during the remaining device/manual pass; preserve existing behavior and reduced-motion semantics.

## Handoff Back To TDD Builder

- **Required fixes:** None.
- **Suggested next failing test:** None required by this review. Add targeted tests only if later device/manual evidence identifies a concrete defect.
- **Verification follow-up:** Track IR-006 and the documented platform/live boundaries. Rerun affected automated gates when source changes justify it; do not relabel historical runs as new execution.
- **Next stage:** Optional UI polish, a manual review comment, or stop. An additional required-change TDD stage is unnecessary unless new evidence is supplied.
