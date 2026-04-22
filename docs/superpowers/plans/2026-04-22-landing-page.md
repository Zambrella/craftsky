# Landing page implementation plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a single static HTML marketing page at `craftsky.social` in a new top-level `web/` directory, deploy it via Cloudflare Pages, wire anonymous PostHog analytics for the two CTAs, and embed a waiting-list form (provider deferred; `<form action>` placeholder).

**Architecture:** One `index.html` with nine sections, one `styles.css` (design tokens copied from `docs/design/colors_and_type.css`), one `main.js` (~30 lines) for the `<dialog>`-based signup modal + PostHog event dispatch. No build step, no framework, no Node dependency. Hero illustration is inline SVG. Assets are hand-placed in `web/assets/`.

**Tech Stack:** HTML5, CSS custom properties + `clamp()`, vanilla ES2020 JS, native `<dialog>`, Google Fonts (DM Serif Display + Outfit + JetBrains Mono), PostHog JS snippet. Cloudflare Pages for hosting.

## Spec

The approved design doc is [docs/superpowers/specs/2026-04-22-landing-page-design.md](../specs/2026-04-22-landing-page-design.md). Read it first — this plan assumes familiarity with it.

## Decisions resolved before plan authoring

From spec §14:

1. **Waiting-list provider** — **deferred**. The form posts to a placeholder `action="#TODO-WAITING-LIST-ENDPOINT"` attribute. A `FIXME:` comment in `index.html` marks the swap point. The modal success flow is hidden behind a JS hook that runs post-submit; when the provider is chosen, only two things change (the `action` URL and possibly the hook). The placeholder form is not wired to a live service, so the acceptance criterion "a successful signup lands in the chosen provider's list" is explicitly **not** satisfied by this PR and will be completed in a follow-up.
2. **PostHog project key** — the plan leaves a `POSTHOG_KEY` constant in `main.js` set to `'REPLACE_ME'`. The comment next to it documents how to get one. Until replaced, the DNT-skip logic in `main.js` also skips PostHog initialisation when the key is still the placeholder.
3. **GitHub repo URL** — **deferred**. Footer links and "Who's behind it" copy use `#TODO-GITHUB-URL` placeholders with `FIXME:` comments. Same swap story as (1).
4. **`.superpowers/` in `.gitignore`** — **in scope**. Added in Chunk 0.

## Note on TDD for static-HTML work

This plan builds a single marketing page: hand-authored HTML, design-system-token CSS, and ~30 lines of vanilla JS. A red-green-refactor loop per file would produce performative tests that assert "element exists in DOM", which duplicates what you can verify by opening the page in a browser.

What is verified, and how:

- **Tokens present in CSS:** compared against `docs/design/colors_and_type.css` with a shell diff.
- **HTML validity:** W3C validator run locally or a `tidy` pass (noted in each task that touches `index.html`).
- **Accessibility:** manual axe-core run in the browser at the end of each chunk that adds markup (§Chunk 6).
- **Keyboard traversal:** manual Tab-through documented in §Chunk 6.
- **JavaScript behaviour:** the only meaningful runtime surface is the modal open/close + analytics dispatch, verified by a **single browser-based smoke test** described in §Chunk 7. PostHog receives a mocked `posthog` global during the smoke test so we assert the exact event payload.
- **Responsive layout:** manual resize of the browser window to 360px / 820px / 1200px, screenshots captured and pasted into the PR.
- **Lighthouse / Cloudflare deploy:** final gate before merge, described in §Chunk 8.

If you find yourself writing a test that asserts "the hero section has three paper rectangles" from JavaScript, you're duplicating a manual visual check — skip it.

## Working directory

All paths are relative to the **repo root** (the worktree root). The landing page lives under `web/`. There is no package manager and no command to run from inside `web/` other than `python3 -m http.server 8000` for local dev.

## File map

Files created by this plan:

| Path | Purpose |
|---|---|
| `.gitignore` | Modify to ignore `.superpowers/`. |
| `docs/roadmap.md` | Modify to add a Web / marketing heading and a line for the landing page. |
| `web/index.html` | Single-page HTML with all nine sections. ~400 lines. |
| `web/styles.css` | Design-token `:root` + section-by-section styles. ~500 lines. |
| `web/main.js` | `<dialog>` open/close, overlay click, PostHog init (DNT-aware), two CTA event dispatchers. ~60 lines including comments. |
| `web/robots.txt` | `User-agent: *\nAllow: /` and sitemap omitted. |
| `web/README.md` | Local dev, deploy notes, token-drift check. |
| `web/assets/favicon.svg` | Cobalt "CS" on cream. |
| `web/assets/logo.svg` | Placeholder wordmark. |
| `web/assets/atproto-mark.svg` | Footer mark (copied from atproto.com brand assets). |
| `web/assets/paper-grain.svg` | Copied from `app/assets/design/paper-grain.svg`. |
| `web/assets/og-image.png` | 1200×630 placeholder. |

---

## Chunk 0: Repo plumbing

One-off housekeeping before touching the page.

### Task 0.1: Add `.superpowers/` to `.gitignore`

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Edit `.gitignore`**

Append a new block at the end:

```
# Superpowers brainstorming artefacts
.superpowers/
```

- [ ] **Step 2: Verify nothing under `.superpowers/` is already tracked**

Run: `git ls-files | grep '^.superpowers/' | head`
Expected: no output (nothing tracked there already).

- [ ] **Step 3: Commit**

```bash
git add .gitignore
git commit -m "chore: ignore .superpowers/ brainstorming artefacts"
```

### Task 0.2: Scaffold empty `web/` structure

**Files:**
- Create: `web/.gitkeep` (deleted in the next chunk once real files land)
- Create: `web/assets/.gitkeep` (same)

- [ ] **Step 1: Create directories**

```bash
mkdir -p web/assets
touch web/.gitkeep web/assets/.gitkeep
```

- [ ] **Step 2: Commit**

```bash
git add web/.gitkeep web/assets/.gitkeep
git commit -m "chore(web): scaffold empty web/ directory"
```

### Task 0.3: Roadmap entry

**Files:**
- Modify: `docs/roadmap.md`

- [ ] **Step 1: Add a `Web / marketing` heading to the v1 section**

Open `docs/roadmap.md`. Note there are **two** `### Ops / infra` headings — one under `## v1` (roughly line 51) and one under `## After v1` (roughly line 112). Insert the new section immediately **above the first one (inside `## v1`)** so it sits between `### Lexicons` and `### Ops / infra` within v1. Content:

```markdown
### Web / marketing

- [ ] Landing page at craftsky.social (hero + 8 sections, Cloudflare Pages, anonymous PostHog) — [`2026-04-22-landing-page-design.md`](superpowers/specs/2026-04-22-landing-page-design.md)
```

- [ ] **Step 2: Commit**

```bash
git add docs/roadmap.md
git commit -m "docs(roadmap): add landing page entry under Web / marketing"
```

---

## Chunk 1: Static shell and design tokens

Write `index.html` with a complete document skeleton (head, meta, fonts, skip-link, empty sections) and the full `:root` token block in `styles.css`. After this chunk, the page loads, uses the right fonts, and has the right background colour — but all sections are empty.

### Task 1.1: Copy token block to `web/styles.css`

**Files:**
- Create: `web/styles.css`

- [ ] **Step 1: Read the canonical tokens**

Read `docs/design/colors_and_type.css` lines 1–119. That is the `:root` block plus the `@import` line for Google Fonts.

- [ ] **Step 2: Write `web/styles.css` with header + token block**

File contents (top of file; the rest of the styles are appended in later chunks):

