# Landing page design

**Status:** Draft
**Date:** 2026-04-22
**Author:** Doug Todd (with Claude)

## 1. Overview

A single static HTML page served at `craftsky.social` that introduces the project to crafters, drives waiting-list signups, and points technical readers at the vision doc. The page is authored by hand (no build step, no framework), deployed to Cloudflare Pages, and styled using the tokens from the existing CraftSky design system (`docs/design/design-system.md`, `docs/design/colors_and_type.css`).

The page lives in a new top-level `web/` directory in this monorepo alongside `app/`, `appview/`, and `lexicon/`.

### 1.1 Goals

- Give a crafter arriving at `craftsky.social` a clear, honest, on-brand explanation of what Craftsky is in under a minute.
- Convert interested visitors onto a waiting list so they can be emailed at launch.
- Give technical visitors a route into the vision doc (primary) and the GitHub repo / lexicons (secondary, via the footer).
- Ship something simple enough that it doesn't rot — no build pipeline, no framework upgrades, no Node dependency.

### 1.2 Non-goals

- Blog, changelog, `/about`, `/docs`, or any multi-page structure.
- Internationalisation. The Flutter app is localised; the marketing page is English-only for now.
- Dark mode. The design system is single-mode (warm paper).
- Server-side anything. No AppView endpoint, no custom form handler, no auth.
- Social-proof copy (signup counts, logos, testimonials) — we have none yet and the voice explicitly avoids manufactured hype.
- An OG image designed in Figma — ship a placeholder for v1 and upgrade later.

## 2. Repository layout

A new top-level directory:

```
web/
  index.html
  styles.css
  main.js
  assets/
    favicon.svg
    og-image.png         (1200×630 placeholder for Open Graph / Twitter)
    logo.svg             (placeholder "CS" wordmark, cobalt on cream)
    atproto-mark.svg     (small atproto mark for footer; sourced from atproto.com brand assets)
    paper-grain.svg      (copied from app/assets/design/paper-grain.svg; ~4% opacity)
  README.md              (local dev + deploy notes)
```

There is **no** `package.json`, `node_modules`, `package-lock.json`, or build tool. Local dev is:

- Open `web/index.html` directly in a browser, or
- `cd web && python3 -m http.server 8000` and visit `http://localhost:8000`.

### 2.1 Design tokens

`web/styles.css` starts with a `:root` block containing the same CSS variables as `docs/design/colors_and_type.css`. The tokens are **copied**, not `@import`ed, so `web/` is self-contained and can be deployed without the rest of the repo being present.

A comment at the top of `styles.css` records the source of truth:

```css
/* Design tokens copied from docs/design/colors_and_type.css on 2026-04-22.
   If the design system changes, re-copy the :root block. */
```

Drift is tolerable; this is a marketing page, not a shared component library. `web/README.md` includes a one-line diff command to check for drift against `docs/design/colors_and_type.css`.

### 2.2 Content width

The spec uses two content widths, both from the design-system tokens:

- `--content-max` (1200px) — the outer container for hero, value-cards band, final CTA band, and footer.
- `--feed-max` (680px) — narrow single-column sections: "Why we're building this" (§4.4) and "Who's behind it" (§4.6).

All other sections (project posts §4.3, how it works §4.5, FAQ §4.7) sit inside `--content-max` with their own internal column layout.

## 3. Deployment

Cloudflare Pages, configured to deploy the `web/` directory on push to `main`.

- **Why Cloudflare Pages:** free at this traffic tier, automatic HTTPS, HTTP/3, Brotli, CDN. Pull-based deploy from GitHub means no CI changes. Independent of the AppView, so a broken AppView deploy doesn't take the marketing page down.
- **DNS:** `craftsky.social` apex points at the Pages project; `www.craftsky.social` redirects to the apex. `app.craftsky.social` is reserved for the Flutter web app and is unaffected.
- **Preview deploys:** every PR gets a preview URL under `pages.dev`, which makes visual review easy.

Netlify, Vercel, or GitHub Pages would all work equivalently. The implementation plan should treat "pick a host" as a 10-minute decision, not a research task.

## 4. Page structure

A single scrollable `index.html`, sections top-to-bottom:

