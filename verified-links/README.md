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
framing. Each fallback is stored as an extensionless-route `.html` file so
Cloudflare Pages serves the exact callback path without a redirect.

AppView's completion button uses a package-targeted Android `intent://` link
when the browser identifies itself as Android. The intent reconstructs the same
HTTPS callback for `social.craftsky.app`; it does not require a custom scheme in
the app. iOS and browsers without JavaScript retain the HTTPS link. Launching
requires a user tap, with no automatic retry or timer-based success detection.
The intent's browser fallback points here without the handoff code or deletion
proof. These static pages give recovery instructions without reflecting queries.

Before release, exercise both completion paths in Samsung Internet and Chrome
with the Google Play build. Check app launch with an installed app, instructions
when the app cannot launch, and iOS Universal Links. Desktop browser tests cannot
verify Android's intent resolution, browser permissions, or Play signing.

Samsung Browser 30.0.5.24 calls its app-link preference **Block opening external
apps**, under **Settings → Security and privacy**. Turning it off permits links
to open external apps. The intent button worked in the emulator with this setting
on, so recovery instructions make this a troubleshooting step rather than a
required configuration. Older browser versions may use different labels.