```css
/* =========================================================================
   Craftsky landing page — styles
   Design tokens copied from docs/design/colors_and_type.css on 2026-04-22.
   If the design system changes, re-copy the :root block. Run this to diff:
     diff <(sed -n '/^:root {/,/^}/p' web/styles.css) \
          <(sed -n '/^:root {/,/^}/p' docs/design/colors_and_type.css)

   NOTE: Fonts are loaded via <link rel="stylesheet"> in index.html (not via
   @import here) to avoid a render-blocking request chain that costs ~5–10
   Lighthouse Performance points. Preconnect hints in index.html do the rest.
   ========================================================================= */

:root {
  /* (paste the exact :root block from docs/design/colors_and_type.css lines 9–119 here) */
}

/* =========================================================================
   Base
   ========================================================================= */

* { box-sizing: border-box; }
html { -webkit-text-size-adjust: 100%; }

body {
  margin: 0;
  background: var(--bg);
  color: var(--fg-1);
  font-family: var(--font-ui);
  font-size: var(--t-body);
  line-height: var(--lh-body);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

p { margin: 0; text-wrap: pretty; }

a {
  color: var(--cobalt);
  text-decoration: underline;
  text-decoration-thickness: 2px;
  text-underline-offset: 3px;
}
a:hover { color: var(--cobalt-deep); }

/* Skip link — visible on focus only */
.skip-link {
  position: absolute;
  top: -100px;
  left: var(--sp-4);
  padding: var(--sp-2) var(--sp-4);
  background: var(--ink);
  color: #FFFFFF;
  border-radius: var(--r-2);
  text-decoration: none;
  font-weight: 700;
  z-index: 100;
}
.skip-link:focus { top: var(--sp-4); }

/* Focus ring — consistent across light and dark contexts */
:focus-visible {
  outline: 2px solid var(--cobalt);
  outline-offset: 2px;
}
.on-cobalt :focus-visible { outline-color: #FFFFFF; }

/* Reduced-motion: disable all transitions on the page */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}
```

**Do not** paste a guessed `:root` block — copy the exact contents of `docs/design/colors_and_type.css` lines 9–119 verbatim.

- [ ] **Step 3: Verify the token block matches**

Run:
```bash
diff <(sed -n '/^:root {/,/^}/p' web/styles.css) \
     <(sed -n '/^:root {/,/^}/p' docs/design/colors_and_type.css)
```

Expected: no output (identical blocks).

If the diff shows differences, re-copy from the source file until the diff is empty.

- [ ] **Step 4: Commit**

```bash
git add web/styles.css
git commit -m "feat(web): add design tokens and base styles"
```

### Task 1.2: Write the `index.html` document skeleton

**Files:**
- Create: `web/index.html`

- [ ] **Step 1: Write the skeleton**

File contents:

```html
<!DOCTYPE html>
<html lang="en-GB">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Craftsky — a social feed for textile crafters.</title>
  <meta name="description" content="A social feed for textile crafters. Share what you're making. Find the pattern everyone's talking about. No algorithm deciding what you see." />
  <link rel="canonical" href="https://craftsky.social/" />

  <!-- Open Graph -->
  <meta property="og:title" content="Craftsky — a social feed for textile crafters." />
  <meta property="og:description" content="A social feed for textile crafters. Share what you're making. Find the pattern everyone's talking about." />
  <meta property="og:type" content="website" />
  <meta property="og:url" content="https://craftsky.social/" />
  <meta property="og:image" content="https://craftsky.social/assets/og-image.png" />

  <!-- Twitter -->
  <meta name="twitter:card" content="summary_large_image" />
  <meta name="twitter:title" content="Craftsky — a social feed for textile crafters." />
  <meta name="twitter:description" content="A social feed for textile crafters. Share what you're making. Find the pattern everyone's talking about." />
  <meta name="twitter:image" content="https://craftsky.social/assets/og-image.png" />

  <!-- Icons -->
  <link rel="icon" href="/assets/favicon.svg" type="image/svg+xml" />

  <!-- Fonts: load via <link> (not CSS @import) so the preconnect hints help. -->
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=DM+Serif+Display:ital@0;1&family=Outfit:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap" />

  <link rel="stylesheet" href="styles.css" />
</head>
<body>
  <a class="skip-link" href="#main">Skip to main content</a>

  <header id="hero"><!-- Chunk 2 fills this --></header>

  <main id="main">
    <section id="value-cards"><!-- Chunk 3 --></section>
    <section id="project-posts"><!-- Chunk 3 --></section>
    <section id="why"><!-- Chunk 4 --></section>
    <section id="how-it-works"><!-- Chunk 4 --></section>
    <section id="whos-behind"><!-- Chunk 4 --></section>
    <section id="faq"><!-- Chunk 5 --></section>
    <section id="final-cta"><!-- Chunk 5 --></section>
  </main>

  <footer id="footer"><!-- Chunk 5 --></footer>

  <!-- Modal (Chunk 7) -->
  <dialog id="waiting-list-modal" aria-labelledby="waiting-list-title"></dialog>

  <script src="main.js" defer></script>
</body>
</html>
```

- [ ] **Step 2: Serve and load in a browser**

Run (from `web/`):
```bash
python3 -m http.server 8000
```

Visit `http://localhost:8000`. Expected: blank cream page (`--paper` background), Tab shows the skip link at top-left. View source confirms the skeleton is intact.

- [ ] **Step 3: Commit**

```bash
git add web/index.html
git commit -m "feat(web): add index.html skeleton with meta, fonts, skip link"
```

### Task 1.3: Add container / layout primitives

**Files:**
- Modify: `web/styles.css`

- [ ] **Step 1: Append container utilities to `styles.css`**

Append to the end of `web/styles.css`:

```css
/* =========================================================================
   Layout primitives
   ========================================================================= */

.container {
  max-width: var(--content-max);
  margin: 0 auto;
  padding: 0 var(--sp-5);
}

.container--narrow {
  max-width: var(--feed-max);
}

/* Full-bleed section with centred content */
.section {
  padding: var(--sp-8) 0;
}

.section-rule {
  border: 0;
  border-top: 2.5px solid var(--rule);
  margin: 0;
}

/* Button atoms — reused in hero, final CTA, modal */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: var(--sp-3) var(--sp-5);
  min-height: 44px;
  border-radius: var(--r-pill);
  border: 1.5px solid var(--rule);
  font-family: var(--font-ui);
  font-weight: 700;
  font-size: var(--t-body);
  text-decoration: none;
  cursor: pointer;
  background: var(--surface);
  color: var(--fg-1);
  box-shadow: var(--sh-drop-sm);
  transition: transform var(--dur-fast) var(--ease),
              box-shadow var(--dur-fast) var(--ease);
}

.btn:hover {
  transform: translate(-1px, -1px);
  box-shadow: 4px 4px 0 var(--ink);
}

.btn:active {
  transform: translate(2px, 2px);
  box-shadow: 0 0 0 var(--ink);
}

.btn--primary {
  background: var(--cobalt);
  color: #FFFFFF;
}

.btn--ghost {
  background: var(--paper-3);
  color: var(--fg-1);
}

.btn--on-cobalt {
  background: #FFFFFF;
  color: var(--cobalt);
}

/* Eyebrow chip (hero + any other eyebrow callouts) */
.eyebrow-chip {
  display: inline-block;
  padding: 4px var(--sp-3);
  background: var(--butter);
  color: var(--cobalt);
  border: 1.5px solid var(--rule);
  border-radius: var(--r-pill);
  font-family: var(--font-ui);
  font-size: var(--t-eyebrow);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.14em;
}
```

- [ ] **Step 2: Visual check**

Temporarily add `<div class="container"><button class="btn btn--primary">Test</button> <a class="btn btn--ghost" href="#">Test</a></div>` inside the empty `<header id="hero">` element. Reload — the two buttons should render with cream-page background, the primary one cobalt, pill radius, 3px hard-offset ink shadow. On hover they should shift (-1px, -1px). On click they should "press in" to (2px, 2px). Remove the test markup before committing.

- [ ] **Step 3: Commit**

```bash
git add web/styles.css
git commit -m "feat(web): add container and button primitives"
```

---

## Chunk 2: Hero

One section at a time, starting with the biggest. The hero is the most visually distinctive and sets the pattern for every following section.

### Task 2.1: Hero markup

**Files:**
- Modify: `web/index.html` (inside `<header id="hero">`)

- [ ] **Step 1: Write the hero HTML**

Replace the empty `<header id="hero">` block with:

```html
<header id="hero" class="hero">
  <div class="container hero__inner">
    <div class="hero__text">
      <span class="eyebrow-chip">Built in the open · on the AT Protocol</span>
      <h1 class="hero__title">Made with <em>stuff</em>.</h1>
      <p class="hero__lede">
        A social feed for textile crafters. Share what you're making. Find the
        pattern everyone's talking about. Follow your local shop — no algorithm
        deciding what you see.
      </p>
      <div class="hero__ctas">
        <button
          type="button"
          class="btn btn--primary js-open-waiting-list"
          data-source="hero"
        >Join the waiting list</button>
        <a
          class="btn btn--ghost js-track-spec-click"
          href="https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit"
          target="_blank"
          rel="noopener"
        >Read the spec</a>
      </div>
    </div>
    <div class="hero__illustration" aria-hidden="true">
      <!-- Inline SVG added in Task 2.3 -->
    </div>
  </div>
</header>
<hr class="section-rule" />
```

