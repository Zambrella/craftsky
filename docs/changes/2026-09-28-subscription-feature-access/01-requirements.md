# Requirements: Subscription Feature Access

## 1. Initial Request

Finish subscription feature allocation and access control. Plus (indicatively ~£2.49/month) includes scheduled posts, saved-post folders, all currently supported profile customisation, pinned posts, and follower growth metrics. Business (indicatively ~£8.99/month) includes all Plus benefits and featured profile products, a profile call-to-action, business information, and upcoming events. Locked Plus entry points show an overlaid icon; tapping them explains the subscription requirement and offers a CTA to the subscription page. The product owner clarified that an effective Business licence determines business account type; the testing-only business account toggle, its mutation API and unused account-type table must be removed. Business options and owner-management routes are unavailable without Business access. Business records remain stored on the user's PDS but hidden from CraftSky surfaces until access returns.

## 2. Current Codebase Findings

- Relevant files: `docs/changes/2026-09-07-account-subscriptions/00-direction.md`, `01-requirements.md`, `adr/014-account-assigned-subscription-licenses.md`, `docs/changes/2026-09-10-flutter-subscriptions/01-requirements.md`, `app/lib/subscriptions/models/subscription_access.dart`, `app/lib/subscriptions/providers/subscription_access_provider.dart`, `app/lib/subscriptions/pages/subscription_page.dart`, `appview/internal/api/subscriptions.go`; feature precedents in `docs/changes/2026-08-04-pinned-profile-posts/`, `2026-08-09-profile-customisation/`, `2026-08-25-follower-growth-metrics/`, and `2026-08-27-business-profiles/`.
- Existing patterns: AppView derives account-effective `free`/`plus`/`business` from an assigned licence; Flutter reads access per account/session. Subscriptions are purchased and explicitly assigned separately. The subscription page is routed under Settings at `/profile/settings/subscriptions`. Existing feature writes and reads use authenticated `/v1/` APIs.
- Current behavior: Subscriptions and feature implementations exist, but paid-feature enforcement was deliberately deferred. Pins currently allow every member; business account classification, declaration, and event writes currently do not require payment. Business content is served based on membership and self-declared account type. Saved posts and their folder UI, scheduled posts, profile customisation, follower growth, and business profile UI already exist.
- Constraints discovered: The free social experience, chronological visibility, and no-paid-reach policy remain intact. The independently writable business account type and toggle were temporary testing behavior; replace their storage/read paths with subscription-derived classification and drop the now-unused account-type table through a new migration. Public business declarations and events are PDS records: CraftSky can suppress them from its AppView/API/UI, but cannot make records on a user's PDS private or delete them for subscription expiry. Private subscription/assignment data stays in AppView. Existing owner/membership/moderation checks remain applicable.
- Test/build commands discovered: `just test`, `just appview-check`, `just app-test`, `just app-analyze` (for a later implementation stage; no tests run in this documentation stage).

## 3. Clarifying Questions And Decisions

### Q1: Which tier gets follower growth?

Answer: Plus, by explicit amendment to the initial list; Business inherits it.

Decision / implication: Do not leave follower growth Business-only as suggested in the earlier subscription direction.

### Q2: What happens to business content without a Business licence?

Answer: Hide until subscribed.

Decision / implication: Retain authored records, but suppress Business-only details, products, CTA, and events from CraftSky public profile and event surfaces while the owner lacks effective Business access. This cannot hide independently accessible PDS records.

### Q3: Which profile customisation choices count as enhanced?

Answer: All currently supported customisations are behind Plus.

Decision / implication: The currently supported profile colour and background selections and presentation require Plus or Business. The previously removed border selector is not part of this change. Without access, render existing standard defaults while retaining saved selections for restoration.

### Q4: Who may set business account type?

Answer: An account holding an effective Business subscription is marked as business; remove the unrestricted manual toggle completely, including its mutation route.

