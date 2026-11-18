# Illegal Content Risk Assessment

| Field | Value |
|---|---|
| Assessment owner | Douglas Todd, Founder |
| Approved by | Individual working ratings agreed by Douglas Todd as recorded below; whole assessment not approved |
| Version | 0.4 |
| Status | Incomplete draft; not yet a suitable and sufficient statutory assessment |
| Assessment date | 9 November 2026 (current evidence reconciliation; original baseline 11 September) |
| Next review | Before launch and before any significant change |
| Publication | Internal regulatory record |

## Purpose and Limitation

This document starts CraftSky's Online Safety Act illegal-content risk
assessment. It records the service design, initial risk hypotheses, controls, and
evidence gaps. It is deliberately marked incomplete because CraftSky has not yet:

- completed Ofcom's current risk-profile and Code applicability worksheets;
- assembled all available core evidence and accounted for the absence of real-user incident/prevalence data;
- approved likelihood, impact, and residual-risk ratings; or
- operationally verified several controls needed for serious and time-critical harms.

Douglas Todd is the accountable safety owner. Lack of launch-scale incident data
is an evidence limitation, not a demand to invent prevalence figures or wait for
harm before assessing the service.

It must not be represented to Ofcom, users, investors, app stores, or other third
parties as CraftSky's completed statutory assessment.

## Launch Review Update — 9 November 2026

The initial baseline below dates from September and is being reconciled against
current implementation. It is not a fresh finding that every listed technical
gap remains unimplemented.

Douglas Todd is the sole company operator and is undertaking the accountable
owner's launch review. He intends to monitor safety reports throughout the day,
including outside UK business hours. No backup or staffed overnight cover has
been arranged. Expected first-year active users are in the thousands.

Douglas selected Brevo email alerts from and to `moderation@craftsky.social` to
notify him away from the office. The alert worker is implemented locally, with
persistent batching/retries and generic console-link emails. Production
configuration, deployment and actual inbox/phone receipt remain unverified.
See [alert validation](../../docs/changes/2026-11-09-urgent-safety-email-alerts/validation.md)
and [launch approval record](../governance/launch-approval.md). This improves the
proposed detection-to-review path; it does not establish 24-hour human cover,
complete the risk ratings/Code mapping, or approve the assessment.

Current source also contains Bluesky chat integration (`appview/internal/app/deps_chat.go`
and `appview/internal/routes/routes_chat.go`). The historical no-DM assumptions
below cannot support launch risk ratings. Douglas Todd confirmed on 9 November
2026 that Bluesky-backed DMs are included at launch. The updated children's risk
assessment records the provider/reporting boundary; live availability remains
unverified in this review.

Urgent report reasons and restricted CSEA workflow chronology now exist in
`appview/internal/moderation/taxonomy.go` and
`appview/internal/safetyincident/csea_workflow.go`. Their presence does not prove
NCA organization registration, trained handling, portal access or a completed
operational exercise. CSEA-01 is being reviewed as a concrete dependency of
OSA-01. The owner reports that the NCA application was submitted; approval,
portal access and a reporting exercise remain unverified.

## Current Evidence Baseline — 9 November 2026

This section supersedes the September control descriptions below. Source
inspection establishes implementation, not production availability or efficacy.
Owner-run exercises are identified separately; none establish that a real
illegal-content incident occurred.

| Evidence | Current finding | Remaining uncertainty |
|---|---|---|
| `appview/internal/moderation/taxonomy.go` | Child exploitation, grooming, abuse material, intimate-image abuse, terrorism, immediate danger, threats, self-harm, stalking, doxxing, evasion and fraud allegations have named routes. Allegations remain separate from legal judgments. | Verify client presentation and production routing. Other offence areas still need an explicit handling mapping; a dedicated UI category for every offence is not presumed necessary. |
| `appview/internal/moderation/policy.go` and `policy_test.go` | Child-safety, sexual-violation, threat/violence and stalking/privacy decision reasons exist. Named reasons other than `other` can receive strikes; severe suspension uses an explicit allowlist and rationale. | Fraud and self-harm are outside the severe-suspension allowlist; assess whether the available response is sufficient in concrete serious cases. Do not label all serious sanctions absent. |
| `appview/internal/moderation/system_detected.go` and `system_detected_integration_test.go` | A human-confirmed safety incident can create a system-detected case without a user report and use existing enforcement/appeals. | This is not a generic ability to create a case from any observation; verify the actual operator/incident path and detection coverage. Tests were inspected, not rerun for this document update. |
| `online-safety/operations/reporting-mailbox-readiness.md` | Owner reports external receipt/reply and routine intake, linked case, acknowledgement and no-action flow passed. Manual five-business-day acknowledgement procedure approved. | Specialized/urgent variants, deadline exercises and deployed shared-URL fix remain outstanding. |
| `online-safety/operations/illegal-content-response.md` | Emergency and disclosure procedures approved; owner reports credible-threat and refused unverified-authority console exercises passed. | Training, genuine agency contact, secure transfer and unavailability handling are not established by synthetic console tests. |
| `online-safety/operations/intimate-image-readiness.md` | Specialist workflow exists; owner approved internal 24-hour target from original receipt; local linked-receipt regression verified. | Live configuration, visibility action, matching-content process and benign exercise remain pending. |
| `docs/operations/restricted-evidence-storage.md` | Dedicated Frankfurt S3 bucket, protections and bucket-only IAM policy created and inspected. Hold-aware cleanup exists in source. | Access key, AppView connection, hold/deletion exercise and retention approval deferred until after branch merge. |
| `docs/changes/2026-11-09-urgent-safety-email-alerts/validation.md` | Durable, privacy-minimized Brevo alerts implemented and locally tested. | Production configuration, delivery and phone notification unverified. No backup moderator or guaranteed overnight cover. |
| `appview/internal/chat/`, `online-safety/assessments/childrens-risk-assessment.md` | Launch DMs use Bluesky; provider reports do not enter CraftSky moderation or Brevo alerts. Owner rejected a duplicate local DM escalation route. | Assess the provider boundary and contact controls without treating outsourcing as removal of risk or silently changing the approved reporting destination. |

### Current service-specific risk judgments to resolve

These are proposed review priorities, not approved Ofcom risk levels or measured
frequencies. Final levels must use the current guidance's risk-level tables,
including its separate image-CSAM, CSAM-URL and grooming tables.

| Risk family | CraftSky pathway and affected people | Controls to credit only to their demonstrated extent | Outstanding assessment decision |
|---|---|---|---|
| CSEA: grooming, image-based CSAM and CSAM URLs (assess each separately) | Public garment/child photos, profiles, replies, links and adult-child contact through DMs; harm includes children pictured who are not users. | Named allegations and sanctions exist; provider DM reports, local public-content reports and emergency procedures have distinct boundaries. | Do not infer low risk from craft focus, few users or zero incident history. Assess discovery/contact, media/link exposure, provider dependency, detection and operator availability; NCA/evidence readiness remains open. |
| Harassment, stalking, threats, hate and coercive behaviour (separate final ratings) | Replies, mentions, quotes, profiles, DMs, impersonation and repeated contact; garment/body imagery and business addresses can identify targets. | Blocks/mutes, case decisions, visibility actions, severe reasons and approved emergency process. | Assess evasion, cumulative behaviour, non-user victims, response delay and inability to prevent independent network copies. |
| Intimate-image abuse, extreme pornography, adult sexual exploitation and cyberflashing (separate final ratings) | Misused garment photos, altered images, unsolicited public replies/mentions, links and private contact. Do not assume native DM image upload without verifying supported payloads. | Named sexual allegations, visibility controls, specialist workflow and approved receipt-based internal target. | Actual matching/removal and deadline performance remain unverified; public content prohibition alone is not prevention evidence. |
| Fraud, financial-services offences and proceeds of crime (separate final ratings) | Fake craft shops/designers, phishing, impersonation, business profiles and external payment links; victims may be non-users. | Fraud/scam allegations, impersonation sanctions, strikes and visibility action. | Assess serious/repeat fraud response and the off-platform transaction boundary; no claim of payment monitoring. |
| Encouraging/assisting suicide and serious self-harm | Targeted replies/DMs, instructions, images and external links; distinguish unlawful encouragement from legitimate health/recovery discussion. | Approved public-content boundary, named reports, visibility action and emergency procedure. | Assess exposure and escalation despite low initial usage; test that supportive discussion is not treated as encouragement merely by topic. |
| Terrorism, trafficking, unlawful immigration, drugs, weapons, foreign interference and animal cruelty (separate final ratings) | Abuse of text/media, links, profiles, discovery and business offers; ordinary craft blades/tools must not be equated with weapons offences. | Terrorism/violence or other relevant named reasons where the facts fit, generic intake and visibility action. | Map each offence to judgment guidance and available sanctions; do not force a mismatched reason to obtain suspension. Absence of expected craft demand is a hypothesis, not evidence that abuse cannot occur. |
| Other relevant illegal content | Malicious links, computer misuse, unlawful threats/disclosures or communications meeting the applicable offence test. | Reporting, visibility and account controls subject to reason/permission boundaries. | Record the offence and context. Keep civil/IP, product-quality and other excluded matters in their correct framework rather than assuming every policy violation is OSA illegal content. |

