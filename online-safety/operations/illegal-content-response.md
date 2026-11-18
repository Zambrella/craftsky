# Illegal Content Response Runbook

| Field | Value |
|---|---|
| Operational owner | Douglas Todd, Founder |
| Child-safety owner | Douglas Todd, Founder |
| Approved by | Not approved |
| Version | 0.1 |
| Status | Restricted draft; not operationally approved |
| Last reviewed | 11 September 2026 |
| Publication | Restricted internal; exclude from static-site builds |

## Safety Notice

Do not use this draft as evidence that CraftSky has an operational CSEA,
emergency, or law-enforcement process. Named people, secure systems, NCA
registration, legal review, training, access controls, and exercises are still
required.

Never copy illegal or traumatic content into tickets, chat, email, source control,
or general-purpose logs. Authorized personnel must nevertheless securely submit
and retain material and data expressly required by law, including the CSEA
reporting and retention regulations, using only approved restricted systems.

## Launch Review Status — 9 November 2026

Douglas Todd is the named accountable online-safety and child-safety owner and
has confirmed he intends to monitor reports during the day and outside UK
business hours. Backup and staffed overnight cover are not arranged. Generic
urgent alerts through Brevo are implemented locally; production activation and
actual mailbox/phone delivery remain outstanding.

Restricted workflow chronology is implemented in AppView. NCA organization
registration application was reported submitted by Douglas Todd on 9 November
2026. NCA approval is pending; no equivalent reporting arrangement is established. Administrator access,
training and an operational exercise remain unverified. These prerequisites
remain open; this update does not approve the runbook or record a live report.

## Intake Priorities

| Priority | Indicators | Initial action |
|---|---|---|
| P1 | Immediate threat to life or immediate risk of serious harm; live child danger | Alert the on-call safety owner immediately; preserve necessary identifiers; consider emergency referral under approved legal process. |
| P2 | Possible serious harm in the near future; urgent safeguarding; suspected CSEA without a P1 indicator; qualifying intimate-image report | Restrict access, assign specialist owner, and begin legal deadline tracking immediately. |
| P3 | Other suspected illegal content | Queue promptly, prevent avoidable spread, and assess without undue delay. |
| P4 | Non-illegal Community Guidelines concern | Use ordinary moderation workflow. Escalate if facts indicate illegality or urgency. |

No automated score or report count may downgrade an apparent P1/P2 concern or
establish guilt.

## General Workflow

1. **Acknowledge safely.** Issue a reference without confirming whether another
   report exists or exposing account information.
2. **Triage.** Identify immediate danger, child safety, intimate-image abuse,
   terrorism, credible threats, fraud, evidence volatility, and staff-safety
   concerns.
3. **Limit exposure.** Keep access to the minimum trained group. Do not invite
   additional people to view traumatic material.
4. **Preserve necessary evidence.** Record source identifiers, timestamps,
   reporter contact where needed, decision context, and integrity information in
   the approved restricted system.
5. **Assess.** Use all relevant information reasonably available. For an illegal-
   content judgment, consider whether there are reasonable grounds to infer the
   elements of an offence and no reasonable grounds to infer a successful
   defence. Escalate uncertainty rather than invent legal certainty.
6. **Act on CraftSky.** Apply a formal warning, visibility warning, visibility
   hide, visibility takedown, strike, or suspension as the implemented taxonomy
   permits. Moderation action does not delete the user's PDS record.
7. **Report externally where required.** Use only approved portals and authorized
   personnel.
8. **Notify carefully.** Give safe, useful outcomes without revealing reporters,
   restricted evidence, investigation methods, or information that creates more
   risk.
9. **Record and review.** Record rationale, action, legal report reference,
   deadlines, retention trigger, and lessons. Update risk assessments after a
   serious incident or control failure.

## CSEA-Specific Workflow

The Online Safety Act CSEA reporting duty for regulated user-to-user services has
applied since 7 April 2026 and is not limited by service size. It concerns CSEA
content CraftSky detects on the service that has not already been reported under
an applicable route. Detection may result from a user report, human moderation,
or another system; the duty does not itself require proactive scanning.

### Before handling any case

- Register the organization with the NCA before the first report.
- Appoint an eligible organization administrator and maintain current contacts.
- Limit handling to trained, authorized personnel with appropriate wellbeing
  support.
- Configure the NCA portal/API and preserve credentials outside source control.
- Approve a secure evidence store and retention/deletion controls.
- Confirm the process for avoiding duplicate reports, including applicable
  arrangements with another reporting body.