The `<em>` inside `<h1>` is semantically correct for the italic "stuff" — it's an emphasis, not just a style.

- [ ] **Step 2: Serve and confirm structure renders**

Reload `http://localhost:8000`. Expected: unstyled text appears, buttons work, links work, but layout is single-column and raw — styling lands in 2.2.

### Task 2.2: Hero styles

**Files:**
- Modify: `web/styles.css`

- [ ] **Step 1: Append hero styles**

Append:

```css
/* =========================================================================
   Hero
   ========================================================================= */

.hero {
  background: var(--paper);
  padding: var(--sp-8) 0 var(--sp-9);
}

.hero__inner {
  display: grid;
  gap: var(--sp-7);
  align-items: center;
}

@media (min-width: 820px) {
  .hero__inner {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }
}

.hero__text { display: flex; flex-direction: column; gap: var(--sp-5); }

.hero__title {
  font-family: var(--font-display);
  font-size: clamp(64px, 9vw, 128px);
  font-weight: 400;
  line-height: 0.95;
  letter-spacing: -0.025em;
  color: var(--ink);
  margin: var(--sp-3) 0 0;
}

.hero__title em {
  font-style: italic;
  color: var(--cobalt);
}

.hero__lede {
  max-width: 520px;
  font-size: 18px;
  line-height: 1.5;
  color: var(--fg-1);
}

.hero__ctas {
  display: flex;
  flex-wrap: wrap;
  gap: var(--sp-4);
  margin-top: var(--sp-3);
}

.hero__illustration {
  aspect-ratio: 1 / 1;
  max-width: 460px;
  justify-self: center;
}
```

- [ ] **Step 2: Reload and visually verify**

At ≥820px: two-column hero with text on the left, empty box on the right.
At <820px (resize to 375px): single column, illustration slot below the text, title scales down via `clamp()`, CTAs wrap if needed.

- [ ] **Step 3: Commit**

```bash
git add web/index.html web/styles.css
git commit -m "feat(web): add hero section markup and styles"
```

### Task 2.3: Hero illustration SVG

**Files:**
- Modify: `web/index.html` (inside `.hero__illustration`)

- [ ] **Step 1: Replace the illustration placeholder with inline SVG**

The illustration is four shapes on top of each other: (1) cobalt rectangle rotated slightly left, (2) sky rectangle rotated right, (3) clay rectangle rotated slightly, with a small cobalt square inside, (4) red circle in the top-right corner. Each rectangle has a hard ink drop shadow.

Replace the empty `<div class="hero__illustration" aria-hidden="true">` comment with:

```html
<div class="hero__illustration" aria-hidden="true">
  <svg viewBox="0 0 400 400" xmlns="http://www.w3.org/2000/svg" role="presentation">
    <!-- Filter for hard ink drop shadow -->
    <defs>
      <filter id="hero-drop" x="-10%" y="-10%" width="130%" height="130%">
        <feOffset dx="10" dy="10" in="SourceAlpha" result="off" />
        <feFlood flood-color="#161210" result="ink" />
        <feComposite in="ink" in2="off" operator="in" result="shadow" />
        <feMerge>
          <feMergeNode in="shadow" />
          <feMergeNode in="SourceGraphic" />
        </feMerge>
      </filter>
    </defs>

    <!-- Cobalt rectangle, tilted left -->
    <g transform="translate(40 60) rotate(-6 110 140)">
      <rect x="0" y="0" width="220" height="280" fill="#1535D6" stroke="#161210" stroke-width="3" filter="url(#hero-drop)" />
    </g>

    <!-- Sky rectangle, tilted right -->
    <g transform="translate(200 40) rotate(8 90 110)">
      <rect x="0" y="0" width="180" height="220" fill="#9BC2E6" stroke="#161210" stroke-width="3" filter="url(#hero-drop)" />
    </g>

    <!-- Clay rectangle, centred, with inset cobalt square -->
    <g transform="translate(120 200) rotate(-2 90 80)">
      <rect x="0" y="0" width="180" height="160" fill="#E27B4A" stroke="#161210" stroke-width="3" filter="url(#hero-drop)" />
      <rect x="70" y="55" width="50" height="50" fill="#1535D6" stroke="#161210" stroke-width="3" transform="rotate(18 95 80)" />
    </g>

    <!-- Red dot, top right -->
    <circle cx="340" cy="60" r="20" fill="#F03A2E" stroke="#161210" stroke-width="3" filter="url(#hero-drop)" />
  </svg>
</div>
```

Colours are hard-coded because SVG `<rect>` `fill=""` doesn't inherit CSS variables in all renderers. The colour values match the tokens in `docs/design/colors_and_type.css`.

- [ ] **Step 2: Reload and verify visually**

The illustration should show three rotated coloured rectangles with hard ink offset shadows, a small cobalt square overlapping the clay one, and a red dot. It should scale to fit the `.hero__illustration` container.

- [ ] **Step 3: Commit**

```bash
git add web/index.html
git commit -m "feat(web): add hero paper-cutout illustration SVG"
```

---

## Chunk 3: Value cards and project posts

Two sections that share card semantics. This chunk produces the first genuinely "finished-looking" screen.

### Task 3.1: Value cards markup

**Files:**
- Modify: `web/index.html` (inside `<section id="value-cards">`)

- [ ] **Step 1: Write the value cards HTML**

Replace the empty section:

```html
<section id="value-cards" class="value-cards">
  <div class="container">
    <ul class="value-cards__list">
      <li class="value-card value-card--butter">
        <h2 class="value-card__title">No ads. Ever.</h2>
        <p class="value-card__body">Not now, not later. We'll figure out sustainability together — advertising is off the table.</p>
      </li>
      <li class="value-card value-card--sky">
        <h2 class="value-card__title">Your data is yours.</h2>
        <p class="value-card__body">Built on the AT Protocol. If we ever disappear, your posts and followers don't.</p>
      </li>
      <li class="value-card value-card--lilac">
        <h2 class="value-card__title">Chronological feed.</h2>
        <p class="value-card__body">You see the people you follow, in the order they posted. No algorithmic guessing.</p>
      </li>
    </ul>
  </div>
</section>
```

### Task 3.2: Value cards styles

**Files:**
- Modify: `web/styles.css`

- [ ] **Step 1: Append styles**

```css
/* =========================================================================
   Value cards
   ========================================================================= */

.value-cards {
  padding: var(--sp-7) 0;
}

.value-cards__list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: var(--sp-5);
  grid-template-columns: 1fr;
}

@media (min-width: 820px) {
  .value-cards__list { grid-template-columns: repeat(3, 1fr); }
}

.value-card {
  padding: var(--sp-5);
  border: 1.5px solid var(--rule);
  border-radius: var(--r-3);
  box-shadow: var(--sh-drop);
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}

.value-card--butter { background: var(--butter); }
.value-card--sky    { background: var(--sky); }
.value-card--lilac  { background: var(--lilac); }

.value-card__title {
  font-family: var(--font-display);
  font-size: 28px;
  font-weight: 400;
  line-height: 1.05;
  color: var(--ink);
}

.value-card__body {
  font-weight: 500;
  color: var(--ink);
}
```

- [ ] **Step 2: Reload and check at 820px / 1200px / 360px**

Three cards side-by-side at ≥820px, stacked vertically below. Each has a hard ink offset shadow and black border.

- [ ] **Step 3: Commit**

```bash
git add web/index.html web/styles.css
git commit -m "feat(web): add value cards band"
```

### Task 3.3: Project-posts section markup

**Files:**
- Modify: `web/index.html` (inside `<section id="project-posts">`)

- [ ] **Step 1: Write the HTML**

