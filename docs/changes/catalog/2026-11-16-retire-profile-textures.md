---
{
  "id": "2026-11-16-retire-profile-textures",
  "title": "Profile background palettes and pinned customisation preview",
  "source": "docs/changes/2026-11-16-retire-profile-textures",
  "implementation": "implemented",
  "archive": "active",
  "components": [
    "app",
    "web-client",
    "appview"
  ],
  "topics": [
    "profiles",
    "customisation",
    "backgrounds"
  ],
  "keywords": [
    "Dither",
    "Grid",
    "Cross stitch",
    "Scallops",
    "Diagonal weave",
    "Crosshatch",
    "retired backgrounds",
    "sticky preview",
    "pinned preview",
    "accent-derived pigments",
    "Aurora",
    "Colour pools",
    "Sunset",
    "Pigment",
    "Marble",
    "Woven"
  ],
  "summary": "Retires six static textures, pins the live preview and derives the six original shader pigment palettes from the selected profile accent.",
  "currentDocs": [
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/profile/models/profile_customisation.dart",
    "app/lib/profile/widgets/profile_header_background.dart",
    "web-client/src/lib/features/settings/model.ts",
    "web-client/src/lib/features/social/model.ts",
    "appview/internal/api/profile_customisation.go",
    "app/lib/settings/pages/profile_customisation_page.dart",
    "web-client/src/lib/features/settings/Preferences.svelte",
    "app/shaders/profile_background.frag",
    "app/test/profile/widgets/profile_shader_background_test.dart",
    "web-client/tests/authenticated-profile-shaders.spec.ts"
  ],
  "evidence": [
    "docs/changes/2026-11-16-retire-profile-textures/README.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Profile background choices and pinned preview

Both clients remove these choices. Stored retired values fall back to None while
preserving the selected colour; AppView rejects retired values in new saves.

The live preview remains below the page title while selecting options further
down the page on mobile and web.

The six original animated backgrounds derive harmonious pigment hues and
saturation from the selected colour, with neutral pigments for Ink.
