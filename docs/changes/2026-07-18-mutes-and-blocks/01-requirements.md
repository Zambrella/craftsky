# Requirements: Account Mutes And Blocks

## 1. Initial Request

Fully implement account mutes and blocks across the Craftsky AppView and Flutter client. The behavior must match Bluesky: mutes are private, blocks are public, blocks use the interoperable `app.bsky.graph.block` collection, and both controls impose the same visibility and interaction restrictions as Bluesky on every equivalent Craftsky surface.

## 2. Current Codebase Findings

- Relevant files:
  - `atproto-craft-social-app-reference.md` already requires public PDS-backed blocks, server-side block filtering, optimistic client filtering during indexing lag, and private AppView-backed mutes.
  - `appview/internal/index/bluesky_follow.go`, `appview/internal/api/follow.go`, and `appview/migrations/000012_atproto_follows.up.sql` provide the nearest public graph-record pattern: write through the user's PDS, consume Tap events, and index the active relationship in Postgres.
  - `appview/internal/app/deps.go` and `appview/internal/routes/routes.go` compose indexers, stores, PDS clients, and the authenticated `/v1/*` surface.
  - `docker-compose.yml` filters Tap collections and currently includes `app.bsky.graph.follow`, but not `app.bsky.graph.block`.
  - `appview/internal/api/profile_response.go` exposes viewer-relative follow state but no mute/block state.
  - `appview/internal/api/post_store.go`, `timeline_store.go`, `search_store.go`, `profile_store.go`, `facet_store.go`, `notification_store.go`, and `notification_newness.go` are the principal viewer-aware read paths requiring policy enforcement.
  - `appview/internal/notifications/` and `appview/internal/push/` create durable notification events and push-delivery outbox rows; mute/block changes must also suppress or cancel pending delivery.
  - `app/lib/profile/widgets/profile_actions.dart` has visitor follow, share, and report actions and explicitly anticipates a blocked-user action variant.
  - `app/lib/settings/pages/settings_page.dart` already links to follower and following lists and is the natural entry point for muted and blocked account management.
  - `app/lib/feed/models/post.dart` and existing quote states support unavailable-content placeholders, but do not yet represent muted-reply or blocked-content policy.
  - `docs/changes/2026-05-25-follow-unfollow-mvp/01-requirements.md` and the implemented follow/profile paths currently allow resolvable non-Craftsky profiles to be viewed and followed. The confirmed hard Craftsky-membership boundary in this document intentionally supersedes that behavior.
- Existing patterns:
  - Public atproto graph writes go through the AppView to the caller's PDS; Flutter never holds PDS credentials.
  - Private-by-intent data is account-scoped in AppView Postgres and exposed only through authenticated endpoints.
  - `/v1/*` uses camelCase JSON, authenticated device-scoped middleware, standard error envelopes, and opaque cursor pagination.
  - Flutter uses Dio clients, repositories, Riverpod providers, localized copy, typed models, and account-aware state.
  - Read-time moderation already filters hidden/taken-down posts and accounts and shapes unavailable quote previews.
- Current behavior:
  - There is no mute or block persistence, indexer, API, relationship state, settings list, profile action, client cache, read filtering, write restriction, notification suppression, or push cancellation.
  - Existing Bluesky block records are not consumed or backfilled.
  - All otherwise-valid follows, likes, reposts, replies, quotes, and mentions are currently permitted regardless of account relationship.
  - Resolvable non-Craftsky accounts can currently be viewed and followed, while mention suggestions/resolution already require Craftsky membership.
