# CraftSky Design System

A social network for textile crafters — built on the AT Protocol, community-owned, no ads, ever.

This design system defines the visual and content foundations for CraftSky across mobile and web.

---

## What is CraftSky?

CraftSky is a social platform built specifically for the textile crafting community: sewists, knitters, crocheters, quilters, embroiderers, and everyone working with fibre and fabric. It's a reaction to what general-purpose social networks lost — chronological feeds, working hashtag search, communities you can actually find.

The product is being built in the open by Doug Todd (the developer behind Stash Hub) on top of the AT Protocol, meaning users own their data, their posts, and their social graph. If CraftSky disappears, their work doesn't.

**Core product pillars**

- **Project posts** — structured sharing with fields crafters actually care about: pattern, fabric/yarn, craft type, techniques, modifications, status (WIP/finished), notes, difficulty.
- **Chronological feed** — no algorithm deciding what you see.
- **Strong search** — hashtag and keyword search that actually works. Find every version of a pattern.
- **Social basics, done well** — follow, like, repost, comment, block, mute, saved posts.
- **Business accounts** — clearly flagged, same feed as everyone else, no pay-to-play reach.
- **Sponsored-content toggle** — built into post creation, normalised disclosure.
- **Craft preferences** — onboarding signal for finding your people.
- **Profile themes** — self-expression is part of craft; profiles reflect it.
- **AT Protocol native** — Bluesky social graph carries over; data is portable.

**Surfaces**

- Mobile app (Flutter — iOS & Android)
- Web app (Flutter web)

---

## Sources

No codebase, Figma, or brand assets were provided. The brand identity in this system was developed from scratch using the product brief. The visual language is inspired by the feel of **Bear** and **Bearable** — playful, confident, photo-forward. Treat everything here as a **first proposal** to iterate on, not final brand guidelines.

If you have existing brand materials (wordmark, palette, app screenshots, Figma), attach them and I'll re-anchor the system.

---

## Index

Core files at the root:

- `README.md` — this document
- `colors_and_type.css` — CSS variables for colours, type, spacing, radii, shadows
- `SKILL.md` — skill manifest
- `assets/` — logo marks, craft icons, paper texture
- `preview/` — small HTML cards surfaced in the Design System tab
- `ui_kits/mobile/` — mobile app UI kit
- `ui_kits/web/` — web app UI kit

---

## CONTENT FUNDAMENTALS

CraftSky's voice is **warm, plainspoken, a little cheeky, and respectful of craft**. It sounds like a maker-friend — someone who'll tell you their linen bled in the wash and laugh about it, not a brand trying to be your brand.

### Tone

- **Direct without being curt.** "We built this for crafters. You deserve better than the feed you've got."
- **A bit of cheek.** "No ads. Ever. (Yes, really.)" Confident, not corporate.
- **Honest about what we don't know.** "We're still figuring out how to sustain this. We'll tell you when we know."
- **Values-forward, not preachy.** We say what we stand for; we don't spend paragraphs attacking other platforms.
- **Respects craft as expertise.** Readers know the difference between a quilt and a coverlet. Don't over-explain craft terms. Do explain tech terms (AT Protocol, data portability).
- **Gentle humour, no snark.** "You know what you're doing. We'll stay out of your way."

### Person & address

- **"We" for the team, "you" for the crafter.** "We're building. You own your posts."
- **Second person is default** for UI copy. "Your feed", "Your projects", "Share what you made."
- **First-person plural stays humble.** "We think" rather than "We believe" when stating preferences.

### Casing

- **Sentence case everywhere.** Titles, buttons, menu items.
- **Proper nouns capitalised normally.** CraftSky, AT Protocol, Bluesky, Instagram.
- **No all-caps for emphasis.** Use weight, size, or colour.
- **Chunky display type carries the emphasis.** The system already shouts — the words don't need to.

### Spelling & punctuation

- **British English** by default (colour, customisation, organise, favourite).
- **Oxford comma: no**, unless it resolves ambiguity.
- **Em dashes welcome** — they fit the conversational rhythm.
- **Contractions, always.** "You're", "we're", "it's".

### Emoji

**Not part of the system's chrome.** No emoji in buttons, empty states, section headings. The type and colour do the expression. Users can use emoji in their own posts — that's their voice.

### Specific examples

| Situation | ✅ CraftSky voice | ❌ Off-brand |
|---|---|---|
| Empty feed | "Quiet in here. Follow someone to start filling it up." | "Your feed is empty! 😢 Start following people to unlock content." |
| Project status | "Work in progress" / "Finished" | "WIP 🚧" / "DONE! 🎉" |
| Report confirmation | "Thanks — we'll take a look." | "Report submitted successfully. Our team will review within 24 hours." |
| Business flag | "Business account" | "VERIFIED SELLER ⭐" |
| Sponsored toggle | "Paid, gifted, or affiliate? Flag it — your people will respect it." | "Disclose paid partnership" |
| Launch copy | "We're not announcing dates. We'd rather get it right than get it out fast." | "Coming soon — join the hype!" |
| Error state | "That didn't load. Try again?" | "Oops! An error occurred." |
| Onboarding craft picker | "What are you into? Pick as many as you like." | "Select your interests to personalise your experience" |

