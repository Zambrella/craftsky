# CraftSky Privacy Policy

| Field | Value |
|---|---|
| Privacy owner | Douglas Todd, Founder |
| Approved by | Pending approval by Douglas Todd after specialist review |
| Version | 0.3 |
| Status | Draft; not effective |
| Proposed effective date | 1 October 2026, subject to launch readiness |
| Last reviewed | 14 September 2026 |
| Next review | Before publication, then at least annually |
| Publication | Public after factual, DPIA, and legal approval |

> This draft is not effective and must not be published until the outstanding
> launch requirements in the internal compliance register are complete.

## 1. Who We Are

The company being established under the assumed name **CraftSky** is the
controller of personal data it processes to operate the CraftSky website, apps,
safety systems, and private features. Company number: **[TO BE CONFIRMED]**.
Country of registration: **[TO BE CONFIRMED]**.

Registered postal address: **[TO BE CONFIRMED]**

Privacy, rights-request, and security contact: `support@craftsky.social`. Douglas
Todd initially owns this inbox and checks it on UK business days. It is not an
emergency-response service.

CraftSky is built on the AT Protocol. PDS operators and independent AT Protocol
services may process public AT Protocol records for their own purposes. They may
be separate controllers. This Policy describes CraftSky's processing, not every
independent service on the network.

## 2. The Important AT Protocol Privacy Boundary

AT Protocol repository records are public by design. Depending on the feature,
they may include your DID, handle, profile, posts, replies, images, video, likes,
reposts, follows, blocks, business information, and events.

Public records:

- are stored by your PDS provider;
- can be distributed through AT Protocol infrastructure;
- are indexed by CraftSky so the service can display and search them; and
- can be retrieved, cached, moderated, or displayed by independent services that
  CraftSky does not control.

Deleting a source record or your CraftSky membership does not guarantee deletion
of every independent copy already obtained by another service.

Private CraftSky data, including reports, moderation cases, mutes, saved content,
recent-search history, push settings, and operational state, is kept by CraftSky
and is not written to your public repository.

## 3. Information We Process

### Information you provide or create

- Waiting-list email address and communication preferences.
- Support messages, feedback, privacy requests, complaints, and appeal
  correspondence.
- Report reasons and details, including information submitted by a representative
  or affected non-user when public reporting is implemented.
- Private saved-content, mute, recent-search, draft, scheduling, import, profile-
  customization, profile-pin, language, notification, and device preferences
  where the relevant feature stores them.
- Instagram verification and migration information where that feature is enabled,
  including your username, account identifiers, verification information,
  account-link status, messages needed to complete verification, and suggested
  connections. Some information is hashed or temporary.
- Business classification or verification information and subscription
  information where applicable.
- Age-assurance result and audit information if CraftSky later introduces age
  assurance. CraftSky's intended design is to receive the minimum threshold result
  needed rather than identity-document images or an exact date of birth where a
  proportionate alternative is available.

### Account and technical information

- DID, handle, PDS service, profile references, membership state, and account-
  lifecycle state.
- CraftSky session identifiers and server-held OAuth credentials needed to act on
  your PDS instructions. CraftSky does not receive your PDS password.
- Device identifiers used for account security and notifications, push tokens,
  app version, platform, language, and basic device information.
- IP address, user agent, request time, route, status, security events, and
  redacted diagnostic information in server and infrastructure logs.

### Public AT Protocol information

CraftSky receives public records and identity updates through AT Protocol
services and may backfill public repositories. This includes records created in
CraftSky and compatible records created through another client.

CraftSky may therefore process public information about people who have not
created a CraftSky membership. The source is the person's PDS and the public AT
Protocol distribution infrastructure.

### Reports and moderation

CraftSky processes reporter identity, subject identifiers, content snapshots or
references, report details, case evidence, moderator decisions, sanctions,
appeals, correspondence metadata, and audit events. A report is an allegation,
not a finding.

