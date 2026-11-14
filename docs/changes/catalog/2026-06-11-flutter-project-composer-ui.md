---
{
  "id": "2026-06-11-flutter-project-composer-ui",
  "title": "Flutter project composer and reusable fields",
  "source": "docs/changes/2026-06-11-flutter-project-composer-ui",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "projects",
    "composer",
    "forms",
    "images"
  ],
  "keywords": [
    "ProjectComposerSheet",
    "ComposerImageAttachmentSection",
    "multi-select",
    "alt text",
    "FormBuilder"
  ],
  "summary": "Built the project composer and closed its original photo-attachment and localization gaps in later execution loops.",
  "currentDocs": [
    "docs/adr/020-project-evolution-search-and-multiple-crafts.md",
    "app/README.md"
  ],
  "code": [
    "app/lib/projects/composer",
    "app/lib/projects/widgets",
    "app/lib/theme"
  ],
  "evidence": [
    "docs/changes/2026-06-11-flutter-project-composer-ui/05-implementation-plan.md#execution-log",
    "docs/changes/2026-06-11-flutter-project-composer-ui/05-implementation-plan.md#verification-results",
    "docs/changes/2026-06-11-flutter-project-composer-ui/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Flutter project composer and reusable fields

The project composer introduces craft-specific sections, common project metadata, local option catalogues, reusable FormBuilder fields and payload helpers. It is a separate full-screen composition path sharing existing publication/image infrastructure, not a backend schema or DTO-enum migration.

The original review correctly rejected an always-disabled add-photo control and hard-coded multi-select copy. Subsequent execution-log loops implement `ComposerImageAttachmentSection`, exercise the real provider-backed add-image and alt-text controls, retain loading-state behavior and localize reusable strings. The final plan records focused regressions, a passing full Flutter suite and further searchable-field UI adjustments. Those later entries qualify the stale Changes required review.

Current project catalogues and image rules have evolved. Read current ADRs and source before carrying forward old field inventories or layout assumptions. The package is a useful entry point for composer payload validation and shared field behavior.

Evidence: [05-implementation-plan.md](../2026-06-11-flutter-project-composer-ui/05-implementation-plan.md#execution-log); [06-implementation-review.md](../2026-06-11-flutter-project-composer-ui/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.

Section pointers: [verification-results](../2026-06-11-flutter-project-composer-ui/05-implementation-plan.md#verification-results).
