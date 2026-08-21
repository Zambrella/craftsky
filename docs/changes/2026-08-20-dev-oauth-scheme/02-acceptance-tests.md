# Development OAuth Scheme Acceptance Tests

| Test ID | Requirements | Acceptance criteria | Observable behavior |
|---|---|---|---|
| BE-001 | DEV-OAUTH-001 | AC-001, AC-003 | Handler/config reject a disabled or production dev scheme before OAuth work. |
| BE-002 | DEV-OAUTH-001, DEV-OAUTH-002 | AC-002, AC-004 | Persisted dev handoff completes to the exact code-only custom URL; deletion uses its exact proof URL. |
| DB-001 | DEV-OAUTH-001 | AC-002 | Migration permits `dev_scheme` with no loopback URI and rejects invalid shapes; up/down/up passes. |
| FL-001 | DEV-OAUTH-003 | AC-005 | Auth client selects its handoff mode from a debug-and-define-gated policy. |
| PLAT-001 | DEV-OAUTH-004 | AC-006 | Android debug and iOS Debug register both paths; release/main configuration remains custom-scheme-free. |

## Test order

1. BE-001
2. DB-001
3. BE-002
4. FL-001
5. PLAT-001

## Verification

- Focused Go auth/config/migration tests, then relevant package tests and race tests.
- Focused Flutter auth/platform tests, `dart analyze`, and release artifact configuration tests.
- `git diff --check` and generated-file consistency.