1. Hero
2. Value cards band (three cards, matches the provided mockup)
3. Project posts — "post projects, find projects"
4. Why we're building this
5. How it works (AT Protocol, plainspoken)
6. Who's behind it
7. FAQ
8. Final CTA band
9. Footer

Detailed breakdowns follow.

### 4.1 Hero

- **Eyebrow chip:** text "Built in the open · on the AT Protocol". Butter background (`--butter`), ink border (1.5px), cobalt text, pill radius, Outfit 700 uppercase with 0.14em tracking.
- **Display heading:** "Made with *stuff*." DM Serif Display, `clamp(64px, 9vw, 128px)`, line-height 0.95. "Made with" in ink, "*stuff*." in cobalt italic.
- **Sub-copy:** Outfit 400, ~18px, ink, max-width ~520px. Exact copy:
  > A social feed for textile crafters. Share what you're making. Find the pattern everyone's talking about. Follow your local shop — no algorithm deciding what you see.
- **Primary CTA — "Join the waiting list":** cobalt fill (`--cobalt`), white text, Outfit 700, pill radius, 3px hard offset shadow (`--sh-drop-sm`), ink border. On hover, translates `(-1px, -1px)` and shadow grows to 4px. On click, opens the waiting-list modal (see §5).
- **Secondary CTA — "Read the spec":** paper-3 fill (white), ink text, ink border 1.5px, pill, 3px hard offset shadow. Links to the vision Google Doc (`https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit`) with `target="_blank" rel="noopener"`.
- **Illustration (right side on desktop, below text on mobile):** inline SVG paper-cutout — three rotated rectangles (one cobalt-filled, one sky-filled, one clay-filled with a small cobalt square overlaid), each with a 10px ink drop shadow, plus a small `--red` circle in the top-right. All colours are design-system tokens. No external image.
- **Section divider:** 2.5px solid ink rule beneath the hero, spanning the content column.

### 4.2 Value cards band

Three cards in a row on desktop, stacked on mobile. Each card:

- Paper swatch fill (card 1: `--butter`; card 2: `--sky`; card 3: `--lilac`).
- 1.5px ink border.
- 14px radius (`--r-3`).
- 6px hard offset shadow (`--sh-drop`).
- DM Serif Display heading, Outfit 500 body.

Copy (title + body, both sentence case, British English, design-system voice):

| Title | Body |
|---|---|
| No ads. Ever. | Not now, not later. We'll figure out sustainability together — advertising is off the table. |
| Your data is yours. | Built on the AT Protocol. If we ever disappear, your posts and followers don't. |
| Chronological feed. | You see the people you follow, in the order they posted. No algorithmic guessing. |

### 4.3 Project posts

Two-column on desktop, stacked on mobile.

- **Left:** DM Serif Display heading "Post projects. Find projects." Outfit body, ~120 words, explaining that Craftsky lets you share what you're making and search for what others are making — fabric, pattern, technique, whatever you want to find. British English throughout.
- **Right:** a mocked-up project card. Paper-3 fill, ink border, 6px hard offset shadow. Placeholder image sitting on a `--clay` swatch (roughly 8px of coloured paper visible around the image). Structured footer with a serif title ("Wiksten Haori"), Outfit 500 metadata row with middle-dot separators ("Sewing · WIP · 2 days"), and two chips. Both chips reuse existing design-system variants: one butter-filled ("Work in progress", the `--wip`-style chip), one ink-outlined on paper ("Linen", the default `.ck-chip` style). No new chip variants introduced.

The project card is static — no hover, no click. It's illustrative.

### 4.4 Why we're building this

Single column, narrow (~680px max).

- DM Serif Display heading: "Why we're building this."
- Outfit 400 body, ~150 words, in the design-system voice. Draws from the vision doc: chronological feed, no ads ever, data portability, strong search, transparent business accounts.
- Ends with a sentence pointing to the vision doc: _"The full vision doc is open for comments — [take a look](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit)."_ (Same URL as the hero secondary CTA.)

The exact copy is drafted during implementation and reviewed with Doug before shipping.

### 4.5 How it works

Three text blocks in a row on desktop, stacked on mobile. Each has a 24×24 Lucide icon (2px stroke, rounded caps) above a short paragraph. Plain-English AT Protocol explanation:

