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

The final procedure must define:

- who is on call and during which hours;
- when to contact UK emergency services or another competent authority;
- how to handle an unknown location or a person outside the UK;
- emergency disclosure authority and legal review;
- preservation requests and authenticity checks;
- communications with a reporter or person at risk; and
- post-incident review and staff support.

Until these are approved, public material must not promise 24/7 monitoring or an
emergency response time.

## Law-Enforcement Requests

- Verify requester identity and authority through an independent official route.
- Require appropriate legal process unless an approved emergency exception
  applies.
- Disclose only data CraftSky controls and only what the lawful request requires.
- Do not disclose PDS credentials, DPoP keys, or unrelated account data.
- Record the request, legal basis, decision, disclosed fields, approver, and date
  in a restricted system.
- Apply any lawful preservation hold without silently turning it into indefinite
  general retention.

## Operational Readiness Checklist

- [ ] Accountable online-safety and child-safety owners named.
- [ ] On-call coverage and escalation contacts approved.
- [ ] NCA organization registration complete.
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
