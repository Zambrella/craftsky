# DPO Requirement Assessment — Launch

| Field | Value |
|---|---|
| Owner | Douglas Todd, Founder |
| Assessment date | 9 November 2026 |
| Status | Approved launch decision |
| Approval | Douglas Todd, 9 November 2026, explicit approval during launch-readiness review |
| Scope | Initial CraftSky launch; UK GDPR DPO requirement |
| Review | Before launch if assumptions change; after three months of operation; before material growth or processing changes |

## Evidence and Assumptions

- Douglas Todd confirmed he is the only person in the company and expects
  **thousands of active users in the first year**. This is a planning estimate,
  not a measured population or a statutory threshold.
- Douglas confirmed that a large amount of the processed data is public.
  Public AT Protocol posts, profiles and interactions are intentionally shared;
  they are assessed separately from private operational and moderation data.
- `render.yaml` configures Tap discovery using `social.craftsky.actor.profile`
  and an explicit collection filter. This supports a CraftSky-related ingestion
  scope rather than an assumption that all Bluesky users are indexed.
- `appview/internal/index/bluesky_profile.go` also supports non-member profiles.
  The affected population therefore cannot be equated exactly with active users.
  Actual production repository/profile counts have not been checked here.
- `web/main.js` gates optional website analytics on consent, disables autocapture
  and session recording, and uses memory persistence. This is source evidence,
  not verification of the deployed artifact or every processing surface.
- Public availability does not make identifiable records anonymous or remove
  data-protection obligations. Public content can also reveal sensitive data.
  Moderation allegations and restricted evidence require separate treatment.

## UK GDPR Appointment Criteria

Based on the [ICO DPO guidance](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/accountability-and-governance/guide-to-accountability-and-governance/data-protection-officers/),
checked on 9 November 2026:

1. **Public authority:** CRAFTSKY LTD is a private company; no public-authority
   function has been identified in the service reviewed.
2. **Core large-scale regular/systematic monitoring:** indexing and ongoing
   social interactions require assessment rather than automatic exclusion.
   The expected initial community, scoped ingestion and limited website
   analytics do not currently establish large-scale monitoring. Thousands of
   active users alone neither triggers nor excludes the requirement. Processing
   is ongoing and geographically broad; volume, non-member coverage and the
   full processing inventory must also inform review.
3. **Core large-scale special-category/criminal-offence processing:** public
   content and safety reports may include these data. No evidence currently
   establishes that processing them on a large scale is a core launch activity.
   This conclusion must be revisited using the DPIA and processing inventory;
   public status is not an exemption.

## Approved Decision

No mandatory DPO appointment has been identified on the initial launch evidence.
Do not voluntarily appoint a formal DPO at this stage. Douglas Todd is the
**privacy lead and data-protection contact**, reachable at
`support@craftsky.social`, and remains responsible for organizing compliance.
This is an owner-approved assessment with moderate confidence, not specialist approval
or a conclusion covering unlimited growth. The separate DPIA and lawful-basis
blockers remain open.

If an updated assessment requires a formal DPO, arrange an independent suitable
appointment, potentially externally. Do not assume the founder can combine
processing decisions and independent DPO oversight without conflict.

Review actual member and non-member coverage, data volumes, analytics/profiling,
private features and sensitive incident processing after three months, and
before material growth, expanded network ingestion, new monitoring or material
sensitive-data processing. Do not wait for a fixed user threshold.

## Sources

- [Companies House](https://find-and-update.company-information.service.gov.uk/company/17482696)
- [ICO DPO guidance](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/accountability-and-governance/guide-to-accountability-and-governance/data-protection-officers/)
- [ICO: publicly available data and transparency](https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/individual-rights/the-right-to-be-informed/are-there-any-exceptions/)
- [Operator details](operator-details.md)

## Change History

| Version | Date | Change | Approval |
|---|---|---|---|
| 0.1 | 9 November 2026 | Initial launch assessment using owner scale estimate, public-data clarification and repository evidence. | Pending owner review |
| 0.2 | 9 November 2026 | Owner approved no formal DPO appointment initially, Douglas Todd as privacy lead/contact, and the recorded review triggers and limitations. | Douglas Todd / approved |
