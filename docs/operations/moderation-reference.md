# CraftSky moderation reference: reports, decisions and follow-up

| Field | Value |
|---|---|
| Operational owner | Douglas Todd |
| Reviewed | 9 November 2026 |
| Status | Working reference to current source and recorded procedures; not final launch approval |
| Scope | CraftSky reports, external intake, moderator review, specialist escalation, appeals and recovery |

Start with diagram 1. Open the numbered detail diagram when it is referenced.
Box text describes a human task unless it explicitly says the system performs it.
A **report** is an allegation; a **case** is the review/decision record; a
**specialist workflow** records safety, reporting or authority work. These may
run together and have separate closure conditions.

This guide consolidates existing decisions rather than approving new powers.
The console does not contact emergency services, submit NCA reports, send
reporter emails or disclose information merely because a workflow form is saved.
Source changes and configuration may still await deployment; check the readiness
notes at the end before relying on them.

## 1. Overall route: receive, triage, act and follow up

```mermaid
flowchart TD
    START["A concern arrives: app, guest report, email or moderator observation"] --> DANGER{"Credible immediate danger?"}
    DANGER -->|Yes| EMERGENCY["Follow emergency response now: diagram 4A<br/>Do not wait for ordinary acknowledgement"]
    DANGER -->|No| ROUTE{"What needs handling?"}
    EMERGENCY --> ROUTE
    ROUTE -->|Bluesky message or conversation report| DM["Use Bluesky report route<br/>No duplicate CraftSky report or copied transcript"]
    DM --> DMEND["Keep provider limits explicit<br/>Emergency action can still be necessary"]
    ROUTE -->|Owner appeal or handling complaint| APPEAL["External intake and verification: diagram 6"]
    ROUTE -->|Authority request| AUTH["Restricted authority review: diagram 5"]
    ROUTE -->|CraftSky content or account concern| INTAKE["Locate existing case or record intake/provenance: diagram 2"]
    INTAKE --> SAFETY{"Any specialist safety indicators?"}
    SAFETY -->|Yes| SPECIAL["Start every relevant specialist route: diagram 4<br/>Track original receipt and required deadlines"]
    SAFETY -->|No| REVIEW["Review subject, context and evidence: diagram 2"]
    SPECIAL --> REVIEW
    REVIEW --> DECISION["No action or supported violation<br/>Choose proportionate consequences: diagram 3"]
    DECISION --> COMMIT["Review and confirm exact command<br/>If uncertain, use diagram 7"]
    COMMIT --> VERIFY["Read saved history and verify actual visibility/standing"]
    VERIFY --> FOLLOWUP["Safe notices, correspondence and records: diagram 6<br/>Continue any outstanding specialist obligations"]
    FOLLOWUP --> CHECK{"All required work complete?"}
    CHECK -->|No| OPEN["Keep the relevant case or specialist work outstanding<br/>Record owner, next step and due time"]
    OPEN --> FOLLOWUP
    CHECK -->|Yes| CLOSE["Close the relevant workflow<br/>Retain only justified records; review holds and lessons"]
```

**Urgency first.** Use these internal triage levels, reassessing when facts change.
These are not the NCA's separate reporting priorities.

| Internal priority | Typical indicator | What to do first |
|---|---|---|
| P1 | Immediate danger to life/serious harm; current or imminent child danger | Take the emergency route immediately; gather minimum facts and use appropriate official contacts. |
| P2 | Near-term serious harm, urgent safeguarding, suspected CSEA without P1 indicators, qualifying intimate-image report | Prioritize specialist review and restriction where supported; assign responsibility and track deadlines immediately. |
| P3 | Other suspected illegal content | Assess promptly, limit avoidable exposure and document the legal judgment. |
| P4 | Ordinary Guidelines concern | Use ordinary case review; upgrade if context shows a more serious concern. |

A report count, keyword, scanner status or risk-assessment rating does not prove a
violation or justify downgrading urgent facts. Several specialist routes can
apply to one concern. An intimate-image concern involving a child can require
both intimate-image and child-safety handling.

## 2. Intake, correct subject and evidence review

