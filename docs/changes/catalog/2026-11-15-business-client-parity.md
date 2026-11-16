---
{
  "id": "2026-11-15-business-client-parity",
  "title": "Business settings and web event/product parity",
  "source": "docs/changes/2026-11-15-business-client-parity",
  "implementation": "implemented",
  "archive": "active",
  "components": [
    "app",
    "web-client"
  ],
  "topics": [
    "business",
    "editing",
    "settings",
    "parity"
  ],
  "keywords": [
    "Business details",
    "businessDetails",
    "BusinessDetailsRoute",
    "BusinessMediaField",
    "RecordTextField",
    "character limits",
    "field placeholders",
    "Select field prompts",
    "conditional primary action",
    "Business About styling",
    "Shop CTA styling",
    "Events tab bar",
    "Event composer placeholders",
    "compose/event routes",
    "PhotoAttachments maxImages",
    "Flutter event Photo details",
    "Event detail pills and CTAs",
    "Card image inner clipping",
    "Upcoming events footer",
    "Web event detail styling",
    "Event creator profile and roles",
    "Inline creator avatar and handle",
    "Canonical event timestamps and edit 422",
    "Event owner postpone cancel and edit actions",
    "Send event via chat",
    "Your role multi-select and validation",
    "All-day event",
    "header Save",
    "preserveProducts"
  ],
  "summary": "Separates business details from personal profile editing and aligns web event/product editors and management with Flutter.",
  "currentDocs": [
    "docs/development/web-client-parity.md",
    "docs/adr/010-business-profile-event-lexicons-and-pinned-data.md"
  ],
  "code": [
    "app/lib/business/pages/business_details_settings_page.dart",
    "app/lib/business/widgets/business_profile_summary.dart",
    "app/lib/business/widgets/event_form.dart",
    "app/lib/business/pages/event_detail_page.dart",
    "app/lib/business/widgets/event_card.dart",
    "app/lib/theme/craftsky_card.dart",
    "app/lib/profile/widgets/profile_tabs/profile_events_tab.dart",
    "app/lib/feed/widgets/composer_image_attachment_section.dart",
    "app/lib/profile/pages/edit_profile_dialog.dart",
    "web-client/src/lib/features/settings/Business.svelte",
    "web-client/src/lib/features/settings/EventComposerWorkspace.svelte",
    "web-client/src/lib/features/composer/PhotoAttachments.svelte",
    "web-client/src/lib/features/settings/BusinessMediaField.svelte",
    "web-client/src/lib/RecordTextField.svelte",
    "web-client/src/lib/features/social/BusinessSummary.svelte",
    "web-client/src/lib/features/social/BusinessEventCard.svelte",
    "web-client/src/lib/features/social/Event.svelte"
  ],
  "evidence": [
    "docs/changes/2026-11-15-business-client-parity/README.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Business settings and web event/product parity

Business details now has a dedicated settings page in both clients. Web composers and management follow the native product/event layout and preserve the existing versioned record contracts. See the linked evidence for scope and validation limits.