Safety material may reveal special-category data, criminal allegations, or highly
sensitive information even when CraftSky did not request it. CraftSky restricts
access and handles it only under an applicable UK GDPR Article 9 condition,
Article 10/DPA 2018 condition, legal obligation, or other lawful basis.

### Website analytics

With the visitor's prior consent, the public website uses PostHog to collect page
views, page exits, and selected interaction events, including waiting-list clicks
and opening the AT Protocol diagram. CraftSky does not use PostHog in the app.

### Error monitoring and diagnostics

CraftSky uses Sentry to detect and diagnose errors in the app and service. Sentry
may receive error messages, diagnostic logs, stack traces, sampled performance
traces, and limited technical context. CraftSky takes steps to exclude content,
credentials, and tokens, but diagnostic data is not anonymous.

### External content and providers

Loading an external link, video, PDS resource, or other embedded service may send
the receiving provider your IP address, user agent, and request details. YouTube
playback loads only after you choose to allow it. You can ask the app to remember
that choice and can revoke it in settings.

## 4. Why We Process Information and Lawful Bases

| Purpose | Information | UK GDPR basis |
|---|---|---|
| Provide membership, sessions, requested features, and PDS actions | Account, session, OAuth, device, preferences, private feature state | Contract; legitimate interests where processing is not necessary for the user's contract |
| Index and display public AT Protocol data | Public records, identities, media references, social graph | Contract for members; legitimate interests for public-network indexing and non-member data |
| Keep the service secure and reliable | Logs, IP, device, security events, diagnostics | Legitimate interests; legal obligation where applicable |
| Receive reports, moderate, protect users, and handle complaints | Reports, public content, case evidence, decisions, correspondence | Legal obligation and legitimate interests; vital interests or recognised legitimate interests may apply in narrowly defined safeguarding or emergency circumstances |
| Report CSEA and respond to lawful authorities | Required content, metadata, associated data, report references | Legal obligation, with an applicable condition for sensitive or criminal-offence data |
| Operate the waiting list and optional updates | Email and preferences | Consent for optional marketing; legitimate interests or contract only where appropriate for a requested service message |
| Understand use of the public website | Limited PostHog events and technical data | Consent |
| Respond to support and rights requests | Contact, request, identity-verification, and response records | Legal obligation, contract, and legitimate interests as applicable |
| Administer paid plans | Account, entitlement, transaction references, support | Contract and legal obligation |

CraftSky does not use consent where the processing is actually necessary to
provide the service contract. Where we rely on consent, you may withdraw it
without affecting processing that was lawful before withdrawal.

CraftSky's legitimate interests include operating and securing a federated social
service, displaying relevant public-network records, preventing abuse, protecting
members and the public, and establishing or defending legal claims. CraftSky
balances those interests against the nature of the information, reasonable
expectations, risks, safeguards, and available rights. You may object as described
in section 9.

## 5. Who Receives Information

CraftSky uses service providers and interacts with independent network operators.
The known inventory includes:

| Recipient/category | How they are involved |
|---|---|
| Brevo | Waiting-list form and email delivery |
| PostHog | Consent-based, limited public-website analytics in the EU |
| Sentry | App and service errors, logs, and sampled traces in the EU |
| Cloudflare | Website delivery, bot protection, security, and network infrastructure |
| Hetzner | Intended EU hosting and database infrastructure |
| Amazon S3 in the AWS Europe (Frankfurt) Region | Temporary private storage for scheduled-post media |
| PDS and AT Protocol infrastructure operators | Store and distribute public records and fulfil user-directed requests; these operators are usually independent controllers |
| Bluesky video service | Process a user-directed video upload and store the processed result on the user's PDS |
| Meta/Instagram | Optional account linking, verification, migration, and related messages |
| Apple and Google | App distribution, push delivery, and in-app subscriptions; they generally act under their own terms and privacy policies |
| YouTube and other external-content providers | Deliver external content after the user chooses to load it; these providers generally act independently |
| Regulators, courts, emergency services, and law enforcement | Receive information where permitted or required by law |

CraftSky does not sell or rent personal information, share it for cross-service
behavioural advertising, or use tracking pixels or tracked links in its email.
CraftSky does not serve advertising.