### Current guidance and completion boundary

The Ofcom regulatory-document index was checked on 9 November 2026. It lists
the June 2026 risk guidance and a newly issued **2 November 2026 U2U illegal
content Code**, replacing the February Code cited by the baseline. The current
guidance and Code were opened for this review; the initial ICU measure-by-measure applicability review is now recorded in
`../registers/ofcom-control-mapping.md`. Evidence closure, conditional feature
inventories and the distinct children's Code mapping remain incomplete.

The feature review must include public business addresses/location descriptions
and location disclosed in text/images, even without live location tracking.
Chronological home-feed ordering does not remove risks from search, discovery,
reposts or external sharing. No ads/paid reach reduces one incentive but does
not eliminate commercial scams or harmful contact.

Douglas Todd confirmed on 9 November 2026 that video uploads are enabled in the
intended launch release. Assess uploaded video, playback, reposting and video
links alongside images and text. DMs remain included as already approved.
This is a scope decision, not proof of live upload availability, video safety
review or scanner coverage. The publication manifest's historical
`videoProductionDisabled` control conflicts with this scope and must be
reconciled with an evidenced video-enabled release path before final approval;
do not satisfy it by falsely recording that video is disabled.

### Agreed Risk Rating: Grooming

**Current design-risk level: High. Agreed by Douglas Todd on 9 November 2026.** This is a
service-specific assessment judgment, not a report of actual offending or an
assertion that every interaction is dangerous.

Children can access CraftSky (approved access assessment), one-to-one DMs are
included, and public profiles can contain descriptions and images from which a
person's likely age may be inferred (`appview/internal/api/profile.go`).
Public interaction and follower/mutual-follower functionality can also support
contact discovery (`appview/internal/routes/routes_profile_notification.go`).
This combination supports the high-risk starting point in Ofcom's June 2026
guidance, table 13.3, page 67 (printed page number).

The credible pathway is an adult identifying a child from public craft/profile
content, building trust through replies/follows, then using DMs or external
links to move into more private contact. Harm may be severe and continue outside
CraftSky. No observed incident count or prevalence estimate is asserted.

Existing contact settings, blocks and report routes offer protection, but the
review has not established strong evidence supporting a lower risk level.
In particular, the missing-declaration `following` fallback is not proof that
every account has that setting, or that contacts are safe. Bluesky-only DM
reporting remains the owner's chosen scope; this judgment does not authorize a
duplicate CraftSky reporting route.

For the next control review, map the applicable content moderation, reporting
and governance measures, and examine current Code measures ICU F1/F2 (child
defaults/support) with their age-determination and functionality conditions.
Do not assume every measure applies solely because the proposed rating is high,
or treat the absence of verified ages as exclusion of children. Agreement to
this rating does not approve the effectiveness of safeguards, acceptance of
remaining risk or overall launch.

#### Initial grooming-control review

- **Accountability and response:** Douglas is the named owner; approved
  emergency procedures and the owner-run console exercises are recorded above.
  Brevo activation and unavailability handling remain unresolved dependencies.
- **Contact choice and reporting:** native `MessagingExplanation` and web
  `MessagesWorkspace.svelte` disclose the shared Bluesky inbox/reporting
  boundary. Everyone/following/nobody choices exist in source. Their presence
  does not prove an appropriate default on every existing account, age-specific
  safeguards or provider response performance.
- **Age information:** `OnboardingAcceptance` in
  `appview/internal/api/onboarding.go` records only minimum-age acceptance and
  policy version. The strict request test rejects date of birth and age-band
  fields. This flow does not distinguish a 16–17-year-old from an adult. The
  review has not established an account-level child classification from another
  source; public-profile age inference as an abuse pathway is not an approved
  system for classifying users.
- **Conditional Code measures:** current ICU F1/F2 are conditional on an
  existing means of determining a user's age/age range, as well as the relevant
  risk/functionality conditions. Record applicability as unresolved pending the
  complete age-information/children's-control review. Do not assert that these
  measures automatically apply in full, or that not collecting birth dates
  disposes of wider child-safety duties. The current read fallback alone does
  not prove compliance with their connected-child-account definitions.
- **Decision boundary:** no new age collection, forced change to shared Bluesky
  settings, child-profile classifier or duplicate DM reporting is authorized by
  the agreed rating. Verify existing behavior before proposing changes.

### Agreed Working Ratings: Image-Based CSAM and CSAM URLs

**Medium as the initial working level for each**, separately, agreed by Douglas
Todd on 9 November 2026. Final high/medium calibration and control review remain
pending. Use these levels to
start the applicable-measures review, not as evidence that production controls
are effective or as a cap on the final assessment.

For image-based CSAM, images and launch videos provide an upload/sharing path;
social networking, child access and DMs provide multiple relevant features.
For CSAM URLs, public text, profile links and messages can distribute links even
when the underlying material is hosted elsewhere. A public-data source does not
remove the risk of CraftSky making that material discoverable.

The present record does not establish comprehensive effective checking of all
images/videos or all links, so a Low conclusion is not supported. There is no
documented real-incident prevalence evidence in this launch review; zero active
users cannot be treated as evidence of effective prevention. A High conclusion
must also be examined against the combined factors and any new evidence,
including unknown provider/detection coverage. No observed offending is asserted.

The applicable Ofcom reference is the June 2026 risk guidance, printed pages
64–66 (the separate image-based CSAM and CSAM-URL tables). These proposals are
CraftSky judgments from the confirmed features and evidence limits above.

### Agreed Working Ratings: Harassment/Stalking/Threats/Abuse and Hate

**Medium working risk** for each of these two priority-harm categories,
separately, agreed by Douglas Todd on 9 November 2026. The ratings reflect the current design and
limited evidence, not incident frequency or an accepted level of harm.

| Category | Likelihood reasoning | Impact and existing controls | Why Medium is proposed; what could change it |
|---|---|---|---|
| Harassment, stalking, threats and abuse | Public replies, quotes, mentions, profiles and DMs permit repeated unwanted contact, intimidation, doxxing and coordinated attacks. Craft/business identity and location disclosures can connect abuse to someone's offline life. Small initial scale and craft focus reduce expected volume, but do not prevent targeting. | Harm can be severe for an individual, including fear, psychological harm or physical danger. Named harassment, threat/violence and stalking/privacy reasons support visibility action, strikes and justified severe suspension. Blocks/mutes and approved emergency procedures exist; urgent alert delivery and complete operational coverage remain unverified. | Multiple credible contact/amplification paths and partially evidenced controls support a material working rating. Do not claim Low because tools exist. The record does not yet establish widespread abuse or demonstrated systematic targeting; if repeated evasion, coordinated targeting, serious credible threats or ineffective controls emerge, reassess promptly, including High. |
| Hate | Text, video, images, profile content, replies and links can target protected groups, including people outside CraftSky. Reposts and federation can spread the material beyond an initial thread. A chronological home feed removes one amplification mechanism but not search or sharing. | Potentially severe psychological/physical harm and exclusion. Named hate allegations and decision reason support visibility action, strikes and justified severe suspension. Context and offence tests must distinguish illegal hate from disagreement or merely offensive expression; policy rules may separately be broader. | Material exposure pathways with unproven live response effectiveness support Medium for initial planning. No reviewed service evidence establishes a sustained hateful-content pattern. Escalate if hateful networks, repeated exposure, serious targeting or control failures become evident; absence of reports alone is not proof of low risk. |

Confidence is limited by the pre-launch evidence base and unverified live
controls. Severe individual impact is expressly retained in these judgments;
Medium is not a statement that the consequences are moderate. Final calibration
must consider the combined likelihood and impact under the general risk-level
table and applicable Risk Profiles. These are CraftSky-specific proposals, not
ratings assigned by Ofcom. Controlling/coercive behaviour remains a separate
category to assess, even where some of its conduct overlaps harassment.

### Agreed Working Rating: Controlling or Coercive Behaviour

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess this separately
from harassment because apparently ordinary individual contacts can form a
pattern of abuse in the context of an intimate or family relationship.

- **Pathway/likelihood:** an abusive partner or family member could use public
  posts, photos, profiles, follows and business/location information to monitor
  a victim, then use replies, impersonation, threats or DMs to pressure, isolate
  or humiliate them. Federation and alternate accounts limit the protection
  achievable through a CraftSky block. There is no reviewed incident-frequency
  evidence; the feature combination establishes a credible pathway rather than
  a measured probability.
- **Impact:** psychological harm, loss of autonomy, fear and potentially physical
  danger may be severe, including for people who are not CraftSky users.
- **Existing controls:** harassment, stalking/privacy, impersonation and
  threat/violence reasons may support action where the evidence fits. Blocks,
  mutes, visibility controls, sanctions and approved emergency procedures exist.
  These are partial protections; no claim is made that a victim can safely end
  abuse by blocking or that Bluesky reports trigger local CraftSky alerts.
- **Reason for Medium:** contact/discovery features and contextual abuse create
  material risk, while there is no service-specific evidence yet of systematic
  exploitation. Live effectiveness and cumulative-pattern review remain
  unverified. Severe impact is retained; reassess towards High if evidence of
  serious targeting, evasion or ineffective safeguards warrants it.
