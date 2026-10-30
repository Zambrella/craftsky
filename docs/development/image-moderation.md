# Image moderation in local development

Local AppView environments default to `IMAGE_SAFETY_SCANNER_MODE=dev-auto-approve`.
Indexing marks image references `clear` immediately, without downloading blobs,
creating scan jobs, or running the image-safety worker. This covers post images,
external-link thumbnails, profile avatars and banners, and business images.
Record validation, moderation decisions, and other visibility checks still apply.
Upload validation and image preparation for new uploads also still run.

These approvals use the reserved scanner identity `development-auto-approve`,
policy `development-auto-approve-v1`, and corpus `not-scanned`. They are local
visibility decisions, not content-safety evidence. Production refuses this mode
and keeps its existing manual moderation default. Do not override the scanner,
policy, or corpus identity while using auto-approval.

## Existing worktrees

Rebuild and restart this checkout's AppView, then requeue unresolved images:

```sh
./scripts/compose-dev up -d --build appview
./scripts/compose-dev exec appview /app/cli --env dev image-safety dev-catch-up
```

The command requeues image-blocked projections (including profiles) and removes
obsolete unresolved scan jobs from the local database. Normal indexing then
creates approvals under the dev identity. It preserves previous scan verdicts,
including matches, and does not unblock work waiting on membership or other
unrelated dependencies. Run it again safely if needed. The command refuses
production and requires auto-approval mode.

## Testing the full image-safety flow

Set this in the checkout's optional `.env.local` file:

```dotenv
IMAGE_SAFETY_SCANNER_MODE=stub
```

Then recreate AppView with `./scripts/compose-dev up -d --build appview`.
The stub fetches blobs and validates image format before clearing them. Its scan
identity differs from auto-approval, so newly indexed or replayed records require
that validation. Switching modes does not automatically replay already completed
records; use a fresh local stack for a complete moderation/backfill exercise.
Remove the override and recreate AppView to restore fast dev indexing.