### Registration action — verified 9 November 2026

Timing distinction: Ofcom describes registration as required **before submitting
reports**. NCA registration is not an approval or licence to launch the service.
CraftSky's checklist asks for registration in advance as an operational-readiness
control so time-critical reporting can work from launch; this is distinct from
claiming the legislation sets a universal pre-launch registration deadline.
Reporting arrangements through an equivalent foreign agency may satisfy the
reporting route where the applicable requirements are met. No such arrangement
has been established for CraftSky.


Use the [NCA industry portal](https://www.uk-child-sexual-abuse-industry-reporting.gov.uk/home)
and select **Register your organisation**, following the
[NCA organization registration guide](https://www.nationalcrimeagency.gov.uk/who-we-are/publications/803-how-to-register-your-organisation-for-the-child-sexual-exploitation-and-abuse-industry-reporting-portal).
Prepare:

- CRAFTSKY LTD, company number 17482696, United Kingdom;
- platform name CraftSky and relevant website addresses;
- the actual headquarters address (do not assume the registered office is the
  operational headquarters);
- the registering administrator's name, role, work email and phone number;
- a shared contact mailbox and an emergency contact's name, phone and email.

The first registrant becomes the organization administrator. Verify the work
email, complete the organization/address/contact fields, review and submit.
Record the application status internally; submission is not approval. NCA
processes registrations Monday–Friday 07:00–17:00, excluding UK bank holidays.
Approval brings portal instructions and introductory-call arrangements. Portal
credentials and verification links must remain outside this repository.

The NCA guide explicitly allows repeating previously supplied emergency-contact
details. Its user-administration guidance requests a deputy for continuity;
CraftSky's sole-person staffing and lack of cover remain separately unresolved.
No registration was submitted by the assistant.

### When suspected CSEA is detected

1. Do not download, screenshot, forward, quote, or duplicate the material into an
   ordinary system. Authorized personnel must use the approved restricted systems
   to submit and retain the content and data required by regulations 6 and 8.
2. Restrict visibility in CraftSky and access by staff as quickly as the evidence
   and safeguarding process permits.
3. Classify NCA priority:
   - Priority 1: report immediately.
   - Priority 2: report as soon as reasonably practicable.
   - Priority 3: report without undue delay.
4. Submit all required information that is available. Do not delay an initial
   report solely because other information is unavailable; supplement it later.
5. Keep the NCA unique report reference for five years from its date of issue.
6. Beginning on the date the report is sent, keep for one year the detected CSEA
   content, information submitted, information used to make the CSEA judgment,
   and every applicable category of associated-user data specified by regulation
   8(2), including relevant data from the preceding two-week period. Apply strict
   access, security, and deletion controls.
7. Respond to an NCA information request as soon as reasonably practicable and no
   later than seven days.
8. Record why a detected item was or was not reportable without placing illegal
   content in the ordinary moderation record.

The approved procedure must separately address non-UK providers/UK linkage if the
operator structure changes.

## Intimate-Image Reports

Douglas Todd approved a **24-hour internal response target** on 9 November 2026,
measured from original receipt. Apply `SAFETY_INTIMATE_IMAGE_RESPONSE_WITHIN=24h`
in production through the normal configuration controls; the checked-in
Blueprint/example are updated, but live application has not been verified.
A review linked to intake uses its saved receipt time and may already be overdue
when entered. The target does not defer action that can be taken sooner.

For a qualifying report under section 20A:

1. capture the declarations, contact details, and information required by section
   20A without imposing an additional proof-of-authority requirement unless a
   legally reviewed, proportionate check is necessary;
2. start a 48-hour deadline at receipt;
3. prioritize the complaint and determine whether either narrow section 10(3B)
   condition applies: the content is not statutory intimate-image content, or the
   reporter is neither the subject nor acting on the subject's behalf;
4. take the reported content down from CraftSky as soon as reasonably practicable;
5. use proportionate means to identify and take down the same or substantially
   same content by the deadline;
6. communicate safely with the reporter; and
7. record the timeline, judgment, search/action taken, exception if relied upon,
   and outcome.

This workflow requires product support and cannot be satisfied merely by mapping
the report to `adult_or_graphic`.

## Credible Threats and Immediate Harm

### Initial response — approved by Douglas Todd, 9 November 2026

Douglas Todd is the operational decision owner. On becoming aware of a credible
immediate threat to life or serious harm:

1. Prioritize it immediately over routine moderation and acknowledgement queues.
   Gather the minimum available facts about the threat, person at risk, location
   and relevant content without waiting for a complete investigation.
2. For an emergency in the UK, contact police on 999. For current/imminent child
   danger while NCA registration is pending, NCA guidance directs contact with
   the relevant local UK police force or the organization's local force if the
   child's location is unknown. Do not wait for portal approval.
3. For a known overseas location, identify an appropriate official police or
   emergency contact route for that location; do not assume UK emergency numbers
   reach it. For unknown location, explain the uncertainty when seeking UK police
   assistance and provide only verified location information.
4. Use existing moderation controls to restrict harmful content/contact in
   CraftSky where appropriate; preserve necessary source identifiers/timestamps
   and relevant records in approved restricted systems. Do not delete PDS
   records as a moderation action or copy harmful evidence into ordinary email.
5. Record the chronology, decision, external agency/reference and follow-up in
   the restricted incident/workflow. Sharing private data requires a documented
   lawful basis and minimized scope; a purported request does not by itself
   authorize disclosure. Apply the law-enforcement verification procedure below.
6. Review the incident afterward, including control failures and required updates
   to risk assessments. Communicate safely with the reporter/person at risk where
   appropriate, avoiding unsupported promises or disclosures that create risk.

Douglas Todd approved this initial response on 9 November 2026. It is a response
on awareness, not a guarantee of staffed 24-hour
monitoring or a fixed emergency response time. Backup/absence arrangements,
restricted evidence configuration, emergency disclosure legal-basis review and
a benign tabletop remain open. Do not call emergency services for a test.

Sources checked 9 November 2026:
[UK police contact guidance](https://www.gov.uk/contact-police) and
[NCA industry reporting guidance](https://www.nationalcrimeagency.gov.uk/what-we-do/crime-threats/child-sexual-abuse-and-exploitation/the-child-sexual-exploitation-abuse-industry-reporting-portal).

### Benign credible-threat workflow exercise — passed, owner-reported

This is a synthetic console-recording exercise only. Do not contact police,
submit a real legal report, restrict a real account, or add real harmful content.

1. Open **Specialist workflows** and **Start credible threat review**.
2. Use the exercise receipt time and enter the following preservation assessment:
   `TEST ONLY — no real threat or evidence. Exercise of recording, prioritisation and resolution; no emergency call.`
3. Review/confirm and check that the workflow appears as an active credible-threat
   review under your operator identity. Its creation does not contact authorities
   or impose moderation consequences.
4. Open **Complete threat review** and enter:
   `TEST ONLY — console workflow completed. No real threat, no disclosure, no emergency call and no moderation action.`
5. Confirm and check the workflow is resolved and its test notes/history remain
   available. Record whether creation, attributed history and resolution worked.

The underlying workflow uses fixed event names such as `threatEscalated` and
`emergencyServicesFirst`; the explicit test assessments and readiness evidence
must make clear that no real escalation occurred. A completed exercise verifies
console operation, not actual police contact, lawful disclosure, staff cover or
the readiness of restricted evidence storage.

Douglas Todd reported “it works” on 9 November 2026 after the exercise above.
Record creation, attributed history and resolution as a successful owner-run
console exercise. This does not establish a real external referral, disclosure,
restricted evidence preservation or continuous staffing.

## Law-Enforcement Requests

### Disclosure procedure — approved by Douglas Todd, 9 November 2026

Douglas Todd makes and records the decision. The console records the workflow;
it does not send data to an authority.

1. Record the request in the restricted authority-request workflow, identifying
   the claimed agency, purpose, requested fields, urgency and relevant dates.
   Verify the requester's identity and authority through independently obtained
   official contact details, not only a supplied address, telephone number or
   email signature. Treat purported urgency as unverified until assessed.
2. Establish whether the request is legally compulsory or voluntary. Record the
   specific legal power/order/duty where applicable; an authentic police request
   is not automatically a compulsory demand. Clarify scope or seek appropriate
   advice where authority or obligations are uncertain.
3. Before disclosing, record the applicable Article 6 lawful basis, necessity,
   proportionality and any additional requirements for special-category or
   criminal-offence data. A crime-prevention exemption does not itself supply a
   lawful basis. Consider international-transfer requirements for overseas
   requests. Do not treat public availability as a blanket exemption.
4. For immediate threats, follow the approved emergency response promptly.
   Contact with emergency services need not await completion of a routine
   disclosure review. Any emergency data disclosure still needs a documented
   lawful basis and necessary/minimized scope; urgency is not blanket permission
   to export private data. Make and record a proportionate emergency decision.
5. Share only the necessary data that CraftSky controls, with the verified
   recipient over an appropriate secure channel. Do not export PDS OAuth tokens,
   DPoP keys, service credentials, unrelated private records or unrestricted
   transcripts. Where data is held only by an independent provider, accurately
   state CraftSky's access/control boundary.
6. Record the verification, purpose, authority/lawful basis, decision, approver,
   disclosed field names, secure-transfer reference and date in the restricted
   system. Record refusal or requests for clarification too. Consider applicable
   transparency obligations and any justified restrictions without automatically
   notifying someone in a way that prejudices an investigation. Any preservation
   hold must have a recorded basis, scope and review/end trigger; it is not
   indefinite general retention.

Douglas Todd approved this operating procedure on 9 November 2026. It does not
complete the lawful-basis inventory,
DPIA, international-transfer assessment or case-specific legal decision. A benign
fraudulent-request tabletop and the relevant restricted-system checks remain
outstanding. No real request or disclosure is being made during this review.

Source checked 9 November 2026:
[ICO: sharing personal data with law enforcement authorities](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/data-sharing/sharing-personal-data-with-law-enforcement-authorities/).

### Benign fraudulent-request exercise — passed, owner-reported

Use a clearly synthetic request; do not contact an authority, enter actual
requester details or share any private data.

1. In **Specialist workflows**, choose **Record authority request**. Use an
   exercise receipt time and requester reference `TEST-ONLY-unverified-request`.
2. Open **Verify authority request**. Enter verification reference
   `TEST ONLY — simulated requester could not be independently verified; no contact or disclosure occurred.`
   Leave **Requester and request verified as genuine** unchecked. Review and
   confirm the refusal.
3. Check the workflow moves to refused and the decision/history remain visible.
   **Record approved disclosure** must not be available for this refused request.
4. Use **Close authority review** with
   `TEST ONLY — unverified request refused; no data disclosed.`
   Confirm closure and retention of its audit history.

Report whether request creation, refusal, absence of a disclosure action, closure
and history worked. This verifies the rejected-request workflow, not actual
agency verification, a real lawful-basis decision, secure transfer or evidence
storage readiness. Douglas Todd confirmed on 9 November 2026 that the flow
works: record request creation, refusal of the unverified requester, absence of
a disclosure action, closure and retained history as an owner-run pass. No real
authority contact or data disclosure is established by this synthetic exercise.

## Operational Readiness Checklist

- [x] Accountable online-safety and child-safety owners named: Douglas Todd. This does not establish backup cover or training.
- [ ] On-call coverage and escalation contacts approved.
- [ ] NCA organization registration approved — application submitted by Douglas Todd on 9 November 2026; approval pending.
- [ ] NCA organization administrator appointed.
- [ ] Secure portal/API access tested.
- [ ] Restricted evidence system approved and access logged.
- [ ] Regulation 8 one-year CSEA content, judgment, submitted-information, and
  associated-user-data retention and five-year report-reference retention are
  automated and reconciled with account deletion.
- [ ] Intimate-image 48-hour timer and duplicate-content process tested.
- [ ] Report taxonomy and moderation sanctions support all urgent categories.
- [ ] Staff training, recusal, QA, and wellbeing measures complete.
- [ ] Tabletop exercises completed for CSEA, credible threat, intimate image, and
  fraudulent law-enforcement request scenarios.
- [ ] Privacy/DPIA and legal review signed off.

## Sources

- [Online Safety Act 2023, section 66](https://www.legislation.gov.uk/ukpga/2023/50/section/66).
- [Online Safety (CSEA Content Reporting by Regulated User-to-User Service Providers) Regulations 2026](https://www.legislation.gov.uk/uksi/2026/268/made).
- [Ofcom, Duty to report CSEA content](https://www.ofcom.org.uk/online-safety/illegal-and-harmful-content/duty-to-report-child-sexual-exploitation-and-abuse-csea-content-know-the-rules-and-how-to-comply), updated 23 July 2026.
- [Online Safety Act 2023, section 10](https://www.legislation.gov.uk/ukpga/2023/50/section/10) and [section 20A](https://www.legislation.gov.uk/ukpga/2023/50/section/20A).
- [Ofcom, Illegal Content Judgements Guidance](https://www.ofcom.org.uk/siteassets/resources/documents/online-safety/information-for-industry/illegal-harms/updates/illegal-content-judgements-guidance.pdf?v=419967).

## Change History

| Version | Date | Change | Author/approver |
|---|---|---|---|
| 0.1 | 11 September 2026 | Initial restricted workflow and readiness checklist. | Working draft / not approved |