- **Handling to verify:** consider the reported pattern and immediate danger;
  distinguish a policy concern from a supported criminal classification. Avoid
  unnecessary disclosure of a reporter's identity or whereabouts, and check a
  safe contact route before correspondence that could expose a victim. Do not
  require the victim to contact the alleged abuser. These are proposed review
  requirements, not evidence that a survivor-sensitive exercise has passed.

Reference: Ofcom June 2026 Risk Profiles, user identification, connections,
communication and location factors. This rating is CraftSky's proposed judgment;
it is not assigned by Ofcom and does not complete the legal offence assessment.

### Agreed Working Rating: Intimate-Image Abuse

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** This is an initial
service-specific judgment, subject to final calibration and live control review.

- **Pathway/likelihood:** images or launch videos may be shared without the
  depicted person's consent, manipulated into intimate imagery, reposted or
  linked from posts/profiles/messages. Garment-modelling photos create an
  identifiable-person context, but ordinary clothing/craft photos must not be
  treated as intimate or illegal merely because they show a body. Consent to
  public craft photography is not consent to abusive alteration or distribution.
- **Impact:** potentially severe distress, humiliation, threats, loss of work or
  physical danger. Depicted people may be non-users; copying/federation can
  prolong harm after CraftSky restricts visibility.
- **Existing controls:** named intimate-image/sexual-violation routes, visibility
  and account sanctions, external mailbox intake and specialist review workflow.
  The owner approved an internal 24-hour target from original receipt. The
  linked-intake clock correction is locally verified; live configuration and
  exercise remain pending.
- **Limits:** workflow outcome fields record a moderator's judgment, searches
  and action; filling them does not itself search for matches or remove content.
  Actual same/substantially-same search and visibility action must be verified.
  Do not claim image checks cover full video, or that CraftSky removes PDS or
  independent copies. Bluesky-only DM reporting remains distinct.
- **Reason for Medium:** upload, contact, discovery and reposting provide
  credible abuse paths; some response controls exist but effectiveness is not
  fully evidenced. The record does not establish a sustained abuse pattern or
  prevalence. This limited-confidence working rating retains severe individual
  impact and must be reconsidered, including High, if targeting, dissemination
  or response failures warrant it. It is not acceptance of the remaining risk.

Ofcom's June 2026 Risk Profiles identify media sharing and reposting as relevant
factors. The specific statutory qualifying-report/deadline duties and Code
measure applicability require separate review; the internal 24-hour target does
not replace them. See `online-safety/operations/intimate-image-readiness.md` for
the outstanding evidence plan. No live action or new policy approval follows
from this proposed rating.

### Agreed Working Rating: Extreme Pornography

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess this separately
from intimate-image abuse, CSAM and the wider policy prohibition on pornography.
A policy violation is not automatically the specific criminal offence.

- **Pathway/likelihood:** image/video uploads and indexed media could be misused
  to publish prohibited extreme pornographic material; text/profile links and
  search can facilitate discovery. Launch video adds moving-image coverage to
  the assessment. CraftSky is not intended as an adult-content service and the
  owner-approved content approach prohibits pornography, but rules and intended
  use alone do not prevent malicious posting or federated exposure.
- **Impact:** potentially severe psychological harm from unsolicited exposure,
  particularly for children or vulnerable people; depicted victims may also be
  affected independently of whether they use CraftSky.
- **Controls:** adult/graphic or sexual-violation allegations and reasons allow
  content review and appropriate visibility/account action where the facts fit.
  Public-content rules, reporting and blocking provide partial protection.
  Current evidence does not establish comprehensive live detection or full
  video checking; do not equate image-scanner infrastructure with that result.
- **Reason for Medium:** media publication and discovery create credible
  exposure routes with incomplete live assurance. There is no reviewed evidence
  of service-specific incidence or a sustained distribution network. Confidence
  is limited; a High rating remains possible if severity, targeting, exposure
  patterns or control failures warrant it. No accepted residual-risk conclusion
  follows from the working proposal.
- **Handling boundary:** apply current offence/judgment guidance with context;
  keep ordinary craft, health or educational material distinct. Use safe review
  and minimum necessary evidence. Do not download or redistribute suspected
  material as part of a benign readiness test.

Reference: Ofcom June 2026 Risk Profiles, media-sharing and UGC-search factors.
The Medium proposal is CraftSky's judgment, not an Ofcom-assigned rating or
approval of the full content policy.

### Agreed Working Rating: Sexual Exploitation of Adults

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** This category covers
content/contact facilitating exploitation; it is distinct from general adult
content, intimate-image abuse and the separately assessed trafficking category.

- **Pathway/likelihood:** an account could use a false craft-business identity,
  modelling/collaboration offer, profile or external link to recruit or pressure
  someone into sexual exploitation. Public replies and DMs permit movement from
  apparently ordinary contact into private coercion. Offers or profiles could
  also advertise or facilitate exploitation. These are plausible misuse
  scenarios, not observed incidents or evidence against legitimate businesses.
- **Impact:** potentially severe sexual, psychological, physical and financial
  harm, including to vulnerable adults and non-users.
- **Controls:** the named sexual-exploitation allegation maps to
  `sexual_violation`; justified severe suspension, strikes and visibility action
  are available where the evidence supports that reason. Public-content rules,
  blocking, reporting and the approved emergency/disclosure procedures provide
  partial safeguards. Bluesky receives DM reports; local Brevo alerts do not
  cover that route. No provider-response performance is established here.
- **Reason for Medium:** public identity/offer and private-contact pathways
  establish material risk with incomplete operational assurance. Craft focus
  and lack of advertising reduce some incentives but cannot exclude targeted
  recruitment. Confidence is limited by the pre-launch evidence base; serious
  recruitment patterns, targeting or response failures would warrant prompt
  reassessment, including High. Severe impact is retained in the judgment.
- **Handling to verify:** assess coercion and immediate danger in context;
  protect necessary evidence and use safe correspondence. Do not infer a
  criminal offence from a person's occupation, appearance or consensual adult
  conduct alone. Apply the relevant offence guidance and distinguish the wider
  platform policy from the legal classification.

Ofcom's June 2026 Risk Profiles identify profiles, offers and private contact as
relevant pathways. Medium is CraftSky's proposed working judgment, not a finding
that exploitation has occurred or acceptance of remaining risk.

### Agreed Working Rating: Human Trafficking

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Consider trafficking
for labour or sexual exploitation separately from adult sexual exploitation
and unlawful immigration; the categories overlap but are not interchangeable.

- **Pathway/likelihood:** false craft-work, workshop, collaboration or modelling
  offers could introduce contact, then use DMs or external links to arrange
  travel or recruitment for exploitation. Business identities, profile details
  and location disclosures could support targeting or coordination. These are
  misuse hypotheses from launch functionality, not evidence of offending by
  craft businesses or a measured incident rate.
- **Impact:** potentially severe loss of liberty, coercion, sexual/physical harm
  and exploitative labour, affecting adults, children and people outside the
  service. Private/off-service continuation can make detection difficult.
- **Controls:** draft Guidelines prohibit trafficking and exploitation; reports,
  visibility action and approved emergency/authority procedures provide partial
  response paths. Use a named sexual-violation, child-safety, threat or fraud
  reason only where its facts fit. There is no dedicated trafficking reason in
  the reviewed taxonomy; `other` does not permit strikes or severe suspension.
  Sanction coverage and safe escalation therefore need explicit mapping.
- **Reason for Medium:** identifiable profiles, offers and private contact create
  multiple relevant pathways, while real-world harm could be severe. No reviewed
  evidence establishes a sustained recruitment pattern. Effectiveness evidence
  and offence/sanction mapping are incomplete, so confidence is limited and Low
  is not supported merely by craft focus. Reassess, including High, if targeting,
  recruitment or control failures emerge. This is not acceptance of remaining
  risk or a claim that every job/travel offer is suspicious.
- **Handling to verify:** prioritize credible danger and necessary preservation,
  avoid exposing a victim's identity/location through correspondence, and apply
  the approved independent-verification/lawful-disclosure process where relevant.
  Do not require a victim to confront or obtain cooperation from an alleged
  exploiter. No real agency referral or specialist exercise is established here.

Reference: Ofcom June 2026 Risk Profiles on profiles, direct contact, location
and offers. Medium is CraftSky's proposed working judgment; the offence test,
control applicability and final calibration remain separate work.

### Agreed Working Rating: Unlawful-Immigration Offences

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess the relevant
offences separately from trafficking and from lawful migration, asylum,
humanitarian support, travel advice or legitimate international recruitment.

- **Pathway/likelihood:** profiles, posts or purported craft-work contacts could
  advertise or arrange assistance that meets a relevant unlawful-immigration
  offence, with DMs/external links moving coordination out of public view.
  Media could be used to promote such offers. This is a misuse scenario, not
  evidence of a criminal recruitment network or suspicion based on nationality.
- **Impact:** potentially serious physical danger, financial loss and exposure
  to exploitation for users or non-users drawn into unsafe arrangements. Assess
  the actual offence and harm rather than assuming all immigration discussion
  is harmful or criminal.
