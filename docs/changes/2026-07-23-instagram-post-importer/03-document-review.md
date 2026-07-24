# Document Review: Instagram Historical Post Importer

## Verdict

Status: Approved with notes
Reviewer: Codex, with an independent acceptance-test audit
Date: 2026-07-23
Risk level: High

## Summary

The amended requirements and acceptance-test specification are consistent,
bidirectionally traceable, and ready for coding-plan work. They define a
standalone, static, local-first browser importer; narrow browser OAuth; direct
PDS publication; a minimal self-asserted post-provenance field; AppView
backfill policy; and subtle Flutter presentation without adding archive
processing to the main app.

The review corrected two technical overclaims before approval:

- Public AT Protocol records are replaceable. CraftSky-aware edit paths must
  retain the optional provenance field, but an arbitrary third-party client
  can omit it. AppView follows the authoritative record instead of inventing
  sticky provenance.
- A browser OAuth client must retain its own session and DPoP material.
  OAuth-library storage is therefore explicitly separate from content-free
  importer progress, is never copied into importer diagnostics, and is cleared
  through the library's sign-out flow.

The review also added quote-preview presentation coverage, direct acceptance
criterion references to regression and manual tests, exact test-ID lists, and
correct terminology for mention facets, reply fields, and quote embeds.

All 49 Must requirements link to acceptance criteria and tests. All 23 defined
acceptance criteria appear in the test specification. The one Should
requirement is also covered. The 60 test definitions are contiguous: ten
acceptance scenarios, nineteen unit tests, eighteen integration tests, eight
regressions, and five manual checks.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Resolved | Public record semantics | Provenance cannot be guaranteed across a replacement by an arbitrary third-party PDS client. The contract now requires CraftSky-aware edit paths to retain it and requires AppView to clear classification when the authoritative replacement omits it. | `01-requirements.md` Q12, FR-023, AC-011, Data / Persistence Impact; `02-acceptance-tests.md` AT-009, IT-010 | Implement create/update/delete index tests with retaining and omitting replacements. Do not make the AppView column sticky. |
| DR-002 | Resolved | OAuth persistence / privacy | OAuth library-managed storage is necessary and previously conflicted with the broad persistence prohibition. The boundary now forbids OAuth material only in importer-owned state and diagnostics and requires library-owned state to be removed on sign-out. | `01-requirements.md` FR-025, AC-017, Data / Persistence Impact; `02-acceptance-tests.md` UT-012, IT-005 | Keep importer progress and OAuth stores separate. Test both stores explicitly. |
| DR-003 | Resolved | Presentation | Imported post provenance can appear in a quoted-post preview, which has a separate AppView/Flutter response shape. Full-post and quote-preview tests now cover the label. | `01-requirements.md` FR-023, AC-011; `02-acceptance-tests.md` AT-009, UT-015, IT-010, IT-015, REG-007 | Propagate the optional provenance field through both response/model shapes and render the same subtle localized label. |
| DR-004 | Note | OAuth authority | Current AT Protocol permissions can express the requested least authority. The canonical request is `atproto repo:social.craftsky.feed.post?action=create&action=delete blob?accept=image/jpeg&accept=image/png&accept=image/webp`. | `01-requirements.md` FR-003, AC-003, AC-018; `02-acceptance-tests.md` AT-003, IT-006, MAN-001 | Assert the exact requested permission string and fail closed for missing or additional granted authority. |
| DR-005 | Note | Large-file wording | “Any” or “arbitrarily large” means no importer-defined whole-ZIP limit; the browser, filesystem, address space, and ZIP implementation still impose platform limits. | `01-requirements.md` FR-005, NFR-001; `02-acceptance-tests.md` AC-005, MAN-003 | State this interpretation in code comments/user copy and test no whole-file copy or configured total-size rejection rather than claiming infinite input support. |
| DR-006 | Note | Measurable non-functional gates | “Responsive enough” and “restrictive” require deterministic implementation baselines. | `01-requirements.md` NFR-003, NFR-004; `02-acceptance-tests.md` IT-004, IT-016, MAN-003, MAN-005 | The coding plan must define progress/cancel yield points and an explicit CSP/header baseline, with application-level validation for dynamic PDS origins. |
| DR-007 | Note | Export publication state | The observed detailed export shape can explicitly identify drafts or unpublished items. These are not eligible historical posts and must fail closed rather than be treated as published. | `01-requirements.md` FR-007, RULE-008; `02-acceptance-tests.md` UT-002, UT-003, IT-002 | Add synthetic draft/unpublished fixtures. Treat explicit draft or unpublished values as skipped; do not copy the consented sample into the repository. |
| DR-008 | Resolved | Rollback ownership | Rkey-only rollback could delete a post edited, replaced, or recreated after import, and a create response lost before IndexedDB persistence cannot safely establish ownership. The contract now persists the returned public record CID only for durably claimed creates and requires a CID precondition on delete. Because record CIDs address content rather than incarnations, this protects content-changed replacements but cannot distinguish a byte-identical delete-and-recreate at the same rkey. | `01-requirements.md` FR-025, FR-027, AC-014, AC-015, AC-017, RISK-012; `02-acceptance-tests.md` AT-007, UT-010, IT-005, IT-007, IT-018 | Model existing/ambiguous creates as unowned. Roll back only CID-matching owned creates, disclose the byte-identical recreation limitation at confirmation, and keep CIDs out of diagnostics. |
| DR-009 | Resolved during implementation review | Static CSP and OAuth popup | True list virtualization requires runtime element height/transform attributes, and `Cross-Origin-Opener-Policy: same-origin` can sever the cross-origin OAuth popup before its same-origin callback. | `04-coding-plan.md` Build And Deployment Security Baseline; `instagram-importer/public/_headers` | Permit style attributes only with `style-src-attr 'unsafe-inline'` while retaining self-only stylesheet/script policy, escaped text-only rendering, and no imported HTML. Use `same-origin-allow-popups`; retain frame/object prohibitions and verify emitted headers. |

