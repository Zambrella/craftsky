# Document Review: Video Posts

## Verdict

Status: Approved with notes

Reviewer: OpenCode

Date: 2026-09-03

Risk level: High

## Summary

The amended requirements and acceptance tests consistently define direct
Flutter upload to the Bluesky video service under ADR 012. OAuth access/refresh
tokens and DPoP keys remain AppView-only; Flutter receives one short-lived,
method-bound service JWT, attaches it only to the approved upload destination,
and never persists it. AppView independently verifies completion, owner DID, and
exact BlobRef before creating the post.

The former AppView upload proxy, durable video jobs, idempotency-key contract,
status proxy, migration, cleanup worker, and account-deletion cleanup are no
longer part of the design. ADR 013 separately approves the additive standard
video embed. Caption delivery and limits authorization are explicit. Coding
planning may proceed; implementation still requires renewed user approval.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | API | Resolved: authenticated preflight remains `GET /v1/blobs/videos/limits`, with normalized camelCase output and standard upstream failures. Its `app.bsky.video.getUploadLimits` authorization remains inside AppView. | `01-requirements.md` FR-028, AC-036, AC-037; `02-acceptance-tests.md` IT-013 | Keep the internal limits credential distinct from the upload credential returned to Flutter. |
| DR-002 | Suggestion | Playback | Resolved: configurable percent-encoded HLS/thumbnail templates, the pinned `media_kit` stack, and the six-platform matrix remain defined. | `01-requirements.md` FR-011, FR-015; `02-acceptance-tests.md` UT-005, UT-010, IT-008, IT-015 | Reverify live CDN, CORS, WebVTT, full-screen, and release-size behavior before release. |
| DR-003 | High | Security | Resolved by ADR 012: the client credential exception is limited to one PDS-signed service JWT for `com.atproto.repo.uploadBlob`, expires no more than 30 minutes after issuance, remains memory-only, rejects redirects, and is attached only to the exact upload origin/path. | ADR 012; `01-requirements.md` BR-003, FR-001 through FR-003, FR-029; `02-acceptance-tests.md` UT-011, UT-012, IT-002, IT-009, IT-010, IT-018 | Treat browser CORS and redirect/header-containment evidence as release gates. |
| DR-004 | High | Trust boundary | Resolved: post creation does not trust client job/blob claims. AppView re-queries public status and requires terminal success, authenticated owner DID, and exact canonical blob equality before any PDS effect. | `01-requirements.md` FR-005 through FR-007; `02-acceptance-tests.md` UT-003, UT-004, IT-003 through IT-005 | Return one generic rejection for unknown, incomplete, foreign, or mismatched proof; do not disclose foreign details. |
| DR-005 | Suggestion | Persistence | Resolved: direct upload/polling removes AppView video-job storage and automatic resume. Ambiguous/manual retry obtains new authorization and starts over; `already_exists` is accepted only with independently verified matching blob. | `01-requirements.md` FR-004, FR-027, FR-029 through FR-031; `02-acceptance-tests.md` UT-011, UT-013, IT-009, IT-012, AT-005 | Add schema-absence and persistence-spy tests; do not create a compatibility migration or idempotency key. |
| DR-006 | Suggestion | Lexicon | Resolved by ADR 013: add `app.bsky.embed.video` to the existing open union and reuse standard caption/aspect-ratio/blob definitions. Pinned Indigo has compatible Go shapes but stale 100 MB descriptive schema data. | ADR 013; `01-requirements.md` FR-008, RULE-001; `02-acceptance-tests.md` IT-006 | Add only `video.json` and `defs.json` to explicit lexgen inputs. Retain the reviewed Indigo pin unless implementation proves an upgrade necessary. |
| DR-007 | Suggestion | Captions | Resolved: captions use one authenticated post-membership-checked route that returns only a bounded valid `text/vtt` blob. There is no generic PDS/blob proxy. | `01-requirements.md` FR-018 and section 16; `02-acceptance-tests.md` IT-008, IT-017 | Verify exact indexed caption membership, MIME, size, WebVTT signature, and auth/device route policy. |
| DR-008 | Suggestion | Observability | Metrics are defined for server-observable authorization, limits, verification, and caption operations plus client-observable upload/poll/playback behavior. Labels exclude identifiers and authored content. | `01-requirements.md` NFR-003 and section 18; `02-acceptance-tests.md` IT-018, IT-020 | Do not imply AppView observes direct-client events unless a bounded client telemetry contract is implemented. |

## Traceability Review

- Planning to requirements: Direct authorization handoff, direct streaming and
  polling, independent completion verification, standard video records,
  HLS-only playback, local drafts, and excluded surfaces are consistent.
- Requirements to acceptance criteria: Every Must requirement references one or
  more acceptance criteria. New credential, verification, persistence-absence,
  limits, caption, and route-policy behavior is externally verifiable.
- Acceptance criteria to tests: AC-001 through AC-045 appear in acceptance,
  unit, integration, regression, or manual cases. Test cases retain requirement
  and acceptance-criterion references.
- Architecture: ADR 012 records the sole client-held service-token exception;
  ADR 013 records the public lexicon change. Project-wide OAuth and credential
  documentation reflects the narrower boundary.

## Coverage Review

- Must requirements have automated targets; manual checks supplement rather
  than replace them.
- Security coverage includes token response minimization, exact URL/path and
  redirect handling, memory-only lifecycle, secret scanning, owner/blob
  verification, and absence of AppView job persistence.
- Live Bluesky CORS, quota policy, CDN behavior, native backend behavior, browser
  playback, and process-level memory remain environmental/manual release gates.
- Boundary tests avoid checking in or allocating a 300 MB fixture by using
  sparse files, generated streams, byte counters, and failure injection.

## Risk And Approval Review

- Risk level: High, unchanged.
- Status: `Approved with notes` for coding-plan preparation only.
- Implementation approval: Not granted. Explicit user approval is required
  after the replacement coding plan is complete and reviewed.
- Residual risks: bearer replay until service-JWT expiry, browser upload CORS,
  live service-contract drift, pinned Indigo schema-description drift,
  cross-platform HLS/WebVTT behavior, and large-file device memory behavior.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- First test split: request-shape/mutual-exclusion checks belong in `UT-003`;
  owner/completion/blob checks belong in `UT-004` and handler integration tests.
- Prohibited implementation: AppView upload/status proxy routes, `video_jobs`,
  job cleanup workers, automatic continuation, generic service-auth minting,
  generic blob proxying, or an invented idempotency-key contract.
- Blocking issues: None.

## Notes For Next Stage

- Preserve the API architecture's `/v1/`, bearer/device auth, camelCase JSON,
  request ID, and error-envelope rules.
- Keep upload authorization, limits authorization, public status polling, and
  final completion verification as distinct capabilities.
- Keep generated Lexicon files generator-owned and inspect the complete diff.
- Keep external video transport isolated from every AppView bearer interceptor.
- Do not start implementation until the user explicitly approves the completed
  high-risk coding plan.
