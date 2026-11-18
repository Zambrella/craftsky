# ADR 024: Bunny Stream video integration

- Status: Accepted for development; production rollout follows manual acceptance
- Date: 2026-11-02

## Context

Bluesky-derived HLS assets have disappeared for CraftSky posts even when the
referenced MP4 remains available on the author's PDS. CraftSky needs independent
processing, delivery and retention. The maintainer selected Bunny Stream and
authorized development setup and implementation before manual acceptance.

## Decision

Retain the standard `app.bsky.embed.video` record and PDS BlobRef. No Lexicon
changes. Preserve the existing Bluesky implementation behind the default
`VIDEO_PROVIDER=bluesky`; the local override selects `bunny`. Production
configuration and deployments are unchanged.

For Bunny, AppView creates an owner/generation-bound durable job and returns
one 30-minute TUS signature scoped to that vendor video ID. Flutter and the Svelte client send the
signature only to the fixed HTTPS Bunny TUS endpoint or validated same-origin
upload resource. Redirects, foreign locations, query strings and fragments are
rejected. The capability stays in the active upload attempt and is cleared on
success, cancellation, failure and lifecycle interruption. The library/account
API keys and PDS OAuth credentials stay server-side.

This extends ADR 012's direct external video-upload decision with a separate
Bunny capability; it does not delegate PDS authority to Bunny or either client.
Bunny cannot publish records or upload into the owner's PDS.

The Svelte client uses the same capability boundary: no AppView bearer token or
browser cookies are sent to Bunny. Status requests use the account-bound web
proxy with an explicitly allowlisted job UUID route. Its uploads use bounded
chunks and recover from the server-confirmed TUS offset.

After transcoding, an authenticated POST status request may download a processed
MP4 into a private, bounded temporary file and upload it through the existing
durable PDS effect executor, with owner/session fencing and content identity
validation. No 300 MB in-memory allocation is required. One handoff per process
and a cross-process job lease bound resource use. Interrupted requests can be
retried; ambiguous PDS writes fail the attempt rather than dispatching twice.
No OAuth credentials are stored in jobs. A ready job durably maps DID/CID to
Bunny asset ID. New publication proofs expire after 24 hours; playback mappings
do not expire while referenced.

## Development policy and operations

The development library uses Premium H.264 encoding (approved on 2026-11-05),
360p/720p/1080p, MP4 fallback,
retained originals and Frankfurt-only storage. Direct playback is enabled for
native clients. There is a rolling allowance of 20 upload authorizations per
owner per 24 hours. The Bunny client accepts MP4 and QuickTime-branded MOV
input, at most 300,000,000 bytes and ten minutes. Source MIME follows the
detected container; the PDS handoff remains a processed MP4. The legacy Bluesky
path remains MP4-only.
The 720p MP4 fallback is preferred for the PDS; smaller available inputs use their
own rendition. Existing PDS and OAuth deadlines remain in force.

`cli video-dev import` copies an already indexed, referenced PDS video after
verifying its content CID, with no PDS record change. `status` finishes its
local playback mapping after encoding. `cleanup` previews unreferenced tracked
assets older than seven days; `--apply` deletes only Bunny copies. These
operator commands refuse production. `--apply` queues the same durable worker
used for automatic post-reference cleanup.

As of 2026-11-05, indexed post deletion or video replacement/removal atomically
queues cleanup in the existing Bunny job. Post-reference writes and deletion
claims share row locks. The worker retains copies still referenced by an indexed
post, then durably transitions through origin deletion and CDN-prefix purge.
The committing deletion claim invalidates the old upload proof; references
introduced after that point need a new import/mapping. Stages survive restarts
and retry independently with bounded exponential backoff and failure reporting.
Verification reserves a two-minute preparation window, and unresolved durable
PDS create/update plans defer cleanup until publication settles.
Abandoned jobs older than seven days and terminal-owner copies also converge.
Completed rows retain allocation accounting for 24 hours. The worker never
deletes PDS blobs or records. CDN purge uses a separate server-only Core API key,
required when selecting Bunny in production; Stream keys remain library-scoped.

Bunny asset URLs are public capabilities for public media, as with the existing
video provider. Hiding a post in AppView does not itself revoke a previously
known CDN URL unless its indexed media reference is removed. Production rollout
still requires reviewed moderation policy, production credentials and
large-upload timing acceptance, alongside normal device and 24–48-hour playback
tests. Remote deletion/purge cannot erase copies already downloaded by clients.

## Consequences

The AppView playback builder performs a bounded database lookup for the media
mapping. Unimported videos retain their previous provider URLs. Images remain
on their existing delivery path. A future batch lookup can remove per-video
queries at higher feed volume. Backups must preserve the media mapping; the
PDS MP4 remains the portable recovery source.

## Hosted web playback trial

Bunny-backed read views may expose an optional `embedUrl` alongside their direct
HLS URLs. The Svelte client uses the current hosted Bunny Player for these videos;
Flutter continues to consume direct HLS. Existing PDS caption tracks select the
standard web player until caption synchronization is implemented. The embed is
public playback metadata, not upload authority. The web CSP permits only Bunny's
current player iframe origin, and its scoped control bridge checks origin and
source window and releases listeners on teardown. The trial does not alter PDS
records, production configuration, or the default provider.
