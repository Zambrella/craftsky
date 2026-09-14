# Illegal Content Risk Assessment

| Field | Value |
|---|---|
| Assessment owner | Douglas Todd, Founder |
| Approved by | Not approved |
| Version | 0.1 |
| Status | Incomplete draft; not yet a suitable and sufficient statutory assessment |
| Assessment date | 11 September 2026 |
| Next review | Before launch and before any significant change |
| Publication | Internal regulatory record |

## Purpose and Limitation

This document starts CraftSky's Online Safety Act illegal-content risk
assessment. It records the service design, initial risk hypotheses, controls, and
evidence gaps. It is deliberately marked incomplete because CraftSky has not yet:

- assigned an accountable safety owner;
- completed Ofcom's current risk-profile and Code applicability worksheets;
- gathered launch-scale incident, user, moderation, or prevalence evidence;
- approved likelihood, impact, and residual-risk ratings; or
- implemented several controls needed for serious and time-critical harms.

It must not be represented to Ofcom, users, investors, app stores, or other third
parties as CraftSky's completed statutory assessment.

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

- No direct messages, group chat, livestreaming, disappearing content, or user
  location sharing.
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
| Grooming | An adult builds contact with a child through follows, replies, mentions, and linked services. | Low-to-medium likelihood if children join; severe impact | Public-only contact, block/report tools; no DMs | No age signal, grooming reason, pattern detection, or child-specific controls. |
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
| Cyberflashing | Unsolicited sexual image is directed at a user through a mention, reply, quote, or link. | Low likelihood because there are no DMs; severe impact | Adult/graphic reporting, blocks, takedown | No cyberflashing reason or expedited/survivor-sensitive handling. |

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

## Required Measures Before Approval

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

## Change History

| Version | Date | Change | Author/approver |
|---|---|---|---|
| 0.1 | 11 September 2026 | Initial risk register, control inventory, and completion actions. | Working draft / not approved |
