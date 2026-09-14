# Document Control Standard

| Field | Value |
|---|---|
| Owner | Douglas Todd, Founder |
| Approved by | Pending approval by Douglas Todd after specialist review |
| Version | 0.2 |
| Status | Working standard |
| Last reviewed | 14 September 2026 |
| Next review | Before publication, then at least annually |
| Publication | Internal |

## Purpose

This standard makes CraftSky's safety and compliance records attributable,
reviewable, and consistent with the product that is actually deployed.

## Statuses

| Status | Meaning |
|---|---|
| Working | Structure or evidence is still being assembled. |
| Draft | Substantive content exists but has not been approved. |
| Ready for owner approval | Evidence and findings are complete enough for the accountable owner to review. |
| Approved | The named approver has accepted the document and any recorded residual risk. |
| Published | The approved version is available to users. |
| Superseded | A newer version applies; retain the old version according to the records schedule. |

No document marked Working, Draft, or Ready for owner approval is an operative
public promise unless an existing public policy already makes the same promise.

## Required Front Matter

Every assessment, policy, and runbook must state:

- title;
- owner and approver;
- version and status;
- effective or assessment date;
- last and next review dates;
- intended audience and publication classification;
- related documents;
- change history; and
- open decisions or evidence gaps.

Markdown tables are the canonical metadata record. Public rendered policies show
only concise version, effective-date, and review information; approval notes,
internal owners, evidence gaps, and blockers remain internal.

## Review Triggers

Review the affected documents before a significant design or operational change,
including:

- adding direct messages, group chat, livestreaming, disappearing content, or
  user location;
- adding a recommender or engagement-ranking system;
- changing who can register or the minimum age;
- introducing or changing age assurance;
- adding new media formats, external embeds, payments, or commercial tools;
- materially changing reporting, complaints, moderation, sanctions, or appeals;
- changing data processors, hosting countries, telemetry, or retention;
- receiving evidence that a risk level or safeguard is no longer accurate;
- a relevant change to law, Ofcom risk profiles or Codes, or ICO guidance; or
- a serious incident, systemic complaint, or control failure.

A further illegal-content risk assessment must be completed before a significant
service change, and compliance must be reviewed as soon as reasonably practicable
after implementation. CraftSky should also perform an annual internal review even
when no trigger occurs.

## Approval and Evidence

The accountable owner must record:

- the evidence considered;
- the risk findings and accepted residual risks;
- the safeguards selected;
- applicable Ofcom Code measures adopted;
- any Code measures not adopted and the alternative measures used;
- implementation tickets for incomplete safeguards; and
- the approval decision and date.

Approval must not erase uncertainty. Assumptions, low-confidence ratings, and
missing usage data remain visible until evidence resolves them.

## Public and Restricted Material

Public policies belong under `policies/`. Internal assessments belong under
`assessments/`. Operational procedures belong under `operations/` and are
restricted by default because they may contain escalation paths, evidence rules,
or security-sensitive details.

The documentation build must use an explicit public allowlist. It must not
publish this entire directory recursively. The public policies are published as
one atomic release at `/terms`, `/privacy`, `/community-guidelines`, `/reporting`,
and `/copyright` only after required routes and P0 controls pass their readiness
checks. Superseded versions are retained internally and provided on request.

## Change History

| Version | Date | Change | Author/approver |
|---|---|---|---|
| 0.1 | 11 September 2026 | Initial standard. | Working draft / not approved |
| 0.2 | 14 September 2026 | Assigns ownership and records canonical-source, publication, metadata, and version-retention decisions. | Douglas Todd / approval pending |