```html
<section id="project-posts" class="project-posts section">
  <div class="container project-posts__grid">
    <div class="project-posts__copy">
      <h2 class="section-title">Post projects. Find projects.</h2>
      <p>
        Craftsky is built for sharing the things you make — sweaters in progress,
        the bag pattern you've been perfecting, the quilt you finished last
        weekend. Post the project, note the pattern, tag the fabric, and move on.
      </p>
      <p>
        When you're trying to remember what everyone did with that one Sewaholic
        pattern, you can search for it. When you want to see everything someone
        has made with Merchant &amp; Mills linen, you can find that too. The
        things crafters actually care about — fabric, pattern, technique,
        modifications — are fields, not guesses.
      </p>
    </div>
    <figure class="project-posts__card-wrap" aria-label="Example project post preview">
      <div class="project-card">
        <div class="project-card__photo-frame">
          <div class="project-card__photo" role="img" aria-label="Placeholder project photo">
            <!-- Placeholder photo area — CSS fills it with a paper swatch until a real image lands -->
          </div>
        </div>
        <div class="project-card__body">
          <h3 class="project-card__title">Wiksten Haori</h3>
          <p class="project-card__meta">Sewing · WIP · 2 days</p>
          <ul class="project-card__chips">
            <li class="ck-chip ck-chip--wip">Work in progress</li>
            <li class="ck-chip">Linen</li>
          </ul>
        </div>
      </div>
    </figure>
  </div>
</section>
```

### Task 3.4: Project-posts styles (including chip styles and section-title helper)

**Files:**
- Modify: `web/styles.css`

- [ ] **Step 1: Append styles**

```css
/* =========================================================================
   Section title helper
   ========================================================================= */

.section-title {
  font-family: var(--font-display);
  font-size: clamp(36px, 5vw, 56px);
  font-weight: 400;
  line-height: 1.05;
  color: var(--ink);
  margin: 0 0 var(--sp-4);
}

/* =========================================================================
   Chips (reused from design system — same names used in app UI kit)
   ========================================================================= */

.ck-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 12px;
  border-radius: var(--r-pill);
  font-family: var(--font-ui);
  font-size: var(--t-meta);
  font-weight: 600;
  line-height: 1.6;
  background: var(--surface);
  color: var(--fg-1);
  border: 1.5px solid var(--rule);
}
.ck-chip--wip { background: var(--butter); }

/* =========================================================================
   Project posts
   ========================================================================= */

.project-posts__grid {
  display: grid;
  gap: var(--sp-7);
  align-items: center;
}

@media (min-width: 820px) {
  .project-posts__grid { grid-template-columns: 1fr 1fr; }
}

.project-posts__copy p {
  margin-top: var(--sp-4);
  max-width: 52ch;
}

.project-posts__card-wrap {
  margin: 0;
  display: flex;
  justify-content: center;
}

.project-card {
  background: var(--paper-3);
  border: 1.5px solid var(--rule);
  border-radius: var(--r-3);
  box-shadow: var(--sh-drop);
  overflow: hidden;
  max-width: 380px;
  width: 100%;
}

.project-card__photo-frame {
  background: var(--clay);
  padding: 8px;
}

.project-card__photo {
  aspect-ratio: 4 / 5;
  background: linear-gradient(135deg, #F2CBB3 0%, #E0A884 100%);
}

.project-card__body {
  padding: var(--sp-4) var(--sp-5) var(--sp-5);
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}

.project-card__title {
  font-family: var(--font-display);
  font-size: 26px;
  font-weight: 400;
  color: var(--ink);
}

.project-card__meta {
  font-weight: 500;
  color: var(--fg-2);
  font-size: var(--t-meta);
}

.project-card__chips {
  list-style: none;
  padding: 0;
  margin: var(--sp-2) 0 0;
  display: flex;
  flex-wrap: wrap;
  gap: var(--sp-2);
}
```

Note: the project-card photo is a CSS gradient placeholder. When a real screenshot lands later (deferred item in spec §13), swap `.project-card__photo` for an `<img>`.

- [ ] **Step 2: Reload and visually verify**

At ≥820px: two columns, copy left, card right, card sits on its clay swatch. At <820px: stacked, card centred.

- [ ] **Step 3: Commit**

```bash
git add web/index.html web/styles.css
git commit -m "feat(web): add project-posts section with mocked project card"
```

---

## Chunk 4: Why / How-it-works / Who's-behind-it

Three narrower text sections. Copy is drafted here; final copy goes into PR review (spec §15).

### Task 4.1: "Why we're building this" section

**Files:**
- Modify: `web/index.html` (inside `<section id="why">`)
- Modify: `web/styles.css`

- [ ] **Step 1: Write markup**

```html
<section id="why" class="why section">
  <div class="container container--narrow">
    <h2 class="section-title">Why we're building this.</h2>
    <p>
      The social networks we used to trust turned against us. Feeds that show
      you what the algorithm wants you to see. Search that doesn't work. A
      tidal wave of ads, sponsored posts, and pay-to-play reach that crowds
      out the people you actually follow. Makers deserve better.
    </p>
    <p>
      Craftsky is a chronological feed — you see what the people you follow
      posted, in the order they posted it. No ads, now or ever. Strong search
      that finds every version of a pattern. Transparent business accounts,
      not pay-to-play. Profiles that feel like yours, not a template. Your
      posts and followers live on the AT Protocol, so if we ever disappear,
      you still have them.
    </p>
    <p>
      The <a href="https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit" target="_blank" rel="noopener">full vision doc</a>
      is open for comments — take a look.
    </p>
  </div>
</section>
```

- [ ] **Step 2: Append styles**

```css
/* =========================================================================
   Why / Who's behind — narrow text sections
   ========================================================================= */

.why,
.whos-behind {
  padding: var(--sp-8) 0;
}

.why p,
.whos-behind p {
  margin-top: var(--sp-4);
}
```

- [ ] **Step 3: Reload**

Expected: narrow (~680px) single column, serif heading, body copy, link to vision doc.

- [ ] **Step 4: Commit**

```bash
git add web/index.html web/styles.css
git commit -m "feat(web): add 'why we're building this' section"
```

### Task 4.2: "How it works" section

**Files:**
- Modify: `web/index.html` (inside `<section id="how-it-works">`)
- Modify: `web/styles.css`

- [ ] **Step 1: Write markup**

```html
<section id="how-it-works" class="how section">
  <div class="container">
    <h2 class="section-title">How it works.</h2>
    <ul class="how__list">
      <li class="how__item">
        <svg class="how__icon" viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <path d="m21 8-9 4-9-4" />
          <path d="M3 8v8l9 4 9-4V8" />
          <path d="m3 8 9-4 9 4" />
        </svg>
        <h3 class="how__heading">Your posts live on a server you control.</h3>
        <p>Every post is a record on your PDS — your corner of the AT Protocol. Craftsky reads from the public network; it doesn't own your data.</p>
      </li>
      <li class="how__item">
        <svg class="how__icon" viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="10" />
          <polygon points="16.24 7.76 14.12 14.12 7.76 16.24 9.88 9.88 16.24 7.76" />
        </svg>
        <h3 class="how__heading">Craftsky reads the network, organises it for crafters.</h3>
        <p>We index posts that use our project format and surface them in a chronological feed. You follow the people you want to see.</p>
      </li>
      <li class="how__item">
        <svg class="how__icon" viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <polyline points="16 3 21 3 21 8" />
          <line x1="4" y1="20" x2="21" y2="3" />
          <polyline points="21 16 21 21 16 21" />
          <line x1="15" y1="15" x2="21" y2="21" />
          <line x1="4" y1="4" x2="9" y2="9" />
        </svg>
        <h3 class="how__heading">Move your account any time.</h3>
        <p>If Craftsky isn't for you, take your account elsewhere. Your followers come with you.</p>
      </li>
    </ul>
  </div>
</section>
```

Icon SVGs are the Lucide `package`, `compass`, and `arrow-right-left` marks at 2px stroke.

- [ ] **Step 2: Append styles**

```css
/* =========================================================================
   How it works
   ========================================================================= */

.how {
  padding: var(--sp-8) 0;
  background: var(--paper-2);
}

.how__list {
  list-style: none;
  padding: 0;
  margin: var(--sp-6) 0 0;
  display: grid;
  gap: var(--sp-6);
  grid-template-columns: 1fr;
}

@media (min-width: 820px) {
  .how__list { grid-template-columns: repeat(3, 1fr); }
}

.how__item {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  color: var(--ink);
}

.how__icon {
  color: var(--ink);
}

.how__heading {
  font-family: var(--font-ui);
  font-size: 20px;
  font-weight: 700;
  line-height: 1.2;
  color: var(--ink);
  margin: 0;
}
```