```mermaid
flowchart TD
    SOURCE{"Where did this concern arrive?"}
    SOURCE -->|In-app or guest report| EXISTING["Open Cases<br/>Find the report and its linked/open case"]
    SOURCE -->|Email or other external correspondence| EMAIL["External intake: Record external intake<br/>Stable message reference, original receipt, subject,<br/>kind, urgency and attachment handling"]
    EMAIL --> KIND{"What kind of intake?"}
    KIND -->|Appeal or handling complaint| REFER["Use diagram 6<br/>Do not create a new violation just for complaining"]
    KIND -->|Content/account allegation| CREATE["Create case from intake<br/>Verify subject DID and exact record URI"]
    SOURCE -->|Moderator observation| OBSERVE["Check for an existing case/incident<br/>Use Create case for an indexed subject when appropriate<br/>Record honest operator provenance; do not invent a user report"]
    EXISTING --> SUBJECT["Confirm subject: post, account, event or profile link<br/>Separate a reply, quote or linked item from the reported item"]
    CREATE --> SUBJECT
    OBSERVE --> SUBJECT
    SUBJECT --> CONTEXT["Read allegations and safe context<br/>Compare captured CID/version with current indexed content"]
    CONTEXT --> SAFE{"Is the relevant evidence safely available?"}
    SAFE -->|Changed, missing or restricted| LIMIT["Record the limitation<br/>Do not reconstruct old content or bypass a restricted preview"]
    LIMIT --> ACCESS{"Authorized restricted evidence access needed?"}
    ACCESS -->|Yes, approved route available| RESTRICT["Use Restricted evidence with a recorded reason<br/>No screenshots, ordinary downloads or forwarding"]
    ACCESS -->|No or unavailable| HOLD["Seek the necessary safe information or approved assistance<br/>Keep urgency and required restrictions under review"]
    RESTRICT --> ASSESS["Assess actual facts, context, policy and any offence elements"]
    SAFE -->|Yes| ASSESS
    HOLD --> ASSESS
    ASSESS --> FINDING{"Supported violation?"}
    FINDING -->|Yes| VIOLATION["Use diagram 3<br/>Separate factual notes, legal rationale and safe explanation"]
    FINDING -->|No finding supported| NOACTION["No action; record why, including evidence limits<br/>This is not a declaration that the report was dishonest"]
    FINDING -->|Further review necessary| PENDING["Keep review outstanding<br/>Record missing facts, next step and urgency"]
    PENDING --> CONTEXT
```

Practical intake details:

- Open the console at `https://appview.craftsky.social/admin/`. The visible tabs
  depend on permissions. Use **Cases**, **Accounts**, **External intake**,
  **Safety incidents**, **Specialist workflows**, **Restricted evidence** and
  **Legal holds** for their separate purposes.
- Use the email's Message-ID or another stable mailbox reference, not its subject
  alone. Record the **original arrival time**, including the correct timezone.
  Keep original emails in the approved mailbox; **Add correspondence** records
  subsequent message references and attachment handling rather than a transcript.
- Use the exact canonical AT URI where required, for example
  `at://<author-DID>/social.craftsky.feed.post/<record-key>`. The reported account
  DID and URI owner must match. Shared app links are normalized by the local fix;
  use the AT URI on a deployed release that still rejects the HTTPS link.
- A new case applies **no consequence**. A repeat report on a subject is context,
  not an automatic new strike. Check provenance/history before opening more work.
- Ordinary previews show safe indexed content. They do not reconstruct a captured
  version that changed, fetch restricted bytes or establish that a missing item
  never existed. Account cases without captured versions show current profiles.
- Use only enough restricted review to make the decision. Access is separately
  audited; closing/hiding the page clears the viewer. Do not preserve illegal or
  intimate material in normal case notes, email, chat, screenshots or logs.

## 3. Decide the finding and select consequences

