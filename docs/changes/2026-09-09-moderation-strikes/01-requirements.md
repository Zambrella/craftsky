# Requirements: Moderation Cases, Strikes, Appeals, And Suspension

## 1. Initial Request

Create a CraftSky moderation communication and enforcement system in which users can review moderation actions affecting their content or account, see a clear three-strike account-standing summary, understand each decision, and contact `moderation@craftsky.social` to appeal through a prefilled email whose subject identifies the moderation case.

Before CraftSky adopts Ozone, use Retool as a small moderation dashboard backed by authenticated AppView admin APIs. Retool should let a trusted moderator review grouped reports and submit decisions. The durable application boundary should be reusable by a future Ozone adapter rather than coupling moderation policy to Retool or allowing direct database writes.

Confirmed policy decisions:

- Three active strikes cause an indefinite CraftSky suspension.
- A documented severe violation may cause immediate suspension without waiting for three strikes.
- Strikes expire after 12 months.
- Enforcement remains active while an appeal is pending.
- The user-facing page shows all consequential moderation decisions, not only strike-bearing decisions; no-action outcomes remain private.
- Suspended users retain read-only browsing, moderation history, appeal, reporting, sign-out, and account-deletion capabilities.
- Severe suspensions remain active until a moderator explicitly restores the account or overturns the decision.
- Reports against the same subject join one open case; a report received after resolution creates a new case.
- Retool reads through admin APIs rather than directly from Postgres.
- The initial dashboard has one trusted moderator.
- Moderation actions create push notifications, but users may disable moderation push delivery in CraftSky notification settings.
- Decisions use separate disposition and consequence dimensions; a violation cannot be consequence-free.
- No-action case outcomes remain private and do not notify the affected user.
- Cases cannot be created without reports or reopened after resolution in V1.
- Reversals may negate selected effects, and a later audited effect-change command may reapply an effect with a required rationale.
- Every case with an owner-visible decision has one appeal lifecycle using the public case reference.
- Decisions reuse the report reason taxonomy, with explicit category limits for strikes and severe suspension.
- Strike expiry converges within one hour and creates a moderation notification only when effective enforcement changes.
- Suspended users may manage blocks and mutes and remove their existing public content.
- Strike expiry and threshold restoration become effective together when the restart-safe worker commits them, no later than one hour after the 12-month deadline.
- Suspended users retain private account-maintenance actions, including notification/language/device preferences, saved-content and recent-search management, and cancellation or deletion of private pending work.
- Later effect application and explicit restoration notify only when they add, remove, or change an owner-visible consequence or effective enforcement; appeal status-only updates do not create push work.
- Existing pre-case reports and moderation outputs are preserved without synthetic cases, owner history, strikes, appeals, or notifications.
- Untrusted appeal correspondence does not create a pending appeal until an authenticated moderator confirms it for the case.
- Initial operational alerts use concrete defaults: five failed admin-auth attempts in five minutes, expiry work past its one-hour deadline, or eligible notification work older than 15 minutes.
- Notification opens reuse the existing account-subscription binding to activate the exact retained account before showing moderation history; raw DIDs are not added to provider payloads for routing.

## 2. Current Codebase Findings

