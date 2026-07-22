# Acceptance Test Specification: Instagram DM Ownership Verification And Follow Discovery

## 1. Test Strategy

This high-risk change uses layered, vertical TDD. Pure tests establish the canonical challenge grammar, exact state machines, signature and minimal-work-item rules, normalization, eligibility, retry, fixed limits, and account fencing. Database and HTTP integration tests prove current-membership authorization, ownership, uniqueness, durability, concurrency, additive import support, retention, actorless-notification coalescing, and exact wire contracts. Flutter and Go consume the same synthetic golden JSON fixtures so route/state drift is detected in both directions. Flutter tests also prove that raw exports stop at the local parser boundary and that verification/import/follow actions remain bound to the initiating account. A small set of Gherkin acceptance scenarios ties those layers into member journeys.

All behavior that does not require a real Meta app is automated. Wholly synthetic canaries and explicitly approved redacted fixtures are permitted only as controlled test inputs; tests assert that they appear solely in the intentional private persistence/API/UI locations and never in diagnostics, push, PDS records, raw-request reserialization, or committed snapshots. Real or user-derived secrets and identity data are never fixtures. Manual checks are limited to the external capability spike, real current redacted export fixtures, physical-device push lifecycle, and final visual/accessibility inspection. Production enablement remains blocked until those checks pass.

The first implementation loop is `UT-001`: generate and digest a challenge with at least 60 bits of formatted entropy, then prove that no plaintext or member data enters the stored form. The server data model and routes must not be built on an untested token grammar.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001–AC-008, AC-014, AC-015, AC-048 | AT-001, AT-002, UT-001–UT-004, IT-001–IT-005, IT-020 | Acceptance / Unit / Integration | Yes, except live Meta delivery in MAN-001 |
| BR-002 | AC-016–AC-019, AC-039 | AT-003, UT-005, UT-009, UT-010, UT-015, IT-007, IT-014 | Acceptance / Unit / Integration | Yes |
| BR-003 | AC-020–AC-025 | AT-004, UT-006, IT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| BR-004 | AC-009, AC-024, AC-026–AC-031, AC-034 | AT-004–AT-006, IT-006, IT-009–IT-012, IT-017 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-040 | AT-001, UT-008, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-002, AC-003 | AT-002, UT-001, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-004, AC-005, AC-048, AC-049 | AT-002, UT-002, IT-002, IT-020, IT-022 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-006 | UT-003, IT-003 | Unit / Integration | Yes |
| FR-005 | AC-007, AC-008, AC-041 | UT-003, UT-004, IT-003 | Unit / Integration | Yes |
| FR-006 | AC-008, AC-010, AC-011 | UT-007, IT-003, IT-004 | Unit / Integration | Yes |
| FR-007 | AC-010–AC-013 | UT-004, IT-004 | Unit / Integration | Yes |
| FR-008 | AC-014, AC-015 | AT-002, UT-002, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-012, AC-015, AC-032 | AT-006, UT-002, UT-006, IT-001, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-009, AC-031, AC-032 | AT-006, IT-006 | Acceptance / Integration | Yes |
| FR-011 | AC-033 | UT-006, IT-006 | Unit / Integration | Yes |
| FR-012 | AC-016–AC-019 | AT-003, UT-005, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-016–AC-018 | AT-003, UT-009, IT-014 | Acceptance / Unit / Integration | Yes; real current fixture in MAN-002 |
| FR-014 | AC-019, AC-020 | UT-005, UT-009 | Unit | Yes |
| FR-015 | AC-020–AC-022, AC-025, AC-048 | UT-006, IT-008, IT-009, IT-020 | Unit / Integration | Yes |
| FR-016 | AC-021, AC-023 | AT-004, IT-008 | Acceptance / Integration | Yes |
| FR-017 | AC-024, AC-025, AC-048 | AT-004, IT-009, IT-020 | Acceptance / Integration | Yes |
| FR-018 | AC-026, AC-027, AC-028, AC-048 | AT-005, AT-008, IT-007, IT-010, IT-020 | Acceptance / Integration | Yes |
| FR-019 | AC-029, AC-030 | AT-005, IT-011 | Acceptance / Integration | Yes |
| FR-020 | AC-029, AC-034–AC-037 | AT-005, AT-007, UT-012, UT-014, IT-011, IT-012, IT-017 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-034 | UT-013, IT-012, IT-017 | Unit / Integration | Yes |
| FR-022 | AC-035–AC-037 | AT-007, UT-012, UT-014, IT-012, IT-017 | Acceptance / Unit / Integration | Yes; physical provider in MAN-004 |
| FR-023 | AC-038, AC-042 | AT-007, IT-016, IT-017 | Acceptance / Integration | Yes |
| FR-024 | AC-003–AC-005, AC-009, AC-014, AC-038, AC-049 | AT-002, IT-015, IT-016, IT-022 | Acceptance / Integration | Yes |
| FR-025 | AC-016–AC-018, AC-023, AC-026, AC-027, AC-038, AC-048 | AT-003, AT-004, AT-008, IT-014–IT-016 | Acceptance / Integration | Yes |
| FR-026 | AC-024, AC-025, AC-042 | AT-004, IT-009, IT-015, IT-016 | Acceptance / Integration | Yes |
| FR-027 | AC-043 | UT-007, IT-004, MAN-001 | Unit / Integration / Manual | Yes except real messaging window |
| FR-028 | AC-028, AC-031, AC-044, AC-048 | AT-006, IT-010, IT-011, IT-020 | Acceptance / Integration | Yes |
| FR-029 | AC-032, AC-045 | IT-018 | Integration | Yes |
| FR-030 | AC-048 | AT-008, IT-002, IT-005, IT-007–IT-012, IT-020, REG-005 | Acceptance / Integration / Regression | Yes |
| NFR-001 | AC-002, AC-012 | UT-001, UT-015 | Unit | Yes |
| NFR-002 | AC-041, AC-046 | UT-016, IT-003, IT-013 | Unit / Integration | Yes; deployment check in MAN-001 |
| NFR-003 | AC-039 | UT-015, REG-007 | Unit / Regression | Yes |
| NFR-004 | AC-004, AC-005, AC-040, AC-048 | IT-002, IT-007, IT-013, IT-020, REG-001 | Integration / Regression | Yes |
| NFR-005 | AC-008, AC-010–AC-015, AC-029 | UT-002, IT-003–IT-005, IT-009, IT-011 | Unit / Integration | Yes |
| NFR-006 | AC-011, AC-040 | UT-007, UT-008, IT-004, IT-013 | Unit / Integration | Yes |
| NFR-007 | AC-038 | IT-016, IT-017, MAN-003 | Integration / Manual | Yes plus final manual inspection |
| NFR-008 | AC-042, AC-049 | UT-011, IT-015, IT-017, IT-022, REG-012 | Unit / Integration | Yes |
| NFR-009 | AC-047 | IT-019, REG-008 | Integration / Regression | Yes |
| RULE-001 | AC-002, AC-003, AC-010, AC-012 | UT-001, UT-002, IT-002, IT-004 | Unit / Integration | Yes |
| RULE-002 | AC-014, AC-015 | AT-002, UT-002, IT-005 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-015, AC-032, AC-033 | AT-006, UT-006, IT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-009, AC-020, AC-031 | AT-006, UT-006, IT-006, IT-008 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-019–AC-022 | UT-005, UT-006, UT-009, IT-008 | Unit / Integration | Yes |
| RULE-006 | AC-018, AC-022, AC-029 | AT-003, UT-006, UT-009, IT-008, IT-011 | Acceptance / Unit / Integration | Yes |
| RULE-007 | AC-016, AC-017, AC-039 | AT-003, UT-010, UT-015, IT-007, IT-014 | Acceptance / Unit / Integration | Yes |
| RULE-008 | AC-024, AC-025 | AT-004, IT-009, REG-004 | Acceptance / Integration / Regression | Yes |
| RULE-009 | AC-026, AC-027–AC-030 | AT-005, IT-010, IT-011 | Acceptance / Integration | Yes |
| RULE-010 | AC-031, AC-044 | AT-006, IT-010, REG-004 | Acceptance / Integration / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Unconfigured Integration Fails Closed

