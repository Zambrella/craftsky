# CraftSky Online Safety and Compliance

This directory is the version-controlled source for CraftSky's online-safety,
privacy, and related regulatory documentation. Selected Markdown files are the
canonical source for public policies. Internal assessments, registers, and
runbooks must not be published.

The documents are working compliance records, not legal advice. They describe
CraftSky as it exists in this repository and identify facts that still require
operator or legal confirmation.

## Current Position

CraftSky should proceed on the basis that it operates a regulated UK
user-to-user service under the Online Safety Act 2023. It allows people to create
profiles, posts, comments, images, video, likes, reposts, follows, and public
events that other people can encounter. There is no small-service exemption,
although size, capacity, features, and assessed risk affect what safeguards are
proportionate.

CraftSky must also proceed on the basis that it is likely to be accessed by
children. It does not currently use highly effective age assurance, and its
creative subject matter is likely to appeal to some under-18s. See the
[Children's Access Assessment](assessments/childrens-access-assessment.md).

The UK Government intends to introduce an under-16 social-media restriction in
spring 2027. The enabling power is law, but the restriction was not operative on
11 September 2026. CraftSky must track the implementing regulations rather than
describe the announced policy as current law.

## Document Map

| Document | Audience | Status |
|---|---|---|
| [Document Control](governance/document-control.md) | Internal | Working |
| [Service Scope and Risk Profile](assessments/service-scope-and-risk-profile.md) | Internal | Draft |
| [Children's Access Assessment](assessments/childrens-access-assessment.md) | Internal regulatory record | Ready for owner approval |
| [Illegal Content Risk Assessment](assessments/illegal-content-risk-assessment.md) | Internal regulatory record | Incomplete draft |
| [Community Guidelines](policies/community-guidelines.md) | Public | Draft for review |
| [Terms of Service](policies/terms-of-service.md) | Public | Draft for legal review |
| [Privacy Policy](policies/privacy-policy.md) | Public | Incomplete draft |
| [Reporting, Complaints, and Appeals](policies/reporting-complaints-and-appeals.md) | Public | Draft; implementation gaps remain |
| [Copyright and Trade Mark Policy](policies/copyright-and-trade-marks.md) | Public | Draft for specialist review |
| [Illegal Content Response](operations/illegal-content-response.md) | Restricted internal | Draft; not operationally approved |
| [Compliance Gap Register](registers/compliance-gap-register.md) | Internal | Active |
| [Source Register](sources.md) | Internal/public references | Active |

## Rules for Maintaining These Documents

1. Public policy must describe implemented behaviour. Planned safeguards must be
   labelled as planned and must not be presented as available.
2. A user report is an allegation, not a finding. Only a recorded moderation
   decision may be described as a violation.
3. CraftSky moderation controls what CraftSky surfaces. It does not delete public
   PDS records as a moderation action and cannot promise removal from independent
   AT Protocol services.
4. Account deletion is different from moderation. After fresh authorization, the
   explicit account-deletion flow may delete the owner's records from the six
   currently allowlisted CraftSky collections: posts, likes, reposts, business
   events, business profiles, and the CraftSky actor profile. It does not delete
   the DID, PDS account, records from other collections, or blobs directly.
5. Significant service changes require safety and privacy review before release.
   Examples include direct messages, livestreaming, recommender systems, age
   assurance, location sharing, new upload types, and material moderation changes.
6. Every approved document must have an owner, approver, effective date, review
   date, and change history under the document-control standard.
7. Restricted runbooks must not be included in a future static-site build.
8. Public policies must be published atomically at extensionless URLs only after
   every required route and P0 control has been tested. Public pages show concise
   version and effective-date metadata; internal approval records stay here.
9. Superseded public policies are retained internally and supplied on request.

## Confirmed Product and Operator Decisions

- CraftSky will launch worldwide for people aged 16 and over, subject to any
  higher local minimum age. This is currently a contractual rule, not an
  age-verification claim.
- CraftSky will be operated by a registered company under the assumed name
  `CraftSky`; its number, jurisdiction, postal address, and final registered-name
  confirmation remain outstanding.
- Private scheduled-post media will use Amazon S3 in the AWS Europe (Frankfurt)
  Region.
- `support@craftsky.social` and `moderation@craftsky.social` will be checked on UK
  business days. Douglas Todd, Founder, initially owns the compliance functions.
- Markdown is canonical. The public policy routes will be `/terms`, `/privacy`,
  `/community-guidelines`, `/reporting`, and `/copyright`.
- The target effective date is 1 October 2026, but no policy becomes effective
  and the service does not launch until the recorded P0 controls are ready.

## Immediate Decisions Needed

- Confirm CraftSky's company details and postal address before publishing the Terms or Privacy Policy.
- Complete AWS and other processor/subprocessor records, transfer records,
  lawful-basis and legitimate-interest assessments, and special-category and
  criminal-offence data records.
- Obtain specialist review of the risk assessments, Terms, Privacy Policy,
  children's position, copyright process, and CSEA reporting workflow.
