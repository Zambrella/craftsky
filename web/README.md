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

## Deploy

Run the repository quality checks from the repository root:

```bash
just public-release-check
```

The command runs build and test checks without requiring the full-launch safety
gaps to be closed. Before full public launch, run `just online-safety-readiness`
and complete the approvals and evidence recorded in the gap register.

Cloudflare Pages is configured to watch `web/` on `main`:

- Framework preset: None
- Build command: `npm run build`
- Build output directory: `/`
- Root directory: `web`

The build command allows beta publication on the production branch while safety
gaps remain open. It does not approve the policy drafts or mark the service ready
for full public launch. Preview branches remain available for policy review.

Every PR gets a preview URL under `pages.dev`. Production deploys land on `craftsky.social` after merging to `main`.

## Open FIXMEs

Grep for `FIXME:` and `FIXME(` in this directory to find items still to be resolved:

- **OG image PNG** at `assets/og-image.png` (1200×630) — meta tags are currently commented out until the file exists.
- **atproto mark** — currently a text fallback at `assets/atproto-mark.svg`. Replace with the official mark from atproto.com brand assets when convenient.
