# Video Hosting Investigation

Date: 2026-09-09

## Summary

Craftsky videos appeared successfully after publication and then became
unavailable. The initial suspicion was that the user's PDS garbage-collected the
video blob despite a repository record referencing it.

Investigation of a live affected post disproved that hypothesis. The Craftsky
record contains a correct standard `app.bsky.embed.video` value, and the
authoritative MP4 remains publicly available from the user's PDS. The missing
resources are the derived HLS media segments and thumbnail hosted by Bluesky's
video infrastructure.

The confirmed failure is therefore:

> The PDS retained the correctly referenced video blob, while Bluesky's derived
> video CDN assets were removed or became unavailable.

The most likely explanation is that Bluesky's video asset lifecycle recognizes
publication through `app.bsky.feed.post` but not through Craftsky's
`social.craftsky.feed.post`. This is consistent with the observed behavior, but
cannot be confirmed because the production `video.bsky.app` backend is not
publicly available.

## Affected Example

Record:

```text
at://did:plc:jt3cxdyrhjhrkpwzdvonheax/social.craftsky.feed.post/3muwkc42rhn22
```

Record browser:

<https://pdsls.dev/at://did:plc:jt3cxdyrhjhrkpwzdvonheax/social.craftsky.feed.post/3muwkc42rhn22>

Video CID:

```text
bafkreienh4khxifbrjtljzlqpc5pt5wg65poh2u4kcq4ft666alplxuroe
```

PDS:

```text
https://puffball.us-east.host.bsky.network
```

Relevant record content:

```json
{
  "$type": "social.craftsky.feed.post",
  "text": "Different video",
  "embed": {
    "$type": "app.bsky.embed.video",
    "video": {
      "$type": "blob",
      "ref": {
        "$link": "bafkreienh4khxifbrjtljzlqpc5pt5wg65poh2u4kcq4ft666alplxuroe"
      },
      "mimeType": "video/mp4",
      "size": 5388115
    },
    "aspectRatio": {
      "width": 720,
      "height": 1280
    }
  },
  "langs": ["en"],
  "createdAt": "2026-09-07T12:50:29Z"
}
```

## Live HTTP Evidence

The following resources were checked on 2026-09-09.

| Resource | Result |
|---|---|
| Authoritative PDS MP4 | `200`, `video/mp4`, 5,388,115 bytes |
| Bluesky master HLS playlist | `200` |
| Bluesky 360p variant playlist | `200` |
| Bluesky 720p variant playlist | `200` |
| Bluesky 360p first media segment | `404` |
| Bluesky 720p first media segment | `404` |
| Bluesky thumbnail through `/watch` | `404` |
| Bluesky thumbnail through the backing CDN | `404` |

The master and variant playlists were cached on September 7, close to the record
creation time. They continued to return `200` with long cache lifetimes, but the
media segments referenced by those playlists no longer existed. This explains
why a superficial playlist check could report success while actual playback
failed.

The authoritative PDS blob remained available at:

```text
https://puffball.us-east.host.bsky.network/xrpc/com.atproto.sync.getBlob?did=did%3Aplc%3Ajt3cxdyrhjhrkpwzdvonheax&cid=bafkreienh4khxifbrjtljzlqpc5pt5wg65poh2u4kcq4ft666alplxuroe
```

## Lexicon and Record Validation

Craftsky's post Lexicon includes `app.bsky.embed.video` in the open `embed`
union at `lexicon/social/craftsky/feed/post.json`.

AppView constructs the standard video embed in
`appview/internal/api/post_create.go`:

```json
{
  "$type": "app.bsky.embed.video",
  "video": {
    "$type": "blob",
    "ref": {"$link": "<cid>"},
    "mimeType": "video/mp4",
    "size": 123
  }
}
```

This is the correct structural Lexicon blob representation. It is not a plain
CID string or a `cid-link`.

The reference PDS implementation recursively enumerates structural blob objects
at any nesting depth. During record creation, it verifies the blob exists,
records the relationship between record URI and blob CID, and promotes the blob
from temporary to permanent storage. A successful Craftsky post creation should
therefore retain the blob normally.

The pinned Indigo schema used for generated Go types historically described a
100 MB maximum while the current public `app.bsky.embed.video` schema permits
300 MB. Craftsky enforces the current 300 MB policy at its boundaries. This
version difference can affect validation compatibility for large files, but it
does not explain a successfully published blob disappearing later.

## Bluesky Video Workflow

The documented Bluesky preprocessing flow is:

1. The client obtains a PDS-signed service JWT with the user's PDS as audience
   and `com.atproto.repo.uploadBlob` as `lxm`.
