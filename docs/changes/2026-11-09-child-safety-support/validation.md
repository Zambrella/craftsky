# Safety information and support signposting — implementation record

Owner: Douglas Todd. Implementation approved 9 November 2026 by the owner's
“approved” reply to the prepared proposal. Implemented locally on that date.
This approval covers this feature; it does not approve the working children's
risk ratings, whole assessment, policy publication or final release.

## Result

- A public guide for young people and parents/carers, with illustrated
  report/block/mute steps and equivalent readable text. Available without an
  account at `/safety` in native/web clients, and prepared as `web/safety.html`
  for the public site. Native welcome, onboarding, settings and messaging link
  to it; web onboarding, settings, messaging and shared information navigation
  link to it. Native signed-out routing and signed-in settings navigation were
  exercised locally; onboarding/provider operation is not asserted live.
- Successful CraftSky reports retain the approved process/timeline confirmation
  and offer optional **Find support**. There is no extra required reporting
  step, report-status/history screen, new endpoint or reporter email automation.
- Submitted relevant text/hashtag searches show an optional dismissible notice.
  Explicit English terms are matched locally with word boundaries and normalized
  spaces/hyphens; ambiguous acronyms and neutral size/fit/diet/craft terms do
  not trigger it. Dismissal is screen-local and resets for a different query.
  Results and search requests remain available and unchanged.
- The guide explains the separate Bluesky route. DM report controllers,
  submission/confirmation and alert delivery have not been changed by this
  feature. No duplicate CraftSky report or transcript forwarding was added.

`shared/safety-support.json` is the single public-only source. Run
`python3 scripts/generate-safety-guide.py` after changing it, then
`python3 scripts/generate-safety-guide.py --check`. It generates native guide
content, web guide content and public HTML together. The new public guide is
separate from the five-policy publication manifest and does not approve or
publish the existing draft policies.

## Resources and privacy review

The eight selected links are public-body information/support routes: GOV.UK
bullying guidance; NHS England eating-disorder, self-harm, urgent mental-health
and parent/carer guidance; NHS inform Scotland urgent help; NHS 111 Wales mental
health and wellbeing; and nidirect Northern Ireland crisis guidance. The
canonical URLs are in the shared source. Official destinations were checked on
9 November 2026, including correcting the eating-disorder path and verifying the
non-www NHS Wales destination. NHS England service access is labelled separately
from the other nations; outside-UK readers are directed to local services.
The links offer information and routes appropriate to the permitted 16+ audience,
with parent/carer material, rather than promising every linked intervention is
suitable for every person. No clinical text, logos or external embeds are copied.

NHS terms: https://www.nhs.uk/our-policies/terms-and-conditions/
Wales official destination: https://111.wales.nhs.uk/encyclopaedia/m/article/mentalhealthandwellbeing

The charity candidates remain unshipped: Childline terms could not be verified;
Samaritans linking approval is not recorded; Beat's conditional homepage option
was not needed for this selected public-body list. No organization was contacted,
and no endorsement, partnership or permission is implied.

External destinations are fixed HTTPS URLs without query, fragment, account,
search, reason or case routing. Web links use `noopener noreferrer` and an
explicit no-referrer policy; public HTML also sets no-referrer. Native uses the
existing external-browser confirmation and launcher. No support-site preloads,
new query logs, analytics events, health classifications, age collection or
persisted support-search attributes were introduced. Existing ordinary search
history behavior is unchanged; this record does not claim searches were never
stored by the existing product.

## Local verification

| Check | Result |
|---|---|
| `flutter gen-l10n`; `dart run build_runner build` | PASS; localized labels and typed public route generated. Unrelated generator churn removed. |
| `flutter analyze --no-pub` | PASS, no issues. |
| Native focused safety, reporting widgets, DM reports, settings model and settings navigation tests | PASS, 53 tests; 5 existing subscription-gated skips. Includes enlarged-text guide, dismiss/reset, neutral/relevant matching and production-router help navigation. |
| Additional signed-out public-router/safety tests | PASS, 5 safety tests in the final run, including actual signed-out router access without account initialization. |
| Web focused safety, toast and settings-presentation unit tests | PASS, 13 tests. Native-settings tests also passed in the earlier 20-test focused run. |
| Web `npm run check`; `npm run build` | PASS, zero Svelte errors/warnings. Build prepares private Sentry maps without uploading. Existing large-chunk advisory remains. |
| Desktop/mobile Chromium guest reporting, safety and search flows | PASS, 12 tests against local mock AppView. Success/support links, cancellation/failure/retry, unchanged guest search, dismiss/revisit, public guide, eight safe links, no third-party preload, 320px/200% text and axe accessibility scan. |
| `python3 scripts/generate-safety-guide.py --check` | PASS; generated content matches the shared source. |
| `python3 scripts/test_online_safety_readiness.py` | PASS, 7 tests. |
| `git diff --check` | PASS. |

DTD discovery found no running debug app. No CraftSky app could be hot reloaded.
Widget checks are not physical iOS/Android or assistive-technology user testing;
automated axe/large-text checks are not a study with children or carers. Browser
flows used local mocks, without real reports, messages, harmful media, support
provider contact, production changes or publication.

## Remaining evidence and maintained scope

Recheck the selected resource destinations before release and when content or
availability changes. Matching is English-only: other-language coverage and
ambiguous/contextual search coverage remain review items. Do not claim complete
multilingual or semantic detection. Confirm launch navigation, report support,
provider disclosures and safe links on the exact deployed clients, including real
native devices and manual keyboard/screen-reader checks.

This supplies local F1/F3/F5 implementation evidence, not final Code compliance
or closure of CHILD-02. Remaining source/provider-removal, age/privacy, risk-rating,
training/capacity, policy-publication and exact-release decisions remain in the
children's assessment and control mapping.
