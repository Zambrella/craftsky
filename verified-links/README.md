# CraftSky verified-link host

Static resources served from `https://app.craftsky.social` for Android App Links,
iOS Universal Links, and safe browser fallbacks for the two OAuth completion
paths.

## Cloudflare Pages

- Framework preset: None
- Build command: empty
- Build output directory: `/`
- Root directory: `verified-links`

Attach only the custom domain `app.craftsky.social`. The association documents
must be available over HTTPS without redirects:

- `/.well-known/assetlinks.json`
- `/.well-known/apple-app-site-association`

The callback pages intentionally contain no scripts, external resources, forms,
or reflected query values. Their response headers prevent referrer leakage and
framing.
