---
{
  "id": "2026-08-09-profile-customisation",
  "title": "Private profile presentation customisation",
  "source": "docs/changes/2026-08-09-profile-customisation",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "profiles",
    "themes",
    "accessibility"
  ],
  "keywords": [
    "avatar border",
    "Chunky controls",
    "texture",
    "profile colours",
    "identity cache invalidation"
  ],
  "summary": "Added AppView-stored profile presentation settings with tolerant reads, strict mutations and account-fenced saves.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/api/profile_customisation.go",
    "app/lib/profile",
    "app/lib/theme"
  ],
  "evidence": [
    "docs/changes/2026-08-09-profile-customisation/05-implementation-plan.md",
    "docs/changes/2026-08-09-profile-customisation/06-implementation-review.md#test-evidence"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical NFR-005 diagnostic enhancement and MAN-001–MAN-003 remain recorded as non-blocking follow-ups."
  ]
}
---

# Private profile presentation customisation

Profile customization remains private AppView state, exposed additively through public identity hydration. Mutations accept a strict supported catalogue while reads tolerate missing or unknown individual fields with defaults. Shared avatars render borders inside their geometry; textures are bundled locally and selected colours remain scoped to profile presentation.

The final review closes six blockers: actual custom-painted controls consume the whole colour selection, identity-cache invalidation has an explicit inventory, query/index evidence uses real PostgreSQL, non-default appearance is asserted across affected Flutter surfaces and Settings lifecycle/accessibility coverage is improved.

A custom Should-level observability enhancement was explicitly deferred, while generic HTTP instrumentation remained available. Representative device, visual and assistive-technology checks were not established by unit/widget results. Consult today's profile hydrators and theme code before treating the original catalogue as current or widening the profile-local colour scope.

Evidence: [05-implementation-plan.md](../2026-08-09-profile-customisation/05-implementation-plan.md); [06-implementation-review.md](../2026-08-09-profile-customisation/06-implementation-review.md#test-evidence). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
