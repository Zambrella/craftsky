# Implementation Review: Instagram Historical Post Importer

## 1. Verdict

**Approved with external release gates.**

The credential-independent implementation matches the approved requirements
and architecture. No blocking correctness, privacy, security, recovery, or
cross-surface regression finding remains in the reviewed diff. The importer is
not ready for production publication until the live OAuth/PDS, current-export,
browser/device, firehose, and Cloudflare checks in section 5 are completed.

This review does not authorize staging, committing, pushing, opening a pull
request, or deploying.

## 2. Reviewed Scope

- `instagram-importer/`: standalone Vite/React/TypeScript static importer
- `lexicon/social/craftsky/feed/post.json` and generated Go/web contracts
- ADR 008 for the public self-asserted provenance field
- AppView migration, indexing, profile/search chronology, timeline and
  notification policy, post responses, and pagination compatibility
- Flutter post models, localized copy, and subtle full/quote provenance label
- workflow documents `01-requirements.md` through
  `05-implementation-plan.md`

The existing Instagram DM verification and follower-import work is outside
this branch and was not changed.

## 3. Conformance Summary

| Area | Review result |
|---|---|
| Local archive boundary | ZIP/ZIP64 directory inspection is Blob-backed; parsing, normalization, media preflight, preview, and publication sanitization run in a dedicated worker. The complete archive is neither uploaded nor persisted. |
| Export parsing | Only versioned Posts metadata/media paths and supported legacy/detailed shapes are accepted. Ambiguous, encrypted, malformed, and unsupported-compression archives fail locally; unrelated content does not cross the minimized worker result. |
| Review and publication | Selection defaults, transformations, warnings, editable captions, explicit text-only confirmation, final DID/count confirmation, and one-at-a-time create-only publication are implemented. |
| OAuth and membership | OAuth begins after local review, requests the exact narrow create/delete/blob scope, rejects authority drift, and checks the authenticated PDS for `social.craftsky.actor.profile/self`. |
| Privacy and persistence | IndexedDB contains bounded DID/session/item/rkey/CID/status metadata only. Captions, filenames, media, archive bytes, OAuth material, and diagnostic identifiers are excluded. |
| Recovery and rollback | Deterministic per-source TIDs, exact-existing reconciliation, durable returned-CID ownership, pause/resume/retry, and CID-preconditioned rollback fail closed around ambiguous or changed records. |
| Public provenance | The Lexicon adds only optional `externalImport.source`; AppView recognizes exact `instagram`, clears classification on authoritative replacement omission, and returns it for full and quote post shapes. |
| Distribution behavior | Imported originals use source chronology on profiles/search, are excluded from authored home-timeline arms, and do not activate source-record notifications. Later engagement and intentional repost/quote behavior remain ordinary. |
| Flutter presentation | Full post cards and quote previews show one subtle localized label for exact Instagram provenance. Ordinary and unknown-source posts remain unchanged. |
| Runtime isolation | Production, stable staging, local loopback, and write-disabled preview modes are origin-bound. Assets are self-hosted; emitted policy files contain the intended CSP, COOP, privacy, and no-index controls. |

## 4. Findings Resolved During Review

| Finding | Resolution |
|---|---|
| IR-001: an exact existing record could require unnecessary media work during owned-session recovery | The returned current CID now provides an owned fast path; a mismatch remains a collision. |
| IR-002: publication and preview transforms needed active cancellation rather than stale-result fencing alone | Abort signals now reach the worker sanitize request; cancellation retains the selected archive and pauses safely. |
| IR-003: rollback absence was indistinguishable from successful deletion | Durable/UI states and aggregate results now distinguish removed, already absent, changed/conflict, and failed. |
| IR-004: rank-based same-time rkeys changed when a later export added an earlier-sorted source | Each canonical 128-bit item identity now selects a stable TID slot within its source UTC day, independent of archive membership/order; true slot collisions fail closed. |
| IR-005: encrypted or unsupported-codec entries could be discovered only when extracted | Every enumerated entry is checked before allow-list filtering; the whole archive fails for encryption or methods outside Store/Deflate/Deflate64. |
| IR-006: Unicode-equivalent media paths were normalized for identity but not equivalence comparison | Equivalent NFC identities now merge consistently rather than becoming a false material conflict. |
| IR-007: configured archive bounds lost their specific safe UI code at the worker boundary | The worker preserves only the three bounded archive-limit codes and still collapses unknown details to a generic safe code. |
| IR-008: later-engagement notification subjects could lose imported provenance | Notification hydration now scans and returns the subject post provenance and has a real-Postgres regression. |
| IR-009: changing profile chronology risked changing the opaque cursor wire key | Posts, projects, and comments retain the legacy `indexedAt` cursor key while carrying `profile_sort_at`; legacy and new cursors have real-Postgres coverage. |
| IR-010: virtualization and OAuth popup behavior needed compatible static headers | CSP permits only inline style attributes needed by virtual row positioning; COOP uses `same-origin-allow-popups` while scripts, framing, objects, referrers, and sensitive capabilities remain restricted. |
| IR-011: some test names could be read as live full-system evidence | Browser tests and implementation evidence now identify their actual mocked, worker-smoke, or UI-slice layer. |
| IR-012: record CIDs cannot distinguish byte-identical record incarnations | The rollback contract and confirmation now state that `swapRecord` protects edited/replaced and CID-different recreations, while a byte-identical delete-and-recreate at the same rkey remains indistinguishable. |
| IR-013: preflight sliced the first four source images before knowing which were usable | Inspection now continues in source order until four valid sanitized images are retained, with an exact regression for an invalid early image and valid fifth image. |
| IR-014: declared ZIP sizes alone could under-bound extraction | Custom bounded writers reject actual metadata or media output beyond the declared/configured envelope, including under-reported hostile fixtures. |
| IR-015: detailed skips could lose available original dates | Publication state fails closed, and content-free adapter/ambiguity skips carry every available normalized root, Creation-time, or media timestamp into review. |
| IR-016: the worker manifest could retain an unbounded raw caption alongside the final caption | Review keeps only bounded initial/final prepared captions; a tail-canary regression proves the raw over-limit source does not cross the worker boundary. |
| IR-017: failed sign-out, completion, rollback, or clear-history actions could move state or show misleading no-transfer copy | Destination switching now waits for successful library sign-out, recovery state remains on failure, and each post-authorization action has accurate safe guidance. |
| IR-018: two concurrent Playwright files caused shared dev-server page reload contention | The checked Playwright configuration uses one worker; the unchanged standard command then passed all real-Chrome flows. |
| IR-019: a current export's equivalent legacy and detailed post representations appeared to conflict | Empty root captions no longer mask media captions, and media-attached timestamps take precedence over unrelated labelled/root metadata timestamps. A synthetic dual-representation regression mirrors the observed structure, and a content-free structural check matched all 72 post pairs with zero conflicts. |
| IR-020: Vite could discover the archive-worker graph after the first browser flow became interactive and reload the page | The checked dev-server configuration prewarms the main app and archive-worker module graph; the standard one-worker real-Chrome command then passed all flows without losing selected-file state. |
| IR-021: 47 posts used current media path forms outside the original flat Posts allow-list, yielding 94 duplicate adapter skips across the two JSON representations | The narrow allow-list now includes only six-digit date-partitioned Posts paths and flat Other paths in addition to flat Posts paths. Arbitrary nesting and unrelated directories remain rejected. Content-free validation recognizes all 72 records and confirms every metadata reference exists in the ZIP. A direct local Chrome review of the supplied 1.7 GB export produced 71 selected posts, 197 selected images, one video-only skip, and no unsupported-shape entries. |
| IR-022: entering the Connect step from a `localhost` development URL failed OAuth initialization with a generic local error | AT Protocol rejects hostname-based loopback callbacks under RFC 8252. The bootstrap now redirects `localhost` to the equivalent `127.0.0.1` URL before archive or OAuth state is created, and runtime metadata uses the canonical IP-literal callback. Unit and real-Chrome regressions cover URL preservation and successful bootstrap on the canonical origin. |