Decision / implication: CraftSky account type follows the assigned DID's effective tier, not a separate owner-set flag. Free and Plus accounts are regular; when Business access ends the effective account type returns to regular, and restoration makes it business again. Remove the account-type toggle from Flutter, unregister its mutation API, replace every live reader of the testing-only account-type table, and drop that table in a forward migration. No member, including a Business subscriber, can override the derived value.

### Q5: What happens to existing paid data after access ends?

Answer: Saved posts remain visible as a flat list, including posts previously in folders; folder memberships are retained and return on resubscription. Pins are removed on unsubscribe. Business records remain but are not shown. Scheduled items show a relevant error.

Decision / implication: Treat “unsubscribe” as loss of effective paid access, not merely cancellation of renewal while the current paid period still grants access. Remove active pin selections on loss of Plus-or-higher access; previous pin choices are not restored. Preserve business source records but suppress their serving. Retain scheduled items; show a subscription-required notice on still-future items while access is absent and a subscription-required error on items blocked when due.

### Q6: What happens to future scheduled items on resubscription?

Answer: Whichever is easier to implement.

Decision / implication: Use the existing scheduled/needs-attention lifecycle: a future item still pending when access is restored may publish at its scheduled time; an item whose due time passed without access moves to needs-attention with a subscription-required error and never auto-publishes after restoration. This is a recommended implementation-compatible choice, not an additional user-imposed policy for every possible scheduling implementation.

### Q7: What happens to Business options and owner routes without a valid Business licence?

Answer: Hide all Business-related actions and options, including settings menu entries and Business management routes, when the account lacks effective Business access. Drop unused tables such as the testing-only account-type table.

Decision / implication: Free/Plus members see no Business settings section or editor controls and cannot enter Business owner-management pages by deep link. Direct owner-management HTTP reads and writes require effective Business access; public profile/event reads remain available to viewers when the **owner** has Business access and existing visibility rules allow them. Retained declaration/event source records stay on the PDS and become eligible again upon restoration; no owner recovery UI or owner-management access is promised during a lapse. The normal subscription page remains reachable through Settings.

## 4. Candidate Approaches

### Option A: Account-scoped, AppView-enforced feature access with Flutter upgrade prompts

Summary: Use the already implemented effective tier of the authenticated DID for server-backed authorization and CraftSky public serving. Flutter presents locked Plus controls and guides members to the existing subscription page.

Pros: Matches licence assignment, protects direct API access, and handles another account on the same device independently.

Cons: Touches several existing feature paths and public business projections.

Risks: Read/write mismatches or stale client state could expose controls or hide retained content incorrectly.

### Option B: Flutter-only locks and paywall messaging

Summary: Hide or intercept controls without enforcing access in AppView.

Pros: Smaller UI change.

Cons: Existing API callers can still use paid capabilities; public serving cannot reliably reflect a lapsed licence.

Risks: Paid access is bypassable and differs across clients.

## 5. Recommended Direction

Recommended approach: Option A. This recommendation follows the prior account-subscription direction and the user's tier and visibility decisions; no separate discovery confirmation was supplied for this feature-access slice.

Why: The AppView is already the authority for effective DID access. The client should explain locked features, while server-backed decisions must follow the assigned account rather than the billing owner or device.

## 6. Problem / Opportunity

Members can currently use capabilities intended for paid tiers regardless of the assigned licence. The product needs a consistent tier boundary, a discoverable upgrade path, and predictable handling of content created before a licence lapses.

## 7. Goals

- G-001: Assign existing capabilities to Free, Plus, and Business without changing the base social experience.
- G-002: Enforce account-specific access even through direct API calls and across account switches.
- G-003: Explain locked Plus capabilities at their entry points and guide members to subscriptions.
- G-004: Preserve authored data where specified while removing active pins and suppressing lapsed paid-only presentation.
- G-005: Make effective Business access the sole basis for business account type.

## 8. Non-Goals

