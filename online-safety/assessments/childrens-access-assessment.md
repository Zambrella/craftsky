# Children's Access Assessment

| Field | Value |
|---|---|
| Assessment owner | Douglas Todd, Founder |
| Approved by | Not approved |
| Version | 0.1 |
| Status | Ready for owner approval |
| Assessment date | 11 September 2026 |
| Internal review target | Within 12 months, and earlier on a trigger |
| Publication | Internal regulatory record |

## Question and Legal Test

This assessment determines whether CraftSky is likely to be accessed by children
for Part 3 of the Online Safety Act 2023. For this test, a child is anyone under
18.

Ofcom's process has two stages:

1. Determine whether children can normally access the service. A conclusion that
   they cannot normally access it requires highly effective age assurance and
   access controls.
2. If children can access it, determine whether the service has a significant
   number of child users or is of a kind likely to attract a significant number
   of children.

## Service Evidence

| Factor | Evidence | Finding |
|---|---|---|
| Age assurance | CraftSky has no age-verification or age-estimation control. | Children can normally access the service. |
| Contractual age | CraftSky has decided to launch for people aged 16 and over. The April 2026 public Terms and July 2026 Community Guidelines still say 13+ and must be replaced. | Terms alone are not highly effective age assurance. The approved position intentionally permits some under-18s. |
| Registration | Registration uses an external AT Protocol identity and CraftSky does not collect date of birth. | CraftSky cannot reliably identify child users. |
| Subject matter | Textile and fibre crafts, tutorials, projects, creativity, and social connection appeal to adults and some children. | The service provides plausible benefits and attractive content to children. |
| Design | Profiles, visual media, likes, follows, comments, search, and notifications are familiar social features. | The design is likely to attract some children. |
| Commercial strategy | No evidence was found of child-directed marketing. | Reduces attraction but does not outweigh access and product characteristics. |
| Telemetry | The website uses limited PostHog analytics; Flutter and AppView can use sanitized Sentry diagnostics when configured. No evidence shows that either is used to infer age or interests. | Child-specific telemetry controls and transparency still require Children's Code/DPIA review. |
| Current users | The service is pre-production with no active users. | No reliable user-age evidence is available. Absence of current users is not evidence that children will not access it after launch. |

## Stage 1 Finding

**Children can normally access CraftSky.** CraftSky has neither highly effective
age assurance nor access controls that establish adult status. Proceed to stage
2.

## Stage 2 Finding

**The child user condition should be treated as met.**

The service's creative subject matter and ordinary social features are capable of
attracting a material number of under-18s. CraftSky has no evidence capable of
supporting the opposite conclusion, and Ofcom advises services to err on the side
of caution. The confirmed 16+ minimum intentionally admits 16- and 17-year-olds,
who are children for the Act.

## Conclusion and Consequences

CraftSky is **likely to be accessed by children**. CraftSky adopts completion of
the resulting assessment and safeguards before UK launch as an internal readiness
requirement. The statutory transitional timetable for a new service ordinarily
allows three months from the relevant start day, but the applicable safety duties
operate from launch and should not be implemented without the assessments needed
to inform them. CraftSky must:

- complete and record a suitable and sufficient children's risk assessment;
- identify which Protection of Children Code measures apply;
- implement proportionate safeguards before launch;
- make reporting and complaints suitable for children;
- ensure Terms explain the protections for children and are applied consistently;
- complete a Children's Code/Data Protection Impact Assessment review;
- provide child-appropriate privacy and safety information; and
- monitor child use and the effectiveness of safeguards.

The contractual minimum age is 16. This conclusion means the safety and privacy
design must account for under-18 access unless and until a later
assessment, supported by highly effective age assurance and access controls or
other compelling evidence, reaches a different lawful conclusion.

## Review Triggers

Review this assessment under CraftSky's annual internal control and:

- before launch and when the minimum-age decision is made;
- before implementing age assurance;
- before adding direct messages, group chat, livestreaming, location, or a
  recommender system;
- when reliable age or usage evidence becomes available;
- when the UK makes or commences regulations under Online Safety Act section
  214A; and
- whenever Ofcom materially changes relevant guidance or risk profiles.

The mandatory annual children's-access reassessment in section 36 applies while
a service is not treated as likely to be accessed by children. Following this
positive finding, CraftSky must keep its children's risk assessment and related
compliance measures up to date.

## Sources

- [Ofcom, Children's access assessment duties under the Online Safety Act](https://www.ofcom.org.uk/online-safety/illegal-and-harmful-content/childrens-access-assessment-duties-under-the-online-safety-act), updated 29 June 2026.
- [Online Safety Act 2023, section 35](https://www.legislation.gov.uk/ukpga/2023/50/section/35) and [section 36](https://www.legislation.gov.uk/ukpga/2023/50/section/36).
- [Ofcom, Age assurance duties under the Online Safety Act](https://www.ofcom.org.uk/online-safety/protecting-children/age-assurance).
- [ICO, Introduction to the Children's Code](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/childrens-information/childrens-code-guidance-and-resources/introduction-to-the-childrens-code/).
- [Online Safety Act 2023, section 214A](https://www.legislation.gov.uk/ukpga/2023/50/section/214A) and the [Government response on growing up online](https://www.gov.uk/government/consultations/growing-up-in-the-online-world-a-national-consultation/outcome/growing-up-in-the-online-world-government-response-july-2026).

## Approval Record

| Decision | Name | Date | Notes |
|---|---|---|---|
| Prepared | Repository review | 11 September 2026 | Based on product design and current official guidance. |
| Accountable owner approval | Pending | Pending | Owner must accept the conclusion and commission the children's risk assessment. |

## Change History

| Version | Date | Change | Author/approver |
|---|---|---|---|
| 0.1 | 11 September 2026 | Initial assessment and likely-access conclusion. | Working draft / not approved |
