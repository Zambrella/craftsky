# Launch Approval Record

| Field | Value |
|---|---|
| Accountable owner | Douglas Todd, Founder |
| Last updated | 9 November 2026 |
| Status | Owner-led review requested; final release approval pending |
| Publication | Internal |

## Review Route

On 9 November 2026 Douglas Todd confirmed that no solicitor or external
online-safety/privacy specialist has reviewed the policies or risk assessments.
He stated that research has been undertaken and that he feels confident the
final release can be approved. Record this as a request to use owner-led review,
not as evidence that external specialist review occurred or as completed approval
of an identified release candidate and all unresolved controls.

External specialist sign-off was an internal launch-governance condition in
GOV-01. The owner-led route will document the research relied on, findings,
uncertainties, controls, and accepted residual risks for the owner's final
approval. Any specifically required legal consultation or regulatory process
must still be assessed separately; changing this governance route does not waive
legal duties.

[Ofcom's illegal-content duties guidance](https://www.ofcom.org.uk/online-safety/illegal-and-harmful-content/illegal-content-duties-under-the-online-safety-act)
was consulted on 9 November 2026. It provides for small-service accountability
through a responsible senior manager and requires evidence-based, up-to-date
risk assessments and records. Owner approval must cover the completed assessment
and controls rather than stand in for them.

## Final Decision Still Required

Return to GOV-01 after the individual launch controls are reconciled. Record:

- the exact release candidate and enabled features;
- approved risk assessments, policies, and source references;
- tested production and operational safeguards;
- remaining uncertainties and any lawful alternatives;
- the owner's decision, date, and accepted residual risks; and
- any required deployment and publication verification.

No overall release approval, deployment, publication, or external specialist
approval is recorded by this entry. GOV-02 approval remains separately recorded.

## Report Monitoring — Owner Statement

On 9 November 2026 Douglas Todd confirmed that there is no current out-of-hours
arrangement or backup cover. He intends to monitor safety reports throughout the
day, including outside business hours. This is owner monitoring while available,
not a staffed 24-hour service or guaranteed response time. Absence, sleep and
unavailability remain operational gaps to resolve in the risk assessment.

Douglas requested email alerts from the moderation platform for urgent safety
reports. Repository inspection found safety work priorities and dashboard alert
flags but no existing AppView outbound transactional-email sender. Douglas then
selected the Brevo API, sending from and to `moderation@craftsky.social`, whose
notifications will reach him away from the office. The implementation and local
verification are recorded in the
[operator runbook](../../docs/operations/moderation-operators.md#urgent-safety-email-alerts-brevo).
Production configuration, deployment, mailbox receipt and phone notification
remain unverified; this safeguard is not yet recorded as operational.

The proposed alert covers newly submitted urgent user/guest reports as well as
relevant detected incidents; triggering must not depend on a moderator first
opening or manually classifying the item. Use generic alert text and a link to
the authenticated moderation console, with no report narrative, personal
identifiers, images, attachments or restricted evidence in the email. Persist
pending deliveries, retry failures and suppress duplicate alert storms. Delivery
and urgent-category coverage must be tested before claiming this safeguard is
operational. The feature does not supply backup human cover.

## CSEA Reporting Registration

On 9 November 2026 Douglas Todd confirmed that NCA organization registration or
an equivalent NCMEC reporting arrangement is not yet established. CSEA-01 stays
open. The current NCA registration steps and required information are recorded
in the restricted response runbook. No external application or report has been
submitted by this review.

Update, 9 November 2026: Douglas Todd reports that he submitted the NCA
organization registration application. Record the status as **submitted /
awaiting NCA approval**. Approval, functioning portal access and operational
readiness remain unverified. The owner submitted the application; the assistant
did not submit it. CSEA-01 remains open for those remaining controls.

## Children's Access Decision

Douglas Todd explicitly approved the conclusion that CraftSky is likely to be
accessed by children on 9 November 2026. CHILD-01 is closed using the approved
access assessment. This is not approval of the children's risk assessment,
child-safety implementation, policies, or overall release. CHILD-02 is being
prepared separately against the actual launch feature set.

## Launch Messaging Scope

On 9 November 2026 Douglas Todd confirmed that launch includes DMs through
Bluesky's service. The children's risk assessment now records that scope, the
Bluesky-only message/conversation report route and the lack of local Brevo
notification for those reports. No automatic copying of private messages or
change to the approved reporting destination is authorized by this confirmation.

Update, 9 November 2026: Douglas Todd explicitly rejected adding a separate
CraftSky escalation route for DM-related safety concerns. Keep DM reports with
Bluesky and preserve the existing reporting destination; do not add duplicate
local reports, transcript copying or DM-triggered Brevo alerts. Record this as
a product decision, with the children's risk/control assessment still open.

## Public-Content Policy Approach

On 9 November 2026 Douglas Todd explicitly approved the public-content approach:
prohibit pornography, sexual exploitation and encouragement or instructions for
suicide, serious self-harm or dangerous eating-disorder behavior for everyone;
allow legitimate craft, health, educational and recovery discussion where it
does not promote harm. This decision covers the specified content boundaries
only. Full Guidelines/publication approval, enforcement verification and
CHILD-02 remain separate outstanding controls.

## External Reporting Mailbox

On 9 November 2026 Douglas Todd confirmed that ordinary external email reaches
`moderation@craftsky.social` and that replies work from that address. Record this
as owner-reported operational mailbox evidence, distinct from the outstanding
intake/acknowledgement/escalation exercise in REPORT-01 and REPORT-02. See
[mailbox readiness](../operations/reporting-mailbox-readiness.md).

Update, 9 November 2026: Douglas Todd approved manual email triage, recording
actionable reports in the moderation console, acknowledging ordinary reports
with a reference within five UK business days, and recording follow-up/outcomes.
Urgent concerns take priority over that acknowledgement target. The benign
end-to-end exercise is prepared but not yet reported completed.

Update, 9 November 2026: following the shared-link rejection and AT URI
workaround, Douglas Todd reports the full routine reporting flow worked. Record
external intake, linked case, acknowledgement and no-action resolution as a
successful owner-run exercise. This does not verify deployment of the HTTPS-link
fix, Brevo delivery, or the separate urgent/specialist workflows.

## Intimate-Image Response Target

On 9 November 2026 Douglas Todd approved a **24-hour internal target** from the
original report receipt, alongside action as soon as practicable and the
separately applicable statutory requirements. Checked-in production configuration
now specifies `24h`. Linked reviews must use the intake's saved original receipt,
not a later form time. Live configuration/deployment and the benign tabletop
remain unverified; this decision does not close II-01 or REPORT-02.

## Initial Emergency Response Procedure

On 9 November 2026 Douglas Todd explicitly approved the initial response:
prioritize credible immediate danger when aware, contact appropriate emergency
services without waiting for NCA registration, use appropriate CraftSky controls,
preserve necessary records securely, record follow-up, and independently verify
requests and establish a lawful basis before private-data disclosure. This is
initial-procedure approval; legal-basis/disclosure detail review, staffing/cover,
restricted-evidence readiness and the benign console exercise remain open under
LE-01. No emergency contact or disclosure was made by this review.

Update, 9 November 2026: Douglas Todd reports that the credible-threat console
exercise works. Record workflow creation, attributed history and resolution as
owner-verified. No real police contact or private-data disclosure is established
by the test. The detailed disclosure procedure remains a separate draft for
owner approval under LE-01.

Update, 9 November 2026: after reviewing background on compulsory/voluntary
requests and emergencies, Douglas Todd explicitly approved the disclosure
operating procedure. Record independent verification, case-specific lawful-basis
and necessity review, minimization, secure transfer and restricted decision
records as approved operating requirements. This is not approval of any actual
disclosure or completion of the separate lawful-basis/DPIA work. The benign
fraudulent-request console exercise is prepared and pending.

Update, 9 November 2026: Douglas Todd confirmed that the unverified-authority
request flow works. Record request creation, refusal, absence of a disclosure
action, closure and retained history as a successful owner-run synthetic
exercise. The approved procedures and both LE-01 console exercises are now
recorded. This does not verify restricted evidence/preservation storage, secure
external transfer, actual agency contact or staffing cover. Those dependencies
remain to reconcile before overall launch approval.

Update, 9 November 2026: Douglas Todd confirmed that the separate restricted
evidence S3 storage has not been configured. A dedicated private bucket setup
and synthetic verification plan are prepared in
`docs/operations/restricted-evidence-storage.md`. No AWS resource creation,
credential creation, production configuration or deployment is approved or
verified by this confirmation. CSEA-02 and the LE-01 preservation dependency
remain open.

Update, 9 November 2026: Douglas Todd explicitly approved creation of the
dedicated evidence bucket and supplied a signed-in CraftSky AWS console.
Created `craftsky-prod-safety-evidence-17482696` in Frankfurt and the dedicated
IAM user `craftsky-appview-safety-evidence` with the bucket-only inline policy
`CraftSkyRestrictedEvidenceOnly`. Inspected saved bucket protections and both
policies; provisioning evidence is recorded in
`docs/operations/restricted-evidence-storage.md`. No access key was created and
AppView configuration, deployment and synthetic hold/deletion verification
remain outstanding. CSEA-02 and LE-01 preservation remain open.

## Launch Video Scope

On 9 November 2026 Douglas Todd explicitly confirmed video uploads are included
in the launch release. Carry video through the illegal-content and children's
risk assessments, moderation verification and final release evidence. This
confirms product scope only; it does not approve the assessments, establish
scanner/provider effectiveness, activate a production flag or authorize a
deployment. Reconcile the publication manifest's historical
`videoProductionDisabled` requirement with this approved scope before final
release approval.

## Grooming Risk Rating

On 9 November 2026 Douglas Todd agreed to record High grooming design risk as
the basis for reviewing safeguards. The rationale and initial control review
are in `online-safety/assessments/illegal-content-risk-assessment.md`. This is
approval of that individual rating only, not acceptance of remaining risk,
approval of the full assessment or overall launch, or authorization to change
DM reporting/settings or collect additional age information.

## Image-Based CSAM and CSAM-URL Working Ratings

On 9 November 2026 Douglas Todd agreed to separate Medium initial working
ratings for image-based CSAM (including video) and CSAM URLs. Record these as
the basis for further control review, with final calibration still open and
with no cap on a higher rating if the evidence warrants it. This does not
establish detection effectiveness, approve remaining risk, close CSEA/OSA gaps
or approve launch. The rationale is recorded in the illegal-content assessment.

## Harassment/Abuse and Hate Working Ratings

On 9 November 2026 Douglas Todd agreed to separate Medium working ratings for
(1) harassment, stalking, threats and abuse, and (2) hate. The assessment records
credible pathways, potentially severe individual impact, existing safeguards
and evidence limits. This agreement does not establish live effectiveness or
approve remaining risk, the full assessment or launch. Controlling/coercive
behaviour is being assessed separately.

## Controlling/Coercive Behaviour Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for
controlling or coercive behaviour. Record the rationale, severe potential
individual impact and proposed handling checks in the illegal-content
assessment. This agreement covers the rating only; survivor-sensitive handling,
live effectiveness, final calibration and overall launch approval remain
separate matters.

## Intimate-Image Abuse Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for
intimate-image abuse. This is separate from his earlier approval of the
24-hour internal response target. Live matching, visibility action, deadline
verification and evidence controls remain outstanding; neither decision closes
II-01, approves remaining risk or authorizes launch. The rating rationale is in
the illegal-content assessment.

## Extreme Pornography Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for extreme
pornography. Record this separately from intimate-image abuse, CSAM and the
broader content-policy prohibition. Live detection/video coverage and final
control review remain unverified; agreement does not close the assessment or
approve remaining risk or launch.

## Sexual Exploitation of Adults Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for sexual
exploitation of adults. The illegal-content assessment records the recruitment/
contact pathways, potentially severe harm, available sexual-violation response
and operational evidence limits. This is approval of the individual working
rating only; it does not establish live control effectiveness or approve the
full assessment, remaining risk or launch.

## Human Trafficking Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for human
trafficking. The assessment includes labour/sexual exploitation pathways,
potentially severe harm and the unresolved offence-to-sanction mapping. This
agreement approves the individual working rating only; it does not approve
remaining risk, establish operational effectiveness or authorize launch.

## Unlawful-Immigration Offences Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for
unlawful-immigration offences. Keep this distinct from trafficking and lawful
migration, humanitarian support and legitimate recruitment. The assessment
records misuse pathways and incomplete offence/sanction mapping. Agreement
covers the working rating only; no operational assurance or overall launch
approval follows.

## Fraud and Financial-Services Offences Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for fraud
and financial-services offences. The assessment records craft-commerce misuse,
potentially serious individual loss, available fraud/impersonation controls
and outstanding serious-fraud sanction mapping/live verification. This approves
the individual working rating only, not control effectiveness, remaining risk,
the completed assessment or launch.

## Proceeds of Crime Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for
proceeds of crime. The assessment records coordination through profiles,
purported commerce, search and DMs, with payments potentially external. The
rating does not establish financial regulation status, transaction monitoring,
live control effectiveness or a particular reporting/disclosure obligation.
Sanction mapping, final calibration and overall approval remain outstanding.

## Drugs and Psychoactive Substances Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for drugs
and psychoactive substances. Keep unlawful supply/facilitation distinct from
legitimate craft, health, educational and recovery discussion. The assessment
records potentially severe harm and unresolved sanction/operational evidence.
This agreement covers the working rating only, not effectiveness, acceptance
of remaining risk or overall launch approval.

## Firearms, Knives and Other Weapons Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for
firearms, knives and other weapons. The assessment distinguishes ordinary craft
tools/demonstrations from unlawful supply, restricted sales and threats. The
rating retains potentially severe harm and outstanding sanction/operational
verification; it does not approve a transaction, remaining risk or launch.

## Suicide and Serious Self-Harm Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for
encouraging or assisting suicide and serious self-harm. The assessment retains
potentially fatal harm and distinguishes harmful promotion from support,
recovery and help-seeking. Urgent response, sanction coverage, final calibration
and live effectiveness remain subject to review. The agreement approves this
working rating only, not remaining risk or overall launch.

## Foreign Interference Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for foreign
interference. The assessment distinguishes the specific offence from political
discussion, advocacy, criticism and overseas participation, and records limited
attribution/operational evidence. This agreement does not establish any campaign,
approve remaining risk or complete the overall launch assessment.

## Animal Cruelty Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for animal
cruelty. The assessment distinguishes unlawful cruelty from legitimate fibre
sourcing, husbandry, shearing, education and welfare criticism. Detection/video
coverage, contextual enforcement and live effectiveness remain unverified;
agreement covers the working rating, not remaining risk or launch.

## Cyberflashing Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for
cyberflashing. The assessment records public directed-media pathways, the
text-only inspected chat send boundary, available sexual-content responses
and unresolved routing/live verification. This agreement does not establish
provider payload support, fulfil the specific offence test or approve remaining
risk or overall launch.

## Terrorism Working Rating

On 9 November 2026 Douglas Todd agreed to the Medium working rating for
terrorism. The assessment retains potentially catastrophic harm, records
relevant content/contact pathways and named reporting/sanction controls, and
distinguishes lawful contextual discussion. Specialist handling, detection,
Code applicability and live response remain to verify. This is agreement to
the individual working rating, not remaining risk or overall launch approval.

## Other Relevant Illegal Content Working Rating

On 9 November 2026 Douglas Todd agreed to the collective Medium working rating
for other relevant illegal content, limited to supported relevant-offence
pathways rather than ordinary civil disputes or policy breaches. The current
assessment consolidates the individual agreements in a single table and
proposes High overall CSEA risk for separate confirmation. The full assessment,
remaining risk and launch remain unapproved; rating agreements are not evidence
that the corresponding safeguards are operational.

Update, 9 November 2026: Douglas Todd did not approve the proposed High overall
CSEA rating. He questioned whether it reflects CraftSky's audience, stating
that the service does not target children or particularly appeal to them, and
requested the full reasoning. Record overall CSEA as unresolved and revisit the
audience, contact/discovery factors and safeguards before seeking agreement.
Preserve the earlier individual subtype agreements as provisional assessment
inputs rather than treating them as automatic overall approval.


## Sanction Coverage Implementation Approval

On 9 November 2026 Douglas Todd explicitly approved
`docs/changes/2026-11-09-moderation-sanction-coverage/proposal.md` and its scoped
implementation: a separately reviewed Other illegal content decision with
private legal rationale and safe owner explanation, and severe-suspension
eligibility for serious fraud and harmful self-harm encouragement/assistance.
Generic Other remains ineligible for strikes/severe suspension. Human review,
proportionality, severity rationale, permissions, audit, appeal and reversal
remain required. Distress/help-seeking and ordinary disputes are not grounds by
themselves. No report/scanner signal becomes a finding automatically.

The approval covers source changes and local verification. It does not approve
policy publication, deployment, a real sanction, final risk calibration, the
unresolved overall CSEA rating or launch. See the change's validation record for
local evidence and remaining release/live checks.


## Overall CSEA Rating Decision — 9 November 2026

After the source/local contact-control review and sanction-coverage work,
Douglas Todd expressly authorized the assistant to select the best-supported
risk rating: “happy to set the risk rating to what you think is best.” Under
that authorization the overall CSEA working rating is **High**, with the
service-specific balancing recorded in assessment version 0.4. This supersedes
the earlier pending/unresolved overall-rating entries in this chronology.

The decision credits CraftSky's specialist audience, expected small scale,
chronological feed and partial contact safeguards. It retains concern about
permitted child access, potentially age-revealing profiles, one-to-one contact,
severe individual impact and incomplete evidence of prevention and timely
response. It asserts no incident or prevalence and is not an automatic maximum
of the subtype labels. Grooming remains High; image/video-based CSAM and CSAM
URLs remain separately Medium. Current Ofcom guidance was rechecked for the
judgment; the final applicability/control mapping remains outstanding.

This is authorization of the rating decision, not approval of the completed
assessment, acceptance of unmitigated remaining risk or authorization to launch.
No new data collection, provider settings, DM-report route, sanction,
publication or deployment is approved by this decision. Reassess against the
recorded review triggers and the deferred hosted evidence before launch.


## Narrowed Post-Report Information Scope — 9 November 2026

Douglas Todd expressly declined UI for reporters to see the status of reports
at this stage and authorized brief process/estimated-timeline copy after a report
is submitted. The implementation updates the existing native/web confirmation:
review and action if needed, a ten-UK-business-day aim for routine decisions,
and priority for urgent safety concerns. This is an estimate, not a guarantee,
measured performance claim or change to shorter specialist deadlines.

No report-status dashboard/history, status API, communication-preference system,
additional contact collection or automated reporter correspondence is approved.
Existing Bluesky-only DM reporting and manual mailbox procedures are preserved.
Remaining current-Code reporting-information and operational findings are
tracked separately; this scope decision does not approve whole-assessment
completion, publication, deployment or launch.


## Children's Assessment Review Progress — 9 November 2026

Douglas Todd requested work on the children's assessment and safeguards after
reviewing the remaining launch blockers. Assessment v0.2 prepares age-group
considerations and separate proposed Medium ratings for the four primary,
eight priority and two Ofcom-identified non-designated content kinds. These
are assessment recommendations, not additional owner approvals, observed
incidence or completed assessment/launch approval. Previously agreed illegal-
content/CSEA ratings and product-scope decisions are unchanged.

The new children's control register covers all 52 PCU application clauses,
including conditional age/ranking/provider questions. It identifies report/search
support and public child/parent materials as separate from a report-status UI.
The first concrete owner decision is the full harmful-content prohibition and
context boundary in `docs/changes/2026-11-09-childrens-content-boundary/proposal.md`.
No broader prohibition or client feature was implemented, no Ofcom notification
was sent, and no third-party/private-message authority is inferred by this review.


## Full Harmful-to-Children Content Boundary Approval — 9 November 2026

Douglas Todd explicitly approved
`docs/changes/2026-11-09-childrens-content-boundary/proposal.md`. The decision
prohibits the covered harmful-content kinds for every user of child-accessible
CraftSky parts, including primary/priority material, body stigma and promotion
of depression/hopelessness/despair, with contextual safeguards. Educational,
medical, artistic or historical context is not a blanket exception where the
harmful-content definition is still met. Ordinary craft, neutral fit/size facts,
non-graphic health/education, grief, recovery and help-seeking remain protected
when they do not meet those definitions.

Guidelines v0.4, Terms v0.5, Reporting v0.3 and internal operator notes now
reflect the decision in source. The approval does not classify lawful policy
breaches as crimes, authorize new sanctions, collect ages, alter Bluesky-only DM
reporting, add report-status UI, or prove all-surface/provider enforcement.
Ratings and the whole children's assessment remain unapproved. Publication,
deployment and launch still require their separate evidence/approval. The
next child-support/materials feature proposal is prepared, not approved.

## Code-based privacy impact assessment

At Douglas Todd's request, a [launch DPIA v0.1](../assessments/data-protection-impact-assessment.md)
and [source evidence register](../assessments/dpia-evidence.md) were prepared on
9 November 2026 from this worktree. They cover the actual public/private,
children, messaging, media, safety, identity-linkage, telemetry and conditional
provider boundaries. No production records or provider settings were inspected.

This is preparation evidence for PRIV-01, not owner approval of the assessment,
a final acceptable-residual-risk finding, completion of LIAs/Article 9/10
conditions, or an Article 36 consultation decision. The DPIA's A01–A12 record the
remaining scope, legal-route, retention, supplier, child-privacy, operational and
exact-release evidence. Existing product decisions remain recorded separately;
no new collection, external communication, deployment or policy publication was
authorized or performed by creating this document.