- NG-001: Change subscription prices, store products, RevenueCat configuration, purchase/restore/assignment flows, or billing periods. The amounts in the request are indicative, not a price-setting instruction.
- NG-002: Add paid reach, feed/search priority, ads, or new limits for free social activity.
- NG-003: Introduce a separately purchasable business-label-only tier, retain a manual business account-type toggle or keep unused testing-only account-type storage.
- NG-004: Delete or conceal records on a user's PDS, alter lexicons, or remove previously published ordinary posts because a licence lapses.
- NG-005: Apply a Plus-style icon overlay to every Business-section control.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Free member | Authenticated DID without effective paid access. | Understand locked capabilities and retain ordinary social use. |
| Plus member | DID assigned an active Plus licence. | Use Plus features without Business-only access. |
| Business member | DID assigned an active Business licence. | Use Plus and Business features. |
| Profile visitor | Member viewing someone else's public profile or events. | See only content currently eligible for CraftSky serving. |
| Billing owner | Purchaser who assigns a licence to a DID. | Existing purchase and assignment path remains available. |

## 10. Current Behavior

Implemented feature flows do not yet consult account-effective subscription access. Business account type and PDS records currently drive business serving without a paid check. The subscription UI already supports account-scoped access and purchasing/assignment separately.

## 11. Desired Behavior

