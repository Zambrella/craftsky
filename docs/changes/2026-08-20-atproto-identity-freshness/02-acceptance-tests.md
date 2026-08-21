# AT Protocol Identity Freshness Acceptance Tests

| Test ID | Requirements | Behaviour |
|---|---|---|
| UT-ID-001 | ID-001, ID-002, ID-006 | Composition exposes cached and authoritative directories and OAuth receives the authoritative one. |
| UT-ID-002 | ID-003, ID-009 | Exact mention resolution ignores a fresh-looking stale DB mapping, performs a fresh bidirectional lookup, and writes through the canonical mapping. |
| IT-ID-003 | ID-004, ID-009 | Deletion intent uses a fresh canonical handle; transient failure creates no intent and stale DB data is never used. |
| UT-ID-004 | ID-005 | Every handle-targeted mutation route receives the authoritative resolver while read routes retain the cached resolver. |
| IT-ID-005 | ID-007, ID-008, ID-009 | Bounded refresh handles missing/stale rows, records retry deferral on failure, avoids starvation, and skips terminal owners. |
| REG-ID-006 | ID-001..ID-009 | Focused API/auth/account-deletion/app/routes/database tests, race tests, vet, Staticcheck, formatting, and diff checks pass. |
| IT-ID-007 | ID-010, ID-012 | Ordinary Tap identity ingestion commits the receipt and one immediate refresh trigger atomically; a duplicate event does not reset retry state, and no resolver is called before return/ACK. |
| UT-ID-008 | ID-011 | Tap invalidation purges the DID plus known old/new handle entries, while successful authoritative refresh repeats the purge after verified write-through. |
| REG-ID-009 | ID-010..ID-012 | Required-PostgreSQL ingestion/Tap/API/app/migration tests, race tests, vet, Staticcheck, formatting, and diff checks pass. |

## Test order

`UT-ID-001`, `UT-ID-002`, `IT-ID-003`, `UT-ID-004`, `IT-ID-005`,
`REG-ID-006`, `IT-ID-007`, `UT-ID-008`, `REG-ID-009`.
