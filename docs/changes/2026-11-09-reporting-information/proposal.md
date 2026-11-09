# Post-submission report confirmation

| Field | Value |
|---|---|
| Owner | Douglas Todd |
| Updated | 9 November 2026 |
| Status | Owner narrowed scope to brief post-submit copy; implemented locally, release pending |
| Scope decision | No reporter-facing status screen, report list or status lookup |

## Authorized scope

Douglas Todd said he does not want UI for a reporter to see report status at
this stage, and is happy to add brief copy after submission describing the
process and estimated timelines. This supersedes the wider implementation
proposal previously in this file. It does not approve new contact preferences,
reporter correspondence automation, private APIs or timing/preference storage.

The shared native/web confirmation is:

> Report received. We’ll review it and take action if needed. We aim to decide
> routine reports within ten UK business days. Urgent safety concerns take priority.

The ten-day figure is the proposed routine estimate carried into the authorized
copy. It is expressed as an aim, not a guarantee, a statutory deadline or a
claim about measured response performance. Urgent/specialist handling and the
existing shorter targets take priority. The approved manual acknowledgement
procedure is unchanged. This step adds no new appeal-decision promise.

## Implementation

- Update the existing native shared success localization, used after accepted
  post/comment/reply, profile, profile-link and business-event reports.
- Use one shared web confirmation constant for accepted post, profile,
  profile-link and business-event submissions.
- Retain existing success-only handling. A failed submission stays in its
  existing error/retry flow and does not show this confirmation.
- Longer informational notifications get reading time based on length, between
  the existing four-second minimum and a thirty-second maximum. Short messages
  keep their existing timeout; sticky warnings/errors and web hover/focus pause
  behavior are preserved. The new copy receives roughly nine seconds.
- Keep Bluesky DM report confirmations separate. No CraftSky handling-time
  promise is applied to provider reports.

There is no status page, report history, status API, tracking UI, backend
migration, preference/contact change, extra pre-submit step, reporter-email
feature or production action in this change.

## Remaining reporting assessment

This narrow step improves the post-submit explanation. It does not close every
ICU D3–D6/D8 finding in `online-safety/registers/ofcom-control-mapping.md`.
Pre-submit disclosure information, relevant acknowledgement/outcome content,
communication choices, monitored targets and operational evidence remain
separate review items. A report-status dashboard is not being treated as the
required way to address them. The earlier broad proposal is not authorized.

Local check results and release boundaries are recorded in `validation.md`.
