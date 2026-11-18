# Moderation sanction coverage — implementation and verification

Owner approved the proposal and scoped implementation on 9 November 2026.
Implementation and local checks completed on the same date. No deployment,
production migration, real sanction, outgoing message or policy publication was
performed. Earlier unrelated worktree changes are preserved.

## Implemented

- Added decision reason `illegal_content`, displayed as Other illegal content.
  Server validation requires factual evidence, exactly `otherIllegalContent`
  classification, separate nonblank private legal rationale and a safe owner
  explanation. The rationale is rejected on other/no-action decisions.
- Added strike/severe eligibility for that reason and severe eligibility for
  existing fraud/scam and self-harm reasons. Severe actions still require the
  existing separate severity rationale and supported human review. Generic
  Other remains ineligible for strikes/severe suspension. Report mappings and
  signal behavior are unchanged.
- Added migration `000122_moderation_illegal_content`, private persistence,
  admin request/history field `illegalContentRationale`, console form/review/
  history and admin OpenAPI documentation. No new route or Lexicon change.
  The database independently rejects incomplete illegal-content records.
- Added native and web owner-facing labels for the new reason and the two
  existing reasons whose severe eligibility changed; unknown fallbacks remain.
  Owner DTOs omit internal evidence, legal rationale and legal classification.
- Updated draft Guidelines, operator instructions, approval and gap records.
  The draft/publication manifest remains subject to separate publication approval.

## Local verification

All database tests used an isolated disposable PostgreSQL 16 container and a
clearly named test database, never production. The new integration test applies
all exact production up migrations, including 000122.

| Check | Result |
|---|---|
| `go test -p 2 ./internal/moderation -count=1`, with required test database | PASS, 8.432s. Includes validation, persisted private rationale, safe owner JSON, database constraint rejection, exact replay, changed-rationale replay rejection, stale revision rejection, appeal reversal and existing moderation regressions. |
| Full `./internal/api` and `./internal/routes` package tests, required test database | PASS, 333.039s / 54.668s respectively. |
| `go test ./internal/api -run 'TestModerationAdmin' -count=1`, after adding the new API field regression | PASS, 0.599s. Proves authenticated field handoff and private rationale omitted from command acknowledgement. |
| `go vet ./internal/moderation ./internal/api ./internal/routes` | PASS. |
| Admin `npm test` | PASS, 55 tests across six files. Includes new form requirements, exact confirmed payload and private operator history. |
| Admin `npm run build` | PASS, TypeScript and Vite. Existing bundle-size advisory remains. |
| Web `npm test -- src/lib/features/settings/native-settings.test.ts` | PASS, 12 tests, including all three reason labels and unknown fallback. |
| Native `flutter test --no-pub test/moderation` | PASS, 41 tests, including model decoding and rendered owner labels. |
| Native `flutter analyze --no-pub lib/moderation test/moderation` | PASS, no issues. |
| `just policy-artifact` | PASS, regenerated draft policy artifact. |
| `python3 scripts/test_online_safety_readiness.py` | PASS, seven tests. |
| `git diff --check` | PASS. |

The initial combined Go invocation compiled before a test constant typo was
corrected; the moderation package was rerun successfully. The API/routes results
above came from that combined invocation, whose aggregate exit was nonzero due
to the earlier moderation compilation failure. The later moderation run passed.
Native checks initially encountered missing generated worktree outputs and then
an absent Rive test library; generation and the repository-prescribed native
setup resolved both before the final passing run. Unrelated generated formatting
and hash churn was removed. The first Python unittest module invocation lacked
its script import path; running the repository test file directly passed.

DTD discovery reported no running debug/DTD processes, so no live hot reload or
restart could be performed. Widget/model tests are local evidence, not physical
phone or hosted end-to-end proof.

## Remaining release and operational evidence

1. Review and merge the branch, then use the approved exact-tag AppView release
   path to apply the new migration and deploy the matching console/server.
2. Release the native/web owner-label changes through their respective workflows.
3. Exercise the hosted operator confirmation/history and owner-safe history/appeal
   path with a benign synthetic case; do not apply a real sanction as a test.
4. Policy publication, final assessment/control mapping, overall CSEA calibration
   and launch approval remain separate. This record does not close those gates.

Rollback: the down migration deliberately refuses to remove support while any
`illegal_content` decision exists, preserving its reason and audit rationale.
Do not delete moderation history to force a rollback; plan an application/schema
recovery that preserves those records.
