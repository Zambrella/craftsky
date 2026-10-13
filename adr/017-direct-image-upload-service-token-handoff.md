## Architecture Decision Record

- Status: Accepted
- Aspect: Immediate image transfer, OAuth credential boundary, image validation
- Date: 2026-10-13
- Decision: Authorize direct Flutter-to-PDS image uploads with a five-minute service JWT

### Context and decision

Immediate images currently pass through AppView before being uploaded to the
owner's PDS. The maintainer approved direct uploads, retaining Flutter image
preparation and accepting PDS enforcement without AppView decoding these bytes.
This extends ADR 012's narrow credential exception and supersedes ADR 011's
server-decoding requirement for immediate uploads only.

`POST /v1/blobs/images/authorization` uses current-member, device-bound session
middleware and upload rate limiting. The fenced OAuth session coordinator
revalidates the owner and PDS authority before requesting
`com.atproto.server.getServiceAuth`. The response contains only `token`,
`expiresAt`, and the exact HTTPS `uploadEndpoint`, derived from that same
session's PDS origin. The JWT audience is the PDS and its `lxm` is
`com.atproto.repo.uploadBlob`; expiry is five minutes after issuance.
OAuth access/refresh tokens and DPoP material remain AppView-only.

Flutter sends prepared image bytes to that exact endpoint using a separate HTTP
client without CraftSky session, device, logging, or error/sign-out interceptors.
It rejects redirects, malformed destinations and expired authorization, bounds
responses, and requires the returned blob's CID and size to match the sent bytes
and MIME to be a supported image type. Capabilities are never persisted,
logged, placed in URLs, or stored in retry state. Each attempt obtains fresh
authorization. Managed clients cancel uploads on account-provider disposal and
app lifecycle interruption. Terminal paths close the isolated transport and
remove authorization headers and request payloads from retained transport errors.

Record creation stays AppView-mediated. PDS upload and publication are separate;
the PDS controls temporary unreferenced blob retention. CraftSky does not add a
client blob-deletion facility.

### Validation boundary and consequences

Flutter keeps existing metadata removal, resizing, compression and image-byte
validation, with 2,000,000 encoded bytes and 4000 pixels per decoded side. The
upload client independently rejects empty, oversized or non-image inputs.
Post APIs keep their blob metadata and Lexicon checks; the PDS verifies blob
existence and metadata when records reference it. This does not establish
server-side decoded geometry for a modified client. Existing asynchronous image
safety processing is not replaced by this upload change.

A service JWT is method- and audience-scoped, not image-, MIME-, size-, CID-,
one-use- or upload-count-scoped. A client can use it for additional generic blob
uploads until expiry, subject to PDS quotas. Local cancellation or logout does
not revoke an already issued JWT. These are accepted residual risks, bounded by
the short lifetime. The token cannot create records or refresh OAuth sessions.

Scheduled/private media retain private S3 storage and server-side validation;
they do not use direct PDS uploads. The old mediated image endpoint remains
available, but the Flutter immediate image path does not fall back to it.

Compared with retaining mediation, direct uploads remove AppView image ingress
and forwarding traffic and avoid immediate server decoding. They add client
credential handling and depend on PDS service-auth support and browser CORS.
Live native/web uploads against supported PDS hosts are release checks; mocked
transport tests do not establish those hosts' deployed compatibility.

### API and schema impact

Adds the authorization endpoint; no record Lexicon, database migration, record
creation route, or video feature-gate change. Profile, business/event, post and
project immediate images use the shared transfer path.

### Validation evidence (2026-10-13)

- Flutter 3.44.9: full `flutter test` passed, 2,917 tests and 38 skipped.
- Headless Chrome: shared upload-client and Fetch transport suites passed, 16 tests.
- `flutter analyze`: no issues.
- `./scripts/appview-check`: all release gates passed, including PostgreSQL/MinIO
  integration tests, race tests, static/vulnerability checks, Docker binary/build
  checks, smoke checks, pinned Tap handshake and resource budgets.
- The three affected Riverpod provider hashes were regenerated; unrelated
  generated output churn was removed. `git diff --check` passed.
- No deployment or live authenticated PDS upload was performed. Native and web
  uploads against supported deployed PDS hosts remain release checks.