2. The client uploads the source video to `video.bsky.app`.
3. Bluesky processes and optimizes the video.
4. Bluesky uses the delegated JWT to upload an optimized MP4 to the user's PDS.
5. `app.bsky.video.getJobStatus` returns the PDS `BlobRef`.
6. The client includes that complete `BlobRef` in a repository record.
7. Bluesky separately serves derived HLS renditions and a thumbnail from its
   application infrastructure.

The PDS MP4 and the Bluesky HLS asset tree have separate lifecycles. AT Protocol
guarantees the repository blob relationship; it does not guarantee that a
particular AppView or CDN will retain derived media.

Public implementation is available for the AT Protocol PDS, video Lexicons,
and official Bluesky client. No public repository was found for the production
Bluesky video backend. Official-client references suggest the backend is named
`tango`, but that repository is private or unavailable.

## Existing Publication Race

Before a record references a newly uploaded blob, the PDS treats the blob as
temporary and may garbage-collect it. This can happen when publication is
cancelled or interrupted after processing.

Craftsky already recognizes this publication-time failure:

- AppView maps PDS `BlobNotFound` to `video_blob_missing`.
- Flutter makes one bounded fresh-upload recovery attempt.
- The same source can still fail because Bluesky deduplication may return a CID
  for a previously processed PDS blob that has already been collected.

This race explains videos that cannot be published. It does not explain the
investigated post because its PDS record was successfully created and its PDS
blob remains present.

## Playback URL Findings

Craftsky's playlist URL already matches the current Bluesky public interface:

```text
https://video.bsky.app/watch/{did}/{cid}/playlist.m3u8
```

Craftsky previously generated thumbnail URLs directly against the backing CDN:

```text
https://video.cdn.bsky.app/hls/{did}/{cid}/thumbnail.jpg
```

The current Bluesky web client instead uses:

```text
https://video.bsky.app/watch/{did}/{cid}/thumbnail.jpg
```

For a healthy Bluesky video, the `/watch` thumbnail endpoint returned `302` to
the same direct CDN URL, and the CDN returned `200`. The direct URL was therefore
functional but bypassed Bluesky's stable public interface.

Craftsky's default thumbnail template was changed to the `/watch` form in:

- `appview/internal/video/playback.go`
- `appview/internal/app/config.go`
- Associated configuration and playback tests
- `appview/environments/dev.env`
- `appview/environments/prod.env.example`

The production deployment must also update `VIDEO_THUMBNAIL_URL_TEMPLATE` if it
sets an external override.

This URL correction does not restore the affected example. Its `/watch`
thumbnail redirects to the same missing CDN object.

The `blob:https://bsky.app/...` source seen on a Bluesky web `<video>` element is
a browser-local MediaSource URL created by the HLS player. It is not a persistent
or externally usable video URL.

## Why Bluesky Images Continue to Work

Bluesky's image service is a pull-through transformation cache. Given a DID,
blob CID, and image preset, it can resolve the PDS, fetch the authoritative
image, verify it, resize it, and cache the result. It does not need to know which
record references the image.

If an image derivative is evicted, it can be regenerated cheaply from the PDS.
The public Bluesky image-server implementation performs this fetch and
transformation on cache misses.

Video delivery differs because HLS output is precomputed. Transcoding can
produce multiple renditions, playlists, segments, audio, thumbnails, and
moderation results from inputs up to 300 MB and ten minutes. Regenerating this
asset tree synchronously on every cache miss would be expensive and slow.

Bluesky therefore retains preprocessed video assets separately. Its cleanup
system likely expects to observe publication through Bluesky's own post
collection. A valid Craftsky record may not satisfy that application-specific
retention rule.

Craftsky's use of Bluesky's image CDN is also not an AT Protocol guarantee. It
is simply more resilient because the image service can rehydrate derivatives
from any publicly available DID/CID pair.

## Direct PDS Playback

Direct playback of the PDS MP4 is technically feasible, particularly as a
fallback. The affected PDS returned the correct MIME type, content length, and
`Access-Control-Allow-Origin: *`.

The important limitation is byte ranges. A request containing
`Range: bytes=0-1023` returned the complete 5.4 MB file with `200`, not a partial
response with `206`. The current reference PDS `getBlob` implementation streams
the complete object and does not implement HTTP range handling.

Consequences include:

- Seeking may download the entire file again.
- Startup can be slow for large files.
- Playback has no adaptive bitrate.
- A maximum-size video can impose substantial bandwidth on the PDS and device.
- Other PDS operators are not required to provide equivalent CORS or delivery
  behavior.

AT Protocol permits fetching originals through `com.atproto.sync.getBlob`, but
recommends that applications use an independent proxy or CDN for end-user media
delivery.

