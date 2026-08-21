# AT Protocol Identity Freshness Requirements

## Context

CraftSky has two identity caches: Indigo's process-local `CacheDirectory` and
the persistent `atproto_identity_cache` search index. Handles are mutable while
DIDs are stable, so a cached handle mapping must not select the target of a
durable or security-sensitive operation.

## Requirements

- **ID-001 (Must):** Keep cached Indigo resolution for display, hydration,
  autocomplete, and other non-authoritative reads.
- **ID-002 (Must):** Provide an uncached authoritative directory using the same
  federated HTTP boundary, DNS policy, deadlines, and response limits as the
  cached directory. It must never fall back to Indigo's default transport.
- **ID-003 (Must):** Final mention resolution must always perform a fresh,
  bidirectionally verified handle lookup, then recheck membership, interaction
  policy, lifecycle state, and write the canonical mapping to the persistent
  index. Cached autocomplete remains allowed.
- **ID-004 (Must):** Account-deletion intent creation must freshly resolve the
  authenticated DID's canonical handle and bind that handle to the intent. A
  transient identity failure must be retryable and must never fall back to a
  stale Postgres row.
- **ID-005 (Must):** User-supplied handles that choose mutation targets
  (follow, unfollow, mute, unmute, block, unblock, and account reports) must use
  authoritative resolution. DID-keyed operations require no handle refresh.
- **ID-006 (Must):** OAuth identity routing must use the authoritative directory.
- **ID-007 (Must):** The persistent handle index must remain lifecycle fenced,
  case-insensitively unique, and refreshed by a bounded background processor in
  addition to login, final mention resolution, and the operator backfill.
- **ID-008 (Should):** Stale persistent rows may be used only for presentation;
  consumers must either enforce the 24-hour freshness threshold or explicitly
  be documented as stale-tolerant and rely on background refresh.
- **ID-009 (Should):** Emit bounded, non-PII observations for authoritative
  lookup result/latency, cache age, refresh results, and handle reassignment.
- **ID-010 (Must):** A supported Tap identity event must atomically commit its
  durable receipt and enqueue an immediate authoritative refresh before it can
  be acknowledged. Duplicate/redelivered events must not reset a retry delay
  or create duplicate work.
- **ID-011 (Must):** A Tap identity event must invalidate the process-local
  Indigo entries for its DID and known old/new handles without performing
  remote identity resolution in the Tap acknowledgement path. A successful
  authoritative refresh must repeat invalidation so a crash between durable
  enqueue and initial invalidation cannot leave the display cache stale.
- **ID-012 (Must):** Tap-provided handles are invalidation hints only. They must
  not be persisted as verified identity mappings until Indigo completes fresh
  bidirectional DID/handle verification. The periodic 24-hour scan remains as
  missed-event and historical repair coverage.

## Acceptance criteria

- **AC-001:** Cached and authoritative lookups use distinct directory objects,
  but the same hardened base-directory network policy.
- **AC-002:** A stale cached handle reassignment cannot affect final mention or
  any handle-targeted mutation.
- **AC-003:** Account deletion binds the freshly verified canonical handle and
  fails without changing state when identity resolution is unavailable.
- **AC-004:** OAuth uses the uncached directory while ordinary AppView display
  resolution uses the cache.
- **AC-005:** The refresh processor is bounded, skips terminal owners, updates
  successful mappings, delays failed rows, and cannot starve later candidates.
- **AC-006:** Terminal cleanup still removes persistent identity rows.
- **AC-007:** Tap identity receipt and immediate refresh scheduling share one
  transaction, ACK follows commit, and same-event redelivery is idempotent.
- **AC-008:** Tap ingestion performs no DNS/PLC/HTTPS request and invalidates
  both sides of known cached mappings; the refresh worker alone writes the
  verified result.
