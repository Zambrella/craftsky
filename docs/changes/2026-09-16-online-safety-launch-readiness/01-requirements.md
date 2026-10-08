# Requirements: Pre-Launch Online Safety Readiness

## 1. Initial Request

Review the documents under `online-safety/` and define the product, engineering,
operational, privacy, and policy changes CraftSky must complete before a worldwide
public launch that includes the UK. The initial concerns were CSAM detection for
images received through Tap or held in CraftSky storage and broader reporting
options. Discovery expanded the scope to the connected launch controls required
to handle detections and reports safely.

Confirmed direction:

- Scan relevant images once at Tap index time, including records created outside
  the CraftSky app.
- Use IWF Image Intercept as the intended production known-CSAM matching service.
- Use a deterministic non-production scanner stub until IWF access and technical
  documentation are available; production must remain fail-closed without IWF.
- Scan every user-controlled image CraftSky renders: post and business images,
  profile avatars and banners, and link-preview thumbnails.
- Hide an entire post or business event until all of its images clear. Profiles
  instead retain their last cleared avatar/banner or use a neutral placeholder.
- Keep the current author-only synthetic post behavior before refresh; AppView
  controls visibility after refresh.
- Treat the first successful scan of an immutable blob as durable for launch; do
  not periodically rescan automatically.
- Keep email as the external reporting and appeal route, provided receipt,
  tracking, escalation, and outcome handling are operationally reliable.
- Extend the existing CraftSky moderation system for this slice and ignore Ozone.
- Launch worldwide for people aged 16 and over, with child-safe controls because
  children are likely to access the service.
- Use a founder-led moderation operation with named help where needed.
- Do not launch video.

## 2. Current Codebase Findings

