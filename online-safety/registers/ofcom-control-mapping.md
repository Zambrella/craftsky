# Ofcom Control Applicability and Evidence Register

| Field | Value |
|---|---|
| Owner | Douglas Todd, accountable safety owner |
| Version | 0.1 |
| Reviewed | 9 November 2026 |
| Status | Initial illegal-content applicability mapping complete; evidence closure, children's mapping and whole-assessment approval pending |
| Publication | Internal only |
| Basis | Illegal-content assessment v0.4; confirmed launch images/video and Bluesky-backed DMs; likely child access |
| Next review | Before launch; material feature/risk/user-number change; relevant Code update; serious incident |

## Reading this register

This is the recommended-Code route assessment, not a declaration that CraftSky
has implemented every recommendation. An applicable row remains open until its
stated evidence and action are complete. No alternative measure is deemed
adopted merely because a feature or provider boundary differs from the Code.
Where an alternative is needed, record the relevant duty, protective effect,
freedom-of-expression/privacy safeguards, evidence and accountable-owner decision.

The source is the [issued U2U Illegal Content Code](https://www.ofcom.org.uk/siteassets/resources/documents/consultations/category-1-10-weeks/statement-protecting-children-from-harms-online/upcoming-codes-of-practice/issued-illegal-content-codes-of-practice-for-user-to-user-services.pdf?v=427438),
amendments in force 2 November 2026. Application clauses in section 4, rather
than the abbreviated index alone, were checked. All measure identifiers are
covered below, including blank/unused sections. References below are to those
clauses; the full recommendations and safeguards remain authoritative.

## Service predicates

| Predicate | Current conclusion and evidence | Review trigger |
|---|---|---|
| Large service | No on the recorded launch evidence: no active users and expected first-year users in the thousands. Code definition is more than 7 million monthly active UK users; forecast total users is not a measurement of that quantity. | Measure using Code section 5 methodology; reassess before any relevant threshold is approached. |
| More than 700,000 monthly active UK users | No on the recorded launch evidence. | Same; do not treat registrations or global totals as the defined UK count. |
| Multi-risk | Yes under ICU 5.6: at least two separate Table C categories are Medium/High, e.g. terrorism Medium and overall CSEA High. Do not count grooming/image/URL subtypes as three independent categories for this purpose. | Reassess with the overall illegal-content assessment. |
| Likely accessed by children | Yes, owner-approved access conclusion; 16+ self-declaration permits 16–17-year-olds and does not reliably exclude younger children. | Access/age-assurance or audience changes. |
| Risk-dependent conditions | Overall CSEA/grooming High; image/video CSAM and URLs separately Medium; other recorded categories Medium. | Subtype conditions use the subtype rating, not overall CSEA as a substitute. |
| Category 1 | Not established; do not claim designation or apply a Category 1 exception merely by platform type. Current launch size does not suggest the large-service threshold is met. | Verify categorisation at final launch review and after material growth/designation. |
| Principal pornography purpose / file-storage and file-sharing service | Not the described service: textile-craft social platform with posts, not a service principally for pornography or remote file storage/sharing. Media hosting alone is not this classification. | Material product-purpose or functionality change; keep definition under review. |
| Existing means to determine an individual user's age/age range | Unresolved across the full service/provider boundary. Reviewed onboarding stores only minimum-age confirmation; it does not distinguish a child from an adult. | Complete age-information inventory, including provider information actually available to CraftSky. Do not classify from arbitrary photos or collect DOB solely to simplify applicability. |
| Recommender testing | Not established. Chronological home ordering does not settle ranked discovery or popular content; search-only ordering is treated differently in the Code definition. | Inventory feed ranking and any real-user experiments before relying on an exclusion. |

## Illegal-content measures

“Applies” means the recommendation's service conditions are met on the recorded
scope. “Not triggered” is a dated applicability conclusion, not permission to
ignore the underlying safety duties. All rows are owned by Douglas Todd.

### Governance — ICU A

| ID / clause | Applicability | Existing evidence | Remaining action / gap |
|---|---|---|---|
| A1.1 | Not triggered: large-service condition not met. | Annual assessment review already contemplated. | Keep voluntary review; recheck size. |
| A2.1–3 | Applies to all services. | Named accountable owner in `governance/launch-approval.md` and `operator-details.md`. | Include the named responsibility in final approval; assignment is not proof of operational effectiveness. GOV-01. |
| A3.1–3 | Applies: multi-risk. | Sole owner and individual decisions recorded. | Consolidate a written responsibility statement covering risk, moderation, reporting, complaints, escalation, absence and review. Staffing is currently one person; do not invent a deputy. |
| A4.1 | Not triggered: requires large and multi-risk. | No independent audit asserted. | Do not present internal checking as independent assurance; recheck size. |
| A5.1–5 | Applies: multi-risk. | Append-only cases and assessment review triggers provide inputs. | Establish a recurring process and record for emerging harm, enforcement errors, source changes and likely illegal-content proxies. Reports alone are not prevalence or guilt. METRIC-01. |
| A6.1–2 | Applies: multi-risk. | Safe handling and disclosure instructions exist. | Consolidate operator conduct standards; cover confidentiality, impartial judgment, rights, escalation and conflicts. Owner approval and operational use pending. |
| A7.1–3 | Applies: multi-risk. | Research and owner-reported console exercises; no external training claimed. | Record role-appropriate learning, materials and competence checks for design/operation; resolve knowledge gaps. CSEA handling is not learned by viewing harmful test material. |

### Moderation — ICU C

| ID / clause | Applicability | Existing evidence | Remaining action / gap |
|---|---|---|---|
| C1.1–3 | Applies to all services. | Private reports, operator/external/confirmed-incident cases, human decisions and assessed reasons in `appview/internal/moderation/`; current 18-reason source taxonomy. | Map offence judgments and safe handling; verify hosted routes for public content and actual provider boundary for DMs. MOD-01, CSEA-01. |
| C2.1–5 | Applies to all services. | Visibility hide/takedown and appeals; source PDS remains user-owned. Local policy/API regressions pass. | Verify swift removal from every relevant CraftSky surface/cache/preview with benign fixtures. Neither federation nor provider outsourcing proves completion. MOD-02, II-01. |
| C3.1–4 | Applies: multi-risk. | Draft Guidelines, operator instructions and new reviewed illegal-content rationale route. | Complete offence/context guidance, policy consistency and update process; approve internal instructions and publish accurate user terms separately. POLICY-01/02. |
| C4.1–5 | Applies: multi-risk. | Five-business-day acknowledgement procedure and receipt-based intimate-image target; case timestamps. | Record and monitor moderation-action time and decision-accuracy targets. An acknowledgement target alone is insufficient; no fictional response results. REPORT-02, METRIC-01. |
| C5.1–2 | Applies: multi-risk. | Priority/specialist queues, urgent alerts in source, approved emergency procedure. | Consolidate priority order using exposure, severity, children and likelihood; test urgent handling. Alert delivery still awaits deployment. |
| C6.1–2 | Applies: multi-risk. | One operator, intended daytime/out-of-hours monitoring disclosed. | Assess achievable targets, demand surges, languages, sleep/absence and escalation; choose a workable continuity arrangement. Email alerts alone do not create cover. |
| C7.1–3 | Applies to the non-volunteer moderation role on this multi-risk service. | Operating materials, research and synthetic exercises. | Document training/competence and remedy gaps, including illegal-content judgments, lawful evidence handling and urgent escalation. |
| C8.1–3 | Service condition met; no volunteer moderator currently appointed. | Sole company operator; no volunteers reported. | Record as no current volunteer role to equip. Give materials and reassess before appointing any volunteer. |
| C9.1 | Not triggered on current Medium image-CSAM rating and small social-service scope. The other branches concern High image-CSAM with size/file-service conditions, or large services with Medium/High image-CSAM. | Existing image scanner is a partial additional control; full video coverage not established. | Do not infer a universal video scanner obligation from overall CSEA High; still assess proportionate detection and verify actual scanner scope/failures. CSEA-01/02. |
| C10.1 | Not triggered on current size/Medium URL-CSAM combination. | Text/profile/message links remain exposure paths. | Assess proportionate URL response independently; recheck size and subtype rating. |
| C11, C12, C13 | Intentionally blank in current issued Code. | No missing recommendation inferred. | Monitor amendments. |
| C14.1 | Not triggered on current Medium intimate-image rating and small specialist-social scope. | Specialist report workflow exists; internal 24-hour target approved. | This does not remove separate qualifying-report duties, actual matching-content response or receipt-based deadlines. II-01 remains open. |
| C15.1 | Not triggered: not large and none of its specified crisis-harm categories is High. The specified set is terrorism, hate, harassment/stalking/threats/abuse and foreign interference; overall CSEA High is not this trigger. | Emergency handling procedure exists. | Keep operational surge/escalation planning under C6; reassess if any specified rating becomes High. |
| C16.1 | Not triggered: large-service condition not met. | Existing authority-verification process/mailbox. | No dedicated crisis-law-enforcement channel claimed; underlying genuine requests/emergencies still require handling. |

### Reporting and complaints — ICU D

| ID / clause | Applicability | Existing evidence | Remaining action / gap |
|---|---|---|---|
| D1.1–2 | Applies to all services. | Signed-in/guest public-content reports, external mailbox and intake; owner-reported routine benign exercise passed. DM reports go only to Bluesky by owner decision. | Verify each relevant complaint type and affected-person/representative access. Assess, do not silently change, the DM responsibility/routing boundary. REPORT-01, CHILD-02. |
| D2.1–4 | Applies to all services. | Content report actions, email fallback, details field; existing native widget coverage. | Exercise native/web accessibility, screen readers, keyboard, young-user comprehension and unnecessary-step checks. A link's existence does not prove usability. |
| D3.1–2 | Applies: likely child access and Medium/High illegal risk. | Private report DTOs; public privacy statements exist in drafts. Reviewed `ReportSubjectSheet`/`ReportForm.svelte` lack a general pre-submit report-disclosure notice. | Add accurate easily accessed information on disclosure of the report's existence, content, identity and appeal disclosures, including exceptions. Optional-email helper text alone is insufficient. |
| D4.1–3 | Applies: Medium/High illegal risk. | Accepted API response has `reportId`/`status`; native success text is a generic thank-you. Manual acknowledgement target approved. | Brief process/ten-UK-business-day routine estimate copy is now implemented in native/web success confirmations under the owner's narrowed scope. Release pending; no status dashboard added. Full acknowledgement coverage and the opt-out exception remain to reconcile; the estimate is not a statutory deadline. |
| D5.1–2 | Applies: likely child access and Medium/High illegal risk. | Draft outcome descriptions; owner-reported manual email exercise. | Acknowledgements must explain possible outcomes and whether the complainant will receive the outcome/action information, safely. |
| D6.1–2 | Applies: Medium/High illegal risk. | Guest contact email optional; that alone is not a communication preference. | Provide and honor opt-out for non-ephemeral complaint communications across relevant paths. Do not add unapproved Brevo reporter emails. |
| D7.1–3 | Applies to all services. | Human review; allegation/decision separation; sanctions never inferred from counts. | Verify urgent queue/handling; complaints give reason to investigate, not guilt. |
| D8.1–4 | Applies: multi-risk. | One appeal lifecycle, confirmation/resolution, exact replay and reversal verified locally. | Set/monitor appeal decision-time and accuracy targets; prioritize serious sanctions and technology-error cases. No universal independent external reviewer requirement appears in this measure; maintain existing recusal/reconsideration policy separately. |
| D9.1 | Not the applicable branch: CraftSky is multi-risk. | D8 is mapped instead. | Do not rely on the smaller-service promptness-only branch. |
| D10.1–2 | Applies to all services. | Effect reversal/standing correction, owner-safe appeal history. | Verify hosted restoration and information to the appellant; no reason to delete source evidence or PDS records to reverse an action. |
| D11.1–4 | Applies to all services when relevant proactive-technology complaints arise. | Scanner review and human confirmation exist. | Document routing, investigation, outcome and resulting action information for non-appeal technology complaints; avoid disclosing sensitive hashes/evidence. |
| D12.1–5 | Applies to all services. | General email complaints and manual intake. | Assign responsibility, investigate other relevant compliance/privacy/expression complaints, take appropriate action and reply; do not misclassify these as only content reports. |
| D13.1–6 | Applies as a conditional permission, not a requirement to reject complaints. | No manifestly-unfounded rejection policy evidenced. | Do not use that exception pending an approved lawful policy with safeguards, error review and annual review. Abuse of a tool is not established by mere disagreement or volume. |
| D14.1 | Not triggered: large-service condition not met, despite Medium fraud. | Ordinary external intake remains available. | No mandatory dedicated trusted-fraud-flagger channel inferred for this launch scope. |

### Other controls — ICU E, F, G, H, J

| ID / clause | Applicability | Existing evidence | Remaining action / gap |
|---|---|---|---|
| E1.1–6 | Risk limb met; on-platform recommender-testing limb unresolved. | Chronological home feed; ranked discovery/search need classification. | Inventory actual ranking and experiments; if conducting covered real-user tests, log/compare safety metrics and retain the prescribed record. Do not exempt all discovery because home is chronological. |
| F1.1–6 | High-grooming limb met; existing individual age/age-range means unresolved. Each functionality has its own subcondition. | Contact/discovery review in `assessments/grooming-control-review.md`. | Inventory age information actually available and network prompts/lists/DMs/automated device-location display. Compare actual connected-account definitions and default behavior; a following fallback alone is not proof. No new age collection, classifier or forced provider setting authorized. |
| F2.1–6 | Same initial age/risk conditions as F1; support also depends on listed interactions. | Some provider-boundary explanation, blocks/reporting; no complete child-specific support evidence. | Assess risk information before weakening defaults, relevant account actions and support prompts. Resolve with F1 and children's review, preserving provider-only DM reports unless separately changed. |
| G1.1–3 | Applies to all services. | Draft Terms/Guidelines/reporting policy; policy publication gate. | Reconcile illegal-content protection, complaints, actual functionality and safeguards; approve/publish matching release. POLICY-01/02. |
| G2.1 | Not established as applicable: Category 1 condition not established. | No designation asserted. | Verify designation at final launch review. |
| G3.1–2 | Applies to all services. | Draft public policies and local HTML generation. | Child-appropriate clarity, accessibility and easy availability need actual review/publication evidence. |
| H1.1–10 | Applies to all services. | Terrorism allegations; `threat_or_violence` and reviewed illegal-content reasons; account takedown outputs and severe participation restriction are distinct controls. Existing profile/post/search read tests include hidden-account fixtures. | Document proscribed-organisation account assessment, evidence/explicit consent boundary for private content, accurate reason, proportionate removal from CraftSky and appeals. Verify combined account visibility/participation effects across surfaces. Do not call participation suspension alone account removal or delete a DID/PDS. |
| J1.1 | Not triggered: large-service condition not met. | Blocks/mutes already protect users and remain relevant to this risk assessment. | Verify existing controls and federation limits; children's J measures have different conditions. |
| J2.1 | Not triggered: large-service condition not met. | Reply/comment controls require their own inventory. | No blanket ICU duty to add comment disabling inferred here; children's J2 remains separate. |
| J3.1 | Not triggered: large-service condition not met. | Commercial/business labels exist. | Commercial disclosure and children's/privacy obligations remain separate; do not treat every badge as a verified identity. |
| B, I | Unused sections in this Code. | Age assurance is addressed in the children's Code and other duties. | Do not infer absence of age-related duties from empty ICU B. |

## Risk-to-response coverage

All rows also depend on the common reporting, review, visibility, appeal,
training and resource controls above. These are supported judgment routes, not
automatic legal classifications or permission to sanction from an allegation.
Use the specific offence guidance and context; record legal classification only
after assessment. `illegal_content` requires evidence, separate legal rationale,
`otherIllegalContent` classification and safe owner explanation; all severe
suspensions additionally require a severity rationale.

| Assessment category | Accurate source decision route to assess | Additional dependency / check |
|---|---|---|
| CSEA overall; grooming; image/video CSAM; CSAM URLs | `child_safety` for supported child-safety findings; assess each subtype separately. | NCA readiness, restricted evidence/holds, media/video scope, URL handling, contact protections and Bluesky-only DM report boundary. |
| Terrorism | `threat_or_violence` where accurate; `illegal_content` when no specific reason accurately fits. | Proscribed-organisation account assessment/removal (H1); genuine imminent danger/authority process. Do not label mere political or religious expression terrorism. |
| Hate | `hate`. | Protected-group/offence/context assessment; distinguish illegal hate from disagreement or mere offence. |
| Harassment, stalking, threats and abuse | `harassment`, `stalking_or_privacy`, `threat_or_violence` as facts fit. | Repeated pattern, evasion, victim safety and credible-threat workflow. |
| Controlling/coercive behaviour | Specific harassment/stalking/threat reason where accurate, otherwise supported `illegal_content`. | Pattern and relationship/offence context; safe communications and no unnecessary disclosure to alleged abuser. |
| Intimate-image abuse | `sexual_violation`. | Qualifying receipt, statutory response and same/substantially-same content work; internal 24h target is not a replacement legal deadline. |
| Extreme pornography | `adult_or_graphic` for accurate policy finding; `illegal_content` where the assessed offence needs the general route. | Offence/context, safe review and video/link exposure. Policy pornography prohibition is broader than illegality. |
| Adult sexual exploitation | `sexual_violation` where accurate, otherwise supported `illegal_content`. | Coercion/context, victim safety and appropriate authority process. |
| Human trafficking | Supported `illegal_content` unless a specific accurate exploitation/threat reason fits. | Advertisements/recruitment/facilitation, consent/coercion context and emergency escalation. |
| Unlawful immigration | Supported `illegal_content`. | Assess the actual facilitation offence; do not equate nationality, migration advice or lawful assistance with offending. |
| Fraud / financial services | `fraud_or_scam` or `impersonation` where accurate; supported `illegal_content` for a distinct offence. | Severe fraud eligibility now in source; ordinary disputes alone do not justify sanctions; no off-platform payment monitoring asserted. |
| Proceeds of crime | Supported `illegal_content`; fraud reason only where independently accurate. | Supported criminal-property/transaction context; no financial-surveillance capability inferred. |
| Drugs / psychoactive substances | Supported `illegal_content`. | Criminal promotion/supply context; distinguish lawful products, recovery and ordinary craft materials. |
| Firearms / knives / weapons | Supported `illegal_content`; threat reason for actual threats. | Offence and sale/possession context; ordinary craft cutters/scissors are not automatically illegal. |
| Suicide / serious self-harm encouragement or assistance | `self_harm` for supported harmful promotion/assistance. | Serious immediate risk response; severe eligibility now in source. Preserve support, recovery and help-seeking. |
| Foreign interference | Supported `illegal_content`. | Specific criminal/foreign-power nexus and actual evidence; foreign identity or political advocacy is insufficient. |
| Animal cruelty | Supported `illegal_content`; graphic reason for accurate separate graphic-policy findings. | Distinguish cruelty from lawful fibre sourcing, husbandry, welfare criticism and education. |
| Cyberflashing | `sexual_violation`; supported `illegal_content` if no specific reason fits. | Directed-media/offence context, victim safety, public directed content and actual DM payload/report boundary. |
| Other relevant illegal content | Supported `illegal_content` where no specific reason fits. | Identify the actual offence. Generic `other` remains ineligible for strikes/severe suspension; civil/IP disputes are not automatically OSA offences. |

## Children's Code boundary and next assessment work

Update, 9 November 2026: the detailed initial inventory of all 52 PCU application
clauses is now in `childrens-control-mapping.md`, using the 14 proposed content
ratings in children's assessment v0.2. It supersedes the earlier boundary-only
review below as to work completed. Ratings/policy decisions and conditional
implementation/effectiveness findings remain open; no whole-assessment approval
is inferred. The first concrete policy decision is the all-user harmful-content
boundary and contextual exception proposal.


The [issued Protection of Children U2U Code](https://www.ofcom.org.uk/siteassets/resources/documents/consultations/category-1-10-weeks/statement-protecting-children-from-harms-online/upcoming-codes-of-practice/issued-protection-of-children-code-of-practice-for-user-to-user-services.pdf?v=427439)
was checked in the same review (in force 2 November 2026). Its application
clauses were inventoried; this is not a completed PCU implementation mapping.

- PCU A2, C1/C2, D1/D2/D7/D10/D13, G1/G3 apply on the likely-child-access
  conclusion; accountability, review, complaints, restoration and clear terms
  require evidence for children, not only adults.
- PCU A3/A5/A6/A7, C3–C8, D8/D11 and F1 have their own large/multi-risk
  (children) conditions. ICU multi-risk is not a substitute: PCU 5.5 uses two
  or more specific kinds of content harmful to children. Those ratings remain
  to be completed in `assessments/childrens-risk-assessment.md`.
- PCU D3–D6 and E1/E2 depend on specific children's risk ratings; recommender
  measures also depend on actual child-accessible content ranking. Do not carry
  the overall CSEA rating across as a rating for pornography, self-harm,
  bullying or other harmful-content kinds.
- PCU B1–B7 require explicit age-assurance/application analysis. Owner-approved
  prohibition of primary-priority content is relevant but not sufficient by
  itself: evaluate the Code's prohibited-content and technical-takedown
  conditions, actual publication/enforcement and feed/provider boundaries.
  Public PDS persistence does not itself mean CraftSky cannot restrict its own
  surfaces. Do not conclude that age assurance is automatically required or
  automatically unnecessary from the current record.
- PCU J1/J2 can apply at High risk even to a small service; their Medium branches
  use the separate 700,000-UK-user condition. Existing ICU large-service exclusions
  must not be reused. PCU J3 depends on group messaging; current confirmed scope
  is one-to-one DMs, so confirm actual exposed functionality before exclusion.
- PCU A1/A4, C12, E3 and F2 are large-service branches not triggered on current size;
  C11 depends on its own specified children's harm ratings. F3–F5, D9/D12,
  D14 and other conditional branches still need individual assessment against
  the completed children's ratings, features and age-treatment approach.

No PCU row is declared compliant by this boundary review. CHILD-02 and privacy/
Children's Code work remain open. An initial common reporting-information gap
can be addressed now without inventing children's ratings or collecting ages.

## Ordered closure work

1. **Reporting information and acknowledgement:** owner authorized brief post-submit
   process/estimate copy, implemented locally, and declined reporter-status UI.
   See `../../docs/changes/2026-11-09-reporting-information/proposal.md`. This
   partial step does not close remaining D3–D6/D8 information, communication and
   monitoring work; no status dashboard is presumed necessary. The earlier
   wider implementation proposal is not approved.
2. **Governance, competence and continuity:** consolidate A3/A5/A6/A7 and C4–C7
   into an achievable one-person plan, including measured targets and safe
   absence/surge response. Do not prescribe a fixed headcount or manufacture
   training completion. Current monitoring intention is not staffed 24h cover.
3. **Age/contact and children's assessment:** complete actual age-information,
   ranking and functionality inventory, specific children's ratings and PCU
   mapping; return any required product decision to the owner.
4. **Specialist readiness / release evidence:** maintain NCA application status;
   finish approved storage/holds, Brevo delivery, intimate-image receipt/action,
   visibility/appeal and client checks after merge/deployment as already deferred.
5. **Policy and final approval:** reconcile/publish accurate approved policies,
   review alternatives where needed and obtain approval for the identified release
   and the completed assessments. No row is closed merely by owner confidence.

## Evidence and change history

This turn performed document/source review against current official Codes;
it made no application change, sent no report/message and changed no production
configuration. Previously passed tests and owner-run exercises are attributed
in their linked records; they were not repeated solely for this mapping.

| Version | Date | Change | Approval |
|---|---|---|---|
| 0.1 | 9 November 2026 | Initial complete ICU identifier/applicability inventory, source-evidence gaps, risk-to-response mapping and PCU dependency boundary. | Prepared for owner review; no whole-assessment or alternative-measure approval |