```mermaid
flowchart TD
    FACTS["Review available facts and context"] --> FINDING{"Is a policy violation supported?"}
    FINDING -->|No| NA["Choose No action<br/>Record factual reasons and any uncertainty<br/>Apply no consequences"]
    FINDING -->|Yes| REASON["Choose the most accurate specific policy reason<br/>Assess illegality separately from policy harm"]
    REASON --> OTHER{"Using Other illegal content?"}
    OTHER -->|Yes| LEGAL["Record offence/legal basis and supporting facts<br/>Choose Other illegal content legal classification<br/>Provide private legal rationale and safe owner explanation"]
    OTHER -->|No| NOTES["Record internal evidence notes<br/>Provide a useful, safe owner explanation"]
    LEGAL --> NOTES
    NOTES --> CONTENT{"Does the content need to become unavailable?"}
    CONTENT -->|Yes| REMOVE["Select Take down in CraftSky or justified Hide content<br/>A warning banner alone does not remove content"]
    CONTENT -->|No| VISIBLE["Select a proportionate permitted effect<br/>Formal warning and Content warning have different purposes"]
    REMOVE --> ACCOUNT{"Separate account consequence justified?"}
    VISIBLE --> ACCOUNT
    ACCOUNT -->|No| REVIEW["Review the complete decision summary"]
    ACCOUNT -->|Strike proportionate| STRIKE["Add Account strike only for an eligible reason<br/>One active strike per case; check existing standing"]
    ACCOUNT -->|Severe violation| SEVERE["Check severe-eligible reason<br/>Record a separate severity rationale<br/>Do not wait for three strikes if severe suspension is justified"]
    STRIKE --> REVIEW
    SEVERE --> REVIEW
    REVIEW --> CONFIRM["Confirm the exact command<br/>Recover an uncertain result with diagram 7"]
    CONFIRM --> CHECK["Verify saved decision/effects and actual affected surfaces<br/>Then use diagram 6"]
    NA --> CONFIRM
```

Choose the finding from the content and conduct, not merely the reporter's chosen
category. A Guidelines breach can be actionable without a criminal finding.
When making an illegal-content judgment, record supported offence elements and
consider relevant context/defences; escalate genuine uncertainty rather than
asserting illegality to obtain a stronger sanction.

| Console consequence | What it actually does | When to consider it |
|---|---|---|
| **Formal warning** | Official notice in owner history; public visibility unchanged; no automatic strike | A supported breach needs an owner warning, with no separate removal/account penalty justified. |
| **Content warning** | Warning shown to viewers of the affected profile/post; content remains accessible; no automatic strike | Permitted content needs a warning. It is insufficient where the policy requires removal or child-access prevention. |
| **Hide content** | Makes the affected subject unavailable to other people in CraftSky | A justified visibility restriction under the applicable policy. Currently this is not a blur/reveal control. |
| **Take down in CraftSky** | Records a takedown and makes the affected subject unavailable to other people in CraftSky | A removal decision, including a supported prohibited-content finding. Use the action that accurately records the decision. |
| **Account strike** | One strike for the case; three active case strikes suspend the account; each is due to expire after one calendar year | A proportionate account consequence for an eligible supported violation. Removal or a warning alone does not add it. |
| **Severe suspension** | Suspends regardless of strike count; remains until explicitly restored | A qualifying severe violation with separate severity reasoning. It is distinct from three-strike suspension. |

Only one visibility effect may be selected in a decision. Other justified,
compatible effects may be combined. A violation decision requires factual internal
notes and at least one supported consequence. **No action** is the route when no
violation is established, rather than a violation with no consequences.

The generic **Other** reason cannot add a strike or severe suspension. **Other
illegal content** is a different, reviewed route; it requires its additional
legal fields and must never be used to relabel lawful content. Severe eligibility
in current source covers harassment, hate, adult/graphic, impersonation, child
safety, sexual violation, threat/violence, stalking/privacy, reviewed illegal
content, fraud/scam and self-harm. Eligibility is not an instruction to suspend
every case in those categories.

For fraud and self-harm, severe action requires supported serious wrongdoing:
ordinary transaction disputes, distress, illness, recovery and help-seeking are
not enough. The approved harmful-to-children boundary applies to all users of
child-accessible parts, while ordinary craft instructions and neutral garment
fit/size discussion remain welcome. Artistic/educational context is relevant,
but not a blanket exemption from a harmful-content definition.