## Traceability Review

- Planning to requirements: The discovery record, user decisions, architecture
  comparison, and recommended approach consistently select a standalone static
  SPA that processes the Instagram export locally and writes only confirmed
  public content directly to the authenticated PDS. It preserves the approved
  separation from the Flutter archive-follow import and avoids a new AppView
  importer API.
- Requirements to acceptance criteria: All 49 Must requirements link to at
  least one of the 23 acceptance criteria. The Should portability requirement
  is also linked. The criteria cover local privacy, archive compatibility,
  review, OAuth, membership, transformations, media safety, publication,
  deterministic resume, rollback, deployment isolation, AppView behavior,
  presentation, accessibility, and regression behavior.
- Acceptance criteria to tests: Every acceptance criterion appears in the
  coverage matrix and at least one detailed test. Regression and manual rows
  carry direct requirement and acceptance-criterion references. Test IDs are
  unique and contiguous.
- Cross-stack contracts: `IT-001` binds the Lexicon, ADR, generated Go type,
  and generated/validated web contract. `IT-010` binds Tap indexing, canonical
  post responses, quoted-post responses, and replacement semantics. `IT-015`
  binds the response shape to Flutter models and widgets.

## Coverage Review

- Must requirements covered: 49 of 49.
- Should requirements covered: 1 of 1.
- Acceptance criteria covered: 23 of 23.
- Automated test definitions: 55.
- Manual checks: 5, all supplemental rather than the sole coverage for a Must
  requirement.
- Release gaps: Four explicit gaps remain for additional live export shapes,
  real OAuth/PDS authority, browser/firehose convergence, and deployed
  Cloudflare behavior. These are external release gates, not reasons to skip
  repository implementation or automated verification.

## Risk And Approval Review

- Risk level: High. The feature processes an untrusted private archive, grants
  public-write authority, uploads public media, changes a public Lexicon,
  changes AppView distribution behavior, and retains resumable browser state.
- Review requirement: Satisfied for coding planning and implementation. The
  user explicitly asked the workflow to continue without stage-by-stage waits.
- Approval notes: Implementation may proceed. Live OAuth/PDS, current-export,
  deployed-origin, and firehose checks must remain reported as pending until
  actually performed.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first failing test: `IT-001`, proving the optional minimal
  Instagram provenance contract and required ADR before changing the Lexicon
  or generated artifacts.
- Blocking issues: None.

## Notes For Next Stage

- Use a standalone `instagram-importer/` npm package and checked lockfile, not a
  root JavaScript workspace and not Flutter.
- Keep source ZIPs Blob-backed. Enumerate ZIP metadata and extract only bounded
  supported JSON/media entries in a dedicated worker. Never retain the raw
  archive or source content in IndexedDB, logs, URLs, diagnostics, snapshots,
  or telemetry.
- Define a central, versioned safety policy and enforce every inclusive
  boundary from `NFR-002`.
- Treat explicit draft or unpublished detailed-export entries as skipped.
  Legacy data found under supported published-post paths remains eligible.
- Use exact least-authority OAuth permissions and inspect granted authority.
  Keep production, stable staging, preview, and localhost origin policies
  explicit; preview must hard-block real OAuth and PDS writes.
- Use application-level `https:` and authenticated-PDS origin validation for
  dynamic PDS requests. Combine this with a restrictive static CSP baseline
  and browser network-observation tests.
- Store only content-free, DID-bound progress in importer-owned IndexedDB.
  Leave OAuth artifacts entirely to the OAuth client and clear them via its
  sign-out API.
- Add the optional provenance object to the public post Lexicon through an ADR,
  regenerate Go/web artifacts, and keep the source self-asserted and
  non-authorizing.
- Persist a separate profile-sort timestamp in AppView. Ordinary rows retain
  indexed-time behavior; exact Instagram imports use original `createdAt`.
- Exclude only the original authored imported row from home-timeline
  selection. Do not filter a later repost/quote arm.
- Skip only notification activation caused by the original import while
  retaining ordinary index relationships and later engagement behavior.
- Propagate provenance through canonical and quoted-post response shapes and
  render one localized subtle label in the shared Flutter presentation.
