# Instagram DM Ownership Verification and Follow Discovery — Design Plan

**Date:** 2026-07-11  
**Status:** Approved for implementation; production enablement blocked on external gates  
**Risk:** High — external identity linking, private social-graph data, and account-discovery controls  
**Scope:** Custom Instagram Messaging API integration, Instagram handle ownership verification, and verified-handle matching for follow suggestions

## 1. Summary

CraftSky will use direct messages to its official Instagram professional account to verify that a signed-in CraftSky member controls a particular Instagram account. The AppView will issue a short-lived, single-use challenge; the member will send that challenge to CraftSky on Instagram; and a custom Meta webhook integration will associate the Instagram-scoped sender ID and current username with the authenticated CraftSky DID after an explicit confirmation in CraftSky.

Verified mappings will allow CraftSky to match handles from a member's Instagram data export or manually entered list against CraftSky accounts with high confidence. Matching produces reviewable follow suggestions. It does not follow anyone automatically in the initial version.

The integration will be implemented directly against Meta's Instagram Messaging API. ManyChat is explicitly not part of the design.

Instagram export parsing will run on the member's device. The raw JSON export will never be uploaded to AppView. This deliberately accepts slower client-update cycles when Meta changes its export shape in exchange for stronger privacy.

## 2. Product Goal

Help people moving from Instagram find the people they already chose to follow, without scraping Instagram, requiring those people to have professional Instagram accounts, or treating an unverified same-name profile as an identity match.

Success means:

- A member with a personal or professional Instagram account can prove control by sending a DM from that account.
- CraftSky retains a stable Instagram-scoped identifier as the identity anchor and treats the mutable Instagram username as an attribute.
- Imported or entered Instagram handles produce suggestions only when they match a valid, discoverable verified mapping.
- Instagram JSON exports are parsed on-device and AppView receives only the minimal normalized handles needed for matching.
- The member reviews suggestions before CraftSky writes public `app.bsky.graph.follow` records to their PDS.
- Raw Instagram exports, unmatched handles, challenges, and discovery preferences remain private AppView data.

## 3. Confirmed Direction

### Selected approach: direct Meta webhook integration

CraftSky will own the complete verification path:

```text
Flutter/Web client
    -> authenticated AppView challenge endpoint
    -> Instagram DM to @craftsky
    -> Meta Instagram messaging webhook
    -> AppView candidate verification
    -> authenticated in-app confirmation
    -> verified, discoverable handle mapping
    -> imported-handle matching
    -> reviewable follow suggestions
```

This approach avoids a third-party automation processor, per-contact pricing, duplicated contact retention, and a critical dependency on a vendor-specific contact model. It costs more engineering effort than ManyChat but keeps security-sensitive state, matching rules, retention, and auditability under CraftSky's control.

### Alternatives considered

#### ManyChat adapter

ManyChat could receive the DM, invoke AppView through an external request, and send the reply. It is useful for a prototype but adds a data processor, contact-based billing, a vendor dependency, and uncertainty around direct access to the raw Instagram-scoped sender ID. It is not selected.

#### Instagram bio challenge

A member could temporarily place a challenge or CraftSky link in their public Instagram bio. This is more awkward, depends on reliably reading Instagram profile pages, and risks reliance on unsupported scraping. It may remain a manual recovery method but is not the primary flow.

#### Export possession as verification

Possession of an Instagram export is evidence of access but is weaker than a live message: archives can be copied, retained after account loss, or contain stale usernames. Exports are an input to graph matching, not proof of current handle ownership.

## 4. Actors and Trust Boundaries

| Actor | Role | Trust treatment |
|---|---|---|
| CraftSky member | Authenticated DID initiating verification or importing handles | Trusted only after normal CraftSky session authentication |
| Instagram sender | Account sending the challenge DM | Identified by Meta's Instagram-scoped ID; not yet linked to a DID |
| Meta | Delivers signed webhook events and profile data | External dependency; validate signatures and minimize retained data |
| AppView | Issues challenges, validates events, stores private mappings, and generates suggestions | System of record for verification and matching |
| PDS | Stores user-approved follows | Receives only explicit follow writes; never receives imported graph or verification data |