**Scope of enforcement:** these moderation actions control CraftSky presentation
and participation. They do not delete the owner's PDS records or guarantee that
independent services, known media URLs or provider messages become inaccessible.
Verify public/guest/member detail, discovery/search, linked previews and relevant
media paths separately. Record anything that remains accessible and escalate the
control gap; saving an outcome code is not proof of removal.

## 4. Specialist safety routes

Run all applicable branches alongside necessary CraftSky moderation. Keep
responsibility, original receipt/detection time, next action and due time visible.
These diagrams summarize the linked restricted runbook; they do not replace the
case-specific assessment or authority submission requirements.

### 4A. Immediate danger and credible threats

```mermaid
flowchart TD
    THREAT["Credible current/imminent threat to life or serious harm"] --> FACTS["Gather minimum reliable facts<br/>Person at risk, relevant content, timing and known location"]
    FACTS --> LOCATION{"Where is the emergency?"}
    LOCATION -->|UK| UK["Contact UK emergency services on 999 when appropriate"]
    LOCATION -->|Known overseas| OVERSEAS["Use an appropriate official emergency/police route for that location"]
    LOCATION -->|Unknown| UNKNOWN["Seek UK police assistance and state location uncertainty<br/>Do not invent or infer a precise location"]
    UK --> PROTECT["Restrict harmful CraftSky content/contact as justified<br/>Use approved restricted preservation"]
    OVERSEAS --> PROTECT
    UNKNOWN --> PROTECT
    PROTECT --> WORKFLOW["Record Credible threat review and chronology<br/>Agency/reference, action, owner and follow-up"]
    WORKFLOW --> PRIVATE{"Private data disclosure proposed?"}
    PRIVATE -->|Yes| DISCLOSE["Use diagram 5<br/>Make a prompt lawful, necessary and minimized emergency decision"]
    PRIVATE -->|No| REVIEW["Reassess safety and complete required follow-up"]
    DISCLOSE --> REVIEW
    REVIEW --> OUTCOME["Record Threat outcome and lessons<br/>Close only when required work is accounted for"]
```

Do not wait for NCA registration, a completed ordinary investigation or a routine
acknowledgement before seeking emergency help. Never call emergency services for
a test. Recording a threat outcome does not itself contact an authority.

### 4B. Suspected child sexual exploitation or abuse

```mermaid
flowchart TD
    CSEA["Suspected CSEA identified by report or review"] --> SAFE["Minimize viewing; restrict CraftSky access as justified<br/>No ordinary download, screenshot or forwarding"]
    SAFE --> LIVE{"Current or imminent danger to a child?"}
    LIVE -->|Yes| EM["Use diagram 4A immediately<br/>Portal approval must not delay emergency contact"]
    LIVE -->|No| ASSESS["Authorized review: classify facts and reportability<br/>Check reliable prior-report information"]
    EM --> ASSESS
    ASSESS --> REQUIRED{"Detected and reportable, without an applicable prior report?"}
    REQUIRED -->|No| RECORD["Record supported no-report or duplicate rationale<br/>Continue any required policy/safeguarding action"]
    REQUIRED -->|Yes| READY{"Approved reporting access and secure handling available?"}
    READY -->|No| BLOCKED["Escalate the readiness failure urgently<br/>Use appropriate official assistance<br/>Do not mark a report sent or invent a reference"]
    READY -->|Yes| PRIORITY["Set NCA priority, responsible operator and deadline<br/>Do not confuse with internal P1-P4"]
    PRIORITY --> SEND["Submit available required information through approved external route<br/>Supplement missing information as required"]
    SEND --> RECEIPT["Authority report record: actual reference and submission details<br/>Set required retention; track information requests"]
    RECEIPT --> FOLLOW["Respond through approved external channel<br/>Record response, ongoing holds and outstanding work"]
    FOLLOW --> DONE["Incident resolution only after reporting and retention obligations reviewed<br/>Linked case consequences remain separate"]
    RECORD --> DONE
```