- [ ] **Step 3: Reload and check at 820/360**

Three columns at ≥820px, stacked below. Icons render as 24×24 ink strokes.

- [ ] **Step 4: Commit**

```bash
git add web/index.html web/styles.css
git commit -m "feat(web): add how-it-works section with Lucide icons"
```

### Task 4.3: "Who's behind it" section

**Files:**
- Modify: `web/index.html` (inside `<section id="whos-behind">`)
- Modify: `web/styles.css`

- [ ] **Step 1: Write markup**

```html
<section id="whos-behind" class="whos-behind section">
  <div class="container container--narrow">
    <div class="whos-behind__accent">
      <h2 class="section-title">Who's behind it.</h2>
      <p>
        Craftsky is being built by Doug Todd, the developer behind Stash Hub,
        in the open on
        <a href="#TODO-GITHUB-URL" target="_blank" rel="noopener"><!-- FIXME: replace with real repo URL -->GitHub</a>.
        It's a community-first project — the
        <a href="https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit" target="_blank" rel="noopener">vision doc</a>
        is open for comments, the code is open for PRs, and the lexicons are
        public. We're not announcing dates. We'd rather get it right than get
        it out fast.
      </p>
    </div>
  </div>
</section>
```

- [ ] **Step 2: Append styles**

```css
.whos-behind__accent {
  border-left: 3px solid var(--cobalt);
  padding-left: var(--sp-5);
}
```

- [ ] **Step 3: Reload and verify**

- [ ] **Step 4: Commit**

```bash
git add web/index.html web/styles.css
git commit -m "feat(web): add 'who's behind it' section"
```

---

## Chunk 5: FAQ, Final CTA, Footer

### Task 5.1: FAQ

**Files:**
- Modify: `web/index.html` (inside `<section id="faq">`)
- Modify: `web/styles.css`

- [ ] **Step 1: Write markup**

```html
<section id="faq" class="faq section">
  <div class="container container--narrow">
    <h2 class="section-title">Questions you might have.</h2>

    <details class="faq__item">
      <summary>When's it launching?</summary>
      <p>We're not announcing dates. We'd rather get it right than get it out fast. Join the waiting list and you'll hear from us when there's something worth trying.</p>
    </details>

    <details class="faq__item">
      <summary>Will it be free?</summary>
      <p>Yes. No ads, ever. We're still figuring out how the project sustains itself long-term — probably a mix of optional premium features and something tied to Stash Hub — and we'll be honest when we know. In the meantime, using Craftsky costs nothing.</p>
    </details>

    <details class="faq__item">
      <summary>What's the AT Protocol?</summary>
      <p>It's the open network Bluesky is built on. Your posts live on a server you can move, your follower graph is portable, and no single company owns the data. If Craftsky disappears, your posts don't.</p>
    </details>

    <details class="faq__item">
      <summary>Can I use my Bluesky handle?</summary>
      <p>Yes. Craftsky is its own app view on top of the AT Protocol, so if you already have a Bluesky account you can sign in with that handle. Your Bluesky follows carry over — no new graph to rebuild.</p>
    </details>

    <details class="faq__item">
      <summary>Is this just for textile crafters?</summary>
      <p>To start, yes. Sewing, knitting, crochet, quilting, embroidery — the textile side of craft. The post structure is designed around those kinds of projects. Other crafts may follow, but we'd rather do one community well than spread thin.</p>
    </details>

    <details class="faq__item">
      <summary>How is this different from Instagram / Pinterest / Ravelry?</summary>
      <p>Chronological feed, no ads, no algorithmic ranking, and the structured fields crafters actually want — pattern, fabric or yarn, techniques, status. You own the data, and you can take it with you. It's not trying to replace any of those platforms — it's trying to be the one crafters asked for.</p>
    </details>
  </div>
</section>
```

- [ ] **Step 2: Append styles**

```css
/* =========================================================================
   FAQ
   ========================================================================= */

.faq {
  padding: var(--sp-8) 0;
}

.faq__item {
  border-bottom: 1.5px solid var(--rule);
  padding: var(--sp-4) 0;
}

.faq__item:first-of-type {
  border-top: 1.5px solid var(--rule);
}

.faq__item summary {
  font-family: var(--font-display);
  font-size: 24px;
  line-height: 1.2;
  cursor: pointer;
  list-style: none;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--sp-3);
  color: var(--ink);
}

/* Hide default marker in Chromium/Firefox/Safari */
.faq__item summary::-webkit-details-marker { display: none; }
.faq__item summary::marker { content: ''; }

/* Chevron indicator */
.faq__item summary::after {
  content: '';
  width: 16px;
  height: 16px;
  border-right: 2px solid var(--ink);
  border-bottom: 2px solid var(--ink);
  transform: rotate(45deg);
  transition: transform var(--dur-fast) var(--ease);
  flex-shrink: 0;
}

.faq__item[open] summary::after {
  transform: rotate(-135deg);
}

.faq__item p {
  margin-top: var(--sp-3);
  max-width: 60ch;
}
```

- [ ] **Step 3: Reload, click each summary, keyboard-tab through**

Each question should open/close, chevron should rotate. Tab key should focus each `<summary>` sequentially.

- [ ] **Step 4: Commit**

```bash
git add web/index.html web/styles.css
git commit -m "feat(web): add FAQ section with native <details> accordion"
```

### Task 5.2: Final CTA band

**Files:**
- Modify: `web/index.html`
- Modify: `web/styles.css`
- Copy: `app/assets/design/paper-grain.svg` to `web/assets/paper-grain.svg`

- [ ] **Step 1: Copy paper-grain SVG**

```bash
cp app/assets/design/paper-grain.svg web/assets/paper-grain.svg
```

- [ ] **Step 2: Write markup**

```html
<section id="final-cta" class="final-cta on-cobalt">
  <div class="container final-cta__inner">
    <h2 class="final-cta__title">Want in?</h2>
    <p class="final-cta__sub">We'll email you when there's something to see.</p>
    <button
      type="button"
      class="btn btn--on-cobalt js-open-waiting-list"
      data-source="final"
    >Join the waiting list</button>
  </div>
</section>
```

- [ ] **Step 3: Append styles**

```css
/* =========================================================================
   Final CTA band
   ========================================================================= */

.final-cta {
  background: var(--cobalt);
  color: #FFFFFF;
  padding: var(--sp-9) 0;
  position: relative;
  overflow: hidden;
}

.final-cta::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image: url('assets/paper-grain.svg');
  background-size: 400px 400px;
  opacity: 0.04;
  pointer-events: none;
}

.final-cta__inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: var(--sp-4);
  position: relative;
}

.final-cta__title {
  font-family: var(--font-display);
  font-size: clamp(48px, 7vw, 80px);
  font-weight: 400;
  line-height: 0.95;
  color: #FFFFFF;
  margin: 0;
}

.final-cta__sub {
  font-size: 18px;
  max-width: 420px;
  color: #FFFFFF;
}
```

- [ ] **Step 4: Reload, verify the cobalt band sits at the bottom of main with the paper-grain overlay subtle and readable**

- [ ] **Step 5: Commit**

```bash
git add web/index.html web/styles.css web/assets/paper-grain.svg
git commit -m "feat(web): add final CTA band with paper-grain overlay"
```

### Task 5.3: Footer and logos

**Files:**
- Create: `web/assets/logo.svg`
- Create: `web/assets/atproto-mark.svg`
- Modify: `web/index.html`
- Modify: `web/styles.css`

- [ ] **Step 1: Placeholder logo.svg**

Write `web/assets/logo.svg`:

```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 48" width="80" height="48">
  <rect x="0" y="0" width="80" height="48" fill="#F5EFE4" stroke="#161210" stroke-width="1.5" rx="4" />
  <text x="40" y="34" text-anchor="middle"
    font-family="DM Serif Display, Times New Roman, serif"
    font-size="28" fill="#1535D6">CS</text>
</svg>
```

- [ ] **Step 2: Grab the official atproto mark, with a text fallback**

**Primary path — use the real mark:**

