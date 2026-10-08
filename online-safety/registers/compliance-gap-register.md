# Compliance Gap Register

| Field | Value |
|---|---|
| Register owner | Douglas Todd, Founder |
| Version | 0.3 |
| Status | Active working register |
| Last reviewed | 14 September 2026 |
| Publication | Internal |

Priority meanings: **P0** blocks a safe/legal UK launch; **P1** should be resolved
before public launch unless the accountable owner records a lawful alternative;
**P2** is planned assurance or documentation work.

| ID | Priority | Gap | Required outcome | Evidence/status |
|---|---|---|---|---|
| GOV-01 | P0 | Specialist approval is outstanding | Douglas Todd approves the release after specialist review and records accepted residual risk. | Accountable owner assigned; review pending |
| GOV-02 | P0 | Legal operator/controller details are incomplete | Confirm company number, jurisdiction, postal address, final registered name, and DPO position. | `CraftSky` is the assumed name; formation details remain unresolved |
| OSA-01 | P0 | Illegal-content assessment is incomplete | Complete current Ofcom worksheets, ratings, Code mapping, owner approval, and records. | Initial draft exists |
| CHILD-01 | P0 | Children's Access Assessment not approved | Owner approves likely-access conclusion. | Ready for approval |
| CHILD-02 | P0 | No children's risk assessment or Protection of Children Code mapping | Complete, implement, test, and approve before launch. | Not started |
| CHILD-03 | P0 | Current public pages still say 13+ | Replace them with the confirmed 16+ position and reconcile Terms, Privacy, Guidelines, onboarding, registration, and enforcement. | Product decision complete; implementation/publication pending |
| CHILD-04 | P1 | No child-appropriate notices or documented child-safe defaults | Produce Children's Code assessment and implementation backlog. | Not started |
| AGE-01 | P1 | No age-assurance architecture or DPIA | Design privacy-preserving threshold result, provider boundary, challenge route, minimization, and accessibility alternatives. | Wait for final 2027 rules where appropriate; design boundary now |
| AGE-02 | P1 | Future under-16 regulations not tracked | Monitor section 214A regulations and Ofcom over-16 standard. | First regulations expected by end of 2026; not current ban |
| MOD-01 | P0 | Report/decision taxonomy omits CSEA, threats, intimate images, self-harm, doxxing, fraud, and other priority harms | Add user-safe intake categories and enforceable internal reasons without leaking sensitive detail. | Current ten-code taxonomy insufficient |
| MOD-02 | P0 | Serious categories cannot always receive a strike/severe suspension | Align sanctions with approved Guidelines and safety assessment. | `other` cannot strike; severe reason allowlist is narrow |
| MOD-03 | P1 | Detected content without a user report cannot enter an adjudication case | Preserve the accepted user-report case boundary and document lawful alternative handling for detected CSEA and emergencies. | Product decision recorded; legal/operational mapping pending |
| MOD-04 | P1 | Independent appeal review is not always available | Use another reviewer where practicable; otherwise record reconsideration by the original moderator and obtain specialist review of the safeguard. | Policy decision recorded; operational review pending |
| REPORT-01 | P0 | External email intake is not operationally tested | Test `moderation@craftsky.social` for non-users, representatives, pseudonymous reporters, acknowledgements, records, and escalation. | Public route and owner defined; test pending |
| REPORT-02 | P0 | Reporting service levels are not operationally tested | Implement and exercise the five-UK-business-day acknowledgement target and shorter legal escalations. | Policy decision recorded; final-decision time remains case-dependent |
| CSEA-01 | P0 | No operational CSEA/NCA process | Register, appoint administrator, train staff, secure evidence, implement priority/report/retention flow, and exercise it. | Restricted draft only |
| CSEA-02 | P0 | Account deletion currently purges moderation reports/cases without a legal-hold boundary | Implement a narrowly scoped, legally reviewed restricted evidence export/hold before deletion cleanup where retention is legally required; delete it when the legal period ends. | Current private cleanup has no hold exception |
| II-01 | P0 | No qualifying intimate-image route or 48-hour workflow | Implement section 20A intake, expedited complaints, deadline tracking, and same/substantially-same process. | Generic adult/graphic report is insufficient |
| LE-01 | P0 | No credible-threat, emergency, or law-enforcement process | Approve authority verification, emergency disclosure, preservation, and escalation procedure. | Draft placeholders only |
| POLICY-01 | P0 | Public Terms do not explain all current illegal-content protections and section 72(1) claim right | Rewrite and obtain legal review. | Replacement draft exists; current April 2026 HTML remains incomplete |
| POLICY-02 | P0 | Existing Community Guidelines overstate automation, warnings, and permanent suspension | Review repository draft and align implementation before publication. | Replacement draft exists |
| POLICY-03 | P0 | Public Privacy Policy misstates deletion, omits Sentry, and claims PostHog runs in the app without repository evidence | Inventory processing by surface, including website PostHog, app/service Sentry, hosting/logging, email, identity linkage, and retention; then rewrite the Privacy Policy. | Replacement draft exists; current April 2026 HTML remains stale |
| PRIV-01 | P0 | No DPIA covering social service, children, moderation, federation, and future age assurance | Complete the DPIA before relevant processing, mitigate risks, and complete any required Article 36 prior consultation before processing if high residual risk remains. | Not started |
| PRIV-02 | P0 | No purpose/lawful-basis record | Map each data purpose to Article 6 and any Article 9/10 condition; complete LIAs where used. | Not started |
| PRIV-03 | P1 | No complete processor and transfer inventory | Record role, contract, subprocessor, location, transfer mechanism, and review. | Amazon S3 in AWS Frankfurt selected for scheduled media; AWS terms, subprocessors, and remaining providers require review |
| PECR-01 | P0 | The public website loads PostHog without an established PECR consent or Schedule A1 route | Stop loading PostHog pending consent, or document and implement all Schedule A1 conditions, including information, a simple free objection control, purpose limits, and sharing restrictions. | Memory-only persistence and Do Not Track handling are insufficient on their own |
| RET-01 | P0 | No unified retention schedule | Cover databases, moderation evidence, CSEA records, logs, Sentry, PostHog, email, support, backups, local drafts, and deletion. | Some code-level periods exist |
| RIGHTS-01 | P1 | No documented rights-request/data-protection complaint procedure | Add intake, identity checks, deadlines, searches, decisions, and ICO escalation. | Public email only |
| BREACH-01 | P0 | No approved personal-data breach plan | Add immediate triage/logging; Article 33 ICO notification without undue delay and, where feasible, within 72 hours when its risk threshold is met; Article 34 notification to affected people without undue delay when its high-risk threshold is met; and processor escalation. | Not located |
| IP-01 | P1 | Copyright and trade-mark procedure needs specialist review and testing | Review the notice, response, restoration, misuse, and federation process and exercise mailbox handling. | Public draft created |
| COMM-01 | P1 | Commercial disclosure policy and labels need ASA/CAP review | Define gifted, affiliate, sponsored, and business treatment; ensure obvious labels. | Sponsored metadata exists; broader states need review |
| SUB-01 | P1 | Subscription terms and lifecycle are not implemented or reviewed | Implement the approved monthly Pro/Business licence, assignment, reassignment, deletion, renewal, cancellation, and store-entitlement rules; obtain consumer/app-store review. | Product and draft Terms decisions complete |
| ACCESS-01 | P1 | No public accessibility statement or release standard | Adopt WCAG 2.2 AA target, publish contact/limitations, and define audit gate. | Some automated coverage exists |
| RECORD-01 | P1 | No Ofcom Code/adopted-alternative measure register | Record applicable measures, deviations, rationale, and reviews. | Not started |
| METRIC-01 | P2 | No safety effectiveness metrics | Define privacy-conscious prevalence, response, appeal, recurrence, and control-failure metrics. | Do not infer guilt from report count |

## Change History

| Version | Date | Change | Author/approver |
|---|---|---|---|
| 0.1 | 11 September 2026 | Initial consolidated gap register. | Working draft / not approved |
| 0.2 | 14 September 2026 | Assigns ownership and distinguishes resolved policy decisions from outstanding implementation, testing, and review. | Douglas Todd / approval pending |
| 0.3 | 14 September 2026 | Records the assumed company name and selection of Amazon S3 in AWS Frankfurt. | Douglas Todd / approval pending |
