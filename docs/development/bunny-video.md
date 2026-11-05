# Bunny video development and manual acceptance

Bunny is configured only for this checkout. No production resources or deployment
configuration were changed. See [ADR 024](../../adr/024-bunny-video-development-integration.md).

## Run locally

The ignored `.env.local` selects `VIDEO_PROVIDER=bunny` and contains the development
library credentials. Do not print that file or copy keys into Dart defines.
`BUNNY_CDN_PURGE_API_KEY` supplies the separate Bunny account API credential for
CDN purges. It is server-only; the Stream library key cannot authorize purges.
Production Bunny configuration requires it. Without it in development, origin
deletion succeeds but cache purge remains queued and retries until configured.
`app/config/local.env` and `local-android.env` enable video composition locally.

From the repository root:

```bash
just dev-d
just app-run-chrome
# Alternatively:
just app-run-ios
just app-run-android
```

These recipes select this worktree's AppView port automatically. This checkout's
AppView currently listens at http://127.0.0.1:18621. Use a real PDS OAuth login;
the development DID bypass cannot publish blobs into a PDS.

## Manual checks

1. Sign in and publish a short video in an ordinary post. Confirm uploading,
   processing and publishing complete. The development library now uses paid
   Premium H.264 encoding. The first server completion request also transfers an
   MP4 to the PDS.
2. Repeat with a project post. Check aspect ratio, alt text, poster, audio and
   seeking. Check feed, profile, thread and quote previews.
3. Reload/restart the app. The same post should play from Bunny. Restart AppView
   and repeat to exercise the persistent DID/CID mapping.
4. Cancel during upload and processing; try an interrupted connection and an
   account switch. No stale attempt should publish for another account.
5. Try a larger video and the supported duration/size limits. The external upload
   is streamed and TUS resumes within the active attempt; reloading starts a new
   attempt. PDS handoff remains subject to existing server/PDS deadlines. Test
   this on the real target network before increasing limits or releasing.
6. Check iOS, Android and web independently. A successful host test is not device
   playback proof.
7. Revisit the same published post after 24 and 48 hours. Confirm the MP4 remains
   referenced on the PDS and actual HLS segments still decode.
8. Delete the post and wait for Tap to remove its index reference. Confirm the
   cleanup row advances through `deleting`, `purging`, then `deleted`, and the
   old playlist, poster and MP4 URLs stop serving. A recent publication reserve
   or unresolved PDS write can defer the deletion claim.

The library uses Premium H.264 encoding (enabled with maintainer approval on
2026-11-05), 360p/720p/1080p, MP4 fallback and retained
originals, stored in Frankfurt without extra replicas. Native direct playback is
allowed. New uploads have a rolling 20-authorizations-per-owner daily allowance.
The canonical PDS MP4 prefers Bunny's 720p fallback, falling back to available
resolutions for smaller inputs. It is limited to 300 MB and ten minutes.

The Bunny client upload path accepts MP4 and QuickTime-branded MOV sources,
including MOV files containing H.264 or HEVC video. Selection checks the file
header instead of trusting the extension or a missing picker MIME value. TUS
metadata carries the detected source MIME (`video/mp4` or `video/quicktime`).
Source files retain the 300,000,000-byte and ten-minute limits; the canonical
processed PDS blob and playback outputs remain MP4/HLS. The legacy Bluesky
upload path remains MP4-only. Test MOV selection, poster, rotation and upload
on iOS, Android and web separately before production rollout.

## Bunny dashboard organisation

New uploads and PDS imports are placed in a collection named with the owner's
full DID. Collections are created on demand and reused by exact name; a
PostgreSQL advisory lock serializes first allocations across service instances.
Collections are dashboard organisation only: the existing owner/job/CID mapping
in PostgreSQL remains authoritative for ownership, playback and cleanup.

Video titles use `YYYY-MM-DD HH:MM UTC · <first eight job-ID characters>`.
The upload authorization includes camelCase `title` and `collectionId` fields;
Flutter forwards these server-selected values in TUS upload metadata so the
transfer cannot reset the title to the old generic name or lose its collection.
Custom metadata contains `craftsky:uploadJobId`, `craftsky:source` (`upload` or
`pds-import`), and `craftsky:pdsBlobCid` once the canonical PDS blob is known.
For imports that CID is already known; for new uploads it is attached after the
durable PDS handoff. Metadata updates use the tracked video ID and can retry
without changing the PDS effect's operation identity. No post/draft text,
original filename, credentials or single-post association is copied to Bunny.

Existing videos are not automatically renamed or moved. Empty collections are
retained for reuse. Do not delete a collection as a way to tidy the dashboard:
Bunny collection deletion also deletes its videos; normal cleanup deletes only
the individually tracked video assets.

During manual acceptance, check the DID collection, unique title and the three
technical metadata properties after publication. Repeat with a second upload
for the same DID to confirm that its collection is reused.

## Import an existing broken video

The source post must already be indexed locally. Obtain its DID and video blob
CID from that public record, then run:

```bash
./scripts/compose-dev exec appview /app/cli video-dev import \
  --did 'did:plc:OWNER' --cid 'CANONICAL_VIDEO_CID'
```

This verifies the PDS source's CID and uploads a copy to Bunny, without changing
any PDS record. It prints a job ID. After encoding, finish the local mapping:

```bash
./scripts/compose-dev exec appview /app/cli video-dev status \
  --did 'did:plc:OWNER' --job 'JOB_UUID'
```

