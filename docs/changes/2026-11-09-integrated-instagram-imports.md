# Integrated Instagram imports: implementation and validation

The web client now offers local Instagram post-export review within verified
Instagram settings. Each source opens the existing standard/project composer.
Full captions and original dates are retained. Images and videos share the
existing preparation/upload/publishing paths; mixed carousels remain unsupported.
The archive worker extracts selected media lazily and enforces archive/image
safety bounds. The archive stays in memory across composer navigation, while
selected drafts/recovery use account-scoped browser storage.

AppView requires active uncontested verification, pins each private source
identity atomically with command preparation, and records confirmed publication
history in the command acceptance transaction. Existing create-only PDS commands
handle concurrent submissions and unknown-result recovery. Original timestamps
and the existing public Instagram provenance preserve ADR 008 distribution.
Migration `000128_instagram_post_imports` adds private history and lifecycle purge
coverage. The migration was renumbered from `000122` while merging main to
preserve main's already allocated migrations `000122`–`000127`. The original
release-gate logs below predate that merge. The initial work covered the web
client and AppView. See [ADR 029](../adr/029-integrated-instagram-post-imports.md)
and the [web client guide](../../web-client/README.md).

The standalone package and its Just recipes/documentation entrypoints were
removed. The separately named directory
`/Users/douglastodd/Projects/craftsky/instagram-importer` was also deleted as
requested. No hosted site, DNS, production data or OAuth grants were changed.

## Validation

After merging `origin/main` at `82b5d790`, the web production build, format,
lint, type checks, 659 unit tests and 67 focused browser tests passed. The
browser runs used isolated local fixture ports and covered Instagram import
on the preview and Worker builds, verification, following import, composer
validation, scheduling and publication recovery. Focused AppView
tests passed against PostgreSQL 16 with migration `000128`, covering import
publication and recovery, duplicate reservations, route diagnostics and owner
purge. The documentation index and checks also passed.

Passed locally:

- Web Svelte/type checking: zero errors/warnings; ESLint and Prettier checks.
- Full web release gate: 535 unit tests, including synthetic archive/parser,
  image-header, source identity, ZIP64, draft/recovery and account-session cleanup;
  1,014 browser passes across preview and Cloudflare Worker, plus 20 Python
  script-wrapper tests. Final normal production build restored successfully.
  Four intentional browser skips cover the disabled-subscription test in enabled
  runs (tested separately) and optional live Bunny playback without a configured
  service. Full log: `/private/tmp/craftsky-instagram-web-release-resume.log`.
- Production web build, including the real archive worker bundle.
- Two new Playwright flows using the actual ZIP worker and composers: verified
  gate, local-only review, thumbnail, cancel/return/list retention, project
  publication handoff, confirmed row status, reupload duplicate detection,
  home-overlay exclusion, and explicit mixed/excess media video selection.
- Two existing Instagram browser regressions for verification/following import,
  including the BZip2 following archive path.
- Focused AppView tests against disposable real PostgreSQL: verification,
  original timestamp/provenance, project/video reuse, source reservations and
  concurrency, create-only collision protection, uncertain-response recovery,
  history surviving command compaction, owner isolation, cleanup inventory,
  lookup bounds and exact accepted-operation replay.
- Route catalogue and suspended-account policy tests after registering the new
  lookup route; server construction check; gofmt and git diff whitespace checks.
- Full AppView gate module verification, generated Lexicon drift check, go vet
  and Staticcheck passed after fixing the initial findings.

The complete AppView release gate passed after Docker disk space was freed and
two integration findings were fixed: import storage is constructed through the
Instagram dependency bundle, and the private lookup route has explicit
diagnostic metadata. Focused architecture/catalogue/diagnostic tests passed
before the full rerun.

The successful full gate covered ordinary and race-enabled Go suites with required
PostgreSQL/MinIO coverage, module/generated-schema checks, go vet, Staticcheck,
source and release-binary vulnerability scans, production/media evidence images,
release CLI dependency probes, migration apply/down/reapply, failure-path
migration checks, startup health, packaged console assets, the pinned Tap
WebSocket handshake, and release media-resource bounds. No reachable
vulnerabilities were found. The pinned Tap check proves readiness/configuration
and handshake; it does not inject live records.

Successful backend evidence:

- Log: `/private/tmp/craftsky-instagram-appview-check-final.log`
- Artifacts: `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.pMPGY6Umd3`

The earlier disk-space failure is resolved. All release-gate disposable services
were cleaned up.

## Flutter extension

The verified Instagram settings now offer the same post and project import on
iOS and Android. Flutter reads the ZIP on the device and keeps a temporary copy
while the review is open. Existing composers reuse saved draft edits, and copy
only the selected photos or video into draft storage. The shared AppView lookup
surfaces publication history from either client, and the composer uses the
existing immediate publication endpoint with the private import reference.

The full Flutter suite passed locally with 4,371 tests and 50 skips. Dart
analysis and the documentation checks passed. Tests cover archive extraction,
matching web source keys, the verified settings entry point, synthetic ZIP
review and duplicate status, draft reuse and publication handoff. The Android
debug APK built and reached the sign-in screen on an emulator. The device file
picker, video playback and upload, and live publication remain unverified. This
extension required no AppView or Lexicon changes.

After the follow-up fixes, the full Flutter suite passed with 4,380 tests and
50 skips. Regression checks cover replaying the original publication request
after reopening a pending draft, saving or removing restored videos, continuing
after unusable attachments, and checking only new rows or failed lookups. Dart
analysis passed. The Android emulator reclaimed a synthetic orphaned archive at
startup, reached the sign-in screen, and hot-reloaded successfully.

## Boundaries

Browser tests mock AppView publication/history; focused backend tests use local
PDS doubles and real PostgreSQL. These do not prove live PDS publication or
production indexing. Nothing was deployed.

Duplicate history covers sources published through the integrated flow. Existing
standalone imports still have legacy deterministic record-key collision
protection at publication; their history is not proactively reconstructed by the
new lookup endpoint. The source-key check does not claim similarity detection for
manually copied or substantially altered exports.
