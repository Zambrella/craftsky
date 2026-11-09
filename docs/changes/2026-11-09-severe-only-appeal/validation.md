# Severe-only appeal resolution

Owner authorized the fix on 9 November 2026. Implemented locally; release and
hosted operator verification remain pending.

## Cause and resulting behavior

The console filtered `severeSuspension` out of the changed-appeal effect selector,
while requiring at least one selected effect. A pending appeal concerning only
severe suspension therefore could not be marked changed. Standalone restoration
removed the suspension but did not resolve the appeal.

The existing backend already accepts `severeSuspension` in `reversedEffects` and
atomically updates the appeal, negates the selected case effect, recomputes
standing and records the outcome/rationale. The fix exposes that supported
choice in the appeal form. A moderator can choose **Change the decision**, select
**Severe suspension**, enter rationale, review and confirm one appeal-resolution
command. Review copy states that unrelated suspension grounds can remain.

The ordinary effect-change selector and standalone restoration flow are unchanged.
No API request/response schema, route, permission, backend implementation,
migration, lexicon, PDS operation or reporting destination changed. The appeal
command retains revision checking and exact-command idempotency/recovery.
The selected effect must be active; this does not introduce an empty-effect
changed-appeal command for consequences previously reversed in another workflow.

## Verification

- Added two console interaction regressions: severe suspension as the sole effect,
  and severe suspension alongside an unselected formal warning. Both failed before
  the fix because the severe-suspension choice was absent; both passed afterward.
  They exercise rationale, review-before-write, final confirmation, existing
  request shape/idempotency key and resolved appeal state.
- Full admin suite: **57 tests passed**. TypeScript and production build passed;
  the existing HLS chunk-size advisory remains.
- Added four PostgreSQL integration scenarios using the existing appeal service:
  severe-only restoration, another case's severe suspension retained, three-strike
  suspension retained, and an unselected warning retained. Each checks changed
  status/rationale, active-effect projection, standing and exact replay without
  duplicated appeal/effect events. All four passed against disposable PostgreSQL 16.
- Related backend checks passed with a required local test database:
  `go test ./internal/moderation -run 'Appeal|Restor|EffectChange|Standing' -count=1`.
- Updated the Mermaid moderation reference and operator runbook to describe atomic
  severe-appeal reversal rather than the former UI workaround. Local links and
  `git diff --check` checked.

The database was a disposable local fixture, not production. The console checks
use mocked HTTP responses; the separate PostgreSQL tests verify transaction and
standing behavior. No real moderation decision, notification, restoration,
production data change or deployment was performed. The hosted console still
needs this frontend release and a benign operator check before live closure.
