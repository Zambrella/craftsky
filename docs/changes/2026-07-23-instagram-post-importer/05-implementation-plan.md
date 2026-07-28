# TDD Implementation Record: Instagram Historical Post Importer

## 1. Inputs And Scope

- Requirements: `01-requirements.md`
- Acceptance tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes
- Coding plan: `04-coding-plan.md`
- Implementation approval: the user explicitly requested implementation on
  2026-07-23 and waived intermediate workflow review waits.
- Worktree:
  `/Users/douglastodd/Projects/craftsky/.worktrees/codex-instagram-post-importer`
- Branch: `codex/instagram-post-importer`
- Git/deployment authority: implementation and verification only. No staging,
  commit, push, pull request, or deployment was requested or performed.

The implemented scope is the standalone historical-post importer plus the
minimum Lexicon, AppView, and Flutter work needed to make imported posts
visible on profiles and in search, absent from home timelines and
import-generated notifications, and subtly labelled wherever a post is shown.
The pre-existing Instagram follower importer and DM verification flow were not
changed.

## 2. Outcome

The planned production code paths are implemented for the deterministic,
credential-independent behavior in `01-requirements.md`. Automated evidence is
reported at its actual layer below; unit or mocked coverage is not presented as
a live OAuth, current-export, deployed-origin, or firehose result.

- Steps 1–15 below are implemented. Their evidence is intentionally layered
  across unit, integration, component, real-browser smoke, AppView/Postgres,
  and Flutter tests; the acceptance IDs describe contracts and are not each a
  standalone end-to-end browser scenario.
- `MAN-001`–`MAN-005` remain External / Pending. They require a live OAuth
  client/PDS, additional consented current exports, large-device/browser
  checks, live firehose convergence, or a deployed Cloudflare Pages project.
- The consented export was used only to understand directory and JSON shape.
  Its archive, content, account name, paths, and values were not copied into
  source, fixtures, diagnostics, snapshots, or emitted artifacts.

## 3. TDD Execution Record

The implementation used focused red/green cycles followed by broad regression
gates. This was a long, multi-agent implementation turn, so transient red
stdout was not retained as a durable artifact. The table records the observed
pre-implementation failing condition without inventing command output. The
durable evidence is the linked test source and the final commands in section 6.

