# AT Protocol audit corrections — 12 October 2026

This change addresses the nine confirmed findings from the documentation audit.
The official documentation inventory contained 108 successfully fetched pages
from the AT Protocol and Bluesky protocol-service documentation trees.

| Finding | Correction |
| --- | --- |
| Repository snapshots strip extensions or fail on unknown open-union variants | Decode verified CAR records as generic AT data, preserving unknown fields and union variants. |
| Ingestion checks only part of the schema | Validate the full record data model and supported Lexicons, including required `$type`, nullability, nested shapes, CID syntax, and protocol datetimes. |
| Custom schemas lack public authority | Publish all 17 schemas under the official CraftSky DID and configure all eight DNS authority groups; provide repeatable publication and verification commands. |
| CraftSky profile updates replace unknown fields | Merge the editable profile fields into the current authoritative record on both write paths. |
| Generic JSON decoding rounds large integers | Preserve JSON numbers on PDS reads, record-list pagination, profile updates and command dispatch; validate signed 64-bit bounds without a float conversion. |
| HTTP 429 becomes a terminal command rejection | Retain upstream retry guidance, journal a known unapplied rate-limit outcome, enforce its cooldown, and resume the original operation key. Flutter retains the frozen command and honors that delay. |
| Request limits count code points | Count grapheme clusters while independently enforcing byte limits. |
| OAuth requests broad transitional authority | Request explicit collection/action, MIME and method grants; reject and clean up callbacks missing required grants. |
| Future timestamps can pin chronological results | Clamp sorting timestamps to indexing time across search, conversations and imported author posts, with matching cursors and a migration for existing profile sort values. |

No custom schema shapes changed. The new rate-limit API error is HTTP 429,
`pds_rate_limited`, with integer `Retry-After` seconds and the standard error
envelope. Repeating a keyed mutation uses the same endpoint, body and key.
Uncertain transport outcomes retain the existing read-only reconciliation path.

OAuth defaults are defined in `appview/internal/auth/scopes.go`, mirrored in the
environment examples and `render.yaml`. CraftSky can create/update its Bluesky
profile and manage its nine used record collections, upload supported media,
and request service tokens restricted to the video limits or blob-upload
methods. No general repository, identity, email, chat, or arbitrary RPC grant
is requested. The RPC audience wildcard supports dynamically hosted PDSes;
the backend still constrains the service-token audience and method.

Backend migrations and OAuth configuration require the normal reviewed release
and environment-update process. Existing credentials granted broader scopes
must be replaced through OAuth reauthentication when rolling out that policy.
This work does not deploy AppView or change production session rows.

Regression coverage includes signed CAR preservation, schema rejection and safe
diagnostics, large-integer round trips, profile extension preservation, denied
OAuth grants and credential cleanup, durable same-key retries after throttling,
and future-date ordering across paginated search and conversations.

Sources: [Lexicon](https://atproto.com/specs/lexicon),
[Data model](https://atproto.com/specs/data-model),
[Permissions](https://atproto.com/specs/permission),
[Bluesky protocol services](https://bsky.network/docs/protocol-services).

## Verification

- Initial audit verification used Flutter 3.44.9 (the CI pin at that time):
  full suite, 2,901 passed and 38 skipped; full static analysis reported no
  issues. Flutter 3.47.7 initially reproduced an existing pending-handoff
  interceptor test failure. The subsequent dependency-upgrade merge fixes
  that failure and passes the full suite on Flutter 3.47.7: 2,912 passed,
  38 skipped, and no analysis issues. Formatting, code generation, and the
  JavaScript/WebAssembly release web build also pass on the upgraded SDK.
- All 17 public schemas match their checked-in source; all eight DNS authorities
  resolve to the official DID. `goat lex check-dns` resolves every schema NSID,
  and `goat lex resolve social.craftsky.feed.post` succeeds through public discovery.
- PostgreSQL regressions cover future-date ordering and cursors, imported-profile
  sorting, denied OAuth grant cleanup, and durable resumption after HTTP 429.
- The complete `scripts/appview-check` release gate passed: generated-code
  drift, Go formatting/vet/staticcheck, source and binary vulnerability scans,
  full Go tests and race tests with PostgreSQL/MinIO, Docker builds, source-free
  media probes, migration rollback/reapply and failure handling, and the pinned
  Tap handshake. Tap v0.1.10 has no deterministic safe event injection API,
  so the gate verifies its handshake without injecting events.
- The final command-journal correction passed the full durable-command race
  suite, and `just lexgen-check` passed after the catalog script cleanup.
- `git diff --check` passed. Code, migrations and OAuth environment defaults
  remain local; the public schema and DNS publication steps are live.