1. Visit https://atproto.com and view source to find the footer/header logo SVG, or check https://github.com/bluesky-social/atproto for a branded mark.
2. Save the SVG to `web/assets/atproto-mark.svg`.
3. Verify it renders at 40×20 (or scale `viewBox` to suit) and uses ink (`#161210`) fills.

**Fallback — only if the brand asset can't be located in 5 minutes:**

Write the following to `web/assets/atproto-mark.svg`:

```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 20" width="40" height="20">
  <text x="20" y="15" text-anchor="middle"
    font-family="JetBrains Mono, ui-monospace, monospace"
    font-size="14" font-weight="500" fill="#161210">at://</text>
</svg>
```

Note the chosen path in the commit message so the PR reviewer knows which one shipped.

- [ ] **Step 3: Write footer markup**

```html
<footer id="footer" class="footer">
  <div class="container footer__grid">
    <div class="footer__about">
      <img src="assets/logo.svg" alt="Craftsky" width="80" height="48" />
      <p class="footer__tagline">Craftsky — a social feed for textile crafters.</p>
    </div>

    <nav class="footer__nav" aria-label="Footer">
      <ul>
        <li><a href="https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit" target="_blank" rel="noopener">Vision doc</a></li>
        <!-- FIXME: replace #TODO-GITHUB-URL with real repo URL once known -->
        <li><a href="#TODO-GITHUB-URL" target="_blank" rel="noopener">GitHub</a></li>
        <li><a href="#TODO-GITHUB-URL/tree/main/lexicon" target="_blank" rel="noopener">Lexicons</a></li>
        <li><a href="#TODO-GITHUB-URL/blob/main/docs/design/design-system.md" target="_blank" rel="noopener">Design system</a></li>
      </ul>
    </nav>

    <div class="footer__atproto">
      <p>Built on the AT Protocol.</p>
      <img src="assets/atproto-mark.svg" alt="AT Protocol" width="40" height="20" />
    </div>
  </div>

  <div class="footer__strip">
    <div class="container footer__strip-inner">
      <span>© <span id="footer-year">2026</span> Craftsky</span>
      <span class="footer__aside">Made with stuff.</span>
    </div>
  </div>
</footer>
```

- [ ] **Step 4: Append footer styles**

```css
/* =========================================================================
   Footer
   ========================================================================= */

.footer {
  background: var(--paper-2);
  border-top: 1.5px solid var(--rule);
  padding: var(--sp-7) 0 0;
}

.footer__grid {
  display: grid;
  gap: var(--sp-6);
  grid-template-columns: 1fr;
}

@media (min-width: 820px) {
  .footer__grid { grid-template-columns: 2fr 1fr 1fr; }
}

.footer__tagline {
  margin-top: var(--sp-3);
  max-width: 32ch;
  color: var(--fg-2);
}

.footer__nav ul {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}

.footer__nav a {
  color: var(--ink);
  text-decoration: none;
  font-weight: 600;
}

.footer__nav a:hover {
  text-decoration: underline;
}

.footer__atproto {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--sp-2);
  color: var(--fg-2);
}

.footer__strip {
  border-top: 1px solid var(--border-hair);
  margin-top: var(--sp-7);
  padding: var(--sp-4) 0;
  font-size: var(--t-meta);
  color: var(--fg-3);
}

.footer__strip-inner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--sp-3);
}
```

- [ ] **Step 5: Commit**

```bash
git add web/index.html web/styles.css web/assets/logo.svg web/assets/atproto-mark.svg
git commit -m "feat(web): add footer with logo, nav, atproto mark"
```

---

## Chunk 6: Accessibility sweep

Pure verification chunk — no new features, only fixes for anything axe/keyboard finds.

### Task 6.1: axe-core run

- [ ] **Step 1: Open the page in the browser**

Serve with `python3 -m http.server 8000` from `web/`. Visit `http://localhost:8000`.

- [ ] **Step 2: Install the axe DevTools browser extension**

Available for Chrome and Firefox. Free.

- [ ] **Step 3: Run the scan**

Open DevTools → axe DevTools tab → Scan All of My Page.

Expected: zero **critical** or **serious** violations. Note any moderate/minor items for the PR description but do not block merge on them.

- [ ] **Step 4: Fix any issues**

Common findings and how to address:

- **Missing heading hierarchy** — ensure the page has exactly one `<h1>` (the hero title) and that `<h2>`/`<h3>` nest correctly under it.
- **Low contrast** — verify with DevTools. The design-system tokens are WCAG AA, but if any combination fails (e.g. light-grey metadata on paper), adjust the colour.
- **Form labels** — the modal form isn't present until Chunk 7, so re-run the axe scan after that chunk.
- **Landmarks** — we have `<header>`, `<main>`, `<footer>`. If axe flags missing landmarks, add `role` attributes.

Commit any fixes as a single commit per class of issue.

### Task 6.2: Keyboard traversal

- [ ] **Step 1: Manual Tab-through**

Reload the page with focus on the URL bar. Press Tab repeatedly. Expected sequence:

1. Skip link (appears top-left on first Tab).
2. "Join the waiting list" (hero).
3. "Read the spec" (hero).
4. Each FAQ `<summary>` (six in order).
5. "Join the waiting list" (final CTA).
6. Each footer link (Vision doc, GitHub, Lexicons, Design system).

Every focus state must show a visible 2px cobalt outline. On the cobalt final-CTA band, the outline should be white.

- [ ] **Step 2: Enter/Space on buttons and links**

Press Enter on each CTA and link. CTAs that don't yet open the modal (Chunk 7) should not throw. Links should follow.

- [ ] **Step 3: Commit fixes (if any)**

```bash
git add web/styles.css web/index.html
git commit -m "fix(web): accessibility sweep — focus states and landmarks"
```

---

## Chunk 7: Waiting-list modal and PostHog

The only JS in the project.

### Task 7.1: Modal markup

**Files:**
- Modify: `web/index.html` (replace the empty `<dialog>` near the bottom)

- [ ] **Step 1: Replace the empty `<dialog id="waiting-list-modal">` with:**

```html
<dialog id="waiting-list-modal" class="modal" aria-labelledby="waiting-list-title">
  <form method="dialog" class="modal__close-form">
    <button
      type="submit"
      class="modal__close"
      value="cancel"
      aria-label="Close"
    >&times;</button>
  </form>

  <div class="modal__body" data-state="form">
    <h2 id="waiting-list-title" class="modal__title">Join the waiting list.</h2>
    <p class="modal__sub">We'll email you when there's something to see. Nothing else.</p>

    <!-- FIXME: set action="" to the chosen waiting-list provider URL. -->
    <form class="modal__form" method="POST" action="#TODO-WAITING-LIST-ENDPOINT" novalidate>
      <label class="modal__label" for="waiting-list-email">Email</label>
      <input
        class="modal__input"
        type="email"
        id="waiting-list-email"
        name="email"
        required
        autocomplete="email"
        placeholder="you@example.com"
      />
      <button type="submit" class="btn btn--primary modal__submit">Sign me up</button>
    </form>
  </div>

  <div class="modal__body" data-state="success" hidden>
    <h2 class="modal__title">Thanks.</h2>
    <p class="modal__sub">We'll be in touch when there's something worth sharing.</p>
    <form method="dialog">
      <button type="submit" class="btn btn--primary">Close</button>
    </form>
  </div>
</dialog>
```

Note: we use two `<div data-state>` containers (form + success) and toggle the `hidden` attribute from JS on submit. The success state is shown when the submit handler fires; once a real provider endpoint is plugged in, the form will POST for real and this flow stays the same.

### Task 7.2: Modal styles

**Files:**
- Modify: `web/styles.css`

- [ ] **Step 1: Append modal styles**