| Step | Test IDs | Initial failing condition | Implementation and durable green evidence | Status |
|---|---|---|---|---|
| 1 | IT-001, UT-009 | The post Lexicon, generated Go/web types, and ADR had no provenance contract. | Added ADR 008 and optional bounded `externalImport.source`; regenerated Go CBOR/types and generated the importer contract. Contract/post-record tests cover valid Instagram, unknown bounded, absent, and invalid shapes. | Complete |
| 2 | IT-009, IT-010, IT-011, UT-014 | AppView had no reversible provenance storage or original-date profile sort. | Added migration 000032, exact source classification, `profile_sort_at`, replacement clearing/retention, response propagation, pagination and query-plan coverage. | Complete |
| 3 | IT-011–IT-014 | Imported originals followed ordinary timeline and notification activation behavior. | Profile/search use original chronology; only the authored home-timeline arm excludes imports; source-record notification activation is suppressed while later engagement remains ordinary. | Complete |
| 4 | REG-001–REG-006 | Cross-stack changes had no ordinary-post controls. | Existing and new AppView suites cover absent provenance, ordinary profile order, timelines, notifications, search, deletion, moderation, relationships, saves, and interaction counts. | Complete |
| 5 | UT-015, IT-015, REG-007 | Flutter models and cards could not decode or display provenance. | Added optional full/quote provenance, generated mapper changes, localized copy, and one subtle accessible label in both shared card shapes. Ordinary and unknown sources remain unlabelled. | Complete |
| 6 | UT-001, UT-013, IT-016, REG-008 | No standalone importer package or runtime policy existed. | Added an independent checked npm/Vite/React/TypeScript package, versioned safety policy, local/staging/production/preview modes, self-hosted fonts, static headers, noindex policy, generation/build scripts, and repository recipes. | Complete |
| 7 | UT-002, UT-003, UT-018, IT-002, IT-003 | No safe archive boundary or export adapters existed. | Added classic ZIP/ZIP64 envelope checks before zip.js allocation, Blob-backed worker enumeration, actual-output-bounded extraction, exact path allow-listing, one-wrapper support, legacy/detailed adapters, fail-closed draft/publication-state handling, conflict handling, operation fencing, cancellation, minimized messages, and available original dates on content-free skipped rows. Current dual-representation exports use the first non-empty supported caption and media-attached timestamp before metadata fallbacks, with a synthetic regression proving equivalent representations merge. The media allow-list covers only the observed flat Posts, six-digit date-partitioned Posts, and flat Other forms; current-export structural validation maps all 72 post records and finds every referenced ZIP entry. | Complete |
| 8 | UT-004–UT-006, UT-008 | No deterministic timestamp, caption, facet, or record-key normalization existed. | Added seconds/milliseconds canonicalization, inclusive date policy, conservative mojibake repair, grapheme truncation, UTF-8 link/hashtag facets, plain Instagram handles, stable per-source deterministic TIDs, and fail-closed canonical collisions. Same-time keys remain stable across archive order, reload, and later overlapping exports. | Complete |
| 9 | UT-007, UT-019, IT-004 | Media could not be safely classified or sanitized in a worker. | Added signature/header inspection, dimensions/pixel/ratio/animation limits, unsupported-format omission, explicit text-only confirmation, first-four-usable-in-source-order policy, one-at-a-time decode/re-encode, metadata stripping, bounded final blobs, cancellation checkpoints, transferable results, and one explicit preview at a time. | Complete |
| 10 | UT-016, UT-020, AT-001, AT-002, AT-004 | There was no local review or public-content acknowledgement UI. | Added acknowledge/select/inspect/review flow, exact counts, warnings, filters, caption edits, per-image selection, bulk actions, text-only confirmation, and TanStack virtualization for both importable and skipped lists. Each mounted post row automatically displays its first sanitized image without generating a separate thumbnail asset; images two through four remain disclosure-gated. Tests exercise 2,000-row lists with bounded DOM, and the worker result retains only the bounded prepared caption rather than an unbounded raw source caption. | Complete |
| 11 | UT-010–UT-012, UT-017, IT-005 | No resumable, content-free progress model existed. | Added Dexie session/item storage bound to DID and manifest fingerprint, per-post durable state, safe codes, deterministic retry state, clear/sign-out ownership, caught recovery-action failures, and fingerprints that exclude captions, media names, archive names, and source values. | Complete |
| 12 | IT-006, AT-003 | No browser OAuth or CraftSky membership preflight existed. | Added popup callback handling, preloaded OAuth metadata, exact granted-authority checks, library-owned session restoration/sign-out, and `social.craftsky.actor.profile/self` membership verification. Only exact record absence maps to non-membership; auth/network/service failures remain distinct. | Complete |
| 13 | IT-007, AT-005–AT-007 | No create-only PDS publisher or safe rollback existed. | Added sanitize/upload/create sequencing, no-overwrite collisions, bounded transient retry, pause/resume/cancel, ambiguous-create reconciliation, exact existing-record/blob comparison, an owned-CID resume fast path, and CID-guarded rollback of only durably owned creates. Rollback separately records deletion, already-absent, changed/conflict, and failure outcomes. | Complete |
| 14 | IT-008, IT-017, IT-018, AT-008, AT-010 | The layers were not composed into a production path. | Added `BrowserImporterServices`, shared publication/recovery transitions, partial-import recovery UI, per-item persisted progress, normal sign-out/clear paths, preview hard-disablement, and a browser mock harness excluded from the production build. Real-Chrome tests cover local archive review and cancellation plus a mocked OAuth/retry/rollback/sign-out UI composition; live PDS and AppView behavior remains separately classified. | Complete |
| 15 | AT-009, REG-008 | Cross-surface provenance and package isolation had no acceptance controls. | AppView response, Flutter model/widget, generated-contract, source/path scan, and emitted-artifact checks bind the feature while keeping historical-post import code out of Flutter without blocking its separate following-list ZIP parser. | Complete |
| 16 | MAN-001–MAN-005 | Live services, current external data, deployed infrastructure, and multiple physical browser environments are unavailable in repository automation. | See section 7. These are release checks and are not reported as passing. | External / Pending |

