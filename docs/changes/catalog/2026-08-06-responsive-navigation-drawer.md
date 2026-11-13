---
{
  "id": "2026-08-06-responsive-navigation-drawer",
  "title": "Responsive app drawer and extended rail",
  "source": "docs/changes/2026-08-06-responsive-navigation-drawer",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "navigation",
    "responsive-layout",
    "accessibility"
  ],
  "keywords": [
    "AppShell",
    "drawer",
    "navigation rail",
    "Profile-owned Drafts",
    "Back stack",
    "expanded semantics"
  ],
  "summary": "The mobile app change records a compact drawer and wide extended rail, moving Saved posts, Scheduled posts and Drafts under the Profile navigation branch.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/router",
    "app/lib/settings/pages/settings_page.dart"
  ],
  "evidence": [
    "docs/changes/2026-08-06-responsive-navigation-drawer/05-implementation-plan.md#final-verification",
    "docs/changes/2026-08-06-responsive-navigation-drawer/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical native edge-gesture and physical assistive-technology checks remain separate from widget tests."
  ]
}
---

# Responsive app drawer and extended rail

## Outcome and decisions

The mobile app change records a compact drawer and wide extended rail, moving Saved posts, Scheduled posts and Drafts under the Profile navigation branch. Hamburger actions reach real branch roots, secondary routes preserve Back behavior and current selection, and legal/feedback actions remain localized. Review corrections restore compact drawer affordances, repeated-selection guards and expanded/collapsed semantics without returning those personal routes to Settings.

## Evidence and limits

The early review required corrections; the final execution checklist closes the Must findings and records 72 navigation-related tests plus clean analysis. Native edge gestures and physical VoiceOver/TalkBack behavior remain device gates. A fresh formal re-review was not recorded after the correction checklist.

Historical evidence: [05-implementation-plan.md — Final Verification](../../../docs/changes/2026-08-06-responsive-navigation-drawer/05-implementation-plan.md#final-verification); [06-implementation-review.md](../../../docs/changes/2026-08-06-responsive-navigation-drawer/06-implementation-review.md).

## Current entry points

[app/README.md](../../../app/README.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