```css
/* =========================================================================
   Modal
   ========================================================================= */

.modal {
  max-width: 520px;
  width: calc(100% - var(--sp-6));
  padding: var(--sp-6) var(--sp-6) var(--sp-7);
  background: var(--paper-3);
  border: 1.5px solid var(--rule);
  border-radius: var(--r-4);
  box-shadow: var(--sh-drop-lg);
  position: relative;
}

.modal::backdrop {
  background: rgba(22, 18, 16, 0.5);
}

.modal__close-form {
  position: absolute;
  top: var(--sp-3);
  right: var(--sp-3);
  margin: 0;
}

.modal__close {
  width: 32px;
  height: 32px;
  background: transparent;
  border: none;
  font-size: 28px;
  line-height: 1;
  cursor: pointer;
  color: var(--ink);
  padding: 0;
}

.modal__title {
  font-family: var(--font-display);
  font-size: 32px;
  font-weight: 400;
  line-height: 1.05;
  margin: 0;
}

.modal__sub {
  margin-top: var(--sp-3);
  color: var(--fg-2);
}

.modal__form {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  margin-top: var(--sp-5);
}

.modal__label {
  font-weight: 500;
  color: var(--fg-1);
}

.modal__input {
  padding: var(--sp-3) var(--sp-4);
  border: 1.5px solid var(--rule);
  border-radius: var(--r-1);
  font-family: var(--font-ui);
  font-size: var(--t-body);
  background: var(--paper-3);
}

.modal__input:focus-visible {
  outline: 2px solid var(--cobalt);
  outline-offset: 2px;
}

.modal__submit {
  align-self: flex-start;
}
```

### Task 7.3: Write `main.js`

**Files:**
- Create: `web/main.js`

- [ ] **Step 1: Write the script**

```js
/*
 * Craftsky landing page — main.js
 *
 * Responsibilities:
 *   - Open/close the waiting-list <dialog> on CTA clicks
 *   - Dispatch two PostHog events (behind DNT + key checks)
 *   - Close modal on overlay click
 *   - Keep current year in footer up to date
 *
 * ESC-to-close and focus trap are native to <dialog> — no JS needed.
 */

(function () {
  'use strict';

  // -----------------------------------------------------------------------
  // Config
  // -----------------------------------------------------------------------

  // FIXME: replace with the real PostHog project key (public write-only key,
  // safe to commit). See https://posthog.com/docs/getting-started/install
  const POSTHOG_KEY = 'REPLACE_ME';
  const POSTHOG_HOST = 'https://eu.i.posthog.com';

  // -----------------------------------------------------------------------
  // PostHog init (DNT-aware, key-aware)
  // -----------------------------------------------------------------------

  function shouldLoadPostHog() {
    if (navigator.doNotTrack === '1') return false;
    if (POSTHOG_KEY === 'REPLACE_ME') return false;
    return true;
  }

  function loadPostHog() {
    if (!shouldLoadPostHog()) return;
    // Minimal PostHog loader. See https://posthog.com/docs/integrate/client/js
    // for the full official snippet. This trimmed version loads the script,
    // initialises with anonymous/no-cookie settings, and exposes window.posthog.
    !function (t, e) {
      var o, n, p, r;
      e.__SV ||
        ((window.posthog = e), (e._i = []),
        (e.init = function (i, s, a) {
          function g(t, e) {
            var o = e.split('.');
            2 == o.length && ((t = t[o[0]]), (e = o[1]));
            t[e] = function () { t.push([e].concat(Array.prototype.slice.call(arguments, 0))); };
          }
          (p = t.createElement('script')).type = 'text/javascript';
          p.async = !0;
          p.src = s.api_host + '/static/array.js';
          (r = t.getElementsByTagName('script')[0]).parentNode.insertBefore(p, r);
          var u = e;
          for (void 0 !== a ? (u = e[a] = []) : (a = 'posthog'),
            u.people = u.people || [],
            u.toString = function (t) { var e = 'posthog'; return 'posthog' !== a && (e += '.' + a), t || (e += ' (stub)'), e; },
            u.people.toString = function () { return u.toString(1) + '.people (stub)'; },
            o = 'capture identify alias people.set people.set_once set_config register register_once unregister opt_out_capturing has_opted_out_capturing opt_in_capturing reset isFeatureEnabled onFeatureFlags getFeatureFlag getFeatureFlagPayload reloadFeatureFlags group updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures getActiveMatchingSurveys getSurveys'.split(' '), n = 0; n < o.length; n++) g(u, o[n]);
          e._i.push([i, s, a]);
        }),
        (e.__SV = 1));
    }(document, window.posthog || []);

    window.posthog.init(POSTHOG_KEY, {
      api_host: POSTHOG_HOST,
      persistence: 'memory',
      autocapture: false,
      disable_session_recording: true,
      capture_pageview: false,
    });
  }

  function track(event, properties) {
    if (!window.posthog || typeof window.posthog.capture !== 'function') return;
    window.posthog.capture(event, properties || {});
  }

  // -----------------------------------------------------------------------
  // Modal
  // -----------------------------------------------------------------------

  const modal = document.getElementById('waiting-list-modal');
  const openButtons = document.querySelectorAll('.js-open-waiting-list');
  const form = modal && modal.querySelector('.modal__form');
  const formBody = modal && modal.querySelector('[data-state="form"]');
  const successBody = modal && modal.querySelector('[data-state="success"]');

  function openModal(source) {
    if (!modal) return;
    track('landing_cta_waiting_list_clicked', { source: source });
    if (formBody && successBody) {
      formBody.hidden = false;
      successBody.hidden = true;
    }
    modal.showModal();
  }

  openButtons.forEach(function (btn) {
    btn.addEventListener('click', function () {
      const source = btn.getAttribute('data-source') || 'unknown';
      openModal(source);
    });
  });

  // Overlay click: <dialog> clicks bubble to the dialog itself when the user
  // clicks the backdrop. Clicks on inner content have different target.
  if (modal) {
    modal.addEventListener('click', function (event) {
      if (event.target === modal) modal.close();
    });
  }

  // After the form submits, show the success state.
  // NOTE: until a real provider endpoint is set in the form action, the
  // browser will attempt to navigate to '#TODO-WAITING-LIST-ENDPOINT', which
  // means the success state won't be seen. When a provider is wired up, one
  // of two paths applies:
  //   (a) Native POST with redirect-after-submit — remove this listener.
  //   (b) Provider endpoint with no-redirect JSON response — keep this
  //       listener and preventDefault/fetch from here.
  if (form) {
    form.addEventListener('submit', function (event) {
      // Until a real provider is set, show the success state optimistically.
      if (form.action.indexOf('#TODO') !== -1) {
        event.preventDefault();
        if (formBody && successBody) {
          formBody.hidden = true;
          successBody.hidden = false;
        }
      }
    });
  }

  // -----------------------------------------------------------------------
  // Spec-click tracking
  // -----------------------------------------------------------------------

  document.querySelectorAll('.js-track-spec-click').forEach(function (el) {
    el.addEventListener('click', function () {
      track('landing_cta_spec_clicked', { source: 'hero' });
    });
  });

  // -----------------------------------------------------------------------
  // Footer year
  // -----------------------------------------------------------------------

  const yearEl = document.getElementById('footer-year');
  if (yearEl) yearEl.textContent = new Date().getFullYear();

  // -----------------------------------------------------------------------
  // Kick off
  // -----------------------------------------------------------------------

  loadPostHog();
})();
```

- [ ] **Step 2: Manual smoke test**

Reload the page.

1. Click the hero "Join the waiting list" button. Modal opens. Header reads "Join the waiting list." Close it with the X, ESC, and backdrop click — all three work.
2. Open it again and submit with an empty email → HTML5 validation blocks it.
3. Enter a valid email and submit → modal swaps to "Thanks." with a Close button.
4. Click the final-band "Join the waiting list" button → same modal opens, and the source `final` is passed.
5. Click "Read the spec" → new tab opens to the vision doc.
6. Open DevTools → Network. Confirm no PostHog script is requested (because `POSTHOG_KEY === 'REPLACE_ME'` by default).
7. In DevTools Console: temporarily set `POSTHOG_KEY = 'test'` at the top of `main.js`, reload → confirm the script loads (with a 404 or similar for the test key is fine; just confirm the network request happens) and that `window.posthog` exists. Revert.
8. In DevTools Console: simulate DNT with `Object.defineProperty(navigator, 'doNotTrack', { get: () => '1' })` before the script runs — or test in Firefox with tracking protection enabled. Confirm PostHog is not loaded.

- [ ] **Step 3: Commit**

```bash
git add web/index.html web/styles.css web/main.js
git commit -m "feat(web): add waiting-list modal and PostHog event tracking"
```

### Task 7.4: Second axe pass (now that the modal is in the DOM)

- [ ] **Step 1: Open the modal in the browser**

