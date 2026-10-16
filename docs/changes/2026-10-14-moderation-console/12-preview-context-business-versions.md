# Preview context, business profiles and record versions

Date: 16 October 2026

This follow-up completes the first three gaps identified in the preview review.
It does not change media retry, video captions or production/mobile rollout.

## Delivered

- Post previews include one quoted post, immediate reply parent and thread root.
  Duplicate parent/root cards are combined. Context does not recursively expand;
  each referenced version has its own incident, lifecycle and image-safety check.
  Missing, changed or unsafe references display a placeholder.
- Account cases and the Accounts tab show indexed business types, offerings,
  tagline, hours, service area, location, primary action and product cards with
  prices/images. Business records have their own image-safety gate. Moderation
  reads do not depend on whether a business subscription currently permits public
  display. The interface identifies them as indexed records.
- Version notices compare the case's captured record CID with the current index.
  A match, change, missing record or absent captured version is explicit. Safe
  current content for an edited post/event is shown separately from the unavailable
  captured version. Accounts without captured profile CIDs are explicitly current
  views. Current personal, CraftSky and business record CIDs are listed separately;
  author details are identified as current profile data.

Older content was not retained and cannot be reconstructed from its CID. This
change provides explicit reference/current comparison and safe current previews;
it does not create a historical content archive or invent an old snapshot. Cases,
report attribution, decisions and their captured references remain unchanged.

## API and safety

`GET /v1/admin/accounts/{did}/preview` requires `moderation.read`, the same current
operator session as case preview, and returns `no-store`. Helpers cannot use it.
The existing case preview adds optional `context`, `business`, `businessState`,
`version` and `current` fields. Case/context queries share a read-only,
repeatable-read database snapshot. Each subject still independently gates
restricted incidents, terminal owners and image-safety state before returning
content. No PDS reads, restricted evidence bytes, raw records, new storage,
permission sets or migrations are introduced.

The route/access inventories, diagnostic manifest, OpenAPI contract, capability
map and operator runbook include the account-preview route. The moderation
contract now documents 75 operations.

## Verification

- Red-to-green registered HTTP regressions cover bounded quote/reply context,
  circular references, blocked/changed/restricted context, safe business product
  hydration, helper denial and account lookup without a case.
- Captured/current HTTP regressions cover edited posts and events, absent account
  capture, missing records, blocked current content and terminal owners.
- All 51 frontend tests pass, including quote/reply cards, business profile/product
  rendering in Accounts, and distinct case/current version notices. TypeScript
  and the production Vite build pass, including the Docker frontend stage.
- Complete route and operator-console race suites pass. OpenAPI validation passes;
  all 75 operations match the capability map and 1,071 schema references resolve.
- Rebuilt desktop preview checks show all three conversation cards, an edited
  case with a separate current card, and an Accounts business card with product,
  price and destination. No alerts or horizontal page overflow were observed.
- Final focused regression and Docker verification results are recorded in local
  logs under `/private/tmp/craftsky-preview-gaps-*`. `git diff --check` passes.

The full release gate must still run on the final release commit. Mobile visual
acceptance remains pending as requested. Production has not been changed; no
commit, push or deployment is part of this follow-up.

## Local review

Run the existing benign seed against a migrated development database:

```sh
./scripts/compose-dev exec -T postgres psql -v ON_ERROR_STOP=1 \
  -U craftsky -d craftsky_dev < tools/moderation-console/seed-previews.sql
```

- Quote/reply context: `/admin/#/cases/MOD-65000000-0000-4000-8000-000000000003`.
- Changed case/current view: `/admin/#/cases/MOD-65000000-0000-4000-8000-000000000004`.
- Account/business preview: `/admin/#/accounts/did:example:console-rich-preview`.

The seed only adds benign local data and preserves existing decisions and access.