The DM is strong evidence that the sender controls the Instagram account at verification time. It does not by itself prove that the sender wants the Instagram-to-CraftSky association exposed for discovery. Verification and discoverability are therefore separate states.

## 5. User Journeys

### 5.1 Verify an Instagram account

1. A signed-in member opens **Find people from Instagram** and chooses **Verify my Instagram**.
2. CraftSky explains that verification will let people who know the Instagram handle find the member on CraftSky. The member explicitly enables or declines discoverability.
3. AppView creates a short-lived challenge bound to the member's DID and returns the formatted challenge plus the official CraftSky Instagram DM link.
4. The app copies the challenge and opens the CraftSky Instagram conversation.
5. The member pastes and sends the challenge.
6. Meta sends a signed messaging webhook containing the message and Instagram-scoped sender ID.
7. AppView validates the webhook, hashes and matches the challenge, fetches the sender's current username through Meta if it is not present in the event, and records a pending candidate.
8. The app polls or refreshes verification status and displays: **We found @handle. Is this your Instagram account?**
9. The member confirms in their existing authenticated CraftSky session.
10. AppView creates or updates the verified mapping and optionally sends an immediate confirmation reply within Meta's allowed messaging window.

The association is not finalized solely from the DM. The in-app confirmation prevents a forwarded, intercepted, or accidentally reused challenge from silently binding the wrong Instagram account.

### 5.2 Import an Instagram following list

1. A signed-in member selects an Instagram JSON export or supplies a text list of handles.
2. The client parses the export on-device. Uploading the raw JSON or ZIP to AppView is prohibited.
3. The client sends only normalized handles, relationship direction, and minimal import metadata to AppView.
4. AppView matches the imported **following** handles against active, discoverable verified mappings.
5. The app shows matched CraftSky profiles as follow suggestions with a clear reason, for example: **You followed @aliceknits on Instagram.**
6. The member selects individual suggestions or explicitly selects all reviewed suggestions.
7. Each confirmed follow uses the existing CraftSky follow-write path and creates a public `app.bsky.graph.follow` record on the member's PDS.
8. Unmatched handles may be retained privately for future matching only with explicit consent, for up to 12 months.

### 5.3 Future match

When a previously unmatched handle becomes verified and discoverable:

1. AppView creates a new pending suggestion for each eligible importer who retained that handle.
2. The suggestion appears in **Find people from Instagram**.
3. AppView creates an `instagramMatch` notification so the importer knows that a person from their retained Instagram following list is now available to follow.
4. No follow is created until the importer approves it.

`instagramMatch` is a new first-class notification category and must be added to the notification requirements, preference model, API, persistence, Flutter rendering, navigation, and push-copy mapping. It must not be represented as `everythingElse`.

This is a system-generated match notification rather than a social action performed by the matched person. Its eligibility comes from the recipient's retained Instagram following list and the matched person's discoverability consent. The ordinary `Everyone` versus `People I follow` actor scope is therefore not meaningful. The notification-design update should treat `instagramMatch` as always eligibility-filtered by the migration rules while still allowing the recipient to disable push delivery for the category. A single digest notification should be preferred when several matches become available together.

## 6. Verification Challenge Design

Challenges must be:

- Generated with a cryptographically secure random source.
- At least 60 bits of entropy after formatting.
- Human-pasteable, case-insensitive, and free of ambiguous characters.
- Prefixed so the Instagram automation can distinguish verification messages from ordinary DMs.
- Single use and valid for approximately ten minutes.
- Bound to one CraftSky DID and one verification attempt.
- Stored as a hash, never as recoverable plaintext.
- Invalidated when redeemed, cancelled, superseded, or expired.