We require processors to act under appropriate data-processing terms and review
their subprocessors and material changes.

## 6. International Transfers

Some providers, PDS operators, users, or public AT Protocol services may be
outside the UK. Public records can be retrieved globally by design.

Where information is transferred from the UK to a country without applicable UK
adequacy regulations, CraftSky uses an appropriate safeguard such as the UK
International Data Transfer Agreement or Addendum, or another lawful mechanism,
and completes any required transfer-risk assessment. Contact
`support@craftsky.social` for information about safeguards relevant to your data.

## 7. Retention

CraftSky keeps personal data only as long as needed for the stated purpose,
security, disputes, or a legal requirement. Maximum periods may be shorter where
information is no longer needed.

| Data | Maximum period or trigger |
|---|---|
| CraftSky sessions | Normally up to 30 days without use and no more than 180 days |
| Unclaimed scheduled media | 24 hours |
| Failed scheduled content and related cleanup records | 30 days |
| Instagram account-linking, verification, and migration data | Between 24 hours and one year, depending on whether the data supports a short-lived link, migration rollback, fraud prevention, or an audit record |
| Active strikes | Count toward standing for 12 calendar months unless reversed |
| Ordinary reports, cases, moderation evidence, complaints, and appeals | Two years after final closure, unless a specific legal hold applies |
| CSEA report reference | Five years from issue where the 2026 Regulations require it |
| Required CSEA content, submitted information, decisions, and associated-user data | One year from the report date where the 2026 Regulations require it |
| Waiting-list data | Until withdrawal, or 90 days after general availability unless the person separately chooses ongoing email |
| Server and infrastructure logs | 30 days |
| PostHog events | 12 months |
| Sentry errors, logs, and traces | 90 days |
| Support, complaints, and rights requests | Two years after final closure |
| Push tokens and device records | Until invalidation, sign-out or deletion, or 90 days after the device becomes stale |
| Backups | Expire within 30 days |

CraftSky uses narrow, case-specific, access-restricted legal holds where required.
Held evidence survives account deletion only for the applicable legal period and
is then deleted. Unrelated account data is still deleted normally.

## 8. Account Deletion

The permanent CraftSky deletion flow requires you to sign in again and confirm
the account you want to delete. Once accepted, it asynchronously removes your
CraftSky membership and private CraftSky data, except limited records needed to
complete deletion, prevent reactivation, or meet legal duties. It also deletes
your CraftSky posts, likes, reposts, business events, business profile, and
CraftSky profile from your PDS.

The process does not:

- delete or deactivate your DID or PDS account;
- delete records in other applications' collections;
- directly delete uploaded media from PDS storage;
- guarantee immediate removal from CraftSky while deletion updates are still
  being processed; or
- delete copies held independently by another AT Protocol service.

The confirming installation attempts to clear its local account data. Failed
local cleanup may require manual deletion or an app-storage reset. Other
installations remove or invalidate session state when they next contact CraftSky;
locally stored drafts and files may remain until separately deleted.

CraftSky may preserve a narrowly limited restricted record where law requires it.
That exception does not permit indefinite retention or continued product use.

## 9. Your Data Protection Rights

Depending on the circumstances, UK data-protection law gives you rights to:

- be informed about processing;
- access your personal data;
- correct inaccurate data;
- erase data;
- restrict processing;
- receive portable data;
- object to processing;
- withdraw consent; and
- obtain safeguards concerning applicable automated decisions.

Contact `support@craftsky.social`. We normally use fresh account authentication
to verify a member's identity and may request proportionate alternative evidence
from a non-member. Portable account data will normally be supplied as machine-
readable JSON. We will respond without undue delay and
normally within one month after the applicable period begins. We may extend that
period by up to two further months where necessary because of the complexity or
number of requests; if so, we will tell you within the first month and explain
why. The period may begin later or be paused where the law permits while we obtain
necessary identification, clarification, or payment of a lawful fee. We will
explain any lawful limitation.