- **Controls:** reports, content visibility actions and approved emergency/
  authority procedures provide partial response paths. There is no dedicated
  reason for this offence family in the reviewed taxonomy; `other` cannot issue
  strikes or severe suspension. Use a fraud, threat or other named reason only
  if independently supported by the facts. Record the sanction-mapping gap.
- **Reason for Medium:** public discovery/identity and private coordination
  supply relevant functionality with incomplete live assurance and offence
  handling evidence. The craft focus makes this unintended use, but does not
  establish effective prevention. No observed prevalence or established pattern
  is claimed. Confidence is limited and final calibration may change, including
  towards High if unsafe coordination, targeting or failed responses emerge.
- **Handling to verify:** use current legal-judgment guidance and assess danger
  without stereotyping users. Keep legitimate support and recruitment distinct;
  minimize disclosure and follow the approved independently verified lawful
  process before sharing private data. No real referral or review exercise is
  established by this proposal.

Reference: Ofcom June 2026 Risk Profiles, social/profile, direct messaging and
media factors. Medium is CraftSky's proposed working judgment, not a statement
that immigration-related content is generally illegal or an accepted residual
risk conclusion.

### Agreed Working Rating: Fraud and Financial-Services Offences

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** This is distinct from
proceeds of crime and from commercial/IP disputes that do not meet an applicable
criminal offence. Assess financial-services offences within the same priority
category even though providing financial services is not CraftSky's purpose.

- **Pathway/likelihood:** fake shops, designer impersonation, non-existent craft
  goods, phishing, fraudulent commissions and deceptive collaboration offers
  can use posts, business profiles, replies, DMs and external payment links.
  Financial promotions or investment scams could be disguised as business
  opportunities. This is closer to expected craft commerce than several other
  misuse scenarios, increasing its practical relevance; no observed prevalence
  or claim against any real business is asserted.
- **Impact:** financial loss, credential theft, loss of a business identity or
  livelihood, distress and secondary exploitation. Users and non-users may be
  affected, with consequences potentially severe for an individual.
- **Controls:** `taxonomy.go` provides fraud, scam, phishing and impersonation
  allegations. `policy.go` permits visibility action and strikes for the fraud
  reason; impersonation can support justified severe suspension. Fraud itself
  is outside that allowlist, so serious/repeat fraud response needs explicit
  mapping. Do not relabel fraud as impersonation unless supported by the facts.
  Business labels and bans on paid reach are not proof that an offer is genuine.
- **Reason for Medium:** multiple commerce/contact/discovery paths and incomplete
  live response evidence establish material risk. The reviewed launch model
  does not establish an on-platform buyer/seller payment-protection service or
  transaction monitoring; external payment is not a risk exemption. There is
  no reviewed evidence of sustained fraud, and control efficacy/prevalence
  confidence is limited. Reassess, including High, on coordinated scams,
  significant losses, repeat offending or response failures.
- **Handling to verify:** distinguish non-delivery/deception with supporting
  facts from an ordinary disagreement, delayed parcel or civil/IP dispute.
  Record relevant content references and minimum necessary evidence; do not
  ask reporters to send passwords, payment credentials or unnecessary financial
  records. Apply current offence/judgment guidance and the approved lawful
  disclosure process where relevant.

Reference: Ofcom June 2026 Risk Profiles, profiles/connections, offers, comments,
direct messaging, search and hyperlinks. This is CraftSky's proposed working
judgment; it does not establish consumer protection, compensation, live control
effectiveness or accepted remaining risk.

### Agreed Working Rating: Proceeds of Crime

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess criminal-property
offences separately from the fraud that may have generated the property.

- **Pathway/likelihood:** fake craft-business identities, purported sales,
  commissions or refund arrangements could be used to solicit transfers,
  recruit people to move funds, or arrange dealings in criminal property.
  Profiles, searchable content, DMs and external links can enable introductions
  and coordination without CraftSky processing a payment. These are plausible
  misuse scenarios, not evidence that ordinary craft sales are laundering.
- **Impact:** financial loss, coercion, involvement of unsuspecting users in
  criminal arrangements and support for further offending. Harm may affect
  non-users and be serious for an individual, even with low initial volume.
- **Controls:** reporting, visibility action, identity/fraud controls and approved
  lawful-authority procedures provide partial response paths. No dedicated
  proceeds-of-crime reason exists in the reviewed taxonomy. `other` cannot
  issue strikes/severe suspension; a fraud or impersonation reason must be
  independently supported rather than used to bypass that restriction.
- **Reason for Medium:** identity, discovery and private-contact features create
  several relevant pathways, and neither lack of payment custody nor a niche
  community eliminates coordination risk. No reviewed prevalence or sustained
  scheme is established. Live efficacy and sanction mapping are incomplete;
  confidence is limited and final calibration may rise if recruitment, repeated
  schemes, serious harm or control failures emerge.
- **Handling to verify:** assess facts and the relevant offence rather than
  treating an unusual price, refund or pseudonymous account as proof. Minimize
  financial evidence and follow approved independent verification/lawful
  disclosure procedures. This assessment does not classify CraftSky as a
  regulated financial business, establish a transaction-monitoring service or
  approve a financial reporting/disclosure obligation or any actual referral.

Reference: Ofcom June 2026 Risk Profiles, profiles, direct messaging and UGC
search factors. Medium is CraftSky's proposed working judgment, not accepted
remaining risk or an Ofcom-assigned rating.

### Agreed Working Rating: Drugs and Psychoactive Substances

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess relevant
unlawful supply, offers and facilitation separately from legitimate health,
recovery, educational or craft-material discussion.

- **Pathway/likelihood:** posts, profiles, images/videos, search and links could
  advertise unlawful substances or supply, with DMs arranging the transaction
  externally. An apparent craft-supply business could conceal unrelated offers.
  Pseudonymous identities and coded wording may impede detection, but neither
  is proof of illegal activity. No actual supply network or incident rate is
  established by this review.
- **Impact:** poisoning, dependency, coercion, financial loss or serious injury,
  including harm to children or non-users. Some consequences may be fatal;
  the working rating does not minimize that potential severity.
- **Controls:** draft Guidelines prohibit illegal-drug facilitation; reporting,
  visibility action and approved emergency/authority procedures provide partial
  safeguards. No dedicated drugs reason exists in the reviewed taxonomy;
  `other` does not support strikes or severe suspension. Another named reason
  may be used only where independently justified by the facts. Response and
  sanction coverage therefore need explicit review.
- **Reason for Medium:** relevant commerce, discovery and private-contact
  features create credible misuse pathways, with no proven comprehensive live
  prevention or handling. Craft focus reduces expected legitimate demand for
  such offers, but does not prevent malicious use. Confidence is limited by the
  pre-launch evidence base; organized supply, child targeting, serious harm or
  ineffective responses would warrant reassessment, including High.
- **Handling to verify:** judge the substance, offer and context using current
  offence guidance. Do not classify ordinary dyes, solvents, adhesives, plant
  fibres, lawful prescriptions or recovery discussion as illegal merely by
  keyword. Product-safety concerns may belong to a different framework unless
  the applicable criminal content test is independently met. Preserve minimum
  necessary evidence and avoid asking anyone to purchase or test substances.

Reference: Ofcom June 2026 Risk Profiles, profiles/connections, media, offers,
search, messaging and hyperlink factors. Medium is CraftSky's proposed working
judgment, subject to final calibration and control verification.

### Agreed Working Rating: Firearms, Knives and Other Weapons

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Ordinary scissors,
rotary cutters, craft knives, needles and demonstrations are not classified as
illegal weapons content simply because they involve a sharp tool. Actual sales
or supply can nevertheless require assessment under the applicable offence,
including age-related restrictions; legitimate craft use is not a blanket
exception to those rules.

- **Pathway/likelihood:** posts, profiles, business offers, search, media or links
  could advertise unlawful weapons supply; private contact could coordinate it
  externally. Tool-related craft content could camouflage an unlawful offer or
  create false positives during review. No actual weapons network or incident
  frequency is established by this assessment.
- **Impact:** potentially severe or fatal physical injury, intimidation and
  facilitation of other offences, affecting users and non-users. Accidental
  craft-tool injuries or product quality issues require their appropriate
  framework unless the criminal content test is independently met.
- **Controls:** Guidelines prohibit illegal weapons facilitation; reporting,
  visibility action and approved emergency procedures provide partial response
  paths. A threat/violence reason supports serious sanctions where the facts
  fit, but an unlawful sale is not automatically a threat. There is no dedicated
  weapons-supply reason in the reviewed taxonomy; `other` cannot issue strikes
  or severe suspension, leaving a sanction-mapping question.
- **Reason for Medium:** offers, discovery and private coordination provide
  relevant paths, with a particularly important context boundary around craft
  tools. No reviewed prevalence or sustained offending is established, but
  neither intended use nor untested controls support treating the risk as
  negligible. Confidence is limited; serious targeting, unsafe supply, threats
  or response failures may justify High on reassessment.
