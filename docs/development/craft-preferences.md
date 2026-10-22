# Craft preference order

The existing `social.craftsky.actor.profile` record's `crafts` array stores
selected crafts in preference order. No additional ordering or default fields
are required. The first craft supported by the current client is the default
primary craft for a new project composer. Empty or unsupported-only preferences
leave the composer unselected.

Profile editing and onboarding display selected crafts first, in stored order.
The first selected craft is marked with a star and “Default”. Dragging or the
row's Move up / Move down menu changes this order. Selecting a craft appends it;
deselecting the first craft makes the next craft the default. An order-only edit
is a change that must be saved. Unknown and legacy craft IDs remain preserved.
Identity-only onboarding saves retain the original array verbatim.

Project browsing tabs, composer craft options, and craft shortcuts use supported
selected crafts in profile order, deduplicated. Tabs and composer options append
unselected crafts in catalog order. Changing preference order preserves an
explicitly selected browsing tab by craft identity.

Defaults apply only to new projects, including when profile loading finishes
after the composer opens. They do not replace a user's selection or selections
restored from a local draft or scheduled project. A prefilled default is part of
the pristine composer baseline and does not itself create unsaved work.

AppView profile writes, PostgreSQL projection, and profile responses preserve
array order. The record shape and HTTP API are unchanged. This convention applies
to profile `crafts`, not the separate `feed.post.crafts` metadata array.