## 4. Implemented Architecture

### Standalone local-first importer

`instagram-importer/` is an independent static SPA. The selected ZIP remains a
browser-owned `File`/`Blob`; the app never creates a whole-archive
`ArrayBuffer`, uploads the archive, or persists it. A module worker:

1. validates classic EOCD or ZIP64 directory metadata and configured limits;
2. enumerates the central directory using zip.js;
3. accepts only exact supported post JSON/media paths under zero or one wrapper;
4. adapts supported legacy and detailed published-post shapes;
5. returns only a minimized, content-bounded review manifest; and
6. later reads and sanitizes one selected image at a time.

Directory progress is emitted every 250 entries or 50 ms, whichever occurs
first. Cancellation is checked around enumeration, extraction, normalization,
media reads, decode, encode, and result transfer. Large review lists are
virtualized rather than materialized as thousands of DOM rows.

### Publication and recovery

OAuth begins only after local review and a user-activated action. The client
requests exactly:

```text
atproto
repo:social.craftsky.feed.post?action=create&action=delete
blob?accept=image/jpeg&accept=image/png&accept=image/webp
```

The selected DID must contain `social.craftsky.actor.profile/self` before any
write. The member then confirms the exact DID and selected post/image counts.
Publication uploads sanitized selected images and creates deterministic
`social.craftsky.feed.post` records without update authority.

Importer-owned IndexedDB stores content-free progress only. Exact returned
record CIDs establish rollback ownership. Existing matching records remain
unowned unless the session already had a durable owned-create CID. Conflicts,
lost responses, edited records, and CID-different recreated records fail
closed. Rollback uses `swapRecord` and never deletes a record whose current CID
differs. The confirmation discloses that a byte-identical delete-and-recreate
at the same rkey has the same content CID and cannot be distinguished.
Cancellation aborts only the active media transform or PDS request, retains the
selected archive for retry, stops scheduling later posts, and records the
import as paused rather than destructively rolling it back.

### Public provenance and AppView policy

The optional public record field is:

```json
{
  "externalImport": {
    "source": "instagram"
  }
}
```

It is self-asserted display/distribution metadata, not account verification.
AppView recognizes only the exact `instagram` token. The authoritative current
record controls classification, so a replacement that omits the field clears
it. Imported originals:

- appear on the author profile using original `createdAt` chronology;
- remain keyword/hashtag searchable;
- do not appear as original authored items in home timelines;
- do not activate notifications from the historical source record; and
- can still produce normal later repost, quote, reply, like, and other
  engagement behavior.

Flutter renders “Imported from Instagram” subtly on full post cards and quote
previews. Ordinary posts and unknown future provenance values render no label.

### Runtime and deployment policy

- Local loopback: real test OAuth/PDS operations enabled with loopback client
  metadata; `localhost` is canonicalized to `127.0.0.1` before archive or OAuth
  state is created so the callback remains on one RFC 8252-compatible origin.
- Production: enabled only at `https://import.craftsky.social` with canonical
  public metadata.
- Stable staging: enabled only when the configured HTTPS origin, client ID, and
  explicit write flag agree.
- Ephemeral preview/unknown origins: OAuth initialization, restoration,
  preflight, upload, create, and delete are hard-disabled.

The static header baseline keeps scripts and stylesheet files self-only,
allows only dynamic style attributes required by list virtualization, permits
OAuth popups with `same-origin-allow-popups`, forbids framing and objects,
disables sensitive browser capabilities, uses noindex/no-store defaults, and
contains no analytics or remote-font allow-list.

## 5. Deviations And Review Fixes

The approved product behavior did not change. The static security baseline was
clarified during review for virtualized positioning and OAuth popup return, as
recorded in `03-document-review.md` DR-009 and `04-coding-plan.md`. File/test
organization was consolidated where one real boundary gave stronger evidence
than several artificial files:

- Browser worker/privacy/media scenarios are in `e2e/local-review.spec.ts`
  instead of separate `archive-worker.spec.ts`, `archive-privacy.spec.ts`, and
  `media-worker.spec.ts` files.
- The full mocked publication/recovery path is in
  `e2e/mocked-flow.spec.ts`, backed by focused service/repository tests,
  instead of separate `import-flow.spec.ts`, `privacy-boundary.spec.ts`, and
  `recovery.spec.ts` files.
- Archive inspection coverage is split across `zipEnvelope.test.ts`,
  `inspectArchive.test.ts`, path/adapter tests, the real worker browser tests,
  and safety-boundary tests.
- The generated web contract is deliberately small and derived directly from
  the repository Lexicon rather than adding a second general Lexicon runtime.

Final independent audit findings were fixed before handoff:

- same-timestamp posts and manifest fingerprints are order-independent;
- the production service and recovery tests share publication state helpers;
- both review lists use true virtualization;
- directory progress honors the entry-count and elapsed-time thresholds;
- profile-read service failures are not mislabeled as non-membership;
- OAuth metadata is prepared before the popup-triggering click;
- preview rendering automatically requests the first image of each mounted
  virtual row while later images remain explicitly disclosure-gated and all
  transforms stay serialized;
- exact-existing recovery preserves rollback ownership;
- exact owned-CID recovery avoids re-sanitizing already-created media;
- partial imports expose rollback and per-item progress;
- active media cancellation retains the archive and returns a retryable paused
  state;
- rollback distinguishes a successful deletion from an already-absent record
  in both durable and visible status;
- deterministic rkeys remain stable when a later export adds another
  same-time post, while true canonical collisions remain fail-closed;
- equivalent media paths in different Unicode normalization forms merge as
  one normalized source identity;
- media selection continues in source order until four usable images are
  retained rather than slicing before validation;
- actual emitted decompressed bytes are bounded even when ZIP size fields
  under-report metadata or media output;
- detailed export publication-state values fail closed, and skipped rows retain
  every available normalized source date without crossing source content;
- only the bounded prepared caption crosses the worker result boundary;
- encrypted and unsupported-compression entries reject the archive instead of
  allowing a partial import;
- configured archive-limit failures retain their bounded safe error codes so
  the UI can give the intended Posts-only guidance;
- notification subject hydration retains imported-post provenance;
- author-list cursors retain the pre-change opaque wire shape while carrying
  the new profile chronology value;
- retries do not restart completed work;
- destination switching does not move on after failed OAuth sign-out, and
  publication, rollback, sign-out, and clear-history failures use
  stage-accurate recovery copy;
- Playwright runs its shared local-server browser flows in one worker after a
  parallel run demonstrated cross-page reload contention;
- the CSP/COOP headers support virtualization and OAuth popups without opening
  script or framing policy; and
- the component-test timeout is explicitly bounded at ten seconds to avoid a
  5.159-second resource-contention flake in the full four-worker suite.

`just lexgen-check` compares generated output with `HEAD`; it therefore reports
the intentional uncommitted generated feature diff. Generator idempotence was
verified instead: the SHA-256 of the relevant binary diff was
`1b1ef55814bf071ac9f47dc6f8821e61385af79b711d4cfa307bc56c460eae0d`
before and after a fresh `just lexgen` run.

## 6. Verification Evidence

### Importer

| Command | Result |
|---|---|
| `npm run lint` | Passed, full package |
| `npm run typecheck` | Passed as part of both release builds |
| `npm test -- --maxWorkers=4` | Passed: 31 files, 150 tests |
| `PLAYWRIGHT_CHANNEL=chrome npm run test:e2e` | Passed: 6/6 real-Chrome scenarios using the checked one-worker configuration |
| `npm run build` | Passed: generation check, TypeScript, production Vite bundle, canonical OAuth metadata |
| `npm run build:preview` | Passed: generation check, TypeScript, Vite bundle, invalid/non-client metadata marker |
| Emitted production artifact inventory/privacy scan | Passed: no test harness, source map, consented-export identifier, archive-name canary, message canary, or private source-name canary |
| Static metadata/header inspection | Passed: exact production origin/scope, restrictive headers, no analytics/remote fonts |