### Microcopy conventions

- **Buttons are verbs.** "Share", "Follow", "Save", "Repost".
- **Field labels describe the thing.** "Pattern name", "Fabric or yarn", "Modifications". Sentence case, no colons.
- **Placeholder text is a real example.** For "Pattern name": _e.g. Wiksten Haori_. For "Fabric": _Merchant & Mills 185 linen, indigo_.
- **Error messages suggest the fix.** "Image needs to be under 20 MB." not "Upload failed."

---

## VISUAL FOUNDATIONS

The system is **paper cutout** — warm cream paper, confident black rules, chunky typography, photography sitting on coloured paper rectangles. It should feel like a well-made zine or a kid's cutout book, not a sterile SaaS app. Reference points: **Bear** (for the warmth), **Bearable** (for the confident colour use), old penguin paperbacks, risograph prints.

### Colour

Two committed accents sit on warm paper. Everything else is ink-black or paper.

**Core**

- **Paper** — `#F5EFE4`. The page. Warm cream, not beige. No pure white backgrounds.
- **Paper 2** — `#EFE7D6`. Sunken surfaces.
- **Paper 3** — `#FFFFFF`. Pure white used only for sheets pinned on top (project cards, modals).
- **Ink** — `#161210`. Near-black with a touch of warmth. Never `#000`.

**Accents** — these do all the colour work.

- **Cobalt** — `#1535D6`. Primary. Brand, links, primary buttons, selected state.
- **Electric red** — `#F03A2E`. Accent. Likes, sponsored flag, destructive, editorial pop.

**Supporting paper swatches** — used as coloured-paper backgrounds behind images, for chips, and for large surface variety. Never as text colour.

- **Butter** `#F7D46A` — WIP chip background, sunny surfaces.
- **Clay** `#E27B4A` — terracotta, hero backgrounds.
- **Moss** `#6E8B3D` — finished chip, calm surfaces.
- **Sky** `#9BC2E6` — pale blue cutouts.
- **Lilac** `#C9B8E8` — pale purple cutouts.

**Rules**

- No gradients.
- No pure black, no pure white (except Paper 3).
- Photography sits on a paper-swatch rectangle, not on the page directly — this is the signature layout move.
- Cobalt is the only blue. Red is the only red. Don't dilute either with tints.

See `colors_and_type.css` for the full token list.

### Typography

Two families, one job each.

- **Display — `DM Serif Display`** (Google Fonts). Chunky high-contrast serif with real personality — old-book, slightly theatrical. Used for hero headings, project titles, editorial moments. Italic is welcome. Size it **big** — display-1 is 96px and that's on the small side of comfortable.
- **UI — `Outfit`** (Google Fonts). Rounded geometric sans, variable weight 400–800. Does all interface work: buttons, body, labels, metadata. Go heavy (600–800) for headings and buttons — this system rewards weight.
- **Mono — `JetBrains Mono`** for pattern IDs, handles, DIDs, timestamps.

**Rhythm**
- Big display jumps: h1 is twice h2. Don't be shy.
- Body is 16px web / 15px mobile, line-height 1.5.
- Display line-height is tight — 0.95–1.02. Let letters almost touch.
- Eyebrow labels are Outfit 700, uppercase, `0.14em` tracking.

### Spacing & sizing

8-point grid with a 4-point half step: `--sp-1` 4 · `--sp-2` 8 · `--sp-3` 12 · `--sp-4` 16 · `--sp-5` 24 · `--sp-6` 32 · `--sp-7` 48 · `--sp-8` 64 · `--sp-9` 96.

Mobile hit targets never under 44px. Slides/presentations 24px+ text.

### Corner radii

Mixed — square for most UI, chunky-rounded for signature moments.

- `--r-0` 0 — full-bleed photos, coloured rectangles
- `--r-1` 2 — form fields, inputs
- `--r-2` 6 — small chips (non-pill)
- `--r-3` 14 — cards
- `--r-4` 22 — chunky statement buttons, hero shapes
- `--r-pill` 999 — chips, avatars, primary buttons

### Borders & rules

Bold, confident, always ink-black.

- Default border: `1.5px solid var(--rule)` where `--rule` is `--ink` (`#161210`).
- Cards get a full black border. This is the **single most recognisable move** — cards don't blend into the page, they're cut out of it.
- Section dividers: `2.5px solid var(--rule)` with generous vertical space either side. Thicker than the card border. Full black.
- Hairlines (`--border-hair`, 15% ink) only inside cards — never as outer borders.

### Elevation & shadow

Two shadow systems, used deliberately.