The canonical display form uses thirteen random symbols drawn uniformly from
`23456789ABCDEFGHJKMNPQRSTVWXYZ` (a 30-symbol alphabet that omits `0`, `1`,
`I`, `L`, `O`, and `U`). Thirteen symbols provide approximately 63.8 bits of
entropy (`13 * log2(30)`). Hyphens and the `CSKY-` prefix are formatting and do
not count toward entropy. Verification messages accept only this complete token
after trimming outer whitespace and folding ASCII case; surrounding prose is
not accepted.

Example display form:

```text
CSKY-7K4P-N9QX-M2RT-H
```

The challenge must not encode the DID, session token, Instagram handle, email address, or other personal data.

Rate limits must apply by source IP for challenge creation, authenticated DID, Instagram-scoped sender ID, and global redemption volume. Invalid DMs should receive either no response or a generic response that does not reveal whether a guessed challenge exists.

## 7. Meta Integration

### 7.1 Meta configuration

- Create or use the official CraftSky Instagram professional account.
- Create a Meta Business app owned by CraftSky.
- Configure Instagram API with Instagram Login.
- Request `instagram_business_basic` and `instagram_business_manage_messages` for the owned CraftSky Instagram account.
- Add the CraftSky Instagram account in the Meta App Dashboard.
- Configure a public HTTPS webhook and subscribe to the required Instagram messaging events.
- Start with Standard Access because CraftSky manages only its own professional account.
- Confirm in a production spike that a non-role personal Instagram account can trigger the webhook and profile lookup when the app is live.
- Complete any dashboard requirements for Live mode, privacy policy, data deletion, business portfolio, and business verification.

### 7.2 Webhook handling

The AppView integration endpoint is an external callback, not a Flutter-facing `/v1/*` route. A proposed shape is:

```text
GET  /integrations/instagram/webhook   # Meta verification challenge
POST /integrations/instagram/webhook   # Signed event delivery
```

The handler must:

- Validate Meta's webhook verification challenge during setup.
- Verify the signature over the raw request body before JSON processing.
- Apply a strict body-size limit and reject unsupported event types.
- Deduplicate deliveries using the Meta message/event ID.
- Acknowledge valid events quickly and avoid slow work on the webhook request path.
- Redact message contents, tokens, access tokens, usernames, and scoped IDs from logs.
- Treat webhook delivery as at-least-once.
- Ignore non-verification conversations without persisting their contents.

If profile lookup is required, a worker should use the sender IGSID and the official account's access token to request only the fields needed for verification, principally `username`. Meta access tokens belong in server-side secret storage and must never be returned to the client.

### 7.3 DM replies

CraftSky may reply immediately with one of a small number of messages:

- Challenge accepted; return to CraftSky to confirm.
- Challenge expired; request a new challenge in CraftSky.
- Challenge invalid; return to CraftSky and try again.
- Verification completed.

Replies must stay within Meta's permitted messaging window and must not be used for later follow-suggestion notifications or marketing.

## 8. AppView API Plan

Proposed authenticated client endpoints, all using the existing `/v1/` authentication, device ID, camelCase JSON, and error-envelope conventions:

```text
POST   /v1/migrations/instagram/verifications
GET    /v1/migrations/instagram/verifications/{verificationId}
POST   /v1/migrations/instagram/verifications/{verificationId}/confirm
DELETE /v1/migrations/instagram/verifications/{verificationId}

POST   /v1/migrations/instagram/imports
GET    /v1/migrations/instagram/suggestions
DELETE /v1/migrations/instagram/imports/{importId}
PATCH  /v1/migrations/instagram/settings
```

The exact route grouping should be reviewed during requirements and coding design. The important boundary is that client operations are authenticated `/v1/*` routes while Meta callbacks are separately authenticated integration routes.

No lexicon change is needed. Verification, discovery consent, imports, unmatched handles, and suggestions are private AppView state. Only a user-approved follow uses the existing public AT Protocol follow lexicon.

## 9. Private Data Model

The following conceptual tables keep verification and migration state separate:

### `instagram_verification_attempts`