An authenticated account gets the capabilities of its effective tier: Business includes Plus; Plus does not include Business. All current profile customisation is Plus-only. Free members see discoverable locked Plus entry points and a feature-specific explanation with a route to subscriptions. Business-only settings options and owner-management routes are available only to an account with effective Business access; Free/Plus members do not see a Business section. Server-backed actions reject unauthorized direct requests. An effective Business licence marks the assigned DID as business; otherwise it is regular, without a manual toggle or legacy account-type table. When access ends, saved posts become a flat list while folder associations remain stored, active pins are removed, customisation renders defaults while saved selections remain stored, blocked scheduled items show an error, and Business-only content is suppressed from CraftSky serving until Business access returns.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Plus shall provide scheduled posts, saved-post folders, all currently supported profile customisation, pinned posts, and follower growth. | Fixes the lower paid tier, including the follower-growth amendment and full customisation scope. | Prompt; annotation Q3; document review DR-001 | AC-001, AC-002, AC-003, AC-015 |
| BR-002 | Business | Must | Business shall provide every Plus benefit plus featured products, profile CTA, business information, and upcoming events. | Establishes the upper tier without double-paying for inherited benefits. | Prompt | AC-001, AC-004 |
| FR-001 | Functional | Must | CraftSky shall base paid feature access on the authenticated DID's current AppView effective tier, not the device, billing owner, independently mutable account-type flag, or RevenueCat customer-wide entitlements. | Prevents access leaking across retained accounts or billing relationships. | Subscription direction; codebase | AC-001, AC-005 |
| FR-002 | Functional | Must | AppView shall deny creation, updates, and execution of Plus-only capabilities without effective Plus or Business access, including scheduled publication, folder management, pin mutations, and profile-customisation mutations; follower-growth data shall be owner-only and accessible only with Plus or Business. | UI-only gates are bypassable. | Prompt; annotation Q3; existing feature/API contracts | AC-002, AC-003, AC-006, AC-015 |
| FR-003 | Functional | Must | Free members shall see an icon overlaid on Plus-only entry points; activating one shall show a feature-specific subscription-required dialog with a CTA opening the existing subscription page. Members with Plus or Business access shall use those entry points normally. | Makes paid capability discoverable without silently failing. | Prompt; existing subscription route | AC-007, AC-008 |
| FR-004 | Functional | Must | Business-only settings options and management pages shall be absent for accounts without effective Business access; direct Business owner-management API reads and mutations shall be denied. Only Business accounts may see and use the Business section. These controls do not require Plus-style icon overlays. | Keeps the Business section exclusive to licensed accounts, including direct routes. | Prompt; user answers Q7; codebase | AC-004, AC-009 |
| FR-005 | Functional | Must | CraftSky profile and event responses shall not serve Business-only details, products, CTA, or events for an owner without current Business access, including the owner's CraftSky views; they shall become eligible again after access is restored, subject to existing membership, moderation, and visibility rules. Public reads of a licensed business shall not depend on the viewer's tier. | Implements hide-until-subscribed without changing PDS records or barring public viewing of licensed businesses. | User answers Q2/Q7; business-profile precedent | AC-010, AC-011 |
| FR-006 | Functional | Must | When Plus-or-higher access ends, saved posts including those in folders shall remain accessible in one flat saved-post list; folder definitions and memberships shall remain stored and reappear with posts in their folders on restored access. All active profile pin selections shall be removed on loss of access and shall not return automatically. Saved profile customisation shall be retained but public presentation shall use standard defaults without access. | Specifies per-feature lapse behavior without destroying saved-post organization or customisation. | Annotation Q3/Q5 | AC-015, AC-016, AC-017 |
| FR-007 | Functional | Must | Pending scheduled items shall remain visible to their owner with a subscription-required notice while Plus-or-higher access is absent. An item due without access shall not publish and shall show a subscription-required error; it shall not auto-publish if access later returns. Items still scheduled in the future when access returns may execute at their planned time. Previously published ordinary posts remain subject to existing visibility rules. | Makes the scheduling lapse visible without backfilling missed posts. | Annotation Q5/Q6; existing scheduled-post states | AC-012, AC-018 |
| FR-008 | Functional | Must | The effective `business` account type shall be derived from the assigned DID's effective Business access; without that access the account shall be `regular`. CraftSky shall remove the manual toggle and mutation API for every tier, replace all live uses of the testing-only `craftsky_account_types` table, and drop that unused table in a forward migration without deleting PDS business records. | Replaces the temporary testing switch and unused state with purchased business identity. | User answers Q4/Q7 | AC-004, AC-011, AC-019 |
| NFR-001 | Non-functional | Must | Locked controls and dialogs shall be usable with screen readers and keyboard/focus input; the icon alone shall not convey the restriction. | Explains access regardless of visual ability. | Existing UI accessibility patterns; prompt | AC-013 |
| NFR-002 | Non-functional | Must | Subscription checks shall preserve existing account-owner boundaries and existing chronological feed/search/discovery behavior. | Paid access must not buy distribution. | `00-direction.md`; business/pin decisions | AC-005, AC-014 |
| RULE-001 | Business rule | Must | Free &lt; Plus &lt; Business for feature access; follower growth is Plus, not Business-only. Effective business account type follows Business access and cannot be separately self-declared. | Prevents drift between public identity and purchased access. | Prompt amendment; annotation Q4 | AC-001, AC-004, AC-019 |
| RULE-002 | Business rule | Must | A subscription lapse shall not delete public PDS records or make those source records private; hide-until-subscribed applies only to CraftSky-controlled serving. | PDS ownership and public-record semantics remain intact. | User answer; architecture rules | AC-011 |
| RULE-003 | Business rule | Must | Cancellation of future renewal alone shall not remove tier access, business classification, or pins while the assigned licence still gives access; the lapse behaviors start when effective access actually ends. | Distinguishes cancellation from expiry/grace-period access. | Existing subscription lifecycle; interpretation of annotation Q5 | AC-020 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, BR-002, FR-001, RULE-001 | Given distinct authenticated Free, Plus, and Business DIDs, when each accesses the same features, then Free has no paid capability, Plus has exactly the agreed Plus capabilities including follower growth, and Business has both sets; switching active DID does not carry access across accounts. |
| AC-002 | BR-001, FR-002 | Given Free, Plus, and Business DIDs, when they call scheduled-post creation/management actions, saved-folder mutations, and pin mutations directly, then only Plus and Business can perform paid actions; saving ordinary posts and ordinary posting remain available under existing rules. |
| AC-003 | BR-001, FR-002 | Given a member requests their own follower-growth metrics, then Plus and Business receive permitted data, while Free cannot obtain it; another member cannot read those private metrics regardless of tier. |
| AC-004 | BR-002, FR-004, FR-008, RULE-001 | Given a formerly manually marked business account with Free or Plus access, when it tries to create or edit products, CTA, business details, or events, then the paid actions are rejected and its effective account type is regular; with effective Business access, its type is business and the actions are allowed under existing ownership and validation rules. |
| AC-005 | FR-001, NFR-002 | Given a billing owner who has assigned a paid licence to another DID, when either account uses a paid endpoint, then only the assigned DID receives the licensed capability and all pre-existing ownership checks still apply. |
| AC-006 | FR-002 | Given a direct HTTP caller bypasses the Flutter lock, when the caller attempts any protected mutation or metric read, then AppView denies it without side effects and uses the normal `/v1/` error contract. |
| AC-007 | FR-003 | Given a Free member sees a Plus-only entry point, when they activate the overlaid lock, then a dialog names the required feature and tier; choosing its learn-more CTA opens the existing subscription page without attempting the paid action. |
| AC-008 | FR-003 | Given a Plus or Business member sees the same Plus entry point, then the paid control works normally without the locked interaction. |
| AC-009 | FR-004 | Given a Free or Plus DID, when it views Settings or attempts to open a Business management page by deep link or call a Business owner-management API directly, then Business menu options and editor controls are absent, the page does not open, and the API denies the operation without side effects; the ordinary subscription page remains available in Settings. A Business DID can use the section without Plus-style overlays. |
| AC-010 | FR-005 | Given a business-profile owner has no effective Business licence, when any viewer including the owner reads CraftSky profile or event surfaces, then business details/products/CTA/events are absent even if records exist; restoring Business access restores eligible serving without recreating records, and permitted Free/Plus visitors can view that licensed public business content. |
| AC-011 | FR-005, FR-008, RULE-002 | Given Business access ends, then the effective account type changes to regular, CraftSky stops serving paid business content, retained PDS records are not deleted or claimed private, and restored Business access restores eligible business classification and serving subject to membership/moderation visibility. |
| AC-012 | FR-007 | Given Plus access is lost before a scheduled publication, then the scheduled item remains owner-visible and does not publish while unlicensed; when access later returns, an item that missed its time does not auto-publish. Existing published posts remain subject to ordinary visibility. |
| AC-013 | NFR-001 | Given keyboard or screen-reader interaction, then the locked Plus control and dialog communicate the feature requirement without relying only on an icon and the CTA is focusable/announced. |
| AC-014 | NFR-002 | Given otherwise identical Free and paid posts/accounts, then changing licence assignment does not confer timeline/search/discovery priority or bypass existing owner/moderation checks. |
| AC-015 | BR-001, FR-002, FR-006 | Given a Free account with saved colour or background selections, when it views or attempts to edit its profile, then CraftSky presents standard defaults and denies all customisation mutations; Plus and Business accounts may select and publicly display all currently supported customisation choices, and restoring access reapplies retained choices. The retired border selector is not restored. |
| AC-016 | FR-006 | Given saved posts previously assigned to folders, when Plus-or-higher access ends, then the owner sees every saved post in a single flat list without folder grouping; after access returns, the folder definitions and memberships reappear without re-saving or reassigning posts. |
| AC-017 | FR-006 | Given one or both profile pin slots are occupied, when Plus-or-higher access ends, then active pin selections are cleared and no pin is promoted on public profiles; restoration leaves slots empty until the owner pins again. |
| AC-018 | FR-007 | Given scheduled items and no Plus-or-higher access, then still-future items remain owner-visible with a subscription-required notice; an item that becomes due is not published, remains visible with a subscription-required error, and does not publish automatically after access returns; an item whose scheduled time is still future when access returns remains eligible for normal execution. |
| AC-019 | FR-008, RULE-001 | Given Free, Plus, and Business DIDs, when they visit account settings or attempt the former account-type mutation directly, then no toggle is available and the mutation route is unavailable to all tiers; the first two DIDs read `regular`, the effective Business DID reads `business`, and loss of Business access returns it to `regular`. A forward migration removes the testing-only `craftsky_account_types` table and no production read or write depends on it. |
| AC-020 | RULE-003 | Given a cancelled subscription still gives access through its paid period or grace state, when account type and feature access are read, then all benefits and pins continue until effective access actually ends. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Billing owner owns both tiers for different DIDs. | Each DID gets only its assigned effective tier; Business inherits Plus. | FR-001, RULE-001 |
| EC-002 | Licence expired but assignment remains dormant. | Effective access is Free until restored; folders and customisation remain stored, pin selections are cleared, and effective business type is regular. | FR-001, FR-006, FR-008 |
| EC-003 | Business owner has existing PDS records and loses access. | CraftSky hides Business fields/events and owner-management options/routes until access returns; source records remain on the PDS for restoration. | FR-004, FR-005, RULE-002 |
| EC-004 | Access fetch is pending or unavailable in Flutter. | Do not grant paid actions from stale state or another account's access; allow retry without losing content. | FR-001, FR-003 |
| EC-005 | Scheduled time passes during a lapse. | Show subscription-required error on the retained item and do not auto-publish on renewal; future-dated items still pending on renewal may execute normally. | FR-007 |
| EC-006 | Account was manually marked business during testing but has only Plus. | Effective type becomes regular; Plus works, while Business-only actions and public content remain unavailable. | FR-004, FR-005, FR-008, RULE-001 |
| EC-007 | Subscription is cancelled but continues to give access until period end. | Keep paid features and business classification until effective access ends; do not clear pins early. | RULE-003 |

