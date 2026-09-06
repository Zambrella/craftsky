# UI Polish Notes: Video Posts

## Summary
Restyled the selected-video attachment to use CraftSky's paper-card and form treatments and removed the publication subtext at the user's request.

## Polish Items
| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User screenshot feedback | Replaced the stock outlined card with CraftSky paper, border, radius, shadow, icon badge, and outlined action treatments. | `app/lib/feed/widgets/composer_video_attachment_card.dart` | Done |
| UIP-002 | User screenshot feedback | Replaced the stock alt-text field with `BrandTextField`. | `app/lib/feed/widgets/composer_video_attachment_card.dart` | Done |
| UIP-003 | User screenshot feedback | Removed the "Published videos are..." subtext and its spacing. | `app/lib/feed/widgets/composer_video_attachment_card.dart` | Done |
| UIP-004 | User screenshot feedback | Bound the remove-video bin icon to the semantic error color so it matches the destructive label. | `app/lib/feed/widgets/composer_video_attachment_card.dart` | Done |

## Verification
- Commands run: Focused selected-video widget tests and Dart analysis.
- Passing evidence: Selected-video card, removed-copy, and video accessibility tests pass; Dart analysis reports no errors.
- Skipped checks and reason: Full app suite is unnecessary for this local visual change.

## Scope Guardrails
- Requirement behavior changed: No.
- Business logic changed: No.
- APIs, data models, migrations, permissions, or dependencies changed: No.
- Notes: Selection, alt-text editing, replacement, and removal behavior are unchanged.

## Follow-ups
- None.