- Relevant files:
  - `appview/migrations/000014_moderation_flow.up.sql` defines private `moderation_reports`, append-only `moderation_outputs`, report subject snapshots, and current moderation indexes.
  - `appview/migrations/000063_business_event_moderation.up.sql` extends moderation support to business events.
  - `appview/internal/api/report.go`, `report_request.go`, `report_response.go`, and `report_store.go` implement authenticated report intake for posts, accounts, and events.
  - `appview/internal/api/moderation.go`, `moderation_request.go`, and `moderation_store.go` implement development-only trusted moderation-output ingestion with idempotency protection.
  - `appview/internal/middleware/current_member.go` enforces current membership but has no reversible moderation-suspension state.
  - `appview/internal/ownerlifecycle/state.go` and `appview/migrations/000038_owner_auth_lifecycle.up.sql` define owner membership and irreversible account-deletion lifecycle states; they are not suitable for moderation suspension.
  - `appview/internal/notifications/category.go` and `appview/internal/push/` provide durable notification and push infrastructure, but no dedicated moderation-notice category or payload.
  - `app/lib/settings/pages/settings_page.dart` provides the natural settings entry point for account standing and moderation history.
  - `app/lib/moderation/` renders generic viewer-facing warning metadata but has no owner-facing decision history.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` fixes `/v1/`, camelCase JSON, standard errors, authentication conventions, and opaque cursor pagination.
  - `docs/changes/2026-05-30-moderation-flow-mvp/01-requirements.md` explicitly deferred appeals, moderator dashboards, email, and live Ozone integration.
- Existing patterns:
  - Reports are private intake records and duplicate reports are intentionally allowed.
  - Moderation outputs are append-only `apply` or `negate` events for `warn`, `hide`, and `takedown` behavior.
  - Active moderation outputs filter content in AppView without deleting records from a user's PDS.
  - Push delivery uses durable fan-out, retries, leases, and account-bound routing.
  - Flutter uses typed routes, repositories/providers, localized copy, and settings row descriptors.
- Current behavior:
  - A report has no review state, case, reviewer, decision, strike, appeal, or enforcement result.
  - There is no production moderator API; explicit moderation-output ingestion is development-only.
  - Users cannot see a history of actions against their content or account.
  - There is no strike ledger, account suspension projection, or write restriction for suspended accounts.
  - Existing warnings are generic notices for content viewers and intentionally hide internal reasons.
  - Existing notification categories are mostly actor-backed and do not support a first-class moderation system notice.
- Constraints discovered:
  - Reports, cases, appeals, strikes, moderator notes, and enforcement state are private AppView data and must not be written to a PDS.
  - A CraftSky suspension controls AppView participation only; it must not delete PDS records, disable a DID, or claim to ban the user from AT Protocol.
  - OAuth and PDS credentials remain server-side.
  - The owner lifecycle remains separate from reversible moderation enforcement.
  - No lexicon changes are required.
  - Future Ozone compatibility is best achieved through an application-service adapter because Ozone uses subject status plus append-only moderation events and supports appeal and strike concepts.
- Test/build commands discovered:
  - `just appview-test-unit` runs the incomplete host-side AppView unit suite.
  - `just dev-d` followed by `just test` runs the AppView suite with PostgreSQL, MinIO, and the race detector.
  - `just appview-check` is the release-equivalent AppView verification gate.
  - Flutter tests run from `app/` with `flutter test <paths>`.
  - Flutter static analysis runs through the project's Dart/Flutter analysis workflow.

## 3. Clarifying Questions And Decisions

### Q1: What should the third active strike do?

Answer: Indefinitely suspend the CraftSky account.

Decision / implication: The suspension is reversible and applies to CraftSky participation, not the user's DID, PDS account, or wider AT Protocol access.

### Q2: Can severe violations bypass the three-strike threshold?

Answer: Yes, with documented reasons.

Decision / implication: Severe suspension is an explicit enforcement outcome and must not fabricate three strikes.

### Q3: How long does a strike remain active?

Answer: 12 months.

Decision / implication: Expired strikes remain in history but stop contributing to the active count.

### Q4: Does a pending appeal pause enforcement?

Answer: No.

Decision / implication: The original decision and any suspension remain active until a moderator changes them.

### Q5: What appears on the user-facing moderation page?

Answer: All consequential moderation decisions.

Decision / implication: Warnings, removals, strike-bearing decisions, suspensions, expirations, and reversals appear in one chronological history with their effect on account standing clearly distinguished. Cases resolved without consequence remain private.

### Q6: What may a suspended user do?

Answer: Use CraftSky in restricted read-only mode with explicit safety and removal exceptions.

Decision / implication: Browsing, moderation history, appeal contact, reporting, sign-out, account deletion, block/mute management, and removal of existing owned public content remain available; public-content creation/editing and ordinary social participation are blocked.

### Q7: How are duplicate reports presented to moderators?

Answer: As one subject case.

Decision / implication: Concurrently open reports against the same canonical subject are evidence for one adjudication and cannot independently create strikes.

### Q8: What happens when a resolved subject is reported again?

Answer: Create a new case.

Decision / implication: Each review episode has a distinct public case reference, evidence set, decision, and appeal history.

### Q9: How does Retool read moderation data?

Answer: Through authenticated admin read APIs.

Decision / implication: Retool receives curated API representations and has no direct table access or database write authority.

### Q10: Who uses the initial dashboard?

Answer: One trusted moderator.

Decision / implication: The initial credential may map to one fixed server-side moderator identity, but the API and audit records must permit later migration to individual moderator identities.

### Q11: How does a severe suspension end?

Answer: Through explicit moderator restoration.

Decision / implication: Strike expiry alone does not lift an active severe suspension.

### Q12: May a suspended user submit reports?

Answer: Yes.

Decision / implication: Reporting is treated as a retained safety capability rather than an ordinary social mutation and remains subject to normal abuse protections.

### Q13: Should moderation actions produce push notifications?

Answer: Yes.

Decision / implication: User-impacting moderation decisions and reversals create first-class system notification work that navigates to the relevant moderation history entry.

### Q14: May users disable moderation push notifications?

Answer: Yes.

Decision / implication: App-level preferences may suppress provider push delivery, but they do not suppress the moderation decision, account-standing update, or durable moderation history.

### Q15: How is a moderator decision represented?

Answer: As separate dimensions rather than one combined outcome enum.

Decision / implication: Every resolution has one disposition (`violation` or `noAction`). A violation then selects one or more explicit consequences: a private owner-facing `formalWarning`, at most one existing viewer-facing visibility output (`warn`, `hide`, or `takedown`), an ordinary strike, or severe suspension.

### Q16: May a violation have no consequence?

Answer: No.

Decision / implication: A `violation` requires at least one explicit consequence. `noAction` requires no consequences, remains private moderator audit history, and does not notify or appear to the affected user.

### Q17: Is a formal owner warning the same as the existing `warn` output?

Answer: No.

Decision / implication: `formalWarning` is a private owner-facing consequence. The existing `warn` output remains viewer-facing subject metadata. A decision may apply either or both.

### Q18: May a moderator create a reportless case?

Answer: No in V1.

Decision / implication: Every V1 case begins with an accepted report. Moderator-initiated cases require a later policy decision and are not part of the initial Retool workflow.

### Q19: May a resolved case reopen?

Answer: No.

Decision / implication: Cases move from open to resolved once. Appeals, appeal outcomes, expiry, effect-level reversals, and later audited effect applications append to the resolved history; a later report opens a new case.

### Q20: May a severe decision also issue an ordinary strike?

Answer: Yes, only when both effects are selected explicitly.

Decision / implication: Severe suspension never synthesizes a strike. An explicitly issued strike remains independent, expires normally, and may still affect standing after severe restoration.

### Q21: What exactly is the 12-month strike lifetime?

Answer: Twelve calendar months from the server-recorded issuance timestamp in UTC.

Decision / implication: The expiry uses the same UTC wall-clock time and clamps to the last valid day of the target month, so a strike issued on February 29 expires on February 28 the following year.

### Q22: How quickly must expiry converge?

Answer: A restart-safe worker must apply expiry and any resulting threshold restoration within one hour of the deadline.

Decision / implication: The system does not promise request-time restoration at the exact timestamp, but must not keep a threshold-only suspension active beyond the one-hour convergence window.

### Q23: Does strike expiry create a notification?

Answer: Only when strike expiry changes effective enforcement, such as restoring participation.

Decision / implication: Expiry always remains visible in moderation history, but creates durable moderation-notification work only when it changes effective enforcement. Provider delivery still respects the user's moderation-push preference.

### Q24: Can a reversal change selected effects?

Answer: Yes.

Decision / implication: A reversal explicitly identifies which effects it negates. Unchanged effects remain active. A later audited effect-change command may reapply an effect within the resolved case when it includes a rationale and a distinct idempotency identity.

### Q25: How many appeals may one case have?

Answer: One appeal lifecycle.

Decision / implication: A case with an owner-visible decision has at most one appeal lifecycle. Repeated emails are correspondence for that appeal. Its status is `pending` until it resolves as `upheld` or `changed`; reversal events identify the effects changed. Later correspondence remains reviewable and may produce another audited effect change without creating a second appeal lifecycle.

### Q26: Which decisions are appealable?

Answer: Every owner-visible violation decision.

Decision / implication: Formal warnings and visibility-only decisions are appealable as well as strikes and suspensions. Automatic expiry and restoration events are not independently appealable.

### Q27: Must an appeal email come from a verified account address?

Answer: No; CraftSky has no verified member email.

Decision / implication: Any sender may provide a valid public case reference as untrusted correspondence for human review. Staff must not disclose private case data by email or infer account ownership from the sender, and email alone cannot change enforcement.

### Q28: What public moderation identifier is exposed?

Answer: One public case reference per review episode.

Decision / implication: The case uses a UUIDv4, presented publicly as `MOD-<case UUID>`. It is used in email subjects, owner-history deep links, and moderator lookup; internal decision and event IDs remain private. Lookup is case-insensitive.

### Q29: How are owner-facing reasons authored?

Answer: A required controlled reason code plus an optional separately stored user-safe detail.

Decision / implication: The code maps to localized policy copy. Every violation also requires private internal evidence notes. The optional user-safe detail is displayed verbatim and must never be populated from report text or internal evidence automatically.

### Q30: Which reason taxonomy is used for decisions?

Answer: Reuse the existing report reason options.

Decision / implication: Decisions use `harassment`, `hate`, `spam`, `misleading`, `suspected_ai_generated`, `adult_or_graphic`, `impersonation`, `off_topic`, `intellectual_property`, or `other`; disposition distinguishes a reporter allegation from a moderator finding.

### Q31: Which categories can cause account punishment?

Answer: Every named category except `other` may issue an explicitly selected ordinary strike. Severe suspension is limited to `harassment`, `hate`, `adult_or_graphic`, and `impersonation` and requires a private severity rationale.

Decision / implication: `other` may produce a formal warning or visibility output but cannot issue a strike or severe suspension. No category creates either effect automatically.

### Q32: How are related multi-subject incidents kept to one strike?

Answer: Moderator operating guidance, not a functional system requirement or technical cross-case invariant.

Decision / implication: Moderator operating documentation advises against issuing multiple strikes for one materially connected incident. The database guarantees at most one strike per case, but V1 does not add an incident-group model or dedicated Retool behavior.

### Q33: Which additional actions remain available during suspension?

Answer: Block and mute management, plus removal of the user's existing public content.

Decision / implication: Suspended users may create or remove blocks and mutes, delete their posts or events, and remove business-profile data or pins. Creating or editing public content remains blocked; user-initiated removal is distinct from prohibited moderator deletion of PDS records.

### Q34: When does strike expiry change effective enforcement?

Answer: When restart-safe expiry processing commits, no later than one hour after the 12-month deadline.

Decision / implication: The calendar deadline makes the strike due for processing. Until the worker atomically persists expiry, standing, and any threshold restoration, the stored projection remains authoritative. Requests do not independently derive restoration from wall-clock time. The worker must converge within one hour and retries must be idempotent.

### Q35: Which private mutations remain available during suspension?

Answer: Private account maintenance remains available; public participation and its preparation do not.

Decision / implication: All authenticated reads remain available. The mutation allowlist includes logout and the full account-deletion flow; report submission; block/mute creation and removal; deletion of owned posts, events, business-profile data, and pins; notification seen state and notification/language/device preferences; recent-search management; saved-post and saved-folder management; and cancellation or deletion of private scheduled or migration work. Uploads, link-preview creation, onboarding completion, profile/public-business edits, account-type changes, follows/unfollows, post/event creation or editing, scheduling/publication, likes/reposts, pin creation, migration/import starts or confirmations, and every unclassified mutation are denied.

### Q36: Which later moderation events notify the owner?

Answer: Only events that add, remove, or change an owner-visible consequence or effective enforcement.

Decision / implication: Initial consequential decisions, effect reversals, effect reapplications, and explicit restorations create notification work when they change a consequence or effective enforcement. Strike expiry creates notification work only when effective enforcement changes. Appeal receipt, pending status, and an upheld outcome without an effect change remain visible in moderation history but do not create push work.

### Q37: How is pre-case moderation data migrated?

Answer: Preserve it without backfilling synthetic cases or owner history.

Decision / implication: Existing reports remain private intake rows and existing moderation outputs continue to govern visibility. Neither becomes a case, owner-history item, strike, appeal, suspension, or notification. Only reports accepted after the case feature is enabled participate in automatic case creation.

### Q38: When does email correspondence become a pending appeal?

Answer: Only after an authenticated moderator confirms receipt for an owner-visible case.

Decision / implication: Correspondence is untrusted intake and does not consume the single appeal lifecycle, change owner-visible appeal state, disclose case data, or affect enforcement by itself. The moderator command records the first confirmed appeal as pending; later correspondence attaches to that lifecycle.

### Q39: What initial operational alert thresholds apply?

Answer: Five failed admin-auth attempts in five minutes, expiry work beyond its one-hour deadline, and eligible notification work older than 15 minutes.

Decision / implication: These defaults are measurable acceptance targets. Configuration may later make them adjustable without weakening the initial defaults or required signals.

### Q40: How does a notification open for a different signed-in account?

Answer: Reuse the existing secure notification account-binding flow and switch automatically before navigation.

Decision / implication: The provider payload carries the existing opaque account subscription identifier, not a raw DID. The app resolves it to one current retained-account lease, activates that exact account, rechecks the lease, and only then opens the case. Invalid, ambiguous, removed, or stale bindings show no case details and do not navigate.

## 4. Candidate Approaches

### Option A: AppView cases and adjudication service with a Retool adapter

Summary: Add private subject cases, append-only decisions, strikes, appeals, enforcement projections, owner-facing history, and admin APIs. Retool is an authenticated client. Future Ozone ingestion maps external events into the same adjudication service.

Pros:

- Preserves AppView as the policy and transaction authority.
- Groups duplicate reports without treating report volume as guilt.
- Supports complete audit history and safe reversals.
- Keeps Retool replaceable and prevents direct database mutation.
- Provides a stable domain boundary for Ozone without copying Ozone's storage model.

Cons:

- Requires new private persistence, admin authentication, enforcement middleware, UI, and notifications.
- Requires careful transactional and concurrency design.

Risks: High; changes account authorization, private safety data, and production moderation operations.

### Option B: Add mutable status fields to moderation reports

Summary: Let Retool list report rows and patch each report to a status or outcome.

Pros:

- Smallest apparent backend change.
- Simple Retool queries.

Cons:

- Duplicate reports can produce duplicate work or strikes.
- Conflates intake, review, adjudication, and enforcement.
- Overwrites audit history and handles reversals poorly.
- Cannot naturally represent moderator-initiated action without a report.
- Is not a good fit for Ozone's subject/event model.

Risks: High; deceptively simple writes can produce inconsistent or unauditable enforcement.

### Option C: Adopt Ozone as the immediate source of truth

Summary: Deploy Ozone now and build the owner-facing history and enforcement integration directly from Ozone status and events.

Pros:

- Avoids a temporary moderation dashboard.
- Uses mature moderation workflow concepts immediately.

Cons:

- Expands operational scope before CraftSky needs the full system.
- Couples initial product policy work to Ozone deployment and integration decisions.
- Delays a small, usable moderator workflow.

Risks: High; external service deployment, authentication, event ingestion, and policy mapping all arrive in one change.

## 5. Recommended Direction

Recommended approach: Option A, delivered in staged slices but governed by this single requirements contract.

Why: It is the smallest approach that supports fair case grouping, append-only decisions, reversible strikes and suspensions, owner transparency, email appeals, and reliable enforcement. Retool remains a replaceable admin client, while a shared adjudication service provides the future Ozone integration seam. Standard UUID references and a compact appeal state avoid unnecessary custom machinery. No client, dashboard, or external adapter receives direct authority to mutate moderation tables.

Suggested delivery stages:

1. Cases, report grouping, adjudication service, production admin authentication, admin APIs, and Retool dashboard.
2. Strikes, suspension enforcement, owner-facing history, email appeals, and moderation notifications.
3. Report forwarding and trusted Ozone event ingestion through adapters to the same application service.

## 6. Problem / Opportunity

CraftSky accepts private reports and can enforce synthetic moderation outputs, but it has no production review workflow or transparent communication with affected users. Treating each report as a decision would allow duplicate or coordinated reporting to cause duplicate punishment. Treating account suspension as account deletion would violate CraftSky's federated, user-owned architecture and deny users the ability to understand or appeal decisions.

A case-based adjudication system gives a trusted moderator a safe interim workflow, gives affected users a coherent account-standing history, and establishes a durable policy boundary that can later consume Ozone events.

## 7. Goals

- G-001: Give affected users a clear, private history of moderation decisions and their account-standing impact.
- G-002: Apply a predictable three-active-strike policy with expiry, reversal, and severe-violation handling.
- G-003: Preserve read-only and safety access for suspended users while preventing ordinary participation.
- G-004: Provide a production-safe Retool moderation workflow without direct database mutation.
- G-005: Support email-based appeals with stable public moderation identifiers and recorded outcomes.
- G-006: Notify affected users when moderation decisions or reversals occur.
- G-007: Preserve a reusable application boundary for future Ozone ingestion.
- G-008: Keep all private moderation data and enforcement state in AppView.

## 8. Non-Goals

- NG-001: Do not deploy or operate Ozone in this change.
- NG-002: Do not reproduce the full Ozone dashboard or schema in CraftSky.
- NG-003: Do not submit reports to a PDS or Ozone yet.
- NG-004: Do not create a native in-app appeal submission form; V1 appeal contact is email.
- NG-005: Do not infer guilt or issue strikes from report counts, reason popularity, or automated scoring.
- NG-006: Do not delete or modify a user's PDS records, deactivate their DID, or delete their PDS account for moderation.
- NG-007: Do not reuse owner account-deletion lifecycle states for moderation suspension.
- NG-008: Do not expose reporter identity, report details, moderator notes, private evidence, or external credentials to the affected user.
- NG-009: Do not give Retool direct database write access.
- NG-010: Do not add or change AT Protocol lexicons.
- NG-011: Do not implement ban-evasion detection or cross-DID identity linkage in V1.
- NG-012: Do not guarantee that Apple, Google, the operating system, or a device will display an enqueued push notification.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Reporting user | A signed-in member reporting a post, account, or event. | Submit a private safety report without controlling the outcome. |
| Affected user | The owner of moderated content or account. | Understand decisions, account standing, expiry, suspension, and appeal options. |
| Suspended user | An affected user under threshold or severe suspension. | Retain transparent read-only and safety/account access. |
| Trusted moderator | The single initial operator using Retool. | Review grouped evidence, make decisions, record appeals, reverse errors, and restore accounts safely. |
| Retool dashboard | The interim moderation admin client. | Read curated queue/detail APIs and submit authenticated commands. |
| CraftSky AppView | The policy, persistence, transaction, and enforcement authority. | Maintain consistent cases, decisions, strikes, suspensions, notifications, and audit history. |
| Flutter app | The affected-user client. | Display account standing/history, launch appeals, honor suspension restrictions, and route notifications. |
| Future Ozone adapter | A later trusted integration source. | Translate Ozone events into the same internal adjudication boundary with replay safety. |
| Push provider and device OS | External best-effort delivery systems. | Receive valid system-notification payloads without becoming a source of moderation truth. |

## 10. Current Behavior

Each user report is stored independently with a canonical subject snapshot and a `prepared_not_submitted` forwarding status. Reports have no review lifecycle. Trusted development requests can append `warn`, `hide`, or `takedown` moderation outputs and matching negations, but production moderators have no equivalent interface. Internal reasons are not exposed to users.

AppView hides or warns content according to active moderation outputs, but it does not record user-facing decisions, count strikes, suspend accounts, or create moderation notifications. Flutter has no account-standing page or appeal action. Existing owner lifecycle states represent membership and account deletion rather than reversible moderation enforcement.

## 11. Desired Behavior

The first report against a canonical subject opens a private moderation case. Further reports join that case while it is open. V1 does not permit reportless or reopened cases. Retool lists and inspects the case through production-safe admin APIs. A trusted moderator submits an idempotent, concurrency-protected decision containing a required disposition, separate internal and user-safe reasoning, and explicit independent consequences.

AppView commits all effects of that decision atomically: decision history, case projection, formal warning and/or moderation output, optional strike, account-standing projection, suspension changes, appeal state where applicable, and notification work. A violation requires at least one consequence; `noAction` dispositions remain private. Closed cases never reopen; appeals, effect-level reversals, and later audited effect applications append to them, while a later report starts a new case.

Affected users see every consequential decision in a localized moderation-history page and see active strikes against a threshold of three without framing unused strikes as permission to offend. They can launch a prefilled email appeal using the case's opaque public reference. Each case has at most one appeal lifecycle that is pending until resolved as upheld or changed. A pending appeal does not change enforcement. Moderator reversals may negate selected effects and update derived standing without deleting the original decision.

Three non-expired, non-overturned strikes cause indefinite threshold suspension. A strike becomes due for expiry 12 calendar months after issuance using clamped UTC calendar arithmetic. A restart-safe worker atomically persists expiry, standing, and any threshold restoration no later than one hour after that deadline; the stored projection remains authoritative until that commit. A documented severe decision may independently cause immediate suspension and may also explicitly issue one ordinary strike. Threshold suspension lifts when processed active count falls below three and no other suspension basis remains. Severe suspension requires explicit moderator restoration or reversal. Suspended users retain all authenticated reads plus the explicit private-maintenance, safety, account-deletion, and owned-content-removal mutation allowlist; public participation, preparation for publication, and unclassified mutations are denied.

Each authoritative event that adds, removes, or changes an owner-visible consequence or effective enforcement creates a durable moderation notification. This includes consequential decisions, effect reversals, effect reapplications, and explicit restorations; strike expiry creates one only when effective enforcement changes. Appeal status-only updates and upheld outcomes without an effect change remain history-only. If the user has not disabled moderation pushes and has an eligible installation, AppView attempts push delivery. Tapping the notification securely activates the exact retained account associated with the existing opaque account-subscription binding before opening the relevant moderation-history entry.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | CraftSky shall provide affected users with a private, understandable record of moderation decisions and current account standing. | Transparent moderation is the primary user need. | Initial request / User answer | AC-001, AC-014, AC-015 |
| BR-002 | Business | Must | CraftSky shall enforce a three-active-strike suspension policy while supporting documented immediate action for severe violations. | Predictability must not prevent urgent safety intervention. | User answers | AC-006, AC-009 |
| BR-003 | Business | Must | CraftSky shall provide an email-based appeal path and record moderator appeal outcomes. | Users need a practical V1 review mechanism. | Initial request / User answer | AC-016, AC-018 |
| BR-004 | Business | Must | CraftSky shall provide a production-safe interim moderation workflow through Retool and AppView admin APIs. | Moderation must operate before Ozone is deployed. | User proposal / User answer | AC-019, AC-020, AC-021 |
| BR-005 | Business | Must | CraftSky shall preserve an internal adjudication boundary that can accept future trusted Ozone inputs without making Retool the policy authority. | The interim dashboard should not become throwaway domain logic. | User proposal / Discovery | AC-022 |
| BR-006 | Business | Must | CraftSky shall create a moderation notification when an authoritative moderation event changes an owner-visible consequence or effective enforcement, including decisions, reversals, reapplications, restorations, and enforcement-changing expiry. | Affected users need timely communication without noisy status-only notices. | User requirement / User answer | AC-023, AC-025, AC-038, AC-039 |
| BR-007 | Business | Must | CraftSky moderation enforcement shall not delete or mutate user-owned PDS records or disable identities outside CraftSky. | Required by CraftSky's federated architecture. | AGENTS.md / Discovery | AC-028 |
| FR-001 | Functional | Must | The system shall create a case only from an accepted report and shall attach reports against the same canonical subject to one currently open moderation case. | Reports are evidence, not independent adjudications, and V1 excludes moderator-created cases. | User answers | AC-002, AC-034 |
| FR-002 | Functional | Must | The system shall create a new case when a report arrives for a subject whose previous cases are all closed. | Each review episode needs distinct history and appeal identity. | User answer | AC-003 |
| FR-003 | Functional | Must | The admin API shall provide a cursor-paginated moderation-case queue with filters needed to find open and historical cases. | Retool needs a stable queue contract. | User answer / API architecture | AC-019, AC-030 |
| FR-004 | Functional | Must | The admin API shall provide case detail containing canonical subject information, attached report evidence, prior decisions, strike context, appeal state, and current case revision. | Moderators need complete adjudication context. | Discovery | AC-020 |
| FR-005 | Functional | Must | The admin API shall accept an idempotent, concurrency-protected moderator decision command rather than arbitrary direct status mutation. | Decisions have coupled policy effects and must be replay-safe. | Confirmed direction | AC-021 |
| FR-006 | Functional | Must | The system shall preserve original decisions and represent effect-level reversal, later effect application, appeal resolution, expiry, and restoration as attributable append-only events. | Audit history must survive correction. | Discovery / User answers | AC-008, AC-018, AC-038, AC-039 |
| FR-007 | Functional | Must | The authenticated affected-user API shall return current account standing and a cursor-paginated chronological history of every consequential moderation decision affecting that user, regardless of whether each decision issued a strike. | Supports complete owner transparency without disclosing dismissed reports. | User answer | AC-001, AC-014, AC-030, AC-033 |
| FR-008 | Functional | Must | Flutter shall expose account standing and moderation history from the settings area and shall render active, expired, overturned, warning, removal, threshold-suspension, and severe-suspension states distinctly. | Users must understand current and historical consequences. | Initial request / Discovery | AC-001, AC-014, AC-015, AC-031 |
| FR-009 | Functional | Must | For an appealable decision, Flutter shall offer an email CTA to `moderation@craftsky.social` with a subject containing the case's opaque public reference and shall provide copyable address and reference fallbacks. | `mailto:` may be unavailable or abandoned. | Initial request / User answer | AC-016, AC-036 |
| FR-010 | Functional | Must | Opening the email CTA shall not mark an appeal as submitted or received. | Launching another app does not prove delivery. | Discovery | AC-017 |
| FR-011 | Functional | Must | The moderator workflow shall allow at most one appeal lifecycle per case with an owner-visible decision. Untrusted correspondence may locate a case using the public reference, but only an authenticated moderator confirmation creates the `pending` lifecycle, which later resolves as `upheld` or `changed`. | The user-facing status needs one compact authoritative source without letting spoofed correspondence consume it. | Discovery / User answers | AC-018, AC-040, AC-043 |
| FR-012 | Functional | Must | A moderator resolution shall separately state its disposition and explicit consequences; report volume, reason category, and moderation visibility action alone shall not issue a strike or suspension. | Prevents automated or duplicate punishment. | Discovery / User answers | AC-004, AC-032, AC-037 |
| FR-013 | Functional | Must | The system shall derive active strike count from issued strikes that are neither expired nor overturned. | Account standing must remain correct after time and reversals. | User answers | AC-005, AC-007, AC-008 |
| FR-014 | Functional | Must | The system shall enter threshold suspension atomically when a decision causes the account to reach three active strikes. | Avoids race conditions around the threshold. | User answer | AC-006, AC-021 |
| FR-015 | Functional | Must | The system shall permit a documented severe-violation decision to enter severe suspension independently of active strike count. | Severe safety cases cannot wait for repeated violations. | User answer | AC-009, AC-042 |
| FR-016 | Functional | Must | The system shall lift threshold suspension when active strike count falls below three, unless another active suspension basis remains. | Expiry or successful appeal must restore eligible accounts. | User answers | AC-007, AC-008, AC-038 |
| FR-017 | Functional | Must | The system shall lift severe suspension only after an explicit authorized restoration or reversal. | Severe suspension has an independently reviewed end condition. | User answer | AC-010 |
| FR-018 | Functional | Must | Suspended users shall retain all authenticated reads. The mutation allowlist shall include logout and account deletion; reports; block/mute management; deletion of owned posts, events, business-profile data, and pins; notification seen state and notification/language/device preferences; recent-search and saved-content management; and cancellation or deletion of private scheduled or migration work. Public creation/editing, engagement, publication preparation, uploads, and every unclassified mutation shall be denied. | Preserves safety, privacy, and account rights without continued participation while making authorization fail closed. | User answers | AC-011, AC-012, AC-013, AC-041 |
| FR-019 | Functional | Must | Every authoritative event that adds, removes, or changes an owner-visible consequence or effective enforcement shall atomically create durable moderation-notification work for the affected user. This includes consequential decisions, effect reversals, effect reapplications, and explicit restorations; strike expiry shall do so only when effective enforcement changes. Appeal status-only changes and upheld outcomes without an effect change shall not create notification work. | Important enforcement communication cannot be lost or made noisy by status-only updates. | User requirement / User answers | AC-023, AC-038, AC-039 |
| FR-020 | Functional | Must | Flutter shall expose and persist a moderation-push preference through the existing notification settings API, and AppView shall suppress moderation provider delivery when that category is disabled without suppressing moderation history or decision effects. | Honors the confirmed user preference end to end. | User answer | AC-024, AC-044 |
| FR-021 | Functional | Must | A moderation push payload shall identify a system moderation notice without impersonating a social actor and shall use the existing opaque account-subscription binding. On open, Flutter shall activate the exact current retained-account lease before routing to the relevant moderation-history entry; invalid, ambiguous, removed, or stale bindings shall expose no case data and shall not navigate. | Existing actor-backed payloads are unsuitable, and account switching must preserve privacy. | Discovery / User answer | AC-025, AC-048 |
| FR-022 | Functional | Must | The system shall reuse the attached report's canonical subject snapshot to explain a decision when the live subject has been edited, hidden, removed, or deleted, adding decision-specific snapshot data only when the report snapshot is insufficient. | Avoids duplicate persistence while preserving understandable history. | Discovery | AC-026 |
| FR-023 | Functional | Must | Production admin endpoints shall use credentials and middleware separate from member authentication and the development moderation token. | Member or development credentials must not grant moderator authority. | Discovery | AC-027 |
| FR-024 | Functional | Must | Each trusted decision input shall carry a source system and replay identifier that cannot create duplicate effects when retried. | Retool and future Ozone delivery are at-least-once. | Discovery | AC-021, AC-022 |
| FR-025 | Functional | Must | The current implementation shall expose a source-neutral trusted-input adapter contract that can invoke only the same adjudication application service used by the admin command path, carries source/replay identity, and exposes no direct moderation-table mutation interface. Live Ozone payload mapping and ingestion are deferred to Stage 3. | Preserves one policy and transaction boundary without claiming unimplemented Ozone conformance. | Confirmed direction | AC-022 |
| FR-026 | Functional | Must | Adjudication shall continue to emit or negate existing moderation outputs used by AppView visibility policy rather than introducing a second visibility-enforcement mechanism. | Existing read-path moderation must remain authoritative. | Codebase / Discovery | AC-029 |
| FR-027 | Functional | Must | A case resolution shall use exactly one disposition: `violation` or `noAction`; `violation` requires at least one explicit consequence, while `noAction` prohibits consequences. | Prevents contradictory or consequence-free decisions without unnecessary private states. | User answers | AC-032, AC-033 |
| FR-028 | Functional | Must | `formalWarning` shall be an owner-facing consequence distinct from the existing viewer-facing `warn` moderation output, and a decision may apply either or both. | The two warnings serve different audiences and policies. | Codebase / User answer | AC-035 |
| FR-029 | Functional | Must | A resolved case shall never reopen; appeals, appeal outcomes, expiries, reversals, and later audited effect applications append to its history, while a later accepted report creates a new case. | Preserves stable review episodes and concurrency semantics. | User answer | AC-003, AC-034 |
| FR-030 | Functional | Must | A reversal command shall identify the exact active effects to negate and leave unselected effects unchanged; a later command may reapply an effect in the same case only with a rationale and distinct idempotency identity. Reapplying the case's strike reactivates its single logical strike and uses the reapplication timestamp for its new 12-month lifetime. | Supports partial correction and later reconsideration without losing auditability or creating duplicate case strikes. | User answers | AC-039 |
| FR-031 | Functional | Must | Every case with an owner-visible violation decision shall permit one appeal lifecycle; automatic expiry and restoration events shall not be independently appealable. | Even non-strike consequences can materially affect a user. | User answer | AC-040 |
| FR-032 | Functional | Must | Each case shall use a UUIDv4 presented publicly as `MOD-<case UUID>`, matched case-insensitively, and exposed as the only moderation identifier in email, deep links, and owner lookup. | Uses one stable, standard, unguessable case identifier without custom encoding or a second public-reference identity. | User answer | AC-036 |
| FR-033 | Functional | Must | A violation decision shall use one existing approved report reason code, require private internal evidence notes, and permit an optional separately authored user-safe detail alongside localized reason copy. | Reuses the established vocabulary while separating allegations, evidence, and owner communication. | User answers | AC-037 |
| FR-034 | Functional | Must | A strike shall become due for processing at its 12-month clamped UTC deadline. The restart-safe worker shall atomically make expiry, standing, and any threshold restoration effective no later than one hour afterward; until that commit, the stored enforcement projection remains authoritative. | Defines one effective-time model and the accepted restoration latency. | User answer | AC-005, AC-007, AC-038 |
| FR-035 | Functional | Must | The notification settings API and Flutter settings UI shall read, display, update, and persist a moderation-push category independently of other notification categories. | Users need a usable way to exercise the confirmed opt-out policy. | Document review / User answer | AC-044 |
| FR-036 | Functional | Must | Migration shall preserve pre-case reports as private legacy intake and existing moderation outputs as effective visibility events without synthesizing cases, owner history, strikes, appeals, suspensions, or notifications; only reports accepted after enablement shall create or join cases. | Existing development data needs deterministic, non-misleading treatment. | Document review / User answer | AC-045 |
| FR-037 | Functional | Must | Flutter shall let an affected user open the current indexed version of their own moderated post from its owner-history entry. AppView shall allow the author through the direct post and thread-root detail reads while retaining normal moderation visibility for every other viewer and surface. Deleted or terminal-owner content remains unavailable. | Lets owners inspect the content referenced by a decision without weakening public moderation or retaining a second content snapshot. | User answer | AC-049 |
| NFR-001 | Non-functional | Must | Case resolution shall commit the decision, case projection, selected consequence records, any selected moderation output, strike changes, suspension state, and any required notification intent atomically. | Partial enforcement would make account state inconsistent. | Discovery | AC-021 |
| NFR-002 | Non-functional | Must | Moderator actions shall be attributable, timestamped, auditable, and protected against replay and stale concurrent decisions. | Safety enforcement requires operational accountability. | Discovery | AC-021, AC-027 |
| NFR-003 | Non-functional | Must | Affected-user responses, error paths, structured logs, metrics labels, and audit representations shall not expose reporter identities, report details, device IDs, internal notes, private evidence, credentials, email contents, or raw moderation source secrets. | Moderation data is sensitive and abuse-prone. | Existing requirements / Discovery | AC-014, AC-020, AC-027, AC-046 |
| NFR-004 | Non-functional | Must | New `/v1/*` APIs shall use camelCase JSON, standard error envelopes, opaque cursor pagination for lists, bounded limits, and canonical DID/AT-URI identifiers. | Required by the AppView API architecture. | API architecture | AC-030 |
| NFR-005 | Non-functional | Must | Suspension enforcement shall be centralized and fail closed for disallowed mutations rather than relying on Flutter to hide actions. | Modified or stale clients must not bypass suspension. | Discovery | AC-012 |
| NFR-006 | Non-functional | Must | Moderation notification dispatch shall use the existing durable retry and account-bound delivery infrastructure and shall treat provider acceptance as best-effort delivery rather than proof of device presentation. | External push systems cannot provide exactly-once presentation. | Codebase / Discovery | AC-023, AC-024 |
| NFR-007 | Non-functional | Must | Owner-facing moderation UI and push copy shall be localized, accessible, and distinguish account standing without suggesting that unused strikes are permitted violations. | Safety communication must be clear and usable. | Discovery | AC-015, AC-031 |
| NFR-008 | Non-functional | Must | AppView shall expose alertable signals using initial thresholds of five failed admin-auth attempts in five minutes, any strike-expiry work past its one-hour processing deadline, and any eligible moderation-notification work older than 15 minutes. | Moderation authority and delayed enforcement communication require measurable operational detection. | Document review / User answer | AC-047 |
| RULE-001 | Business rule | Must | Only an authenticated trusted moderator decision or trusted future adapter event may issue or overturn a strike or apply or lift a suspension; only the configured time policy may expire a strike automatically. | Reports and clients are not enforcement authorities. | Discovery | AC-004, AC-005, AC-027 |
| RULE-002 | Business rule | Must | A strike that was issued and not overturned becomes due for expiry processing at the same UTC wall-clock timestamp 12 calendar months after its server-recorded issuance time, clamped to the last valid target-month day. It stops counting only when expiry processing commits within the one-hour convergence window. | Implements the confirmed strike lifetime and effective-time model precisely. | User answer | AC-005, AC-038 |
| RULE-003 | Business rule | Must | One case may have at most one logical strike and contribute at most one active strike, regardless of attached reports or later effect reapplication. | Prevents duplicate punishment for one review episode. | User answer / Discovery | AC-004, AC-039 |
| RULE-004 | Business rule | Must | Three active strikes produce threshold suspension; a severe suspension remains a separate basis and does not require or fabricate three strikes. | Keeps standing and urgent enforcement truthful. | User answers | AC-006, AC-009, AC-042 |
| RULE-005 | Business rule | Must | A pending appeal does not pause, remove, or reduce the effect of the appealed decision. | Email appeal is not an authenticated automatic reversal. | User answer | AC-017, AC-018 |
| RULE-006 | Business rule | Must | Expired and overturned decisions remain visible in the affected user's history but do not count as active strikes. | Preserves transparency and correct standing. | User answers | AC-007, AC-014 |
| RULE-007 | Business rule | Must | At most one case may be open for a canonical subject at a time; later reports join it, and reports after closure create a new case. | Defines deterministic case grouping. | User answers | AC-002, AC-003 |
| RULE-008 | Business rule | Must | Suspension does not itself alter public PDS records; content visibility continues to be governed by explicit moderation outputs. | Moderation and user-owned data must remain separate. | AGENTS.md / Discovery | AC-028, AC-029 |
| RULE-009 | Business rule | Must | Notification work is created only for an authoritative event that adds, removes, or changes an owner-visible consequence or effective enforcement: consequential decisions, effect reversals, effect reapplications, explicit restorations, and expiry that changes enforcement. Report activity, `noAction`, appeal receipt/pending status, upheld outcomes without effect changes, and expiry without enforcement change create none. | Reports and status-only outcomes remain quiet while consequence changes are communicated. | Discovery / User answers | AC-023, AC-033, AC-038, AC-039 |
| RULE-010 | Business rule | Must | Disabling moderation pushes affects provider delivery only and does not hide account standing, history, or enforcement. | Notification preference must not become a policy bypass. | User answer | AC-024 |
| RULE-011 | Business rule | Must | `noAction` resolutions remain private moderator audit records and neither appear in owner history nor create moderation notifications. | Dismissed allegations should not disclose that a report exists. | User answer | AC-033 |
| RULE-012 | Business rule | Must | A severe suspension may coexist with an explicitly selected ordinary strike, but severe suspension shall never create a synthetic strike. | Keeps independent enforcement effects truthful. | User answer | AC-042 |
| RULE-013 | Business rule | Must | All approved reason codes except `other` are eligible for an explicitly selected ordinary strike; `other` may produce only a formal warning or visibility output. | Prevents catch-all account punishment. | User answers | AC-037 |
| RULE-014 | Business rule | Must | Severe suspension is eligible only for `harassment`, `hate`, `adult_or_graphic`, or `impersonation` and requires a private severity rationale. | Bounds immediate suspension authority. | User answer | AC-037 |
| RULE-015 | Business rule | Must | No reason category, visibility output, or report count automatically issues a strike or severe suspension. | Consequences require deliberate moderator action. | User answers | AC-004, AC-037 |
| RULE-016 | Business rule | Must | Each case with an owner-visible decision has at most one appeal lifecycle. Untrusted correspondence does not create or consume it; an authenticated moderator confirmation creates the pending lifecycle, after which repeated or later correspondence remains reviewable and may result in an audited effect change. | Keeps V1 appeal state compact without allowing spoofed intake to consume it. | User answer | AC-040, AC-043 |
| RULE-017 | Business rule | Must | Untrusted appeal correspondence from any email sender may be reviewed when it includes a valid public case reference, but only authenticated moderator confirmation may mark an appeal pending. Sender identity does not prove account ownership, authorize enforcement changes, or permit disclosure of private case data. | CraftSky has no verified member email. | Codebase / User answer | AC-043 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-007, FR-008 | Given an authenticated affected user, when they open account standing, then they can identify their current enforcement state, active strike count, threshold, and moderation history. |
| AC-002 | FR-001, RULE-007 | Given one or more reports concurrently target the same canonical subject while no case is open, when intake completes, then exactly one case is open and every accepted report is attached to it. |
| AC-003 | FR-002, FR-029, RULE-007 | Given all prior cases for a subject are resolved, when a new report is accepted, then a new case with a new public case reference is created without reopening or modifying prior case history. |
| AC-004 | FR-012, RULE-001, RULE-003, RULE-015 | Given any number of reports are attached to a case, when no trusted decision explicitly applies a strike, then no strike exists; when a qualifying decision applies one, then that case has one logical strike and contributes at most one active strike. |
| AC-005 | FR-013, FR-034, RULE-001, RULE-002 | Given a strike is issued and not overturned, when less than 12 clamped UTC calendar months have elapsed it counts as active; when its exact deadline is reached it becomes due for expiry processing; and it stops counting only when restart-safe processing commits no later than one hour afterward. |
| AC-006 | BR-002, FR-014, RULE-004 | Given an account has two active strikes, when an authorized decision atomically issues a third active strike, then the resulting account state is threshold-suspended and the stored active count is three. |
| AC-007 | FR-013, FR-016, FR-034, RULE-006 | Given a threshold-suspended account has no other suspension basis, when expiry processing atomically expires one strike and the count becomes two within one hour of its deadline, then threshold suspension is lifted in that commit and the expired strike remains in history; before that commit the stored suspension projection remains authoritative. |
| AC-008 | FR-006, FR-013, FR-016 | Given a threshold-suspended account, when an authorized moderator overturns one strike and no other suspension basis exists, then the original decision remains auditable, active count decreases, and access is restored. |
| AC-009 | BR-002, FR-015, RULE-004 | Given an account has fewer than three active strikes, when an authorized moderator records a documented severe-violation suspension, then the account becomes severely suspended without synthetic strikes being created. |
| AC-010 | FR-017 | Given an active severe suspension, when associated strikes expire or threshold count falls below three, then severe suspension remains; when an authorized explicit restoration or reversal is recorded, then that severe basis is removed. |
| AC-011 | FR-018 | Given a suspended user, when they use any authenticated read; moderation history or appeal contact; reporting; logout/account deletion; notification seen, notification/language/device preference, recent-search, or saved-content management; or cancellation/deletion of existing private scheduled or migration work, then the capability remains available without changing suspension. |
| AC-012 | FR-018, NFR-005 | Given a suspended user uses a modified or stale client, when they attempt an upload, link preview, onboarding completion, public profile/business/account-type edit, follow/unfollow, public post/event create or edit, scheduling/publication, like/unlike, repost/unrepost, pin creation, migration/import start or confirmation, or any unclassified mutation, then AppView rejects it using the standard error contract before any PDS, local, upload, or external side effect. |
| AC-013 | FR-018 | Given a suspended user, when they submit a valid report, then the report is accepted and grouped normally without changing their suspension state. |
| AC-014 | BR-001, FR-007, FR-008, NFR-003, RULE-006 | Given an affected user has formal-warning, visibility, active-strike, expired, and overturned consequential decisions, when history is requested, then all appear chronologically with user-safe explanations and consequences, while private reports, no-action outcomes, evidence, and moderator data are absent. |
| AC-015 | BR-001, FR-008, NFR-007 | Given an account has zero through three active strikes or a severe suspension, when the page renders, then account standing is clearly and accessibly communicated without implying that remaining strikes are an allowance. |
| AC-016 | BR-003, FR-009 | Given an appealable decision, when the user invokes or cannot invoke the appeal CTA, then they have a prefilled `mailto:` option addressed to `moderation@craftsky.social` whose subject contains the public case reference and copyable address/reference fallbacks. |
| AC-017 | FR-010, RULE-005 | Given the user opens or abandons the email composer, when they return to CraftSky, then no appeal receipt or enforcement change is inferred from the CTA launch. |
| AC-018 | BR-003, FR-006, FR-011, RULE-005 | Given moderation staff receive correspondence for an owner-visible case, when an authenticated moderator confirms one pending appeal and later resolves it as `upheld` or `changed`, then the affected user's history reflects the authoritative state and reversal events identify any effects changed; correspondence alone creates no pending state. |
| AC-019 | BR-004, FR-003 | Given the Retool moderator is authenticated, when they request the queue with supported state filters and pagination, then AppView returns only the authorized curated case representation and an opaque next cursor when more results exist. |
| AC-020 | BR-004, FR-004, NFR-003 | Given the Retool moderator requests a case, then AppView returns the evidence and decision context needed for review, while affected-user APIs cannot obtain those private fields. |
| AC-021 | BR-004, FR-005, FR-014, FR-024, NFR-001, NFR-002 | Given a valid decision command, when it is retried with the same idempotency identity it produces no duplicate effects; when it uses a stale revision it is rejected; and when it succeeds all required projections and outbox work commit together. |
| AC-022 | BR-005, FR-024, FR-025 | Given trusted commands from the admin adapter and a source-neutral test adapter, when equivalent normalized inputs invoke the adjudication service, then the same policy validation and transactional effects apply, replayed source identities are no-ops, and neither adapter has a direct moderation-table mutation interface. Live Ozone payload conformance is not asserted in V1. |
| AC-023 | BR-006, FR-019, NFR-006, RULE-009 | Given a consequential decision, effect reversal, effect reapplication, or explicit restoration adds, removes, or changes an owner-visible consequence or effective enforcement, or strike expiry changes effective enforcement, then one durable moderation notification intent is created atomically and is eligible for retry. Report activity, `noAction`, appeal status-only changes, upheld outcomes without effect changes, and expiry without enforcement change create none. |
| AC-024 | FR-020, NFR-006, RULE-010 | Given moderation pushes are disabled, when a moderation decision commits, then history and enforcement update but no provider delivery is attempted; when enabled, eligible installations receive normal durable delivery attempts. |
| AC-025 | BR-006, FR-021 | Given a moderation push is delivered and tapped, then it is presented as a CraftSky system notice without an actor and opens the corresponding moderation-history entry only after the intended retained account is active. |
| AC-026 | FR-022 | Given moderated content is later edited, hidden, removed, or deleted, when the owner or moderator opens the case, then the attached report's canonical reference and safe snapshot identify what was decided, or decision-specific snapshot data is used only where that report data was insufficient. |
| AC-027 | FR-023, NFR-002, NFR-003, RULE-001 | Given a member token, development moderation token, missing credential, or invalid admin credential, when an admin moderation endpoint is called, then access is denied without sensitive leakage; a valid admin action is attributed to the fixed trusted moderator identity. |
| AC-028 | BR-007, RULE-008 | Given any strike, threshold suspension, severe suspension, expiry, or reversal, when enforcement completes, then no PDS record, DID, or PDS account is deleted or mutated by that enforcement transition. |
| AC-029 | FR-026, RULE-008 | Given a decision applies or reverses warning, hide, or takedown behavior, then existing moderation-output policy governs AppView visibility and no parallel visibility state disagrees with it. |
| AC-030 | FR-003, FR-007, NFR-004 | Given a new moderation list or error response, then it follows `/v1/` conventions for camelCase JSON, opaque cursors, bounded limits, canonical identifiers, and `{error, message, requestId}` errors. |
| AC-031 | FR-008, NFR-007 | Given supported locales, text scaling, screen readers, light/dark appearance, and narrow or wide layouts, then moderation standing, history, appeal actions, and status distinctions remain understandable and operable. |
| AC-032 | FR-012, FR-027 | Given a moderator resolves a case, then exactly one allowed disposition is required; `violation` without a consequence and `noAction` with any consequence are rejected without side effects. |
| AC-033 | FR-007, FR-027, RULE-009, RULE-011 | Given a case resolves as `noAction`, then the resolution remains in the private audit trail but creates no owner-history item, account effect, or moderation notification. |
| AC-034 | FR-001, FR-029 | Given no accepted report exists, when the moderator attempts to create a V1 case, then the request is rejected; given a resolved case, when an appeal or reversal is recorded, then history appends without changing the case back to open. |
| AC-035 | FR-028 | Given a violation selects `formalWarning`, viewer-facing `warn`, or both, then owner history and subject visibility receive exactly the selected independent effects. |
| AC-036 | FR-009, FR-032 | Given a case is created, then its UUIDv4 is presented as `MOD-<case UUID>`, is matched case-insensitively in appeal/deep-link surfaces, and exposes no internal decision or event ID. |
| AC-037 | FR-012, FR-033, RULE-013, RULE-014, RULE-015 | Given each approved reason and consequence combination, then every approved reason except `other` may explicitly issue an ordinary strike, only the four approved severe categories with a severity rationale may suspend immediately, `other` cannot punish the account, and no reason produces either effect automatically. |
| AC-038 | BR-006, FR-006, FR-016, FR-019, FR-034, RULE-002, RULE-009 | Given a strike reaches its clamped UTC calendar deadline, then it becomes due while the stored projection remains authoritative; restart-safe processing atomically makes expiry, standing, and threshold-only restoration effective no later than one hour afterward, and creates one notification only when effective enforcement changes. |
| AC-039 | BR-006, FR-006, FR-019, FR-030, RULE-003, RULE-009 | Given a decision has several active effects, when an authorized reversal selects a subset, then only that subset is negated; a later command may reapply an effect in the same resolved case when it supplies a rationale and distinct idempotency identity, and the full sequence remains auditable. Reapplying a strike reactivates the case's one logical strike with a new issuance timestamp and 12-month lifetime rather than creating a second strike. Each reversal or reapplication creates exactly one notification only when it changes an owner-visible consequence or effective enforcement. |
| AC-040 | FR-011, FR-031, RULE-016 | Given a case has an owner-visible decision, then authenticated moderator confirmation may create one pending appeal and resolve it as `upheld` or `changed`; repeated or later emails attach to that lifecycle and remain reviewable, reversal events identify changes, and expiry/restoration events offer no separate appeal action. |
| AC-041 | FR-018 | Given a suspended user, when they add or remove a block/mute or remove an owned post, event, business profile, or pin, then the action is permitted; creating or editing public content remains denied. |
| AC-042 | FR-015, RULE-004, RULE-012 | Given an eligible severe decision, when the moderator explicitly selects severe suspension and an ordinary strike, then both independent effects are recorded; when only severe suspension is selected, no strike is created. |
| AC-043 | FR-011, RULE-016, RULE-017 | Given correspondence from any email sender includes a valid public case reference, when staff receive it, then it remains untrusted intake and creates no pending appeal, disclosure, or enforcement change; when an authenticated moderator confirms it for an owner-visible case, then the one appeal lifecycle becomes pending without treating the sender as authenticated. |
| AC-044 | FR-020, FR-035 | Given an authenticated user opens notification settings, when they change the moderation-push preference, then Flutter displays and persists the independent category through the existing API; after reload the saved value is shown without changing other categories. |
| AC-045 | FR-036 | Given pre-case reports and moderation outputs exist, when the migration and feature enablement complete, then the rows remain intact and old outputs retain existing visibility behavior, but no synthetic case, owner-history item, strike, appeal, suspension, or notification is created; a newly accepted report follows normal case grouping. |
| AC-046 | NFR-003 | Given successful and failing moderation operations contain unique sensitive sentinels, when responses, errors, structured logs, metrics labels, and audit representations are captured, then reporter identity, report/evidence/email content, device IDs, credentials, and raw source secrets are absent from every unauthorized surface. |
| AC-047 | NFR-008 | Given operational signals are observed, when five admin-auth failures occur within five minutes, expiry work passes its one-hour deadline, or eligible moderation-notification work becomes older than 15 minutes, then the corresponding alert condition is active; values below each threshold do not activate it. |
| AC-048 | FR-021 | Given a moderation notification targets a different retained account, when it is opened, then Flutter resolves the existing opaque account-subscription binding, activates that exact current account lease, rechecks it, and only then navigates to the case; invalid, ambiguous, removed, or stale bindings show no case details and do not navigate. |
| AC-049 | FR-037 | Given owner history contains a valid post snapshot, when the affected author selects View post, then Flutter opens the existing post-thread route and AppView returns the current indexed post even when a hide or takedown output applies; the same direct reads by another viewer remain not found, and malformed, mismatched, deleted, or terminal-owner references do not become navigable content. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Two first reports for one subject arrive concurrently. | One open case is created and both reports attach to it. | FR-001, RULE-007 |
| EC-002 | Two moderators or dashboard tabs resolve the same case. | Only the command using the current revision succeeds; the stale command has no effects. | FR-005, NFR-002 |
| EC-003 | A successful decision response is lost and Retool retries. | The idempotency identity returns/reconstructs the existing result without another strike, output, suspension, or notification. | FR-005, FR-024 |
| EC-004 | Several reports describe one incident. | Report count is visible only as evidence and cannot create more than one case strike. | FR-012, RULE-003 |
| EC-005 | One incident involves several different records. | Each canonical subject has its own case; moderator operating guidance advises against multiple strikes for one materially connected incident, but V1 does not technically group incidents across cases. | FR-012, RULE-003 |
| EC-006 | Third strike is issued concurrently with another strike expiry or reversal. | Serialized transactional evaluation produces one correct final count and enforcement state. | FR-013, FR-014, NFR-001 |
| EC-007 | Threshold and severe suspension are both active. | Removing one basis does not restore participation while the other remains active. | FR-016, FR-017, RULE-004 |
| EC-008 | User appeals from an email address unrelated to their account. | The public case reference locates untrusted correspondence, but no appeal becomes pending until a moderator confirms it; sender identity receives no private case disclosure and cannot authorize reversal. | FR-009, FR-011, RULE-001, RULE-017 |
| EC-009 | Device has no configured email application. | The user can copy the moderation address and public case reference manually. | FR-009 |
| EC-010 | User edits the generated email subject. | No appeal state changes automatically; moderation staff may locate the case through other supplied information or request the public case reference. | FR-010, FR-011 |
| EC-011 | Moderated subject is deleted before review. | Moderator and owner views use retained safe snapshot/reference data or clearly state that live content is unavailable. | FR-022 |
| EC-012 | Push preference changes while delivery is queued. | Dispatch applies the defined current-preference policy without changing the underlying moderation decision or history. | FR-020, NFR-006 |
| EC-013 | Push is accepted by the provider but not shown by the OS. | Delivery remains best effort; moderation history is the durable source of truth and no duplicate decision is created. | NFR-006, NG-012 |
| EC-014 | Suspended user attempts to report repeatedly or abusively. | Reporting remains available but uses existing and future report abuse/rate controls; it never changes the reporter's suspension automatically. | FR-018 |
| EC-015 | A handle changes after report or decision. | Canonical DID and AT URI references remain authoritative; handle snapshots are display/audit context only. | FR-022, NFR-004 |
| EC-016 | Retool credential is compromised or removed. | Credential rotation/revocation denies future requests without altering prior attributed audit history. | FR-023, NFR-002 |
| EC-017 | Strike is issued on February 29. | Its 12-calendar-month expiry is February 28 at the same UTC wall-clock time in the following non-leap year. | RULE-002 |
| EC-018 | Expiry worker is stopped at the deadline. | The strike is due but the stored projection remains authoritative until restart-safe processing atomically converges expiry, standing, and restoration within one hour; retries do not duplicate events, and notification is created only if effective enforcement changes. | FR-019, FR-034 |
| EC-019 | Severe decision also selects a strike. | Both effects are recorded independently; later severe restoration does not remove the strike, and strike expiry does not remove severe suspension. | RULE-004, RULE-012 |
| EC-020 | An appeal changes only the strike effect. | The strike and any resulting threshold basis are negated while an unselected takedown or severe basis remains active. | FR-030 |
| EC-021 | Moderator reapplies a reversed effect in a resolved case. | The command succeeds only with a rationale and distinct idempotency identity; the original application, reversal, and later application remain auditable. | FR-029, FR-030 |
| EC-022 | Report resolves without a violation finding. | The private audit records `noAction`, but the affected owner is neither notified nor shown the case. | RULE-011 |
| EC-023 | Suspended user deletes an owned post. | The user-requested deletion proceeds normally; moderation enforcement itself still never deletes PDS records. | FR-018, RULE-008 |
| EC-024 | Suspended user updates private settings or saved content. | Notification/language/device preferences, notification seen state, recent searches, saves, and saved folders remain manageable without changing suspension. | FR-018, FR-035 |
| EC-025 | Suspended user starts an upload, migration/import, onboarding completion, or publication workflow. | The operation is denied before any local, external, upload, or PDS side effect; cancellation/deletion of existing private pending work remains available. | FR-018, NFR-005 |
| EC-026 | Strike deadline passes while expiry work is pending. | The projected strike and threshold suspension remain effective until the worker commit, which must occur no later than one hour after the deadline. | FR-034, RULE-002 |
| EC-027 | Untrusted correspondence arrives before the owner's real appeal. | It remains intake only and does not consume the single appeal lifecycle; only moderator confirmation creates pending status. | FR-011, RULE-016, RULE-017 |
| EC-028 | Notification targets another retained account. | The existing opaque binding activates the exact current lease before navigation; stale, ambiguous, invalid, or removed bindings reveal no case data. | FR-021 |

## 15. Data / Persistence Impact

- New data concepts:
  - Moderation cases keyed by a UUIDv4 presented publicly with a `MOD-` prefix.
  - Association from existing reports to review cases.
  - Append-only moderator decisions with explicit disposition and independent formal-warning, visibility, strike, and suspension effects.
  - Effect-level reversals and later rationale-backed effect applications preserved in append-only history.
  - Shared approved report/decision reason codes, localized owner copy, optional user-safe detail, and required private evidence notes for violations.
  - Strike issuance, expiry, and overturn history.
  - Account moderation/enforcement projection with independent threshold and severe suspension bases.
  - One appeal lifecycle per case with repeated correspondence, a pending state, and `upheld` or `changed` resolution.
  - Moderation notification category/detail and delivery preference.
  - Trusted source-system replay identities and moderator attribution.
- Changed data concepts:
  - Existing reports gain case membership but remain immutable intake evidence.
  - Existing moderation outputs remain the visibility-enforcement stream and become an adjudication effect.
  - Notification preferences and payloads gain a moderation category/system-notice representation.
- Migration required: Yes. Migrations preserve existing report and moderation-output rows without synthesizing cases or owner-facing/enforcement state. Existing outputs remain effective for visibility; only reports accepted after feature enablement create or join cases.
- Backwards compatibility: There are no production users, but the full migration chain, existing tests, and existing API behavior must remain valid. New API fields and routes should be additive under `/v1/`.
- Retention: Case, decision, strike, suspension, and appeal audit history is durable unless a separate approved owner-deletion or retention policy requires removal.

## 16. UI / API / CLI Impact

- UI:
  - Add a settings entry and authenticated account-standing/moderation-history page.
  - Display the three-strike threshold, active count, enforcement state, decision consequences, reasons, dates, expiry, and appeal state.
  - Add an appeal `mailto:` CTA and public case-reference copy fallbacks.
  - Add a moderation push preference.
  - Add dedicated system moderation-notification handling and deep-link routing through the existing secure account-subscription binding and automatic exact-account activation.
  - Ensure suspended sessions enter restricted behavior without relying on UI hiding for server enforcement, while preserving the complete private-maintenance, safety, deletion, and owned-public-content-removal allowlist.
- API:
  - Add authenticated affected-user account-standing/history reads.
  - Add production admin queue, case-detail, decision, appeal, effect-change, and restoration capabilities.
  - Add centralized suspension authorization behavior and a stable error contract.
  - Keep route names and exact bodies for coding design, while preserving `/v1/`, camelCase, standard errors, and opaque pagination.
- CLI:
  - No moderator CLI is required when Retool is operational.
  - A narrowly scoped operational inspection or recovery command may be considered later but must reuse the adjudication service and authorization/audit requirements.
- Background jobs:
  - Evaluate strike expiry and converge threshold suspension restart-safely within one hour of each expiry deadline.
  - Dispatch moderation push notifications through the existing durable delivery workers.
  - Future Ozone ingestion and report forwarding are Stage 3, not required for initial Retool operation.
- Retool:
  - Configure queue, case detail, decision, appeal, effect-change, and restoration actions against admin APIs.
  - Store the scoped credential in Retool secret management.
  - Do not connect Retool with database write privileges.
  - Present disposition and consequence dimensions separately and validate reason/effect eligibility. Keep one-strike-per-connected-incident guidance in moderator operating documentation rather than dedicated Retool behavior.

## 17. Security / Privacy / Permissions

- Authentication:
  - Affected-user APIs use normal CraftSky session authentication.
  - Admin APIs use a distinct production moderator credential; member sessions and development moderation credentials are insufficient.
  - The initial credential maps server-side to one fixed moderator identity and can later be replaced by individual identity authentication.
- Authorization:
  - Admin reads and commands are denied by default outside the moderator scope.
  - Suspension enforcement is server-side and deny-by-default for mutations.
  - Allowed while suspended: all authenticated reads; logout and all account-deletion steps; reports; block/mute add/remove; owned post/event deletion; business-profile deletion; pin removal; notification seen state; notification/language/device preferences; recent-search create/delete; saved-post add/remove; saved-folder create/update/delete; and cancellation/deletion of existing private scheduled-post media/posts and Instagram migration verification/import/account data.
  - Denied while suspended: video authorization; image/scheduled-media uploads; link-preview creation; onboarding completion; profile/customisation/account-type/business updates; follow/unfollow; event/post creation or update; scheduled-post/media creation or update and publication; like/unlike; repost/unrepost; pin creation; Instagram verification/import creation, confirmation, update, or suggestion acceptance; and any mutation not explicitly classified.
  - Source and moderator identities are derived from authenticated context, not trusted from arbitrary request fields.
- Sensitive data:
  - Reporter identity, report details, device IDs, internal notes, evidence, credential material, and raw source metadata remain private.
  - User-safe reason codes and explanations are separately authored and validated for affected-user display.
  - Public case references are opaque, unguessable references and do not themselves grant decision authority or prove email-sender ownership.
- Abuse cases:
  - Duplicate or coordinated reports cannot automatically issue strikes.
  - Idempotency and optimistic concurrency prevent replayed or competing decisions.
  - Report submission retained during suspension remains subject to rate limiting and abuse monitoring.
  - Email sender identity cannot trigger automatic restoration.
  - `other` cannot be used as a catch-all justification for strike or severe suspension.
  - Severe suspension is restricted to the approved reason categories and requires a private severity rationale.
  - Retool never receives database mutation credentials.
  - Logs use safe case/request identifiers and avoid raw sensitive moderation content.

## 18. Observability

- Audit events record case creation, decisions, moderator-confirmed appeals, effect changes, strikes, expiries, and suspension changes with safe actor and request identifiers.
- Structured logs exclude report text, private evidence, email contents, tokens, and raw push credentials.
- Initial operational metrics cover failed admin authentication and stalled strike-expiry or notification work.
- Alerts activate for five failed admin-auth attempts in five minutes, strike-expiry work past its one-hour processing deadline, and eligible moderation-notification work older than 15 minutes. Broader moderation-product analytics may be added after operations establish a need.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Report intake, adjudication, strikes, and outputs become conflated. | Duplicate punishment and unclear audit history. | Keep reports as evidence, cases as review units, decisions append-only, and outputs as visibility effects. |
| RISK-002 | Suspension is implemented through owner lifecycle. | Users lose appeal/account access or moderation becomes accidentally irreversible. | Add separate moderation enforcement state and explicit suspended capability rules. |
| RISK-003 | Retool can mutate database tables directly. | Invalid transitions, missing audit records, and bypassed policy. | Permit admin API access only; no database write credential. |
| RISK-004 | Two moderators or retries issue duplicate strikes. | Incorrect suspension and unfair enforcement. | Require idempotency, source replay IDs, optimistic concurrency, uniqueness constraints, and atomic transactions. |
| RISK-005 | User-facing explanation leaks reporter or internal evidence. | Privacy harm, retaliation, or moderation evasion. | Store and serialize user-safe explanation separately; test field-level non-disclosure. |
| RISK-006 | Three-strike UI appears to permit two violations. | Policy is misunderstood or gamified. | Lead with account standing and consequences; present threshold visually without allowance language. |
| RISK-007 | Push delivery is mistaken for guaranteed notice. | Users miss decisions or support disputes arise. | Keep history durable, expose it in-app, retry delivery, and describe push as best effort. |
| RISK-008 | Push preferences suppress all evidence of moderation. | Affected users cannot understand enforcement. | Preference suppresses provider push only; history and standing remain mandatory. |
| RISK-009 | Severe and threshold suspension bases overwrite each other. | Account restores too early or remains suspended incorrectly. | Model independent active bases and derive effective enforcement state. |
| RISK-010 | Future Ozone semantics are copied prematurely or cannot map to CraftSky policy. | Rework or policy drift. | Keep adapters outside a stable CraftSky adjudication service and retain source/event identities. |
| RISK-011 | Email appeals are treated as authenticated submissions. | Spoofing or automatic unauthorized restoration. | Require manual trusted receipt/resolution and never infer status from `mailto:` launch or sender alone. |
| RISK-012 | Suspended users bypass restrictions through stale clients or unguarded endpoints. | Continued prohibited participation. | Centralize fail-closed mutation authorization and test every route category. |
| RISK-013 | One incident produces several cases and strikes because V1 has no incident-group invariant. | A user may reach suspension from one connected episode. | Document the operating policy, train the single moderator, and preserve audit data for later incident grouping if operations demonstrate a need. |
| RISK-014 | Optional owner-facing free text leaks private report or evidence details. | Reporter exposure, retaliation, or policy evasion. | Keep the field physically separate, never derive it automatically, provide safe-writing guidance, and test API field isolation. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Retool can securely call HTTPS APIs with a stored scoped credential. | A different admin client or authentication mechanism will be needed. |
| ASM-002 | One trusted moderator is sufficient for initial operations. | Individual authentication, roles, assignment, and conflict workflows must move into initial scope. |
| ASM-003 | The existing report reason taxonomy provides sufficient coverage for initial moderator findings and localized owner explanations. | The shared taxonomy must be expanded or split before launch. |
| ASM-004 | Email volume is low enough for manual appeal receipt and resolution tracking in Retool. | Native appeal intake or email automation must be added sooner. |
| ASM-005 | Existing durable push infrastructure can be extended with a system moderation category. | Notification storage, fan-out, or client parsing may require a broader redesign. |
| ASM-006 | Restart-safe background work can persist every strike expiry and threshold restoration within one hour of the clamped UTC calendar deadline. | A request-time fallback or a wider communicated restoration window will be required. |
| ASM-007 | CraftSky remains pre-production with no active users during initial migration. | Backfill, rollout, compatibility, and user-notification plans become required. |
| ASM-008 | The existing opaque account-subscription binding and retained-session activation flow remains the authoritative notification routing mechanism. | Moderation notification routing needs a separate reviewed account-binding design rather than adding raw DIDs to provider payloads. |

## 21. Open Questions

- [ ] Non-blocking for requirements: Choose exact admin and affected-user route names during coding design.
- [ ] Non-blocking for requirements: Choose the initial scoped moderator credential format, rotation procedure, and deployment configuration.
- [ ] Non-blocking for requirements: Approve localized moderation page, email template, notification, and suspension error copy.
- [ ] Non-blocking for requirements: Decide whether moderation system notices also appear in the general in-app notification inbox or only in moderation history plus push.
- [ ] Non-blocking for requirements: Bind the confirmed suspended-capability categories to every concrete current and queued route during coding design; the product policy is fixed above and unclassified mutations fail closed.
- [ ] Non-blocking for requirements: Define safe moderator guidance for optional owner-facing details, severe rationales, and identifying materially connected incidents.

No blocking product questions remain for acceptance-test design.

## 22. Review Status

Status: Revised after document review; pending explicit approval

Risk level: High

Review recommended: Required

Reviewer:

Date:

Notes: This feature changes production moderator authority, private safety data, account mutation permissions, and suspension behavior. The requirements include confirmed decisions through Q40 and resolve document-review findings DR-001 through DR-014. Explicit approval remains required before implementation proceeds.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-09-09-moderation-strikes/01-requirements.md`
- Next test specification: `docs/changes/2026-09-09-moderation-strikes/02-acceptance-tests.md`
- Must-cover requirement IDs: `BR-001` through `BR-007`, `FR-001` through `FR-036`, `NFR-001` through `NFR-008`, and `RULE-001` through `RULE-017`.
- Suggested test levels:
  - Store/integration tests for report-origin case grouping, no-reopen lifecycle, dimensional decision validation, append-only effect changes, public references, reason eligibility, strike expiry, overlapping suspension bases, appeals, replay, concurrency, and privacy.
  - Route tests for admin authentication, user history, no-action privacy, pagination, error envelopes, and the complete suspended capability matrix including blocks, mutes, and owned-content removal.
  - Regression tests for existing report intake and moderation-output visibility behavior.
  - Push tests for atomic enqueue, preference suppression, system payloads, retry, account routing, and deep links.
  - Flutter repository/model tests for wire parsing and standing derivation boundaries.
  - Flutter widget tests for all history/standing states, appeal fallback, notification routing, accessibility, localization, and responsive layouts.
  - Manual Retool verification for queue, case review, stale conflict handling, decisions, appeal recording, effect changes, and credential revocation.
- Blocking open questions: None for test design; high-risk review approval remains the stage gate.
