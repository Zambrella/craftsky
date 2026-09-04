## Architecture Decision Record

- Status: Accepted
- Aspect: OAuth credential boundary, Flutter networking, video upload
- Date: 2026-09-03
- Decision: Permit one ephemeral upload-only service JWT in Flutter

### Why I needed to decide this

CraftSky's confidential OAuth session stores access tokens, refresh tokens, and
DPoP private keys in AppView. The original video-post plan therefore proxied up
to 300,000,000 bytes through AppView to `video.bsky.app` and persisted video-job
state there.

Bluesky's documented video flow instead lets a client request
`com.atproto.server.getServiceAuth`, present the resulting PDS-signed service JWT
directly to `app.bsky.video.uploadVideo`, and poll public job status. Direct
upload removes duplicated AppView bandwidth and the private job store, but it
temporarily gives the untrusted client a replayable bearer credential.

The service JWT is not an OAuth access token. It carries no refresh authority or
DPoP private key and is bound by audience, lexicon method, and expiry.

### Options I considered

**Option 1: AppView proxies the upload and owns jobs - not chosen**

Pros:

- Keeps every PDS-issued credential server-side.
- Centralizes upload and status normalization.

Cons:

- Sends every large video through AppView ingress and egress.
- Adds large-body admission, timeout, durable-job, restart, and cleanup work.

**Option 2: Flutter receives a narrow service JWT and uploads directly - chosen**

Pros:

- Follows the documented direct video-service flow.
- Removes large AppView media proxying and private video-job persistence.
- Keeps OAuth access/refresh tokens and DPoP keys server-side.

Cons:

- Flutter temporarily holds a replayable bearer.
- Requires exact destination and redirect controls plus client-side secret scans.
- Requires AppView to verify untrusted completion data before publication.

**Option 3: Flutter receives OAuth access and DPoP credentials - not chosen**

Pros:

- Would enable a general Token-Mediating Backend client.

Cons:

- Broadly changes every authenticated PDS operation and device session.
- Requires on-device DPoP key lifecycle, nonce handling, refresh, and revocation.
- Is unnecessary for direct video upload.

### What I decided

AppView may return one PDS-signed service JWT to an authenticated, device-bound
Flutter request only for direct Bluesky video upload. The JWT must:

- have the current user's PDS as audience;
- bind `lxm` to `com.atproto.repo.uploadBlob`;
- expire after issuance and no more than 30 minutes later;
- be returned with expiry only, without OAuth access/refresh tokens, DPoP
  material, PDS URL, or any broader credential.

Flutter must keep the JWT in memory only. A dedicated external HTTP client may
attach it only to the exact configured HTTPS `video.bsky.app` origin and
`/xrpc/app.bsky.video.uploadVideo` path. Redirects are rejected. The normal
CraftSky bearer interceptor is not installed on that client. The JWT and raw
authorization header are excluded from persistence, logs, telemetry, crash
reports, URLs, equality/debug output, and retry objects, and are cleared on every
terminal, cancellation, account-change, and lifecycle-interruption path.

Flutter polls public `app.bsky.video.getJobStatus` without the JWT. When creating
the post it submits the job ID and BlobRef. AppView independently queries job
status and requires completed status, `jobStatus.did` equal to the authenticated
DID, and exact BlobRef equality before writing the post. It fails closed without
disclosing foreign job details.

The authenticated limits endpoint uses a separate method-bound service
authorization for `app.bsky.video.getUploadLimits` only inside AppView. That
credential is never returned to Flutter or persisted.

### Compatibility and evolution

- This is not a general TMB migration. OAuth access/refresh tokens and DPoP keys
  remain AppView-only.
- Flutter still does not contact a PDS directly and still reads records through
  AppView.
- Existing PDS writes remain AppView-mediated. The sole direct external write is
  the purpose-bound video upload, whose result is stored by the service on the
  user's PDS.
- Any additional client-held PDS-issued credential requires a separate ADR and
  an explicit amendment to the architecture rules.
- If `getJobStatus` can no longer support independent completion verification,
  publication must fail closed until a replacement verifiable receipt or narrow
  registration design is approved.

### Consequences and notes

- Add authenticated `POST /v1/blobs/videos/authorization`; remove the planned
  AppView video upload/status proxy routes and video-job migration.
- Browser CORS and native redirect behavior are release gates.
- The service JWT's bearer nature remains residual risk until expiry; use the
  shortest lifetime accepted for the upload, capped at 30 minutes.
- `already_exists` may supply a reusable BlobRef, but AppView still verifies the
  owner and exact blob. CraftSky does not invent an unsupported idempotency key.
