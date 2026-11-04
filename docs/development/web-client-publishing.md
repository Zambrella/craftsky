# Web client publishing

The composer retains its originating in-app URL and closes back to that page after
an accepted creation. A directly opened composer falls back to its normal parent
(feed, draft list, schedule list or reply thread). It does not navigate to the new
post's detail page. Existing edit navigation remains unchanged.

`Publications` is scoped to the authenticated shell/account. Only a validated,
accepted AppView post response is inserted. AppView provides that response after
the PDS write is confirmed; no direct PDS credentials or new backend endpoints are
used. Uncertain writes retain their exact durable request and operation key and
never create a visible successful post prematurely.

Confirmed records bridge delayed list indexing in Home, qualifying primary-craft
feeds, profile lists and comment/reply threads. URI deduplication prevents a second
row when indexing catches up. Authoritative protected rows take precedence; later
record versions reconcile the local copy. Local insertion expires after five
minutes and is bounded to fifty records. Confirmed deletion removes a pending
local row. Reply/quote counters update for the source post without double-counting
a repeated idempotent acknowledgement. None of this state is persisted or shared
between accounts.

The composer preloads the official, pinned `@rive-app/canvas` runtime and canonical
`app/assets/animations/publishing.riv`. Vite serves both the animation and WASM from
the web client's own origin; remote WASM fallback and Rive asset CDN loading are
disabled. CSP permits `wasm-unsafe-eval` for the runtime while retaining the existing
script source restrictions. No general JavaScript `unsafe-eval` is enabled.

The blocking publishing dialog uses the native app's 48px indicator, twenty-pixel
spacing, status copy and 94-percent surface barrier. It covers submission and
media preparation, announces progress, prevents dismissal during submission, and
closes on completion or failure. Video stages retain their progress indicator.
The animation binds its thread colour to the current theme primary. Reduced motion
holds the readable 0.2-second pose; animation/runtime failure retains a stitched
fallback and does not prevent publishing. Playback resources and listeners are
released when the overlay/composer is disposed.

Validation: real-asset browser tests inspect rendered canvas pixels, theme colour,
normal/reduced-motion frame changes, stale feed reads, origin return, comments,
uncertain outcomes, fallback loading and destructive dialog colours. Unit tests
cover reconciliation, privacy boundaries, deduplication, expiry, ownership, craft
eligibility and comment counters.

## Like feedback

Post like buttons reuse the canonical `app/assets/animations/like.riv`. The public
animation bytes and local WASM runtime are shared through `web-client/src/lib/rive.ts`;
each playback owns and releases its Rive controller.

Feedback starts immediately on the initiating card, independently of network
completion. It lasts 1.4 seconds, or holds the completed 0.65-second pose for
400 milliseconds with reduced motion. Unlike, failure, identity replacement and
disposal cancel it. Existing mutation keys, retry controls, counters and button
semantics retain their established behavior.

The 160px decorative heart sits inside the visible intersection of the card and
its ancestor clips/viewport. It scales down in a narrow visible area and follows
scroll/resize without restarting playback. It ignores pointers and contributes no
accessibility announcement. Missing, late or malformed animation assets use the
static red heart without delaying or preventing the like; late readiness does not
replace a fallback halfway through playback.