- **Handling to verify:** review the actual object, transaction, recipient and
  context against current offence guidance. Preserve necessary references,
  prioritize credible immediate danger, and avoid asking a reporter to buy or
  handle an item as proof. Test the distinction using benign descriptions,
  without arranging real weapons transactions.

Reference: Ofcom June 2026 Risk Profiles, commerce/offers, identity, messaging
and search factors. Medium is CraftSky's proposed working judgment, not approval
of a transaction, accepted remaining risk or evidence of effective detection.

### Agreed Working Rating: Encouraging or Assisting Suicide and Serious Self-Harm

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess harmful
encouragement/assistance separately from supportive discussion, recovery,
personal experience and help-seeking. A distressed person is not automatically
an offending account; protective action and sanctions require distinct judgment.

- **Pathway/likelihood:** targeted replies/DMs, videos, images, reposts or links
  could encourage or assist harm. Body-image/garment-fit or health discussions
  may expose vulnerable people to targeting, though their topic is not itself
  evidence of unlawful content. Discovery and sharing can repeat exposure
  without an engagement-ranked home feed. No incidence estimate or established
  encouragement network is asserted.
- **Impact:** potentially fatal or life-changing harm to users or non-users,
  including children. This severity is retained despite the proposed Medium
  working level and uncertainty over frequency.
- **Controls:** the owner-approved policy boundary prohibits harmful promotion
  while allowing legitimate support. A self-harm allegation maps to a named
  reason supporting visibility action/strikes; that reason is outside the
  severe-suspension allowlist. Immediate-danger/threat paths and approved
  emergency procedures exist where relevant facts support them. Do not relabel
  content as a threat merely to obtain a sanction. Live urgent-alert receipt,
  operator unavailability handling and specialist exercises remain unverified;
  Bluesky-only DM reports do not reach local Brevo alerts.
- **Reason for Medium:** private/public targeting, media, links and repeated
  exposure establish material design risk with only partly evidenced controls.
  No reviewed service-specific evidence establishes sustained activity, but
  absence of reports is not proof of prevention. Confidence is limited; serious
  targeting, clusters, extensive exposure or ineffective responses can justify
  High on reassessment. Complete combined likelihood/impact calibration before
  approving the final assessment.
- **Handling to verify:** prioritize credible immediate danger under the
  approved emergency procedure; assess context and avoid punishing help-seeking
  simply for mentioning suicide or self-harm. Review without reproducing harmful
  methods, preserve only necessary restricted evidence and use a benign
  synthetic exercise. Do not claim the service provides clinical care or
  guaranteed round-the-clock intervention.

Reference: Ofcom June 2026 Risk Profiles, messaging, comments, media, reposts
and links. Medium is CraftSky's proposed working judgment, not an Ofcom-assigned
rating, operational assurance or acceptance of remaining risk. Dangerous
eating-disorder content also remains in the separate children's assessment and
broader approved policy boundary; its topic alone does not establish this offence.

### Agreed Working Rating: Foreign Interference

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess the specific
foreign-interference offence, not political opinion, overseas participation,
criticism, advocacy or misinformation as automatically criminal content.

- **Pathway/likelihood:** deceptive accounts or impersonated craft organizations
  could build trust around trade, labour, cultural or environmental topics, then
  use media, links, reposts and user connections to conduct activity meeting the
  applicable offence. Private contact can support coordination, but do not count
  unsupported group-chat/encrypted-chat functionality as present. No foreign
  campaign or actual offender is identified by this assessment.
- **Impact:** intimidation, interference with participation, harm to targeted
  communities and wider public consequences can be serious, including beyond
  CraftSky's users. Exact impact depends on the conduct, reach and targets.
- **Controls:** reports and supported spam, misleading-content, manipulation or
  impersonation decisions can address policy breaches where the facts fit.
  Impersonation can support justified severe suspension; generic `other` cannot
  issue strikes/severe suspension. The current platform-manipulation allegation
  maps to `misleading_commercial`, so do not assume a complete coordinated-state-
  activity or offence-specific response. No specialist attribution capability
  or live effectiveness is established.
- **Reason for Medium:** public identities, connection building, media, links and
  resharing supply multiple relevant functions. Small initial scale, craft focus,
  no paid reach and a chronological home feed reduce some expected opportunities
  but do not remove targeting or external amplification. There is no reviewed
  campaign prevalence evidence. Confidence is limited and final calibration
  must consider actual reach/impact; organized targeting or ineffective response
  could warrant High, while stronger evidence may support a lower level.
- **Handling to verify:** separate platform-policy findings from criminal
  classification; use the current offence guidance, including its required
  connection/intent/conduct elements. Nationality, language, pseudonymity,
  unpopular views or a coordinated legitimate campaign are not enough. Apply
  the approved verified lawful-authority process before private disclosure.

Reference: Ofcom June 2026 Risk Profiles, identity, connections, media,
reposting and links. Medium is CraftSky's proposed working judgment, not an
Ofcom-assigned rating, attribution finding or acceptance of remaining risk.

### Agreed Working Rating: Animal Cruelty

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Keep the relevant
animal-welfare offence distinct from distressing-but-lawful footage, policy
breaches and disagreement about animal-derived craft materials.

- **Pathway/likelihood:** image/video uploads and replies could show, promote
  or solicit unlawful cruelty; profiles, posts, links and private contact could
  direct people to such content or coordinate offending. Wool/fibre sourcing,
  husbandry and animal-related crafts create a context where legitimate
  discussion and abuse footage might both appear. No offending by any real
  producer or actual cruelty network is asserted.
- **Impact:** distress or intimidation to users, including children; underlying
  acts may cause serious suffering to animals and psychological harm to people
  encountering or affected by them. Assess content, context and the applicable
  offence rather than assuming all images of animals or husbandry are illegal.
- **Controls:** graphic-content allegations map to `adult_or_graphic`, which
  supports visibility action, strikes and justified severe suspension where
  that reason fits. Emergency/authority procedures can support relevant serious
  concerns. Non-graphic facilitation may instead reach `other`, which cannot
  issue strikes/severe suspension; record this coverage question. Image-safety
  infrastructure is not proof of animal-cruelty classification or video review.
- **Reason for Medium:** media and contact functions supply credible pathways,
  with incomplete live detection/response evidence. Craft-specific sourcing
  discussions add contextual ambiguity; they do not prove higher prevalence.
  No sustained pattern or incident count is established. Confidence is limited;
  organized abuse, repeated exposure, intimidation or control failures may
  warrant High. The proposed level retains potentially serious consequences.
- **Handling to verify:** distinguish lawful shearing, veterinary care,
  education, reporting/documentation and advocacy from content meeting the
  offence or a broader platform rule. Do not suppress welfare criticism merely
  because it is uncomfortable. Minimize graphic evidence and use benign
  descriptions for readiness exercises rather than real cruelty material.

Reference: Ofcom June 2026 Risk Profiles, social/contact, comments and media
factors. Medium is CraftSky's proposed working judgment, not a finding about
material producers or accepted remaining risk.

### Agreed Working Rating: Cyberflashing

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess the specific
offence and its required content/conduct/intent elements separately from the
broader platform prohibition on unsolicited sexual content. A sexual image,
ordinary garment photograph or link is not automatically proof of that offence.

- **Pathway/likelihood:** malicious users could direct sexual imagery at someone
  through media replies, quotes, mentions or profile interactions. Links and
  private messages can facilitate related exposure or harassment, but require
  their own factual/legal assessment. The inspected CraftSky chat send path
  accepts text; do not invent native DM image/video uploads or assume every
  provider payload is supported. Public image/video posting is confirmed for
  launch, so lack of DM attachments would not remove the public targeting path.
- **Impact:** potentially severe distress, intimidation and sexual harassment,
  including for children or vulnerable users; targeted people may be non-users
  whose identities or images appear on the service.
- **Controls:** draft Guidelines explicitly prohibit cyberflashing. Adult/graphic
  and sexual-violation reasons can support visibility action and justified
  serious sanctions where facts fit, alongside blocks and reporting. There is
  no dedicated cyberflashing allegation in the reviewed taxonomy; verify users
  can find an appropriate route without forcing an inaccurate intimate-image
  complaint. Bluesky-only DM reports remain separate from CraftSky alerts.
- **Reason for Medium:** public media and directed contact create credible
  exposure paths with incomplete live prevention/response evidence. No actual
  incident rate or sustained targeting pattern is asserted. Potential severity
  remains high for an affected person; confidence is limited and repeated
  targeting, child exposure or ineffective controls may warrant High on review.
- **Handling to verify:** consider context, intentional targeting, receipt and
  relevant offence guidance without blaming recipients or treating ordinary
  craft/health imagery as criminal. Minimize redisplay and copied evidence;
  use benign synthetic descriptions for tests. Overlap with harassment, child
  safety or intimate-image abuse requires separate supported judgments rather
  than automatic equivalence.

Reference: Ofcom June 2026 Risk Profiles, profiles/connections, media and contact
factors. Medium is CraftSky's proposed working judgment, not a criminal finding,
proof of available provider payloads or acceptance of remaining risk.

### Agreed Working Rating: Terrorism

**Working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Assess the relevant
terrorism offences separately from controversial opinion, religion, identity,
news, historical material, counterspeech and legitimate advocacy.

