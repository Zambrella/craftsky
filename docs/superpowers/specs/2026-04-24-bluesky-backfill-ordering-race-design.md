# Bluesky profile backfill on Craftsky membership — design

Date: 2026-04-24
Status: approved

## 1. Problem

`bluesky_profiles` stays empty for a freshly onboarded Craftsky member
even though the user's PDS has an `app.bsky.actor.profile` record and
Tap is dispatching events for it.

The cause is an ordering race during Tap's backfill pass:

1. User OAuths with Craftsky. `InitializeProfile` writes
   `social.craftsky.actor.profile` to the user's PDS.
2. Tap discovers the repo via `TAP_SIGNAL_COLLECTION=social.craftsky.actor.profile`
   and backfills it by walking the repo's MST.
3. MST iteration is key-sorted. `app.bsky.actor.profile` sorts before
   `social.craftsky.actor.profile`, so Tap emits the Bluesky profile
   event first.
4. `BlueskyProfile.Handle` runs, checks the membership gate (`SELECT
   EXISTS FROM craftsky_profiles WHERE did = ?`), sees no row yet, and
   **silently drops the event**.
5. `social.craftsky.actor.profile` emits next and `CraftskyProfile.Handle`
   writes the membership row. Too late — the Bluesky event has already
   been discarded.

Because dropped events aren't retried, the row never lands unless the
user happens to edit their Bluesky profile later on some other client.

## 2. Goal

When `CraftskyProfile.Handle` writes a genuinely new row, eagerly
populate `bluesky_profiles` for that DID by fetching
`app.bsky.actor.profile` from the user's PDS and feeding the result
back through the existing `BlueskyProfile` indexer.

Non-goals:

- Retries/backoff on failed backfill. Acceptable to lose the backfill
  attempt; a subsequent firehose event (or a future reconciliation job)
  can fill the gap.
- Explicit `/repos/add` and `/repos/remove` calls against Tap.
  `TAP_SIGNAL_COLLECTION` plus this fix covers the timing race; explicit
  enrollment isn't required until we have account-deletion semantics
  and need symmetric teardown.
- Stricter lexicon validation of either record.

## 3. Approach

### 3.1 Synchronous, in-process backfill

When `CraftskyProfile.Handle` detects it has *just created* a row (not
updated, not replayed), it calls a `BlueskyBackfiller.Backfill(ctx,
did)` after its upsert commits. The backfiller:

1. Fetches `app.bsky.actor.profile` / `self` from the user's PDS via
   an **anonymous** read-only client (`com.atproto.repo.getRecord` is
   public).
2. Synthesises a `tap.Event{Action: "create", …}` from the fetched
   record + CID.
3. Calls `BlueskyProfile.Handle` with the synthesised event. The
   membership gate passes because we just committed the craftsky row.

This reuses every line of `BlueskyProfile.Handle`'s parse+upsert logic.
No duplicated JSON field decoding, no second SQL statement.

### 3.2 Synchronous, not async

The backfill runs inline with the `CraftskyProfile.Handle` call. Tap
acks only after the handler returns, so a slow PDS does block Tap's
pipeline — we mitigate with a tight HTTP client timeout (5s, well under
`TAP_ACK_TIMEOUT=10s`). Async goroutines are rejected for this
iteration: they introduce cancellation edges, replay-overlap risk, and
process-exit races that aren't worth handling now.

### 3.3 New-row detection via `RETURNING xmax = 0`

Today's upsert (`ON CONFLICT ... WHERE IS DISTINCT FROM`) returns
nothing, so we can't tell inserts from updates. We change the returning
clause to surface Postgres's `xmax` system column — on an INSERT that
produced a new tuple `xmax` is 0; on an UPDATE of a conflicting row
it's the updating transaction's xid. The Go code scans a single bool.

Replays of the same event don't trigger backfill: the row already
exists and the `WHERE ... IS DISTINCT FROM` clause skips the UPDATE, so
`RETURNING` yields no row at all — the bool stays false.