| Icon | Heading | Body |
|---|---|---|
| `box` | Your posts live on a server you control. | Every post is a record on your PDS — your corner of the AT Protocol. Craftsky reads from the public network; it doesn't own your data. |
| `compass` | Craftsky reads the network, organises it for crafters. | We index posts that use our project format and surface them in a chronological feed. You follow the people you want to see. |
| `arrow-right-left` | Move your account any time. | If Craftsky isn't for you, take your account elsewhere. Your followers come with you. |

Icons are inline SVG using `currentColor` for the stroke, sized 24×24, ink.

### 4.6 Who's behind it

Single paragraph, ~60 words. Small cobalt vertical accent bar on the left (3px wide). Copy to the effect of:

> Craftsky is being built by Doug Todd, the developer behind Stash Hub, in the open on GitHub. It's a community-first project — the vision doc is open for comments, the code is open for PRs, and the lexicons are public. We're not announcing dates. We'd rather get it right than get it out fast.

Includes inline links to GitHub (`github.com/<org>/craftsky` — resolve actual URL during implementation) and the vision doc.

### 4.7 FAQ

Native HTML `<details>`/`<summary>` — no JavaScript. Each item:

- 1.5px ink rule above and below (consolidate into single rules between items).
- `<summary>`: DM Serif Display, question in sentence case, small chevron indicator (CSS-only rotation on open).
- Body: Outfit 400, ~60–100 words.

Questions, in order:

1. When's it launching?
2. Will it be free?
3. What's the AT Protocol?
4. Can I use my Bluesky handle?
5. Is this just for textile crafters?
6. How is this different from Instagram / Pinterest / Ravelry?

Answers follow the design-system voice — honest about what we don't know, gently cheeky, no marketing-speak. Drafted during implementation; Doug reviews before shipping. (Note: "Who's building it?" is deliberately omitted from the FAQ; it lives in §4.6 only to avoid duplication.)

### 4.8 Final CTA band

Full-width cobalt band with a small amount of paper-grain SVG overlay (~4% opacity, per design system). White text.

- DM Serif Display heading: "Want in?"
- Outfit 400 sub-line: "We'll email you when there's something to see."
- Primary CTA: white pill button, cobalt text, ink border, 3px hard offset shadow. Label: "Join the waiting list". Opens the same modal as the hero CTA (with `source: 'final'` analytics property — see §6).

### 4.9 Footer

Three columns on desktop, stacked on mobile. Paper-2 background, thin ink top rule.

- **Left:** logo mark (placeholder cobalt "CS" wordmark) + tagline "Craftsky — a social feed for textile crafters."
- **Middle:** stacked links — Vision doc, GitHub (repo root), Lexicons (GitHub URL for `lexicon/`), Design system (GitHub URL for `docs/design/design-system.md`). All open in a new tab.
- **Right:** "Built on the AT Protocol." with a small atproto mark (SVG).
- **Bottom strip:** hairline ink rule, © year, small "Made with stuff" aside (reprising the hero pun).

## 5. Waiting-list form

### 5.1 Provider choice

A hosted, no-backend email-collection service. The specific provider is picked **at the start of the implementation plan**, not during implementation, because the two embed shapes (`<form action>` POST vs. iframe) lead to different modal markup and different success-state handling (§5.2).

Candidates, ranked by fit:

1. **Buttondown** — indie, privacy-respecting, clean embed, GDPR-friendly. Best voice fit. Supports native `<form>` POST.
2. **Tally** — free-tier generous, nice form builder. Iframe-first.
3. **ConvertKit** / **Mailchimp** — bigger, works, slightly corporate. Native `<form>` POST.
4. **Formspree** — posts raw submissions; email delivery happens elsewhere.

**Default assumption for the spec:** Buttondown with a native `<form>` POST. If the plan picks an iframe-only provider, §5.2's success-state handling needs to adapt (the success swap happens inside the iframe's own thank-you page rather than in our modal body).

### 5.2 UX

Clicking either "Join the waiting list" CTA opens a centred modal.

