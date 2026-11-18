# Child safety information and support signposting

| Field | Value |
|---|---|
| Owner | Douglas Todd |
| Prepared | 9 November 2026 |
| Status | Owner approved 9 November 2026; implemented locally, release verification pending |
| Decision requested | Public child/parent safety guide and a brief optional support action after reports/relevant searches |

## Proposed experience

1. A short public **Safety and support** guide, accessible without signing in,
   with sections for young people and parents/carers. Make it easy to find in
   the app/web help navigation and provide it during registration. Use a simple
   illustrated or interactive report/block/mute walkthrough with equivalent
   readable text; ordinary illustrations only, no disturbing examples.
2. Keep the approved report confirmation and its routine estimate. After a
   successful CraftSky report, offer **Find support** linking to the guide's
   appropriate resources. It is optional, adds no required reporting step and
   exposes no report-status screen or report history. If the category cannot
   reliably identify the concern, the guide offers support covering bullying,
   eating problems, self-harm and suicidal feelings rather than silently omitting
   a category. Report/search support is not conditional on collecting age.
3. Show a short, dismissible support notice for relevant suicide/self-harm/
   eating-disorder searches, without blocking a search or treating the person
   as having a condition. Initial explicit English phrase matching must avoid
   ambiguous acronyms and broad craft/body terms such as ED, size, fit or diet.
   Other supported languages and ambiguity need a maintained review record.
4. Do not create a child/adult classifier, diagnosis or risk score. The current
   service has no HEAA adult determination, so offer the relevant support
   without assuming a 16+ declaration establishes adulthood. No additional DOB,
   identity document or health disclosure is requested.
5. Keep Bluesky-only DM reports on their existing provider path. Offer the
   general safety guide from messaging/help navigation without creating a
   duplicate local report, forwarding a transcript or promising CraftSky handles
   the provider report. Any remaining Code/provider dependency stays recorded.

This proposal is separate from the approved copy-only report-confirmation change.
No additional client UI is implemented unless this proposal is approved.

## Proposed short copy

After report (alongside existing confirmation):

> If this has upset you, support is available for bullying, eating problems,
> self-harm and suicidal feelings.
>
> Find support

Search notice:

> Looking for support? You can find help with eating problems, self-harm or
> suicidal feelings. You can still continue your search.
>
> Find support · Dismiss

Guide, young-person section:

> You can enjoy CraftSky without sharing your address, school, phone number or
> private information. You do not have to reply to someone who makes you
> uncomfortable. Use the report action, block or mute an account, or talk to
> someone you trust. If the concern is in a Bluesky message, use its message
> report action. Do not copy or forward harmful images to make a report.

Guide, parent/carer section:

> CraftSky's minimum age is 16, and age is self-declared. Public posts can be
> copied by independent AT Protocol services. Talk together about privacy,
> unwanted contact and where to get help. The guide explains reports, blocking,
> muting and the separate Bluesky messaging route. CraftSky is not an emergency
> service; use local emergency services for immediate danger.

The final guide must be checked against the actual enabled controls and should
not promise isolation across AT Protocol or absolute reporter confidentiality.
Clinical/health support text should point to suitable expert/public-body
resources rather than invent advice or claim expert endorsement of CraftSky.

## Resource review — 9 November 2026

The following records the researched candidates and linking constraints. The
selected public-body resources and implementation evidence are recorded in
[validation.md](validation.md); this does not approve a production release. Use official text links, no copied clinical guidance, logos, embeddings,
tracking parameters or claims of partnership. Confirm UK coverage, youngest-
permitted-user suitability, accessibility and live destinations before shipping.

| Resource | Verified destination / use | Constraint and next check |
|---|---|---|
| NHS self-harm support | `https://www.nhs.uk/mental-health/feelings-symptoms-behaviours/behaviours/self-harm/getting-help/` provides expert/public-body support information and routes, including young people. | NHS terms describe nhs.uk as England-focused. Label regional scope; do not present English service access as UK-universal. Use original links rather than copying clinical content. |
| NHS parent support | `https://www.nhs.uk/mental-health/children-and-young-adults/advice-for-parents/` verified parent/teen mental-health support directory. | Same regional/suitability caveat; obtain suitable equivalents for Scotland, Wales and Northern Ireland before claiming full UK coverage. |
| Childline | Official `https://www.childline.org.uk/get-support/` verified advice/call/chat pathways; NHS/Ofcom identify under-19 support. | Published terms URL returned HTTP 403 on two attempts, including the official footer link. Terms/permission check remains unresolved; do not claim permission or bypass it. No contact made. |
| Beat | Official help-for-myself page verified for eating-disorder support; proposed launch link may use `https://www.beateatingdisorders.org.uk/`. | Published terms section 14 allows qualifying homepage links and restricts deep linking without further permission. Use homepage under those conditions; do not ship the researched deep link as though authorized. Confirm applicable terms/content standards and regional/age suitability. |
| Samaritans | Official contact page verifies free telephone support on 116 123, day/night. | Published legal terms 3.2 require notification/approval for website links. No approval is recorded or contact authorized; keep a Samaritans website link on hold or use an approved suitable public-body resource. No live-chat/email promise copied. |

Sources: [Ofcom issued children's Code, PCU F1/F3/F5](https://www.ofcom.org.uk/siteassets/resources/documents/consultations/category-1-10-weeks/statement-protecting-children-from-harms-online/upcoming-codes-of-practice/issued-protection-of-children-code-of-practice-for-user-to-user-services.pdf?v=427439),
[NHS terms](https://www.nhs.uk/our-policies/terms-and-conditions/),
[Childline support](https://www.childline.org.uk/get-support/),
[Beat terms](https://www.beateatingdisorders.org.uk/terms-conditions/),
[Samaritans legal terms](https://www.samaritans.org/legal/).

PCU F3/F5 resource suitability and third-party linking conditions must be checked
before publication. No service is contacted or implied to have reviewed/endorsed
CraftSky. Approval to build this experience would not itself authorize sending
messages to those organizations or inventing permission.

## Privacy and implementation boundaries

- Do not send a report reason, search term, DID, case reference, account/health
  inference or other private context to an external support destination.
- Do not add analytics events, persisted support-search tags, new query logs or
  inferred-health profiles. Maintain privacy-safe diagnostics and ordinary
  reporting behavior; review existing search persistence separately rather than
  falsely claiming raw searches are never stored anywhere.
- Use the existing native/web navigation and safe external-link handling; make
  the off-site boundary clear. Avoid external referrer/query leakage and
  third-party preloads in the resource UI. No auto-call or auto-message.
- Keep report submission and ordinary search usable if a support resource fails;
  no false report failure, hidden retry or accidental duplicate submission.
- The guide explains existing controls. It does not promise new comment controls,
  age assurance, provider message removal, a universal scanner or 24h CraftSky cover.

## Verification if approved

Check report success/failure/cancel/retry separately; relevant and neutral craft
searches; ambiguous acronyms; immediate dismiss/revisit; keyboard/screen reader
and large text; signed-in/guest access; registration and child/parent content;
UK resource availability/linking terms; and serialized external URLs/diagnostic
canaries. Validate native and web separately. No real report/message, harmful
material, clinical experiment or disclosure to a support provider is needed.

The children's working ratings, remaining privacy/age/provider questions,
whole assessment, policy publication and exact-release evidence remain separate.