Craftsky's architecture also requires Flutter reads to come through AppView,
not directly from a PDS. A compliant fallback would therefore be an
AppView-mediated streaming endpoint or an app-owned cache/CDN. A basic streaming
proxy can restore playback but cannot provide efficient range requests when the
upstream PDS ignores ranges. Efficient seeking requires caching the complete
object in range-capable storage.

## Craftsky-Owned Video Service

A fully controlled service would require these components:

| Component | Responsibility |
|---|---|
| AppView control plane | Authentication, jobs, quotas, PDS upload, publication verification |
| Private source storage | Temporary resumable source uploads |
| Durable queue | Processing dispatch, retries, and crash recovery |
| Sandboxed FFmpeg workers | Validation, optimization, HLS packaging, thumbnails |
| Public object storage and CDN | Immutable HLS assets and posters |
| Postgres | Jobs, DID/CID mappings, publication state, cleanup state |
| Tap reconciler | Authoritative record-reference reconciliation |

Recommended asset paths would be stable and content-addressed by repository
identity:

```text
https://video.craftsky.social/hls/{did}/{cid}/playlist.m3u8
https://video.craftsky.social/hls/{did}/{cid}/720p/video.m3u8
https://video.craftsky.social/hls/{did}/{cid}/720p/segment-00001.ts
https://video.craftsky.social/hls/{did}/{cid}/thumbnail.jpg
```

The source and staging paths should remain private and expire automatically.
Published derivatives should remain until no current Craftsky record references
the DID/CID, followed by a deletion grace period. The service must never delete
the authoritative PDS blob.

The preferred custom flow would upload the source directly to a signed private
object-storage URL, process it asynchronously, upload the optimized MP4 to the
PDS through AppView's server-side OAuth session, and then return a verified
BlobRef for normal post creation.

This requires an amendment to ADR 012 because the current decision narrowly
permits direct upload only to Bluesky's video service. It does not require a
record Lexicon change if Craftsky continues to use `app.bsky.embed.video`.

## Managed Cloudflare Option

Cloudflare Stream can replace the transcoding workers, HLS storage, thumbnail
generation, and CDN. Craftsky would still need the AT Protocol control plane.

Cloudflare Stream currently provides:

- One-time direct creator upload URLs
- Resumable TUS uploads for files over 200 MB
- Adaptive HLS and DASH output
- Processing status and signed webhooks
- Hosted thumbnails
- Optional generated downloadable MP4 files
- API-controlled deletion and access policy

Cloudflare Stream does not understand DIDs, PDS service authentication,
BlobRefs, Craftsky records, or record-reference lifecycle.

### Stream-First Flow

Flutter could upload directly to Cloudflare Stream. After processing, AppView
would request a downloadable encoded MP4, stream that file into the user's PDS,
and receive the authoritative BlobRef. Cloudflare does not make the exact input
file downloadable, but it can generate a new MP4 suitable for this purpose.

### R2 Staging Flow

The more controllable managed design is:

```text
Flutter -> signed private R2 upload
                 |
                 +-> Cloudflare Stream imports and transcodes the source
                 |
                 +-> AppView streams the source into the user's PDS
```

Benefits of R2 staging include one client upload, independent retries, retention
of the exact source until both downstream operations succeed, and no dependency
on generation of a downloadable Stream MP4. The source object can be deleted
after successful publication.

If the source is not sufficiently optimized for the PDS, the Stream-first
downloadable-MP4 flow may be preferable despite its extra processing step.

Cloudflare playback URLs are keyed by a Stream UID rather than the PDS blob CID.
AppView must durably retain this mapping:

```text
(owner DID, PDS blob CID) -> Cloudflare Stream UID
```

The mapping should also be copied into Stream metadata and protected by database
backups and reconciliation. A stable Craftsky playback endpoint can hide the
vendor UID and redirect or proxy to Cloudflare:

```text
https://video.craftsky.social/watch/{did}/{cid}/playlist.m3u8
```

The raw PDS MP4 remains the portability and disaster-recovery fallback if the
private mapping or managed provider becomes unavailable.

## Atmosphere Community Research

A targeted search of the Atmosphere Community forum found no discussion of
`video.bsky.app`, `app.bsky.embed.video`, custom-record video retention, or HLS
assets disappearing while the source PDS blob remains available. The forum does
not confirm or disprove the suspected Bluesky cleanup behavior.

The relevant discussions do support the architecture recommended here:

- The Media PDS / Service proposal separates object storage, media processing,
  metadata, and distribution. Its proposed API allows playback URLs to be
  supplied by external providers while retaining an AT Protocol control plane.