- Opaque verification ID.
- CraftSky DID.
- Challenge hash.
- Expiry and redemption timestamps.
- Pending Instagram-scoped sender ID and username.
- State: `pending_dm`, `pending_confirmation`, `confirmed`, `expired`, `cancelled`, or `rejected`.
- Creation and update timestamps.

### `instagram_account_links`

- Instagram-scoped ID as the identity anchor within CraftSky's Meta integration.
- Current normalized username and last-observed display form.
- CraftSky DID.
- Verification method and timestamps.
- Discoverability consent and its update timestamp.
- State: `active`, `superseded`, `revoked`, or `disputed`.
- Last successful re-verification timestamp.

The relationship is strictly one-to-one: one CraftSky DID may have only one active Instagram link, and one Instagram-scoped ID may belong to only one CraftSky DID. A normalized username can also have only one active discoverable mapping. Collisions must enter a dispute/re-verification path rather than silently transferring ownership.

### `instagram_graph_imports`

- Import ID and importing DID.
- Source type and import timestamp.
- Whether unmatched handles may be retained for future matching.
- No raw archive payload.

### `instagram_graph_handles`

- Import ID.
- Normalized Instagram username.
- Direction: `following` or `follower`.
- Match state and matched DID when applicable.
- Retention expiry.

### `instagram_follow_suggestions`

- Importing DID and suggested DID.
- Source normalized username and relationship direction.
- Source link verification ID/version.
- State: `pending`, `accepted`, `dismissed`, `invalidated`, or `alreadyFollowing`.
- Created, updated, and acted-on timestamps.

Do not store raw Instagram export ZIPs, photos, biographies, profile pictures, follower counts, or full webhook message histories.

## 10. Matching Semantics

### High-confidence match

A suggestion is high confidence only when all of the following are true:

- The imported normalized handle exactly matches the current normalized username on an active Instagram link.
- The link was established through the DM verification flow.
- The linked CraftSky member has explicitly enabled Instagram-handle discoverability.
- The mapping is not expired, revoked, disputed, or superseded.
- The suggested DID is not the importer and is not already followed.

### Direction matters

- `following`: evidence that the importing member chose to follow the Instagram account; eligible for ordinary follow suggestions.
- `follower`: evidence only that the Instagram account followed the importing member; shown separately, if at all, and never treated as equivalent intent.

The initial version should base proactive suggestions and future-match retention on `following`. Supporting follower-derived discovery should require a separate product decision and clear UI.

### Handle changes

The IGSID is the retained identity anchor; the username is mutable. When Meta reports a new username for an existing IGSID:

- Update the current username after validation.
- Invalidate suggestions created solely from the old handle if they have not been accepted.
- Do not transfer the old username's verified status to a new IGSID.
- Require re-verification or a conflict workflow before another DID can claim a recently released username.

Accepted PDS follows remain follows of a DID and are not undone by later Instagram username changes.

## 11. Privacy and User Control

- Verification does not automatically enable discoverability; discovery is an explicit opt-in.
- The member must understand that enabling discovery lets another CraftSky member who supplies the verified Instagram handle find their CraftSky profile.
- A member can disable discovery, revoke the Instagram link, or re-verify a changed account.
- Revocation invalidates pending suggestions but does not undo follows that users already approved.
- Loss of current `craftsky_profiles` membership and terminal account deletion
  are different events. Membership loss immediately disables discovery and all
  member-facing Instagram operations, invalidates dependent pending
  suggestions/notifications, and retains private owner state only under its
  normal retention policy. Rejoining does not silently re-enable discovery.
  A terminal atproto identity-deletion event or a future explicit whole-account
  deletion permanently purges the member's private Instagram state.
- Importers must explicitly opt in before unmatched handles are retained for future matching.
- Unmatched imported handles may be retained for at most 12 months and must have a delete-now control. The member must renew consent before retention extends beyond that period.
- A verified member is not told which specific people imported or searched for their handle.
- Importers should not be shown verification timestamps, IGSIDs, or other private link metadata.
- Matching and imported graph data remain in AppView Postgres and are never written to a PDS.
- Data-export and account-deletion flows must include these private records.

