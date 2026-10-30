# UI Polish Notes: Basic Image Editing

## Summary

Applied a consistent CraftSky dark theme to the photo editor following the maintainer's iPhone screenshot showing a light toolbar above the package's dark canvas.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | Maintainer screenshot and dark-theme request, 2026-10-30 | Supply `AppTheme.darkThemeData` to the standalone editor and package configuration for controls and export overlays. Align canvas, toolbars, foreground and crop handles with the same dark palette. | `basic_image_editor_page.dart`; `basic_image_editor_test.dart` | Done |
| UIP-002 | Maintainer request to remove edit buttons from overview thumbnails, 2026-10-30 | Remove per-photo edit overlays from the overview. Retain the Photos card's details entry and per-photo Edit actions within details, shared by post/project composers. | `composer_image_attachment_section.dart`; `composer_image_attachment_section_test.dart` | Done |

## Verification

- `just app-test test/shared/image/basic_image_editor_test.dart`: 5 tests passed, including dark editor configuration and actual toolbar theme under a light app for both composer and avatar modes.
- `fvm flutter analyze lib/shared/image/basic_image_editor_page.dart test/shared/image/basic_image_editor_test.dart` from `app/`: no issues found.
- `git diff --check`: passed.
- Overview follow-up: `just app-test test/feed/widgets/composer_image_attachment_section_test.dart test/projects/widgets/project_composer_images_test.dart`: 19 tests passed. The updated entry test verifies no overview overlays and successful editing of the selected image from details.
- Targeted Flutter analysis of the attachment section and its test: no issues found.
- No fresh physical-device or browser visual check performed. The supplied screenshot establishes the original mismatch; widget tests verify the theme wiring after the change.

## Scope Guardrails

- Requirement behavior changed: The maintainer explicitly revised the entry placement: overview thumbnail Edit actions are removed; individual editing remains within photo details. This supersedes the earlier overview-entry design and test expectation without changing image editing behavior.
- Business logic changed: No.
- APIs, data models, migrations, permissions, or dependencies changed: No.
- Changes are confined to editor colours/theme wiring and the explicitly requested overview action removal. No commits or push created.

## Follow-ups

- A fresh device check can confirm the final appearance.
