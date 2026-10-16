# Retired prototype cleanup

Date: 16 October 2026

The standalone AppView console is the supported moderation interface. The
obsolete generated app snapshot, synthetic proof server, proof-only tests and
database bearer authenticator have been removed. The operator login test now
creates only the minimal synthetic incident it needs, without the retired
server's fixture dependency. Shared moderation tests use neutral console source
names. The workflow documents now live in `2026-10-14-moderation-console/`.

The current operator identity response and context no longer expose unused
external UI IDs or proof-only token metadata. Atmosphere DID allowlisting,
cookie sessions, current authorization and stable audit attribution remain the
active access path.

Migration 100 drops the obsolete external UI binding from `safety_operators`.
It preserves operator UUIDs, DIDs, permission sets, browser sessions and audit
history. Its downgrade restores an empty optional column; discarded bindings
cannot be recovered. Migration 93 remains unchanged because applied migration
history must remain reproducible.

The old platform identifier remains only where required to upgrade/downgrade
the historical schema, in the corresponding migration regression test, and in
historical design/validation reports. The first two experiment reports are
explicitly archived; their old commands are not current setup instructions.
Earlier reports describe the state on their recorded dates. Follow the
[current plan](00-plan.md) and [operator runbook](../../operations/moderation-operators.md)
for the supported console.

## Verification

- The migration regression failed before the cleanup migration existed and
  passed after implementation. It checks populated operator identity, audit and
  session preservation, column removal and the empty-column downgrade.
- Focused database, operator OAuth/session, middleware and route checks passed.
- The rebuilt development image passed all 43 frontend tests and its TypeScript
  check/build. Its database reports `version=100 dirty=false`; the retired
  binding column is absent and both approved admin identities remain active.
- The existing signed-in browser session survived the rebuild and migration.
  The identity request returned HTTP 200 with the same operator UUID/DID and
  the current console loaded its case queue and navigation.
- Both retained OpenAPI contracts parse as JSON. Relative links resolve in the
  renamed workflow folder and the updated runbook, ADR and API architecture spec.
- The full backend normal and race runs each passed all 47 test packages,
  without failures. The two retired proof packages are no longer part of the
  test inventory.
- `scripts/appview-check` exited successfully with all release gates passed:
  generated-code/module checks, static analysis, vulnerability checks, Docker
  builds, migration round trips/failure handling, media-memory probes, packaged
  console assets, unauthenticated identity denial and the real pinned Tap
  WebSocket handshake. No events were injected into Tap; deterministic event/ACK
  behavior remains covered by the existing fake-server tests. Local artifacts:
  `/private/tmp/craftsky-prototype-cleanup-verified/`; combined log:
  `/private/tmp/craftsky-prototype-cleanup-verified.log`.
- `git diff --check` passed.

No production mutation, remote resource deletion, deployment, commit or push is
part of this cleanup. Mobile visual acceptance remains pending as requested.