- **Pathway/likelihood:** text, images/videos, replies, profiles, links and
  reposts could distribute terrorist content or direct people to recruitment,
  fundraising or other offending. User connections and DMs could support
  movement into private contact or coordination. Craft subject matter is not
  the intended context for this material, but public federation and ordinary
  social functions do not exclude deliberate misuse. No terrorist actor or
  actual campaign is identified by this assessment.
- **Impact:** potentially catastrophic physical harm, intimidation and serious
  psychological harm to users and non-users. Unsolicited media can affect
  children and other vulnerable people even without further engagement.
- **Controls:** a named terrorism allegation maps to `threat_or_violence`, which
  supports visibility action, strikes and justified severe suspension. Draft
  Guidelines prohibit terrorism promotion/facilitation, and approved emergency
  and verified authority procedures provide response paths. Their existence is
  not specialist judgment expertise, comprehensive detection or live urgent
  response proof. Automated exclusion of proscribed-organization accounts and
  applicable Code measures still require explicit evidence review.
- **Reason for Medium:** public dissemination, links, contact and coordination
  functions create multiple credible paths with incomplete operational
  assurance. There is no reviewed evidence of sustained terrorist use. Small
  initial reach, craft focus, no paid reach and chronology reduce some expected
  opportunities, but not the possible severity or targeting. Confidence is
  limited and final combined likelihood/impact calibration may require High
  if evidence of actual targeting, dissemination or control failure warrants it.
- **Handling to verify:** use the current terrorism content-judgment guidance
  with context and the relevant offence; do not infer illegality from a person's
  identity or lawful political/religious expression. Prioritize credible
  immediate danger under the approved procedure, preserve only necessary
  restricted evidence and avoid replaying or redistributing harmful material.
  Exercise the path with benign descriptions rather than actual propaganda.

Reference: Ofcom June 2026 Risk Profiles, social/contact, comments, media, search
and links. Medium is CraftSky's proposed working judgment, not an Ofcom finding,
proof of absence of terrorist content or accepted remaining risk.

### Agreed Working Rating: Other Relevant Illegal Content

**Collective working risk: Medium. Agreed by Douglas Todd on 9 November 2026.** Limit this
to content meeting a relevant offence within the Act's scope, rather than every
unlawful act, civil dispute or breach of CraftSky rules. Assess credible pathways
proportionately; do not claim that every conceivable offence has been examined.

| Pathway considered | Likelihood/impact reasoning | Controls and evidence limits |
|---|---|---|
| Relevant false or threatening communications outside the already assessed priority categories | Posts, replies, mentions and messages permit targeted communications; some may cause serious psychological or physical harm if the actual offence elements are met. Satire, mistakes or disagreement are not automatically offences. | Reporting, harassment/threat reasons, visibility/account controls and approved emergency procedures exist where facts fit. Contextual judgment, repeated targeting and live response remain to verify. Avoid double-counting conduct already assessed as a priority harm. |
| Malicious links and content related to computer misuse, where independently within the relevant-offence test | Apparent patterns, tutorials, shops or private links could expose people to malicious downloads or instructions facilitating a relevant offence; potential outcomes include loss of data, access or funds. Legitimate technical/security education is distinct. | Fraud/misleading or other supported reasons may allow response, but this review establishes no universal URL/malware detection or full offence/sanction coverage. A bad link alone is not proof of the specific offence. |
| Unlawful private-information disclosure where an applicable relevant criminal offence is met | Posts, profiles, images and messages can disclose identifying information about users/non-users, potentially leading to intimidation or physical danger. Privacy complaints are not automatically criminal, and public availability is not blanket permission for harmful targeting. | Stalking/privacy reporting and sanctions support some cases; the DPIA, lawful-basis and private-disclosure procedures remain separate work. Establish context and the actual legal test before criminal classification. |

**Reason for Medium:** these credible communication/link/disclosure paths can
cause serious individual harm, while existing response tools provide only
partially evidenced protection. No incident frequency or universal offence
coverage is established. Confidence is limited and the collective level must
be revisited when evidence or service features change, including if High is
warranted. Where a new relevant pathway is identified, add the reasoning rather
than treating this umbrella rating as an automatic assessment of it.

Keep copyright/trade-mark disputes, ordinary product-quality concerns, civil
discrimination and consumer-policy matters under their correct frameworks;
their importance does not automatically make them OSA illegal content. Content
may independently meet another relevant criminal offence, which then requires
its own supported judgment. No sanctions are authorized solely by this rating.

Reference: Ofcom June 2026 guidance, other-illegal-content assessment and the
collective-rating approach (printed pages 21 and 23), with current offence
judgment guidance to apply to individual cases. This is CraftSky's proposed
working judgment, not accepted remaining risk or a completed statutory record.

## Consolidated Working Ratings — 9 November 2026

These are the owner's individual working-rating agreements, not approval of
live safeguards, final calibration, accepted remaining risk or this assessment
as a whole. Read each row with its pathway/impact/control reasoning above.

| Priority category | Working rating | Agreement status |
|---|---|---|
| Terrorism | Medium | Agreed |
| CSEA overall | High | Adopted following owner authorization to select the best-supported rating; reasoning below |
| CSEA: grooming | High | Agreed |
| CSEA: image-based CSAM, including video | Medium | Agreed |
| CSEA: CSAM URLs | Medium | Agreed |
| Hate | Medium | Agreed |
| Harassment, stalking, threats and abuse | Medium | Agreed |
| Controlling or coercive behaviour | Medium | Agreed |
| Intimate-image abuse | Medium | Agreed |
| Extreme pornography | Medium | Agreed |
| Sexual exploitation of adults | Medium | Agreed |
| Human trafficking | Medium | Agreed |
| Unlawful-immigration offences | Medium | Agreed |
| Fraud and financial-services offences | Medium | Agreed |
| Proceeds of crime | Medium | Agreed |
| Drugs and psychoactive substances | Medium | Agreed |
| Firearms, knives and other weapons | Medium | Agreed |
| Encouraging or assisting suicide and serious self-harm | Medium | Agreed |
| Foreign interference | Medium | Agreed |
| Animal cruelty | Medium | Agreed |
| Cyberflashing | Medium | Agreed |
| Other relevant illegal content (collective category, outside the 18 priority kinds) | Medium | Agreed |

### Adopted Overall CSEA Rating — High

**Overall CSEA working risk: High. Adopted on 9 November 2026 following
Douglas Todd's express authorization to select the best-supported rating.**
This resolves the previously open overall-rating decision. Confidence is limited
by the pre-launch evidence base; the whole assessment and launch remain pending.

#### Service-specific balance

- **Factors reducing expected exposure:** CraftSky is a specialist textile-craft
  community, does not target children, and the owner does not consider it
  particularly appealing to children. Expected first-year active users are in
  the thousands, and the home feed is chronological. These are relevant
  mitigating characteristics. There is no measured age distribution or reviewed
  service-specific offending prevalence; do not infer offending or high
  incidence from this rating, or treat the absence of active users as proven
  prevention.
- **Child-contact pathway:** the approved children's-access assessment and 16+
  policy permit access by 16–17-year-olds. Public profile descriptions, images
  and ordinary interaction may reveal that an account holder is a child;
  a child depicted in someone else's photo is not itself a child account.
  Replies/follows and DMs can support trust-building and movement into private
  or off-service contact. There is no evidence that CraftSky offers structured
  child-age labels or age-filtered discovery. Reviewed graph routes expose own
  lists and another account's mutual followers, not unrestricted full lists
  for every other account.
- **Impact:** exploitation or grooming can cause severe, enduring harm to an
  individual child, including harm outside CraftSky. Low expected absolute
  numbers do not make that consequence minor.
- **Credit for safeguards:** everyone/following/nobody incoming choices,
  acceptance-gated replies, suppressed rich previews before acceptance,
  either-direction blocking, reporting routes and human moderation are relevant
  partial protections. Six focused backend contact tests passed locally. An
  existing Everyone setting is respected; a missing-declaration following
  fallback does not establish a universal enforced default. Initial request
  text can be exposed before acceptance. Neither following nor accepting a
  request proves a contact safe. Source/local tests do not establish live
  provider enforcement, detection coverage or reporting performance.
- **Response limits:** accountable ownership and procedures exist, and sanction
  coverage has been improved in source. Production alert delivery, aspects of
  specialist reporting/evidence readiness, live checks and operator absence
  cover remain outstanding. These gaps limit the protection we can credit;
  they do not themselves prove harm or independently force a High rating.

#### Judgment and guidance

The combination of a credible child-identification/contact pathway, potentially
severe individual harm and incompletely evidenced prevention supports **High**
for current launch planning, despite the mitigating audience and scale factors.
The existing High grooming assessment remains a substantive part of this
judgment. The image/video-CSAM and CSAM-URL ratings remain separately Medium.
This is a reasoned overall judgment, not an average, an automatic maximum of
subtypes or a rule that every social platform must be High.