### 3.4 Read-only PDSClient

We introduce a second implementation of the existing `auth.PDSClient`
interface that:

- Resolves the PDS URL from the DID doc (shared with the authenticated
  path).
- Builds an unauthenticated `atclient.APIClient` — no DPoP, no OAuth
  session, no token refresh.
- Implements `GetRecord` normally (delegates to the existing
  `translateGetRecordError` helper for `RecordNotFound` handling).
- `PutRecord` returns an error — reads only.

The existing `IndigoPDSClient` is not changed. The new type
(`AnonymousPDSClient` or similar, name bikesheddable during
implementation) sits alongside it.

### 3.5 GetRecord signature: adding the CID

`bluesky_profiles.record_cid` is `NOT NULL`. The synthesised tap event
must carry a CID, and `getRecord` responses include one. Today
`PDSClient.GetRecord` discards it. We change the signature to return
`(cid string, err error)`:

```go
GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) (cid string, err error)
```

Affected call sites: `InitializeProfile` (ignores the CID, underscores
it), the backfiller (uses it), and the test mocks. Small blast radius.

### 3.6 Error policy

The backfill is best-effort. `CraftskyProfile.Handle` logs any
backfill error and returns nil — the craftsky row is already committed
and the firehose event is acknowledged. Specifically:

| Case | Handling |
|------|----------|
| PDS `RecordNotFound` for Bluesky profile | No-op (not every user has one). |
| PDS fetch error (timeout, 5xx, network) | Log + continue. |
| `BlueskyProfile.Handle` error on synthesised event | Log + continue. |

We do not propagate backfill failures up to Tap. The craftsky
membership row must persist even if the Bluesky side fails — otherwise
Tap retries the whole Craftsky event and we loop forever on a broken
PDS.

## 4. Components

### 4.1 `auth.PDSClient` interface change

```go
type PDSClient interface {
    GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) (cid string, err error)
    PutRecord(ctx context.Context, repo syntax.DID, collection, rkey string, record any) error
}
```

### 4.2 `auth.AnonymousPDSClient`

Read-only implementation, constructed with a DID-doc resolver and an
`*http.Client` with a short timeout. `PutRecord` returns
`errors.New("pds: read-only client")`.

### 4.3 `index.BlueskyBackfiller` interface

```go
type BlueskyBackfiller interface {
    Backfill(ctx context.Context, did syntax.DID) error
}
```

Narrow seam so `CraftskyProfile` is testable with a fake.

### 4.4 `index.blueskyBackfiller` concrete impl

Struct with:

- `reader auth.PDSClient` — the anonymous client.
- `indexer *BlueskyProfile` — the same instance wired in `deps.go`.

`Backfill` fetches, synthesises, and dispatches as described in §3.1.
`ErrRecordNotFound` returns nil. Other errors return wrapped.

### 4.5 `index.CraftskyProfile` changes

- Constructor gains a `BlueskyBackfiller` parameter.
- The create/update branch changes to:

  ```sql
  INSERT INTO craftsky_profiles (did, crafts, record_cid)
  VALUES ($1, $2, $3)
  ON CONFLICT (did) DO UPDATE SET
      crafts = EXCLUDED.crafts,
      record_cid = EXCLUDED.record_cid,
      indexed_at = now()
  WHERE craftsky_profiles.record_cid IS DISTINCT FROM EXCLUDED.record_cid
  RETURNING xmax = 0 AS created;
  ```

- `QueryRow` the result; if no row (replay) or `created == false`
  (update), skip backfill. If `created == true`, call
  `backfiller.Backfill`, log any error, return nil.

### 4.6 `deps.go` wiring

- Build an anonymous `auth.PDSClient` (no OAuth session needed).
- Build a `BlueskyProfile` instance as today.
- Build a `blueskyBackfiller{reader, indexer}`.
- Pass the backfiller into `NewCraftskyProfile`.