## 15. Data / Persistence Impact

- New fields: None identified as a business requirement; implementation may need read-model joins or access checks.
- Changed fields: Effective business account type must no longer be independently mutable; response projections derive it from effective Business access. Retire the owner-facing toggle and its mutation route. Folder associations and saved customisation persist across lapse; pin selections do not.
- Migration required: Yes. After replacing live reads/writes and lifecycle cleanup that reference it, drop the unused `craftsky_account_types` table with a new migration; do not edit historical migrations or delete any PDS business records.
- Backwards compatibility: The temporary unrestricted business switch and previously free customisation are intentionally superseded. Public PDS records remain portable and public even if CraftSky stops serving them.

## 16. UI / API / CLI Impact

- UI: Overlaid locked Plus icons and explanation/CTA dialogs (including all supported profile customisation); Business section, menu options and management pages shown only with effective Business access; no manual account-type toggle; flat saved-post list without Plus, no active pins on lapse, and a subscription-required scheduled-item error; existing Settings subscription page remains accessible.
- API: Remove the manual account-type mutation route; derive business type on reads. Require Business access on authenticated owner-management routes while allowing public reads of licensed businesses. Enforce paid access on other existing feature endpoints using `/v1/` conventions. Exact paid-access error code belongs in test/design review.
- CLI: None identified.
- Background jobs: Scheduled publication must recheck effective tier at execution time and expose blocked due items as needing attention rather than publishing or silently backfilling them.

