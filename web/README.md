# CraftSky landing page

Static public site served at https://craftsky.social.

## Contents

- `index.html` — landing page, nine sections
- `waitlist.html` — focused waiting-list signup page at `/waitlist`
- `privacy.html`, `terms.html`, `community-guidelines.html`, `reporting.html`,
  `copyright.html` — generated policy drafts; approval remains recorded in the readiness manifest
- `styles.css` — all styles, design tokens copied from `../docs/design/colors_and_type.css`
- `main.js` — waiting-list modal + PostHog event tracking
- `assets/` — favicon, logo, atproto mark, paper-grain texture
- `robots.txt` — allow all crawlers

## Local dev

From the repository root, start the local server (no build step needed):

```bash
python3 web/test/server.py
```

Visit http://127.0.0.1:4173/. The server maps clean URLs such as `/privacy`
to their `.html` files. Stop it with Ctrl+C. Python's built-in
`http.server` does not resolve these URLs and will return 404 for site links.

## Check for token drift

When the design system (`../docs/design/colors_and_type.css`) changes, re-copy the `:root` block into `styles.css`. Check for drift with:

````bash
diff <(sed -n '/^:root {/,/^}/p' styles.css) \
     <(sed -n '/^:root {/,/^}/p' ../docs/design/colors_and_type.css)
````

Expected: no output. If there's a diff, re-copy from the source file.

## Browser tests

```bash
npm ci
npx playwright install chromium
npm run test:consent
```

The suite uses clean browser contexts for every public route and verifies network,
script, cookie, local-storage, and session-storage behavior before consent, after
denial, after grant, and under Do Not Track.

## Build and deploy

The existing **Cloudflare Worker with static assets** is `craftsky-landing`,
serving `https://craftsky.social`. Production releases come from Git branch `main`. Deployments
are run locally; merging a PR does not automatically publish the website.
The non-secret account ID, Worker name and asset settings live in `wrangler.json`;
`deploy.json` contains the production Git branch and public origin. The Worker
has no custom runtime code. HTML handling supports clean routes such as `/privacy`.
The website deploys independently of AppView and Flutter releases.

### One-time setup

Use Node.js 22 or newer, npm, Python 3.10 or newer, Git, and `just`:

```bash
just web-setup
just web-login
just web-whoami
```

Run these commands from the repository root. `web-setup` installs the locked
Wrangler/browser-test dependencies and Chromium. `web-whoami` shows your account
IDs. Deployment uses the account ID checked into `wrangler.json`, overriding any
inherited `CLOUDFLARE_ACCOUNT_ID`; no account environment variable is needed.
The scripts do not load the repository's backend `.env.local` or write Cloudflare credentials.
Wrangler manages the local OAuth login. Alternatively, supply
`CLOUDFLARE_API_TOKEN` in your shell with Account → Workers Scripts → Edit
permission scoped to the intended account. Never commit a token.

### Build and check

```bash
just web-build
just web-check
```

`web-build` creates a fresh `web/dist/` with the homepage, waitlist, all five
policy pages, JavaScript, CSS, robots.txt and image/font assets. It excludes
packages, tests, email templates, documentation, configuration and policy
approval metadata. New public pages must be added to `ROUTES` in
`scripts/cloudflare_pages_build.py`; other root files must be added to
`PUBLIC_FILES`. Hidden asset files and unsupported asset extensions are excluded.

`site-release.json` contains the Git commit, dirty-checkout flag and SHA-256
hashes of the public files. This metadata is public. Build output and local
deployment receipts are Git-ignored. `web-check` runs the build/deployment unit
tests and the consent browser suite **against the built artifact**. Existing
`web-test-consent` still tests the source website.

### Preview

```bash
just web-preview
```

This runs `web-check`, confirms the existing Worker is accessible, and uses
`wrangler versions upload` to upload a new version without assigning it live
traffic. The `preview-<current-branch>` alias is capped to Cloudflare's DNS length
limit. Uncommitted changes are allowed and recorded in the version tag and local
receipt. The script verifies the unique `workers.dev` Version URL and records it
in `web/.deployments/<version-id>.json`. Review the preview before merging.

Version URLs must be enabled in Cloudflare → Workers & Pages → `craftsky-landing`
→ Settings → Domains & Routes → Version URLs (sometimes labelled Preview URLs).
The script preserves the existing setting. If disabled, it reports how to enable
it and stops without promoting the uploaded version to live traffic.

### Production

After merging the reviewed PR, check out the updated `main` and run:

```bash
git switch main
git pull --ff-only
just web-deploy
```

The command requires a clean checkout equal to freshly fetched `origin/main`,
confirms the existing Worker is accessible in the configured account, runs
`web-check`, and uploads the complete artifact as a Worker version. It verifies
every published file at that version's URL, including clean `/privacy` and
`/terms` routes, before running `wrangler versions deploy <version-id>@100`.
It confirms the deployed version and percentage via deployment discovery, then
verifies all files at `https://craftsky.social`. These commands preserve the
existing custom domains/routes and do not run `wrangler triggers deploy`.

Keep the checkout and build output unchanged while deployment runs. Verification
retries while changes propagate. If Version URL verification fails, production
promotion is skipped. If public-origin verification fails after promotion, the
version may already be live: inspect the printed receipt before retrying or
rolling back. There is no automatic rollback. Receipts record the commit, dirty
flag, file hashes, Worker version ID, production deployment ID (when promoted),
URL and verified origins.

The checked-in policy HTML is generated from `online-safety/policies/` via
`just policy-artifact`. Generate and review those files before deploying policy
changes. Deployment copies their existing content without approving drafts or
changing their effective dates. `web/policy-publication-manifest.json` continues
to track approval independently. Before full public launch, run
`just online-safety-readiness` and complete the approvals/evidence in the gap
register. Open readiness gaps do not prevent beta website publication.

### Rollback

In Cloudflare → Workers & Pages → `craftsky-landing` → Deployments, select the
previous known-good version and use the rollback/deployment control to assign
it 100% of traffic. Alternatively, from `web/`, use the pinned CLI:

```bash
npm exec -- wrangler versions deploy <previous-version-id>@100 --config wrangler.json
```

This restores the complete site's version while preserving custom domains.
Verify the homepage and policy routes and compare
`https://craftsky.social/site-release.json` with the intended previous release
(older dashboard-uploaded versions may not have this file). Keep the deployment
ID/receipt as evidence and reconcile reverted website changes in Git before the
next production release.

Cloudflare references: [static assets](https://developers.cloudflare.com/workers/static-assets/),
[Version URLs](https://developers.cloudflare.com/workers/versions-and-deployments/version-urls/),
[versions and deployments](https://developers.cloudflare.com/workers/versions-and-deployments/),
[rollbacks](https://developers.cloudflare.com/workers/versions-and-deployments/rollbacks/).

## Open FIXMEs

Grep for `FIXME:` and `FIXME(` in this directory to find items still to be resolved:

- **OG image PNG** at `assets/og-image.png` (1200×630) — meta tags are currently commented out until the file exists.
- **atproto mark** — currently a text fallback at `assets/atproto-mark.svg`. Replace with the official mark from atproto.com brand assets when convenient.