- External behavior verified on 2026-07-18:
  - Bluesky's [user FAQ](https://bsky.social/about/blog/5-19-2023-user-faq) defines a mute as private, suppressing notifications and top-level posts while replacing muted replies with a revealable placeholder.
  - Bluesky's [block implementation guide](https://docs.bsky.app/blog/block-implementation) defines blocks as symmetric mutual mutes plus interaction restrictions. It explicitly suppresses feeds, reply threads, replies, quotes, embeds, mentions, follows, likes, and notifications while leaving source records and follow records intact.
  - The canonical [`app.bsky.graph.block`](https://github.com/bluesky-social/atproto/blob/main/lexicons/app/bsky/graph/block.json) schema is a public TID-keyed record containing the subject DID and creation time.
  - The canonical [`app.bsky.graph.muteActor`](https://github.com/bluesky-social/atproto/blob/main/lexicons/app/bsky/graph/muteActor.json) API describes actor mutes as private authenticated state, not repository records.
  - Bluesky's current profile viewer state distinguishes `muted`, `blockedBy`, and `blocking`, and its client leaves block state visible on the profile while hiding profile details and interaction controls.
  - Bluesky's current AppView tests keep underlying follow records but suppress viewer follow state and follower/following list entries across a block; omit blocked accounts from ordinary search while allowing exact-handle lookup; hide block-violating replies/embeds from third-party viewers; and preserve record-based aggregate contributions for unrelated viewers while hiding metrics across the blocked pair.
  - Bluesky filters mute/block notifications dynamically at read time, so retained notification history can become visible again after unmute/unblock; already-sent pushes cannot be retracted.
- Constraints discovered:
  - No local Craftsky lexicon change is required or permitted for this feature; blocks reuse `app.bsky.graph.block` and mutes are private relational data.
  - Enforcement must be viewer-relative and account-scoped, including when multiple accounts are signed in on one device.
  - A successful block must take effect before Tap/firehose convergence; safety cannot depend on the normal indexing delay.
  - Public AT Protocol content remains technically accessible outside an authenticated, compliant Craftsky/Bluesky view. Product behavior adds friction and suppresses delivery; it cannot make public repository data secret.
- Test/build commands discovered:
  - Full stack: `just dev`.
  - AppView tests: `just test` from the repository root, with focused Go tests available from `appview/` when the compose Postgres is running.
  - Flutter tests: `flutter test <paths>` from `app/`.
  - Flutter analysis: `flutter analyze` from `app/`.

## 3. Clarifying Questions And Decisions

### Q1: What defines “exactly the same” restrictions?

Answer: The user's request names Bluesky as the behavioral reference.

Decision / implication: Bluesky's documented actor-mute and block contract, verified against its canonical lexicons and current open-source client, is normative for Craftsky-equivalent surfaces. Where Craftsky lacks a Bluesky feature, that feature is a non-goal rather than a reason to invent a substitute.

### Q2: Where is each relationship stored?

Answer: Mutes are private; blocks are public and can use the same collection as Bluesky.

Decision / implication: Craftsky mutes are stored only in account-scoped AppView Postgres. Blocks are authored as `app.bsky.graph.block` records on the blocker's PDS and indexed by the AppView. Existing compatible block records must be honored.

### Q3: Does a block delete existing content, interactions, or follows?

Answer: Match Bluesky.

Decision / implication: No. A block changes visibility, delivery, and allowed future interaction without deleting either account's records. Old content can become visible again after unblock; existing follow records remain but cannot deliver content or enable interaction while blocked.

### Q4: Is a clarification required before requirements can be written?

Answer: No blocking clarification is required.

Decision / implication: Individual account mutes and blocks, their lists, and all equivalent current Craftsky surfaces are in scope. Broader mute/list/DM controls are explicitly excluded.

### Q5: Are non-Craftsky accounts visible or actionable in Craftsky?

Answer: No. A Craftsky profile must exist; otherwise return `404 profile_not_found`.

Decision / implication: Craftsky membership is a hard eligibility boundary for every user-facing account surface. Non-members are absent from profile reads, search, mention suggestions, follow/follower/mutual lists, mute/block lists, reports, and relationship mutations. This intentionally supersedes the earlier ability to view and follow non-Craftsky accounts.

### Q6: What happens to relationships when the subject leaves or later rejoins Craftsky?

Answer: Keep the underlying relationship hidden and reactivate it if the same DID rejoins.

Decision / implication: Public follow/block records and private mute rows are not deleted merely because the subject lacks a current Craftsky profile. They are not surfaced or counted while the subject is absent. If the subject later regains membership, Tap backfill and stored private state restore otherwise-current relationship behavior before the profile becomes interactable. A block against an absent subject is not manageable in Craftsky until the subject rejoins.

### Q7: What happens to private mutes when their owner permanently leaves Craftsky?

Answer: Delete them.

Decision / implication: Sign-out, device removal, or account switching does not delete server mute state. Permanent removal of the owning Craftsky membership deletes that owner's AppView-private mute rows. Public PDS block/follow records retain their normal repository lifecycle.

### Q8: How should Craftsky-specific muted content and nested replies behave?

Answer: Mutes cover both ordinary and project posts. A muted reply collapses its full descendant branch, and reveal is temporary.

Decision / implication: Ordinary/project posts and repost activity are suppressed from feeds and discovery. Direct profile/post navigation remains viewable. A muted reply branch is revealed as a unit only for the current thread view and collapses again on refresh, navigation, or account switch.

### Q9: Where should Flutter expose the controls?

Answer: On profiles and every non-self post-shaped item.

Decision / implication: Profiles keep Follow and Share visible and move Mute/Unmute, Block/Unblock, and Report into one More menu. Post, project-post, comment, and reply menus expose Mute/Unmute author, Block/Unblock author, and Report post. Mute acts immediately with feedback; Block and Unblock require confirmation.

### Q10: Should existing public follows to non-members be deleted when the membership boundary ships?

Answer: No.

Decision / implication: Craftsky stops creating new non-member follows and hides existing ones from UI/counts, but it never silently deletes user-owned PDS follow records. Such relationships become visible again if the subject joins Craftsky.

## 4. Candidate Approaches

### Option A: Interoperable PDS blocks, private AppView mutes, and one server policy

Summary: Enforce Craftsky membership at all user-facing account boundaries; write public `app.bsky.graph.block` records through the PDS; index and backfill them in the AppView; store private mutes in account-scoped Postgres; and apply one viewer-relative relationship policy across reads, writes, notifications, push, and Flutter optimistic state.

Pros:

- Matches Bluesky privacy and interoperability semantics.
- Enforces safety at the trusted AppView boundary, not only in Flutter.
- Recognizes blocks created by compatible atproto clients.
- Makes behavior consistent across all current and future endpoints.
- Allows immediate post-write enforcement while Tap later reconciles canonical state.
- Avoids exposing or directing interaction toward accounts that have not joined Craftsky.

Cons:

- Cross-cuts most read and interaction surfaces.
- Requires careful cursor, notification, and cache handling.
- Requires both synchronous write-path updates and asynchronous indexer reconciliation.

Risks: High; omissions can expose blocked content, deliver unwanted notifications, or leak private mute relationships.

### Option B: Client-side filtering with minimal server support

Summary: Persist relationships but let Flutter hide content and disable actions.

Pros:

- Smaller server change.
- Fast UI iteration.

Cons:

- Raw APIs, stale clients, notifications, push, and alternate clients would bypass restrictions.
- Cannot provide Bluesky-equivalent symmetric interaction enforcement.
- Duplicates policy across screens and is prone to safety regressions.

Risks: Unacceptable privacy and abuse-control gaps.

### Option C: Delegate private mutes to Bluesky's AppView APIs

Summary: Call Bluesky's `app.bsky.graph.muteActor` service and consume its private state.

Pros:

- Could share mute state with the Bluesky service for accounts using it.

Cons:

- Makes Craftsky's private controls dependent on a third-party AppView.
- Does not work as a general federated or self-hosted Craftsky design.
- Conflicts with Craftsky's AppView-owned product read model and does not give Craftsky a reliable enforcement source.

Risks: External dependency, portability failure, and unclear privacy/account ownership.

## 5. Recommended Direction

Recommended approach: Option A.

Why: It is the only approach that satisfies all defining requirements together: a hard Craftsky membership boundary, Bluesky-compatible public blocks between members, truly private account-scoped mutes, and server-enforced restrictions across every delivery and interaction path. A central relationship policy should be reused by SQL/read stores, write authorization, notification eligibility, push delivery, and response shaping, while Flutter provides immediate optimistic UX and never becomes the security boundary.

## 6. Problem / Opportunity

Craftsky users can currently follow, mention, reply to, quote, repost, like, search for, and receive notifications from any indexed account. Reporting and platform moderation exist, but users have no personal safety controls. Adding complete Bluesky-compatible mutes and blocks gives each person immediate control over attention and interaction while preserving AT Protocol interoperability and Craftsky's public/private data boundary.

## 7. Goals

- G-001: Let a signed-in user privately mute and unmute any non-self Craftsky member.
- G-002: Let a signed-in user publicly block and unblock any non-self Craftsky member using `app.bsky.graph.block`.
- G-003: Apply Bluesky-equivalent mute visibility behavior to all current Craftsky read and notification surfaces.
- G-004: Apply Bluesky-equivalent symmetric block visibility and interaction restrictions to all current Craftsky surfaces.
- G-005: Recognize compatible block records created outside Craftsky, including historical records backfilled when a DID joins or rejoins as a member.
- G-006: Provide complete Flutter actions, state, feedback, placeholders, and settings lists for both controls.
- G-007: Keep mute data and derived mute state isolated to the authenticated account on the AppView and device.
- G-008: Make enforcement immediate after successful mutation and eventually consistent with PDS/Tap canonical state.
- G-009: Make current Craftsky membership a hard eligibility boundary for profiles, graph controls, discovery, lists, and directed interaction without deleting pre-existing public records.

## 8. Non-Goals

- NG-001: No muted words, tags, phrases, or languages.
- NG-002: No thread mute, post hide, snooze, temporary mute, or mute expiry.
- NG-003: No moderation-list mute/block subscriptions, list blocks, or starter-pack/list-specific block behavior.
- NG-004: No direct-message or group-chat restrictions because Craftsky has no DM product surface.
- NG-005: No changes to reports, Ozone labels, platform moderation policy, or moderation reason taxonomy.
- NG-006: No local record lexicon and no changes under `lexicon/`.
- NG-007: No synchronization of Craftsky-private mute state with Bluesky's hosted AppView or another third-party service.
- NG-008: No promise that public content or public block records are inaccessible outside compliant authenticated AppViews and clients.
- NG-009: No destructive deletion of existing posts, replies, likes, reposts, follows, mentions, or notifications from either user's PDS.
- NG-010: No anonymous/public Craftsky browsing changes; current product APIs remain authenticated.
- NG-011: No pixel-for-pixel copy of Bluesky UI. Behavioral parity, clear localized copy, and equivalent state/action affordances are required.
- NG-012: No user-facing non-Craftsky profiles, follows, mutes, blocks, reports, account references, or relationship management. Existing public records involving non-members are preserved but hidden.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Muting user | Signed-in Craftsky member choosing not to see another member's unsolicited content or notifications | Private, one-way attention control without alerting or restricting the muted member. |
| Muted member | Craftsky member muted by another member | No disclosure of the mute and no change to their ability to view or interact. |
| Blocking user | Signed-in Craftsky member preventing mutual viewing and interaction | Immediate symmetric enforcement and a manageable public block list containing current members only. |
| Blocked member | Craftsky member targeted by a public block | Clear block-state annotation in compliant clients and no permitted interaction with the blocker. |
| Non-member atproto account | Resolvable DID without a current `craftsky_profiles` membership row | Remain absent from all user-facing Craftsky account surfaces and mutations. |
| Flutter client | Account-aware Craftsky mobile client | Render relationship state, actions, placeholders, lists, optimistic updates, and safe errors. |
| AppView | Trusted read/write and policy boundary | Persist private mutes, mediate PDS block writes, index public blocks, and enforce policy everywhere. |
| User PDS | Repository host for the blocking account | Store and delete canonical `app.bsky.graph.block` records. |
| Tap/firehose | Public record event source | Deliver block create/update/delete events for reconciliation and cross-client interoperability. |

## 10. Current Behavior

Mutes and blocks do not exist. Profile responses only describe follow state; profiles expose follow and report actions; Settings only links to followers/following; all visible posts, projects, comments, quotes, search results, notifications, and push events ignore personal relationships; and interaction endpoints accept any otherwise-valid target. The AppView does not subscribe to or backfill `app.bsky.graph.block`. Contrary to the newly confirmed boundary, current profile/follow code can view and follow resolvable non-Craftsky accounts.

## 11. Desired Behavior

A user can mute, unmute, block, or unblock another current Craftsky member from profile and post More menus and manage current muted/blocked members from Settings. A non-member handle or DID returns the same `404 profile_not_found` boundary across account reads and mutations. Existing relationships with absent subjects remain hidden until the subject rejoins.

Mutes are silent, Craftsky-local, private, one-way controls: the viewer stops receiving the actor's notifications and ordinary/project top-level discovery content, while muted reply branches and muted quote content are collapsed behind an explicit reveal and direct profile/post navigation remains available. A branch reveal lasts only for the current thread view. Mutes do not stop either party from following or interacting.

Blocks are public and symmetric. The profile remains identifiable and annotated so the relationship can be understood, reported, and reversed, but its bio, metrics, content tabs, and normal interaction controls are unavailable across the block. Neither party receives the other's posts, replies, repost activity, quote content, mentions, notifications, or push delivery in an authenticated Craftsky view. Neither can create a follow, like, repost, reply, quote/embed, or mention targeting the other through Craftsky. Cleanup actions and reporting remain possible. Underlying public records and existing follow records are not deleted; compliant visibility can return after unblock.

The AppView is authoritative for membership and relationship enforcement. Flutter optimistically removes or collapses affected content and updates relationship state immediately, but all reads, writes, notification eligibility, new-count computation, and pending push delivery independently enforce the same policy. Public block events from other clients reconcile into the same state. Underlying public follow/block and interaction records are preserved; viewer-relative state, lists, search, and block-violating references are shaped without destructive rewrites.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky shall provide complete individual account mute and block controls on both AppView and Flutter. | Personal safety controls must work beyond one screen or client. | Prompt | AC-001, AC-002 |
| BR-002 | Business | Must | Craftsky block behavior shall interoperate with Bluesky by using and honoring `app.bsky.graph.block`. | A public block must travel with the user's atproto repository and be enforceable across compatible services. | Prompt; canonical block lexicon | AC-003, AC-004 |
| BR-003 | Business | Must | Craftsky mute relationships shall remain private to the muting account and Craftsky's trusted AppView. | A muted account must not learn or infer the private control from Craftsky data surfaces. | Prompt; Bluesky FAQ | AC-005, AC-006 |
| BR-004 | Business | Must | Craftsky shall expose and permit account relationships only for current Craftsky members; a non-member account shall be indistinguishable from an unknown profile through user-facing Craftsky APIs and UI. | Craftsky is a closed product surface over a public protocol, and membership is the eligibility boundary. | Grilling decision | AC-051 |
| FR-001 | Functional | Must | The system shall resolve a mute/block target from a valid handle or DID, canonicalize it to a DID, require a current Craftsky profile, reject self-targeting, and return `404 profile_not_found` for an unknown or non-member target. | Relationship identity must survive handle changes without exposing or targeting accounts outside Craftsky. | Codebase; Grilling decision | AC-007 |
| FR-002 | Functional | Must | The AppView shall create and delete account-scoped mute rows idempotently and make a successful mutation effective immediately. | Private mute state must be reliable and retry-safe. | Bluesky mute contract | AC-008, AC-009 |
| FR-003 | Functional | Must | The AppView shall create a canonical `app.bsky.graph.block` record on the caller's PDS and delete that exact record when unblocking; repeated requests shall converge without duplicate active pairs. | The PDS record is public canonical state. | Prompt; block lexicon | AC-010, AC-011 |
| FR-004 | Functional | Must | The AppView shall subscribe to, validate, idempotently index, update, and delete `app.bsky.graph.block` events keyed by record URI/CID and active blocker/subject pair. | Cross-client changes and firehose replay must converge safely. | Architecture; block lexicon | AC-012 |
| FR-005 | Functional | Must | When an account joins or rejoins Craftsky, the AppView shall persist a fail-closed activation gate, Tap shall backfill compatible block records owned by the joining repository, and activation shall verify already-indexed block records owned by current members that target the joining DID before the account becomes interactable. Interrupted or failed activation shall resume or retry safely after process restart, and the system shall thereafter honor indexed blocks in either direction between current Craftsky members. | Existing Bluesky blocks must take effect at the membership boundary without exposing non-members or opening a restart race. | Grilling decision; interoperability goal; Document review DR-001, DR-002 | AC-013, AC-059 |
| FR-006 | Functional | Must | A successful block/unblock API response shall reflect the new relationship in server reads before Tap convergence, and later Tap events shall reconcile without reversing newer canonical state. | The normal indexing delay cannot create a safety gap or race regression. | Reference architecture; Discovery | AC-014 |
| FR-007 | Functional | Must | Profile and profile-summary responses shall expose viewer-relative `muted`, `blocking`, and `blockedBy` state only to the authenticated viewer for whom it is meaningful. | Flutter needs state without leaking another account's private mute data. | Bluesky viewer-state pattern | AC-015 |
| FR-008 | Functional | Must | The API shall provide opaque-cursor paginated lists of the authenticated account's muted and blocked current Craftsky members, with default 50 and maximum 100 items, stable profile summaries, and no cross-account access. | Users must be able to review and reverse eligible controls at scale without surfacing former/non-members. | Bluesky getMutes/getBlocks; Grilling decision; API conventions | AC-016 |
| FR-009 | Functional | Must | Muted actors' top-level ordinary posts, project posts, straight repost activity, and reposts of their content shall be omitted from the viewer's home timeline, post/project/hashtag search, project discovery, and other top-level discovery results. | Bluesky mutes suppress unsolicited top-level content across Craftsky's equivalent surfaces. | Bluesky FAQ; Grilling decision | AC-017 |
| FR-010 | Functional | Must | In threads, a muted actor's reply and its full descendant branch shall be represented by a localized “post from an account you muted” placeholder; explicit reveal shall reveal that branch as a unit only for the current thread view and reset on refresh, navigation, or account switch. | This preserves thread structure without making a temporary reveal global or persistent. | Bluesky FAQ; Grilling decision | AC-018 |
| FR-011 | Functional | Must | A quote/embed whose quoted author is muted shall hide its preview behind a revealable muted-content placeholder while preserving the unmuted quoting post. | Mute policy must apply when content is embedded by someone else. | Bluesky mute behavior; current quote surface | AC-019 |
| FR-012 | Functional | Must | Direct navigation to a muted account's profile and posts, and the account's explicit profile content tabs, shall remain available with mute state shown; the user may reveal muted content without unmuting. | Mute is attention control, not a mutual access restriction. | Bluesky mute behavior | AC-020 |
| FR-013 | Functional | Must | Muting shall suppress notification listing, new-count contribution, badges, and future push delivery from the muted actor; the underlying notification event may be retained, but pending unsent deliveries shall be cancelled. | “No notifications” must include every delivery channel while remaining compatible with dynamic history filtering. | Bluesky FAQ; Bluesky current tests | AC-021, AC-022 |
| FR-014 | Functional | Must | Blocking in either direction shall omit both accounts' authored posts/projects, replies, repost activity, and blocked-author content from each other's timelines, discovery results, explicit content lists, threads, and direct content responses. | Blocks are symmetric mutual mutes. | Bluesky block guide | AC-023, AC-024 |
| FR-015 | Functional | Must | A quote/embed whose quoted author is blocked in either direction shall render an unrevealable blocked/unavailable placeholder; a straight repost of blocked-author content shall be omitted. | An intermediary must not bypass the mutual block. | Bluesky block guide; quote surface | AC-025 |
| FR-016 | Functional | Must | A blocked profile shall retain only the minimum identity and relationship annotation needed to understand, report, manage, or reverse the block; bio, metrics, mutuals, content tabs, follow controls, and activity content shall not be exposed across the block. | Bluesky keeps block state understandable without serving normal profile content. | Bluesky current client; block guide | AC-026 |
| FR-017 | Functional | Must | The AppView shall reject creation of follows, likes, reposts, replies, quotes/embeds, and mentions when either account blocks the other, using the standard error envelope and a stable `interaction_blocked` code. | Client hiding alone cannot enforce a symmetric interaction block. | Bluesky block guide | AC-027 |
| FR-018 | Functional | Must | Blocked users shall remain able to delete their own existing follow/like/repost records, delete their own content, report the other account or content when addressable, and block the other account independently. | Cleanup, safety reporting, and reciprocal public blocks must remain possible. | Bluesky record-preservation behavior; existing reports | AC-028 |
| FR-019 | Functional | Must | Blocks shall suppress notification eligibility/listing, new-count contribution, badge state, and push delivery involving the other account in either actor/recipient direction; the underlying notification event may be retained, but pending unsent deliveries shall be cancelled. | No notification path may bypass a block while dynamic history remains recoverable. | Bluesky block guide; Bluesky current tests | AC-029, AC-030 |
| FR-020 | Functional | Must | Existing posts, replies, mentions, likes, reposts, follow records, public block records, and moderation reports shall remain stored when a mute/block is created or its subject leaves Craftsky; removing the relationship or the same DID rejoining shall restore otherwise-eligible state without reconstructing records. | Policy and membership eligibility are view-time concerns and must not rewrite user-owned public records. | Bluesky block guide; Grilling decision | AC-031, AC-052 |
| FR-021 | Functional | Must | Across a block, follow/follower/mutual lists shall suppress the pair's entries and viewer follow state; ordinary actor search shall omit the blocked member, while an exact-handle lookup may return only the minimum annotated blocked-profile shell. | Secondary account surfaces must match Bluesky's block-safe shaping without preventing block management. | Bluesky current tests | AC-032, AC-060 |
| FR-022 | Functional | Must | On an eligible unblocked visitor profile, Flutter shall keep Follow and Share as primary visible actions and place localized Mute/Unmute, Block/Unblock, and Report actions in a More menu; Block and Unblock shall each require confirmation, and Mute/Unmute shall apply immediately with feedback. Blocked-profile restrictions in FR-016 take precedence. | The agreed action hierarchy keeps common actions visible and makes public block changes deliberate. | Grilling decision | AC-033 |
| FR-023 | Functional | Must | Flutter shall show distinct localized profile states for “muted,” “blocked by you,” and “has blocked you,” with only actions valid for that state. | The three relationships have different privacy and reversal semantics. | Bluesky viewer state | AC-034 |
| FR-024 | Functional | Must | Flutter Settings shall provide paginated Muted accounts and Blocked accounts screens containing current Craftsky members only, with loading, empty, retry, pagination, unmute/unblock, and profile navigation behavior allowed by policy. | A control is incomplete if users cannot find and reverse it later, but absent accounts must not be surfaced. | Bluesky list behavior; Grilling decision; current Settings | AC-035 |
| FR-025 | Functional | Must | Flutter shall optimistically update the active account's relationship state and currently loaded content after a mutation, roll back on failure, refresh affected feeds/profile/notification counts, and never apply the state to another signed-in account. | Immediate UX must not sacrifice multi-account isolation or correctness. | Reference architecture; multi-account client | AC-036, AC-037 |
| FR-026 | Functional | Must | All new endpoints shall use existing `/v1/` authentication/device middleware, camelCase JSON, standard error envelopes, read/write rate classes, and opaque cursors. | The feature must conform to the governing API contract. | AGENTS.md; API architecture | AC-038 |
| FR-027 | Functional | Must | Every non-self ordinary post, project post, comment, and reply More menu shall expose current-state Mute/Unmute author, Block/Unblock author, and Report post actions; after a successful mute, list/discovery items from that author disappear immediately, a directly viewed root remains visible with muted state, and other muted reply branches collapse. | Safety controls must be available where content is encountered and update the current context predictably. | Grilling decision | AC-054, AC-055 |
| FR-028 | Functional | Must | Profile reads, search, mention resolution/suggestions, follower/following/mutual lists and counts, mute/block lists, reports, and all account relationship or directed-interaction mutations shall require every referenced account to be a current Craftsky member; non-members shall be omitted or return `404 profile_not_found`, and no new public relationship record shall be written for them. | Membership must be a single, consistent server-enforced product boundary. | Grilling decision | AC-051, AC-052 |
| FR-029 | Functional | Must | Removing a mute subject's Craftsky membership shall retain but hide the private mute row; permanently removing the mute owner's Craftsky membership shall delete all private mutes owned by that DID, while sign-out, device removal, and account switching shall not. | Subject return must restore the preference, but private AppView state must not outlive its owner. | Grilling decision | AC-052, AC-053 |
| FR-030 | Functional | Must | The AppView shall hide block-violating replies, mentions, quotes/embeds, and other references from third-party viewers when rendering them would connect or expose a blocked pair, even when the third-party viewer blocks neither account. | An intermediary or public thread must not route around a block. | Bluesky current tests | AC-056 |
| FR-031 | Functional | Must | Notification records may be retained and filtered dynamically; after unmute or unblock, previously hidden notification history may reappear if it remains within normal retention and is otherwise eligible, but already-sent pushes shall never be replayed solely because the relationship was removed. | This matches Bluesky's view-time filtering without duplicating or recreating delivery. | Bluesky current tests | AC-057 |
| NFR-001 | Non-functional | Must | Mute data, cache keys, provider state, logs, traces, and metrics shall not expose a mute relationship to another account or include the target DID/handle as telemetry dimensions. | Mutes are sensitive private preferences. | Prompt; privacy boundary | AC-006, AC-039 |
| NFR-002 | Non-functional | Must | Relationship policy shall be enforced server-side through a shared, auditable policy abstraction or predicates used by every affected read, write, notification, and push path. | Scattered ad hoc checks create high-risk omissions. | Discovery | AC-040 |
| NFR-003 | Non-functional | Must | List pagination and filtered content pagination shall fill pages from eligible rows where available, avoid duplicates/skips caused by post-query filtering, and keep cursors opaque. | Safety filtering must not break list usability or leak hidden rows through page shape. | API conventions | AC-041 |
| NFR-004 | Non-functional | Should | Relationship checks shall use indexed lookups suitable for feed/search/thread query plans and avoid per-item database queries. | Viewer-relative filtering touches high-volume paths. | Codebase | AC-042 |
| NFR-005 | Non-functional | Must | All new Flutter copy and controls shall be localized, accessible by semantics/tooltip, keyboard/screen-reader usable, and visually distinguish destructive block actions. | Safety controls must be understandable and operable. | Flutter conventions | AC-043 |
| NFR-006 | Non-functional | Must | Block indexer lag, mutation failures, policy denials, and push cancellations shall be observable without recording private mute pairs or public block targets as unbounded metric labels. | Operations need evidence without privacy leaks or cardinality hazards. | Observability conventions | AC-044 |
| RULE-001 | Business rule | Must | A mute is private, one-way, and affects only what the muting account sees or receives; it does not restrict either account's reads or interactions. | Matches Bluesky mute semantics. | Prompt; Bluesky FAQ | AC-005, AC-045 |
| RULE-002 | Business rule | Must | A block is public and symmetric for visibility, delivery, and interaction whenever either account blocks the other. | Matches Bluesky block semantics. | Prompt; Bluesky block guide | AC-003, AC-046 |
| RULE-003 | Business rule | Must | Block policy takes precedence over mute policy, and existing platform hide/takedown policy takes precedence over either relationship when shaping content. | The strictest applicable safety decision must win deterministically. | Existing moderation; Bluesky behavior | AC-047 |
| RULE-004 | Business rule | Must | A user cannot mute or block their own DID. | Self-relationships have no valid product meaning and complicate policy. | Bluesky graph behavior | AC-007 |
| RULE-005 | Business rule | Must | Blocking does not remove an existing follow record and block/unblock cannot be used as a soft-block follower-removal mechanism. | Matches Bluesky's public repository model. | Bluesky block guide | AC-031, AC-048 |
| RULE-006 | Business rule | Must | Muted or blocked content shall never be silently revealed by repost, quote, notification subject hydration, push open routing, stale client cache, or pagination. | Indirect references are common policy bypasses. | Bluesky behavior; current surfaces | AC-019, AC-025, AC-049 |
| RULE-007 | Business rule | Must | Only the authenticated account may enumerate or mutate its mutes; only the block-record owner may create/delete its block through Craftsky, although inbound public blocks remain readable for enforcement. | Preserves private ownership and PDS authorization. | Architecture | AC-050 |
| RULE-008 | Business rule | Must | Mute, block, and Craftsky-membership filtering shall not delete or rewrite public interaction records or their record-based aggregate contributions for unrelated third parties; across a blocked pair, viewer-relative state and profile metrics remain hidden. | Bluesky applies relationship policy to delivery and views, not canonical public counts. | Bluesky current tests; Grilling decision | AC-058 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001 | Given a signed-in user visits another current Craftsky member, when the profile loads, then complete mute and block actions are available on Flutter and backed by authenticated AppView operations. |
| AC-002 | BR-001 | Given a relationship exists, when any equivalent Craftsky surface is read or acted upon, then the server and client apply the same relationship policy. |
| AC-003 | BR-002, RULE-002 | Given Alice blocks Bob, when Alice's repository is inspected, then one valid public `app.bsky.graph.block` record names Bob's DID. |
| AC-004 | BR-002 | Given a valid block was created by another compatible client and indexed/backfilled, when Craftsky serves either account, then it enforces that block without requiring recreation in Craftsky. |
| AC-005 | BR-003, RULE-001 | Given Alice mutes Bob, when Bob reads any Craftsky profile/API response as Bob, then no field or behavior tells Bob that Alice muted him. |
| AC-006 | BR-003, NFR-001 | Given a mute exists, when another account, telemetry consumer, or unrelated device account is observed, then the mute pair and mute-derived state are absent. |
| AC-007 | FR-001, RULE-004 | Given a valid handle or DID for another current Craftsky member, when targeted, then it resolves to the canonical DID; an invalid identifier or the caller's own DID is rejected without persistence, while an unknown or non-member target returns `404 profile_not_found`. |
| AC-008 | FR-002 | Given no mute exists, when mute succeeds, then exactly one account-scoped row exists and subsequent reads enforce it before the response is observed. |
| AC-009 | FR-002 | Given mute/unmute is retried, when the requested state already exists, then the operation succeeds idempotently without duplicate rows or cross-account effects. |
| AC-010 | FR-003 | Given no block exists, when block succeeds, then the PDS write succeeds before the API reports success and one indexed/optimistic active pair identifies the returned record. |
| AC-011 | FR-003 | Given an active block exists, when unblock or either operation is retried, then the exact owned record is deleted at most once and the final state converges without duplicate active blocks. |
| AC-012 | FR-004 | Given create, update, duplicate replay, stale replay, and delete Tap events, when indexed, then URI/CID and pair uniqueness converge to the newest canonical active state idempotently. |
| AC-013 | FR-005 | Given a joining account owns historical `app.bsky.graph.block` records targeting current members, when membership activation starts, fails, is interrupted, or the process restarts, then a persisted fail-closed gate keeps the account non-interactable until Tap backfill resumes and completes, and every owned block is enforced before either profile can be viewed or interacted with. |
| AC-014 | FR-006 | Given a PDS mutation succeeds but Tap has not delivered its event, when the caller immediately refreshes or acts, then new policy is enforced; the later matching event is a no-op reconciliation rather than a reversal. |
| AC-015 | FR-007 | Given Alice reads Bob's profile, when Alice has muted Bob, blocks Bob, is blocked by Bob, or has no relationship, then only Alice's response contains the correct viewer-relative booleans and no other viewer receives Alice's mute state. |
| AC-016 | FR-008 | Given more than one page of mutes/blocks containing current and former members, when Alice paginates, then she receives each current-member subject once in stable order with opaque cursors and full eligible pages, while former members are absent and Bob cannot request Alice's private list. |
| AC-017 | FR-009 | Given Alice muted Bob, when Alice loads timeline, post/project/hashtag search, discovery, or repost feed items, then Bob's top-level content and repost activity are absent and pages fill from later eligible rows. |
| AC-018 | FR-010 | Given Bob is muted and replied above one or more descendants, when Alice opens the thread, then Bob's full branch is collapsed with localized mute copy; revealing shows the branch as a unit only until refresh, navigation away, or account switch. |
| AC-019 | FR-011, RULE-006 | Given an unmuted account quotes muted Bob, when Alice sees the quoting post, then the quoting post remains but Bob's quote preview is hidden behind a revealable muted-content control. |
| AC-020 | FR-012 | Given Alice muted Bob, when Alice intentionally opens Bob's profile, profile tabs, or a direct Bob post, then the content can be viewed, mute state remains visible, and Alice need not unmute. |
| AC-021 | FR-013 | Given Bob is muted before notification activity is indexed, when Bob likes/follows/replies/mentions/quotes/reposts toward Alice, then any retained event is ineligible for Alice's notification list, new count, badge, and push delivery. |
| AC-022 | FR-013 | Given a Bob notification has a pending/retry/leased-unsent delivery, when Alice mutes Bob, then it is no longer listed/counted and no future send succeeds; already-sent pushes are not claimed to be retractable. |
| AC-023 | FR-014 | Given either Alice blocks Bob or Bob blocks Alice, when either loads feeds, searches, profile content lists, projects, comments, or threads, then the other's authored/attributed content is absent. |
| AC-024 | FR-014 | Given a blocked post is fetched directly or reached through a stale deep link, when the request is served, then no post text/media/private response detail is returned and Flutter shows only a generic blocked/unavailable state. |
| AC-025 | FR-015, RULE-006 | Given an unblocked account quotes/reposts a blocked actor's post, when the protected viewer loads it, then the quote is unrevealably unavailable and the straight repost item is omitted. |
| AC-026 | FR-016 | Given a block exists in either direction, when either account opens the other's profile, then minimum identity and accurate block annotation/report or owned-unblock action remain, while bio, metrics, mutuals, tabs, follow, and activity are absent. |
| AC-027 | FR-017 | Given a block exists in either direction, when either user attempts a new follow, like, repost, reply, quote/embed, or mention through the API, then no PDS write occurs and `interaction_blocked` is returned in the standard envelope. |
| AC-028 | FR-018 | Given a block exists, when a user deletes their own follow/like/repost/content, reports an addressable subject, or creates their own reciprocal block, then the valid safety/cleanup operation remains available. |
| AC-029 | FR-019 | Given a block exists before activity ingestion, when either actor generates a notification event involving the other, then any retained event is ineligible for lists/counts/badges and no push outbox delivery is created. |
| AC-030 | FR-019 | Given pending or listed notifications already involve the newly blocked pair, when the block becomes effective, then they disappear from lists/counts and all pending unsent deliveries are cancelled. |
| AC-031 | FR-020, RULE-005 | Given content, interactions, and a follow predate a block, when block then unblock occurs, then the source records were never deleted and otherwise-eligible content/relationships become visible again. |
| AC-032 | FR-021 | Given Alice and Bob are blocked in either direction, when Alice or Bob loads viewer follow state or follower/following/mutual lists, then the relationship entry and follow affordance are suppressed without deleting the underlying follow record. |
| AC-033 | FR-022 | Given an eligible unblocked visitor profile, when it renders and its More menu is opened, then Follow and Share remain primary, localized mute/block/report choices reflect current state, Mute/Unmute applies with immediate feedback, and both Block and Unblock open confirmations describing their consequences; a blocked profile instead follows FR-016. |
| AC-034 | FR-023 | Given each mute/block direction, when Flutter renders the profile, then it uses a distinct accessible state and exposes only valid actions such as Unmute, Unblock-owned-block, Report, or reciprocal Block. |
| AC-035 | FR-024 | Given zero, one, failing, or multiple pages of muted/blocked current members, when Settings screens are used, then localized empty/loading/error/list/pagination states work, successful reversal removes the row, and no non-member/tombstone row is shown. |
| AC-036 | FR-025 | Given a mutation is in flight, when it succeeds or fails, then Flutter optimistically updates once, blocks duplicate taps, rolls back on failure, shows feedback, and refreshes affected server-backed state. |
| AC-037 | FR-025 | Given two accounts are signed in on one device, when one mutates or reveals muted content, then the other account's providers, caches, lists, and UI remain unchanged. |
| AC-038 | FR-026 | Given each new route, when requests cover auth, missing device ID, invalid input, rate limits, success, and failure, then existing `/v1/` middleware, camelCase, status, and envelope conventions hold. |
| AC-039 | NFR-001 | Given diagnostics are captured during mute operations, when logs/traces/metrics are inspected, then they contain bounded operation/result data but no mute target DID, handle, pair, or list content. |
| AC-040 | NFR-002 | Given the affected endpoint inventory, when policy tests exercise each read/write/notification/push path, then every path uses the shared relationship decision and no client-only enforcement dependency exists. |
| AC-041 | NFR-003 | Given hidden rows occur before/between visible rows, when pages are traversed, then eligible items are neither skipped nor duplicated and page size is filled when sufficient eligible rows exist. |
| AC-042 | NFR-004 | Given a representative feed/search/thread page, when query plans and store-call tests are inspected, then relationship enforcement uses indexed joins/batched lookups and not one query per item. |
| AC-043 | NFR-005 | Given supported locales and assistive navigation, when controls/placeholders/dialogs/lists are used, then copy is localized, actions have semantics, focus order is valid, and destructive block styling is distinguishable. |
| AC-044 | NFR-006 | Given index lag, PDS/store failures, denied interactions, and cancelled pushes, when operations are observed, then bounded metrics/logs identify operation and failure stage without target identifiers. |
| AC-045 | RULE-001 | Given Alice mutes Bob, when either follows, likes, reposts, replies, quotes, mentions, or views the other's profile/content directly, then the operation remains permitted even though Alice's unsolicited delivery is suppressed. |
| AC-046 | RULE-002 | Given either direction of block, when both accounts are tested, then the same visibility, delivery, and interaction restrictions apply symmetrically. |
| AC-047 | RULE-003 | Given content is both muted/blocked and hidden/taken down by platform moderation, when shaped, then hide/takedown wins; otherwise block wins over mute and no reveal weakens the stricter result. |
| AC-048 | RULE-005 | Given Alice follows Bob and then blocks/unblocks Bob, when the graph is inspected, then the follow record still exists and the block did not remove Bob as a follower or followed account. |
| AC-049 | RULE-006 | Given protected content is reachable through quote hydration, notification subject, push deep link, stale cache, or a later page, when opened, then current server policy is rechecked and content is not silently shown. |
| AC-050 | RULE-007 | Given Alice and Bob, when either attempts to mutate/enumerate the other's private mutes or delete a block record they do not own, then authorization denies it without changing state; inbound public blocks remain usable only for enforcement. |
| AC-051 | BR-004, FR-028 | Given a resolvable DID or handle has no current Craftsky profile, when any profile, search, suggestion, graph list/count, mute/block list, report, follow/mute/block, or directed-interaction surface targets it, then it is omitted or returns `404 profile_not_found`, creates no new relationship record, and reveals no identity detail. |
| AC-052 | FR-020, FR-028, FR-029 | Given Bob leaves Craftsky while Alice has an existing public follow/block record or private mute row naming Bob, when Bob is absent then all such relationships are hidden and unmanageable without deleting them; when the same DID rejoins and backfill completes, otherwise-current state reappears before interaction is allowed. |
| AC-053 | FR-029 | Given Alice owns private mute rows, when she signs out, removes a device, or switches accounts, then the rows remain; when Alice's Craftsky membership is permanently removed, then all private mute rows owned by Alice are deleted without deleting public PDS records. |
| AC-054 | FR-027 | Given any non-self ordinary post, project post, comment, or reply, when its More menu opens, then it offers current-state Mute/Unmute author, Block/Unblock author, and Report post actions with the same confirmations and feedback as the profile controls. |
| AC-055 | FR-027 | Given Alice mutes Bob from a content menu, when the mutation succeeds, then Bob-authored items disappear immediately from list/feed/search/discovery contexts, a directly viewed Bob root remains visible with muted state, and any other Bob reply branch collapses as a unit. |
| AC-056 | FR-030 | Given Alice blocks Bob and Carol blocks neither, when Carol loads a thread, quote/embed, mention, or other reference that violates the Alice-Bob block, then the protected reference/content is hidden according to its surface and cannot be used to connect the blocked pair. |
| AC-057 | FR-031 | Given a retained notification was hidden by mute/block, when the relationship is removed while the notification remains within normal retention, then it may reappear once in list/count state if otherwise eligible, and no already-sent or cancelled push is replayed. |
| AC-058 | RULE-008 | Given public follows, likes, reposts, replies, or aggregate contributions predate a mute, block, or subject membership loss, when unrelated third parties inspect eligible aggregate views, then record-based contributions remain intact; the protected pair still receives hidden viewer state and metrics. |
| AC-059 | FR-005 | Given an already-current member owns an indexed historical block targeting a non-member, when that target later joins Craftsky, then the retained block is verified during activation and enforced before the joining account becomes visible or can interact, including after activation failure, interruption, or process restart. |
| AC-060 | FR-021 | Given a block exists between two current members, when either performs ordinary actor search then the other is absent; when an exact-handle lookup is required to manage the relationship, then only the minimum annotated blocked-profile shell is returned. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Target handle changes after relationship creation | DID remains canonical; the current handle is resolved only for display. | FR-001, FR-007 |
| EC-002 | Target account has no current Craftsky profile | Profile or mutation returns `404 profile_not_found`; search, suggestions, graph counts/lists, reports, and relationship lists omit it without hydrating external identity. | BR-004, FR-001, FR-028 |
| EC-003 | Duplicate block records for the same pair exist from another client | AppView selects/collapses active pair state deterministically, retains enough owned record identity to unblock, and prevents new duplicates through Craftsky. | FR-003, FR-004 |
| EC-004 | PDS block succeeds but optimistic AppView persistence fails | API does not claim a fully applied success; client keeps a safe optimistic hide while Tap/backfill reconciles, and retry is idempotent. | FR-003, FR-006 |
| EC-005 | AppView state changes but PDS unblock fails | Public block remains authoritative and restrictions remain; Flutter rolls back and reports failure. | FR-003, FR-025 |
| EC-006 | Tap delivers old create after newer delete, or duplicates events | URI/CID/version-aware reconciliation must not resurrect stale state. | FR-004, FR-006 |
| EC-007 | Both accounts block each other | Both public records may coexist; removing one still leaves the other-direction block and all block restrictions active. | RULE-002 |
| EC-008 | User mutes then blocks the same account | Block policy wins; mute row may remain private and becomes effective again if the owned block and any inbound block are removed. | RULE-003 |
| EC-009 | User blocks then mutes | Mute action is unnecessary/hidden while blocking; server remains idempotent and block policy wins if requested by a stale client. | RULE-003, FR-023 |
| EC-010 | Muted reply is a parent of unmuted replies | The muted branch is initially collapsed as a unit; explicit reveal can show the branch for that view without unmuting globally. | FR-010 |
| EC-011 | Blocked reply is a parent of unblocked replies | No blocked content is revealed; thread shaping must not attach a visible subtree in a way that exposes or attributes the blocked parent. | FR-014 |
| EC-012 | Quoting post is allowed but quoted content becomes muted/blocked after caching | Current relationship policy reshapes the quote on refresh/open and stale preview content is discarded. | FR-011, FR-015, RULE-006 |
| EC-013 | Notification was already pushed before mute/block | It cannot be retracted; subsequent list/open hydration rechecks policy, while a later unmute/unblock may restore retained history but never replays that push. | FR-013, FR-019, FR-031, RULE-006 |
| EC-014 | Notification is leased concurrently with mute/block | Transactional/conditional delivery checks prevent a send after the relationship becomes effective where the provider send has not already occurred. | FR-013, FR-019 |
| EC-015 | User unblocks before Tap indexes the create | Synchronous state plus record identity ensures delete targets the created record and late events cannot restore the old state. | FR-006 |
| EC-016 | Session expires during PDS block write | Standard PDS-session expiry behavior applies; no success is shown and local state does not become canonical without a PDS record. | FR-003, FR-026 |
| EC-017 | Account deletion/takedown overlaps mute/block | Moderation takedown wins. A departed subject's relationship rows/records remain hidden for same-DID return; permanent removal of the mute owner deletes their private mute rows, while public PDS records follow repository lifecycle. | FR-020, FR-029, RULE-003 |
| EC-018 | Pagination page contains mostly protected rows | Filtering occurs in the query/selection process so eligible rows fill the page and cursors do not leak protected row identities. | NFR-003 |
| EC-019 | Multi-account switch during mutation/reveal | Completion is applied only to the initiating account's keyed state; the newly active account is not modified. | FR-025, NFR-001 |
| EC-020 | User reports an account they block or that blocks them | Profile-level report remains available from the annotated identity; hidden raw content is not re-exposed solely to report it. | FR-018 |
| EC-021 | Existing public follow points to a non-member | Craftsky neither deletes the follow nor surfaces/counts it and creates no new non-member follow; it becomes eligible again if the same DID joins. | FR-020, FR-028, RULE-005, RULE-008 |
| EC-022 | Blocked or muted subject leaves then rejoins | No list tombstone or management row is shown while absent; retained state and backfilled public blocks reactivate before the same DID becomes interactable. | FR-005, FR-020, FR-029 |
| EC-023 | Third-party thread connects a blocked pair | Block-violating reply, mention, or embed is shaped out even though the viewer is not a party to the block. | FR-030 |
| EC-024 | Process exits during membership block backfill | Persisted activation state remains fail-closed; after restart the AppView resumes or retries backfill and verification idempotently, and the joining account cannot be viewed or interacted with until completion. | FR-005 |

## 15. Data / Persistence Impact

- New private data:
  - Account-scoped mute relationship keyed at minimum by `(viewer_did, subject_did)` with creation/update timestamps and a unique active pair.
  - No mute record, target list, or mute-derived field is written to a PDS.
  - Subject membership removal must not cascade-delete a mute; permanent owner membership removal must delete every mute owned by that DID. Sign-out, device removal, and account switching do not affect persistence.
- New public indexed data:
  - Active `app.bsky.graph.block` rows retaining URI, blocker DID, rkey, CID, subject DID, record JSON, created time, indexed time, and uniqueness needed for idempotent pair/rkey/URI reconciliation.
  - Indexes must support both `blocker -> subject` and `subject -> blocker` checks and paginated owned block lists.
- Migration required: Yes; add mute and block tables/indexes with reversible down migration and database migration tests.
- Tap/indexing impact:
  - Add `app.bsky.graph.block` to collection filters and dispatcher wiring.
  - Index and retain valid block records owned by current members even while their subject is not a current member; membership still prevents those records or subjects from appearing on user-facing surfaces.
  - Persist the joining account's fail-closed activation/backfill state. Resume or retry unfinished work idempotently after process restart rather than reopening visibility or interaction.
  - Gate membership activation/profile visibility on both Tap backfill of historical blocks owned by the joining repository and verification of retained blocks owned by current members that target the joining DID, so both directions are enforced before interaction.
- Existing data:
  - Existing follow/content/interaction/public block rows remain unchanged when a subject loses membership; they are hidden from Craftsky surfaces and may reactivate for the same DID on rejoin.
  - Existing public follows to non-members are not deleted, but are excluded from UI, relationship state, and counts, and no new non-member follow is created.
  - Existing notification rows may remain durable audit data but become ineligible/hidden and pending deliveries are cancelled by policy; retained history may reappear after unmute/unblock within ordinary retention, without replaying push.
  - Record-based aggregate contributions are not rewritten for unrelated third-party views, while cross-block viewer-relative metrics remain hidden.
- Backwards compatibility:
  - Additive response fields and routes stay within `/v1/`.
  - Older clients may not render controls, so server enforcement is mandatory.
  - Existing public block records are compatible by design.

## 16. UI / API / CLI Impact

- UI:
  - Visitor profiles keep Follow and Share primary; their More menu contains Mute/Unmute, Block/Unblock, and Report.
  - Every non-self ordinary post, project post, comment, and reply More menu contains Mute/Unmute author, Block/Unblock author, and Report post.
  - Block and Unblock use confirmations; Block explains public record visibility and restrictions. Mute/Unmute applies immediately with feedback.
  - Profile header supports muted, blocking, and blocked-by states with policy-appropriate details and actions.
  - Threads and quote previews support revealable muted placeholders and unrevealable blocked placeholders.
  - Settings gains Muted accounts and Blocked accounts list screens.
  - Timeline, discovery, profile tabs, notifications, badges, and deep-link opens react to relationship changes.
  - A non-member never receives a profile, action, suggestion, report target, list row, or tombstone UI; direct lookup renders the standard not-found experience.
- API (exact handler naming may be refined during planning while preserving these contracts):
  - `POST /v1/profiles/{handleOrDid}/mutes` — idempotently mute; return the updated viewer-relative profile relationship.
  - `DELETE /v1/profiles/{handleOrDid}/mutes` — idempotently unmute; return updated relationship.
  - `POST /v1/profiles/{handleOrDid}/blocks` — create public block and synchronously enforce it; return updated relationship.
  - `DELETE /v1/profiles/{handleOrDid}/blocks` — delete the caller-owned public block and return updated relationship.
  - `GET /v1/profiles/me/mutes?limit=&cursor=` — private paginated muted-account list.
  - `GET /v1/profiles/me/blocks?limit=&cursor=` — paginated accounts blocked by the caller.
  - Profile and profile-summary shapes gain camelCase viewer-relative mute/block fields.
  - Post/thread/quote response shaping gains stable muted/blocked placeholder states without protected content.
  - Blocked writes return `interaction_blocked` without a PDS mutation.
  - Any account-targeted endpoint returns `404 profile_not_found` for a resolvable non-member exactly as for an unknown account; collection endpoints omit non-members.
- CLI: None.
- Background jobs:
  - Tap indexes public block lifecycle events.
  - Push dispatcher rechecks relationship eligibility and cancels/suppresses affected deliveries.
  - No scheduled mute expiry job is required.

## 17. Security / Privacy / Permissions

- Authentication: Every mute/block/list endpoint requires the existing Craftsky bearer token and device ID.
- Authorization:
  - Mutations always derive the owner/viewer DID from the authenticated context, never from request JSON.
  - Only the block record's repository owner may create/delete it through Craftsky.
  - A private mute list can only be enumerated by its owning account.
  - Every referenced actor passes one canonical current-membership check before profile hydration, enumeration, reporting, or record creation; unknown and non-member actors share the same not-found response.
- Sensitive data:
  - A mute pair is sensitive private preference data even though both DIDs are independently public.
  - Do not put mute target identifiers in logs, tracing attributes, analytics events, error messages, caches shared across accounts, or metric labels.
  - Public-block confirmation must plainly state that blocks are public.
- Abuse cases:
  - Stale/alternate clients cannot bypass server-side write denial or read filtering.
  - Quote/repost/notification/deep-link indirection cannot reveal protected content.
  - Rapid retry cannot create duplicate block records or inconsistent active states.
  - Rogue atproto clients may still write public records directly, but Craftsky will not deliver or surface policy-violating interaction to the protected account.
  - Logging out, incognito browsing, other accounts, screenshots, and public repository inspection remain outside the guarantee, consistent with Bluesky's public-network limitation.
- Account isolation:
  - Every Flutter provider/cache key and AppView private query includes the active authenticated DID.
  - Switching/signing out one account must not expose or clear another account's private mute state except through that account's own authorized request.
  - Permanent membership removal runs owner-scoped private mute deletion; subject membership removal must not delete another member's private preference.

## 18. Observability

- Events/metrics:
  - Bounded counters/timers for mute/block create/delete outcomes, block indexer outcomes/lag, interaction denials by operation type, notification suppressions, and push cancellations.
  - No target DID, handle, relationship pair, post URI, or record rkey as metric labels.
- Logs:
  - Structured operation, result, failure stage, run/request ID, collection, and bounded error class.
  - Mute logs omit both the target and pair; block indexer logs may identify a malformed public record only at debug/error investigation level under existing sanitization policy, not routine success logs.
- Traces:
  - Show policy-check stage and database/PDS/indexer latency without relationship identifiers.
- Alerts:
  - Sustained block-indexer failures or lag.
  - Elevated PDS block-write failure rate.
  - Push sends that lose the final relationship-eligibility race.
  - No alert is required for ordinary `interaction_blocked` denials.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | A read, write, notification, or push path omits relationship enforcement. | Blocked content or interaction reaches a protected user. | Central policy abstraction, endpoint inventory, integration tests, and explicit indirect-reference tests. |
| RISK-002 | Private mute state leaks through API fields, telemetry, shared cache, or multi-account state. | Privacy breach; muted account may infer the action. | Owner-scoped authorization, account-keyed caches/providers, telemetry rules, and cross-account tests. |
| RISK-003 | PDS write and AppView/Tap state diverge. | Block appears successful but is not enforced, or stale state remains after unblock. | PDS-first canonical writes, synchronous enforcement before success, idempotent reconciliation, and race tests. |
| RISK-004 | Historical or external-client blocks are missed. | Existing safety choices disappear in Craftsky. | Collection subscription plus membership-triggered repository backfill and interoperability tests. |
| RISK-005 | Post-query filtering breaks pagination. | Short/empty pages, duplicates, skips, or side-channel row counts. | Filter in SQL/selection, fetch `limit + 1` eligible rows, and traverse multi-page mixed-policy fixtures. |
| RISK-006 | Notification already leased when relationship changes. | Unwanted push may be sent after mute/block. | Transactional cancellation and final pre-send eligibility check; clearly exclude already-provider-sent pushes from retractability. |
| RISK-007 | Quote/repost/stale-cache hydration bypasses policy. | Protected content is exposed indirectly. | Shape every hydration using viewer policy and revalidate on deep-link open. |
| RISK-008 | Broad query joins regress feed/search performance. | Slow core reads or database load. | Bidirectional indexes, batched/shared predicates, query-plan tests, and no N+1 checks. |
| RISK-009 | Public-block wording is unclear. | Users assume a block is private. | Explicit destructive confirmation and Settings explanation. |
| RISK-010 | “Exactly Bluesky” drifts as Bluesky evolves. | Future semantic mismatch. | Pin this requirements baseline to verified sources/date and treat later upstream changes as a reviewed requirement change. |
| RISK-011 | Membership filtering is applied inconsistently or leaks resolvable non-member identity. | Craftsky exposes accounts outside its intended community or permits orphaned relationships. | One canonical membership predicate, identical not-found behavior, endpoint inventory tests, and query-time list/count filtering. |
| RISK-012 | Membership activation races Tap block backfill or process restart. | A newly joined blocker or blocked account can briefly view or interact despite an existing block. | Persist a fail-closed activation gate; retain current-member-owned blocks whose subject is absent; resume/retry idempotently after restart; and test both record-owner directions across interruption and restart. |
| RISK-013 | Owner and subject lifecycle cleanup are confused. | Private mutes are deleted when a subject leaves or retained after their owner permanently leaves. | Distinct owner/subject storage semantics, explicit membership lifecycle hooks, and deletion/rejoin tests. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | “Mutes and blocks” means individual actor controls, not lists, threads, words, or temporary controls. | Scope would need new models, APIs, policy rules, and UI. |
| ASM-002 | Behavioral parity means Bluesky's verified 2026-07-18 published contract on equivalent Craftsky surfaces, not dependence on Bluesky's hosted private state. | Requirements would need revision if hosted-state synchronization was intended. |
| ASM-003 | Craftsky-private mutes do not need to appear in Bluesky, and Bluesky-hosted private mutes do not automatically appear in Craftsky. | A cross-AppView private-preference protocol/service integration would be required. |
| ASM-004 | Directly requested content from a muted account remains viewable and muted replies/quotes are revealable without changing the global mute. | Mute read shaping and UI placeholders would change substantially. |
| ASM-005 | Across a block, Craftsky should show a minimal annotated profile shell rather than a raw 404 so users can understand, report, reciprocally block, or reverse their own block. | Profile response/error contracts and navigation tests would change. |
| ASM-006 | Existing follow records remain stored and may become effective again after all blocks between the pair are removed. | Block mutation would need destructive follow writes contrary to the verified Bluesky model. |
| ASM-007 | Reports remain available at profile level across a block, but hidden content is not re-exposed solely for reporting. | Report target resolution/UI would need a special exception. |
| ASM-008 | The current authenticated-only Craftsky API surface remains unchanged; anonymous behavior is outside this feature. | Separate public-read policies and tests would be required. |
| ASM-009 | A current `craftsky_profiles` membership row remains the canonical server-side signal that an account is an active Craftsky member. | The membership predicate, activation gate, lifecycle hooks, and all related tests would need to target a different source of truth. |

## 21. Open Questions

- None blocking. Any later request to include moderation lists, mute words/threads, DMs, anonymous reads, or Bluesky-hosted private mute synchronization is a scope change.

## 22. Review Status

Status: Approved with notes

Risk level: High

Review recommended: Required

Reviewer: Codex

Date: 2026-07-19

Notes: Document review is recorded in `03-document-review.md`. DR-001 and DR-002 were incorporated into FR-005, AC-013, AC-059, persistence impact, edge cases, and test-design handoff. Approval permits coding-plan work only; implementation still requires explicit approval.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001–BR-004; FR-001–FR-031; NFR-001–NFR-003; NFR-005–NFR-006; RULE-001–RULE-008.
- Suggested test levels:
  - Unit: relationship policy precedence, request validation, response shaping, cursor encoding, optimistic provider state, localization/semantics.
  - Database: migrations, pair uniqueness, bidirectional indexes, owner deletion versus subject retention, eligible-member pagination, notification/push cancellation.
  - Indexer: create/update/delete/replay/stale ordering, duplicate pairs, retention of current-member-owned blocks targeting absent subjects, joining-owned historical backfill, persisted activation restart/retry, and notification interaction.
  - Handler/integration: every read/write route in the affected inventory, uniform non-member 404/omission, PDS success/failure, immediate pre-Tap enforcement, standard errors.
  - Flutter widget/provider/repository: primary/profile More actions, every post-shaped More menu, confirmations, settings lists, branch-scoped muted reveal, blocked placeholders, retry/rollback, multi-account isolation.
  - End-to-end: two-account mute, one-way/inbound/mutual block, third-party block reference, leave/rejoin, both join-time record-owner directions, activation interruption/process restart, owner deletion, block/unblock races, deep links, notification restoration, push eligibility, and existing-record restoration.
  - Performance/security: query plans/no N+1, pagination under dense filtering, cross-account authorization, telemetry privacy.
- Blocking open questions: None.
- Exit gate: High-risk requirements require explicit user approval before `write-acceptance-tests` or implementation.