- Participants describe Bluesky's video uploader as supplemental processing
  infrastructure rather than an authoritative PDS. A Bluesky team member agreed
  that blob storage and baseline video/streaming guidance remain underspecified
  for production PDS deployments.
- The VOD discussion repeatedly recommends decoupling content metadata from
  distribution. Suggested delivery backends include Cloudflare Stream, Bunny
  Stream, Backblaze B2 with a CDN, Vimeo, PeerTube, and Streamplace.
- VidSky currently uses Bunny Stream while it works toward publishing records to
  users' PDSes. Streamplace similarly hosts its VOD data separately for now,
  while making it content-addressed and public as a basis for future cooperative
  storage.
- The storage-limits discussion favors keeping repositories small and using
  specialized storage for large files. It also notes that applications must
  handle each PDS provider's limits and capabilities rather than assuming
  unlimited media storage or high-performance delivery.
- A community Lexicon discussion quotes the AT Protocol blob specification:
  applications should proxy blobs through an independent CDN or media service
  before browser delivery, and PDSes should not perform media transcoding without
  strong sandboxing.

These discussions are architectural evidence, not evidence about Bluesky's
private retention implementation. They reinforce using the retained PDS MP4 as
the portable source of truth while making Craftsky responsible for dependable
playback delivery and derivative retention.

## Recommended Direction

The pragmatic sequence is:

1. Add an AppView-mediated raw PDS MP4 fallback for resilience and recovery of
   existing affected posts.
2. Introduce private signed source uploads and a durable video-job control plane.
3. Use Cloudflare R2 plus Cloudflare Stream for initial managed processing and
   delivery.
4. Upload or preserve an MP4 in the user's PDS and continue writing the standard
   `app.bsky.embed.video` record shape.
5. Make Tap-observed Craftsky record references authoritative for managed asset
   retention.
6. Preserve stable Craftsky playback URLs so the media provider can be replaced
   without changing records or Flutter's API contract.
7. Consider a Craftsky-owned FFmpeg pipeline later if cost, control, portability,
   or provider policy justifies replacing Cloudflare Stream.

## Open Questions

- Does Bluesky intentionally restrict durable video hosting to
  `app.bsky.feed.post`, or is custom-record cleanup a bug?
- Should the first implementation provide only PDS MP4 fallback or immediately
  replace Bluesky processing for new uploads?
- Should Cloudflare Stream receive the original source through R2 import, or
  should its generated downloadable MP4 become the PDS blob?
- What maximum duration, resolution, frame rate, and daily upload quota should
  Craftsky support?
- What publication and deletion grace periods are appropriate?
- Is a vendor-specific DID/CID-to-asset mapping acceptable if the PDS MP4 remains
  portable?
- When should Craftsky replace its analogous dependency on Bluesky's image CDN?

## References

- [AT Protocol blob specification](https://atproto.com/specs/blob)
- [Bluesky video upload tutorial](https://docs.bsky.app/docs/tutorials/video)
- [Bluesky video Lexicons and AppView views](https://github.com/bluesky-social/atproto/pull/2751)
- [Bluesky video service issues](https://github.com/bluesky-social/atproto/issues/3026)
- [AT Protocol reference implementation](https://github.com/bluesky-social/atproto)
- [Bluesky official client](https://github.com/bluesky-social/social-app)
- [Cloudflare Stream direct creator uploads](https://developers.cloudflare.com/stream/uploading-videos/direct-creator-uploads/)
- [Cloudflare Stream webhooks](https://developers.cloudflare.com/stream/manage-video-library/using-webhooks/)
- [Cloudflare Stream downloadable MP4](https://developers.cloudflare.com/stream/viewing-videos/download-videos/)
- [Cloudflare Stream upload by URL](https://developers.cloudflare.com/stream/uploading-videos/upload-via-link/)
- [Atmosphere: Media PDS / Service](https://discourse.atmosphere.community/t/media-pds-service/297)
- [Atmosphere: Video on Demand options for ATProto](https://discourse.atmosphere.community/t/video-on-demand-vod-options-for-atproto/259)
- [Atmosphere: Storage-Heavy Applications and PDS Limits](https://discourse.atmosphere.community/t/storage-heavy-applications-and-pds-limits/825)
- [Atmosphere: VidSky](https://discourse.atmosphere.community/t/vidsky-an-atproto-powered-video-platform/918)
- [Atmosphere: standalone image Lexicon discussion](https://discourse.atmosphere.community/t/proposal-a-community-lexicon-for-standalone-images/1091/28)
- `adr/012-ephemeral-video-service-token-handoff.md`
- `adr/013-standard-video-embed.md`
- `docs/changes/2026-09-03-video-posts/01-requirements.md`