| NCA reporting priority | Timing in the response runbook | Console classification label |
|---|---|---|
| Priority 1 | Immediately | `immediate` |
| Priority 2 | As soon as reasonably practicable | `urgent` |
| Priority 3 | Without undue delay | `standard` |

Apply the actual NCA criteria rather than mechanically translating internal
triage. The console requires a reporting deadline; do not choose an arbitrary
future time that postpones the applicable standard. Do not delay an initial
report solely to obtain every possible field. A previous report elsewhere must
be assessed under the applicable duplicate-report rules, not guessed from the
number of CraftSky reports.

Use the restricted runbook for statutory record classes: NCA unique report
reference, required retained content/information/associated-user data and any
applicable hold. Its recorded periods distinguish five-year reference retention
from one-year required material retention from the report date, including
relevant associated-user data from the two-week period before the content was
uploaded, created, shared or received, rather than an assumed two weeks before
the report. NCA information requests must
be handled as soon as reasonably practicable and no later than seven days under
the recorded procedure. Do not dump unrelated account data or credentials into a
report. Do not treat a mailbox attachment as an approved evidence store.

**Current readiness:** NCA registration is recorded as submitted, awaiting
approval; portal readiness and restricted evidence connection are not verified.
This branch is a procedure reference, not a claim those prerequisites work.
Bluesky-only DM reporting remains unchanged; a concern learned through messaging
may still need emergency/legal escalation, and unresolved provider/duty
boundaries must not be dismissed simply because a provider report was sent.

### 4C. Intimate-image abuse

```mermaid
flowchart TD
    IMAGE["Intimate-image report received"] --> RECEIPT["Record original receipt immediately<br/>Open linked Intimate image review"]
    RECEIPT --> CLOCK["Track approved 24-hour internal target<br/>Also track applicable 48-hour statutory outer limit"]
    CLOCK --> TEST["Assess content, reporter standing and required declarations<br/>Do not impose unnecessary proof-of-authority hurdles"]
    TEST --> QUALIFY{"Qualifying statutory report?"}
    QUALIFY -->|Yes| TAKE["Take down reported content in CraftSky as soon as reasonably practicable<br/>Do not wait until the target/deadline"]
    TAKE --> MATCH["Use proportionate means to find same/substantially-same content<br/>Act on identified copies within the applicable deadline"]
    MATCH --> VERIFY["Verify actual visibility changes and matching coverage<br/>Record any limitations, including media/provider paths"]
    QUALIFY -->|No| OTHER["Record the supported standing/content assessment<br/>Assess ordinary policy, illegality, CSEA or emergency routes anyway"]
    VERIFY --> OUTCOME["Complete Intimate image outcome<br/>Judgment, searches, actual action, assessed exception and outcome"]
    OTHER --> OUTCOME
    OUTCOME --> CONTACT["Communicate safely; track expedited handling complaints<br/>Retain justified evidence and chronology"]
```

The original receipt clock does not restart when the console review is opened,
when a duplicate arrives or when you become available. Linked intake supplies its
saved receipt time; enter the true original time for an unlinked review. An old
item may already be overdue. The internal 24-hour target does not extend a legal
deadline. A failed or unconfigured workflow form must not postpone protective
action; record the problem and escalate it.

The statutory branch has narrow content/standing conditions; a failed standing
check does not establish that the content is permissible. Use the runbook for the
specific assessment and expedited complaint requirements. Outcome/search fields
record the work; they do not search for matching material or apply a takedown.

## 5. Authority requests and disclosures

