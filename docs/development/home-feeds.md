# Home feeds

Home combines the following timeline and craft project discovery in one retained
navigation branch. `/feed` opens Following; `/feed?craft=knitting` opens the named craft.
Legacy `/projects` links redirect into the same branch. Tab changes replace the
current URL, so browser Back does not step through every tab selection.

`HomeFeedDefinition` gives each tab a stable ID, source, display metadata and
supported controls. IDs, rather than tab positions, identify retained scroll
controllers and sort choices. Craft preferences determine craft tab order; Following
always stays first. The Home page is keyed by account to reset local state when
the signed-in account changes.

The shell owns the title, tabs and contextual actions. Following exposes the existing
new-post chooser on compact layouts; project tabs expose their existing Filters
FAB and sort menu. Wide layouts retain the shell's shared New post action.
Reselecting Home scrolls and refreshes the active tab. Returning from another
section or a post retains the selected tab and its loaded content.

## Data boundaries

The following timeline continues to use `timelineProvider` and
`GET /v1/feed/timeline`. This includes the viewer's own posts and repost activity
from the viewer and followed accounts. Feed item identity and repost attribution
must be preserved; a timeline is not simply a list of distinct post URIs.

Craft feeds continue to use `projectFeedProvider(ProjectBrowseQuery)` and
`GET /v1/projects`, with separate filters and sort choices per craft. Each source
retains its existing pagination, language filtering, visibility policy and
optimistic interaction handling. The merge does not change API or record schemas.

## Future custom feeds

Add further built-in sources, such as Mutuals, through feed definitions with
stable IDs and explicit capabilities. Evaluate membership and paginate on the
server: filtering a downloaded following timeline would produce incomplete pages
and incorrect cursors. Preserve chronology by default.

User-saved feeds should store validated, versioned query definitions rather than
result snapshots. Separate a definition (what to show) from user preferences
(which feeds are pinned and their order) and runtime state (rows, cursor, offset).
Include account identity, definition identity/revision and effective query in
cache identity, and discard pagination when the definition changes.

Private saved definitions and pinned order belong in AppView private storage for
cross-device sync. Publishing or sharing a definition requires an explicit
product decision and, for PDS records, the normal lexicon and ADR process. Export
can support portability without making saved preferences public by default.

A feed management surface can eventually choose and order pinned feeds without
making the tab strip an unlimited directory. This initial merge does not add a
feed editor, saved-feed persistence or a general query engine.