- Overlay: `rgba(22, 18, 16, 0.5)` flat ink, no blur (per design system).
- Modal container: native `<dialog>` element (see §11). Paper-3 fill, 1.5px ink border, `--r-4` radius (22px), 10px hard offset shadow (`--sh-drop-lg`), max-width ~520px, padded generously.
- DM Serif Display heading: "Join the waiting list."
- Outfit 400 sub: "We'll email you when there's something to see. Nothing else."
- Native `<form>` (or iframe) with a single email field, Outfit 500 label "Email", cobalt pill submit button "Sign me up".
- Close paths: X button in top-right, ESC key, overlay (backdrop) click. Focus trap, ESC handling, and `::backdrop` are provided natively by `<dialog>` — see §5.3 and §11.
- Success state: form body swaps to a thank-you message ("Thanks. We'll be in touch when there's something worth sharing."). A close button is available.

### 5.3 JavaScript

A single small vanilla JS helper in `main.js` handling:

- Opening the modal (`dialog.showModal()`) and closing it (`dialog.close()`).
- Closing on overlay click (event target === the dialog element).
- Firing the PostHog events (§6).
- Closing via the X button inside the dialog.

ESC-to-close and focus-trap are provided by `<dialog>` natively.

Estimated ~30 lines of JS total. No dependencies.

## 6. Analytics

### 6.1 Provider

PostHog, loaded via their official snippet at the bottom of `<body>`. Configuration:

- `persistence: 'memory'` — no cookies, no localStorage.
- `autocapture: false` — we only want the events we explicitly send.
- `disable_session_recording: true`.
- `capture_pageview: false` — we're not tracking page views.

The PostHog project key is a plain constant in `main.js`. It's a public write-only key; committing it is safe.

### 6.2 Events

Exactly two custom events:

| Event | Properties | Fires on |
|---|---|---|
| `landing_cta_waiting_list_clicked` | `source: 'hero' \| 'final'` | Click of either "Join the waiting list" button (before the modal opens). |
| `landing_cta_spec_clicked` | `source: 'hero'` | Click of the "Read the spec" button. |

No other events. No scroll depth, no time-on-page, no UTM capture.

### 6.3 Do Not Track

If the browser sends `navigator.doNotTrack === '1'`, skip loading the PostHog snippet entirely. This is simpler and stricter than PostHog's in-library opt-out and fits the "your data is yours" brand.

## 7. Accessibility

- Semantic HTML: `<header>` (hero container), `<main>` (all body sections), `<footer>`. Each section is a `<section>` with a heading.
- Skip link to `#main` at the top of `<body>`, visible on focus only.
- All interactive elements are real `<button>` or `<a>` — no `onclick` on `<div>`.
- Visible focus ring: 2px cobalt outline with 2px offset. Works on paper, white, and cobalt backgrounds (on cobalt, the ring becomes white).
- Colour contrast: the design-system tokens already pass WCAG AA for the combinations used (ink on paper, white on cobalt, ink on butter/sky/lilac). Verified during implementation with axe / Lighthouse.
- `prefers-reduced-motion: reduce` honoured. The only motion on the page is the button hover/press translate; the media query disables it.
- Images have descriptive `alt` text. Decorative SVG (the hero illustration, the paper-grain) is marked `aria-hidden="true"` with an empty `alt`.
- Modal: native `<dialog>` element (role + modal semantics built in), `aria-labelledby` pointing at the modal heading. `<dialog>` restores focus to the triggering element on close by default.
- FAQ `<summary>` elements are keyboard-accessible natively.

## 8. SEO & social

- `<html lang="en-GB">` — British English is explicit in the voice guidelines.
- `<title>`: "Craftsky — a social feed for textile crafters."
- `<meta name="description">`: ~155 chars, first line of the hero sub-copy adapted.
- `<link rel="canonical" href="https://craftsky.social/">`.
- Open Graph: `og:title`, `og:description`, `og:image` (1200×630), `og:url`, `og:type=website`.
- Twitter card: `twitter:card=summary_large_image`, same title/description/image.
- Favicon: cobalt "CS" on cream, SVG with PNG fallback.
- `robots.txt` at the root allowing all.
- No sitemap (single page).

The OG image for v1 is a placeholder — a 1200×630 PNG of the hero headline on paper background. An upgraded version can land later.

## 9. Performance

Targets:

- Lighthouse ≥95 on Performance, Accessibility, Best Practices, SEO (desktop and mobile).
- Total page weight <200 KB gzipped (excluding fonts).
- No layout shift during font swap (use `font-display: swap` and `size-adjust` if needed).
- Single Google Fonts request with `preconnect` to `fonts.googleapis.com` and `fonts.gstatic.com`.