## 17. Security / Privacy / Permissions

- Authentication: Continue using authenticated DID and existing session/account-boundary rules.
- Authorization: AppView effective tier plus existing owner/member/moderation checks; no client-only or RevenueCat-customer-wide grant.
- Sensitive data: Follower-growth metrics and subscription/assignment information remain private; no public billing data on PDS.
- Abuse cases: Direct API use, stale access, and switching between retained accounts must not bypass gates. Business records on a PDS cannot be made private by CraftSky suppression.

## 18. Observability

- Events: None identified as a new product event requirement.
- Logs: Paid-access rejections and scheduled-post skips should use existing structured/redacted patterns without leaking billing identities or private metrics.
- Metrics: None identified as a new mandatory metric.
- Alerts: None identified.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Gating only Flutter or using the billing owner's tier. | Bypass or paid access on the wrong DID. | AppView authorization and multi-account/direct-call tests. |
| RISK-002 | Suppression is mistaken for deletion or privacy of public PDS business records. | Data loss or misleading privacy claims. | Keep records; document CraftSky-serving-only boundary and test lapse/restore. |
| RISK-003 | Current profile customisation and manual business type were previously free. | Existing accounts could continue displaying premium choices or a stale business label after access ends. | Gate writes and effective public projections; test defaults and derived classification across lapse/restore. |
| RISK-004 | A queued scheduled job publishes after Plus expiry or backfills on renewal. | Unlicensed or unexpected posting. | Recheck tier at execution; test lapse and missed schedules. |
| RISK-005 | Existing business-profile availability is based on self-declared type alone. | Paid business content stays visible after lapse or leaks through alternate read surfaces. | Audit all CraftSky profile/event reads and ensure tier-aware serving. |
| RISK-006 | Clearing pins at access loss can race with a new assignment or publication. | Pins may survive in one projection or disappear prematurely while renewal is cancelled but access remains. | Key removal to effective access transitions and verify public pin reads and cancellation timing. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Prices are indicative copy only; actual checkout prices remain store/RevenueCat sourced. | Pricing and catalog work would need separate requirements. |
| ASM-002 | Without Plus, ordinary saved-post access and existing published posts stay free; retained saved posts are shown without folder grouping, and all existing profile customisation renders standard defaults. | Lapse presentation would need revision. |
| ASM-003 | On loss of effective Business access, the effective account type reverts to regular; historical testing-only account-type rows are removed by migration rather than consulted. | A separate account-type transition policy would be needed. |
| ASM-004 | Retained Business PDS records need not remain manageable through CraftSky owner routes during a lapse; they reappear when the owner regains effective Business access. Source PDS records remain public and untouched. | A separate recovery route would need explicit scope if management without subscription is later required. |
| ASM-005 | A future scheduled item remains pending through a temporary lapse and executes normally if access is restored before it becomes due; a due item blocked for lack of access enters needs-attention and requires owner action after renewal. | Scheduler state and resume requirements would change. |