**Hard offset (signature)** — black rectangle behind the card, offset down-right. Feels like paper on paper.
- `--sh-drop-sm` — `3px 3px 0 var(--ink)` — buttons, small chips
- `--sh-drop`    — `6px 6px 0 var(--ink)` — cards, hero elements
- `--sh-drop-lg` — `10px 10px 0 var(--ink)` — posters, landing hero

**Soft paper** — realistic drop for true floating sheets (modals, popovers).
- `--sh-paper-1` — subtle, modal-lite
- `--sh-paper-2` — full modal

Never mix hard and soft in the same surface. Never use coloured shadows.

### Backgrounds & textures

- Primary background is flat `--paper` — no gradient.
- A subtle paper-grain SVG (`assets/paper-grain.svg`) sits at ~4% opacity on landing hero and profile banners. Never on dense UI.
- Full-bleed photography is cropped tight and laid onto a coloured paper swatch (butter, clay, moss, sky, lilac) — roughly 4–12px of coloured paper visible around the image. This is the **signature layout pattern**.
- Images are warm-biased, filmy, matte. No slick studio work.

### Imagery direction

- **Warm natural light.** Morning, afternoon, not strip-lit studio.
- **Hands + material in-frame.** In-progress sleeve on a lap, yarn in a basket, tissue paper on a cutting mat.
- **Subject over process-porn.** Real makers, real homes.
- **Lightly grainy.** Film-adjacent, matte finish.

### Motion & animation

**Quiet with occasional pop.** The page is already shouting — motion is mostly calm.

- Default easing: `cubic-bezier(0.22, 0.61, 0.36, 1)` (ease-out)
- Pop easing (for buttons, likes): `cubic-bezier(0.34, 1.56, 0.64, 1)` — springy, light bounce
- Default duration: 120ms UI feedback, 220ms entrances, 320ms modals
- Buttons "press in" on click: translate 2px down-right, shadow flattens to 0. This is the one place motion is playful.
- Heart / like: single scale pop 1 → 1.2 → 1 over 260ms, with pop easing.
- No slide-in from off-screen on list items. Cross-fade only.

### Hover states

- Text links: colour deepens; underline stays 2px.
- Primary buttons: background deepens, translate `(-1px, -1px)`, shadow grows to `4px 4px 0 var(--ink)` — it lifts toward you.
- Ghost buttons: fill to `--paper-2`.
- Cards: no lift by default. Clickable cards translate `(-2px, -2px)`, shadow grows.

### Press / active states

- Buttons: translate `(2px, 2px)`, shadow collapses to `0 0 0 var(--ink)` — it gets pressed into the page.
- Links: colour deepens further.

### Transparency & blur

- Sticky headers: `rgba(245, 239, 228, 0.9)` over `backdrop-filter: blur(12px)`.
- Modal overlays: `rgba(22, 18, 16, 0.5)` flat — **no blur** behind modals; papery, not glassy.
- No blur over imagery for chip legibility — use solid fills.

### Layout rules

- Mobile tab bar: 60px + safe area.
- Web nav: 64px top.
- Feed column max width: 680px.
- Profile max width: 1120px.

### What cards look like

- Fill: `--paper-3` (white) on paper pages, or a coloured paper swatch.
- Border: `1.5px solid var(--rule)`, full ink-black.
- Radius: `--r-3` (14px).
- Shadow: `--sh-drop` (6px 6px hard ink offset) on clickable cards; none at rest for inline cards.
- Project cards lead with the image on a coloured swatch, then a structured footer: title in DM Serif Display, metadata in Outfit 500, chips with black outlines.

---

## ICONOGRAPHY

**Chunky stroke-based icons, 2px stroke**, drawn in the same confident hand as the rest of the system. Thicker than Lucide default to match the heavy type.

### Approach

- **Stroke 2px** for 24×24 icons. 1.75px for 20×20. Never filled except for "liked" heart.
- **Rounded line caps + joins** — matches the soft-ended chunky display.
- **24×24 default** for interface chrome, 20×20 for dense rows, 16×16 for inline.
- **Stroke inherits `currentColor`.**
- Icons are functional only — no decorative flourishes next to headings.

### Where they come from

- **Lucide** (https://lucide.dev) at 2px stroke is the base set.
- **Custom craft-specific marks** — needle, spool, skein, stitch — in `assets/icons/`. Drawn in Lucide's geometry, 2px stroke, 24×24, round caps.
- No icon font. SVG inline or `<img>`. No PNG icons.

### Emoji

Not part of the system. Not in buttons, not in headings, not in empty states.

### Unicode

- `·` (middle dot) separator between metadata items: _Sewing · WIP · 2 days_.
- `—` (em dash) in editorial copy.
- `✓` never — use a Lucide `check` icon.

### Logo

Placeholder only: a stylised "CS" set in DM Serif Display, cobalt, on a cream paper tape label with a thin black border. See `assets/logo/`. Treat as scaffolding until a proper wordmark is commissioned.

---

_See also: `colors_and_type.css` for tokens, `ui_kits/` for product recreations, `SKILL.md` for skill integration._
