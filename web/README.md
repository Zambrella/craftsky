# CraftSky landing page

Static public site served at https://craftsky.social.

## Contents

- `index.html` — landing page, nine sections
- `waitlist.html` — focused waiting-list signup page at `/waitlist`
- `privacy.html`, `terms.html`, `community-guidelines.html`, `reporting.html`,
  `copyright.html` — generated policy staging artifact; publication remains blocked
  until the readiness manifest passes
- `styles.css` — all styles, design tokens copied from `../docs/design/colors_and_type.css`
- `main.js` — waiting-list modal + PostHog event tracking
- `assets/` — favicon, logo, atproto mark, paper-grain texture
- `robots.txt` — allow all crawlers

## Local dev

No build step. Pick either:

```bash
# Quickest — open the file directly
open index.html

# Or serve with python for correct MIME types
python3 -m http.server 8000
# Then visit http://localhost:8000
```

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

## Deploy

Run the complete repository-side release workflow from the repository root:

```bash
just public-release-check
```

The command intentionally fails while any policy approval, control evidence, or P0
remains open. A passing command is necessary but does not replace the external atomic
deployment and rollback evidence.

Cloudflare Pages is configured to watch `web/` on `main`:

- Framework preset: None
- Build command: `npm run build`
- Build output directory: `/`
- Root directory: `web`

The build command runs the same fail-closed readiness evaluator for the production
branch. Cloudflare must retain this build command; an empty or overridden command
would bypass the repository-owned publication gate. Preview branches remain
available for policy review without publishing them to the production domain.

Every PR gets a preview URL under `pages.dev`. Production deploys land on `craftsky.social` after merging to `main`.

## Open FIXMEs

Grep for `FIXME:` and `FIXME(` in this directory to find items still to be resolved:

- **OG image PNG** at `assets/og-image.png` (1200×630) — meta tags are currently commented out until the file exists.
- **atproto mark** — currently a text fallback at `assets/atproto-mark.svg`. Replace with the official mark from atproto.com brand assets when convenient.
