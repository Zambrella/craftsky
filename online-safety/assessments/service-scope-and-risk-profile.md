# Service Scope and Risk Profile

| Field | Value |
|---|---|
| Assessment owner | Douglas Todd, Founder |
| Approved by | Not approved |
| Version | 0.1 |
| Status | Draft |
| Assessment date | 11 September 2026 |
| Next review | Before launch or any significant change |
| Publication | Internal |

## Service Assessed

CraftSky is a social service for textile and fibre crafters. It is built on the
AT Protocol. Public records are stored in users' personal data repositories and
indexed by CraftSky's AppView. Private-by-intent data, including reports, mutes,
moderation cases, push tokens, saved content, recent-search history, and
operational state, is stored by CraftSky.

This assessment covers the CraftSky application, AppView APIs, public website,
and the CraftSky-specific presentation of indexed AT Protocol records. It does
not claim control over independent PDS operators or other AT Protocol clients.

## Online Safety Act Scope

Provisional conclusion: **CraftSky is a regulated user-to-user service with UK
links.**

Reasons:

- users generate posts, comments, profile content, images, video, likes, reposts,
  follows, and event content;
- other users can encounter that content through feeds, profiles, threads,
  search, notifications, and shared links;
- the service is operated for and available to UK users; and
- no identified Schedule 1 exemption fits the service as a whole.

CraftSky is not known to be on Ofcom's register of categorised services. This
record therefore does not assume Category 1, 2A, or 2B duties. Categorisation must
be checked at each annual regulatory review.

## Current Functionality

| Area | Current position | Safety significance |
|---|---|---|
| Public posting | Text, images, video, links, project metadata, replies, and quote posts | Users can publish illegal or harmful material to an open network. |
| Discovery | The following/home feed is chronological and not engagement-ranked; search and discovery use documented relevance or popularity ordering | Search and sharing can increase reach even without an engagement-ranked home feed. |
| Contact | Public replies, mentions, quotes, follows, likes, and reposts | Enables unwanted contact and public harassment. |
| Private messaging | Not provided | Materially reduces grooming, private abuse, fraud, and hidden-sharing risk. |
| Livestreaming | Not provided | Reduces real-time abuse and intervention risk. |
| Ephemeral content | Not provided | Improves auditability compared with disappearing content. |
| Blocking | Public AT Protocol block with symmetric CraftSky restrictions | Helps users stop interaction in CraftSky but cannot promise network-wide isolation. |
| Muting | Private, one-way CraftSky control | Helps users manage attention without notifying the muted account. |
| Reporting | Signed-in current members can report indexed posts, current-member profiles/accounts, and business events | Reports remain local to AppView and are not forwarded to a PDS or Ozone. The reason taxonomy omits several legally important categories. |
| Moderation | Report-origin cases, warnings, visibility actions, strikes, suspension, appeals, and moderator administration APIs | Technical decisions are auditable and reversible; production staffing and dashboard operation are not demonstrated. Moderation removes visibility from CraftSky rather than deleting PDS records. |
| Commercial model | Permanent product commitment: no ads or paid reach; planned complete free social tier and optional paid convenience/business tools | The current implementation contains no ad-serving system. Commercial disclosures and consumer-subscription duties still apply. |
| Telemetry | Website PostHog uses memory-only analytics with no autocapture or session recording; Flutter and AppView can send sanitized diagnostics to Sentry when configured | The public Privacy Policy does not accurately describe every surface, provider, purpose, or retention period. |
| Minimum age | Launch policy is 16+ | This intentionally permits 16- and 17-year-olds, who are children under the Online Safety Act. |
| Age assurance | None | Younger children can normally access despite the 16+ contractual rule; terms alone do not establish exclusion. |

## User and Content Characteristics

- The intended community includes hobbyists, creators, shops, pattern designers,
  and other craft businesses.
- Craft content is commonly visual and may include people modelling garments,
  children wearing handmade items, external shops, tutorials, pattern materials,
  and health or safety advice.
- The niche topic and chronological product principles reduce some mass-reach
  incentives but do not prevent illegal content or coordinated abuse.
- Federation increases portability and resilience but means public content can be
  copied, cached, or displayed independently of CraftSky.
- There are no active production users according to the repository's current
  project guidance. This limits incident evidence and makes initial ratings
  dependent on design evidence and external risk profiles.

## Protective Design Decisions

- No advertising or paid reach.
- No direct messages or livestreaming at launch.
- Chronological feeds rather than engagement-ranked home feeds.
- Reports are private and report volume does not determine guilt.
- Moderator decisions use append-only evidence and effect records.
- Owner-visible decisions can be appealed.
- Blocks and mutes remain available during suspension.
- Suspended users retain reporting, moderation history, appeal, deletion of their
  posts, events, and business declaration, and the complete CraftSky account-
  deletion flow.
- Moderator visibility actions do not delete users' PDS records.

## Material Gaps

- No approved illegal-content risk assessment or recorded Ofcom Code mapping.
- No dedicated CSEA or urgent-harm intake and escalation workflow.
- No qualifying intimate-image reporting route or demonstrated 48-hour process.
- No emergency or law-enforcement response procedure.
- Report and decision reason codes do not cover all serious policy categories.
- No age assurance, child-specific defaults, or approved children's risk
  assessment.
- Public Terms, Privacy Policy, and Community Guidelines conflict with current
  implementation in several places.
- Operational ownership, service-level targets, evidence handling, training,
  and independent appeal-review capacity are not confirmed.

## Change History

| Version | Date | Change | Author/approver |
|---|---|---|---|
| 0.1 | 11 September 2026 | Initial scope and feature record. | Working draft / not approved |