## 12. Security and Abuse Controls

- Require a valid CraftSky session to create, inspect, confirm, or cancel an attempt.
- Require confirmation by the same DID that created the attempt; do not rely on a browser session identifier alone.
- Verify Meta webhook signatures using the raw body and current app secret.
- Keep Meta tokens and app secrets in production secret storage.
- Hash challenges and use constant-time comparison where applicable.
- Deduplicate webhook events and make redemption idempotent.
- Prevent a redeemed token from being moved to a different Instagram sender.
- Prevent one active Instagram identity from linking to multiple CraftSky DIDs.
- Notify or visibly warn both affected CraftSky sessions when a conflicting claim is attempted, without revealing unnecessary identity information.
- Provide operator tooling for revocation and disputes with an audit trail.
- Rate-limit challenge creation, invalid redemption, profile lookup, confirmation, and import matching.
- Treat usernames and message text as untrusted input throughout logging, rendering, and persistence.
- Monitor verification success, expiry, conflict, replay, signature-failure, and Meta API error rates without recording challenge or message contents.

## 13. Failure Behaviour

| Failure | User-visible behaviour | System behaviour |
|---|---|---|
| Expired challenge | Ask the member to create a new challenge | Mark attempt expired; never reuse it |
| Invalid DM text | Generic retry guidance | Do not reveal whether another challenge exists |
| Duplicate webhook | No duplicate reply or link | Return success after idempotent lookup |
| Meta profile lookup unavailable | Show verification still processing | Retry within a bounded window |
| Candidate handle differs from expectation | Show the actual handle and require confirmation | Do not finalize automatically |
| IGSID already linked to another DID | Explain that the account is already linked and offer recovery | Record conflict; do not reassign |
| Username already mapped to another IGSID | Temporarily block discovery for that handle | Require re-verification/dispute resolution |
| User revokes before confirming | Show cancelled | Delete or expire the pending candidate |
| Instagram integration outage | Keep import and CraftSky available; disable new verification gracefully | Alert operators; do not lose existing mappings |

## 14. Observability

Track aggregate metrics for:

- Challenges issued, redeemed, expired, cancelled, and confirmed.
- Time from challenge creation to DM and from DM to confirmation.
- Invalid-token and rate-limit counts.
- Webhook signature failures, duplicates, processing latency, and queue depth.
- Meta profile lookup and reply success/failure rates.
- Link conflicts, revocations, username changes, and disputes.
- Import size, exact-match rate, suggestion acceptance, dismissal, and invalidation.

Logs and telemetry must not contain challenge plaintext, raw message text, Meta access tokens, raw export contents, or complete imported handle lists.

## 15. Delivery Phases

### Phase 0: Meta capability spike

- Configure a development Meta app and owned CraftSky professional Instagram account.
- Receive a signed webhook from an unrelated personal Instagram account.
- Confirm that the webhook yields an IGSID and that profile lookup returns the current username.
- Confirm Standard Access versus Live-mode and business-verification requirements.
- Confirm token lifetime, renewal, webhook subscription, and app-review requirements in the actual dashboard.
- Record the result before committing to production implementation.

### Phase 1: Secure ownership verification

- Challenge issuance and cancellation.
- Signed webhook verification and event deduplication.
- Candidate profile lookup.
- Authenticated in-app confirmation.
- Link revocation and discoverability consent.
- Focused security, replay, conflict, and expiry tests.

### Phase 2: Manual-handle matching

- Accept a pasted list of Instagram handles.
- Normalize and match only against active discoverable verified mappings.
- Present reviewable suggestions.
- Use the existing follow-write path after explicit approval.

### Phase 3: Instagram export import