## 5. Data flow

```
Tap event: social.craftsky.actor.profile/self create
  → CraftskyProfile.Handle
     → INSERT ... RETURNING xmax=0 → created=true
     → backfiller.Backfill(ctx, did)
        → anonymous PDSClient.GetRecord(app.bsky.actor.profile/self)
        → synthesise tap.Event{create, cid, record}
        → BlueskyProfile.Handle(ev)
           → membership gate: craftsky_profiles row exists → pass
           → UPSERT bluesky_profiles
```

Tap sends a second, real firehose event for the Bluesky record later
in the same backfill stream. That event hits `BlueskyProfile.Handle`,
passes the gate (row now exists), and upserts again. The upsert is
idempotent on `record_cid` so it's a no-op — same CID, `WHERE IS
DISTINCT FROM` skips the write.

## 6. Testing

Unit tests in `appview/internal/index`:

- `CraftskyProfile_Handle_NewRow_CallsBackfill` — fake backfiller is
  invoked exactly once on a first-seen DID.
- `CraftskyProfile_Handle_Replay_SkipsBackfill` — same event twice;
  backfiller invoked on the first call only.
- `CraftskyProfile_Handle_Update_SkipsBackfill` — two events with the
  same DID but different CIDs (or same DID + changed crafts); backfill
  fires on create only.
- `CraftskyProfile_Handle_BackfillError_DoesNotFail` — fake backfiller
  returns a synthetic error; handler still returns nil; craftsky row
  is still present.
- `blueskyBackfiller_Backfill_RecordPresent_CallsIndexer` — fake
  PDSClient returns a valid Bluesky record + CID; indexer sees the
  synthesised event.
- `blueskyBackfiller_Backfill_RecordNotFound_NoOp` — fake PDSClient
  returns `ErrRecordNotFound`; indexer is not called; returns nil.
- `blueskyBackfiller_Backfill_PDSError_Propagates` — fake returns
  arbitrary error; helper wraps and returns it.

Integration tests in `appview/internal/index` using `testdb.WithSchema`:

- New DID with stubbed PDS reader → both tables populated after a
  single `CraftskyProfile.Handle(create)`.

Existing `BlueskyProfile_*` tests need no changes — the gate logic is
unchanged and the new synthesised path goes through the same entry
point.

`auth.PDSClient` signature change: update `InitializeProfile` and its
tests (ignore returned CID).

## 7. Risks and mitigations

- **Slow PDS blocks Tap pipeline.** Mitigated by a 5s HTTP timeout on
  the anonymous client. If a user's PDS is chronically slow, we lose
  their Bluesky backfill; they recover on their next profile edit.
- **Anonymous reads against a PDS that requires auth for `getRecord`.**
  `com.atproto.repo.getRecord` is defined as public in the atproto
  lexicon; no known PDS requires auth. If this ever changes, the
  anonymous client returns an auth error and the backfill logs+skips.
- **Race with live firehose delivery.** Tap *may* deliver the live
  Bluesky event to `BlueskyProfile.Handle` concurrently with our
  synthesised call. Both paths do an idempotent upsert keyed on
  `record_cid`; the second write is a no-op. Safe.

## 8. Rollout

One migration-free code change. Deploy is a single binary roll. No
config flags — the behaviour change is always on. No data backfill
required; existing members whose Bluesky profile is missing recover on
their next edit or when a future reconciliation job ships.

## 9. Out of scope / follow-ups

- A background reconciliation job that finds members with no
  `bluesky_profiles` row and schedules backfill. Handy for recovering
  current stuck users without making them edit their profile.
- Migrating to explicit `/repos/add` and `/repos/remove` when account
  deletion lands.
- Promoting `BlueskyBackfiller` to a general "fetch record from PDS and
  feed through indexer" utility if we need the same pattern elsewhere
  (e.g. future post or list indexers).