The production JavaScript chunk is 1,669.30 kB minified (381.46 kB gzip),
which produces Vite's advisory 500 kB chunk warning but does not fail the
build. The archive worker is emitted as a separate 215.42 kB module.

### Lexicon and AppView

| Command | Result |
|---|---|
| `just lexgen` plus pre/post generated-diff hash | Passed; idempotent |
| `cd appview && go test ./... -count=1` | Passed; non-race package suite |
| `TEST_DATABASE_URL=… go test -race -p 1 ./...` | Passed across every package against Compose Postgres |
| `cd appview && go vet ./...` | Passed |
| `just test` | Passed; canonical race-enabled suite with `TEST_DATABASE_URL` set against Compose Postgres, so database tests did not silently skip |

### Flutter

| Command | Result |
|---|---|
| `flutter analyze` | Passed |
| `flutter test` | Passed earlier in this implementation turn: 946 tests |
| `flutter test test/feed/models/post_test.dart test/feed/widgets/post_card_test.dart test/l10n/imported_post_l10n_test.dart` | Passed after final code changes: 65 tests |
| `dart format --output=none --set-exit-if-changed …` | Passed for all 7 changed Dart files |

A second full Flutter run was stopped after 333 passing tests because concurrent
local tooling made it take more than thirteen minutes; it had no observed test
failure. It is not counted as a pass. No Flutter production behavior changed
after the earlier full 946-test green run; the only later Flutter edit wrapped
a long test description, and the final focused 65-test provenance suite plus
analysis are green.

### Repository and traceability

- The acceptance specification contains each intended automated ID as a
  complete set:
  `AT-001`–`AT-010`, `UT-001`–`UT-019`, `IT-001`–`IT-018`, and
  `REG-001`–`REG-008` (55 automated IDs).
- All five `MAN-*` checks are explicitly external/pending.
- `git diff --check`, formatting checks, ignored-output checks, and the final
  privacy/scope scan passed after the final code changes.

The four broad browser-flow contracts are represented by five focused
real-Chrome scenarios and deeper lower-layer tests, not by a live full-system
acceptance suite. The repository also does not directly prove push delivery,
multi-page production traffic boundaries, every configured numeric boundary
through a real browser, or the deployed query plan for every joined profile
query. Those evidence gaps are non-blocking for the credential-independent
implementation but remain part of the external/manual release work below.

## 7. External / Pending Release Checks

| ID | Pending evidence | Why it remains external |
|---|---|---|
| MAN-001 | Real browser OAuth, exact granted scope, profile/self read, create/delete against target PDS implementations | No production/staging OAuth application and test PDS matrix are configured |
| MAN-002 | Additional current consented export shapes in Chrome, Edge, Firefox, and Safari | Two consented July 2026 structures informed synthetic fixtures. The latest 1.7 GB export completed a local Chrome review with 71 selected posts, 197 selected images, one video-only skip, and no unsupported-shape entries; further private exports and Edge/Firefox/Safari remain external |
| MAN-003 | Very large/sparse ZIP64 archives, browser suspension/reload, memory behavior, and real rate limiting on representative devices | Device/browser/PDS resource behavior cannot be proven by deterministic repository tests |
| MAN-004 | Live PDS → relay/Tap → AppView → Flutter convergence | Requires a live test identity, PDS publication, firehose delivery, and client inspection |
| MAN-005 | Deployed Cloudflare Pages project isolation, effective headers/CSP, network panel, rollback, and dedicated subdomain | No deployment was authorized or performed |

These checks must be completed before enabling production publication. A
repository pass does not convert them to green.

## 8. Handoff

- Implementation review: `06-implementation-review.md`
- Importer operating/deployment guide: `../../../instagram-importer/README.md`
- No stage, commit, push, pull request, or deployment action was performed.