```mermaid
flowchart TD
    REQUEST["Purported law-enforcement or authority request received"] --> RECORD["Record Authority request in restricted workflow<br/>Purpose, requested fields, urgency and dates"]
    RECORD --> VERIFY["Verify requester and request using independently obtained official contact details"]
    VERIFY --> GENUINE{"Genuine and adequately verified?"}
    GENUINE -->|No| REFUSE["Do not disclose<br/>Record refusal or clarification needed; close or keep under review"]
    GENUINE -->|Yes| POWER["Is it compulsory or voluntary?<br/>Record actual power/order/duty where applicable"]
    POWER --> BASIS["Assess lawful basis, necessity, proportionality,<br/>sensitive-data conditions and any international-transfer requirements"]
    BASIS --> ALLOWED{"Disclosure justified and approved?"}
    ALLOWED -->|No| REFUSE
    ALLOWED -->|Yes| MINIMIZE["Choose only necessary data CraftSky controls<br/>No OAuth tokens, credentials, unrelated records or unrestricted transcripts"]
    MINIMIZE --> SEND["Transmit separately to verified recipient over approved secure channel"]
    SEND --> AUDIT["Disclosure record: actual approved fields, basis,<br/>approver, date and safe transfer reference"]
    AUDIT --> CLOSE["Authority closure after review<br/>Any preservation hold has its own scope and expiry/review"]
    REFUSE --> CLOSE
```

An authentic police request is not automatically a compulsory demand. A purported
emergency is not blanket permission to export private information. Immediate
emergency contact can proceed promptly while the necessary disclosure decision
is made and recorded. A crime-prevention exemption alone is not the lawful basis.
The console's **Disclosure record** records an approved disclosure; it is not a
secure-transfer tool. If a provider alone holds the requested data, explain
CraftSky's actual access/control rather than pretending it is available locally.

## 6. Communication, appeals, restoration and closure

```mermaid
flowchart TD
    SAVED["Decision committed and checked"] --> NOTICES["Verify safe owner-facing outcome/history<br/>Handle reporter correspondence through available approved channel"]
    NOTICES --> APPEAL{"Owner appeal received?"}
    APPEAL -->|No| TASKS["Continue specialist deadlines, correspondence and retention tasks"]
    APPEAL -->|Yes| VERIFY["External intake kind: appeal<br/>Verify sender is the affected owner and match case reference"]
    VERIFY --> OWNER{"Ownership and case match established?"}
    OWNER -->|No| CLARIFY["Request proportionate clarification<br/>Do not disclose private case details from a reference alone"]
    CLARIFY --> VERIFY
    OWNER -->|Yes| LINK["Link verified appeal/correspondence to existing case<br/>Confirm an appeal in case follow-up"]
    LINK --> REVIEW["Review evidence and original reasons<br/>Different reviewer where practicable;<br/>otherwise record same-reviewer reconsideration"]
    REVIEW --> OUTCOME{"Appeal outcome?"}
    OUTCOME -->|Upheld| UPHELD["Record rationale; retain justified effects"]
    OUTCOME -->|Changed| CHANGE["Select the active effects to reverse, including severe suspension<br/>Record rationale and confirm one appeal-resolution command"]
    CHANGE --> VERIFYEFFECTS["Check remaining effects and actual standing/visibility<br/>Other cases or suspension grounds may still apply"]
    UPHELD --> NOTIFY["Communicate the safe appeal outcome"]
    VERIFYEFFECTS --> NOTIFY
    NOTIFY --> TASKS
    TASKS --> FINISH{"All required actions accounted for?"}
    FINISH -->|No| TRACK["Keep relevant workflow outstanding<br/>Record next action, owner and due time"]
    TRACK --> TASKS
    FINISH -->|Yes| CLOSE["Close appropriate case/specialist records<br/>Review retention/holds and record lessons"]
```

Communication and timing:

| Task | Operational reference |
|---|---|
| Ordinary email acknowledgement | Approved target: within five UK business days, with a safe reference. Urgent action takes priority. This is manual; the mailbox is not automatically ingested. |
| Routine report decision | Native/web receipt copy gives an aim of ten UK business days from receipt. Complexity can affect handling; urgent/specialist/legal duties take priority. Receipt/support UI is not a completed review or a final-outcome message. |
| Reporter outcome | Where an appropriate contact route exists, communicate a safe handling outcome. Do not reveal private sanctions/evidence or another person's account details. Record whether contact was possible and any applicable communication preference. |
| Affected owner's notice | Verify owner-safe explanation/history and the available notice path. Do not put reporter identity, raw evidence, private legal rationale or authority routing in it. |
| Appeal window | Working Reporting Policy: normally within 30 calendar days of notice; good reasons for late appeals can be considered. This is a policy window, not an automatic legal-right cutoff. |
| Appeal/handling-complaint acknowledgement | Working policy aims for five UK business days; acknowledge and review without undue delay. No fixed universal final appeal deadline is implemented. Qualifying intimate-image complaints require expedited treatment. |

