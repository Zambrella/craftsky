# Home feeds

Home combines Following, Extended network and craft project discovery in one
retained navigation branch. `/feed` opens Following;
`/feed?source=extended-network` opens Extended network;
`/feed?craft=knitting` opens the named craft.
Legacy `/projects` links redirect into the same branch. Tab changes replace the
current URL, so browser Back does not step through every tab selection.

`HomeFeedDefinition` gives each tab a stable ID, source, display metadata and
supported controls. IDs, rather than tab positions, identify retained scroll
controllers and sort choices. Craft preferences determine craft tab order; Following
always stays first, followed by Extended network and then crafts. The Home page
is keyed by account to reset local state when the signed-in account changes.

The shell owns the title, tabs and contextual actions. Following and Extended
network expose the existing new-post chooser on compact layouts; project tabs
expose their existing Filters FAB and sort menu. Wide layouts retain the shell's
shared New post action.
Reselecting Home scrolls and refreshes the active tab. Returning from another
section or a post retains the selected tab and its loaded content.

## Data boundaries

The following timeline continues to use `timelineProvider` and
`GET /v1/feed/timeline`. This includes the viewer's own posts and repost activity
from the viewer and followed accounts. Feed item identity and repost attribution
must be preserved; a timeline is not simply a list of distinct post URIs.

Extended network uses `GET /v1/feed/extended-network`. An author qualifies when
the viewer follows an account that follows that author. One such path is enough;
multiple paths do not duplicate a post or boost its position. The viewer and
accounts they already follow are excluded. Only original root posts qualify,
including quote posts; replies and straight repost activity are excluded.
Instagram-imported history follows the same exclusion as authored Following
items. Ordering uses the existing stable, clamped post chronology.

Both the intermediary and author must have an indexed CraftSky profile and pass
the viewer's mute/block and owner-lifecycle checks. Blocks between the
intermediary and author also invalidate that path. Posts retain the existing
moderation and content-language policy. The optional item-level `followedBy`
author identifies one eligible intermediary, selected deterministically, for the
"Followed by …" attribution. A follow is an eligibility signal, not an endorsement
of an individual post. Membership is evaluated before pagination and reflects
indexed follow changes on refresh. The source has independent loaded content,
cursor and scroll position, and never inserts the viewer's optimistic posts.
Successful follow, mute and block changes invalidate feed state in both clients;
the web client also discards offscreen history and adjacent-tab snapshots so
returning from an intermediary's profile re-evaluates membership.
Web invalidation runs at the account-bound API client for every relationship
mutation, including Notifications and Messages. Loads started before a policy
change cannot overwrite the refreshed feed or its cache; a successful fresh load
restores normal page retention. Flutter applies accepted follow intent while
indexing catches up and refreshes the network again when a profile read confirms
the indexed follow change, superseding older in-flight feed reads.

Coverage is limited to the indexed graph: Tap discovers repositories with
CraftSky profile records, rather than collecting every AT Protocol account's
follows. A path through an untracked Bluesky-only intermediary cannot qualify
until that intermediary's graph is indexed. Extending collection to the whole
network is a separate ingestion decision. No PDS record or lexicon change is
needed for this read-side feed.

The Extended network query materializes first-degree connections, groups
second-degree paths by author, then checks each distinct author's eligibility
before joining posts. Grouping avoids repeated visibility checks per shared
path. Evaluate query-plan changes against both broad overlapping networks and
sparse networks whose eligible posts are old: scanning recent posts until enough
qualify can be quick for dense feeds but expensive for sparse or empty results.
The result-page limit does not cap network membership or truncate post history.

Craft feeds continue to use `projectFeedProvider(ProjectBrowseQuery)` and
`GET /v1/projects`, with separate filters and sort choices per craft. Each source
retains its existing pagination, language filtering, visibility policy and
optimistic interaction handling. Craft-feed APIs and record schemas are unchanged.

## Future custom feeds

Add further built-in sources through feed definitions with stable IDs and
explicit capabilities. Evaluate membership and paginate on the
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