People who have not joined CraftSky may also use this address to object to the
indexing of compatible public AT Protocol records about them. CraftSky will
verify the relevant identity where proportionate and assess the request under
applicable law.

These rights apply to data CraftSky controls. CraftSky cannot answer for an
independent PDS or AT Protocol service, but it must still act on copies and private
data that it controls.

## 10. Complaints

The data-protection complaint contact is `support@craftsky.social`.
CraftSky aims to acknowledge it within five UK business days and will do so within
30 days, investigate it appropriately,
provide progress information, and communicate the outcome without undue delay as
required by Data Protection Act 2018 section 164A.

You may also complain to the UK Information Commissioner's Office. See
[Make a complaint to the ICO](https://ico.org.uk/make-a-complaint/). You may have
the right to seek a judicial remedy.

## 11. Children and Age Assurance

CraftSky is for people aged **16 and over**. Because 16- and 17-year-olds are
children under the Online Safety Act and UK data-protection framework, CraftSky
treats the service as likely to be accessed by children and applies the resulting
safety and Children's Code assessment duties.

CraftSky does not currently collect a date of birth or operate age assurance. A
contractual age statement alone does not reliably prevent younger children from
accessing the service.

Before introducing age assurance, CraftSky will complete a DPIA, explain the
provider and method, choose a proportionate and accessible approach, minimize
inputs and outputs, prevent unrelated reuse, set retention, test accuracy and
fairness, and provide a challenge route. CraftSky intends to retain a threshold
result such as `16 or over` rather than an identity document where possible.

## 12. Cookies, Device Storage, and Similar Technologies

The public website uses the limited analytics described in section 3 only after
the visitor opts in. CraftSky does not use analytics cookies or session recording,
and honours the browser's Do Not Track signal. Consent can be withdrawn using the
website's privacy control.

The app uses device storage for sign-in details, account and profile information,
preferences, caches, account files, and local drafts. It also keeps an app-
installation identifier for account security; this remains after sign-out. The
Instagram import tool stores progress information in your browser so an import
can be resumed or rolled back. You can remove local app or browser data using the
relevant app, browser, or device controls. Account deletion does not guarantee
removal of data held locally on every device.

CraftSky does not use advertising cookies or cross-service advertising trackers.

## 13. Security and Breaches

CraftSky uses measures intended to protect the data it controls, including
encryption, access controls, data minimization, diagnostic-data filtering, and
software maintenance. No system is completely secure.

CraftSky must record and assess personal-data breaches. Unless a breach is
unlikely to result in a risk to people's rights and freedoms, CraftSky will notify
the ICO without undue delay and, where feasible, within 72 hours after becoming
aware of it. If a breach is likely to result in a high risk, CraftSky will also
notify affected people without undue delay unless a statutory exception applies.

Report a suspected security issue to `support@craftsky.social`. This inbox is
checked on UK business days and is not an emergency-response service.

## 14. Automated Decisions

Moderation consequences are decided by authorized moderators. CraftSky does not
currently use solely automated decisions that produce legal or similarly
significant effects. If that changes, CraftSky will update this Policy and provide
the safeguards required by law before using the new process.

## 15. Changes to This Policy

CraftSky may update this Policy to reflect service, provider, or legal changes.
The published version will state its effective date and version. CraftSky will
give proportionate notice of material changes and request consent where the law
requires it.

## Change History

| Version | Date | Change | Author/approver |
|---|---|---|---|
| 0.1 | 11 September 2026 | Initial repository draft; records the contractual 16+ eligibility rule, CraftSky controller name, and `craftsky.social` contact domain. Age is not currently verified. | Prepared with repository review; not approved |
| 0.2 | 14 September 2026 | Records decisions on contacts, analytics consent, providers, retention, rights requests, legal holds, and publication naming. | Douglas Todd / approval pending |
| 0.3 | 14 September 2026 | Records CraftSky as the assumed company name, adds registration-detail placeholders, and identifies Amazon S3 in AWS Frankfurt for scheduled-media storage. | Douglas Todd / approval pending |