Requirement IDs: `FR-001`, `NFR-004`, `NFR-006`
Acceptance Criteria: `AC-001`, `AC-040`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/instagram_routes_test.go`, `app/test/instagram_migration/instagram_migration_page_test.dart`

```gherkin
Feature: Instagram integration availability
  Scenario: Meta-dependent work is unavailable without locking private controls
    Given AppView has no complete Instagram credential bundle or its fake Meta adapter is unavailable
    And the member already has private link, import, and suggestion state
    When the signed-in member opens Find people from Instagram and requests verification
    Then the page explains that verification is unavailable
    And AppView returns the standard unavailable error without exposing configuration
    And local parsing, import create/list/renew/delete, current suggestion review/dismiss/accept, existing-link status/disable/revoke, and notification preferences remain available when their own dependencies are healthy
    And unrelated CraftSky routes remain available
```

### AT-002: Verify And Confirm An Instagram Account

Requirement IDs: `BR-001`, `FR-002`, `FR-003`, `FR-008`, `FR-024`, `RULE-001`, `RULE-002`
Acceptance Criteria: `AC-002`–`AC-005`, `AC-014`, `AC-015`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/instagram/verification_flow_test.go`, `app/test/instagram_migration/instagram_verification_flow_test.dart`

```gherkin
Feature: Instagram ownership verification
  Scenario: Confirm the account found by a signed DM
    Given Alice creates a ten-minute verification challenge
    And a signed Meta fixture delivers that challenge from IGSID 100
    And the fake profile client returns username alice_knits
    When Alice's fixed CraftSky account polls the attempt and confirms @alice_knits with discovery enabled
    Then the public states advance from pendingDm through processing and pendingConfirmation to confirmed
    And exactly one active link is created for Alice and IGSID 100
    And no other CraftSky account can inspect or confirm the attempt
    And replaying the webhook or confirmation does not change ownership
```

### AT-003: Parse An Instagram Export Locally

Requirement IDs: `BR-002`, `FR-012`–`FR-014`, `FR-025`, `RULE-006`, `RULE-007`
Acceptance Criteria: `AC-016`–`AC-019`, `AC-039`
Priority: Must
Level: Acceptance
Automation Target: `app/test/instagram_migration/instagram_import_privacy_test.dart`

```gherkin
Feature: Private Instagram graph import
  Scenario: Select a supported JSON following export
    Given a bounded JSON fixture contains accounts-followed, follower, media, profile, and message fields
    When Alice selects the file and previews the import
    Then parsing occurs on-device
    And only accounts-followed usernames are normalized and previewed
    And follower data is discarded locally
    And the mocked AppView request contains only source type, normalized usernames, and retention consent
    And no raw JSON, filename, message, media, or unrelated value crosses the repository boundary
```

### AT-004: Review And Explicitly Accept Suggestions

Requirement IDs: `BR-003`, `FR-015`–`FR-017`, `FR-026`, `RULE-005`, `RULE-008`
Acceptance Criteria: `AC-020`–`AC-025`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/instagram/suggestion_flow_test.go`, `app/test/instagram_migration/instagram_suggestions_page_test.dart`

```gherkin
Feature: Instagram follow suggestions
  Scenario: Follow one reviewed exact match
    Given Alice imported a following handle that exactly matches Bob's active discoverable DM-verified link
    And Alice and Bob are current visible CraftSky members
    And neither member blocks the other, Alice has not muted Bob, and Alice does not already follow Bob
    When Alice reviews and explicitly accepts Bob's suggestion
    Then the same InstagramSuggestionEligibilityPolicy is evaluated immediately before the PDS write
    Then the shared follow service creates at most one app.bsky.graph.follow
    And the suggestion becomes accepted only after success or already-following
    And importing, selecting, viewing, dismissing, or a failed PDS write creates no follow
```

### AT-005: Retain Handles And Deliver A Future-Match Digest

Requirement IDs: `BR-004`, `FR-018`–`FR-021`, `RULE-006`, `RULE-009`
Acceptance Criteria: `AC-026`–`AC-030`, `AC-034`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/instagram/future_match_test.go`, `app/test/instagram_migration/instagram_future_match_test.dart`

```gherkin
Feature: Future Instagram matches
  Scenario: Several retained following handles become discoverable together
    Given Alice explicitly retained unmatched following handles within their expiry
    And two additive active imports support one prospective Alice-to-Bob suggestion
    When several matching members enable verified discovery during the same fixed five-minute window
    Then Alice receives deduplicated pending suggestions
    And AppView creates one actorless kind system instagramMatch digest for Alice with current bounded count and activity/newness
    And additions inside that fixed window schedule at most one push after the window closes
    And deleting one import keeps the suggestion while the other active import still supports it
    And renewing consent moves only that import's expiry to no later than twelve months from renewal
    And its push preference is independently disableable
    And no matched member learns that Alice imported or searched for them
```

### AT-006: Revoke Or Conflict Without Transferring Ownership

Requirement IDs: `BR-004`, `FR-009`–`FR-011`, `FR-028`, `RULE-003`, `RULE-004`, `RULE-010`
Acceptance Criteria: `AC-009`, `AC-031`–`AC-033`, `AC-044`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/instagram/link_lifecycle_test.go`, `app/test/instagram_migration/instagram_link_settings_test.dart`

```gherkin
Feature: Instagram link control and conflicts
  Scenario: A second member claims an owned Instagram identity
    Given IGSID 100 is actively linked to Alice
    When Bob confirms an attempt from IGSID 100
    Then Alice remains the owner
    And both account-link surfaces expose only a generic private conflict warning
    And pending dependent suggestions are never reassigned
    When Alice revokes the link
    Then pending suggestions and unsent match pushes are invalidated
    And previously accepted PDS follows remain unchanged
```

### AT-007: Open An Actorless Match Notification Under The Correct Account

Requirement IDs: `FR-020`–`FR-023`, `NFR-007`, `NFR-008`
Acceptance Criteria: `AC-034`–`AC-038`, `AC-042`
Priority: Must
Level: Acceptance
Automation Target: `app/test/instagram_migration/instagram_notification_open_flow_test.dart`

```gherkin
Feature: Instagram match notifications
  Scenario: Open a match notification for an inactive retained account
    Given account A is active and a bound instagramMatch notification belongs to retained account B
    When the notification is opened
    Then the app activates B before navigation
    And opens B's typed Find people from Instagram route
    And the notification contains no actor, handle, IGSID, DID, challenge, or suggestion list
    And a stale completion from A cannot update B's page
```

### AT-008: Enforce Current Membership Independently Of Session Validity

Requirement IDs: `BR-001`, `FR-003`, `FR-015`, `FR-017`, `FR-018`, `FR-025`, `FR-028`, `FR-030`, `NFR-004`
Acceptance Criteria: `AC-048`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/instagram_membership_test.go`, `app/test/instagram_migration/instagram_membership_boundary_test.dart`

