# Quote preview image selection

- Status: Accepted
- Date: 2026-10-29
- Aspect: Public record contracts

## Decision

Add optional `imageCid` (a CID-formatted string) to
`social.craftsky.feed.post#quoteEmbed`, alongside the existing `record` strong
reference. The CID identifies an image belonging to the quoted post, without
copying its blob or granting access to hidden content. Omission retains the
first-image default. Reordering the original images preserves the selection;
removing the selected image falls back to the first currently available image.
Quotes continue to hydrate the current original post under existing visibility
and relationship policies, rather than pinning historic image content.

The immediate create API accepts `embed.imageCid` only with `embed.quote`.
Validate its canonical CID syntax before publication. The existing raw embed
retains the preference through PDS publication, indexing and text edits; no
migration, collection or OAuth scope is required. During quote hydration, place
the selected image first in that quote's compact `quoteView.post.images` list.
Keep the remaining images and their metadata, and the original post's image
order, unchanged. Resolve each quote independently, even when targets are shared.

For multiple images, the share context menu's Quote row includes an unchecked
“Use current image (3 of 4)” toggle. Changing it keeps the menu open; selecting
Quote passes the chosen carousel image to the composer for publication. The
composer retains its existing text-only target preview. Hide the control when
there is only one image. Reset the toggle when reopening the menu. Ordinary
reposts retain their existing behavior.
Quotes cannot currently be saved as drafts or scheduled; their existing rules
remain in force. Editing quote text preserves the complete original embed.

## Alternatives and consequences

An image index is smaller but can point at the wrong photo after a reorder.
A device-local preference would not survive publication or federation. Copying
the image would duplicate media and detach it from the original's visibility.
The optional CID is an additive schema change; regenerate JSON/CBOR types and
the embedded schema catalog, validate consumers, and publish the reviewed
schema through the normal release process before production use.
