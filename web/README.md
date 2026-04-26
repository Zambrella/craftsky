# Craftsky landing page

Single static HTML page served at https://craftsky.social.

## Contents

- `index.html` — landing page, nine sections
- `privacy.html` — privacy policy (linked from footer)
- `terms.html` — terms of service (linked from footer)
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

## Deploy

Cloudflare Pages is configured to watch `web/` on `main`:

- Framework preset: None
- Build command: (empty)
- Build output directory: `/`
- Root directory: `web`

Every PR gets a preview URL under `pages.dev`. Production deploys land on `craftsky.social` after merging to `main`.

## Open FIXMEs

Grep for `FIXME:` and `FIXME(` in this directory to find items still to be resolved:

- **Waiting-list provider endpoint** in `index.html` (modal form `action` — currently `#TODO-WAITING-LIST-ENDPOINT`).
- **PostHog project key** in `main.js` (currently `REPLACE_ME`; until replaced, PostHog does not load).
- **GitHub repo URL** in footer and "Who's behind it" section (currently `#TODO-GITHUB-URL`).
- **OG image PNG** at `assets/og-image.png` (1200×630) — meta tags are currently commented out until the file exists.
- **atproto mark** — currently a text fallback at `assets/atproto-mark.svg`. Replace with the official mark from atproto.com brand assets when convenient.
