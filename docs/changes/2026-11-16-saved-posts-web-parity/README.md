# Saved posts web parity

The Svelte saved-posts page and its shared save/move components now follow the
Flutter client. Flutter was the reference and was not changed in this update.

## Changes

- Plus and Business show Folders followed by Unfiled posts; Standard retains the
  flat Saved posts list. The extra web All saved/Unfiled tabs are removed.
- Folder rows use the native folder/chevron icons and spacing. Section headings
  stick below the shared app bar. Folder pages put the labelled Newest/Oldest
  control beside the folder menu, without a second body heading. Sort options
  use the native descriptions and semantic icons with a trailing checkmark.
- Saved rows reuse PostSummary with a saved-only navigation/media variant,
  twelve-pixel content inset, saved-date footer, full date tooltip, action menu
  and divider. The saved summary opens the post, including its author header;
  replies open their root thread with the reply focused. Image previews take
  precedence over video thumbnails in this variant.
- Saved menus match Copy link, Share link, Send via chat, Move and destructive
  Unsave. Protected entries retain their saved date and private Move/Unsave
  controls without hydrating or sharing protected content.
- New folder, Rename folder and Delete folder dialogs use mobile wording and
  actions. Deletion explicitly offers Keep saved posts or Delete saved posts.
  Folder names share trim, 100-Unicode-scalar and slash/control validation with
  Flutter/AppView. Failed local validation stays at the input.
- The save/move dialog opens immediately while folders load, keeps No folder
  usable if folder loading fails, provides inline retry, and refreshes after Plus
  unlock. Standard accounts do not request folders. The overview also updates
  its scope after a confirmed subscription change without page navigation.
- Folder loading failure does not prevent saved posts from loading. Empty states
  distinguish an empty folder from an empty collection, and hide an empty
  Unfiled section when folders exist.

Existing AppView endpoints, private storage ownership, opaque cursors, account
fencing and subscription paywall gates are retained. There is no API, database,
lexicon or provider-configuration change.

## Verification

- `npm run check`: zero errors and warnings.
- `npm run lint`: passed.
- `npm test`: 612 tests across 81 files passed.
- `npm run build`: passed (existing bundle-size notice).
- Authenticated production-preview browser run covering saved parity,
  subscription feature gates, carousel/shared summaries and social-detail
  parity: 93 passed, one disabled-release case skipped. That disabled-release
  case passed separately with `WEB_CLIENT_SUBSCRIPTIONS_ENABLED=false`.
- After the final dialog-button polish, all ten saved-specific cases and the
  shared bookmark save/cancel case passed again (11 total).
- Ten new saved-specific browser cases cover paid/free scope and access changes,
  sorting, rename, both delete choices, Unicode validation, folder-load failure,
  protected saves, reply focus and compact overflow. Existing tests were updated
  for the native labels.
- Compared the running iPhone simulator Saved posts screen with the local web
  client at desktop and 402-pixel widths. The real local test account completed
  inline folder creation, moving a saved post, reopening its folder, renaming the
  folder and deleting it with Keep saved posts. The temporary folder was removed
  and the saved post returned to Unfiled. The browser viewport was restored.

Preview snapshots intermittently timed out during the rename check. The
alternate computer-use browser host was unavailable; Preview DOM controls
completed the real rename/delete flow and confirmed the final state. This is
local validation, not a production deployment or physical-device purchase test.
