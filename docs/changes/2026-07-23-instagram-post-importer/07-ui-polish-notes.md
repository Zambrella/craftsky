# UI Polish Notes: Instagram Historical Post Importer

## Summary

Simplified the standalone importer from an editorial, heavily decorated
presentation into a compact application interface suited to reviewing large
amounts of structured post data. The CraftSky palette, logo, and Outfit/DM
Serif Display fonts remain, while surfaces, typography, controls, status
summaries, review rows, and responsive spacing now prioritize hierarchy and
functionality.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User request: simplify a data-heavy application | Replaced heavy black rules, offset shadows, large radii, decorative backgrounds, and oversized controls with subtle dividers, restrained elevation, compact radii, and flatter application surfaces. | `instagram-importer/src/styles/app.css` | Done |
| UIP-002 | User request: retain existing colors and fonts | Retained the CraftSky palette and both self-hosted fonts while tightening the type scale and using the display face only where it supports page hierarchy. | `instagram-importer/src/styles/app.css` | Done |
| UIP-003 | Data-dense review workflow | Condensed the step strip, review heading, five summary statistics, filters, post cards, caption fields, media controls, action bar, progress rows, and result surfaces. | `instagram-importer/src/styles/app.css` | Done |
| UIP-004 | Responsive functionality | Made short virtualized review and skipped lists size to their actual content while retaining capped scrolling for large imports; kept the mobile action bar in document flow to prevent it obscuring post controls. | `instagram-importer/src/app/components/ReviewList.tsx`, `instagram-importer/src/styles/app.css` | Done |
| UIP-005 | Accessibility check | Darkened secondary text to meet contrast requirements and added explicit group semantics to labelled statistics and filter controls. | `instagram-importer/src/app/App.tsx`, `instagram-importer/src/app/components/ReviewList.tsx`, `instagram-importer/src/styles/app.css` | Done |
| UIP-006 | User follow-up: collapsible post details | Added independent native caption and image disclosures that start closed, show the image count in the image heading, and auto-size the expanded caption editor to its content. Retuned virtualization for the shorter collapsed rows. | `instagram-importer/src/app/components/ReviewList.tsx`, `instagram-importer/src/app/components/ReviewList.test.tsx`, `instagram-importer/src/styles/app.css` | Done |
| UIP-007 | User follow-up: disclosure arrow alignment | Replaced the font glyph with a fixed-size CSS chevron centered by the summary flex row so the indicator aligns with its text when closed and open. | `instagram-importer/src/styles/app.css` | Done |
| UIP-008 | User follow-up: hide caption repair warning | Kept confidently reversible caption repair as an internal transformation marker, but removed it from row badges, warning totals, and the warnings filter because it requires no user action. | `instagram-importer/src/domain/types.ts`, `instagram-importer/src/review/reviewState.ts`, `instagram-importer/src/app/components/ReviewList.tsx`, tests and workflow docs | Done |
| UIP-009 | User follow-up: expanded image review | Made expanded image sections progressively load every selected sanitized thumbnail through the existing serial worker queue, increased thumbnails from 44 px to 144 px, and added an accessible native-dialog full-screen lightbox with close-button, Escape, and backdrop dismissal. | `instagram-importer/src/app/components/ReviewList.tsx`, `instagram-importer/src/styles/app.css`, component/application/browser tests and workflow docs | Done |
| UIP-010 | User follow-up: importer chrome and footer | Removed the review-only disconnected-account badge, reused the supplied CraftSky app icon for the header, footer, browser icon, and OAuth metadata, and added accessible Privacy, Terms, and GitHub footer links. | `instagram-importer/src/app/App.tsx`, `instagram-importer/src/styles/app.css`, `instagram-importer/index.html`, `instagram-importer/public/app_icon.png`, OAuth metadata and tests | Done |

## Verification

- Commands run:
  - `npm run --prefix instagram-importer test -- --maxWorkers=4`
  - `npm run --prefix instagram-importer typecheck`
  - `npm run --prefix instagram-importer lint`
  - `npm run --prefix instagram-importer build`
  - `PLAYWRIGHT_CHANNEL=chrome npm run --prefix instagram-importer test:e2e -- e2e/local-review.spec.ts e2e/mocked-flow.spec.ts`
  - `git diff --check`
  - Agent Browser desktop and 390 px mobile visual checks of select, review,
    connect, and final-confirmation states
  - Agent Browser axe audit with `wcag2a,wcag2aa`
- Passing evidence:
  - 32 Vitest files and 157 tests passed.
  - The focused caption-warning regression suite passed 24 tests across the
    review list, review counts, and application flow.
  - Type checking, linting, and the production build passed. The built output
    contains the supplied PNG icon and canonical OAuth metadata URLs.
  - All six local-review and mocked publication/recovery scenarios passed in
    installed Chrome.
  - The final axe audit reported zero WCAG A/AA violations and zero incomplete
    checks on the open full-screen mobile lightbox.
  - Desktop and mobile checks confirmed that short review lists no longer
    reserve a large empty viewport and the mobile action bar does not obscure
    review controls. Additional checks covered collapsed rows, independent
    caption/image expansion, image counts, a multiline auto-sized caption, and
    centered disclosure chevrons in both open and closed states. The later
    image-review pass covered 144 px automatic thumbnails, full-screen desktop
    and mobile containment, close-button/Escape/backdrop dismissal, and focus
    restoration to the invoking thumbnail.
  - Header/footer checks confirmed the supplied app icon, exact Privacy and
    Terms URLs, repository link, and responsive single-column mobile footer;
    the removed disconnected-account badge no longer appears on review.
- Skipped checks and reason:
  - The first Playwright attempt using bundled Chromium could not start because
    that optional browser binary is not installed. The same checked scenario
    passed against installed Chrome.
  - No live OAuth/PDS, private export, deployment, or cross-browser release
    gate was exercised; the external gates from the implementation review
    remain unchanged.

## Scope Guardrails

- Requirement behavior changed: Yes
- Business logic changed: No
- APIs, data models, migrations, permissions, or dependencies changed: No
- Notes:
  - The virtual-list adjustment changes only viewport sizing. Selection,
    filtering, virtualization, publication, and privacy behavior are unchanged.
  - The user-requested follow-up changes only review presentation behavior:
    caption and image controls now start collapsed and can be expanded
    independently.
  - Expanded image sections now load sanitized thumbnails progressively
    through the existing serial worker queue; the lightbox does not persist
    source media or broaden network access.
  - The pre-existing uncommitted IPv4 loopback Vite fix and its regression test
    were preserved and were not part of this polish pass.
  - No commit, push, pull request, or deployment was performed.

## Follow-ups

- [ ] Complete the existing external/manual release gates before enabling
  production publication.