Appeal email does not change enforcement. Existing consequences remain while a
confirmed appeal is reviewed. One appeal lifecycle is stored per case; later
correspondence can still be considered and any later effect change must be
audited. **Effect change** and **Restore account** are distinct follow-up actions
with their own confirmation/rationale; use the supported action for the intended
change. Reversing a strike, restoring a severe suspension or lifting one case's
restriction does not automatically clear every other active restriction.

**Severe-only appeal resolution — fixed locally:** for a pending appeal, choose
**Change the decision**, select **Severe suspension** under **Effects to reverse
on appeal**, enter the appeal rationale, review and confirm. The existing appeal
command reverses that case's selected effect and resolves the appeal together.
Unselected consequences, other cases' severe suspensions and an independent
three-strike suspension remain in force. Do not run a separate restoration first
for this appeal flow. The standalone **Restore a severe suspension** action remains
available for justified restoration outside appeal resolution. Deployment and the
hosted operator check are still pending; see the [fix validation](../changes/2026-11-09-severe-only-appeal/validation.md).

A complaint about CraftSky's handling can be recorded as intake kind **complaint**
and investigated without treating the complainant as a violator. Verify the
right requester before accessing private case details. A data-protection rights
request or privacy complaint also needs the privacy procedure/contact at
`support@craftsky.social`, with its separately applicable requirements; do not
close it merely because moderation correspondence is resolved. Copyright/trade-
mark notices require the separate [IP procedure](../../online-safety/policies/copyright-and-trade-marks.md).

There is no reporter-status/history dashboard, automatic reporter-resolution
email or guaranteed staffed 24-hour monitoring. Do not promise them. Current
remaining communication/disclosure/preference gaps are tracked in REPORT-02;
this reference does not claim those have been implemented.

## 7. Safe command confirmation and recovery

```mermaid
flowchart TD
    PREPARE["Prepare a decision or follow-up command<br/>Read current case/resource revision"] --> REVIEW["Review exact subject, effects, notes and visible explanation"]
    REVIEW --> SEND["Confirm once"]
    SEND --> RESULT{"What happened?"}
    RESULT -->|Confirmed success| READ["Read saved history and verify resulting state"]
    RESULT -->|Validation or permission failure| FIX["Read error; correct facts/fields or obtain legitimate access<br/>Do not bypass permissions"]
    FIX --> PREPARE
    RESULT -->|Revision conflict| REFRESH["Load fresh state; compare history<br/>Prepare and review a new command only if still necessary"]
    REFRESH --> PREPARE
    RESULT -->|Timeout or uncertain outcome| KEEP["Keep page open; do not submit a newly recreated action"]
    KEEP --> RETRY["Use Retry exact command<br/>Preserve original body, revision and idempotency key"]
    RETRY --> RESULT
    RESULT -->|Success but refresh failed| COMMITTED["Treat command as committed<br/>Refresh/read history before any further action"]
    COMMITTED --> READ
    RESULT -->|Page/session lost| INSPECT["Sign in again; inspect case/resource history first<br/>In-memory recovery is gone"]
    INSPECT --> NEEDED{"Is the intended change already committed?"}
    NEEDED -->|Yes| READ
    NEEDED -->|No, established from fresh state| PREPARE
    NEEDED -->|Still uncertain| WAIT["Resolve uncertainty before a fresh action<br/>Do not create duplicate sanctions or disclosures"]
```

Do not run the same recovery command from another tab or invent a new key to
force it through. Restricted evidence **access** is a separate audited read:
it has no command replay key, and each new access attempt is separately recorded.
Restricted evidence **preservation** uses its supported exact-command recovery;
do not confuse the two.

## Before leaving a report or incident

Check that the record tells the next reviewer:

1. What arrived, when, about which exact subject/version, and how urgent it is.
2. What facts were established, what remains uncertain, and why the decision fits
   the actual policy/legal test.
