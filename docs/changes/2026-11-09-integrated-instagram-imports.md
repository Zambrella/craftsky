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
Migration `000122_instagram_post_imports` adds private history and lifecycle purge
coverage. No Lexicon change or mobile implementation was needed for this
explicitly web-client request. See ADR 027 and web-client/README.md.

The standalone package and its Just recipes/documentation entrypoints were
removed. The separately named directory
`/Users/douglastodd/Projects/craftsky/instagram-importer` was also deleted as
requested. No hosted site, DNS, production data or OAuth grants were changed.

## Validation

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
were cleaned up; unrelated Docker data was left untouched.

## Boundaries

Browser tests mock AppView publication/history; focused backend tests use local
PDS doubles and real PostgreSQL. These do not prove live PDS publication or
production indexing. Nothing was deployed.

Duplicate history covers sources published through the integrated flow. Existing
standalone imports still have legacy deterministic record-key collision
protection at publication; their history is not proactively reconstructed by the
new lookup endpoint. The source-key check does not claim similarity detection for
manually copied or substantially altered exports.