Rechecked [Ofcom's current Risk Assessment Guidance, June 2026](https://www.ofcom.org.uk/siteassets/resources/documents/online-safety/information-for-industry/illegal-harms/updates/risk-assessment-guidance-and-risk-profiles.pdf?v=419947)
on 9 November 2026: Part 3 paragraphs 3.3–3.6 call for an overall and separate
subtype assessment, caution where evidence is inconclusive, and judgment rather
than mechanical table use. General Table 13 includes severe individual impact;
Table 13.3 gives a High starting point for possible child access plus one-to-one
contact and profiles from which likely child status may be determined, subject
to strong reasons for a lower conclusion. The current audience expectations and
partial safeguards do not yet provide sufficiently strong counter-evidence.

The owner's earlier challenge is retained in the approval chronology. It
prompted the narrower discovery review recorded in
`grooming-control-review.md`; his subsequent authorization resolves the rating
choice without approving the whole assessment or accepting remaining risk.
No new age collection, child classifier, shared-provider setting change,
reporting destination, sanction, deployment or public promise follows from it.

#### Review conditions

Reassess before launch after the deferred benign hosted checks and full control
mapping, after a serious incident or material functionality/audience change,
and at least annually. Evidence that could justify a different conclusion
includes a better-supported audience/access assessment and demonstrated
contact, detection and timely-response safeguards. Do not promise an automatic
downgrade when a checklist is complete; evaluate their combined effectiveness.
Do not collect additional child data solely to support a lower rating without
a separate necessity/privacy assessment. Applicable Code measures must still be
mapped using their actual risk, functionality, size and age-information conditions.

### Initial Current-Code Mapping — 9 November 2026

`../registers/ofcom-control-mapping.md` records every ICU measure identifier,
its application conditions, current evidence and remaining work, together with
the assessed-harm-to-decision routes. CraftSky is multi-risk under ICU 5.6 even
at small scale: terrorism Medium and overall CSEA High already meet the separate
category condition. The subtype ratings are not counted as extra categories.
The mapping identifies governance/training/resources and reporting-information
work; it does not impose all large-service measures or substitute overall CSEA
for image/URL-specific triggers. No alternative measure is deemed approved.

The immediate source-demonstrated gap is pre-submit disclosure information,
acknowledgement with indicative decision time/outcome information and an
opt-out from non-ephemeral follow-up (D3–D6), with appeal targets under D8.
The proposal at `../../docs/changes/2026-11-09-reporting-information/proposal.md`
was subsequently narrowed by the owner to brief post-submit process/estimate
copy, implemented locally with release pending. No reporter-status UI or wider
preference/contact workflow was approved. The remaining information and
operational findings stay open.
Children's ratings and their distinct PCU conditions remain outstanding.

### Outstanding Work After Individual Ratings

1. Complete the feature/Risk Profile and current Code measure applicability
   record, including conditional child-default measures and evidence limits.
2. Resolve the offence-to-decision/sanction mappings identified above. In
   particular, do not force unsupported named reasons to obtain severe action;
   document the actual capability and justified alternatives.
3. Complete or explicitly track the deferred live checks for alerts, specialist
   deadlines, video/detection coverage, evidence/holds and visibility actions.
4. Reconcile policies, operator capacity/unavailability, detection/provider
   dependencies and monitoring with these risks. Recalibrate where evidence is
   inconclusive, with the caution recommended by the guidance.
5. Obtain whole-assessment approval only after the evidence and unresolved
   actions are reviewed. Rating agreements do not close OSA-01.

Video uploads and DMs are confirmed for launch. Map that scope and each relevant
Risk Profile factor to applicable measures and evidence. Keep source inspection,
local checks, owner exercises and production results distinguishable.

## Historical September Baseline — Superseded Control Descriptions

The following hypotheses and tables are retained for traceability. Their
statements that dedicated categories, workflows, ownership or proactive cases
are absent are historical and must not be used as current findings. The
current evidence table above governs those implementation facts; the historical
likelihood labels are not final November ratings.

## Method

The working scale follows Ofcom's four risk levels: negligible, low, medium, and
high. Each risk requires separate consideration of likelihood and impact,
including indirect harm to non-users. A low expected frequency does not make a
catastrophic harm unimportant.

The review considered intended and unintended use, user characteristics,
functionalities, algorithms, business model, governance, and existing controls.
Ratings below are hypotheses for validation, not approved findings.

## Service-Wide Risk Factors

### Risk-increasing factors

- Public text, image, video, link, profile, reply, mention, and quote-post
  publication.
- Public discovery through feeds, profiles, search, notifications, and sharing.
- Possible use by children and no current age assurance.
- Federated publication and independent third-party copying or display.
- External links can direct users to illegal content outside CraftSky.
- A single configured moderator-administration actor model; named human staffing
  and cover are unconfirmed.
- No dedicated emergency, CSEA, intimate-image, or law-enforcement workflow.
- Report taxonomy does not map to all priority offences.

### Risk-reducing factors

- No group chat, livestreaming, disappearing content, or user location sharing
  identified in the reviewed launch design. Bluesky-backed one-to-one DMs are
  included and require their own control assessment.
- Craft-focused niche and no advertising or paid reach.
- Chronological home feed rather than engagement-ranked recommendation.
- Authenticated reporting on posts and profiles.
- Server-side visibility enforcement, blocking, muting, case records, appeals,
  strikes, and suspension.
- Reports are private and do not automatically establish wrongdoing.

## Priority Illegal Content Working Register

Ofcom's June 2026 guidance says providers must assess 18 kinds of priority
illegal content after adding serious self-harm and cyberflashing. Ofcom's summary
also lists CSEA and certain other areas in more granular lines. This register
keeps those granular lines visible so a serious subtype is not lost during
implementation mapping.

| Risk area | Credible CraftSky pathway | Initial hypothesis | Current controls | Material gap |
|---|---|---|---|---|
| Terrorism | Public propaganda, praise, recruitment links, or instructions in posts/media. | Low likelihood; severe impact | General reporting and takedown | No dedicated reason, expertise, or urgent escalation. |
| CSEA | Public grooming, exploitative media, links, or contact involving a child. | Low likelihood; catastrophic impact; residual rating requires specialist review | Generic reporting, blocking, and CraftSky visibility takedown; `adult_or_graphic` can support suspension in some cases | No dedicated intake, restricted handling, NCA workflow, trained owner, or complete sanction mapping. |
| Grooming | An adult builds contact with a child through follows, replies, mentions, and linked services. | Low-to-medium likelihood if children join; severe impact | Public contact plus Bluesky-backed DMs, block/report tools; DM reports go to Bluesky | No age signal, grooming reason, pattern detection, or child-specific controls. |
| Image-based CSAM | Image/video upload contains child sexual abuse material. | Low likelihood; catastrophic impact | Generic reporting and CraftSky visibility takedown; `adult_or_graphic` can support suspension | No dedicated safe review, detection, preservation, NCA reporting, or one-year evidence control. |
| CSAM URLs | Posts or profiles link to CSAM hosted elsewhere. | Low likelihood; catastrophic impact | Link reporting and takedown | No dedicated URL handling, escalation, or NCA workflow. |
| Hate | Posts, profiles, replies, or coordinated attacks target protected groups. | Medium likelihood; major-to-severe impact | Dedicated report/decision reason, blocking, visibility action, strikes, severe suspension | Policy-to-offence mapping, response targets, and moderator guidance incomplete. |
| Harassment, stalking, threats, and abuse | Repeated replies/mentions, block evasion, dogpiling, threats, doxxing, or off-service coordination. | Medium likelihood; major-to-severe impact | Dedicated reason, blocking/muting, cases, sanctions, appeals | No dedicated threat/doxxing/block-evasion reasons or emergency workflow. |
| Controlling or coercive behaviour | Public monitoring, threats, humiliation, or contact as part of domestic abuse. | Low likelihood; severe impact | Blocking and generic reporting; a harassment mapping may support sanctions where facts fit | No specialist category, contextual evidence guidance, or survivor-sensitive process. |
| Intimate image abuse | Non-consensual intimate images, altered intimate images, or links are posted or reposted. | Low likelihood; severe impact; control gap may make residual risk material | Adult/graphic reports and takedown | No qualifying reporter route, identity/authority declaration, expedited complaint, duplicate search, or demonstrated 48-hour process. |
| Extreme pornography | Prohibited extreme pornographic images or links are uploaded. | Low likelihood; severe impact | Adult/graphic reason and takedown | No offence-specific guidance or restricted evidence handling. |
| Sexual exploitation of adults | Posts advertise, facilitate, coerce, or profit from sexual exploitation. | Low likelihood; severe impact | General reporting and takedown | No dedicated reason or escalation. |
| Human trafficking | Recruitment, facilitation, advertising, or coordination through public content/links. | Low likelihood; severe impact | Generic reporting and CraftSky visibility takedown | `other` cannot produce a strike or severe suspension; no dedicated reason, expertise, or law-enforcement protocol. |
| Unlawful immigration | Public content facilitates relevant unlawful immigration offences. | Low likelihood; major impact | General reporting and takedown | No dedicated reason or assessment guidance. |
| Fraud and financial services offences | Fake shops, phishing, counterfeit sales, deceptive links, or investment/payment scams. | Medium likelihood; major impact | Spam, misleading, and impersonation reasons; business labels; takedown/suspension | No fraud reason, trusted reporting, payment/link intelligence, or victim-support process. |
| Proceeds of crime | Accounts use sales or links to conceal, transfer, or benefit from criminal property. | Low likelihood; major impact | Generic reporting and CraftSky visibility action | `other` cannot produce a strike or severe suspension; no dedicated reason, transaction visibility, or escalation criteria. |
| Drugs and psychoactive substances | Sale, supply, or facilitation through posts, profiles, or external links. | Low likelihood; major impact | General reporting and takedown | No dedicated reason or moderator guidance. |
| Firearms, knives, and other weapons | Sale, supply, or facilitation through posts, profiles, or links. Craft tools create contextual ambiguity. | Low likelihood; major-to-severe impact | General reporting and takedown | No specialist reason or context guidance distinguishing ordinary craft tools. |
| Encouraging or assisting suicide and serious self-harm | Instructions, encouragement, targeting, or communities promoting serious self-harm. | Low likelihood; severe impact | Community rule against encouragement; general report/takedown | No dedicated reason, crisis/escalation guidance, or proactive discovery. |
| Foreign interference | Coordinated deceptive political activity uses accounts, craft-adjacent topics, or links. | Low likelihood; potentially major impact | Off-topic, misleading, spam, and impersonation controls | No coordinated-behaviour capability or dedicated assessment evidence. |
| Animal cruelty | Graphic or instructional cruelty content is posted or linked. | Low likelihood; major impact | Adult/graphic and general reports | No dedicated reason or guidance. |
| Cyberflashing | Unsolicited sexual image is directed at a user through a mention, reply, quote, or link. | Severe impact; likelihood requires reassessment with launch messaging and its supported content types | Adult/graphic reporting, blocks, takedown | No cyberflashing reason or expedited/survivor-sensitive handling. |

## Other Legal and Platform Risks

The final statutory assessment must address non-priority illegal content that
independently amounts to a relevant offence under Online Safety Act section 59.
This includes credible pathways involving computer misuse, malicious
communications, threats, or unlawful disclosure where the statutory offence test
is met.

Other matters remain important to Community Guidelines, intellectual-property,
consumer-protection, product-safety, data-protection, and general platform-risk
controls, but must not be classified as Online Safety Act illegal content merely
on that basis. Section 59(6) excludes intellectual-property infringement,
offences concerning the safety or quality of goods, and offences under Chapter 1
of Part 4 of the Digital Markets, Competition and Consumers Act 2024. Civilly
unlawful discrimination is not necessarily a criminal offence.

CraftSky must therefore assess copyright and trade mark infringement, copied
patterns and media, counterfeit goods, unsafe products, commercial claims, and
data-protection issues under their correct legal and policy frameworks while
separately capturing any associated content that independently meets the
statutory illegal-content test.

## Existing Control Assessment

| Control | Evidence in repository | Assessment |
|---|---|---|
| Reporting | Signed-in current-member post/profile/business-event reports with private storage | Implemented, but categories and non-user/affected-person access are incomplete. Reports are not forwarded to a PDS or Ozone. |
| Case review | Grouped reports, evidence, append-only decisions and effects | Implemented for accepted reports. Moderator cannot start a proactive case in v1. |
| Visibility action | Warn, hide, and takedown in CraftSky reads | Implemented. Does not remove source PDS records or third-party copies. |
| Account action | Formal warnings, 12-month strikes, threshold suspension, severe suspension | Implemented, but severe suspension is unavailable for several serious policy categories. |
| Appeals | One appeal lifecycle per owner-visible case, with a prefilled email handoff and moderator confirmation/resolution APIs | Implemented foundation; monitored mailbox intake, correspondence ingestion, response targets, and independent review are not demonstrated. |
| Blocking and muting | Public interoperable blocks; private one-way mutes | Implemented, with federation limitations disclosed in draft Guidelines. |
| Governance | One configured moderator-administration actor and admin APIs | Technical controls exist; human staffing, accountable owner, training, cover, QA, and recusal are unconfirmed. |
| CSEA reporting | Public policy promises reporting | Not operationally demonstrated. Treat as a critical gap. |
| Intimate-image reports | General adult/graphic reports | Does not meet the documented qualifying-report and expedited-process requirements. |

## Historical Completion Actions (Reconcile Against Current Baseline)

1. Assign the accountable online-safety owner and child-safety escalation owner.
2. Complete Ofcom's current risk-profile and Code applicability records.
3. Add report and decision categories for urgent and priority illegal harms.
4. Establish a safe CSEA process, NCA registration, report prioritisation,
   evidence retention, access restriction, and trained cover.
5. Implement a qualifying intimate-image report route, expedited complaints, and
   a measured 48-hour removal workflow.
6. Add credible-threat, emergency, and lawful-authority escalation procedures.
7. Define moderation service targets, out-of-hours handling, conflicts, QA,
   training, and wellbeing support.
8. Test that reports are accessible to users and qualifying affected persons,
   including people who cannot sign in.
9. Reconcile the policy taxonomy with enforceable decision reasons and sanctions.
10. Gather and monitor report volumes, response times, prevalence samples,
    reversals, repeat offenders, and control failures without using report volume
    as proof of guilt.

## Review and Monitoring

Complete a new assessment before a significant service change. Review this
record after implementation, after a serious incident, when Ofcom changes a
relevant risk profile, and at least annually. Record every adopted Code measure
and every alternative measure with its compliance rationale.

## Sources

- [Online Safety Act 2023](https://www.legislation.gov.uk/ukpga/2023/50/contents), particularly sections 9, 10, 20, 21, 23, and 59.
- [Ofcom, Illegal content duties under the Online Safety Act](https://www.ofcom.org.uk/online-safety/illegal-and-harmful-content/illegal-content-duties-under-the-online-safety-act), updated 25 June 2026.
- [Ofcom, Risk Assessment Guidance and Risk Profiles](https://www.ofcom.org.uk/siteassets/resources/documents/online-safety/information-for-industry/illegal-harms/updates/risk-assessment-guidance-and-risk-profiles.pdf?v=419947), June 2026.
- [Ofcom, Illegal Content Judgements Guidance](https://www.ofcom.org.uk/siteassets/resources/documents/online-safety/information-for-industry/illegal-harms/updates/illegal-content-judgements-guidance.pdf?v=419967), June 2026.
- [Ofcom, Illegal content Codes of Practice for user-to-user services](https://www.ofcom.org.uk/siteassets/resources/documents/online-safety/information-for-industry/illegal-harms/illegal-content-codes-of-practice-for-user-to-user-services-24-feb.pdf?v=391889), February 2025.
- [Ofcom, current regulatory documents](https://www.ofcom.org.uk/online-safety/illegal-and-harmful-content/online-safety-regulatory-documents), checked 9 November 2026.
- [Ofcom, issued U2U Illegal Content Code](https://www.ofcom.org.uk/siteassets/resources/documents/consultations/category-1-10-weeks/statement-protecting-children-from-harms-online/upcoming-codes-of-practice/issued-illegal-content-codes-of-practice-for-user-to-user-services.pdf?v=427438), 2 November 2026; use for the current applicability review rather than the historical February reference.

## Change History

| Version | Date | Change | Author/approver |
|---|---|---|---|
| 0.1 | 11 September 2026 | Initial risk register, control inventory, and completion actions. | Working draft / not approved |
| 0.2 | 9 November 2026 | Reconciles current source, owner approvals/exercises and provisioned storage; separates historical gaps, pending live evidence and current Code review. Risk ratings and overall assessment remain unapproved. | Working draft / not approved |
| 0.3 | 9 November 2026 | Records individual working-rating agreements and their service-specific reasoning; consolidates all priority categories, CSEA subtypes and other illegal content. Overall CSEA judgment proposed; full control mapping, final calibration and whole-assessment approval remain pending. | Individual agreements recorded; whole assessment not approved |
| 0.4 | 9 November 2026 | Adopts High overall CSEA working risk under express owner authorization after audience/contact-control reassessment; retains separate subtype ratings, evidence limits and review triggers. | Rating decision authorized; whole assessment not approved |


## Sanction Coverage Update — 9 November 2026

The owner approved the scoped sanction proposal and implementation. Current
source adds the reviewed Other illegal content decision route with factual
evidence, assessed other-illegal-content classification, a separate private
legal rationale and a safe owner explanation. It permits strikes and
proportionate severe suspension with a separate severity rationale. Fraud/scam
and self-harm reasons are also severe-eligible under supported human review;
ordinary disputes, distress and help-seeking are not sufficient grounds.
Generic Other remains ineligible for strikes or severe suspension. Reports and
automated signals are not findings. The original taxonomy inventory above is a
historical baseline; there are now 18 decision reasons in source.

See `docs/changes/2026-11-09-moderation-sanction-coverage/validation.md` for local
migration, persistence, privacy, replay, appeal and client checks. Deployment
and hosted operational evidence remain pending. This closes the approved
source-level sanction coverage work only; it does not resolve staffing,
detection coverage, final Code mapping or launch approval. The overall rating
was subsequently adopted as High in version 0.4; effectiveness and full
assessment review remain pending.