3. Which consequences actually committed and what visibility/standing was checked.
4. Whether an authority was genuinely contacted or a disclosure/report sent,
   through which approved channel, and its safe reference.
5. Whether the owner/reporter could be contacted safely, what was communicated,
   and what appeal/complaint or specialist work remains.
6. What evidence is retained, its basis/expiry/holds, and the next owner/action/time.

Resolving a moderation case is not confirmation that a linked authority report,
information request, retention obligation or intimate-image search is complete.
Use the separate records and do not manufacture a completion code to empty a queue.

## Readiness limits and sources

This reference was checked against current source and operational records on
9 November 2026. It documents workflows; it is not a claim that all launch
controls are live or that all policies are published.

- **External intake:** ordinary mailbox/reply/intake exercise is owner-reported
  as passed. Intake remains manual. Shared-link normalization awaits deployment.
- **Urgent email:** Brevo delivery/retry code is local; production activation,
  receipt and phone notification remain unverified. Alerts are generic prompts
  to inspect queues, not automatic authority reports or deadline reminders.
- **Evidence:** dedicated S3 bucket/IAM provisioning is recorded; AppView
  connection and synthetic hold/deletion exercise are deferred until merge.
  Do not improvise another evidence store if the approved route is unavailable.
- **CSEA:** NCA application is recorded as submitted, approval/access unverified.
  Emergency contact must not wait for portal access.
- **Intimate images:** 24-hour internal target is approved/in source; live target,
  matching process and actual action exercise remain to verify.
- **Decision taxonomy:** Other illegal content and expanded severe eligibility
  require the corresponding migration/release; verify the deployed console.
- **Policies:** use the applicable published policy and reviewed legal duties;
  the approved content-boundary decision and repository drafts do not establish
  that new wording is already effective. Report that mismatch rather than claim
  the reporter/owner was already given unpublished terms.

Canonical internal references:

- [Operator runbook](moderation-operators.md), including permissions, preview
  boundaries, restricted evidence, consequences, alerts and content boundaries.
- [Restricted response runbook](../../online-safety/operations/illegal-content-response.md).
- [Mailbox readiness and manual intake](../../online-safety/operations/reporting-mailbox-readiness.md).
- [Intimate-image readiness](../../online-safety/operations/intimate-image-readiness.md).
- [Restricted evidence storage](restricted-evidence-storage.md).
- [Working Reporting, Complaints and Appeals Policy](../../online-safety/policies/reporting-complaints-and-appeals.md).
- [Working Community Guidelines](../../online-safety/policies/community-guidelines.md).
- [Launch approval and recorded decisions](../../online-safety/governance/launch-approval.md).

Source checks: [case decisions](../../appview/admin/src/cases/CaseActions.tsx),
[consequence help](../../appview/admin/src/cases/ConsequenceHelp.tsx),
[follow-up actions](../../appview/admin/src/cases/FollowUp.tsx),
[workspace actions](../../appview/admin/src/workspaces/actions.ts),
[case provenance](../../appview/internal/moderation/operator_case.go),
[decision validation](../../appview/internal/moderation/policy.go) and
[standing/expiry rules](../../appview/internal/moderation/standing.go).

Official guidance and legislation checked for this guide (public CSEA XML was
used where the web reader could not parse the XHTML):
[Ofcom CSEA reporting guidance](https://www.ofcom.org.uk/online-safety/illegal-and-harmful-content/duty-to-report-child-sexual-exploitation-and-abuse-csea-content-know-the-rules-and-how-to-comply),
[NCA portal and emergency guidance](https://www.nationalcrimeagency.gov.uk/what-we-do/crime-threats/child-sexual-abuse-and-exploitation/the-child-sexual-exploitation-abuse-industry-reporting-portal),
[CSEA Reporting Regulations 2026](https://www.legislation.gov.uk/uksi/2026/268/contents/made),
and [enacted intimate-image provisions](https://www.legislation.gov.uk/ukpga/2026/20/part/5/enacted).
The detailed statutory periods above follow the canonical restricted runbook;
this summary does not expand the reporting or disclosure authority recorded there.
