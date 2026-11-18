# Children's Risk Assessment

| Field | Value |
|---|---|
| Assessment owner | Douglas Todd, Founder |
| Approved by | Content boundary approved by Douglas Todd, 9 November 2026; ratings and whole assessment not approved |
| Version | 0.3 |
| Status | Incomplete working draft; not a completed statutory assessment |
| Assessment started | 9 November 2026 |
| Review | Before approval and significant service changes; after incidents and relevant Ofcom updates |
| Publication | Internal regulatory record |

## Starting Position

Douglas Todd approved the [children's access assessment](childrens-access-assessment.md)
on 9 November 2026. CraftSky is treated as likely to be accessed by children.
The contractual minimum age is 16, with a self-declaration rather than verified
ages. Assess risks to 16–17-year-olds and younger children who may access despite
the contractual rule. Do not infer exclusion from the rule alone.

The owner expects thousands of active users in year one, is the sole company
operator, and intends to monitor safety reports throughout the day and outside
UK business hours. Backup and staffed overnight cover are not arranged. Brevo
alerts are implemented locally but not activated or delivery-verified.

## Confirmed Launch Messaging Scope

Launch media update, 9 November 2026: Douglas Todd confirmed video uploads are
included at launch. Include video publication, playback and links in the content
and exposure assessments. This confirmation does not establish live video
moderation, scanner coverage or provider response effectiveness.

Current source includes Bluesky direct messaging through
`appview/internal/app/deps_chat.go` and `appview/internal/routes/routes_chat.go`.
Douglas Todd confirmed on 9 November 2026 that DMs are included at launch and
use Bluesky's service. The old service-scope draft's no-DM assertion cannot be
used as a protective measure.

Source inspection establishes the following boundaries:

- `appview/internal/chat/declaration.go` exposes everyone/following/nobody
  incoming-contact settings. If no declaration record exists, the read fallback
  is following. An existing user's setting is read and respected; this does not
  prove every launch account uses that default or that the fallback was persisted.
- `appview/internal/chat/report_service.go` sends message and conversation reports
  to Bluesky via `com.atproto.moderation.createReport`. The transport pins that
  operation to the configured Bluesky report audience.
- `appview/internal/chat/report_integration_test.go` explicitly checks that these
  reports do not create CraftSky moderation reports. Consequently the new Brevo
  alert worker does not notify Douglas of DM reports through that path.
- CraftSky provides the client and AppView integration and must assess the
  contact/reporting controls it exposes. Bluesky provides the underlying message
  service and receives those reports. Do not claim this division establishes
  either provider's complete legal responsibility or proves Bluesky escalation,
  outcomes, response times or external reporting on CraftSky's behalf.

Douglas Todd explicitly confirmed on 9 November 2026 that users should not
have a separate CraftSky escalation route for DM-related concerns. Preserve
Bluesky-only DM reporting: no duplicate CraftSky report, automatic local case,
transcript copying or Brevo alert is added for those submissions. This closes
the product-scope question, not CHILD-02 or the legal/control assessment.

Follow-up contact/discovery review on 9 November 2026 is recorded in
`online-safety/assessments/grooming-control-review.md`. Requests expose text
before acceptance while rich previews are gated; acceptance controls replies.
Existing incoming settings are respected rather than universally narrowed.
No child label/age-filtered discovery was identified in the reviewed profile
paths. Focused local backend checks passed; live provider/native/web checks and
age-dependent Code applicability remain outstanding.

Assess the existing contact settings, blocking/reporting experience and provider
dependencies within that scope. Any finding that the chosen arrangement fails
an applicable requirement must be documented with supporting evidence and
brought back to the owner, rather than silently adding the rejected flow or
assuming that the owner's scope decision establishes legal compliance.

## Approved Public-Content Approach — 9 November 2026

Douglas Todd explicitly approved prohibiting pornography, sexual exploitation,
and encouragement or instructions for suicide, serious self-harm or dangerous
eating-disorder behavior for all users. Legitimate craft, health, educational
and recovery discussion remains allowed where it does not promote harm.

This approves that content-policy approach only. It does not approve the full
Guidelines, establish publication, complete the children's risk assessment or
prove the effectiveness of enforcement.

Repository enforcement evidence:

- `appview/internal/moderation/taxonomy.go` maps child-safety, sexual/intimate,
  threat/violence and self-harm allegations to named decision reasons while
  keeping initial legal classifications unassessed.
- `appview/internal/moderation/policy.go` supports visibility takedown, strikes,
  and explicitly justified severe suspension for approved serious reasons.
  A report does not itself impose those consequences.
- `appview/internal/api/moderation_policy.go` makes hide/takedown dominate warnings
  in the read-policy calculation. These controls remove CraftSky visibility,
  not records from the user's PDS.
- These code controls support action after identification. Their presence does
  not demonstrate detection effectiveness, prompt response, production scanner
  readiness, all-client exposure prevention, or fulfilled Code measures.

## Initial Risk Pathways to Assess

On 9 November 2026 Douglas Todd agreed to the High grooming design-risk rating
in the illegal-content assessment. Carry that finding into this assessment;
it does not by itself complete the children's risk ratings or controls. The
initial ICU F1/F2 review records an unresolved age-information applicability
question: current onboarding only stores minimum-age acceptance, not an
under-18/adult distinction. Keep the approved Bluesky-only report route while
assessing existing contact settings and the applicable safeguards.

Douglas Todd also agreed on 9 November 2026 to separate Medium initial working
ratings for image-based CSAM (including video) and CSAM URLs in the
illegal-content assessment. Carry those findings forward without treating them
as final children's risk ratings, proven controls or accepted remaining risk.

These are review topics, not approved ratings or findings about actual incidents.

| Pathway | Evidence and question to resolve |
|---|---|
| Sexual/explicit and exploitative content | Owner approved prohibition for all users; draft Guidelines contain it. Verify publication, enforcement, exposure controls and the treatment of legitimate craft/health imagery. |
| Suicide, self-harm and eating-disorder content | Evaluate harmful encouragement/instruction separately from supportive discussion; examine body-image risks around garment fit and appearance. |
| Bullying, hate, abuse and violent content | Consider posts, replies, mentions, profiles, links and any launch messaging. Verify reporting, blocks, mutes and sanctions on both clients. |
| Grooming and unwanted contact | Assess adults contacting minors, privacy boundaries, contact defaults, external links and moving conversations off-platform. Launch DMs are confirmed; assess the documented Bluesky/CraftSky boundary. |
| Dangerous challenges and substances | Consider unsafe craft tools, techniques and linked tutorials, distinguishing ordinary craft instruction from harmful content. |
| Discovery and repeated exposure | Examine chronological feeds, search, reposts, quotes and sharing; chronology does not eliminate harmful-content exposure. |
| Privacy and commercial pressure | Consider identifiable photos of children, profile disclosures, public federation, business content, scams and links. Children's Code/DPIA review remains separate. |
| Reporting and response failures | Verify child-appropriate reporting and complaints, urgent routing, time-critical handling, operator absence and external escalation. |

## Service-Specific Risk Review — 9 November 2026

The following is a prepared assessment proposal, not a record of additional owner
approval. Previously agreed illegal-content/CSEA ratings remain unchanged. No
child incident, measured age distribution or real-user prevalence is inferred.
The owner-authorized post-report copy and no-status-dashboard scope are preserved.

### Evidence and method

Reviewed Ofcom's current [Children's Risk Assessment Guidance and Risk Profiles](https://www.ofcom.org.uk/siteassets/resources/documents/consultations/category-1-10-weeks/statement-protecting-children-from-harms-online/main-document/childrens-risk-assessment-guidance-and-childrens-risk-profiles.pdf?v=396653),
[content-harm guidance](https://www.ofcom.org.uk/siteassets/resources/documents/consultations/category-1-10-weeks/statement-protecting-children-from-harms-online/main-document/guidance-on-content-harmful-to-children.pdf?v=395445)
and [issued children's U2U Code](https://www.ofcom.org.uk/siteassets/resources/documents/consultations/category-1-10-weeks/statement-protecting-children-from-harms-online/upcoming-codes-of-practice/issued-protection-of-children-code-of-practice-for-user-to-user-services.pdf?v=427439)
on 9 November 2026. The Code amendments are in force from 2 November 2026;
the risk-guidance index lists 9 May 2025 while the PDF cover states publication
24 April 2025. This record uses that currently linked version rather than
inventing a newer risk-guidance edition.

Assess each of four primary-priority, eight priority and two Ofcom-identified
non-designated content kinds separately, using likelihood, impact, age and
service features. The risk level is a qualitative conclusion; no numerical
average or fabricated probability is used. Table 10 advises caution where the
evidence is inconclusive and judgment rather than mechanical classification.

Core evidence considered: approved access assessment; actual product/source;
current operating/approval records; relevant illegal-content assessment;
available synthetic/local checks; provider/reporting boundaries. No user-age,
incident-frequency or moderation-prevalence dataset was available for this
pre-launch review. The owner does not consider the specialist craft community
targeted at or particularly appealing to children. Expected users in the
thousands, no ads/paid reach, craft focus, chronological home ordering and the
available controls mitigate expected exposure; they do not establish that harmful
content cannot appear or that the admitted 16–17-year-olds are adults.

Source inspection narrows the following feature conclusions:

- `api/timeline_store.go` orders the followed home timeline by activity chronology.
  `api/search_project_store.go` also ranks empty-query project browsing by
  likes/replies/reposts and time when Popular is selected. `GET /v1/projects`
  is distinct from query-driven project search. Thus an exemption for systems
  used exclusively to answer a search query cannot be assumed for all browsing.
  Classification against the Code definition and live presentation still need
  review. No ranking or experiment was changed.
- `chat/report.go:directPeer` rejects conversations with other than two members
  and group-kind conversations. `chat/send.go` constructs a text message input;
  do not infer native DM image/video uploading from public-post uploads. Links
  and private text still carry harmful-content/contact risks. Incoming messages
  from independent clients/providers need their own display/enforcement review.
- Native `OnboardingGuidelinesStep` records minimum-age acceptance, not an
  under-18/adult classification or HEAA. No new age data collection is proposed
  by this assessment.
- Existing visibility outputs, case review, blocks/mutes and report handling are
  partial controls with separately attributed local tests. The video-enabled
  release, all-surface removal, production scanner/provider performance and
  operator continuity remain unverified.

### Age-Group Consideration

These are assessment considerations, not claims about actual CraftSky users.
No age band is presumed technically excluded by the self-declared 16+ rule.

| Group | CraftSky-specific exposure and impact considerations | Safeguard consequence |
|---|---|---|
| 0–5 | Incidental viewing on a family device or a shared public craft link is plausible even without an account. Very young children may have difficulty interpreting distressing media or instructions. | Public/guest views matter; a registration declaration alone cannot protect them. Avoid adult-only assumptions for public previews. |
| 6–9 | Craft tutorials and garment/animal imagery can be attractive; limited experience can increase confusion about unsafe instructions and distressing images. | Clear explanations and contextual review; appropriate protection from explicit/graphic content and unsafe encouragement. |
| 10–12 | Growing independence and peer contact can expose children to appearance-based bullying, exclusion and harmful imitation despite the contractual rule. | Reporting/blocking must be usable; assess cumulative contact and public replies, not just isolated posts. |
| 13–15 | Body comparison, peer pressure and online relationships can increase the effects of eating-disorder/self-harm promotion, body stigma and targeted abuse. | Distinguish support/healthy craft discussion from encouragement or stigmatization; credible contact and urgent response controls matter. |
| 16–17 | Deliberately admitted users may post identifiable garment/fit photos, participate independently and contact adults/businesses. More autonomy is not adult status or immunity from harm. | Preserve expressive/creative benefits while protecting against harmful content, private coercion, body stigma and repeated abuse. |

Across groups consider disability, protected characteristics, mental-health
vulnerability, prior victimization, language/media literacy, socioeconomic access
to support and children pictured who are not account holders. Do not infer a
condition from an image, profile or search term, or collect sensitive information
merely to document a lower rating.

### Recommended Working Content Ratings

**Each row is a separate recommendation, not an owner-approved final rating.**
Confidence is limited. Medium reflects credible exposure pathways and partial
controls without service-specific evidence sufficient to conclude very likely or
systematic exposure; it does not mean that individual harm is only moderate.
Severe/fatal outcomes remain possible. No row is capped at Medium: targeting,
vulnerable-user exposure, repeated promotion, rising incidence or failed response
could justify High on review. A policy ban alone does not prove Low, and lack of
incident data alone does not establish High. Each conclusion must be revisited
against demonstrated controls and final policy scope before whole-assessment
approval. The common level does not collapse the separate assessments below.

| ID | Kind / class | Proposed level | Likelihood and CraftSky pathway | Impact, context and control limits |
|---|---|---|---|---|
| CH-P1 | Pornography / primary priority | Medium | Public image/video uploads, profile/preview media, links and incoming messages permit exposure. No adult-service purpose or child-targeted marketing is evidenced; ordinary craft/lingerie imagery is not automatically pornography. | Sexual distress/exploitation and inappropriate exposure may be severe, especially for younger children. Owner approved all-user prohibition; no comprehensive deployed detection or all-surface removal evidence. |
| CH-P2 | Suicide encouragement/promotion/instructions / primary priority | Medium | Text, videos, links and targeted private contact could encourage suicide; body/health/recovery discussion can be misused. No observed promotion network or incidence is asserted. | Potentially fatal impact, including for a small number of vulnerable children. Prohibition, self-harm reports and emergency process exist; urgent alert/cover and provider efficacy remain unproven. Legitimate help-seeking is protected. |
| CH-P3 | Deliberate self-injury encouragement/promotion/instructions / primary priority | Medium | Image/video/text and links can promote injury, including misuse of accessible craft tools. Ordinary safe tool instruction is not self-injury encouragement. | Serious injury and reinforcement of harmful behaviour; targeted exposure is particularly concerning. Existing policy/response are partial; full video/contextual detection and timely action are unverified. |
| CH-P4 | Eating-disorder encouragement/promotion/instructions / primary priority | Medium | Garment sizing, body/fit posts, transformation imagery and replies can be repurposed into restrictive-eating promotion; private links can reinforce it. No actual incidence or child body-image dataset exists. | Potentially serious physical/psychological harm, particularly for vulnerable teenagers. Owner approved prohibition and body-shaming rule; separate detection/routing/context exercise remains needed. Neutral size/fit facts, recovery and health discussion are not automatically harmful promotion. |
| CH-R1 | Abusive content targeting specified protected characteristics / priority | Medium | Identifiable posts/profiles, comments/quotes and DMs permit identity-based attacks. Inclusive craft context does not prevent targeted abuse; no sustained pattern is established. | Distress, exclusion and compounding vulnerability. Harassment/hate controls exist but privacy, contextual judgment, evasion and production coverage need verification. Mere disagreement is not the harmful-content test. |
| CH-R2 | Content inciting hatred against specified protected groups / priority | Medium | Public text/video/links and reposting can distribute hateful incitement, including outside the craft topic. No observed network or high-volume dissemination is established. | Fear, dehumanization and potential offline harm. Hate policy and sanctions exist; bans alone do not establish prevention. This assessment is distinct from the criminal hate threshold. |
| CH-R3 | Encouragement/promotion/instructions for serious violence against a person / priority | Medium | Threatening replies/DMs and text/video instructions could facilitate violence; blades/tools are available in legitimate craft content but not themselves violent instructions. | Potentially severe injury/death and intimidation. Violence/threat review and emergency workflow exist; practical instruction/context and private-provider handling need evidence. |
| CH-R4 | Bullying / priority | Medium | Garment/appearance criticism, pile-ons through replies/quotes, repeated mentions and peer exclusion provide concrete pathways. No measured child-use/abuse pattern establishes very likely exposure. | Serious distress, withdrawal and cumulative mental-health effects; sustained targeted bullying may warrant High. Blocks/mutes and human sanctions help but do not eliminate alternate-account/private abuse or prove prompt response. |
| CH-R5 | Real/realistic serious violence or graphic serious injury of a person / priority | Medium | Photos, launch videos, thumbnails and linked previews can depict serious injury, including material offered as medical/educational context. The broad draft exception needs reconciliation. | Distress/trauma, greater difficulty contextualizing for younger viewers. The harm definition and context must be assessed; an adult warning or asserted educational purpose alone is not a child-exposure safeguard. No verified age filter exists. |
| CH-R6 | Real/realistic serious violence or graphic serious injury of animals, including relevant fictional-creature depictions / priority | Medium | Fibre sourcing, husbandry/welfare discussion, media and links can expose children to violent or graphic injury content; ordinary shearing and lawful husbandry are not automatically this content. | Distress and normalization of cruelty; younger children may struggle with realistic fictional depictions. Graphic/cruelty rules provide partial action; public educational exceptions, video and previews require review. |
| CH-R7 | Dangerous challenges/stunts highly likely to cause serious injury / priority | Medium | Unsafe tool/heat/dye demonstrations and social imitation can become a dangerous challenge when encouraged, filmed or linked. CraftSky has no known dedicated challenge mechanic; general UGC/contact/reposting still permits it. | Serious burns, poisoning or other injury. Assess the encouragement/challenge and serious-injury threshold, not any technique with ordinary manageable risk. Current general danger rule lacks a fully explicit category/handling map. |
| CH-R8 | Encouraging harmful ingestion/injection/inhalation/self-administration of substances / priority | Medium | Dyes, solvents, cleaning chemicals and linked health claims could be misrepresented as safe to consume or deliberately inhaled; routine safe material use is not this category. | Serious poisoning or death. General dangerous-content handling exists; explicit policy coverage/context guidance and timely removal need evidence. Ordinary material warnings and safe tutorials remain valuable. |
| CH-N1 | Body stigma / Ofcom-identified non-designated | Medium | Garment fit/sizing and modelling make appearance criticism a direct contextual risk; posts/comments could stigmatize bodies even without targeting one identifiable child. No service incidence established. | Shame, exclusion, unhealthy body image and interaction with eating-disorder risk, especially for vulnerable teenagers. Draft body-shaming prohibition exists; reporting/contextual handling and repeated-exposure controls need verification. Neutral measurements, inclusive fit discussion and legitimate health context are distinct. |
| CH-N2 | Promotion of depression, hopelessness and despair / Ofcom-identified non-designated | Medium | Text/video/links and private interactions may glorify hopelessness or discourage seeking help; grief, illness and recovery discussion are plausible craft-community topics. No actual incidence established. | May worsen vulnerability, isolation or loss of help-seeking. Explicit promotion prohibition now approved and aligned in the draft Guidelines; publication and contextual enforcement remain pending. Do not treat honest sadness, depression disclosure, grief or seeking support as prohibited promotion. |

### Other Potential Non-Designated Content

Unsafe craft misinformation was considered. Deliberate dangerous challenges or
encouraged harmful self-administration are covered above; ordinary accidents or
low-quality instructions do not automatically establish a further statutory
non-designated kind. Privacy disclosures, commercial pressure and grooming remain
important design/privacy/illegal-content risks, not automatic extra content kinds
under this test. No additional distinct kind meeting the material-risk/significant-
harm threshold is established by the current evidence. Keep that conclusion
under review, including products/instructions likely to cause serious injury.

Ofcom guidance Part 3 section 2.10 and its footnote 97 address notification for
non-designated-content findings, including the relationship with a non-negligible
rating. Before finalizing CH-N1/CH-N2, record the section 11(5) notification
analysis and prepare any required notification with truthful incidence/evidence
limits. Do not invent incidents or assume zero current users creates an
exemption. No notification was sent or authorized in this review.

### Policy and Safeguard Decisions Exposed by This Review

1. **Content boundary — decided:** Douglas Todd approved the full all-user
   harmful-content boundary and context safeguards on 9 November 2026. Guidelines,
   Terms/reporting references and operator instructions are aligned locally;
   see `../../docs/changes/2026-11-09-childrens-content-boundary/proposal.md`.
   This removes the draft policy's blanket graphic-context exception. It does
   not establish effective publication or control of every child-accessible
   surface, rule out age-assurance conditions or approve the draft ratings.
2. **Detection/takedown coverage:** establish effective action on CraftSky public,
   guest, discovery, media/preview and other child-accessible surfaces; review DM
   provider removal/control capability separately. A surviving public PDS copy
   does not by itself prove that CraftSky cannot remove its own presentation,
   while a local case outcome does not prove removal of a private provider message.
3. **Child support:** on the recommended Medium findings, PCU F3 and F5 require
   assessment for prompt support signposting after reports and relevant searches,
   treating users without HEAA adult determination as relevant users. Small size
   does not remove these conditions. The new generic receipt is not that support.
   The owner subsequently approved the narrow guide/report/search support
   proposal on 9 November 2026. Local implementation and validation are recorded
   in `../../docs/changes/2026-11-09-child-safety-support/validation.md`; deployed
   verification, language/context coverage and provider boundaries remain open.
   No status dashboard was introduced.
4. **Child materials and operation:** multi-risk (children) would follow from at
   least two of the proposed content ratings, independently of ICU multi-risk.
   Public child/parent materials are now implemented locally in the approved
   support guide; deployed and youngest-user usability evidence remains pending.
   Complete governance,
   resourcing, training, moderation/appeal targets and evidence. Never label
   research/exercises as formal completed training or continuous staff cover.
5. **Ranking and age information:** complete content-recommender classification
   for popular project browsing and actual age-information availability; assess
   age assurance and targeted/whole-service protections against the final policy
   and technical facts. No product pivot, DOB collection or provider setting
   change is authorized by the draft risk recommendations.

Detailed PCU applicability and open evidence are in
`../registers/childrens-control-mapping.md`. No application changes or new live
exercise occurred during this document/source review. Previously attributed
checks remain local or owner-reported as recorded in their original evidence.

## Completion Work

- Confirm remaining enabled launch features and relevant service/user evidence;
  Bluesky-backed DMs are confirmed.
- Consult current Ofcom children's Risk Profiles and complete risk judgments by
  content type and affected child age groups, with evidence and uncertainties.
- Map each applicable Protection of Children Code measure or adopted lawful
  alternative to the assessed risks, implementation, tests and review owner.
- Verify actual safeguards and policy consistency; code presence is not live
  operational evidence.
- Record residual risks, unresolved controls, effectiveness monitoring and the
  accountable owner's approval. Access-assessment approval does not supply it.

## Official Sources

- [Ofcom protection of children duties](https://www.ofcom.org.uk/online-safety/protecting-children/protection-of-children-duties-under-the-online-safety-act)
- [Ofcom regulatory documents and guidance](https://www.ofcom.org.uk/online-safety/illegal-and-harmful-content/online-safety-regulatory-documents)

The overview guidance was consulted on 9 November 2026. This draft does not claim
completion of the underlying current risk-profile or Code applicability review.

## Change History

| Version | Date | Change | Approval |
|---|---|---|---|
| 0.1 | 9 November 2026 | Starts assessment from approved access conclusion and current source; records launch-scope and evidence gaps. | Not approved |
| 0.2 | 9 November 2026 | Adds age-group analysis, 14 separate proposed content ratings, additional-content review, source-backed ranking/chat scope and PCU evidence dependencies. | Prepared for owner review; ratings/whole assessment not approved |
| 0.3 | 9 November 2026 | Records approved full harmful-content boundary and aligned draft policies/operator notes; publication and control effectiveness remain separate. | Content boundary approved; ratings/whole assessment not approved |


## Current-Code Mapping Dependency — 9 November 2026

`../registers/ofcom-control-mapping.md` records the illegal-content Code inventory
and the separate PCU dependency boundary against both Codes in force
2 November 2026. ICU multi-risk does not establish multi-risk (children): the
latter uses two or more specific harmful-content ratings under PCU 5.5. Those
ratings and the resulting PCU implementation/effectiveness mapping remain open.
The common reporting-information proposal can proceed independently if approved;
it neither settles age assurance nor authorizes a duplicate DM report route.