Repeat status while it reports `JOB_STATE_PROCESSING`. Reload the post when
completed. Unimported old videos retain their previous playback URLs.

## Automatic deletion and cleanup

Removing a post's video reference (post deletion or replacement/removal by edit)
atomically queues cleanup with the indexed record change. This also covers
deletion performed through another AT Protocol client once Tap indexes it.
After the last indexed reference disappears, the background worker claims the
asset, deletes it from Bunny Stream, and purges the CDN prefix containing its
playlists, segments, MP4 renditions, posters and previews. Other indexed posts
by the same owner referencing that video keep their Bunny copy.

The worker polls every five seconds and runs independently of the client.
References and deletion claims share database row locks. A reference that
commits before the deletion claim prevents deletion; fresh references cancel
pending work. Once deletion is claimed, the old upload proof is invalid and the
asset cannot be reused. A later external PDS post may need a new import to get
a Bunny mapping. Public PDS blobs are never deleted by this worker.
Recently verified upload proofs reserve a two-minute publication window;
unresolved durable PDS create/update commands also defer cleanup until settled.

Origin deletion and CDN purge are separate durable stages. A crash, missing
credential, provider error or rate limit leaves the unfinished stage queued.
Retries back off up to one hour, with warning logs initially and an issue after
five failures; retries continue. An already absent Stream asset counts as a
successful origin deletion. Completion means Bunny acknowledged deletion and
the synchronous prefix purge, not that already downloaded client copies vanish.
Normal provider/Tap propagation delays still apply.

Unreferenced abandoned jobs older than seven days are queued automatically.
Terminal-owner cleanup follows the same reference checks and purge stages.
Completed tombstones preserve upload quota accounting for 24 hours before being
discarded; terminal-owner tombstones can be discarded immediately.

Inspect unfinished work without printing credentials:

```sql
SELECT id, state, cleanup_requested_at, cleanup_next_attempt_at, cleanup_attempts
FROM bunny_video_jobs WHERE cleanup_requested_at IS NOT NULL AND state <> 'deleted';
```

### Development operator commands

Preview tracked, unreferenced copies older than seven days:

```bash
./scripts/compose-dev exec appview /app/cli video-dev cleanup
```

Add `--apply` to queue those copies for the same automatic delete-and-purge
worker. These operator commands refuse production. Referenced copies remain.
Owner purge retains cleanup authority until remote deletion and purge succeed.

A synthetic three-second development fixture was uploaded directly with the
Bunny CLI. It is intentionally retained for the 24–48-hour delivery check. It is
not an AppView job or PDS post, so the AppView cleanup tool does not select it.
List it with `bunny stream videos list --lib craftsky-dev`; remove it explicitly
through Bunny when the retention check is finished.

## Before production

Manual acceptance must include real PDS publication, native/browser playback,
large-file timing and the delayed retention check. Production needs a separately
reviewed library and purge credentials, moderation policy, deployment
configuration and release. Hiding a post without removing its indexed record
does not revoke its CDN URL; moderation-triggered revocation needs a separate
policy. The automatic worker handles reference removal, abandoned uploads and
terminal-owner copies. Image delivery is unchanged.

## Automated validation performed

- Full backend gate: `./scripts/appview-test full` (PostgreSQL/MinIO, race detector).
- Focused video/authentication/route checks after the final authentication fix.
- Flutter analysis of changed sources and transport tests.
- Bunny TUS resume, destination rejection and credential canary tests; existing
  video API, publication, credential lifecycle and both composer widget suites.
- Docker build, local migration refresh, healthy `/health` and `/healthz`, CLI
  cleanup preview, early refusal of production operator commands.
- Real Bunny fixture: poster, MP4 and HLS resources returned successfully;
  FFmpeg decoded all three seconds of HLS video/audio.
- Cleanup regression tests: last/shared reference, edit removal, rollback,
  concurrent reference insertion, in-flight publication, stage persistence,
  origin 404, retry backoff, missing purge credential, protected diagnostics and
  production purge configuration.
- Live development deletion on 2026-11-05: the already-deleted post's tracked
  copy completed origin deletion and CDN purge; its playlist, thumbnail and
  720p MP4 returned HTTP 404. The separate delivery-test fixture was retained.
- Organisation checks: concurrent allocations across service instances, exact
  DID matching, paginated collection lookup, UTC titles, upload/import metadata,
  PDS CID attachment and retained cleanup tracking after metadata failure;
  authorization response fields and Flutter TUS metadata forwarding.
- Live development metadata check on 2026-11-05: collection assignment, unique
  title and custom metadata round-tripped through the Bunny API. The temporary
  empty video and collection were removed; no media upload or encoding was
  triggered. The rebuilt local AppView passed `/healthz`.
- MOV regression checks cover header detection without picker MIME, exact size
  and duration boundaries, spoofed/truncated headers, resumed QuickTime TUS
  transfer through the shared composer binding and an MP4 publication proof.
  Both composer publication suites and transport security/resource bounds pass.
- Live development MOV check on 2026-11-05: one-second H.264/AAC and HEVC/AAC MOV
  inputs encoded successfully in `craftsky-dev`; their HLS decoded and their
  processed MP4s contained H.264/AAC. Both temporary Bunny videos were deleted.
  This verifies Bunny processing, not real-device picker/PDS publication.

Real PDS publication, physical devices and delayed 24–48-hour retention have not
been exercised automatically and remain the manual acceptance steps above.