## 21. Open Questions

- [x] All currently supported customisation choices (colour and background) require Plus; saved selections are retained but standard defaults display without access. The retired border selector stays removed (Q3; DR-001).
- [x] Business account type is subscription-derived; remove the manual toggle and its mutation route entirely (Q4; product-owner clarification during review).
- [x] Saved posts flatten while folder membership persists, pins are removed, business records are hidden from CraftSky but retained, and scheduled items show a relevant error (Q5).
- [x] Future-dated items still pending at restoration may run normally; missed items require owner attention rather than auto-publishing (Q6; chosen from the permitted easier behavior).
- [x] Without Business access, hide Business settings/options and deny Business owner-management routes, including reads and deletes; restore them only with access. Public reads of other licensed businesses remain available (Q7).
- [x] Drop the testing-only `craftsky_account_types` table in a forward migration after replacing all live references; retain public PDS business records (Q7).
- [ ] **Non-blocking for draft:** Exact dialog copy, lock icon placement per entry point, and API error codes.

## 22. Review Status

Status: Reviewed for test design; revised after document review
Risk level: High
Review recommended: Required
Reviewer: Product owner
Date: 2026-09-28
Notes: The product owner approved the earlier requirements for test design. Subsequent document review confirmed the retired border is out of scope and removed the manual business toggle/mutation. Coding-plan feedback additionally requires dropping the unused account-type table and hiding all Business owner routes/options during lapse. Earlier feature documents intentionally allowed free use; these requirements change that policy. Explicit approval of the final high-risk document set is still required before implementation.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001–BR-002, FR-001–FR-008, NFR-001–NFR-002, RULE-001–RULE-003.
- Suggested test levels: account/tier authorization unit tests; direct API, job, and public-serving integration tests; Flutter lock/dialog/navigation widget tests; expiry/restoration and account-switch regression tests.
- Blocking open questions: None on product behavior. The product owner approved test design; explicit approval of the final high-risk document set is required before implementation.