Click "Join the waiting list" so the modal is on screen.

- [ ] **Step 2: Run axe scan again**

Expected: zero critical/serious violations.

Common findings specific to the modal:

- **Missing labels** — the form has a `<label for="...">` already, confirm no warnings.
- **Focus not restored** — native `<dialog>` handles this; manually Tab out of the modal and press ESC to confirm.

- [ ] **Step 3: Commit fixes (if any)**

```bash
git add web/styles.css web/index.html web/main.js
git commit -m "fix(web): address a11y findings from modal axe scan"
```

---

## Chunk 8: Assets, SEO polish, README, deploy

The remaining items are placeholder assets, meta-level polish, the `web/README.md` doc, and the Cloudflare Pages config.

### Task 8.1: Favicon

**Files:**
- Create: `web/assets/favicon.svg`

- [ ] **Step 1: Write a minimal favicon**

```xml
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32" width="32" height="32">
  <rect width="32" height="32" rx="6" fill="#F5EFE4" />
  <text x="16" y="22" text-anchor="middle"
    font-family="DM Serif Display, Times New Roman, serif"
    font-size="18" fill="#1535D6">CS</text>
</svg>
```

- [ ] **Step 2: Reload the page, confirm the favicon appears in the browser tab**

- [ ] **Step 3: Commit**

```bash
git add web/assets/favicon.svg
git commit -m "feat(web): add SVG favicon"
```

### Task 8.2: Placeholder OG image

**Files:**
- Create: `web/assets/og-image.png` (1200×630)

- [ ] **Step 1: Generate a placeholder PNG**

Any method is fine — a screenshot of the page hero at 1200px width, cropped to 630px tall, saved as PNG, is the simplest route. Budget: 10 minutes. The spec explicitly allows a placeholder; a properly designed OG image is deferred.

- [ ] **Step 2: Confirm size**

Run: `file web/assets/og-image.png`
Expected: `PNG image data, 1200 x 630`.

- [ ] **Step 3: Test the Open Graph tags**

Paste the local URL into https://www.opengraph.xyz/ (after deploying to a preview URL in §8.5) — the image should appear.

- [ ] **Step 4: Commit**

```bash
git add web/assets/og-image.png
git commit -m "feat(web): add placeholder OG image"
```

### Task 8.3: `robots.txt`

**Files:**
- Create: `web/robots.txt`

- [ ] **Step 1: Write the file**

```
User-agent: *
Allow: /
```

- [ ] **Step 2: Commit**

```bash
git add web/robots.txt
git commit -m "feat(web): add robots.txt"
```

### Task 8.4: `web/README.md`

**Files:**
- Create: `web/README.md`

- [ ] **Step 1: Write the README**

```markdown
# Craftsky landing page

Single static HTML page served at https://craftsky.social.

## Contents

- `index.html` — one page, nine sections
- `styles.css` — all styles, design tokens copied from `../docs/design/colors_and_type.css`
- `main.js` — waiting-list modal + PostHog event tracking
- `assets/` — favicon, logo, atproto mark, paper-grain texture, OG image
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

```bash
diff <(sed -n '/^:root {/,/^}/p' styles.css) \
     <(sed -n '/^:root {/,/^}/p' ../docs/design/colors_and_type.css)
```

Expected: no output. If there's a diff, re-copy from the source file.

## Deploy

Cloudflare Pages is configured to watch `web/` on `main`:

- Framework preset: None
- Build command: (empty)
- Build output directory: `/`
- Root directory: `web`

Every PR gets a preview URL under `pages.dev`. Production deploys land on `craftsky.social` after merging to `main`.

## Open FIXMEs

Grep for `FIXME:` in this directory to find items still to be resolved:

- Waiting-list provider endpoint in `index.html` (modal form `action`)
- PostHog project key in `main.js`
- GitHub repo URL in footer and "Who's behind it" section
```

- [ ] **Step 2: Commit**

```bash
git add web/README.md
git commit -m "docs(web): add README with local dev and deploy notes"
```

### Task 8.5: Delete `.gitkeep` placeholders

- [ ] **Step 1: Remove stubs**

```bash
rm web/.gitkeep web/assets/.gitkeep
```

- [ ] **Step 2: Commit**

```bash
git add -u web/
git commit -m "chore(web): remove .gitkeep placeholders"
```

### Task 8.6: Cloudflare Pages setup (done from the dashboard, verified here)

These steps aren't in code. Do them manually, then verify the deploy.

- [ ] **Step 1: In the Cloudflare dashboard, connect the GitHub repo**

- [ ] **Step 2: Create a Pages project with:**

- Framework preset: **None**
- Build command: (leave empty)
- Build output directory: `/`
- Root directory: `web`
- Production branch: `main`

- [ ] **Step 3: Trigger a preview deploy from a PR, note the URL**

- [ ] **Step 4: Run Lighthouse on the preview URL**

Open Chrome DevTools → Lighthouse → "Mobile" → "Analyze page load". Expected: ≥95 on Performance, Accessibility, Best Practices, SEO.

Repeat for "Desktop".

Record the scores in the PR description. If any category is under 95, address and re-run before merging.

- [ ] **Step 5: Verify responsive at 360 / 820 / 1200px**

Use DevTools responsive mode. Take three screenshots and add to the PR description.

- [ ] **Step 6: Point DNS**

After the PR merges, in the Cloudflare dashboard under Pages → craftsky-landing → Custom domains:
- Add `craftsky.social` (apex)
- Add `www.craftsky.social` with a redirect rule to the apex

DNS propagation up to an hour.

- [ ] **Step 7: Confirm HTTPS, then announce in the PR**

---

## Chunk 9: Copy review and final polish

This chunk doesn't produce code beyond any final tweaks — it's the human checkpoint before merging.

### Task 9.1: Copy review with Doug

- [ ] **Step 1: Export the page's copy**

From the preview URL, run the page through a simple text-extraction tool (browser "Reader Mode" is fine, or `pbpaste` after select-all). Paste into a Google Doc or a PR comment.

- [ ] **Step 2: Doug reviews**

Specifically:

- Headline "Made with *stuff*." — still loves it?
- Value cards copy — voice on?
- §4.3 project-posts copy — reads right?
- §4.4 "Why we're building this" — matches the vision doc?
- §4.7 FAQ six answers — each one honest, not marketing-y?
- Final-CTA "Want in?" — right register?

- [ ] **Step 3: Apply edits as commits**

Prefer one small commit per section touched so the diff is easy to read.

### Task 9.2: Final commit and PR description

- [ ] **Step 1: Summarise in the PR description**

Include:

- Summary (2–3 sentences)
- Lighthouse scores (desktop and mobile)
- Screenshots at 360 / 820 / 1200px
- List of FIXMEs still in the codebase (waiting-list endpoint, PostHog key, GitHub URLs) with links to the open-question list in spec §14.
- Test plan checklist:
  - [ ] Load `/` and see the hero
  - [ ] Click both "Join the waiting list" buttons
  - [ ] Click "Read the spec" → vision doc opens in new tab
  - [ ] Tab through the page with keyboard
  - [ ] Open on mobile, verify responsive stack
  - [ ] axe scan shows zero critical/serious
  - [ ] `POSTHOG_KEY` is still `REPLACE_ME` (intentional for PR; plan swap in follow-up)

- [ ] **Step 2: Request review**

Merge after approval and Cloudflare Pages DNS is pointed.

---

## Acceptance sign-off

When every chunk above is complete, this plan's deliverables match the spec's acceptance criteria **except for the ones explicitly deferred**:

Met by this PR:
- Page renders at `craftsky.social` with HTTPS. (after DNS)
- All nine sections present.
- "Read the spec" CTA opens vision doc.
- DNT-enabled browsers do not load PostHog. (also: unset-key does not load PostHog)
- Responsive down to 360px.
- Lighthouse ≥95.
- axe shows zero critical/serious.
- Keyboard nav works.
- `prefers-reduced-motion` respected.
- Page weight <200 KB.
- Cloudflare Pages deploy pipeline live.

Deferred to follow-up:
- "Signup lands in the chosen provider's list" — requires provider choice (§14.1).
- PostHog events confirmed in dashboard — requires real key (§14.2).
- Footer/about GitHub links — require repo URL (§14.3).

Open a follow-up issue tracking those three so they don't get lost.