- Relevant files:
  - `online-safety/registers/compliance-gap-register.md` records 22 P0 launch
    blockers, 12 P1 gaps, and one P2 gap.
  - `online-safety/assessments/illegal-content-risk-assessment.md` is incomplete
    and not yet a suitable and sufficient approved assessment.
  - `online-safety/assessments/childrens-access-assessment.md` concludes that
    CraftSky is likely to be accessed by children but awaits approval.
  - `online-safety/operations/illegal-content-response.md` describes the intended
    restricted response process but is not operationally approved.
  - `appview/internal/api/report_request.go` defines the current limited report
    taxonomy.
  - `appview/internal/api/report_store.go` requires an authenticated reporter DID,
    stores private reports, and attaches accepted reports to moderation cases.
  - `appview/migrations/000072_moderation_cases.up.sql` defines cases, append-only
    events, decisions, effects, strikes, and appeals.
  - `appview/internal/api/image_validator.go` validates image formats and resource
    limits but performs no harmful-content detection.
  - `appview/internal/api/blob.go` uploads immediate images to the owner's PDS.
  - `appview/internal/scheduledposts/media_service.go` stores private scheduled
    media with integrity metadata but no CSAM scan.
  - `appview/internal/accountdeletion/` implements narrow owner-authorized PDS
    record deletion and private-data cleanup, but there is no legal-hold boundary.
  - `app/lib/moderation/` contains signed-in report UI, account standing, history,
    and an email-based appeal entry point.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` fixes the
    `/v1/`, authentication, error, JSON, identifier, and pagination conventions.
- Existing patterns:
  - Reports are allegations and duplicate reports are allowed.
  - Moderation actions are append-only and reversible.
  - AppView warnings, hides, and takedowns affect CraftSky presentation without
    deleting source PDS records.
  - Blocks are public AT Protocol records; mutes, reports, cases, and appeals are
    private AppView data.
  - Viewer-relative policy is centralized and already filters major surfaces.
- Current behavior:
  - No CSAM hash matching, quarantine state, scanner queue, specialist incident
    record, legal hold, restricted evidence store, or NCA reporting workflow exists.
  - Indexed images may be surfaced after technical validation without a safety scan.
  - Current report reasons omit several serious harms, and sanction eligibility is
    narrower than the draft policies require.
  - Reporting requires a signed-in member. External email and appeal handling are
    not proven end to end.
  - There is no age declaration, age assurance, suspected-underage workflow, or
    child-specific product state.
  - Account deletion can purge reports and cases without preserving evidence that
    must be retained under an approved hold.
  - Existing public pages still contain stale 13+ and privacy/safety statements.
- Constraints discovered:
  - CraftSky should proceed as a regulated UK user-to-user service likely to be
    accessed by children.
  - Moderation controls CraftSky surfaces only and must not be represented as
    network-wide deletion.
  - Reports, incidents, restricted evidence, legal holds, and age signals are
    private data and must not become PDS records.
  - Automated detection must not be treated as guilt or directly create sanctions.
  - Index-time-only scanning allows CraftSky-mediated bytes to reach a user's PDS
    before Tap ingestion; specialist review must approve this boundary.
  - Public policies may describe only implemented and tested behavior.
  - No lexicon change is expected unless implementation introduces a new public
    safety field or `social.craftsky.*` record.
- Test/build commands discovered:
  - `just test` runs the AppView test suite against the development PostgreSQL stack.
  - `just appview-check` is the release-equivalent AppView verification gate.
  - Flutter tests run from `app/` with `flutter test <paths>`.
  - Dart/Flutter static analysis uses the repository analysis workflow.

## 3. Clarifying Questions And Decisions

### Q1: What geography does the launch gate cover?

Answer: Worldwide, including the UK.

Decision / implication: UK P0 obligations are launch blockers, while wider
jurisdictional review remains required for worldwide operation.

### Q2: Where should image scanning occur?

Answer: Once at Tap index time.

Decision / implication: All relevant records, including those created outside the
CraftSky app, follow one scan path. AppView must not surface an image until every
required blob clears. On 17 September 2026, the accountable owner confirmed that
specialist/provider approval of the PDS-before-index tradeoff had been obtained.
The documentary reference remains required before implementation completion or
launch approval.

### Q3: Which scanner should CraftSky use?

Answer: IWF Image Intercept.

Decision / implication: Build a narrow adapter and non-production stub now. Do not
invent the private IWF protocol. Production cannot report healthy or clear images
until real IWF access is configured and validated. Coding planning is explicitly
limited to the protocol-neutral state model, persistence, queues, visibility,
readiness guards, and deterministic stub; the production IWF adapter remains out of
scope until the approved contract is available.

### Q4: Should CraftSky use a novel-content classifier at launch?

Answer: No.

Decision / implication: Launch detection is limited to the approved IWF
known-content matching capability. Novel-CSAM classifiers are deferred.

### Q5: Is email enough for non-user reports and appeals?

Answer: Yes, if it is operationally tracked.

Decision / implication: A public web form is not required for this release. Email
receipt must create an auditable external allegation, incident, complaint, or
appeal without fabricating a reporter DID.

### Q6: What moderation platform is in scope?

Answer: The existing CraftSky moderation system; ignore Ozone for this slice.

Decision / implication: Requirements extend current private reports, cases,
decisions, effects, standing, and admin APIs.

### Q7: What is the launch staffing model?

Answer: Founder-led, with help where needed.

Decision / implication: Named cover, authorization, training, availability,
conflicts, and wellbeing safeguards must be documented and approved.

### Q8: What is the minimum launch age?

Answer: 16 worldwide, subject to a higher local minimum.

Decision / implication: Registration, product copy, policies, enforcement, and
support must be consistent. This is a contractual threshold, not a claim of highly
effective age assurance.

### Q9: Is video included at launch?

Answer: No.

Decision / implication: Production video creation/upload must be unavailable, and
the launch scanner scope does not need video support.

### Q10: What response deadline applies to qualifying intimate-image complaints?

Answer: Use the response target confirmed through specialist review.

Decision / implication: The current compliance register states a 48-hour workflow,
but this document does not replace legal review. The approved target is a launch
blocker and must be measurable from recorded receipt.

### Q11: Which image surfaces are in scope?

Answer: Every user-controlled image CraftSky renders.

Decision / implication: Post and business images, Bluesky profile avatars and
banners, and link-preview thumbnails use the scan gate regardless of namespace or
which compatible client created the record.

### Q12: What is hidden while scanning is pending?

Answer: Hide complete posts and business events, including posts with link-preview
thumbnails. Profiles remain available with their last cleared image or a neutral
placeholder.

Decision / implication: The current author-only synthetic post may remain visible
locally before refresh because the author already possesses the image. It does not
qualify the post for AppView visibility to anyone.

### Q13: What should happen after a scanner match?

Answer: Suppress the subject immediately, create a restricted incident, and do not
notify the owner until human review produces a moderation decision.

Decision / implication: Owner notices use the broad reason `Child safety violation`
and do not reveal IWF, hashes, reporting status, evidence, or investigative methods.

### Q14: Which scan states are required?

Answer: `pending`, `clear`, `match`, `unavailable`, and `error`.

Decision / implication: Do not invent a generic `suspected` state before IWF's real
contract demonstrates a distinct uncertain outcome.

### Q15: What happens after repeated scanner failure?

Answer: The subject remains non-visible indefinitely until a valid result arrives
or its source is deleted.

Decision / implication: Use bounded exponential retries, then an operator-visible
dead-letter state with manual retry. Never auto-clear or delete the PDS source.

### Q16: Are cleared blobs periodically rescanned?

Answer: No, not automatically at launch.

Decision / implication: Reuse the first valid result for an immutable blob. Permit
targeted operator-triggered rescans and add broader rescanning only if IWF guidance
or specialist review requires it.

### Q17: What evidence is preserved after a match?

Answer: Preserve identifiers, provider reference, timestamps, and integrity
metadata immediately. Preserve image bytes only when the approved CSEA/NCA
procedure requires it.

Decision / implication: A match does not automatically create another copy of the
image. Required bytes use the restricted evidence store and approved expiry.

### Q18: How does a confirmed detection enter ordinary enforcement?

Answer: An authorized safety moderator creates a `systemDetected` moderation case
linked to the restricted incident.

Decision / implication: The case is not a user report and contains only safe
incident references. Existing decisions, effects, standing, notices, history, and
appeals then apply.

### Q19: Must a moderator view matched bytes?

Answer: No.

Decision / implication: Default review uses the trusted IWF result, safe metadata,
account context, and chain-of-custody data. Only specially authorized, trained
personnel may view bytes when the approved procedure specifically requires it.

### Q20: How should external email attachments be handled?

Answer: Do not accept image or video attachments for safety reports.

Decision / implication: Publish URL/AT URI instructions and configure the mailbox
to reject or quarantine media without previewing or forwarding it. An email
provider that cannot support safe handling is not launch-ready.

### Q21: How are report reasons presented?

Answer: Use a grouped two-step flow rather than one flat list.

Decision / implication: First show broad understandable groups, then the precise
allegation used for routing. Intellectual-property reports route to the dedicated
email process instead of creating a generic in-app report.

### Q22: What should immediate-danger reporting say?

Answer: Direct the person to local emergency services first while still allowing a
CraftSky report.

Decision / implication: CraftSky does not present its app or mailbox as an
emergency service, although credible threats receive the highest internal priority.

### Q23: How are suspected under-16 accounts handled?

Answer: Apply a reversible eligibility restriction after moderator review of
specific credible evidence.

Decision / implication: Preserve sign-in, appeal, privacy, content removal,
account deletion, blocks, and mutes. Do not infer age from appearance, photos,
writing, interests, location, or report volume. Do not collect an age band or use
facial-age estimation at launch; apply safer baseline defaults to everyone.

### Q24: Where is restricted evidence stored and who may access it?

Answer: Use a dedicated private S3 bucket in AWS Frankfurt, separate from scheduled
media, with dedicated IAM/KMS controls and lifecycle deletion. Keep metadata and
access audit in Postgres.

Decision / implication: Ordinary moderators see safe metadata only. A safety
administrator may access bytes, report to the NCA, create holds, or approve
disclosures only when the approved procedure requires it. Named helpers receive
only explicitly assigned permissions, and every restricted access records a reason.

## 4. Candidate Approaches

### Option A: Index-Time IWF Scan With Fail-Closed Visibility

Summary: Persist image-bearing posts/events in a non-visible state, scan immutable
image blob CIDs through an IWF adapter, and expose those records only after all
images clear. Profile records use the last cleared image or a placeholder.

Pros:

- Covers CraftSky and third-party-client writes through one path.
- Avoids separate upload and index scans.
- Uses immutable blob CIDs for idempotency and result reuse.
- Keeps the AppView presentation boundary under CraftSky control.

Cons:

- CraftSky-mediated bytes may reach the user's PDS before scanning.
- Tap/scanner outages delay visibility.
- Requires pending-state support across indexing and every read surface.

Risks: Specialist review may require a pre-PDS check for CraftSky-mediated uploads.

### Option B: Upload-Time And Index-Time Scanning

Summary: Scan app uploads before PDS transfer and separately scan records received
through Tap.

Pros:

- Can stop known material before CraftSky facilitates PDS storage.
- Still covers third-party records at indexing.

Cons:

- Introduces two enforcement paths and potential duplicate scanning.
- Adds latency and failure handling to image composition/upload.
- Does not eliminate the need for index-time scanning.

Risks: Divergent decisions or cache keys could create bypasses.

### Option C: Report-Only Safety Handling

Summary: Expand reporting and incident response without proactive image matching.

Pros:

- Lowest initial integration complexity.
- Avoids scanner-provider dependency.

Cons:

- Known CSAM may be displayed before a user reports it.
- Does not satisfy the confirmed launch direction.

Risks: Unacceptable child-safety and regulatory exposure.

## 5. Recommended Direction

Recommended approach: Option A, with IWF Image Intercept behind a narrow scanner
interface and a deterministic local/test stub. Add a separate restricted incident
and evidence path for detections and legally sensitive complaints. Expand the
existing report taxonomy and operationalize external email intake. Keep ordinary
moderation cases report-originated while allowing detected serious content to enter
the restricted incident path.

Why: This direction scans every record CraftSky may surface regardless of client,
avoids duplicate provider calls, preserves the existing AppView/PDS boundary, and
does not misrepresent an automated match as a user report or moderation decision.

## 6. Problem / Opportunity

CraftSky has a strong general moderation foundation but cannot safely launch while
images can be surfaced without known-CSAM matching, serious harms lack dedicated
report and decision paths, external reporting is unproven, and legally sensitive
evidence has no restricted lifecycle. The opportunity is to add one coherent
launch-safety layer that uses existing moderation and visibility controls while
meeting the documented child-safety, privacy, complaint, and operational promises.

## 7. Goals

- G-001: Prevent unscanned in-scope images from appearing on CraftSky.
- G-002: Detect and safely escalate IWF-known illegal images without automatic
  guilt or sanctions.
- G-003: Let signed-in users and external reporters report every material harm in
  the approved taxonomy.
- G-004: Provide auditable, restricted incident, evidence, hold, reporting, and
  retention workflows for serious harms.
- G-005: Make launch behavior, 16+ eligibility, privacy processing, and public
  policies accurate and consistent.
- G-006: Prove readiness through automated tests, tabletop exercises, operational
  evidence, and closure of every P0 compliance gap.

## 8. Non-Goals

- NG-001: Ozone evaluation or integration.
- NG-002: Video upload, indexing, playback, or video safety scanning at launch.
- NG-003: Novel-CSAM machine-learning classification.
- NG-004: DMs, group chat, livestreaming, ephemeral media, or precise location.
- NG-005: Highly effective age assurance or collection of full dates of birth.
- NG-006: Network-wide removal or moderation deletion of source PDS records.
- NG-007: Automatic account sanctions based only on a scan, report count, or score.
- NG-008: A public web reporting form, unless email fails readiness testing.
- NG-009: Public moderation, incident, report, hold, or age-signal lexicon records.
- NG-010: Pro/Business subscription implementation, recommender feeds, or broad
  commercial-feature expansion.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in member | CraftSky member viewing or creating public content | Scanned content, clear reporting choices, blocking, outcomes, and appeals |
| External reporter | Non-user, affected person, representative, or qualifying organization | Safe email intake, reference, acknowledgement, and outcome |
| Content owner | Owner of reported, detected, or actioned content | Accurate notice, standing, history, appeal, and privacy protections |
| Child user | A 16- or 17-year-old member, or a younger person who reaches the service | Understandable notices and accessible safety controls |
| Moderator | Founder or named authorized helper | Prioritized queues, safe evidence access, deadlines, decisions, and audit trail |
| Safety administrator | Person authorized for CSEA/NCA reporting and restricted records | Secure reports, references, retention, follow-up, and disclosure controls |
| Privacy/operator owner | Accountable CraftSky operator | DPIA, lawful basis, retention, incident response, approvals, and evidence |
| IWF Image Intercept | Intended known-illegal-content matching provider | Minimal, authorized image-check requests under approved terms |
| NCA or verified authority | Authorized recipient/requester under applicable process | Timely, accurate, minimized reports or disclosures |

## 10. Current Behavior

Tap indexers make technically valid records available without a CSAM scan. The
report flow supports authenticated reports for posts, accounts, and events using a
ten-reason taxonomy. Reports group into cases that support decisions, effects,
strikes, suspension, owner history, and email-initiated appeals. AppView applies
moderation and relationship policy across major surfaces without deleting PDS
records.

There is no restricted detected-content incident type, IWF integration, pending
scan visibility state, external email allegation record, legal hold, CSEA reporting
workflow, approved intimate-image workflow, or unified retention enforcement.
Public age and privacy/safety statements are not yet aligned with the confirmed
launch position.

## 11. Desired Behavior

Every in-scope post or business record received through Tap remains unavailable
until all of its image blobs clear the current IWF scan policy. Profile records
remain usable with the last cleared avatar/banner or a neutral placeholder. A
match, scanner failure, or unavailable result remains suppressed and enters the
appropriate retry or restricted incident workflow. Detection never directly
sanctions an account.

Signed-in users receive an expanded, understandable report taxonomy. External
email allegations and appeals are timestamped, referenced, attached to the right
case or incident, and handled without inventing an authenticated reporter. Serious
incidents have restricted evidence, access audit, legal holds, retention, reporting,
and disclosure controls. The product consistently enforces 16+ eligibility and
child-readable safety access. Production video is disabled. Public launch occurs
only after assessments, operations, privacy controls, policies, and all P0 gaps are
approved and evidenced.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | CraftSky shall not launch worldwide, including in the UK, until every P0 item in the compliance gap register is closed with linked evidence and accountable-owner approval. | The current documents classify each P0 as a safe/legal UK launch blocker. | Prompt; `compliance-gap-register.md`; user launch answer | AC-001 |
| BR-002 | Business | Must | CraftSky shall prevent any in-scope image from being displayed on a CraftSky surface before its current scan requirement is satisfied. | Known illegal imagery must not be surfaced while scanning is pending or unavailable. | Prompt; confirmed index-time direction | AC-002, AC-003 |
| BR-003 | Business | Must | CraftSky shall provide usable reporting, complaint, and appeal routes to signed-in users and eligible external reporters. | Current signed-in-only reporting does not cover non-users and representatives. | REPORT-01/02; user email decision | AC-013, AC-014, AC-015 |
| BR-004 | Business | Must | CraftSky shall operate an approved CSEA/NCA response capability before handling production detections or reports. | Scanning without safe reporting, retention, and follow-up creates unmanaged legal and safety risk. | CSEA-01/02; discovery | AC-010, AC-011, AC-012 |
| BR-005 | Business | Must | CraftSky shall make the contractual 16+ launch position and child-safety controls consistent across product, enforcement, support, store listings, and public policy. | Current public pages still state 13+, while children are likely to access the service. | CHILD-01/02/03/04; user answer | AC-019, AC-020 |
| BR-006 | Business | Must | Public policies shall describe only implemented, tested, and approved launch behavior. | Current drafts contain present-tense claims for controls that are not operational. | Document-control rules; POLICY-01/02/03 | AC-026 |
| FR-001 | Functional | Must | The Tap indexing pipeline shall identify all user-controlled image blob CIDs CraftSky renders, including post/business images, profile avatars/banners, and link-preview thumbnails, regardless of namespace or creating client. | One ingestion path must cover every rendered user-controlled image. | Confirmed direction; interview Q11 | AC-002, AC-004, AC-038 |
| FR-002 | Functional | Must | The scanner shall cache results by immutable blob CID and scanner policy/corpus version and reuse only current valid results. | Prevent duplicate checks without allowing stale results to bypass a changed policy. | Confirmed direction | AC-004, AC-005 |
| FR-003 | Functional | Must | The scanner workflow shall represent `pending`, `clear`, `match`, `unavailable`, and `error` and define visibility and escalation behavior for each state. | The model should cover known required outcomes without inventing an IWF uncertainty contract. | Interview Q14 | AC-003, AC-006 |
| FR-004 | Functional | Must | A post or business event containing any image, including a link-preview thumbnail, shall become visible only when every required image blob is `clear`. | Multi-image and preview records must not partially bypass scanning. | Interview Q12 | AC-002, AC-007 |
| FR-005 | Functional | Must | A `match` shall keep the subject suppressed and create or link a restricted incident without automatically creating a violation, strike, or suspension. | A match is a safety signal requiring human adjudication. | Online-safety rules; interview Q13 | AC-006, AC-008 |
| FR-006 | Functional | Must | `unavailable` and `error` results shall remain fail-closed indefinitely, use bounded exponential retries, enter an operator-visible dead-letter state after exhaustion, and permit manual retry. | Provider failure must not bypass scanning or delete user-owned PDS content. | Interview Q15 | AC-003, AC-009 |
| FR-007 | Functional | Must | The image scanner shall be exposed through a narrow adapter with an IWF Image Intercept production implementation boundary and a deterministic local/test stub. | IWF access is unavailable today and its private protocol must not be invented. | User-selected provider and stub decision | AC-027, AC-028 |
| FR-008 | Functional | Must | Production configuration shall reject the scanner stub and shall fail readiness when valid IWF configuration is absent. | A permissive stub cannot satisfy launch scanning. | User-selected stub direction | AC-028 |
| FR-009 | Functional | Must | The system shall create restricted incidents independently of user reports for scanner detections and other approved serious-content sources. After human confirmation, an authorized safety moderator may create a `systemDetected` moderation case linked by safe reference to the incident. | Detected content needs existing enforcement without a fabricated allegation or duplicated decision engine. | MOD-03; interview Q18 | AC-008, AC-043 |
| FR-010 | Functional | Must | Restricted incidents shall record subject identifiers, record/blob versions, source, severity, state, owner, applicable deadline, escalation, decision, disclosure, and resolution history. | Serious-content handling requires stable evidence and accountability. | CSEA-01/02; illegal-content runbook | AC-010, AC-021 |
| FR-011 | Functional | Must | Restricted evidence shall support audited access, minimized export, scoped legal holds, retention expiry, and automatic deletion; a match shall preserve safe metadata immediately but preserve image bytes only when the approved procedure requires it. | Evidence must remain separate and limited without creating unnecessary copies of suspected CSAM. | CSEA-02; RET-01; interview Q17 | AC-011, AC-012, AC-022, AC-042 |
| FR-012 | Functional | Must | Account deletion shall delete unrelated private data while preserving only evidence covered by a valid approved hold, and shall delete held evidence when the hold expires. | Current cleanup conflicts with required preservation. | CSEA-02; account-deletion architecture | AC-012 |
| FR-013 | Functional | Must | The CSEA workflow shall support required priority classification, initial reporting, later supplementation, duplicate-report handling, authority reference storage, retention, and information-request tracking. | The NCA process must be operational, not merely documented. | CSEA-01; illegal-content runbook | AC-010, AC-011 |
| FR-014 | Functional | Must | The signed-in report flow shall first show grouped user-safe categories, then collect the precise allegation needed for routing across child safety; sexual/intimate content; threats/violence/self-harm; harassment/hate/stalking/privacy; fraud/scams/impersonation; spam/misleading commercial/off-topic content; and other concerns. | Current categories omit priority harms, while one long flat list would be difficult to use. | MOD-01; interview Q21 | AC-013, AC-046 |
| FR-015 | Functional | Must | User allegations, internal policy decisions, legal judgments, restricted incident classes, sanction eligibility, and notification-safe text shall be represented as separately mapped concepts. | One overloaded enum would leak detail and constrain lawful sanctions. | Confirmed taxonomy direction | AC-016 |
| FR-016 | Functional | Must | Serious decision reasons shall permit proportionate warnings, visibility actions, strikes, or severe suspension when selected by an authorized human decision-maker. | Current `other` and severe allowlists leave serious violations unsanctionable. | MOD-02 | AC-016, AC-017 |
| FR-017 | Functional | Must | The report UI shall show safe CSEA instructions telling the reporter to identify the location and not download, attach, forward, or redistribute suspected material. | Reporting must not cause further distribution. | Community/reporting policy drafts | AC-013 |
| FR-018 | Functional | Must | External email intake shall timestamp receipt, assign a reference, classify urgency, record ownership, and create a distinct `externalEmail` allegation, complaint, incident, or appeal linked to the canonical subject. | Email is the approved external route but must preserve external provenance. | Interview Q15; REPORT-01/02 | AC-014, AC-015 |
| FR-019 | Functional | Must | External intake shall not populate `reporter_did` with a moderator, fake DID, or reported account and shall keep contact details restricted and minimized. | External reporters are not authenticated members and their provenance must remain accurate. | Codebase finding; user decision | AC-014 |
| FR-020 | Functional | Must | The appeal process shall bind correspondence to an owner-visible decision, verify the appellant proportionately, preserve enforcement while pending, record reconsideration and outcome, and support audited reversals. | Existing email launch is partial and every consequential decision needs review. | MOD-04; existing moderation requirements | AC-015, AC-018 |
| FR-021 | Functional | Must | Qualifying intimate-image complaints shall record receipt, standing, declarations, applicable approved response target, judgment, same/substantially-same search, action, exception, and outcome. | A generic adult-content report is insufficient. | II-01; user deadline direction | AC-021 |
| FR-022 | Functional | Must | Credible-threat and authority workflows shall support emergency-services-first user guidance, highest-priority internal escalation, preservation, independent authority verification, lawful-process review, minimized disclosure, and post-incident review without presenting CraftSky as an emergency service. | No approved emergency or authority process exists. | LE-01; interview Q22 | AC-023, AC-048 |
| FR-023 | Functional | Must | Registration/onboarding shall require an explicit 16+ declaration and record the accepted policy version. | The worldwide 16+ rule must be expressed in product behavior. | User answer; CHILD-03 | AC-019 |
| FR-024 | Functional | Must | The system shall apply a reversible eligibility restriction only after moderator review of specific credible evidence that an account may be under 16, preserve safety/account-maintenance access and appeal, and avoid full DOB collection unless separately approved. | Eligibility enforcement must be recoverable and data-minimizing. | Interviews Q19-Q20 | AC-020, AC-049 |
| FR-025 | Functional | Must | Pending, matched, hidden, and taken-down content shall remain unavailable in feeds, profiles, pins, threads, direct reads, comments/replies, search, notifications, push, events, quotes, reposts, previews, and caches. | Partial suppression can continue harmful exposure. | AppView audit; discovery | AC-007, AC-024 |
| FR-026 | Functional | Must | Tap replay, reconnect, backfill, source update, source delete, and identity updates shall preserve scan and moderation state according to blob CID and subject identity. | Re-ingestion must not resurrect suppressed content. | Tap/federation discovery | AC-005, AC-024 |
| FR-027 | Functional | Must | Production video creation, upload authorization, and publication shall be unavailable for launch. | Video is explicitly deferred and is not covered by image scanning. | User answer | AC-025 |
| FR-028 | Functional | Must | The moderation admin workflow shall provide prioritized views for untriaged reports, urgent incidents, detections, deadlines, appeals, awaiting information, and retention expiry. | Founder-led operations require actionable prioritization and deadline visibility. | Discovery; staffing answer | AC-029 |
| FR-029 | Functional | Must | The service shall implement one enforceable retention schedule covering databases, restricted evidence, email, object storage, logs, Sentry, PostHog, backups, local drafts, and deletion. | Published periods are not yet enforced consistently. | RET-01 | AC-022 |
| FR-030 | Functional | Must | The public website shall not load PostHog before valid consent unless an approved alternative PECR route is implemented and documented. | Current behavior and draft policy conflict. | PECR-01 | AC-030 |
| FR-031 | Functional | Must | CraftSky shall provide documented and exercised privacy-rights, privacy-complaint, and personal-data-breach procedures with applicable clocks and escalation. | Public commitments currently lack operating procedures. | RIGHTS-01; BREACH-01 | AC-031 |
| FR-032 | Functional | Must | While a replacement profile avatar/banner is non-clear, CraftSky shall continue showing the last cleared image or a neutral placeholder if none exists. | Profiles should remain usable without displaying unscanned imagery. | Interview Q7 | AC-039 |
| FR-033 | Functional | Must | The existing author-only synthetic newly created post may display the author's own local image before refresh, but it shall not establish AppView visibility or scanner clearance. | The author already possesses the selected image and the current read-after-write UX should remain unchanged. | Interview Q3 | AC-040 |
| FR-034 | Functional | Must | A successful scan result for an immutable blob shall remain valid without automatic periodic rescanning; authorized targeted rescans may replace it when required. | Launch should not repeatedly retransmit or hide previously cleared content without provider guidance. | Interview Q16 | AC-041 |
| FR-035 | Functional | Must | Default match review shall use the IWF result, safe metadata, account context, and chain-of-custody information without rendering the matched image. | Minimize staff exposure and unnecessary access to suspected CSAM. | Interview Q19 | AC-044 |
| FR-036 | Functional | Must | A confirmed match decision shall notify the owner only after human adjudication using the user-safe reason `Child safety violation`; the notice shall not reveal IWF, match mechanics, reporting status, evidence, or investigative methods. | Avoid tipping off malicious users and avoid premature accusation. | Interviews Q4 and Q13 | AC-045 |
| FR-037 | Functional | Must | The moderation mailbox shall reject or quarantine image/video attachments without rendering previews or forwarding them and shall instruct reporters to provide a CraftSky URL or AT URI instead. | Email must not become an ordinary storage/distribution path for illegal material. | Interview Q14 | AC-047 |
| FR-038 | Functional | Must | The in-app intellectual-property option shall direct claimants to the dedicated copyright/trade-mark email process rather than create a generic moderation report. | Rights complaints require identity, authority, work, location, and declaration information outside the normal report form. | Interview Q17 | AC-046 |
| NFR-001 | Non-functional | Must | Image scanning and moderation visibility changes shall be idempotent and restart-safe. | Tap replay and worker restarts are normal operating conditions. | Architecture constraints | AC-004, AC-005, AC-009 |
| NFR-002 | Non-functional | Must | Restricted evidence and sensitive safety data shall be encrypted, least-privilege, access-audited, and excluded from ordinary logs, analytics, Sentry, push, and user-facing APIs. | Exposure would compound harm and privacy risk. | Runbook; privacy assessment | AC-011, AC-032 |
| NFR-003 | Non-functional | Must | Safety/reporting/appeal flows and child-readable notices shall meet the approved WCAG 2.2 AA release target. | Safety controls must be usable by disabled users and children. | ACCESS-01; child assessment | AC-033 |
| NFR-004 | Non-functional | Must | Scanner and urgent-work queues shall expose health, backlog, age, retries, dead letters, and alert state without revealing restricted content. | Fail-closed systems require timely operational awareness. | Discovery | AC-009, AC-029 |
| NFR-005 | Non-functional | Should | Scanner requests should minimize transferred content and retained provider data consistent with the approved IWF integration. | CSAM matching processes highly sensitive data. | DPIA/provider review | AC-027 |
| NFR-006 | Non-functional | Must | Every moderator, evidence, hold, disclosure, and appeal action shall be attributable to an authorized actor and append-only event. | Founder/helper operations need defensible accountability. | Existing moderation pattern; runbook | AC-018, AC-021, AC-023 |
| NFR-007 | Non-functional | Must | Required restricted evidence bytes shall be stored in a dedicated private S3 bucket in AWS Frankfurt, separate from ordinary media, using dedicated IAM and KMS controls, no public/CDN path, access logging, and lifecycle deletion. | Restricted evidence needs a distinct security and retention boundary. | Interview Q24 | AC-050 |
| RULE-001 | Business rule | Must | Automated scan results and report counts shall not independently establish a violation or apply an account sanction. | Reports and detections are signals, not findings. | Online-safety README; user direction | AC-006, AC-017 |
| RULE-002 | Business rule | Must | Moderation shall not delete source PDS records or claim network-wide removal. | CraftSky controls only its AppView surfaces. | Architecture rules; policy drafts | AC-024, AC-034 |
| RULE-003 | Business rule | Must | Reports, cases, incidents, evidence, legal holds, mutes, and age signals shall remain private AppView data. | Private-by-intent data must not be published to a PDS. | Architecture rules | AC-032, AC-034 |
| RULE-004 | Business rule | Must | The production scanner shall be IWF Image Intercept; the stub shall be restricted to local development and automated tests. | This is the confirmed provider direction while access is pending. | User answer | AC-027, AC-028 |
| RULE-005 | Business rule | Must | Image scanning shall occur at Tap index time for this release; a pre-upload scan may be added only if specialist/provider review rejects the approved boundary. | Maintain one scan path while preserving a legal-review gate. | User answer | AC-002, AC-035 |
| RULE-006 | Business rule | Must | External email may be the launch route only if monitoring, references, acknowledgement, escalation, records, and outcomes pass readiness testing. | A mailbox alone is not an operational process. | User answer; REPORT-01/02 | AC-014, AC-015 |
| RULE-007 | Business rule | Must | Founder-led operation shall identify and authorize help/cover where the approved workflow requires it. | Severe-content and complaint processes cannot rely on undocumented availability. | User staffing answer | AC-029, AC-036 |
| RULE-008 | Business rule | Must | CraftSky shall not introduce Ozone in this requirements slice. | The user explicitly removed Ozone from scope. | User feedback | AC-037 |
| RULE-009 | Business rule | Must | Only an authorized safety administrator may access restricted evidence bytes, report to the NCA, create legal holds, or approve disclosures; ordinary moderators see safe metadata only. | Restrict traumatic and legally sensitive operations to trained personnel. | Interview Q24 | AC-050, AC-051 |
| RULE-010 | Business rule | Must | CraftSky shall collect only a 16+ declaration at launch, shall not collect age bands or use facial-age estimation, and shall apply safer baseline defaults to every user. | Avoid unnecessary child profiling while retaining baseline protections. | Interviews Q20-Q21 | AC-049 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001 | Given a proposed public release, when the launch review is performed, then every P0 register item has approved linked evidence and no P0 remains open. |
| AC-002 | BR-002, FR-001, FR-004, RULE-005 | Given a Tap post or business record containing one or more in-scope image blobs, when it is indexed, then the parent record remains absent from public responses until every required blob has a current clear result. |
| AC-003 | BR-002, FR-003, FR-006 | Given a scan is pending, unavailable, or fails, when the affected content is requested, then the gated parent or image is not surfaced and no failure path treats it as clear. |
| AC-004 | FR-001, FR-002, NFR-001 | Given duplicate Tap delivery of the same record/blob under the same scanner policy version, when workers process it, then the operation is idempotent and reuses the valid blob result. |
| AC-005 | FR-002, FR-026, NFR-001 | Given a replay or source edit, when the blob CID is unchanged the valid result is preserved, and when the CID changes the new blob returns to pending. |
| AC-006 | FR-003, FR-005, RULE-001 | Given an IWF match, when processing completes, then the content stays suppressed and an incident is created without a strike, suspension, or violation decision. |
| AC-007 | FR-004, FR-025 | Given a multi-image record with one clear blob and one non-clear blob, when any CraftSky surface is queried, then neither the record nor an unsafe derivative exposes the non-clear image. |
| AC-008 | FR-005, FR-009 | Given a detection without a user report, when the system escalates it, then a restricted incident is created with detection provenance and no fabricated report/reporter. |
| AC-009 | FR-006, NFR-001, NFR-004 | Given scanner timeout, worker restart, or exhausted retries, when processing continues, then visibility remains closed, retries are bounded/idempotent, dead-letter state is visible, and an alert is raised. |
| AC-010 | BR-004, FR-010, FR-013 | Given a synthetic CSEA detection, when the approved tabletop is run, then priority, initial report, supplement, reference, assignment, and resolution are recorded end to end. |
| AC-011 | BR-004, FR-011, FR-013, NFR-002 | Given CSEA evidence and an authority follow-up, when authorized staff access or export it, then access is logged, the export is minimized, required retention is applied, and unrelated data is excluded. |
| AC-012 | BR-004, FR-011, FR-012 | Given account deletion with no hold, an active valid hold, and an expired hold, when cleanup runs, then unrelated data is deleted, only active held evidence survives, and expired evidence is removed. |
| AC-013 | BR-003, FR-014, FR-017 | Given a signed-in user reporting content/account/event, when the report flow opens, then all approved categories are understandable and the CSEA category displays safe non-redistribution instructions. |
| AC-014 | BR-003, FR-018, FR-019, RULE-006 | Given email from a non-user or representative, when a moderator accepts it, then receipt/reference/provenance/contact minimization are recorded and no fake `reporter_did` is created. |
| AC-015 | BR-003, FR-018, FR-020, RULE-006 | Given an external report or appeal email, when the process runs, then acknowledgement, subject binding, escalation, correspondence, reconsideration, and outcome are auditable against the approved targets. |
| AC-016 | FR-015, FR-016 | Given any approved allegation category, when a moderator adjudicates it, then the selected internal reason, legal classification, sanction options, and user-safe wording are separately represented. |
| AC-017 | FR-016, RULE-001 | Given multiple reports or a scanner match, when no authorized violation decision exists, then no sanction is applied; given a supported human violation decision, proportionate approved effects are available. |
| AC-018 | FR-020, NFR-006 | Given an owner-visible decision and subsequent appeal, when it is confirmed and resolved, then enforcement remains until changed and all reconsideration, outcome, reversal, and actor events are append-only. |
| AC-019 | BR-005, FR-023 | Given a new registration, when onboarding completes, then the person explicitly confirms the applicable 16+ threshold and the accepted policy version is recorded. |
| AC-020 | BR-005, FR-024 | Given a reasonably suspected under-16 account, when eligibility review occurs, then access/action and appeal follow the approved process without requiring full DOB unless separately authorized. |
| AC-021 | FR-010, FR-021, NFR-006 | Given a qualifying intimate-image complaint, when it is handled, then receipt, approved target, standing, judgment, similarity search, action, exception, outcome, and responsible actors are recorded. |
| AC-022 | FR-011, FR-029 | Given data in every listed storage class, when its retention or valid hold expires, then automated deletion follows the approved schedule and produces non-sensitive evidence of completion. |
| AC-023 | FR-022, NFR-006 | Given a credible threat or authority request, when the tabletop is run, then escalation/preservation/verification/legal approval/minimized disclosure/post-review are attributable and credentials are absent. |
| AC-024 | FR-025, FR-026, RULE-002 | Given suppressed content, when feeds, direct reads, search, notifications, derivatives, caches, or Tap replay are exercised, then CraftSky does not surface it and does not delete the PDS source as moderation. |
| AC-025 | FR-027 | Given production launch configuration, when a user or client requests video upload authorization or publication, then the feature is unavailable and no video record can be published through CraftSky. |
| AC-026 | BR-006 | Given the five public policy routes, when publication approval occurs, then every present-tense control is implemented/tested and all policies publish atomically with approved version/effective metadata. |
| AC-027 | FR-007, NFR-005, RULE-004 | Given IWF documentation/access is not available, when implementation is built, then only a protocol-neutral interface and fixture-driven stub exist; no speculative IWF payload contract is embedded. |
| AC-028 | FR-007, FR-008, RULE-004 | Given a production build/configuration using the stub or lacking valid IWF configuration, when readiness is checked, then readiness fails and no image result becomes clear. |
| AC-029 | FR-028, NFR-004, RULE-007 | Given urgent, pending, overdue, appealed, dead-lettered, or expiring work, when an authorized operator views admin status, then prioritized state, age, owner/cover, and alert status are visible without restricted content leakage. |
| AC-030 | FR-030 | Given a visitor has not provided valid consent and no approved alternative route applies, when a public page loads, then PostHog does not load or write identifiers. |
| AC-031 | FR-031 | Given synthetic rights, privacy-complaint, and breach scenarios, when the approved exercises run, then identity checks, searches, decisions, applicable clocks, escalation, and notifications are recorded. |
| AC-032 | NFR-002, RULE-003 | Given ordinary logs, analytics, Sentry, API errors, push payloads, and owner APIs, when safety workflows run, then restricted content, hashes, evidence, contact data, credentials, and internal judgments are absent. |
| AC-033 | NFR-003 | Given report, appeal, safety notice, and eligibility flows, when tested with keyboard, screen reader, text scaling, and error recovery, then they meet the approved WCAG 2.2 AA gate. |
| AC-034 | RULE-002, RULE-003 | Given a moderation action, when public/private storage is inspected, then the PDS source remains untouched and no private report/incident/evidence/hold/age signal is written to a PDS. |
| AC-035 | RULE-005 | Given specialist/provider review rejects index-only containment, when the blocker is recorded, then launch remains blocked until an approved pre-upload control is specified and tested. |
| AC-036 | RULE-007 | Given the founder is unavailable during a synthetic urgent workflow, when escalation is triggered, then an authorized named helper can receive and perform only the approved responsibilities. |
| AC-037 | RULE-008 | Given the delivered design and change set for this slice, when dependencies, services, schemas, and routes are reviewed, then no Ozone integration or Ozone-dependent behavior has been introduced. |
| AC-038 | FR-001 | Given post images, business images, avatars, banners, and link-preview thumbnails created by CraftSky or another compatible client, when Tap indexes them, then every rendered user-controlled blob enters the scan workflow. |
| AC-039 | FR-032 | Given a profile with a previously cleared image, when a replacement is pending, matched, unavailable, or errored, then the previous cleared image remains; without one, a neutral placeholder is shown. |
| AC-040 | FR-033 | Given the current client successfully creates an image post, when the author has not refreshed, then the local synthetic post may appear only in that author's cache and does not make the record available from AppView. |
| AC-041 | FR-034 | Given an immutable blob with a successful clear result, when the same blob is encountered later, then the result remains valid without scheduled rescanning unless an authorized targeted rescan is requested. |
| AC-042 | FR-011 | Given an IWF match, when the incident is created, then safe identifiers/provider/integrity metadata are preserved and image bytes are copied only when the approved procedure explicitly requires it. |
| AC-043 | FR-009 | Given a safety moderator confirms a detected violation, when enforcement begins, then a `systemDetected` case linked by safe incident reference drives the existing decision/effect/history/appeal pipeline without a fabricated report. |
| AC-044 | FR-035 | Given a matched-image incident, when an ordinary review occurs, then the moderator can decide from safe match/context metadata without rendering bytes; any byte access requires the separately authorized procedure. |
| AC-045 | FR-036 | Given human adjudication confirms a matched child-safety violation, when the owner is notified, then the notice uses `Child safety violation`, identifies the action and appeal route, and omits IWF/match/evidence/reporting details. |
| AC-046 | FR-014, FR-038 | Given the in-app report flow, when it opens, then broad groups precede precise reasons and selecting intellectual property directs the claimant to the dedicated email process rather than submitting a generic report. |
| AC-047 | FR-037 | Given a moderation email containing image/video attachments, when the mailbox receives it, then media is rejected or quarantined without preview/forwarding and operators receive only the approved safe handling path. |
| AC-048 | FR-022 | Given a user selects immediate danger, when guidance appears, then it directs them to local emergency services first, permits a CraftSky report, and states CraftSky is not an emergency service. |
| AC-049 | FR-024, RULE-010 | Given suspected under-16 use, when reviewed, then action requires specific credible evidence rather than inference/report volume, applies a reversible restriction, preserves approved safety/account actions, and does not collect age band/full DOB or use facial-age estimation. |
| AC-050 | NFR-007, RULE-009 | Given restricted evidence bytes are legally required, when stored or accessed, then they use the dedicated Frankfurt bucket, IAM/KMS and lifecycle controls, no public path, an authorized safety administrator, and a reasoned access event. |
| AC-051 | RULE-009 | Given an ordinary moderator or helper without safety-administrator permission, when restricted bytes, holds, NCA reporting, or disclosure approval are requested, then access is denied while safe metadata remains available according to role. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | One blob is reused by multiple records/accounts | Reuse the current blob result, but retain subject-specific visibility and incident links where required. | FR-002, FR-005 |
| EC-002 | Record is deleted while scan is pending | Remove it from public eligibility, cancel avoidable work, and retain only approved incident/evidence metadata. | FR-001, FR-011 |
| EC-003 | Source record changes CID but reuses the same blob | Re-evaluate record identity while reusing the current valid blob result. | FR-002, FR-026 |
| EC-004 | Scanner corpus/policy version changes | Previously clear blobs become pending only when the approved policy requires re-scan; no stale result silently qualifies. | FR-002 |
| EC-005 | IWF is unavailable for an extended period | New image records remain non-visible, retries are bounded, backlog/age alerts fire, and launch/readiness remains failed. | FR-006, FR-008 |
| EC-006 | Stub is accidentally configured in production | Startup/readiness fails; the stub cannot issue production clear results. | FR-008, RULE-004 |
| EC-007 | External reporter has no DID or withholds a name | Preserve external provenance and pseudonymity where possible; do not invent member identity. | FR-018, FR-019 |
| EC-008 | Repeated emails concern one report or appeal | Store them as correspondence for the existing allegation/appeal rather than independent guilt signals. | FR-018, FR-020 |
| EC-009 | Reporter emails suspected CSAM as an attachment | Do not place it in ordinary email/case storage; follow the approved restricted handling/escalation process. | FR-011, FR-017, FR-018 |
| EC-010 | Account deletion overlaps an active incident or appeal | Delete unrelated data and preserve only data covered by an approved active hold. | FR-012 |
| EC-011 | A held source disappears from the PDS | Public presentation follows source deletion; restricted evidence follows the approved hold and expiry. | FR-011, FR-026 |
| EC-012 | Scan match is a false positive | Content remains contained during review; no sanction occurs without a human violation decision; correction/reversal is audited. | FR-005, RULE-001 |
| EC-013 | Founder is conflicted or unavailable | Use named authorized help where practicable and record recusal/reconsideration constraints. | FR-028, RULE-007 |
| EC-014 | User is suspended | Reporting, appeals, blocks/mutes, privacy actions, content removal, sign-out, and account deletion remain available under existing suspension rules. | BR-003, FR-020 |
| EC-015 | Link preview contains an image | The entire containing post remains unavailable until the thumbnail satisfies the same index-time scan gate. | FR-001, FR-004, FR-025 |
| EC-016 | A client attempts video publication | Production rejects or withholds the feature even if dormant video code remains in the repository. | FR-027 |
| EC-017 | Author sees a synthetic post before Tap scanning | The author's local cache may show it, but no other client or AppView response treats it as visible. | FR-033 |
| EC-018 | New avatar fails or matches | The previous cleared avatar remains, or a neutral placeholder is used; the unsafe replacement never renders. | FR-032 |
| EC-019 | Known hash database later changes | Existing clear results remain valid unless an authorized targeted or future approved rescan is initiated. | FR-034 |
| EC-020 | External email includes media | Mailbox controls prevent ordinary preview/forwarding/storage and invoke the restricted handling procedure. | FR-037 |

## 15. Data / Persistence Impact

- New fields/entities:
  - Blob scan result keyed by blob CID and scanner policy/corpus version.
  - Record-to-blob scan dependency and pending visibility state.
  - Scanner attempts, retry/dead-letter state, and non-sensitive provider reference.
  - Restricted safety incidents, incident-subject links, actions, access audit,
    disclosures, deadlines, assignments, and outcomes.
  - Restricted evidence metadata, legal holds, hold scope/expiry, and retention jobs.
  - External email allegations/correspondence with source type, receipt/reference,
    minimized sender reference, and canonical case/incident link.
  - Expanded allegation, decision, legal-judgment, incident, and sanction mappings.
  - Registration policy acceptance/version and suspected-underage review state.
  - `systemDetected` case origin and restricted incident reference.
- Changed fields:
  - Existing report and moderation reason constraints require additive categories or
    mapped replacement structures.
  - Existing account deletion cleanup requires active-hold precedence.
- Migration required: Yes. Private AppView migrations are required for scan,
  incident/evidence/hold, external intake, taxonomy, and age-policy state.
- Backwards compatibility:
  - There are no production users or image records requiring a production backfill.
  - Existing development records may be reset or explicitly held pending in local
    environments.
  - Existing stored report reasons should be mapped if retained; no public wire
    meaning should silently change.
  - No lexicon migration is expected under the current private-data design.

## 16. UI / API / CLI Impact

- UI:
  - Expand Flutter report categories and category-specific safety guidance.
  - Add/update 16+ onboarding declaration and child-readable safety copy.
  - Preserve current account standing/history and email appeal entry points while
    making email references and instructions reliable.
  - Disable or remove production video affordances.
  - Admin UI gains scan/incident/deadline/appeal/retention queue views and restricted
    actions; implementation stays within the existing moderation platform.
- API:
  - Existing signed-in report request/response gains mapped categories.
  - Private/admin APIs gain scanner state, incident, hold, evidence, email-intake,
    deadline, and action commands.
  - All `/v1/` bodies remain camelCase with standard errors and typed AT identifiers.
  - No unauthenticated public API is required while email remains the external route.
- CLI: None required for users. Operational/bootstrap commands may be specified in
  later design only if needed for IWF setup or restricted administration.
- Background jobs:
  - Tap-triggered scanner queue and worker.
  - Retry/dead-letter and scanner-health monitoring.
  - Retention and legal-hold expiry.
  - Deadline/escalation monitoring.
  - Existing account deletion must consult hold state.

## 17. Security / Privacy / Permissions

- Authentication:
  - Signed-in reports and owner appeals retain current member/session controls.
  - Admin actions require trusted server-side operator authentication.
  - External email identity is proportionately verified during manual intake; it is
    not converted into member authentication.
- Authorization:
  - Separate ordinary moderation access from restricted incident/evidence access.
  - Only authorized safety administrators may view/export restricted evidence,
    create holds, submit authority reports, or approve disclosure.
  - Named helpers receive only their approved responsibilities.
- Sensitive data:
  - Image bytes, match results, perceptual hashes, evidence, sender contact details,
    child-safety judgments, and criminal-offence data require DPIA/lawful-basis,
    encryption, access audit, minimization, and retention controls.
  - IWF credentials remain server-side and must not be logged.
  - PDS OAuth credentials, DPoP keys, and service JWTs are never evidence.
  - Required evidence bytes use a dedicated private Frankfurt S3 bucket with
    dedicated IAM/KMS controls and no public/CDN route; incident metadata and
    reasoned access audit remain in Postgres.
- Abuse cases:
  - Scanner bypass through replay, updates, alternate clients, malformed images,
    decompression bombs, provider outage, or stub configuration.
  - Malicious/false reports, repeated correspondence, impersonated representatives,
    evidence exfiltration, fraudulent authority requests, and block evasion.
  - Report-abuse restrictions must preserve urgent and legally required access.

## 18. Observability

- Events:
  - Scan requested/completed/failed/retried/dead-lettered and record visibility
    transitioned, identified by non-sensitive IDs.
  - Incident created/assigned/escalated/decided/disclosed/closed.
  - Evidence accessed/exported, hold created/expired, retention deletion completed.
  - External email received/acknowledged/linked/outcome sent.
- Logs:
  - Structured operational metadata only. No image bytes, hashes, IWF payloads,
    evidence, contact details, credentials, or sensitive internal judgments.
- Metrics:
  - Pending scan count/age, throughput, result/error counts, retries/dead letters,
    provider latency/availability, incident queue age, acknowledgement/response
    timeliness, appeals/reversals, retention work, and control failures.
  - Metrics must not treat report count as evidence of guilt.
- Alerts:
  - IWF unavailable/invalid configuration, pending age beyond target, dead letters,
    urgent unassigned incident, applicable deadline at risk, overdue authority
    request, failed retention deletion, unauthorized evidence access, and unavailable
    named coverage.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | IWF does not approve access or its integration cannot support index-time use. | Production scanning cannot launch. | Treat real IWF access as a blocker; keep stub non-production; reassess provider only through a new approved decision. |
| RISK-002 | Index-only scanning is judged insufficient because CraftSky facilitates pre-scan PDS upload. | Architecture fails the approved legal/safety boundary. | Obtain specialist/provider approval before implementation exit; add a content-addressed pre-upload control if required. |
| RISK-003 | Fail-closed scanning causes prolonged content invisibility during provider/Tap outages. | Poor availability and user confusion. | Health checks, retries, dead-letter queues, clear user-safe pending state, backlog alerts, and incident response. |
| RISK-004 | False positive match exposes a user to wrongful enforcement. | User harm and legal/reputational risk. | Immediate containment but no automatic sanction; restricted human adjudication and audited reversal. |
| RISK-005 | Restricted evidence leaks through logs, email, analytics, or admin access. | Severe victim, staff, legal, and security harm. | Separate encrypted store, least privilege, access audit, redaction, leak tests, and breach exercise. |
| RISK-006 | Founder-led staffing misses urgent work or creates conflicted appeal review. | Missed obligations or weak procedural fairness. | Named trained help, alerts, recusal/reconsideration records, tabletop exercises, and specialist approval. |
| RISK-007 | Email attachments cause unsafe redistribution/storage. | CraftSky or staff may possess/distribute illegal material in ordinary systems. | Prominent instructions, attachment controls, restricted escalation, mailbox handling procedure, and training. |
| RISK-008 | Legal hold becomes indefinite retention. | Privacy noncompliance and excess sensitive data. | Require basis/scope/approver/expiry and automatic deletion with audit. |
| RISK-009 | Expanded taxonomy leaks legal conclusions to users. | Privacy, safety, or procedural harm. | Separate allegation, decision, legal, incident, sanction, and user-safe mappings. |
| RISK-010 | Policies are published before controls exist. | Misleading public promises and contractual/regulatory exposure. | Atomic publication gate tied to P0 evidence and factual verification. |
| RISK-011 | Worldwide launch creates obligations beyond the UK analysis. | Unassessed jurisdictional risk. | Complete wider legal/process review or constrain availability before launch. |
| RISK-012 | Existing dormant video paths remain callable. | Unscanned launch media bypasses scope. | Deny production authorization/publication and test all UI/API paths. |
| RISK-013 | The moderation email provider previews, forwards, or retains prohibited media attachments unsafely. | Illegal material enters ordinary communication systems or exposes staff. | Require attachment rejection/quarantine without preview and validate provider behavior before launch. |
| RISK-014 | A trusted known-content match still causes unnecessary staff exposure. | Moderator trauma and excess sensitive-data handling. | Default to metadata-only review; restrict bytes to trained safety administrators under a reasoned procedure. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | IWF Image Intercept will consider CraftSky eligible and provide suitable production access. | A new provider decision and requirements amendment are required. |
| ASM-002 | IWF can be invoked for PDS/blob content at index time under acceptable terms. | The integration point or provider must change. |
| ASM-003 | Blob CIDs are immutable and can safely key scan-result reuse with a policy/corpus version. | A stronger content identity and invalidation design is required. |
| ASM-004 | There is no production image corpus requiring a launch backfill. | Existing in-scope content must be scanned before visibility/launch. |
| ASM-005 | Email volume at launch is low enough for reliable manual intake by the founder and named help. | A public form and automated case-mail integration become necessary. |
| ASM-006 | Existing CraftSky moderation cases, decisions, effects, strikes, and appeals remain the general moderation foundation. | Requirements must be remapped if the foundation is replaced. |
| ASM-007 | The service can disable video completely in production without changing launch-critical image behavior. | Video safety becomes a separate P0 scope. |
| ASM-008 | The contractual 16+ declaration is the approved launch age control pending future rules. | Stronger age assurance and a DPIA become launch requirements. |
| ASM-009 | Specialist review will provide the authoritative intimate-image response target and approve the operating process. | Launch remains blocked. |
| ASM-010 | Named help can be authorized and trained for workflows that require coverage beyond the founder. | Launch scope/staffing must be reduced or expanded. |

## 21. Open Questions

- [ ] **Blocking for production-adapter completion and launch:** Will IWF approve
  CraftSky for Image Intercept, and what exact request, result, retention, security,
  rate-limit, and benign-test contracts apply? A bounded provider-neutral coding
  plan is authorized while this remains unresolved.
- [ ] **Blocking for implementation completion and launch evidence:** The
  accountable owner confirmed on 17 September 2026 that specialist/provider review
  approved index-time containment when CraftSky-mediated bytes may reach the user's
  PDS before Tap ingestion. Record and verify the reviewer, approval date, evidence
  reference, and any conditions before closing this item.
- [ ] **Blocking:** What response target and operating coverage are approved for
  qualifying intimate-image complaints? The current compliance register states a
  48-hour workflow and must be reconciled with the final legal advice.
- [ ] **Blocking:** Who are the eligible NCA administrator and named authorized
  helpers, and what training/access/availability do they require?
- [ ] **Blocking:** What are CraftSky's final legal operator/controller details,
  company number, jurisdiction, postal address, and DPO position?
- [ ] **Blocking:** Does worldwide launch require additional jurisdiction-specific
  controls or availability restrictions?
- [ ] **Blocking:** Which email provider/configuration can reject or quarantine
  image/video attachments without rendering previews or forwarding the media?
- [ ] **Non-blocking for requirements; blocking for policy publication:** Will
  subscriptions and unsupported commercial states be implemented or removed from
  launch policy text?

## 22. Review Status

Status: Approved for bounded coding planning

Risk level: High

Review recommended: Required

Reviewer: Accountable owner; online-safety/privacy specialist approvals remain launch blockers

Date: 17 September 2026

Notes: The accountable owner approved this requirements artifact on 17 September
2026 for acceptance-test design and bounded coding planning. The plan may cover only
provider-neutral scanning infrastructure and the deterministic non-production stub;
it must exclude production IWF adapter implementation. The owner reports that the
index-time boundary has specialist/provider approval, but its documentary reference
is still required. This approval does not authorize implementation completion or
production launch while Section 21 blockers remain unresolved.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: `BR-001` through `BR-006`, `FR-001` through
  `FR-038`, `NFR-001` through `NFR-004`, `NFR-006` through `NFR-007`, and
  `RULE-001` through `RULE-010`.
- Suggested test levels:
  - Unit: taxonomy mapping, scan state transitions, cache invalidation, retention
    calculation, redaction, deadline calculation, and configuration guards.
  - Integration: Tap-to-scan-to-visibility, IWF adapter contract stub, incident and
    hold persistence, account deletion, external email intake, moderation effects,
    and observability leak checks.
  - Flutter widget: report categories/instructions, 16+ declaration, child-readable
    copy, appeal email/reference behavior, disabled video UI, and accessibility.
  - End-to-end: every AppView surface, Tap replay/update/delete, provider outage,
    external allegation/appeal, suspension exceptions, and policy route publication.
  - Manual/tabletop: CSEA/NCA, intimate-image complaint, credible threat, fraudulent
    authority request, evidence breach, founder unavailable/named help, rights
    request, and personal-data breach.
- Bounded-plan constraints: exclude the production IWF adapter and do not invent its
  contract. Treat index-time containment as the approved planning boundary, while
  retaining its documentary reference as required launch evidence.
- Remaining launch blockers: IWF production contract, index-only approval evidence,
  intimate-image target, authorized staffing, safe mailbox provider/configuration,
  legal operator details, and worldwide jurisdiction review.