Deferred loading:

- PostHog snippet loaded with `defer` at the bottom of `<body>`.
- Hero illustration is inline SVG (no request).
- OG image is referenced only in `<meta>` — no render impact.

## 10. Responsive

One breakpoint at 820px.

- **≥820px:** Two-column hero (text left, illustration right). Three-column value cards. Two-column project-posts section. Three-column "how it works". Three-column footer.
- **<820px:** Everything stacks vertically. Hero illustration sits below the text. Display heading scales down via `clamp()`. CTAs remain pill-shaped, full-width of their column (not the page).

No hamburger menu — there is no on-site navigation to hide.

Mobile hit targets are minimum 44px per the design system.

## 11. Browser support

Modern evergreen browsers: last 2 versions of Chrome, Safari, Firefox, Edge. Mobile Safari ≥ iOS 15, Chrome Android ≥ last 2 versions. No IE11.

No polyfills needed for the features used: `<details>`, `clamp()`, CSS custom properties, and `<dialog>` (Safari 15.4+, which matches the iOS 15 floor).

The modal uses the native `<dialog>` element. It provides focus trap, ESC-to-close, and backdrop handling for free, which reduces `main.js` from ~60 lines to ~30. The backdrop is styled via `::backdrop`. Overlay-click-to-close is a few lines of JS (detect clicks on the dialog element itself rather than its content).

## 12. Acceptance criteria

- [ ] Page renders at `craftsky.social` with HTTPS and a valid cert.
- [ ] All nine sections present with the copy and visual treatment described.
- [ ] "Join the waiting list" CTAs in both hero and final band open the modal; a successful signup lands in the chosen provider's list.
- [ ] "Read the spec" CTA links to the vision Google Doc in a new tab.
- [ ] PostHog fires `landing_cta_waiting_list_clicked` (with correct `source`) and `landing_cta_spec_clicked` on the appropriate clicks. Confirmed in PostHog project dashboard.
- [ ] DNT-enabled browsers do not load PostHog.
- [ ] Responsive down to 360px width with no horizontal scroll.
- [ ] Lighthouse scores ≥95 on all four categories, desktop and mobile.
- [ ] axe-core reports zero critical or serious violations.
- [ ] Keyboard-only navigation works for every interactive element, including the modal.
- [ ] `prefers-reduced-motion: reduce` disables all transitions.
- [ ] Page weight <200 KB gzipped (excluding fonts).
- [ ] Deploy pipeline: pushing to `main` deploys to Cloudflare Pages; preview URLs work on PRs.

## 13. What's deferred

Not in this spec; revisit later if useful:

- A `/docs` or `/about` page. Vision doc covers "about" for now.
- A launch-countdown timer or other urgency mechanics.
- Localisation.
- A properly designed logo / wordmark.
- An OG image designed in Figma.
- Replacing the placeholder project-card illustration with real screenshots once the Flutter app has screens.
- A blog or changelog.

## 14. Open questions (to resolve in the plan)

- **Waiting-list provider.** Spec assumes Buttondown; plan's first step is to confirm or swap.
- **PostHog project key.** Doug to create the project and paste the key into the implementation PR.
- **GitHub repo URL.** Resolve the actual `github.com/<org>/craftsky` URL for the footer and "Who's behind it" links.
- **`.superpowers/` in `.gitignore`.** The brainstorming tool writes there and it's currently not ignored. Low priority; mention in the PR description.

## 15. Copy drafting

Two sections have sample copy that is indicative, not final:

- §4.4 **Why we're building this** — ~150 words drafted during implementation, drawing from the vision doc.
- §4.7 **FAQ** — six answers, each ~60–100 words, drafted during implementation.

Both sections must be reviewed by Doug before the page ships. The implementation plan should treat copy drafting as an explicit step (not an in-flight detail), with a review checkpoint before merging the PR.

## 16. Roadmap entry

The landing page is not currently tracked in `docs/roadmap.md`. The implementation PR should add a single line under a new **Web / marketing** heading in the v1 section (or under "Product / community" if that fits better) so it's visible alongside the other v1 workstreams.
