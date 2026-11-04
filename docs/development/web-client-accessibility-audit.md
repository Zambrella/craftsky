# Web client accessibility audit — 3 November 2026

The audit uses the local AppView fixtures, Playwright keyboard interactions and
axe-core WCAG 2 A/AA and WCAG 2.1 AA rules. No live account data was changed.

## Coverage

- Expanded `authenticated-visual-parity.spec.ts` from main-content-only checks to
  the whole document. All 49 listed app screens are checked at a 402px light
  viewport and a 1280px dark viewport. This includes feeds, search, notifications,
  profile tabs and editing, posts/projects, saved/draft/scheduled posts, both
  composers, messages and settings.
- Open sponsored, language, destructive-confirmation and project-selection
  controls; project filters and multi-select choices; populated comment/reply
  branches; account and navigation sheets.
- Keyboard opening, selection, Escape dismissal and focus restoration for those
  controls. Sponsored Close and photo upload now have visible keyboard focus.
- Sign-in account suggestions and each of the three onboarding steps in both
  themes. These checks do not submit sign-in or complete onboarding.
- Public profile and project pages in both themes, keyboard skip link, no-JavaScript
  navigation, narrow viewports and missing-image fallback.

## Fixed findings

| Finding | Change |
| --- | --- |
| Dark-mode profile actions used pale profile colours behind light text; Edit profile measured 1.17:1. | Action controls use the dark surface in dark mode. Craft badges retain their existing pale background and dark text. |
| Dark Growth change badge had blue text on blue fill, 2.87:1. | Its background now uses a low-opacity primary tint over the page colour in either theme. |
| Light comment reply controls measured 3.38:1. | Darkened the light green to `#597231`, retaining green semantics and reaching 4.73:1 on paper. |
| Reply handles measured 4.29:1 on the inset reply background. | Use the secondary ink token inside reply branches. |
| Dark selected sign-in account suggestions had a 4.4:1 handle label. | Suggestion handles use secondary ink so the active/hover tint remains readable. |
| White Delete text on bright red measured 3.92:1. | Shared danger-fill token `#b82016` provides 6.45:1 with white text; draft and generic confirmations use it. |
| Sponsored options had a visually hidden but keyboard-focusable Close button. | The control becomes visible when reached with the keyboard. |
| Photo upload's transparent file input had no visible keyboard-focus indicator. | The surrounding upload target displays a focus ring when the file input has keyboard focus. |

## Evidence and limits

The initial expanded 106-test run passed 100 tests and exposed six failures from
the profile and Growth contrast issues. Populated-comment and open-confirmation
checks then exposed the additional contrast issues above. Regression tests retain
these states, rather than limiting scans to the initial empty pages.

The final consolidated visual/accessibility run passed all 111 tests. The three
populated-comment/navigation scenarios also passed on the final normal build,
for 114 authenticated scenarios including sign-in suggestions and all onboarding
steps in both themes. The nine public-page tests also passed. `git diff --check`
passed.

This is an automated and keyboard accessibility review, not a WCAG conformance
certification. VoiceOver/NVDA speech output, every possible user-generated colour
or image, browser zoom combinations and physical assistive devices have not been
exhaustively tested. The viewport/theme pairs are representative, not every
combination. Browser screenshots are emitted into the Playwright test results.
