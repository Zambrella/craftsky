---
{
  "id": "2026-10-05-business-profile-save",
  "title": "Business profile save and conflict recovery",
  "source": "docs/changes/2026-10-05-business-profile-save",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "business",
    "editing",
    "concurrency"
  ],
  "keywords": [
    "preserveProducts",
    "preserveUnknownCatalogValues",
    "new operation key",
    "record version",
    "conflict refresh"
  ],
  "summary": "Preserved independent business-product/catalogue data and added explicit retry after stale-version save conflicts.",
  "currentDocs": [
    "docs/adr/010-business-profile-event-lexicons-and-pinned-data.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/api/business_profile.go",
    "appview/internal/pdscommands",
    "app/lib/business"
  ],
  "evidence": [
    "docs/changes/2026-10-05-business-profile-save/README.md#business-profile-save-recovery"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Business profile save and conflict recovery

Business detail editing and product management replace the same versioned public profile record. Display-safe API responses may omit unsupported product fields, so this fix adds API-only preserve controls: detail saves preserve current raw products, and taxonomy replacement preserves unrecognized existing catalogue values within the Lexicon count bound. Those controls enter immutable command intent but never the PDS record.

After a version conflict, the client refreshes the owner's AppView profile without automatically writing again. It retains edited fields, updates untouched values and uses a newer record version only for an explicit retry with a fresh operation key. Failed/stale refresh and account changes preserve or fence the draft appropriately.

The README documents resulting behavior and a widget-fixture screenshot, not a complete release certificate. Current business record validation and command execution remain authoritative; this package explains why copying a filtered response into a full replacement is unsafe.

Evidence: [README.md](../2026-10-05-business-profile-save/README.md#business-profile-save-recovery). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
