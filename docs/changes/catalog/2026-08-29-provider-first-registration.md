---
{
  "id": "2026-08-29-provider-first-registration",
  "title": "Provider-first OAuth registration",
  "source": "docs/changes/2026-08-29-provider-first-registration",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "authentication",
    "registration",
    "federation"
  ],
  "keywords": [
    "bsky.social PAR",
    "authorization issuer",
    "registration_start",
    "handoff exchange",
    "stale exchange"
  ],
  "summary": "Implemented provider-first registration with durable handoff, stale-exchange recovery and a constrained live issuer fallback.",
  "currentDocs": [
    "docs/adr/025-browser-consumer-handoff.md",
    "docs/operations/oauth-permissions.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/auth",
    "app/lib/auth",
    "app/lib/onboarding"
  ],
  "evidence": [
    "docs/changes/2026-08-29-provider-first-registration/05-implementation-plan.md#live-registration-start-correction",
    "docs/changes/2026-08-29-provider-first-registration/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Signed-build live provider registration smoke remains a release-only validation requirement in this package."
  ]
}
---

# Provider-first OAuth registration

Registration starts with the selected provider and retains OAuth credentials server-side. Durable exchange/confirmation must handle lost responses and stale client continuations without promoting another account or leaving malformed-success credentials unreachable. Flutter maps start failures into actionable outcomes.

The final review accepts correction of credential lifecycle, stale exchange, migration shape and client error mapping. Local runtime checks also found an incrementally applied early migration shape and a Bluesky issuer whose protected-resource discovery returns 404. Replaying the final schema and allowing the exact reviewed issuer-origin fallback reached live PAR; malformed or transient discovery failures do not receive that fallback.

Automated tests and live registration-start evidence do not prove a complete account signup on a signed build. The package explicitly retains that live provider/device smoke as a release blocker. Current browser-consumer-handoff and OAuth permission guidance govern today's client handoff and authority boundaries.

Evidence: [05-implementation-plan.md](../2026-08-29-provider-first-registration/05-implementation-plan.md#live-registration-start-correction); [06-implementation-review.md](../2026-08-29-provider-first-registration/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
