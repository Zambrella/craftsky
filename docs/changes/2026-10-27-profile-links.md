# Shared profile links

Implemented against Bluesky social-app revision
`0d30bbead1df8409fac03370e4097bb8f33000fa` and ADR 022.

The PDS stores `app.bsky.actor.link` records and the ordered strong references
in `app.bsky.actor.profile/self`. CraftSky defines no competing link lexicon.
Pinned upstream schemas are in `appview/cmd/lexgen/external/profile-links` and
participate in the runtime schema-catalog synchronization. The policy lists
and provider marks retain their upstream attribution.

The editor supports ten links, optional 30-grapheme titles, missing-scheme
normalization, HTTPS and shortener checks, duplicate detection, drag ordering,
keyboard/screen-reader ordering, editing and removal. Changes remain local
until profile Save. Favicons are advisory and uploaded through the existing
image capability flow. The record itself retains Bluesky's 40-grapheme /
320-byte title and 100,000-byte PNG/JPEG/WebP icon constraints.

Public profiles show a two-line wrapping row with expandable overflow,
recognized support-platform logos/green pills, direct opening, hover URL
tooltips, and long-press destination/copy/report actions. Profile-link reports
use the existing private intake and case adjudication system. Operator previews
show the captured link version without automatically fetching its destination
or rendering its icon.

Writes use the existing durable compound command and repository CAS. New
record keys/timestamps and request bodies stay fixed across retries. Unknown
record extensions and self-labels survive edits. The approved deletion
exception is confined to explicitly removed owner link records referenced by
the authoritative profile; deletions occur atomically with reference removal.

Editor records remain available during image scanning and moderation. Public
presentation and optimistic acknowledgements retain the clearance boundary.
Terminal-owner cleanup includes the new projection table. Account deletion
still leaves the shared `app.bsky.*` namespace on the owner's PDS, consistent
with its existing CraftSky-only deletion scope.

## Release and live checks

- Apply migrations 102 and 103 through the existing release path.
- The Tap collection-filter and OAuth scope changes in `render.yaml` require the normal manual
  Blueprint review/sync; auto-sync remains disabled.
- Existing sessions must sign in again to approve `app.bsky.actor.link` create,
  update and explicit-delete authority. After activating the new scopes, the
  supported `/app/cli --env prod sessions revoke --all` command can force this
  for every existing non-terminal account through the normal logout-all flow.
  No production session revocation was performed during implementation.
- After the new AppView and Tap filter are active, request the existing durable
  `pds_reconcile` jobs for already-tracked accounts to import historical profile
  links. New onboarding and later repository reconciliation use the expanded
  registry automatically. Do not rely on a filter restart alone to replay old
  records. No production reconciliation was run during implementation.
- Verify on a disposable live account: edit/save/reopen in CraftSky, view the
  same links in Bluesky, reorder in Bluesky and observe CraftSky convergence,
  remove in CraftSky and verify only the explicitly removed link record is gone.
- Physical-device browser opening, long press, drag gestures, and real favicon
  scanning remain separate from host widget/database validation.

## Local validation

- `./scripts/appview-check`: all release gates passed, including the complete
  Go test and race suites, packaged builds, vulnerability checks, console
  boundary smoke checks and pinned Tap handshake. Log:
  `/tmp/craftsky-profile-links-gate.log`.
- Final focused Go race tests passed for the changed API, indexer, compound
  commands, link policy and favicon paths; changed-package `go vet` passed.
- Flutter profile, business-boundary and moderation suites: 371 tests passed.
  The final editor/save interaction rerun passed all eight tests.
- `fvm dart analyze`: no issues found.
- Operator console type checking and all 51 tests passed.
- `render blueprints validate --output json`: `valid: true`.
- Real-widget host screenshot: `/tmp/craftsky-profile-links.png`.

These checks did not deploy infrastructure, mutate production accounts or
verify interoperability on a live Bluesky account or physical device.
