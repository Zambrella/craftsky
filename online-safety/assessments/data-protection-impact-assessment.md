# CraftSky launch data protection impact assessment

| Field | Value |
|---|---|
| Controller | CRAFTSKY LTD, company number 17482696, trading as CraftSky |
| Accountable owner and privacy lead | Douglas Todd; `support@craftsky.social` |
| Version / prepared | 0.2 / 9 November 2026; provider scope reconciled with incoming main `7bd29193` |
| Status | Code-based working assessment; actions and residual-risk decision pending; not approved |
| Scope | Launch native iOS/Android app, Svelte web client, public website and associated AppView/operational processing |
| Evidence snapshot | Worktree `t3-6e17dab3`, based on commit `7848c49c2967661530a0c6a70364416093e44a68`, including uncommitted launch-readiness changes on 9 November 2026 |
| Audience | Internal; do not publish the operational assessment as the privacy notice |
| Review triggers | Before launch sign-off; material functionality, audience, provider, ingestion, retention, security or legal change; after a significant incident; confirm an ongoing review date at sign-off |

## 1. Decision summary and limits

A DPIA is needed for the intended child-accessible social service. CraftSky
intentionally permits 16–17-year-olds, processes public and private identifiable
information, and handles potentially sensitive reports about members and
non-members. The [ICO Children's Code DPIA guidance](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/childrens-information/childrens-code-guidance-and-resources/age-appropriate-design-a-code-of-practice-for-online-services/2-data-protection-impact-assessments/)
identifies services likely to be accessed by children as requiring a DPIA.
This assessment concerns privacy risks to people; it is distinct from the
Online Safety Act illegal-content and children's content-risk ratings.

Source review establishes meaningful safeguards. It does **not** establish a
completed lawful-basis record, approved retention schedule, supplier/transfer
review, effective live evidence storage, or a final acceptable residual-risk
position. Completing this document does not supply those missing decisions.
The draft therefore makes **no launch approval or final Article 36 conclusion**.

Production infrastructure already exists. The repository says there are no active
users, but that does not prove there is no production personal data: owner/test
accounts, logs, waiting-list entries, reports and non-member indexed profiles may
exist. No live database, mailbox, vendor account, production secret or personal
record was accessed for this assessment. No production configuration, application
behavior or public policy was changed to prepare it.

Evidence identifiers E01–E21 refer to the linked
[code and verification register](dpia-evidence.md). **Source** means implemented
in the reviewed worktree; **configuration** means a checked-in setting;
**reported** means an attributed earlier owner/operational record; **open** means
not established. Tests referenced from earlier records were not rerun as a
security audit for this document.

## 2. Nature, context, people and scale

The purpose is to provide a craft-focused social network: people publish craft
projects and related content, find people and posts, interact, keep private
working material/preferences, use optional messaging and receive notifications.
Safety, security, service reliability, support and statutory responses support
that service. Advertising, sale of personal data and cross-service advertising
profiling are not approved purposes.

The owner expects **thousands of active users in the first year**. This is a
planning estimate, not measured scale or a legal threshold. Tap collection and
membership-signal filters limit ingestion, but the non-member Bluesky profile
index means affected people are not limited to registered/active CraftSky users.
Measure the real indexed population, countries and stored volumes under A01.
Processing is continuing, internet-accessible and potentially international.

People affected include adult members; intended 16–17-year-old members; younger
children who access despite the self-declared 16+ rule; people pictured or
mentioned in content; non-member authors/profiles; reporters, witnesses, victims,
reported people and representatives; message correspondents; website visitors,
waiting-list subscribers and the sole operator. Children in images or reports
may be younger than the account holder. No date of birth or reliable adult
classification is evidenced. Do not infer adulthood from the minimum-age checkbox.

Public posts/profile prose, images, videos, recent searches, drafts, messages and
reports can reveal health, disability, religion, sexuality or other sensitive
matters. Reports can contain criminal allegations about suspected offenders.
Ordinary photos are not automatically biometric identification data; no biometric
identification or age-estimation system is established by this review. Hashes,
opaque IDs and public DIDs can remain personal data. Public availability is not
an anonymity or lawful-basis exemption.

## 3. Processing and data-flow inventory

| Flow / purpose | Data and source | Processing, storage and recipients | Boundary / evidence |
|---|---|---|---|
| F01 Public identity and social content | User/PDS/relay: DID, handle, biography/pronouns, public profile, project/post/reply fields, links, image/video references, likes, reposts, follows, blocks, business profiles/events | AppView mediates record writes to the owner's PDS; Tap ingests filtered public records; PostgreSQL projections supply public/member/guest reads, discovery and search. Independent PDSes/relays/services and public visitors can obtain public records/media. | Public does not mean CraftSky controls independent copies. Non-member profiles can be indexed. Blocks are public records; mutes are private. E01–E02. |
| F02 Authentication and account lifecycle | Account/DID, handle, OAuth/PDS discovery metadata, server-held OAuth access/refresh material and DPoP keys; child session token, installation/routing identifiers, expiry and revocation metadata | AppView/PostgreSQL stores OAuth session JSON and lifecycle state. Native retains CraftSky session registry in secure storage. Web Worker handles credentials and sets HttpOnly cookies containing CraftSky credentials/metadata. PDS authentication is separate. | No generic PDS OAuth credential handoff to clients. No claim of field-encrypted OAuth JSON: the reviewed store serializes it into PostgreSQL. Browser cookie encoding is base64url, not encryption. E03–E04. |
| F03 Private preferences and saved activity | Member settings, language/customization, mutes, saved folders/posts, recent search labels/payloads, notification preferences and follower-count history | Owner-scoped AppView data and local preferences; search history is capped at 50 rows per viewer, with individual deletion. Follower-growth snapshots are per-account history. | Search history is private, potentially sensitive, and not anonymous. A count cap is not a time limit. No universal private-data expiry established. E05. |
| F04 Drafts and scheduling | Unpublished post/project text, alt text, images/video, editing and retry state | Native local files; web IndexedDB private drafts. Scheduled payloads/state in PostgreSQL and private media in S3; publication sends the selected content/blob to PDS. | Ordinary device/browser drafts differ from scheduled server content. Device compromise/backups and other retained devices are separate deletion boundaries. E06. |
| F05 Messaging | Bluesky inbox/conversation/message bodies, participants and request/contact settings; send/report reasons; local unsent drafts | AppView fetches/processes provider bodies in memory to support the clients and sends provider operations/reports. Durable CraftSky tables retain operational metadata, IDs/dedupe/receipts/routing, not message/report bodies. Native DM drafts use secure device storage; web DM drafts use account-partitioned localStorage. | Bluesky owns the provider inbox/report destination; CraftSky still processes bodies in transit. Not end-to-end encryption. No duplicate CraftSky safety report or DM Brevo alert. E07. |
| F06 Notifications | Device/push token, account subscription routing, preferences, public actor/subject references, chat open reference, delivery status | PostgreSQL delivery state → Firebase transport/device delivery ecosystem → OS/browser notification. Chat body is generic but title may contain sender display name; social notifications include public names/routing facts. | Generic body does not mean anonymous or invisible on lock screens. Delivery recipients and stale-token cleanup require verification. E08. |
| F07 Reports, moderation and appeals | Member/guest/representative inputs, optional contact, subject URI/DID, allegation/reason/private detail, evidence/safe snapshot, legal rationale, sanctions, appeals and operator/audit history | Restricted PostgreSQL case systems and operator console; distinct user-safe explanations/standing projections. External intake records minimized metadata; original email/correspondence remains in mailbox systems. | Intake contact uses an unkeyed SHA-256 reference, not anonymization. Allegations are not findings. No default disclosure of report identity/prose to reported users. E09–E10. |
| F08 Restricted incidents and authority response | Sensitive evidence, incident subjects, legal holds, authority/request/report references, deadlines, retention/access events | S3 evidence objects plus PostgreSQL restricted metadata and application legal holds; authorized review and justified authority disclosure. Dedicated evidence store is separate from scheduled media. | Provisioning record says bucket/IAM created, AppView connection and synthetic hold/deletion exercise outstanding. Do not collect new evidence on the assumption this path is operational. E10–E11. |
| F09 Operator alerts and correspondence | Urgent-work signal, opaque delivery UUID; support/reporting/rights emails; waiting-list email/preferences | New safety-alert code sends only fixed text, moderation mailbox address and authenticated console origin via Brevo. Mailbox correspondence and Brevo waiting-list submissions have separate content and retention. | Alerts do not export report prose/images/subject IDs. Production alert delivery unverified. Waiting-list iframe has fixed `src`; loading behavior needs review. E12, E15. |
| F10 Diagnostics, security and status | Selected technical errors/logs/traces, release/device context, request/correlation references, transport IP/user agent; service-status requests | Protected local logs, server/infrastructure logs, optional native/web Sentry, configured server Sentry, Cloudflare/Render transport. Website PostHog is a separate consent-gated source path. | Sanitization does not eliminate provider transport metadata. Client consent does not gate server operational logging. No diagnostic consent is implied by joining. E13–E15. |
| F11 Instagram verification, import and matching | Instagram username/account identifiers, verification challenge/message/webhook fields, imported following handles/digests, account-link and private suggestion/import/rollback state | AppView private verification/import/matching stores; local import tool/browser progress; Meta/provider webhook boundary; eligible private suggestions. Some content import later results in public PDS writes. | Keyed challenge/hash handling reduces exposure but matching/linking is identifiable cross-source processing. No assumption that a friend consented because an uploader supplied their handle. E16. |
| F12 Link previews and external media | Submitted URL, metadata/image; public image/video identifiers; viewer network information | AppView bounded/pinned server fetches; PDS and configured video processing/delivery (Bunny by default after the main merge; legacy Bluesky path retained); client external resources/players. YouTube player asks permission or uses a previously saved choice; thumbnail loading is separate. | SSRF defenses do not establish tracking consent. Remote media can reveal viewer IP/timing. Direct image/video capabilities are short-lived and purpose-bound. E17–E18. |
| F13 Conditional purchases | Random RevenueCat app-user ID linked to DID, entitlement/license, store transaction/subscription state and receipts | AppView/secure registry, RevenueCat and Apple/Google store flows if enabled. Web checked-in purchases flag is false; native defaults off. | No raw payment-card processing evidenced. Do not count conditional code as an enabled launch feature; re-assess actual release flags and provider terms before enablement. E19. |

Incoming main `7bd29193` now selects **Bunny** as the default video provider,
with video enabled in checked-in production configuration. The legacy Bluesky
path remains available. This updates the initial draft's Bluesky-default finding.
Treat Bunny as intended launch processing in the provider/transfer and media-
retention review, including original/rendition/CDN copies and independent asset
URLs. ADR 024's development/manual-acceptance history does not prove actual
production credentials, deployment, contractual safeguards or acceptance; those
still require evidence (E18). No production setting was changed by this merge.

Production image-safety configuration defaults to manual format validation and
moderation, not a validated universal automated content scanner. Any new external
scanner needs its own data/decision review (E18).

Future age assurance is also a **change trigger**, not an
existing ID-document/age-provider flow authorized by this DPIA.

## 4. Necessity, proportionality and lawful-basis working analysis

These are **proposed legal routes**, not completed LIAs or blanket authority.
Select and record the actual basis for each purpose under A02; do not switch
bases merely to override a withdrawal or objection. The draft Privacy Policy's
statements are intended policy, not proof of lawful processing or live behavior.

| Purpose | Candidate Article 6 route and necessity test | Less intrusive option / decision still needed |
|---|---|---|
| Requested account, posts and private features | Article 6(1)(b) only to the extent objectively necessary for the requested contract/service; otherwise assess a separate legitimate interest | Keep public/private boundaries and minimal scopes; optional matching, analytics and diagnostics are not automatically contract necessities. Consider child users' expectations and contractual position. |
| Public/non-member indexing, search and discovery | Candidate Article 6(1)(f), with purpose, necessity and balancing assessment; a non-member has no CraftSky contract | Scoped ingestion instead of whole-network harvesting; limit non-member fields/use, accurate provenance, objection/removal process and Article 14 transparency assessment. Public-source or disproportionate-effort exceptions require recorded analysis, not assumption. |
| Account security, abuse prevention and proportionate operational logs | Candidate Article 6(1)(f); Article 6(1)(c) only for a specified applicable obligation | Minimal identifiers, bounded retention, no private payload logging, least privilege; justify persistent installation identifiers and lifetime dedupe/audit ledgers. |
| Reports, safety decisions, evidence and authority disclosure | Map each operation to a specific obligation under 6(1)(c), or justified 6(1)(f); emergency 6(1)(d) only where its test is met | Intake only necessary detail; restricted evidence separate; safe explanations; targeted legal holds; verify authorities and necessity before disclosure. A general “online safety” label is insufficient. |
| Client diagnostics and website analytics | Existing source choices support candidate 6(1)(a) consent for optional collection; assess device-access/PECR requirements separately | Off by default pending choice, withdrawal, no replay/autocapture; verify actual requests, payloads and provider settings. Source consent alone does not prove every embedded resource complies. |
| Optional Instagram verification/matching | Assess requested verification necessity separately from candidate 6(1)(f) matching and balancing, or a valid consent design where appropriate | Exact private suggestions, unlink/delete/control, minimal raw graph persistence and notices for affected non-users. Linking accounts can expose identities even if neither source was secret. |
| Push, correspondence and waiting list | Requested notifications/service correspondence may support 6(1)(b) or justified 6(1)(f); separate consent/marketing assessment for waiting-list or promotional communications | Explain lock-screen exposure and preferences; minimal payloads; withdraw/unsubscribe without unrelated service loss; no assumption that browser permission is the complete legal basis. |
| Conditional billing | 6(1)(b) for requested purchase/entitlement and 6(1)(c) for specified record duties where applicable | Random provider ID instead of public DID as vendor customer ID, minimal receipt data, justified accounting retention, actual enabled-feature decision. |

Article 9 conditions need a separate operation-specific record where special
category data is processed. Article 9(2)(e) is **not** a general license for all
publicly retrievable data: examine whether the person deliberately made those
particular sensitive data public. It cannot automatically cover someone else's
allegation, a private draft, search or message. Article 9(2)(g) and relevant DPA
2018 Schedule 1 safeguarding/unlawful-act routes are candidates to assess,
including any required appropriate policy document; no condition is adopted here.
See [ICO special-category conditions](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/lawful-basis/special-category-data/what-are-the-conditions-for-processing/).

Information about suspected offenders/allegations can engage Article 10. Ordinary
CraftSky moderation does not make the company an official law-enforcement
competent authority. Record the relevant domestic authorization/condition and
any policy-document requirement, alongside Article 6, before relying on it.
Victim/witness information is not automatically criminal-offence data, though it
may be special-category data. [ICO criminal-offence guidance](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/lawful-basis/criminal-offence-data/what-is-criminal-offence-data/)
and [conditions](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/lawful-basis/criminal-offence-data/what-are-the-rules-on-criminal-offence-data/).

Benefits include user-controlled publishing, craft knowledge/community, portability,
recovery of private work, control over contact, and safety response. Those benefits
justify examining proportionate processing; they do not justify publishing private
information, indefinite records or unconstrained sensitive-data reuse. Complete
the child-specific balancing and best-interests assessment, including wider age
bands and vulnerabilities, without presuming the child's choice or public profile
removes the risk.

## 5. Children, transparency and rights

The self-declared 16+ rule is not age assurance. The current safety-support guide
is public, has child/parent sections and offers report/search support without
collecting age; its local verification is recorded in E20. This is useful privacy-
respecting signposting, but does not establish full Children's Code compliance.

Assess all relevant standards under A03: best interests, age-appropriate
application, clear notices, detrimental use, privacy defaults, minimization,
sharing, geolocation, parental controls where relevant, profiling, nudges,
connected-device scope where relevant and accessible online rights tools.
Record any non-applicability with reasons. Search can prioritize followed
accounts and rank textual relevance; project browsing offers popularity sorting
(E05). Assess these uses of public relationship/content data and their effects
in the profiling/defaults review rather than assuming every discovery surface
is chronological. No targeted advertising, biometric
identification, exact DOB collection or health-profile feature is established.
That does not mean no data could reveal location/health: free text, photos,
events and media metadata may do so. Media metadata stripping and default
visibility need explicit release evidence, not assumptions.

The public PDS model and deletion limits need understandable explanation before
publication, including for teenagers and non-member people affected by indexing.
Public blocks must not be described as private mutes. The guide is not a
replacement for a complete, accessible Article 13/14 notice. Assess non-member
notification and any claimed exception using the [ICO transparency guidance](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/individual-rights/individual-rights/right-to-be-informed/).

Source deletion requires fresh PDS OAuth and owner confirmation, narrows PDS
record deletion to registered CraftSky collections, and uses lifecycle fencing
and private-data cleanup. It does not delete the DID/PDS account, another
namespace, already delivered messages or every independent network/device copy.
Legal holds are narrow exceptions, not a reason to retain an unrelated account.
E21 includes source and test evidence; actual recovery/deletion/hold exercises,
backup reintroduction and time-to-convergence need verification.

Operational access, correction, portability, objection, restriction, erasure and
complaints searches must cover PostgreSQL, objects, email, vendor records and
local state within CraftSky's control. An account-deletion button alone is not
all rights fulfillment. A complete cross-store export/rights procedure has not
been evidenced in this review. Use proportionate identity checks, protect third
parties and avoid collecting extra identity documents by default (A10).

## 6. Recipients, roles and international access

| Recipient / boundary | Evidence of use | Role / location conclusion and open work |
|---|---|---|
| Render | AppView, Tap and PostgreSQL checked in for Frankfurt, private-network isolation and protected environment | Candidate processor for hosting; verify contracting entity, Article 28 terms, logs/backups, access/support/subprocessors and actual deployment. Region is not the complete transfer analysis. |
| AWS S3 | Scheduled media configured for `eu-central-1`; separate restricted-evidence provisioning record | Candidate processor/object storage; verify separate least-privilege principals, encryption, non-public buckets and deletion behavior; complete evidence connection separately. |
| Cloudflare | Public web Worker/assets, web app proxy, DNS/status infrastructure | Candidate processor for relevant hosting/proxy operations; global edge/support processing and logging settings need inventory. Do not assume German-only processing because AppView is Frankfurt. |
| PDS operators, relay, legacy Bluesky video and Bluesky messaging | User-selected PDS; configured Bluesky relay/shared inbox/provider reports and retained legacy video path | Independent or other roles must be assessed per operation/contract, not blanket “our processors”. Public federation differs from private DM provider processing. Map network access and provider retention/control. |
| Firebase/Apple/Google notification ecosystem | Firebase configured for push; native/browser delivery paths | Device tokens, routing, public names and generic alert text may be processed. Verify entities, platform-specific paths, retention and international access. |
| Sentry | Opt-in clients and separately configured AppView operational reporting | Candidate processor for selected diagnostics; actual organization region, IP handling, retention, support access and DPA not verified. German ingestion endpoint is limited evidence. |
| PostHog | Public-site loader only after saved grant, DNT guard, memory persistence, no autocapture/replay | Candidate processor for optional analytics. EU UI/configuration is not proof of all processing location or event retention. Verify payloads and any query/referrer capture, consent and withdrawal. |
| Brevo and mailbox provider(s) | Embedded waiting list; optional new generic operator alerts; external support/moderation correspondence | Separate purposes/content. Verify contracts, actual mailbox operator, access, email storage, signup tracking, unsubscribe and retention. A Brevo alert integration does not identify the mailbox host. |
| Meta/Instagram and import-tool boundary | Verification/webhook/migration/matching code | Assess roles and source legitimacy, permitted use and graph/linkage transparency; actual enabled integration/settings unknown. |
| External sites/media/support services | Previews, images, video and optional YouTube player; fixed public-body support links | Distinguish server fetch from viewer requests/embeds and intentional off-site navigation. No support report/search context should be added to destinations. |
| Bunny | Default video provider and enabled video in incoming main production configuration; source copies/processed PDS handoff and vendor delivery | Intended launch provider after the merge; verify actual credentials/deployment, original/rendition/CDN retention, moderation controls, contracts and transfers. Prior development acceptance is not live proof. |
| RevenueCat/store providers | Conditional purchases | Not established as active launch processing by this source audit. Include contracts/transfers before enabling; development personal data is not exempt. |
| Authorized authorities and operator | Case-specific operational disclosure; sole-owner console access | Record recipient/authority, purpose, minimum fields, authorization and retention. Human authority review is required; generic alerts are not authority disclosure. |

A04 must establish contracting entities, roles, processing/support countries,
subprocessors, Article 28 requirements and any applicable transfer mechanism and
assessment. UK-to-EEA hosting and onward access are different questions. Do not
invent signed DPAs, adequacy coverage, an IDTA/Addendum or completed transfer
assessment from a URL or selected region. [ICO international-transfer guidance](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/international-transfers/a-guide-to-international-transfers/).

## 7. Retention and deletion findings

| Class | What is evidenced | Remaining decision / verification |
|---|---|---|
| OAuth/session state | Checked-in settings: 180-day absolute session lifetime, 30-day inactivity, 30-minute auth request expiry, 24-hour terminal auth-request retention and bounded revocation/cleanup workers. Web cookie default lifetime 180 days. | Confirm effective settings and cleanup, invalidation of revoked credentials and backup treatment. Cookie lifetime is not authority beyond live AppView session validity. E03–E04. |
| Scheduled media/content | Source maxima: unclaimed media 24 hours; needs-attention private content 30 days; publication tombstone 30 days; successful publication cleanup immediately due. | These are state-specific, not “all drafts deleted after 30 days”; verify running workers and failure recovery. Device/browser drafts need their own lifecycle. E06. |
| Chat operational metadata | Source maintenance prunes expired view/continuation state and terminal notification/delivery details; SQL uses a 24-hour delivery expiry and 30-day terminal detail windows. Lifetime operation/disposition/generation ledgers are explicitly not TTL-purged. | Justify minimal lifetime receipts and define/account for deletion scope; provider history and local drafts have different lifecycles. E07. |
| Recent searches / follower history | Search rows capped at 50 with per-item delete; per-account follower snapshot history exists and ordinary retention deliberately does not purge it. | Decide time limits/clear controls and necessity of account-lifetime history. A bounded row count does not limit age or sensitivity. E05. |
| Instagram | Dedicated sweeper and state-specific expiry exist; moderation restoration detail reduces after 30 days and DID-free history expires after 365 days. Persistent graph/import and linkage states differ. | Inventory each current state; do not apply an old blanket “24 hours to one year” claim to all records. Verify unlink/account deletion and running maintenance. E16. |
| Restricted evidence / legal holds | Per-evidence retention expiry and hold-aware deletion worker; owner cleanup excludes active holds and anonymizes links where unheld. | Approve period/grounds per class and verify connected storage, expiry, hold release, retry and admin access. Application holds are not S3 Object Lock. E10–E11, E21. |
| Ordinary moderation, email, logs, analytics, diagnostics, push and backups | Draft Privacy Policy proposes periods, but code/provider coverage is incomplete. | Do not represent proposed two-year/30-day/90-day/12-month periods as enforced. Validate statutory CSEA periods separately against the actual report class/law. Provider logs/backups may differ. A05. |

Account deletion and legal retention are reconciled per data class. A restore
procedure must prevent erased records from silently becoming active again and
must reapply deletion/hold state. No live restore/expiry proof was obtained here.

## 8. Risks to people and existing safeguards

Ratings below are **reasoned draft assessments**, not measured incident rates.
Likelihood: U = unlikely on the available design; P = plausible; L = likely
exposure/opportunity without safeguards. Impact: limited, serious or severe
(harm to safety, identity, finances, autonomy or reputation). High includes
plausible severe harm; Medium includes plausible serious harm. Severity can
justify attention despite low probability. “Pending” means evidence is inadequate
to conclude residual acceptability, not that the safeguards are absent.

| ID / privacy harm | Inherent likelihood / impact / rating | Implemented or recorded mitigation | Current residual conclusion / action |
|---|---|---|---|
| R01 Unwanted permanent public exposure, including a child pictured/identified | P / severe / High | Deliberate public PDS separation, scoped ingestion, owner editing/deletion and guide explaining independent copies | Pending: discoverability, pre-publication understanding, location/media metadata and non-member rights need assessment. No guarantee of federation-wide erasure. A01, A03, A10. |
| R02 Private drafts, searches or saved activity exposed to the wrong user/session | P / severe / High | Owner/account partitions, lifecycle fences, local private drafts; search history private and capped | Pending live isolation/local deletion and shared-device checks; sensitive query history still exists. A05, A06, A11. |
| R03 Credential compromise enables identity takeover or unauthorized PDS writes | P / severe / High | Server-held OAuth/DPoP, scoped ephemeral media capabilities, child-session validation/expiry, secure native registry, HttpOnly/Secure production cookies, SSRF protections | Pending privileged-access/key separation, host/database backup protection, revocation and recovery verification. Do not claim field-encrypted OAuth session JSON. A06, A11. |
| R04 Sensitive allegation/evidence disclosure, retaliation or stigma | P / severe / High | Restricted operator/evidence paths, safe owner-facing details, separate object store design, payload-free alerts and diagnostic filtering | Pending evidence-store connection/permissions and disclosure/retention conditions. Email originals remain a separate exposure. A02, A05–A07. |
| R05 Incorrect allegation or automated/mistaken presentation causes unfair exclusion | P / serious / Medium | Allegation/decision distinction, operator reason/rationale, evidence and revision checks, safe reasons, appeals/reversal/source history | Pending actual appeal capacity, accuracy/identity correction and all-client visibility convergence. Scanner “clear” is not a legal finding. A03, A07, A10–A11. |
| R06 DM confidentiality/contact expectations differ from provider reality | P / severe / High | Separate opt-in/scopes, two-person boundary, provider reports, no durable server bodies, private local drafts, generic push body | Pending provider role/control/retention review, shared-inbox explanation, localStorage/shared-device and request/blocked-user tests. Messages are not end-to-end encrypted by CraftSky. A03–A06, A11. |
| R07 Cross-source Instagram linkage identifies someone unexpectedly | P / serious / Medium | Exact private matching, keyed challenge representation, eligibility rules, private storage and retention jobs | Pending lawful-basis balancing, non-user transparency, unlink/withdrawal and persistent graph necessity. Hashes are not anonymous. A02–A05, A10. |
| R08 Telemetry/embedded resource reveals sensitive browsing or identifiable activity | P / serious / Medium | Client Sentry opt-in, protected sinks, PostHog consent/DNT and replay off, no-referrer web app, optional YouTube playback, support links without context | Pending serialized live payloads, transport metadata and provider retention; static waiting-list iframe may load before dialog interaction. Third-party thumbnails are separate from player permission. A04–A06, A08, A11. |
| R09 Excessive retention or restore reintroduces erased sensitive data | P / severe / High | State-specific cleanup, limited queries, legal-hold-aware deletion and terminal purge/lifecycle design | Pending complete class schedule, provider/backup enforcement and restore exercise; no assumption all private data expires. A05–A07, A10–A11. |
| R10 Unjustified international access or vendor secondary use | P / serious / Medium | Frankfurt infrastructure selection, scoped APIs and minimized provider payloads | Pending contracts, role/subprocessor/transfer mapping and effective restrictions. Regions alone do not mitigate all access. A04. |
| R11 Young users cannot understand or exercise meaningful choices | P / serious / Medium | Minimum-age statement, public child/parent guide, simple report/support and control paths, no new age/health inference | Pending youngest-foreseeable-user notices/defaults/rights and accessibility consultation. No verified adult separation. A03, A09–A10. |
| R12 Notification/lock-screen or email exposure reveals relationships | P / serious / Medium | Generic chat/moderation bodies, limited routing, user preferences, static safety alert with console-only link | Pending public-name visibility, permission/default and device/mailbox access checks; minimize routing retention. A04–A06, A08, A11. |
| R13 Operator error, overbroad access or absence delays containment/rights | P / severe / High | Single named accountable operator, permissions/assignments, audit history, approved disclosure procedure and queued alert design | Pending account hardening/backup response arrangements, competence evidence and personal-data breach/rights plan. Owner monitoring is not continuous staffed coverage. A06–A07, A09–A11. |
| R14 People cannot discover indexing, obtain copies/correction or effective erasure | P / serious / Medium | Owner deletion flow, individual search removal, public policy drafts and contact address | Pending published accurate notices, non-member Article 14 analysis, cross-store rights/export workflow and retention exceptions. A01–A02, A05, A10–A11. |
| R15 Optional purchases or changed video-provider scope expands processing unnoticed | P / serious / Medium | Purchases default off and checked-in web flag false; incoming main explicitly selects Bunny for enabled video, with legacy Bluesky retained | Pending exact release flag/provider inventory; repeat assessment before enablement, not retrospective approval through code presence. A01, A04, A11. |

No final overall “Low/Medium/High CraftSky privacy risk” is assigned while these
material verification and legal-route questions remain unresolved. The risk
ratings are not a finding that an incident has happened or that every public
post is harmful. They direct the evidence/actions needed to reach a defensible
residual-risk decision.

## 9. Action and decision register

Owner for each action is **Douglas Todd**, who may obtain implementation or
specialist assistance. No external contact or new feature is authorized merely
by listing an action. Evidence may justify a smaller scope or a different
proportionate control; record the decision rather than silently inventing one.

| Action | Concrete output / acceptance evidence | Timing / linked risks |
|---|---|---|
| A01 Confirm exact processing scope and scale | Release/features matrix including purchases, Instagram, video provider, member/non-member ingestion, actual counts/volume/countries and existing production data; minimum fields and purposes per flow | Before final DPIA sign-off and scope changes; R01, R14–R15 |
| A02 Complete lawful-basis and sensitive-data records | Operation-specific Article 6 record; LIAs where used; Article 9/10 and Schedule 1 conditions, statutory obligation mapping, any required appropriate policy document; public-source transparency decisions | Before relying on the relevant processing; R01, R04, R07, R14 |
| A03 Complete child-specific privacy/defaults review | All applicable Children's Code standards, age-band/best-interests analysis; verify actual public/profile/contact/location/profiling defaults and notices; assess age-assurance necessity separately without collecting new age data by default | Before launch sign-off; R01, R05–R06, R11 |
| A04 Complete provider/transfer inventory | Actual entities, roles, contracts, subprocessors, countries/support access, retention and transfer route/assessment where needed, with dated evidence | Before relevant provider processing/enablement and final sign-off; R06–R08, R10, R12, R15 |
| A05 Approve and enforce retention schedule | Each class/state/local copy/vendor/log/backup mapped to purpose, trigger, period and deletion job or manual procedure; decide search/follower/lifetime-ledger treatment; verify expiry and restore behavior | Before final sign-off; R02, R04, R06–R09, R14 |
| A06 Verify security and authorization | Host/vendor/operator MFA/access review as appropriate; OAuth DB/backup/key protection and rotation; account isolation, cookies/CSRF, restricted role controls, device/browser storage and credential lifecycle evidence | Before final sign-off; R02–R04, R06, R08–R09, R12–R13 |
| A07 Complete evidence/disclosure/breach readiness | Connect dedicated evidence store after merge under existing authorization; benign hold/release/deletion and least-privilege exercise; authority identity/minimization checks; documented breach assessment/notification and unavailable-owner response | Before relying on evidence/disclosure paths and launch sign-off; R04–R05, R09, R13 |
| A08 Resolve optional-resource and transport exposure | Observe landing-page iframe before/open/close, PostHog before grant/after withdrawal/DNT, player/thumbnail and push payload behavior; choose necessary gates/notices/minimization based on actual purpose | Before affected release surfaces are approved; R08, R12 |
| A09 Consider appropriate consultation and record outcome | Decide proportionate child/parent/user consultation, targeted accessibility/usability review and expert/vendor input; record participants/scope or reasoned deferral/alternative evidence, feedback and changes | Before concluding DPIA where appropriate; R11, R13 |
| A10 Establish rights and transparency procedure | Accurate published privacy information; non-member notification/objection route; cross-store request search/export/correction/erasure with proportionate identity checks, third-party protection and applicable deadlines | Before final sign-off; R01–R02, R05, R07, R09, R11, R14 |
| A11 Verify exact deployed controls | Identify release/tag and enabled features; local tests plus deployed native/web/guest/operator/storage/provider checks, backups/retention and actual diagnostic exports; record dates/results without synthetic harmful media | Before final sign-off; all risks |
| A12 Decide residual risks and consultation | Reassess each risk after evidence; record adopted/alternative controls, owner acceptance and reasons; determine any Article 36 prior-consultation requirement; set review date | After relevant actions, before approving affected processing/launch |

Existing launch records remain authoritative for decisions already made: no
formal DPO initially; no reporter-status/history UI; Bluesky-only DM reports;
public-content boundary and safety-support implementation approved; restricted
storage connection deferred until merge. This DPIA does not reopen those as new
permissions or declare the remaining privacy questions answered by them.

## 10. Consultation, sign-off and follow-through

Douglas Todd is the controller's decision-maker and privacy lead. The approved
DPO assessment did not identify a mandatory initial appointment; no independent
DPO advice is claimed. The owner-led review route does not require an external
certificate. No child/parent/user consultation, external expert review, vendor
confirmation or ICO consultation occurred for this draft. A09 records the need
to consider appropriate consultation and document its outcome, rather than
inventing feedback or treating generic desk research as participation.

Owner approval can accept a documented proportionate residual position; it does
not waive unresolved legal requirements. Where high residual risk cannot be
sufficiently mitigated, determine and complete required prior ICO consultation
before the affected processing. Do not submit every DPIA automatically or infer
that consultation is already required solely because an inherent-risk row is
High. [ICO DPIA process and consultation guidance](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/accountability-and-governance/data-protection-impact-assessments-dpias/how-do-we-do-a-dpia/).

| Sign-off item | Current record |
|---|---|
| Preparation | Code-based assessment v0.1, 9 November 2026 |
| Owner factual review / decisions | Pending |
| Consultation/advice and response | Not completed; A09 pending |
| Action completion / adopted alternatives | A01–A12 pending evidence and decisions; existing safeguards credited above |
| Final residual ratings and reasons | Pending; no overall acceptable-risk finding |
| Article 36 decision / reference if applicable | Pending A12; no submission made |
| Exact approved release and features | Pending A01/A11 |
| Approval / date / next review date | Not recorded; no launch authorization |

Integrate accepted actions with the compliance gap register, amend the privacy
notice to actual behavior, retain dated evidence and repeat the assessment when
scope changes. Recheck current ICO guidance at final legal sign-off: its DPIA
process page notes updates following the Data (Use and Access) Act. Do not treat
that review notice as suspending existing duties.

## 11. Change history

| Version | Date | Change | Approval |
|---|---|---|---|
| 0.1 | 9 November 2026 | Initial code-evidenced launch assessment: 13 flows, 15 risks, 12 assigned actions and conditional residual-risk/sign-off record. | Draft; not approved |

| 0.2 | 9 November 2026 | Reconciles video-provider findings with incoming main `7bd29193`: Bunny selected/enabled in source configuration; production/provider evidence still pending. | Draft; not approved |