## 5. Verification

| Surface | Final evidence |
|---|---|
| Importer static analysis | `npm run lint`, `npm run typecheck`, `npm run generate:check`, and `npm ls --depth=0` passed. |
| Importer tests | `npm test -- --maxWorkers=4` passed: 31 files, 150 tests. |
| Real browser | `PLAYWRIGHT_CHANNEL=chrome npm run test:e2e` passed: 6/6 scenarios using the checked one-worker configuration. |
| Build modes | Preview and production builds passed. Preview emitted `{preview: true, oauthEnabled: false}`; production emitted the canonical origin and exact narrow scope. |
| Production artifact | No source maps, test harness, archive, consented-export identifiers, analytics, remote fonts, or private canaries were found. Main JS is 1,669.30 kB minified/381.46 kB gzip; the worker is 215.42 kB. |
| Lexicon | A fresh `just lexgen` left the generated binary diff hash unchanged at `1b1ef55814bf071ac9f47dc6f8821e61385af79b711d4cfa307bc56c460eae0d`. |
| AppView | `go vet ./...` and canonical `just test` passed; the latter ran the race-enabled full suite against Compose Postgres. The 000032 up/down/up migration also passed uncached. |
| Flutter | `flutter analyze` and the 65-test focused provenance suite passed. The earlier complete 946-test Flutter suite also passed; a later second complete run was stopped after 333 passing tests due local resource contention and is not counted. |
| Repository hygiene | Go/Dart formatting, `git diff --check`, ignored generated-output checks, dependency-isolation scans, and privacy scans passed. |

The production bundle triggers Vite's advisory 500 kB chunk warning. It is a
performance follow-up, not a correctness/privacy blocker, because the worker
is already split and the current package remains a one-purpose static tool.

## 6. External And Manual Gates

The following remain **External / Pending** and must not be represented as
passing:

- `MAN-001`: live browser OAuth, exact returned authority, membership read,
  create, resume, and CID-guarded delete against supported PDS implementations
- `MAN-002`: further consented current Instagram export shapes and
  Edge/Firefox/Safari coverage. The latest provided dual representation has
  completed local Chrome review without retaining its private contents.
- `MAN-003`: representative very large/sparse ZIP64, suspension/reload,
  memory, cancellation, and real rate-limit behavior
- `MAN-004`: live PDS to relay/Tap to AppView to Flutter convergence,
  including profile chronology, timeline absence, search, labels, and push
- `MAN-005`: dedicated Cloudflare Pages project, custom subdomain, effective
  headers/CSP, network inspection, preview write-disablement, and deployment
  rollback

Automated evidence is intentionally layered. The five Playwright scenarios are
real-browser slices, not a live end-to-end OAuth/PDS/AppView suite. Direct push
delivery, every numeric safety boundary in every browser, multi-page production
network behavior, and every joined-query production plan remain evidence gaps
covered by the manual gates rather than hidden as repository passes.

## 7. Handoff

- Implementation evidence: `05-implementation-plan.md`
- Importer runbook: `../../../instagram-importer/README.md`
- Architecture decision: `../../../adr/008-instagram-import-provenance.md`
- Worktree:
  `/Users/douglastodd/Projects/craftsky/.worktrees/codex-instagram-post-importer`
- Branch: `codex/instagram-post-importer`
- No stage, commit, push, pull request, or deployment action has been performed.