```gherkin
Feature: Current CraftSky membership boundary
  Scenario: A departed member still has a cryptographically valid session
    Given Alice has a valid CraftSky bearer session and existing Instagram private state
    And Alice's DID is removed from craftsky_profiles without a terminal identity-deletion event
    When Alice calls any authenticated Instagram operation or a worker tries to match, notify, or accept for Alice
    Then each client operation returns 404 profile_not_found through the standard error envelope
    And workers perform membership inactivation rather than exposing state or surfacing a foreign-key/internal error
    And Alice's link becomes membershipInactive, discovery is off, owner imports are paused, and dependent pending suggestions and unsent system notifications are invalidated
    And the private owner data is retained under its normal retention policy
    When Alice rejoins CraftSky
    Then discovery and imports remain inactive until Alice explicitly reactivates the link and each unexpired import
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-002, NFR-001, RULE-001 | AC-002, AC-003, AC-012 | Generate, format, normalize, digest, and compare the canonical challenge. | Deterministic secure bytes; 10,000 generated values; 30-symbol `23456789ABCDEFGHJKMNPQRSTVWXYZ` alphabet; lower/upper case and outer-whitespace variants; embedded whitespace, extra prose, ambiguous characters, wrong keyed digest, synthetic DID/email/token canaries. | Exactly 13 random symbols render as `CSKY-XXXX-XXXX-XXXX-X` for about 63.8 bits; only ASCII case and outer whitespace normalize; the complete message must be the token; storage/diagnostics contain only keyed digest metadata. | `appview/internal/instagram/challenge_test.go` |
| UT-002 | FR-003, FR-008, FR-009, NFR-005, RULE-001, RULE-002 | AC-004, AC-005, AC-010, AC-014, AC-015 | Enforce the exact verification-attempt and link transition tables. | Public attempts `pendingDm`, `processing`, `pendingConfirmation`, `confirmed`, `expired`, `cancelled`, `superseded`, `rejected`, `conflicted`; links `active`, `membershipInactive`, `revoked`, `superseded`, `disputed`; every allowed/forbidden event, expiry boundary, retry code, concurrent conflict, and wrong DID. | Only §12.1 transitions succeed; terminal attempt states stay terminal and replay idempotently; `processing` is serialized; retry exhaustion has the documented safe rejected/retry code; wrong owner sees not-found; confirmation cannot precede candidate. | `appview/internal/instagram/state_test.go` |
| UT-003 | FR-004, FR-005 | AC-006–AC-008 | Verify callback query and raw-body HMAC semantics. | Valid/invalid mode/token/challenge; exact/mutated/empty/oversized bodies; missing/malformed/case variants of signature header. | Only valid verification echoes challenge; constant-time HMAC over exact bytes gates decoding; invalid/oversized inputs fail generically. | `appview/internal/instagram/webhook_test.go` |
| UT-004 | FR-005–FR-007 | AC-008, AC-010, AC-012, AC-013, AC-039 | Decode supported Meta message events and immediately reduce them to the exact minimal durable work item. | Official-account incoming text; echo/self/deleted/non-text/unsupported/wrong object/wrong account; one, 100, and 101 events; unknown fields; synthetic raw-body/message/challenge canaries. | One through 100 supported unique messages yield only keyed message-ID digest, sender IGSID, configured official-account ID, keyed normalized-challenge digest, event timestamp, and job/lease/retry fields; 101 returns a limit error with no partial work; raw ID/body/text/plain challenge and unrelated fields are absent from persistence and diagnostics. | `appview/internal/instagram/meta_payload_test.go` |
| UT-005 | FR-012, FR-014, RULE-005, RULE-006 | AC-018–AC-020 | Normalize and validate server import entries and the following-only wire boundary. | Whitespace, one `@`, case, duplicates, dots/underscores/digits, Unicode, empty/overlong/invalid values, display names, and request entries containing `direction` or follower-specific fields. | Deterministic normalized usernames; invalid values reject; strict request decoding rejects direction/follower fields; no fuzzy/display-name inference. | `appview/internal/instagram/username_test.go`, `appview/internal/api/instagram_imports_test.go` |
| UT-006 | FR-009, FR-011, FR-015, FR-017, RULE-003–RULE-006 | AC-020–AC-022, AC-025, AC-032, AC-033, AC-048 | Evaluate links, username changes, and the single `InstagramSuggestionEligibilityPolicy`. | Importer/target current or departed; active/disabled/revoked/superseded/disputed/membershipInactive link; verified/unverified; discoverable/hidden/taken-down; exact/stale handle; self/already-followed; block each direction; importer mute; IGSID/username collision; unavailable block/mute provider in production and explicit fake-test mode. | Only exact, current, conflict-free, DM-verified, discoverable imported accounts-followed evidence passes; every safety exclusion fails at match/list/notify/accept stages; missing safety data fails closed outside explicit tests; collision disputes and old suggestions invalidate without transfer. | `appview/internal/instagram/eligibility_test.go` |
| UT-007 | FR-006, FR-027, NFR-006 | AC-011, AC-043 | Classify Meta provider results and compute the fixed provider/worker retry and reply behavior. | 2xx, 4xx auth/permission/not-found/rate limit, 5xx, timeout/cancel, 64 KiB and 64 KiB+1 bodies, 5-second deadline, allowed/expired interaction window, attempts one through six, 60-second lease, 15-minute processing age. | Retry only transient cases for at most five attempts with capped backoff; Meta calls stop at 5 seconds and 64 KiB; a job cannot process past 15 minutes; replies are optional/idempotent/window-bound; cancellation is not a provider failure. | `appview/internal/instagram/meta_client_test.go`, `appview/internal/instagram/retry_test.go` |
| UT-008 | FR-001, NFR-002, NFR-006 | AC-001, AC-040, AC-046 | Validate disabled/complete/partial configuration, trusted-proxy policy, fixed hard maxima, and API URL construction. | Empty bundle; each missing/invalid secret, account ID, API version/base URL, HTTPS DM URL, timeout/worker/replica/shared-limit setting; untrusted and configured proxy chains; values at/above maxima. | Empty disables safely; partial/unsafe or limit-loosening config fails; complete produces redacted bounded options; arbitrary forwarding headers never select the limiter IP; unsafe multi-replica mode fails closed. | `appview/internal/app/instagram_config_test.go` |
| UT-009 | FR-013, FR-014, RULE-005–RULE-007 | AC-016–AC-019, AC-022 | Parse manual text and versioned accounts-followed JSON locally under the fixed file/entry caps. | Known following shapes; mixed follower/unrelated fields; changed nesting; malformed/Unicode/duplicates/follower-only/manual text; 20 MiB and 20 MiB+1 files; 10,000 and 10,001 deduplicated entries. | Returns only normalized accounts-followed usernames and bounded warnings; follower data and the raw model cannot cross parser output; follower-only/oversized/unsupported input fails before repository call. | `app/test/instagram_migration/services/instagram_import_parser_test.dart` |
| UT-010 | BR-002, RULE-006, RULE-007 | AC-016–AC-018, AC-039 | Prove the Flutter import request type cannot carry raw archive values, directions, or follower data. | Parser result plus wholly synthetic canary raw bytes, filename, JSON keys, follower username, media URL, message, and profile value. | Repository request serializes only `sourceType`, normalized username `entries`, and `retainUnmatched`; controlled canaries may exist in parser input but never in the serialized request, diagnostics, snapshots, or server persistence. | `app/test/instagram_migration/models/instagram_import_request_test.dart`, `app/test/observability/secret_scan_test.dart` |
| UT-011 | NFR-008 | AC-042 | Fence Instagram async work by account lease/generation. | Poll/import/confirm/accept/navigation started under A; switch to B; late success/error/rollback. | Late A work has no B-visible state, mutation, cache update, or navigation; current-account work proceeds. | `app/test/instagram_migration/providers/instagram_migration_provider_test.dart` |
| UT-012 | FR-020, FR-022 | AC-035–AC-037 | Decode the notification discriminated union and infer actorless `instagramMatch` destinations. | Existing `kind: social` rows; `kind: system`, `type: instagramMatch`, `system: {count, countCapped, destination}`; absent actor/source facts; malformed/unknown social and system categories; stale/current suggestions. | Known match requires no actor/URI/CID/rkey, renders generic bounded copy, targets the Instagram route, and resolves current state; social rows retain required facts; unknown system and social variants use separate safe generic behavior. | `app/test/notifications/models/notification_test.dart`, `app/test/notifications/services/notification_destination_inference_test.dart` |
| UT-013 | FR-021 | AC-034 | Enforce fixed notification scope and push-only UI model. | GET preference; patch push; patch scope; future unknown fields. | Wire scope remains `everyone`; push changes; scope mutation rejects; Flutter creates no actor-scope control and preserves unknown fields. | `appview/internal/notifications/preferences_test.go`, `app/test/notifications/models/notification_preferences_test.dart` |
| UT-014 | FR-020, FR-022 | AC-035, AC-036 | Build the private actorless feed union, push payload, and generic copy. | Count 1, 99, and 100+; countCapped; synthetic secret/identity canaries; account-subscription binding. | Feed system object has only required fields; provider data has only category, stable notification ID, opaque account binding, count/countCapped, and bounded navigation fact; copy names nobody and no social actor/AT or private canary leaks. | `appview/internal/push/payload_test.go`, `appview/internal/api/notification_store_test.go` |
| UT-015 | NFR-001, NFR-003, RULE-007 | AC-002, AC-039 | Scan server/client diagnostic and unintended-output surfaces using controlled data. | Wholly synthetic canaries, plus separately approved redacted fixtures, for challenge/digest, body/message, username, IGSID, handle list, Meta token/secret/signature, raw export/upstream response; no real or user-derived values. | Canaries occur only in the explicitly intended private test input/DB/API/UI field under test; none appears in logs, errors, spans, Sentry, metrics, push, PDS records, URLs, raw-request reserialization, `String()`/`toString`, or committed snapshots. | `appview/internal/observability/instagram_redaction_test.go`, `app/test/observability/secret_scan_test.dart` |
| UT-016 | NFR-002 | AC-041, AC-046 | Apply the exact shared abuse keys, windows, defaults, hard maxima, and response policy atomically. | Challenge DID 5/15m, device 10/15m, IP 30/15m; invalid-redemption IGSID 10/15m and IP 30/15m; confirmation DID 20/hour and device 30/hour; imports DID 10/hour and device 20/hour; webhook global 1,000/minute and IP 300/minute; lookup concurrency 20 and IGSID 5/hour; values immediately before/at/one after each boundary, concurrent requests, expiry, untrusted forwarded IP. | Client limits are generic 429; pre-auth webhook IP and post-signature global excess are generic 429 with `Retry-After: 60` and no partial persistence; per-IGSID invalid excess is terminally deduped/ignored with 200 and no lookup; lookup pressure defers durable work. Each key/window is atomic/shared, never trusts arbitrary forwarding headers, and can tighten but not exceed the maximum. | `appview/internal/instagram/limiter_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-009, FR-028, NFR-005 | AC-015, AC-028, AC-031, AC-044 | Apply/revert the private Instagram migration and enforce checked invariants. | Fresh `testdb.WithSchema`; active and departed member DIDs; every attempt/link/import/suggestion/conflict/system-event/work-item state. | Exercise check/partial-unique violations, cross-user invalidation, membership departure, explicit scoped purge, terminal identity purge, and down migration. | Tables/checks/indexes/FKs match the state and notification unions; one-to-one ownership is enforced; profile membership loss does not broadly cascade private owner data; explicit terminal/scoped purge leaves no orphaned dependent rows. | `appview/internal/db/instagram_migration_test.go` |
| IT-002 | FR-002, FR-003, FR-030, NFR-004, RULE-001 | AC-002–AC-005, AC-040, AC-048, AC-049 | Enforce authenticated attempt create/read/cancel/current ownership and exact wire contracts. | Route mux with fake clock/random/config; Alice/Bob/current/departed sessions and device IDs; shared request/response golden fixtures; every public attempt state. | Create twice; read current and by ID under both DIDs; confirm/cancel under both DIDs; cancel absent and post-tombstone IDs; cross expiry; serialize all states; send invalid/unknown/oversized bodies. | Documented success codes/camelCase match golden JSON; current returns only the caller's non-terminal attempt or `{verification: null}` after expiry; `processing` and every terminal state round-trip; reads/confirms across owners are not-found; every owned/foreign/absent/purged DELETE is the same 204 and mutates only owned state; current-member absence is 404 `profile_not_found`. | `appview/internal/api/instagram_verifications_test.go`, `appview/internal/routes/instagram_routes_test.go` |
| IT-003 | FR-004–FR-006, NFR-002, NFR-005 | AC-006–AC-008, AC-039, AC-041 | Enforce callback verification, raw signature, fixed ingress limits, exact minimal inbox, durable deduplication, and quick acknowledgement. | Real mux/database; controlled synthetic signed fixtures; blocked worker; 256 KiB boundary; 100/101 events; pre-auth IP/post-signature global/per-IGSID/worker-pressure boundaries. | GET verify; POST valid/duplicate/multi/forged/oversized/rate-limited payloads; inspect response/Retry-After, committed columns, lookup calls, and diagnostics. | Valid unique messages commit only §12.2 fields before 200; duplicates are 200/no-op; IP/global excess is generic 429/no partial work; per-IGSID invalid excess is terminally deduped with cleared sensitive fields, 200, and no lookup; worker pressure defers with 200; 256 KiB/100 pass while larger reject; no raw ID/body/text/challenge is stored/emitted. | `appview/internal/integrations/instagram_webhook_test.go` |
| IT-004 | FR-006, FR-007, FR-027, FR-030, NFR-005, NFR-006 | AC-010–AC-013, AC-043, AC-048 | Lease/process/retry a minimal webhook item through candidate lookup and optional reply. | Database attempts/work items; fake Meta profile/reply client; four workers plus a fifth; fake clock; transient/terminal results; member removed before each transition. | Race workers; replay/reorder/cancel/expire; advance 60-second leases/five attempts/15-minute age; return valid/invalid usernames; remove membership. | At most four workers run; one owns a lease and one candidate binds; transient work is bounded; sensitive work fields clear on terminal state; departed-owner work invokes inactivation and no link/notification; reply never controls correctness. | `appview/internal/instagram/dispatcher_test.go`, `appview/internal/instagram/processor_test.go` |
| IT-005 | FR-008, FR-009, FR-030, NFR-005, RULE-002, RULE-003 | AC-014, AC-015, AC-032, AC-048 | Confirm links and resolve concurrent uniqueness/current-membership outcomes. | Pending candidates for same/different DID, IGSID, username; concurrent transactions; Alice/Bob routes; token refresh/new device for same DID; member removal. | Confirm/retry/race under same DID with different valid sessions, wrong DID, and departed DID. | One active ownership result; same-DID replay is idempotent across sessions/devices; wrong owner is resource not-found; departed owner is `profile_not_found`; conflict/audit rows are generic and existing ownership never transfers. | `appview/internal/api/instagram_confirmation_test.go`, `appview/internal/instagram/link_store_test.go` |
| IT-006 | FR-010, FR-011, FR-028, FR-030, RULE-003, RULE-004, RULE-010 | AC-009, AC-031–AC-033, AC-048 | Read/update/revoke/reactivate/refresh a link and invalidate dependents. | Every exact link state, pending/accepted suggestions, system events/deliveries, conflict rows, membership depart/rejoin, fake changed username. | Toggle discovery, revoke, refresh username/collision, depart/rejoin/reactivate, and inspect owner/cross-owner status. | Wire states are exact; dependents retract/cancel; departure becomes `membershipInactive` without purge; rejoin never silently enables discovery; explicit reactivation is required; generic conflict only; accepted follows and old-username non-transfer remain. | `appview/internal/api/instagram_account_test.go`, `appview/internal/instagram/link_lifecycle_test.go` |
| IT-007 | FR-001, FR-012, FR-018, FR-030, NFR-004, RULE-006, RULE-007 | AC-001, AC-016–AC-019, AC-026–AC-028, AC-040, AC-048 | Enforce the strict following-only additive import create/list/read/retention/reactivation/delete lifecycle and ownership contract. | Authenticated routes; Meta disabled/up; zero through 10,001 valid/invalid username entries; direction/follower/raw/archive-like unknown fields; Alice/Bob/departed/rejoined sessions; retained/non-retained and two-support imports. | POST/list/page/GET; reject old directional/follower payloads; PATCH renew/withdraw/reactivate; try enabling consent after discard; DELETE owned/foreign/absent/purged IDs; remove one supporting import; simulate Meta outage. | Immutable snapshots and exact limits hold; storage has no direction or follower-count columns; non-retained unmatched rows disappear immediately and cannot be recovered; unexpired inactive imports reactivate explicitly without expiry extension; ownership/membership/outage/support behavior remains unchanged. | `appview/internal/api/instagram_imports_test.go`, `appview/internal/db/instagram_migration_test.go` |
| IT-008 | FR-015, FR-016, FR-030, RULE-004–RULE-006 | AC-020–AC-023, AC-048 | Apply the single eligibility policy during matching, persistence, listing, and hydration with opaque pagination. | Full UT-006 policy matrix, safety-source failure, fake profiles, default 20/max 50 limits, cursor churn/invalidation. | Create imports; match/persist; fail/restore safety sources; omit limit and request 50/51; change policy facts between pages. | Only currently eligible exact following rows persist/list; unavailable safety state fails closed; private metadata stays hidden; omitted limit uses 20 and 51 clamps to 50; cursor is opaque/deterministic and stale targets disappear safely. | `appview/internal/instagram/matcher_test.go`, `appview/internal/api/instagram_suggestions_test.go` |
| IT-009 | FR-015, FR-017, FR-026, FR-030, NFR-005, NFR-008, RULE-008 | AC-024, AC-025, AC-042, AC-048 | Dismiss or explicitly accept through last-moment policy revalidation and a deterministic idempotent follow operation. | Pending suggestion; fake PDS/follow service; stable suggestion operation key/rkey; delayed firehose; PDS failures; concurrent calls; policy fact changed immediately before put; Alice/Bob/departed sessions. | View/select/dismiss/accept/retry/race/switch account, then block/mute/hide/depart/follow target just before the external call. | No non-accept write; final ineligible states write nothing and invalidate/already-follow safely; eligible retries use one deterministic `putRecord`; state advances only after success/already-following; failures retry safely; owner/membership/account fences hold. | `appview/internal/api/instagram_suggestion_actions_test.go`, `appview/internal/api/follow_service_test.go` |
| IT-010 | FR-018, FR-028, FR-030, RULE-009, RULE-010 | AC-026–AC-028, AC-031, AC-044, AC-048 | Enforce exact retention, additive support, export, reversible membership inactivation, and terminal/scoped purge. | All private record classes immediately before/at/after §15 boundaries; retained/declined matched/unmatched following rows; multi-supported imports; departed/rejoined/terminal owner; accepted-follow sentinel. | Renew/withdraw/delete, resolve pending suggestions, cross each consent/non-consent aggregate/support bound, export/purge, depart/rejoin/reactivate each import, terminal-delete identity, and scoped-delete link/import. | Exact field clearing/periods hold; non-consented matched support lasts only while pending and within 12 months, aggregate metadata lasts at most its stated 90-day/12-month bound, unmatched rows do not linger; reactivation never extends retention; other support survives; departure pauses; terminal/scoped purge is narrow; accepted follow remains. | `appview/internal/instagram/retention_test.go`, `appview/internal/instagram/account_data_test.go` |
| IT-011 | FR-019, FR-020, FR-028, FR-030, RULE-006, RULE-009 | AC-029–AC-031, AC-037, AC-048 | Create, coalesce, update-newness, and retract future matches transactionally. | Retained accounts-followed handles; initial import plus every future-notification trigger; events just before/at/after five-minute window; duplicate runs; 1/99/100 suggestions; revocation, policy loss, departure. | Run initial import; then trigger link confirmation/enable/reactivation, validated username change, explicit membership reactivation, and safety restoration; add/remove matches; close windows; inspect feed/newness/outbox. | Initial import creates eligible suggestions but no notification; future triggers dedupe suggestions and open one deterministic recipient/group event per fixed window; additions update count/countCapped and `indexedAt`/newness without extending the deadline or adding a push; one push schedules at close; post-window additions create a new event; zero retracts and partial removal recounts; ineligible/departed work cancels unsent delivery. | `appview/internal/instagram/future_match_test.go`, `appview/internal/notifications/instagram_match_test.go` |
| IT-012 | FR-020–FR-022, FR-030 | AC-034–AC-037, AC-048 | Extend notification schema/API/newness/preferences/push with an exact checked social/system union. | Existing seven `kind: social` fixtures; `kind: system` Instagram match fixtures; malformed column/JSON combinations; unknown social/system type; fixed-scope preference; member/policy invalidation. | Migrate; insert invalid/valid variants; activate/read/page/count/mark seen/patch/push/retract each type. | DB checks require actor/source facts only for social and system fields only for match; API union and cursor/activity ordering match golden JSON; fixed scope rejects mutation; one bounded private push survives only while currently eligible; existing social types remain unchanged. | `appview/internal/api/instagram_notification_test.go`, `appview/internal/push/instagram_payload_test.go` |
| IT-013 | FR-001, NFR-002, NFR-004, NFR-006 | AC-001, AC-040, AC-046 | Wire fail-closed configuration, shared limiter, dependencies, route policies, worker lifecycle, and readiness before exposing routes. | Dev/prod complete/empty/partial configs; trusted/untrusted proxy setup; Postgres limiter; fake clients/workers; single/multi-replica flags; server shutdown. | Load config/dependencies before mux; start routes/workers; inspect readiness; disable/re-enable Meta dependency; shutdown with jobs leased. | Disabled mode creates no external client and gates only Meta-dependent work; complete mode wires the narrow client/dispatcher/shared limiter; policy inventory is exact; partial/unsafe scaling fails before route readiness; shutdown is bounded. | `appview/internal/app/instagram_wiring_test.go`, `appview/internal/routes/policy_test.go`, `appview/cmd/appview/instagram_lifecycle_test.go` |
| IT-014 | FR-013, FR-025, RULE-007 | AC-016–AC-018, AC-023, AC-026, AC-039, AC-040 | Make the Flutter API/repository consume the same exact wire fixtures as AppView while sending only approved contracts. | `http_mock_adapter`; shared golden success/error/state/cursor fixtures; parser outputs; unknown additive response fields; controlled synthetic raw-export canary. | Exercise every verification/account/settings/import/suggestion method, including list/detail/PATCH import and all public states/errors. | Methods, IDs, success codes, camelCase unions, null/omitted fields, cursors, and standard errors match §12.1; fixed-account Dio is used; no raw canary crosses the request boundary; response models tolerate additive fields and safe unknown client enums. | `app/test/instagram_migration/data/instagram_api_client_test.dart` |
| IT-015 | FR-024–FR-026, NFR-008 | AC-003–AC-005, AC-009, AC-014, AC-023–AC-026, AC-042 | Coordinate account-scoped verification/import/suggestion provider state. | A/B fixed repositories; controllable poll/mutation completers; expiry timer; success/error/conflict/unavailable states. | Start under A, switch to B, release stale completions; retry and perform actions. | State transitions are correct, timers stop, no stale UI/mutation/navigation crosses account, and actions remain idempotent/retryable. | `app/test/instagram_migration/providers/instagram_migration_provider_test.dart` |
| IT-016 | FR-023–FR-026, NFR-007 | AC-003, AC-005, AC-009, AC-014, AC-016–AC-018, AC-023–AC-027, AC-038, AC-048 | Render and use the typed Instagram page in every state. | Provider fakes for disabled/empty/loading/error/attempt/candidate/link/import preview/active and membership-inactive imports/suggestions/conflict; semantics tester. | Enter from Settings; copy/open/cancel/confirm/toggle/revoke/select file/delete/renew/withdraw/reactivate each import/review/select-all/dismiss/accept/retry. | Localized accessible controls/disclosures match state; pending confirmation places the selector directly below the account, defaults to discovery allowed, updates its explanation for both options, and confirms the displayed value; an import-only rejoined member can explicitly reactivate each unexpired import without silently extending consent; candidate/reason copy is accurate; select-all is limited to reviewed eligible rows. | `app/test/instagram_migration/instagram_migration_page_test.dart`, `app/test/settings/settings_page_test.dart`, `app/test/router/instagram_route_test.dart` |
| IT-017 | FR-020–FR-023, NFR-007, NFR-008 | AC-034–AC-038, AC-042 | Decode/render/configure/open actorless notifications under exact account. | Known social/match/unknown feed rows and provider payloads; A active/B retained; current/stale suggestions. | Render feed/settings; patch push; open foreground/background simulation while switching. | Match row is actorless/generic; scope hidden; push patch works; exact account activates before typed route; stale/empty current state is safe; existing categories unchanged. | `app/test/notifications/instagram_match_notification_test.dart`, `app/test/instagram_migration/instagram_notification_open_flow_test.dart` |
| IT-018 | FR-029 | AC-032, AC-045 | Exercise bounded operator conflict/job/revoke/purge/resolve commands and exact conflict states. | Isolated DB; `open`, `resolvedKeepExisting`, `resolvedRevokeExisting`, and `expired` conflicts; jobs; expired imports; controlled synthetic identity/secret canaries. | Run each CLI command with missing/wrong/correct opaque IDs, 500/501 row batches, each transition, and repeat. | Explicit opaque arguments, maximum 500-row pagination, bounded redacted output, valid audited transitions only, idempotency, evidence anonymization on expiry, and no silent transfer. | `appview/cmd/cli/instagram_test.go` |
| IT-019 | NFR-009 | AC-047 | Preserve canceled/499 observability across Instagram handlers and polling. | Handler/store/worker operations returning `context.Canceled`; real middleware/observer; genuine 5xx control. | Cancel requests and compare captured status/log/Sentry decisions. | Cancellation is classified 499/canceled and skipped by Sentry; genuine 5xx remains captured. | `appview/internal/api/instagram_observability_test.go`, `appview/internal/observability/error_classifier_test.go` |
| IT-020 | BR-001, FR-003, FR-015, FR-017, FR-018, FR-025, FR-028, FR-030, NFR-004 | AC-048 | Prove one shared current-member guard across every authenticated route and worker transition. | Valid unexpired Alice session/device ID whose DID is absent from `craftsky_profiles`; one instance of every owned resource; queued verification/match/notification/accept work; control current member. | Enumerate every authenticated Instagram route and run every worker transition for departed Alice, then reinsert her profile and retry before and after explicit link/per-import reactivation. | Every route is `404 profile_not_found` rather than resource-specific/500; no link/suggestion/system event/PDS write is created; owner state is inactivated/paused and dependent pending work cancelled; rejoin alone restores nothing; link and each unexpired import require explicit reactivation without extending consent; the current-member control still works. | `appview/internal/api/instagram_membership_test.go`, `appview/internal/instagram/membership_transition_test.go` |
| IT-021 | FR-002–FR-018, FR-020–FR-022, NFR-004 | AC-003–AC-005, AC-009, AC-014–AC-017, AC-023–AC-027, AC-032, AC-034–AC-037, AC-040 | Lock the route-by-route and notification-union wire contract with shared synthetic golden JSON. | One fixture for each §12.1 request/success/error, every public enum, social/system notification variant, default/max cursor page, optional/omitted field, privacy-preserving DELETE, and identical replay. | Serialize AppView responses and decode/encode the same corpus in Flutter; compare POST verification/import 201, reads/PATCH/actions 200, all DELETE variants 204, callback 200/403/429 plus Retry-After, and documented errors. | Go/Dart agree byte-semantically on camelCase, enums, safe unknown client behavior, opaque IDs/cursors, standard errors, absence/foreign DELETE no-ops, webhook retry contract, and actor/source omission; no internal/private field is added. | `appview/internal/api/instagram_wire_contract_test.go`, `app/test/instagram_migration/data/instagram_wire_contract_test.dart`, `docs/changes/2026-07-11-instagram-dm-verification/fixtures/instagram_wire/` |
| IT-022 | FR-003, FR-024, NFR-008 | AC-042, AC-049 | Resume one current verification attempt across Flutter page disposal without weakening account boundaries. | Alice/Bob current-attempt fakes; matching/missing/mismatched/expired secure snapshots; pending-DM, processing, and pending-confirmation server states; controllable late reads. | Create, dispose, reopen, switch accounts, expire, cancel, confirm, and simulate server supersession. | AppView current state is authoritative; matching local display data is restored only for its DID and verification ID; polling/confirmation resumes; missing display data is not reconstructed; terminal/mismatch/session-invalidated snapshots clear narrowly; no reopen creates a new attempt. | `appview/internal/instagram/verification_store_test.go`, `appview/internal/api/instagram_verifications_test.go`, `app/test/instagram_migration/data/instagram_verification_storage_test.dart`, `app/test/instagram_migration/providers/instagram_migration_provider_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing `/v1/*` auth, device ID, body limits, route policy inventory, camelCase, and error envelopes remain consistent. | NFR-004 | AC-004, AC-005, AC-040 | Extend route-policy/contract tests and rerun representative auth/error middleware suites. |
| REG-002 | Existing seven notification categories retain actor/source semantics, preference defaults, feed hydration, newness, retraction, and push behavior. | FR-020, FR-021 | AC-034–AC-037 | Run all AppView notification/index/push tests with `instagramMatch` added only through the system path. |
| REG-003 | Existing Flutter social and unknown notification rows/settings/open destinations remain functional and forward compatible. | FR-020–FR-023 | AC-034–AC-038 | Run notification model/page/settings/destination/navigation/open suites after actorless model changes. |
| REG-004 | Ordinary profile follow/unfollow remains explicit and idempotent; imports never bypass it; no accepted follow is deleted by Instagram lifecycle. | FR-017, RULE-008, RULE-010 | AC-024, AC-025, AC-031, AC-044 | Run existing follow handler/store/provider suites plus follow-service extraction tests and accepted-follow lifecycle sentinel. |
| REG-005 | Current CraftSky membership and moderation/visibility boundaries remain authoritative even while an old session remains valid. | FR-015, FR-030 | AC-020, AC-021, AC-048 | Run profile/search/moderation tests; assert Instagram uses the shared current-member guard and one named eligibility policy, returns `profile_not_found` after departure, and fails closed when required block/mute state is unavailable. |
| REG-006 | No PDS token, Meta secret, private import/link value, or new lexicon is introduced into Flutter/PDS records. | BR-002, NFR-003, NFR-006 | AC-039, AC-040 | Run controlled-canary scans, inspect dependency/API boundaries, and assert `lexicon/` is unchanged; do not introduce real or user-derived fixtures. |
| REG-007 | Existing observability redaction, bounded attributes, and Sentry privacy remain intact. | NFR-003 | AC-039 | Run AppView/Flutter observability and secret-scan suites with Instagram sentinels. |
| REG-008 | Client cancellation remains 499/no-Sentry while real server/provider failures retain existing error capture. | NFR-009 | AC-047 | Run shared error-classifier, route metrics/logging, and cancellation regressions. |
| REG-009 | Multi-account auth, notification binding, and account-scoped providers remain isolated outside the new feature. | NFR-008 | AC-042 | Run account-switch routing, boundary, fixed-account Dio, and notification-open suites. |
| REG-010 | Database migration up/down and clean bootstrap still work from an empty schema without turning membership departure into broad private-data deletion. | FR-009, FR-020, FR-028, FR-030 | AC-015, AC-028, AC-034, AC-044, AC-048 | Run migration/full-schema setup, membership removal/rejoin, scoped purge, and terminal identity purge after the new migration. |
| REG-011 | AppView and Flutter cannot silently diverge on Instagram states, fields, notification unions, success codes, or errors. | FR-002–FR-025, NFR-004 | AC-003–AC-005, AC-009, AC-014–AC-017, AC-023–AC-027, AC-032, AC-034–AC-040 | Run both consumers against TD-011 and fail on an unreviewed golden-fixture delta. |
| REG-012 | Resumable verification storage cannot outlive or cross its owning account boundary. | FR-024, NFR-008 | AC-042, AC-049 | Run account-switch/session-invalidation tests with two independent snapshots and controlled late completions; verify only the invalidated account is cleared. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Stable CraftSky actors | Alice/Bob/Carol current-member DIDs; Alice with valid session after profile removal and after rejoin; hidden/taken-down DID; separate sessions/devices for the same DID; cross-DID sessions; active-follow, block-both-directions, and importer-mute variants. | AT-001–AT-008, IT-001–IT-021 |
| TD-002 | Challenge/state fixtures | Deterministic secure random stream; canonical 30-symbol alphabet and 13-symbol `CSKY-XXXX-XXXX-XXXX-X` token; keyed digest secret; fake clock around ten-minute boundaries; every exact attempt/link/import/suggestion/conflict state and allowed/forbidden transition. | AT-002, UT-001, UT-002, IT-001, IT-002, IT-004–IT-006, IT-021 |
| TD-003 | Meta callback fixtures | Wholly synthetic exact raw valid/mutated JSON bytes/signatures; one, 100, and 101 incoming events with `mid`, sender/recipient IDs, and text; echo/self/deleted/non-text/unsupported/wrong-account/unknown variants; expected minimal hashed work rows. | UT-003, UT-004, IT-003, IT-004, MAN-001 |
| TD-004 | Meta profile/reply fixtures | IGSID 100/200; valid/changed/missing/invalid usernames; 2xx/4xx/429/5xx/timeout bodies with secrets removed; messaging-window boundaries. | UT-006, UT-007, IT-004–IT-006 |
| TD-005 | Import/parser fixtures | Manual lines and wholly synthetic versioned accounts-followed JSON with follower and unrelated media/message/profile canaries; changed/malformed/20 MiB boundary/Unicode/duplicate/follower-only variants; approved redacted current following shapes only in the separate manual-fixture lane. | AT-003, UT-005, UT-009, UT-010, IT-007, IT-014, MAN-002 |
| TD-006 | Eligibility/link matrix | Every exact link state; discovery/verification/conflict/current-username facts; same IGSID/username; old/new usernames; importer/target membership; self/follow; hide/takedown; block either direction; importer mute; safety-source outage. | AT-004–AT-006, AT-008, UT-006, IT-005, IT-006, IT-008–IT-012, IT-020 |
| TD-007 | Import/retention matrix | Two additive accounts-followed imports that jointly support one suggestion; retained/declined following entries; creation/latest renewal and each §15 private-record boundary; deletion/withdrawal/expiry/membership-inactivation/reactivation/terminal-purge variants. | AT-005, AT-008, IT-001, IT-007, IT-010, IT-011, IT-020 |
| TD-008 | Notification matrix | Existing seven `kind: social` categories; exact `kind: system` match union; unknown kind/social/system; counts 1/99/100; five-minute boundary; active/retracted events; pending/retry/leased/sent deliveries; A/B account bindings. | AT-005, AT-007, UT-012–UT-014, IT-011, IT-012, IT-017, IT-021 |
| TD-009 | Controlled privacy canaries | Unique wholly synthetic challenge/digest, username, IGSID, handle list, webhook message/body, Meta token/app secret/verify token/signature, raw export filename/content, and upstream response; separate explicitly approved redacted fixture lane; no real/user-derived values. | UT-004, UT-010, UT-014, UT-015, IT-003, IT-007, IT-014, IT-018, REG-006, REG-007 |
| TD-010 | Concurrency/work traces | Barriers/completers for duplicate webhook, four workers plus fifth claimant, concurrent confirmation/acceptance, PDS failure/firehose delay, safety change at final revalidation, membership removal, account switch, and late response. | UT-002, UT-011, IT-003–IT-005, IT-009, IT-011, IT-015, IT-017, IT-020 |
| TD-011 | Shared wire golden corpus | Synthetic JSON for every §12.1 request/success/error; all attempt/link/import/suggestion/conflict enums; social/system unions; optional/omitted fields; 20/50 pages; owned/foreign/absent/purged DELETE 204; callback 200/403/429 and Retry-After; repeated idempotent results. | IT-002, IT-005–IT-009, IT-012, IT-014, IT-021 |
| TD-012 | Limit boundaries | Exact §12.4 maxima and values one below/at/one above: 256 KiB/100 events, rate buckets and response policy, one MiB/10,000 imports, 20 MiB client file, 20/50 pagination, 5-second/64 KiB Meta calls, concurrency 20/four workers, 60-second lease/five attempts/15 minutes, five-minute digest/count 99, 500-row operator batch. | UT-004, UT-007–UT-009, UT-016, IT-003, IT-004, IT-007, IT-008, IT-011, IT-018 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-001, FR-004–FR-007, FR-027, NFR-002, NFR-006 | AC-006–AC-013, AC-040, AC-041, AC-043, AC-046 | Meta capability and production-configuration spike. | Create/configure the owned professional account and Meta Business app; provision secrets; subscribe HTTPS webhook; send from an unrelated personal account; inspect signed payload/IGSID; fetch username; send allowed reply; validate Standard/Advanced access, Live mode, token renewal, privacy/deletion/review and deployment/shared-limit requirements. | Real redacted fixtures match or update adapters; personal sender verifies; profile/reply work; integration remains disabled until every checklist item passes. |
| MAN-002 | FR-013 | AC-016–AC-019 | Current Instagram export compatibility. | Obtain consented redacted current accounts-followed JSON exports; import them on each supported platform without network inspection shortcuts. | Parser recognizes supported following shapes locally, sends only normalized usernames, ignores follower data, and provides clear local guidance for unsupported variants. |
| MAN-003 | FR-023–FR-026, NFR-007 | AC-003, AC-009, AC-014, AC-016–AC-018, AC-023–AC-026, AC-038 | Final responsive, accessibility, clipboard, file-picker, and external-link behavior. | On phone/tablet/desktop targets, use keyboard/screen reader; inspect long lists, errors, disabled integration, consent copy, candidate confirmation, picker cancel, DM link, selection and conflict states. | Focus/semantics/copy make identity and consent unambiguous; no raw data or server error is exposed; native interactions work safely. |
| MAN-004 | FR-020–FR-023 | AC-035–AC-037, AC-042 | Physical-device push lifecycle for an inactive account. | Retain two accounts; produce single/digest matches; disable/enable push; deliver foreground/background/terminated pushes; revoke before delivery; open after suggestions disappear. | Copy names nobody; correct account activates before route; disabled/retracted work does not deliver; stale opens show current safe state. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Real Meta payload, profile lookup, reply, access level, token lifecycle, and messaging window cannot be proven hermetically. | BR-001, FR-004–FR-007, FR-027, NFR-006 | No Meta app or owned professional account is configured. | Keep adapters synthetic-fixture-driven and integration disabled; MAN-001 is a hard production gate and may contribute redacted schema observations from which non-user synthetic fixtures are created. |
| GAP-002 | Current Instagram export JSON has no stable public schema and no real project fixture exists yet. | FR-013 | Synthetic known-shape tests prove privacy/robustness but not current Meta output. | Manual text ships as fallback; require MAN-002 and convert approved redacted shape observations into wholly synthetic committed fixtures before enabling that parser version; never commit the user-derived archive. |
| GAP-003 | Physical push and OS process lifecycle cannot be fully reproduced in unit/widget tests. | FR-020–FR-023 | Firebase/APNs and terminated launches depend on provider/OS state. | Keep provider-neutral simulations automated and require MAN-004 before release. |
| GAP-004 | Platform secure storage/network proxy and clipboard/file-picker guarantees are outside Dart/Go tests. | NFR-003, NFR-007 | Mocks validate app behavior, not OS internals. | Run MAN-003, platform configuration review, and privacy proxy inspection before release. |
| GAP-005 | Repository-wide member data-export/account-deletion routes do not yet exist. | FR-028 | This slice can test scoped export/purge, reversible membership inactivation, and terminal identity purge, but cannot wire a nonexistent general endpoint/UI. | Treat `IT-010`/`IT-011` as the reusable contract and require composition when the general lifecycle feature lands. |
| GAP-006 | The Postgres-backed shared limiter has not yet been exercised in the eventual live multi-replica deployment. | NFR-002 | Correct code and compose tests cannot prove the final edge/proxy/replica topology. | Keep unsafe multi-replica configuration fail-closed and validate the trusted-proxy plus shared-bucket behavior in MAN-001 before enablement. |
| GAP-007 | Full dispute evidence policy remains manual and exceptional. | FR-029 | Email/support evidence cannot be truthfully automated. | Automated tests prove no transfer and explicit audited commands; human evidence review remains operator procedure. |

## 10. Out Of Scope

- Scraping, Instagram follower/following API reads, OAuth-only verification, ManyChat, export-possession proof, or server-side archive parsing.
- ZIP archive parsing in the initial client implementation.
- Collection or persistence of accounts that follow the importing member, follower-derived suggestions, or automatic follows.
- PDS storage of private Instagram data or a lexicon change.
- Marketing/future-match Instagram DMs.
- A repository-wide member data-export/account-deletion feature; only scoped composable primitives are tested here.
- Automatic conflict adjudication or silent identity transfer.
- Production enablement before every manual release gate is complete.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-11-instagram-dm-verification/`
- Risk level: **High**; document review is required. The user has already explicitly approved proceeding through document review, coding plan, and feasible implementation.
- Recommended first failing test for implementation: `UT-001` in `appview/internal/instagram/challenge_test.go` for entropy, display grammar, keyed digest storage, normalization, and secret-safe diagnostics.
- Suggested test order for implementation:
  1. `UT-001`: canonical challenge grammar, entropy, digesting, and diagnostics.
  2. `UT-008`, `UT-016`, `IT-013`: fail-closed configuration, trusted-proxy/shared limiter, dependency wiring, and readiness before any route depends on them.
  3. `IT-001`, `UT-002`: schema/check constraints plus exact state machines.
  4. `IT-020`: shared current-member guard and reversible membership-transition seam before authenticated operations are added.
  5. `IT-021`: establish the shared synthetic Go/Flutter wire corpus, initially failing until each route/model lands.
  6. `IT-002`, `UT-003`, `UT-004`, `IT-003`: attempt routes, callback verification, raw signature, fixed ingress limits, minimal work, and durable dedup/ack.
  7. `UT-007`, `IT-004`: leased processing, bounded Meta client/retries, membership recheck, and optional replies.
  8. `UT-006`, `IT-005`, `IT-006`: single eligibility policy, confirmation, one-to-one links, exact conflict/link states, and lifecycle.
  9. `UT-005`, `IT-007`, `IT-008`: strict additive imports, list/detail/renew/delete, normalization, eligibility, limits, and paginated suggestions.
  10. `IT-009`: last-moment eligibility revalidation, deterministic follow service, and explicit suggestion actions.
  11. `IT-010`: exact retention, multi-import support, scoped export/delete, membership inactivation, and terminal purge.
  12. `UT-013`, `UT-014`, `IT-011`, `IT-012`: checked actorless union, fixed five-minute coalescing/newness, preference, payload, and retraction.
  13. `IT-018`, `IT-019`: bounded operator and cancellation hardening.
  14. `UT-009`, `UT-010`, `IT-014`: bounded local parser plus minimal Flutter API and shared-wire boundary.
  15. `UT-011`, `IT-015`, `IT-016`: account-scoped providers, typed route, verification/import/suggestion UI.
  16. `UT-012`, `IT-017`: Flutter actorless notification settings/rendering/open flow.
  17. `UT-015`, `REG-001`–`REG-011`: controlled privacy scans and broad regressions.
- Commands discovered:
  - From `appview/`: focused `go test ./internal/instagram ./internal/integrations ./internal/api ./internal/routes ./internal/notifications ./internal/push ./internal/app`
  - From repository root: `just test`
  - From repository root: `just fmt`
  - From `app/`: `dart run build_runner build --delete-conflicting-outputs`
  - From `app/`: `flutter gen-l10n`
  - From `app/`: `flutter test test/instagram_migration test/notifications test/router test/settings`
  - From `app/`: `flutter analyze`
  - From `app/`: `flutter test`
- Blocking gaps: None for implementation. `GAP-001`–`GAP-004` and `GAP-006` block relevant production enablement/release confidence; `GAP-005` blocks only future repository-wide lifecycle composition.