- Support selected known JSON export shapes through versioned, tolerant client-side parsers.
- Parse exclusively on-device; the client must never upload the raw JSON, ZIP, or unrelated export fields.
- Treat required client releases after Meta export-shape changes as an accepted privacy trade-off rather than adding server-side raw-export processing.
- Separate `following` from `follower` relationships.
- Add import deletion and retention controls.

### Phase 4: Future-match suggestions

- Retain unmatched following handles only with consent.
- Create suggestions when matching discoverable links appear.
- Add the first-class `instagramMatch` notification category and prefer a digest when multiple matches are created together.

### Phase 5: Hardening and recovery

- Username-change refresh.
- Account-link dispute and recovery flow.
- Operator tooling and audit history.
- Periodic re-verification policy based on observed abuse and Meta identifier behaviour.

## 16. Testing Strategy

- Unit-test challenge generation, hashing, expiry, normalization, matching, and state transitions.
- Verify that challenges have the required entropy and cannot be redeemed twice.
- Test webhook signature validation against exact raw request bytes.
- Test replayed and out-of-order webhook events.
- Test concurrent redemption and confirmation attempts.
- Test IGSID and normalized-username uniqueness conflicts.
- Contract-test Meta webhook fixtures and profile responses with secrets removed.
- Integration-test authenticated `/v1/*` routes using the standard CraftSky error envelope.
- Test that export parsing runs on-device and that no client request can upload raw archive contents.
- Test supported, unsupported, partially changed, and malformed Instagram export shapes without sending their raw contents to AppView.
- Test discoverability disabled, revoked, expired, and disputed mappings.
- Test that follower-only imports do not become ordinary follow suggestions.
- Test that no PDS follow is written until explicit member approval.
- Test deletion cascades and account deletion/export coverage.
- Perform an end-to-end sandbox test using an unrelated personal Instagram account before release.

## 17. Non-Goals

- Reading follower or following lists through the Instagram API.
- Scraping Instagram profiles or follower pages.
- Treating Instagram OAuth for professional accounts as the only verification path.
- Using possession of an Instagram export as equivalent to live ownership verification.
- Importing posts, photos, videos, messages, likes, comments, or saved collections.
- Automatically following verified matches in the initial version.
- Sending marketing DMs or follow-suggestion DMs through Instagram.
- Publishing Instagram mappings as AT Protocol records.
- Making imported or unmatched handles visible to other members.
- Using ManyChat or another automation SaaS in the production path.
- Server-side parsing or storage of Instagram JSON exports or ZIP archives.

## 18. Settled Product Decisions

1. Discoverability is an explicit opt-in after ownership verification.
2. Unmatched imported handles may be retained for 12 months, with delete-now and renewal controls.
3. Verification is re-checked opportunistically. Re-verification is required only after conflict, revocation, or evidence of an account or username change.
4. Instagram links are one-to-one: one Instagram account per CraftSky DID and one CraftSky DID per Instagram account.
5. Future matches use a new `instagramMatch` notification category.
6. A link conflict should be exceptional. Recovery starts with an email to CraftSky support and is handled manually; email does not automatically authorize reassignment. The existing link remains in place until an operator has enough evidence to resolve the conflict, and the system must never silently transfer it.
7. Instagram JSON exports are parsed exclusively on-device. Supporting changed export shapes through client updates is an accepted privacy trade-off.

## 19. Implementation Readiness

This high-risk feature was not taken directly from this design into coding. The
user explicitly approved formalization and feasible implementation, and the
required workflow artifacts now provide the authoritative implementation
contract:

1. `01-requirements.md`, approved requirements and exact subcontracts.
2. `02-acceptance-tests.md`, privacy, replay, conflict, lifecycle, and follow-write coverage.
3. `03-document-review.md`, approved independent re-review.
4. `04-coding-plan.md`, created before implementation and covering migrations,
   integration secrets, background processing, routes, Flutter state/UI, and
   external release gates.

The requirements and acceptance-test documents supersede earlier sketch-level
details in this plan where they are more precise. Production enablement still
requires the Meta capability spike and every manual release gate.
