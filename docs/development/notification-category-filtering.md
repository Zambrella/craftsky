# Notification category filtering

`GET /v1/notifications` accepts an optional `category` query parameter alongside
`limit` and `cursor`. Omit it (or send an empty value) for all categories.

Accepted values: `like`, `follow`, `reply`, `mention`, `quote`, `repost`,
`instagramMatch`, `moderation`, and `everythingElse`. The last category is
reserved and currently has no producer. The client presents the eight active
categories and All categories.

Example: `GET /v1/notifications?category=mention&limit=20`.

Responses retain the existing `{items, cursor?}` shape. Filtering runs before
pagination and retains the existing visibility rules and reverse chronological
ordering. Cursors belong to their category; using one with another category
returns HTTP 400 `invalid_cursor`. Unknown category values return HTTP 400
`invalid_category`, using the standard `{error, message, requestId}` envelope.

The Flutter AppBar selector reuses the shared sort menu button. It opens a
scrollable bottom sheet on compact screens and an anchored menu on larger
screens. It replaces the notification settings icon in the AppBar. Selection
applies immediately; dismissal preserves the current filter. Selection survives
navigation and refresh, but resets to All when the active account changes or
the app starts again. Changing category resets the list position and pagination.
Old pending page responses cannot replace the newly selected category's list.

This filter does not change notification delivery preferences. Every item includes
an `isNew` boolean derived from its newness revision and the viewer's existing
account-wide seen marker. The flag is independent of category and pagination.

New rows have a subtle tint, a dot, and an accessible "New notification" label.
The app acknowledges the account-wide seen marker when leaving the notification
page after a successful list render, including a filtered list. Switching tabs,
opening a notification destination, and switching accounts end a visit. Changing
category, refreshing, and loading more retain the highlights during that visit.
The badge clears after acknowledgement succeeds. Returning reloads server state;
failed loads alone do not acknowledge notifications. This uses the existing
account-wide marker rather than individual read receipts; leaving a filtered
visit also acknowledges other categories.
