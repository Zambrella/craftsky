# Acceptance Test Specification: Instagram DM Ownership Verification And Automatic Following

> **2026-08-14 AppView audit amendment:** Section 12 supersedes every test in
> this document that expects an import, matcher, or background worker to write
> a PDS follow. The approved behavior is private suggestions plus an explicit
> current-member Follow action.

## 1. Test Strategy

This high-risk change uses layered, vertical TDD. Pure tests establish the canonical challenge grammar, exact state machines, signature and minimal-work-item rules, normalization, eligibility, retry, fixed limits, account fencing, and manual-unfollow suppression. Database and HTTP integration tests prove current-membership authorization, ownership, uniqueness, durability, concurrency, additive import support, background owner-session selection, deterministic automatic PDS writes, actorful notification completion, retention, and exact wire contracts. Flutter and Go consume the same synthetic golden JSON fixtures so route/state drift is detected in both directions. Flutter tests also prove that raw exports stop at the local parser boundary and that verification/import/notification actions remain bound to the initiating account. A small set of Gherkin acceptance scenarios ties those layers into member journeys.

All behavior that does not require a real Meta app is automated. Wholly synthetic canaries and explicitly approved redacted fixtures are permitted only as controlled test inputs; tests assert that they appear solely in the intentional private persistence/API/UI locations and never in diagnostics, push, PDS records, raw-request reserialization, or committed snapshots. Real or user-derived secrets and identity data are never fixtures. Manual checks are limited to the external capability spike, real current redacted export fixtures, physical-device push lifecycle, and final visual/accessibility inspection. Production enablement remains blocked until those checks pass.

The first implementation loop is `UT-001`: generate and digest a challenge with at least 60 bits of formatted entropy, then prove that no plaintext or member data enters the stored form. The server data model and routes must not be built on an untested token grammar.

For the completed 2026-07-23 ZIP extension, the first failing loop was `UT-017`:
parse the wholly synthetic equivalent of the observed `_u/<username>` URL plus
agreeing-title shape. `UT-018` then proves file-backed ZIP target selection,
integrity, and limit behavior before the picker/UI is changed. The approved
user-derived archive is manual evidence only and is never copied into the
repository.

For the 2026-07-27 automatic-follow revision, the first new failing loop is
`IT-024`: prove that a background operation can select only the most recently
active usable OAuth session owned by the importing DID and cannot write with
another DID's session. `IT-009` then replaces the internal suggestion-action
path with an automatic worker operation, and `IT-025` locks in manual-unfollow
suppression before reconciliation is changed.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001–AC-008, AC-014, AC-015, AC-048 | AT-001, AT-002, UT-001–UT-004, IT-001–IT-005, IT-020 | Acceptance / Unit / Integration | Yes, except live Meta delivery in MAN-001 |
| BR-002 | AC-016–AC-019, AC-039, AC-054 | AT-003, AT-009, UT-005, UT-009, UT-010, UT-015, UT-017, UT-018, IT-007, IT-014, IT-023, REG-013 | Acceptance / Unit / Integration / Regression | Yes |
| BR-003 | AC-020–AC-025, AC-029, AC-055 | AT-004, AT-005, UT-006, IT-008, IT-009, IT-011, IT-024 | Acceptance / Unit / Integration | Yes |
| BR-004 | AC-009, AC-024–AC-031, AC-034, AC-056 | AT-004–AT-007, IT-006, IT-009–IT-012, IT-017, IT-025 | Acceptance / Integration | Yes |
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
| FR-012 | AC-016–AC-019, AC-026 | AT-003, UT-005, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-016–AC-018, AC-050–AC-052 | AT-003, AT-009, UT-009, UT-017, UT-018, IT-014, IT-023 | Acceptance / Unit / Integration | Yes; real current archive in MAN-005 |
| FR-014 | AC-019, AC-020, AC-051 | UT-005, UT-009, UT-017 | Unit | Yes |
| FR-015 | AC-020–AC-022, AC-025, AC-031, AC-048 | UT-006, IT-008, IT-009, IT-020 | Unit / Integration | Yes |
| FR-016 | AC-021, AC-023, AC-038 | AT-004, IT-008, IT-014, IT-016, REG-014 | Acceptance / Integration / Regression | Yes |
| FR-017 | AC-024, AC-025, AC-029, AC-048, AC-055, AC-056 | AT-004, AT-005, IT-009, IT-011, IT-020, IT-024, IT-025 | Acceptance / Integration | Yes |
| FR-018 | AC-026, AC-027, AC-028, AC-031, AC-048 | AT-005, AT-008, IT-007, IT-010, IT-011, IT-020 | Acceptance / Integration | Yes |
| FR-019 | AC-029, AC-030, AC-056 | AT-005, IT-011, IT-025 | Acceptance / Integration | Yes |
| FR-020 | AC-029, AC-034–AC-037 | AT-005, AT-007, UT-012, UT-014, IT-011, IT-012, IT-017 | Acceptance / Unit / Integration | Yes |
| FR-021 | AC-034 | UT-013, IT-012, IT-017 | Unit / Integration | Yes |
| FR-022 | AC-035–AC-037 | AT-007, UT-012, UT-014, IT-012, IT-017 | Acceptance / Unit / Integration | Yes; physical provider in MAN-004 |
| FR-023 | AC-038, AC-042 | AT-007, IT-016, IT-017 | Acceptance / Integration | Yes |
| FR-024 | AC-003, AC-005, AC-009, AC-014, AC-038, AC-049 | AT-002, AT-006, IT-015, IT-016, IT-022 | Acceptance / Integration | Yes |
| FR-025 | AC-018, AC-023, AC-026, AC-034, AC-038, AC-048, AC-050 | AT-003, AT-004, AT-008, AT-009, IT-014–IT-016, IT-023 | Acceptance / Integration | Yes |
| FR-026 | AC-024, AC-025, AC-035, AC-037, AC-042 | AT-004, AT-007, IT-009, IT-015–IT-017 | Acceptance / Integration | Yes |
| FR-027 | AC-043 | UT-007, IT-004, MAN-001 | Unit / Integration / Manual | Yes except real messaging window |
| FR-028 | AC-028, AC-031, AC-044, AC-048 | AT-006, IT-010, IT-011, IT-020 | Acceptance / Integration | Yes |
| FR-029 | AC-032, AC-045 | IT-018 | Integration | Yes |
| FR-030 | AC-048 | AT-008, IT-002, IT-005, IT-007–IT-012, IT-020, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-031 | AC-050, AC-052, AC-053 | AT-009, UT-018, IT-023, MAN-005 | Acceptance / Unit / Integration / Manual | Yes plus physical-device validation |
| FR-032 | AC-055 | IT-024, REG-009 | Integration / Regression | Yes |
| NFR-001 | AC-002, AC-012 | UT-001, UT-015 | Unit | Yes |
| NFR-002 | AC-041, AC-046 | UT-016, IT-003, IT-013 | Unit / Integration | Yes; deployment check in MAN-001 |
| NFR-003 | AC-039 | UT-015, REG-007 | Unit / Regression | Yes |
| NFR-004 | AC-004, AC-005, AC-040, AC-048 | IT-002, IT-007, IT-013, IT-020, REG-001 | Integration / Regression | Yes |
| NFR-005 | AC-008, AC-010, AC-015, AC-025, AC-029, AC-055 | UT-002, IT-003–IT-005, IT-009, IT-011, IT-024 | Unit / Integration | Yes |
| NFR-006 | AC-011, AC-040 | UT-007, UT-008, IT-004, IT-013 | Unit / Integration | Yes |
| NFR-007 | AC-038 | IT-016, IT-017, MAN-003 | Integration / Manual | Yes plus final manual inspection |
| NFR-008 | AC-042, AC-049, AC-053 | UT-011, IT-015, IT-017, IT-022, IT-023, REG-012 | Unit / Integration | Yes |
| NFR-009 | AC-047 | IT-019, REG-008 | Integration / Regression | Yes |
| NFR-010 | AC-052, AC-053 | UT-018, IT-023, MAN-005 | Unit / Integration / Manual | Yes plus physical-device validation |
| RULE-001 | AC-002, AC-003, AC-010, AC-012 | UT-001, UT-002, IT-002, IT-004 | Unit / Integration | Yes |
| RULE-002 | AC-014, AC-015 | AT-002, UT-002, IT-005 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-015, AC-032, AC-033 | AT-006, UT-006, IT-005, IT-006 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-009, AC-014, AC-020, AC-031 | AT-002, AT-006, UT-006, IT-006, IT-008 | Acceptance / Unit / Integration | Yes |
| RULE-005 | AC-019–AC-022 | UT-005, UT-006, UT-009, IT-008 | Unit / Integration | Yes |
| RULE-006 | AC-016, AC-018, AC-022, AC-029 | AT-003, UT-006, UT-009, IT-008, IT-011 | Acceptance / Unit / Integration | Yes |
| RULE-007 | AC-016, AC-017, AC-039, AC-054 | AT-003, AT-009, UT-010, UT-015, UT-018, IT-007, IT-014, IT-023 | Acceptance / Unit / Integration | Yes |
| RULE-008 | AC-024, AC-025, AC-029 | AT-004, AT-005, IT-009, IT-011, REG-004 | Acceptance / Integration / Regression | Yes |
| RULE-009 | AC-026–AC-030 | AT-005, IT-010, IT-011 | Acceptance / Integration | Yes |
| RULE-010 | AC-031, AC-044 | AT-006, IT-010, REG-004 | Acceptance / Integration / Regression | Yes |
| RULE-011 | AC-050, AC-054 | AT-009, UT-010, IT-023, REG-013 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-012 | AC-025, AC-056 | AT-004, IT-025, REG-014 | Acceptance / Integration / Regression | Yes |

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
    And the member already has private verification, import, and queued automatic-follow state
    When the signed-in member opens Find people from Instagram and requests verification
    Then the page explains that verification is unavailable
    And AppView returns the standard unavailable error without exposing configuration
    And verified-account import create/list/delete, queued automatic-follow processing, existing verification status/disable/revoke, and notification preferences remain available when their own dependencies are healthy
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
    And exactly one active verified account mapping is created for Alice and IGSID 100
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
  Scenario: Select a supported standalone JSON following export
    Given a bounded JSON fixture contains accounts-followed, follower, media, profile, and message fields
    When Alice selects and imports the file
    Then parsing occurs on-device
    And only accounts-followed usernames are normalized and uploaded
    And follower data is discarded locally
    And the import copy explains that exact current and future matches are automatically followed publicly
    And no normalized-preview or retention-consent step is shown
    And the mocked AppView request contains only source type and normalized usernames
    And no raw JSON, filename, message, media, or unrelated value crosses the repository boundary
```

### AT-004: Automatically Follow A Current Eligible Match

Requirement IDs: `BR-003`, `FR-015`–`FR-017`, `FR-020`, `FR-026`, `RULE-005`, `RULE-008`
Acceptance Criteria: `AC-020`–`AC-025`, `AC-029`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/instagram/automatic_follow_flow_test.go`, `app/test/instagram_migration/instagram_migration_page_test.dart`

```gherkin
Feature: Instagram automatic following
  Scenario: Follow one exact current match without a review queue
    Given Alice imported a following handle that exactly matches Bob's active discoverable DM-verified account
    And Alice and Bob are current visible CraftSky members
    And neither member blocks the other, Alice has not muted Bob, and Alice does not already follow Bob
    When AppView processes the import-created automatic-follow operation
    Then the same InstagramSuggestionEligibilityPolicy is evaluated immediately before the PDS write
    And the shared follow service creates exactly one deterministic app.bsky.graph.follow
    And Alice receives exactly one actorful instagramMatch notification identifying Bob
    And Bob receives only the ordinary effects of Alice following him and no Instagram/import evidence
    And no People You May Know list, suggestion endpoint, accept action, or dismiss action exists
    And a failed PDS write creates no automatic-follow notification
    And a follow that already existed before the operation creates neither a duplicate write nor a false automatic-follow notification
```

### AT-005: Retain Handles And Automatically Follow Future Matches

Requirement IDs: `BR-003`, `BR-004`, `FR-017`–`FR-021`, `RULE-006`, `RULE-008`, `RULE-009`, `RULE-012`
Acceptance Criteria: `AC-026`–`AC-030`, `AC-034`, `AC-056`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/instagram/future_match_test.go`, `app/test/instagram_migration/instagram_future_match_test.dart`

```gherkin
Feature: Future Instagram matches
  Scenario: Retained handles become discoverable and manual unfollow wins
    Given Alice verified her Instagram account and imported following handles
    And two additive active imports support one prospective Alice-to-Bob operation
    When Bob later verifies his matching handle and enables discovery
    Then AppView writes one deterministic Alice-to-Bob follow
    And Alice receives one separate actorful instagramMatch notification for Bob
    And deleting one import keeps unwritten work while the other active import still supports it
    And its push preference is independently disableable
    And disabling push does not suppress the private in-app notification
    And Bob cannot learn that Alice imported or searched for him
    When Alice manually unfollows Bob
    And reconciliation, duplicate imports, and eligibility-restoration triggers run
    Then Bob is not automatically followed again during that verification lifetime
    When Alice revokes Instagram verification
    Then all of Alice's imports and retained handles are deleted
    And dependent unwritten operations are invalidated
    And successful PDS follows and historical notifications remain unchanged
    And later verification plus a fresh import is new authorization
```

### AT-006: Revoke Or Conflict Without Transferring Ownership

Requirement IDs: `BR-004`, `FR-009`–`FR-011`, `FR-024`, `FR-028`, `RULE-003`, `RULE-004`, `RULE-010`
Acceptance Criteria: `AC-009`, `AC-031`–`AC-033`, `AC-044`
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/instagram/link_lifecycle_test.go`, `app/test/instagram_migration/instagram_link_settings_test.dart`

```gherkin
Feature: Instagram verification control and conflicts
  Scenario: A second member claims an owned Instagram identity
    Given IGSID 100 is actively verified by Alice
    When Bob confirms an attempt from IGSID 100
    Then Alice remains the owner
    And both verification surfaces expose only a generic private conflict warning
    And dependent unwritten operations are never reassigned
    When Alice chooses Revoke Instagram verification at the bottom of the page
    Then Flutter asks for destructive confirmation
    And explains that imported handles and automatic-follow authority are deleted while existing CraftSky follows remain
    When Alice cancels the dialog
    Then Instagram verification remains active
    When Alice chooses to revoke again and confirms
    Then dependent unwritten operations and invalid unsent pushes are cancelled
    And revocation is submitted exactly once
    And previously successful PDS follows and notification history remain unchanged
```

### AT-007: Open An Actorful Automatic-Follow Notification Under The Correct Account

Requirement IDs: `FR-020`–`FR-023`, `FR-026`, `NFR-007`, `NFR-008`
Acceptance Criteria: `AC-034`–`AC-038`, `AC-042`
Priority: Must
Level: Acceptance
Automation Target: `app/test/instagram_migration/instagram_notification_open_flow_test.dart`

```gherkin
Feature: Instagram match notifications
  Scenario: Open and toggle a match notification for an inactive retained account
    Given account A is active and an actorful instagramMatch notification for Bob belongs to retained account B
    When the notification is opened
    Then the app activates B before navigation
    And opens Bob's CraftSky profile
    And the row shows Bob's current Following state and ordinary follow/unfollow control
    And the notification has Bob as actor but has no URI, CID, rkey, system payload, client destination, IGSID, challenge, import, or match evidence
    And the provider push contains no Bob identity and opens B's Notifications feed
    When B unfollows Bob from the notification row
    Then the control changes to Follow without deleting the historical notification
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
    When Alice calls any authenticated Instagram operation or a worker tries to match, auto-follow, or notify for Alice
    Then each client operation returns 404 profile_not_found through the standard error envelope
    And workers perform membership inactivation rather than exposing state or surfacing a foreign-key/internal error
    And Alice's verified mapping becomes membershipInactive, discovery is off, owner imports are paused, and dependent unwritten operations are invalidated
    And the private owner data is retained under its normal retention policy
    When Alice rejoins CraftSky
    Then discovery and imports remain inactive until Alice explicitly reactivates the verified mapping and each paused import
```

### AT-009: Import A Large Instagram ZIP Without Reading Unrelated Data

Requirement IDs: `BR-002`, `FR-013`, `FR-014`, `FR-025`, `FR-031`,
`NFR-008`, `NFR-010`, `RULE-006`, `RULE-007`, `RULE-011`
Acceptance Criteria: `AC-016`–`AC-019`, `AC-038`, `AC-042`, `AC-050`–`AC-054`
Priority: Must
Level: Acceptance
Automation Target:
`app/test/instagram_migration/instagram_import_privacy_test.dart`,
`app/test/instagram_migration/instagram_migration_page_test.dart`

```gherkin
Feature: Private Instagram ZIP import
  Scenario: Select an all-information ZIP containing one following export
    Given a verified mobile member selects a large file-backed ZIP
    And the ZIP contains media, messages, follower data, and exactly one
      connections/followers_and_following/following.json
    And that JSON uses exact Instagram _u username URLs with agreeing titles
    When CraftSky processes the export
    Then ZIP inspection and JSON parsing run in a background isolate
    And only the canonical following entry is decompressed within fixed limits
    And no unrelated entry is read or extracted to disk
    And the UI remains responsive and labels the import Instagram export
    And Instagram export is the selector's default option
    And the page explains that exact current and future matches are automatically followed
    And the request uses sourceType instagramJson plus normalized usernames only
    And raw ZIP, filename, paths, URLs, follower data, messages, and media never
      cross the repository boundary
    And switching CraftSky accounts before completion discards the late result
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-002, NFR-001, RULE-001 | AC-002, AC-003, AC-012 | Generate, format, normalize, digest, and compare the canonical challenge. | Deterministic secure bytes; 10,000 generated values; 30-symbol `23456789ABCDEFGHJKMNPQRSTVWXYZ` alphabet; lower/upper case and outer-whitespace variants; embedded whitespace, extra prose, ambiguous characters, wrong keyed digest, synthetic DID/email/token canaries. | Exactly 13 random symbols render as `CSKY-XXXX-XXXX-XXXX-X` for about 63.8 bits; only ASCII case and outer whitespace normalize; the complete message must be the token; storage/diagnostics contain only keyed digest metadata. | `appview/internal/instagram/challenge_test.go` |
| UT-002 | FR-003, FR-008, FR-009, NFR-005, RULE-001, RULE-002 | AC-004, AC-005, AC-010, AC-014, AC-015 | Enforce the exact verification-attempt and link transition tables. | Public attempts `pendingDm`, `processing`, `pendingConfirmation`, `confirmed`, `expired`, `cancelled`, `superseded`, `rejected`, `conflicted`; links `active`, `membershipInactive`, `revoked`, `superseded`, `disputed`; every allowed/forbidden event, expiry boundary, retry code, concurrent conflict, and wrong DID. | Only §12.1 transitions succeed; terminal attempt states stay terminal and replay idempotently; `processing` is serialized; retry exhaustion has the documented safe rejected/retry code; wrong owner sees not-found; confirmation cannot precede candidate. | `appview/internal/instagram/state_test.go` |
| UT-003 | FR-004, FR-005 | AC-006–AC-008 | Verify callback query and raw-body HMAC semantics. | Valid/invalid mode/token/challenge; exact/mutated/empty/oversized bodies; missing/malformed/case variants of signature header. | Only valid verification echoes challenge; constant-time HMAC over exact bytes gates decoding; invalid/oversized inputs fail generically. | `appview/internal/instagram/webhook_test.go` |
| UT-004 | FR-005–FR-007 | AC-008, AC-010, AC-012, AC-013, AC-039 | Decode supported Meta message events and immediately reduce them to the exact minimal durable work item. | Official-account incoming text; echo/self/deleted/non-text/unsupported/wrong object/wrong account; one, 100, and 101 events; unknown fields; synthetic raw-body/message/challenge canaries. | One through 100 supported unique messages yield only keyed message-ID digest, sender IGSID, configured official-account ID, keyed normalized-challenge digest, event timestamp, and job/lease/retry fields; 101 returns a limit error with no partial work; raw ID/body/text/plain challenge and unrelated fields are absent from persistence and diagnostics. | `appview/internal/instagram/meta_payload_test.go` |
| UT-005 | FR-012, FR-014, RULE-005, RULE-006 | AC-018–AC-020 | Normalize and validate server import entries and the following-only wire boundary. | Whitespace, one `@`, case, duplicates, dots/underscores/digits, Unicode, empty/overlong/invalid values, display names, and request entries containing `direction` or follower-specific fields. | Deterministic normalized usernames; invalid values reject; strict request decoding rejects direction/follower fields; no fuzzy/display-name inference. | `appview/internal/instagram/username_test.go`, `appview/internal/api/instagram_imports_test.go` |
| UT-006 | FR-009, FR-011, FR-015, FR-017, RULE-003–RULE-006 | AC-020–AC-022, AC-025, AC-031–AC-033, AC-048 | Evaluate verified mappings, username changes, and the single `InstagramSuggestionEligibilityPolicy`. | Importer/target current or departed; active/disabled/revoked/superseded/disputed/membershipInactive mapping; verified/unverified; discoverable/hidden/taken-down; exact/stale handle; self/already-followed; block each direction; importer mute; IGSID/username collision; unavailable block/mute provider in production and explicit fake-test mode. | Only exact, current, conflict-free, DM-verified, discoverable imported accounts-followed evidence passes; every safety exclusion fails at operation creation and final write/notification validation; missing safety data fails closed outside explicit tests; collisions dispute and old-handle unwritten operations invalidate without transfer. | `appview/internal/instagram/eligibility_test.go` |
| UT-007 | FR-006, FR-027, NFR-006 | AC-011, AC-043 | Classify Meta provider results and compute the fixed provider/worker retry and reply behavior. | 2xx, 4xx auth/permission/not-found/rate limit, 5xx, timeout/cancel, 64 KiB and 64 KiB+1 bodies, 5-second deadline, allowed/expired interaction window, attempts one through six, 60-second lease, 15-minute processing age. | Retry only transient cases for at most five attempts with capped backoff; Meta calls stop at 5 seconds and 64 KiB; a job cannot process past 15 minutes; replies are optional/idempotent/window-bound; cancellation is not a provider failure. | `appview/internal/instagram/meta_client_test.go`, `appview/internal/instagram/retry_test.go` |
| UT-008 | FR-001, NFR-002, NFR-006 | AC-001, AC-040, AC-046 | Validate disabled/complete/partial configuration, trusted-proxy policy, fixed hard maxima, and API URL construction. | Empty bundle; each missing/invalid secret, account ID, API version/base URL, HTTPS DM URL, timeout/worker/replica/shared-limit setting; untrusted and configured proxy chains; values at/above maxima. | Empty disables safely; partial/unsafe or limit-loosening config fails; complete produces redacted bounded options; arbitrary forwarding headers never select the limiter IP; unsafe multi-replica mode fails closed. | `appview/internal/app/instagram_config_test.go` |
| UT-009 | FR-013, FR-014, RULE-005–RULE-007 | AC-016–AC-019, AC-022 | Parse manual text and versioned accounts-followed JSON locally under the fixed file/entry caps. | Known following shapes; mixed follower/unrelated fields; changed nesting; malformed/Unicode/duplicates/follower-only/manual text; 20 MiB and 20 MiB+1 files; 10,000 and 10,001 deduplicated entries. | Returns only normalized accounts-followed usernames and bounded warnings; follower data and the raw model cannot cross parser output; follower-only/oversized/unsupported input fails before repository call. | `app/test/instagram_migration/services/instagram_import_parser_test.dart` |
| UT-010 | BR-002, RULE-006, RULE-007 | AC-016–AC-018, AC-039 | Prove the Flutter import request type cannot carry raw archive values, retention settings, directions, or follower data. | Parser result plus wholly synthetic canary raw bytes, filename, JSON keys, follower username, media URL, message, and profile value. | Repository request serializes only `sourceType` and normalized username `entries`; controlled canaries may exist in parser input but never in the serialized request, diagnostics, snapshots, or server persistence. | `app/test/instagram_migration/models/instagram_import_request_test.dart`, `app/test/observability/secret_scan_test.dart` |
| UT-011 | NFR-008 | AC-042 | Fence Instagram async work by account lease/generation. | Poll/import/confirm/notification follow-toggle/navigation started under A; switch to B; late success/error/rollback. | Late A work has no B-visible state, mutation, cache update, or navigation; current-account work proceeds. | `app/test/instagram_migration/providers/instagram_migration_provider_test.dart`, `app/test/notifications/instagram_match_notification_test.dart` |
| UT-012 | FR-020, FR-022, FR-026 | AC-035–AC-037 | Decode type-discriminated actorful `instagramMatch` notifications and infer destinations in Flutter. | Existing social types; `type: instagramMatch` with hydrated target actor/viewer state and no `kind` or server destination; forbidden `system`/URI/CID/rkey facts; malformed/unknown types; push category with no actor identity. | Known match requires the target actor and forbids system/social-source/route fields, renders automatic-follow copy and current Follow/Following action, opens the actor profile from the row, and opens the Notifications feed from identity-free push category; existing social types retain required facts; unknown types remain inert. | `app/test/notifications/models/notification_test.dart`, `app/test/notifications/instagram_match_notification_test.dart`, `app/test/notifications/services/notification_destination_inference_test.dart` |
| UT-013 | FR-021 | AC-034 | Enforce fixed notification scope and push-only UI model. | GET preference; patch push; patch scope; future unknown fields. | Wire scope remains `everyone`; push changes; scope mutation rejects; Flutter creates no actor-scope control and preserves unknown fields. | `appview/internal/notifications/preferences_test.go`, `app/test/notifications/models/notification_preferences_test.dart` |
| UT-014 | FR-020, FR-022 | AC-035, AC-036 | Build the actorful feed variant, identity-free push payload, and automatic-follow copy. | Hydrated target actor/viewer state; forbidden system/count/group/URI/CID/rkey/destination fields; synthetic Instagram/import/identity canaries; account-subscription binding. | Feed row contains only the target's ordinary CraftSky profile plus notification metadata; provider data contains only category, stable notification ID, and opaque account binding; no target identity, Instagram evidence, count/group fact, server destination, synthetic AT source, or private canary leaks. | `appview/internal/push/payload_test.go`, `appview/internal/api/notification_store_test.go` |
| UT-015 | NFR-001, NFR-003, RULE-007 | AC-002, AC-039 | Scan server/client diagnostic and unintended-output surfaces using controlled data. | Wholly synthetic canaries, plus separately approved redacted fixtures, for challenge/digest, body/message, username, IGSID, handle list, Meta token/secret/signature, raw export/upstream response; no real or user-derived values. | Canaries occur only in the explicitly intended private test input/DB/API/UI field under test; none appears in logs, errors, spans, Sentry, metrics, push, PDS records, URLs, raw-request reserialization, `String()`/`toString`, or committed snapshots. | `appview/internal/observability/instagram_redaction_test.go`, `app/test/observability/secret_scan_test.dart` |
| UT-016 | NFR-002 | AC-041, AC-046 | Apply the exact shared abuse keys, windows, defaults, hard maxima, and response policy atomically. | Challenge DID 5/15m, device 10/15m, IP 30/15m; invalid-redemption IGSID 10/15m and IP 30/15m; confirmation DID 20/hour and device 30/hour; imports DID 10/hour and device 20/hour; webhook global 1,000/minute and IP 300/minute; lookup concurrency 20 and IGSID 5/hour; values immediately before/at/one after each boundary, concurrent requests, expiry, untrusted forwarded IP. | Client limits are generic 429; pre-auth webhook IP and post-signature global excess are generic 429 with `Retry-After: 60` and no partial persistence; per-IGSID invalid excess is terminally deduped/ignored with 200 and no lookup; lookup pressure defers durable work. Each key/window is atomic/shared, never trusts arbitrary forwarding headers, and can tighten but not exceed the maximum. | `appview/internal/instagram/limiter_test.go` |
| UT-017 | FR-013, FR-014, RULE-005, RULE-006 | AC-016–AC-020, AC-051 | Parse the observed current following shape without accepting titles as standalone identity evidence. | Wholly synthetic records with exact HTTPS `www.instagram.com/_u/<username>` URLs and agreeing titles; case variants; legacy direct `value`; title-only; mismatch; HTTP; userinfo/port; lookalike host; other path; trailing slash; query/fragment; percent-encoded separator; invalid/overlong username; zero/two string-list candidates. | Valid legacy values and exact agreeing URL/title pairs normalize/deduplicate. Every ambiguous, mismatched, title-only, fuzzy, or noncanonical URL record is ignored/rejected with bounded counts and no retained URL/title. | `app/test/instagram_migration/services/instagram_import_parser_test.dart` |
| UT-018 | FR-013, FR-031, NFR-010, RULE-007 | AC-016–AC-018, AC-052, AC-053 | Inspect and decode a ZIP from a file path without loading or extracting unrelated archive payloads. | Wholly synthetic temporary ZIPs: valid stored/deflated target plus large unrelated sentinel; missing/duplicate target; nested/lookalike path; encrypted flag; unsupported compression; malformed/truncated directory; CRC corruption; declared/actual target at 20 MiB and +1; directory entries/bytes immediately below/at/above 100,000/64 MiB; ZIP64 metadata; 10,000/10,001 usernames. | File-backed decode selects exactly one canonical target, verifies integrity, enforces every bound before/through output, returns normalized result only, never creates extracted files, and closes resources on success/failure. Complete ZIP bytes and unrelated sentinels never enter parser results or diagnostics. | `app/test/instagram_migration/services/instagram_export_file_parser_test.dart` |
| UT-019 | FR-032 | AC-055 | Select a background OAuth session without crossing owner boundaries. | No sessions; revoked/expired/inactive sessions; two usable sessions for Alice with equal/different activity timestamps and stable IDs; newer usable Bob session; provider-invalid selected session. | Selection filters by exact importer DID and usable state, uses deterministic most-recent activity plus stable tie-break, never returns Bob's session, returns retryable absence when none is usable, and invalidates only the exact provider-rejected Alice session before reselection. | `appview/internal/auth/background_session_selector_test.go` |
| UT-020 | FR-017, FR-019, RULE-012 | AC-025, AC-056 | Enforce automatic-follow operation terminal and suppression transitions. | `pending`, `writing`, `followed`, `alreadyFollowing`, `invalidated`; duplicate/import/eligibility triggers before and after follow; manual unfollow; revocation/new verification generation. | Retryable failures return to pending; only successful deterministic write becomes followed; already-following emits no automatic notification; followed/alreadyFollowing suppress same-generation recreation after unfollow; invalidated work may be reconsidered only when current support becomes eligible; revocation deletes the old generation so fresh evidence may authorize new work. | `appview/internal/instagram/automatic_follow_state_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-009, FR-028, NFR-005 | AC-015, AC-028, AC-031, AC-044 | Apply/revert the private Instagram migration and enforce checked invariants. | Fresh `testdb.WithSchema`; active and departed member DIDs; every attempt/mapping/import/automatic-follow/conflict/actorful-notification/work-item state. | Exercise check/partial-unique violations, cross-user invalidation, membership departure, explicit scoped purge, terminal identity purge, actorful notification shapes, and down migration. | Tables/checks/indexes/FKs enforce one-to-one ownership and operation/notification idempotency; `instagramMatch` requires actor and forbids system/source fields; profile membership loss does not broadly cascade private owner data; explicit terminal/scoped purge leaves no orphaned dependent rows. | `appview/internal/db/instagram_migration_test.go` |
| IT-002 | FR-002, FR-003, FR-030, NFR-004, RULE-001 | AC-002–AC-005, AC-040, AC-048, AC-049 | Enforce authenticated attempt create/read/cancel/current ownership and exact wire contracts. | Route mux with fake clock/random/config; Alice/Bob/current/departed sessions and device IDs; shared request/response golden fixtures; every public attempt state. | Create twice; read current and by ID under both DIDs; confirm/cancel under both DIDs; cancel absent and post-tombstone IDs; cross expiry; serialize all states; send invalid/unknown/oversized bodies. | Documented success codes/camelCase match golden JSON; current returns only the caller's non-terminal attempt or `{verification: null}` after expiry; `processing` and every terminal state round-trip; reads/confirms across owners are not-found; every owned/foreign/absent/purged DELETE is the same 204 and mutates only owned state; current-member absence is 404 `profile_not_found`. | `appview/internal/api/instagram_verifications_test.go`, `appview/internal/routes/instagram_routes_test.go` |
| IT-003 | FR-004–FR-006, NFR-002, NFR-005 | AC-006–AC-008, AC-039, AC-041 | Enforce callback verification, raw signature, fixed ingress limits, exact minimal inbox, durable deduplication, and quick acknowledgement. | Real mux/database; controlled synthetic signed fixtures; blocked worker; 256 KiB boundary; 100/101 events; pre-auth IP/post-signature global/per-IGSID/worker-pressure boundaries. | GET verify; POST valid/duplicate/multi/forged/oversized/rate-limited payloads; inspect response/Retry-After, committed columns, lookup calls, and diagnostics. | Valid unique messages commit only §12.2 fields before 200; duplicates are 200/no-op; IP/global excess is generic 429/no partial work; per-IGSID invalid excess is terminally deduped with cleared sensitive fields, 200, and no lookup; worker pressure defers with 200; 256 KiB/100 pass while larger reject; no raw ID/body/text/challenge is stored/emitted. | `appview/internal/integrations/instagram_webhook_test.go` |
| IT-004 | FR-006, FR-007, FR-027, FR-030, NFR-005, NFR-006 | AC-010–AC-013, AC-043, AC-048 | Lease/process/retry a minimal webhook item through candidate lookup and optional reply. | Database attempts/work items; fake Meta profile/reply client; four workers plus a fifth; fake clock; transient/terminal results; member removed before each transition. | Race workers; replay/reorder/cancel/expire; advance 60-second leases/five attempts/15-minute age; return valid/invalid usernames; remove membership. | At most four workers run; one owns a lease and one candidate binds; transient work is bounded; sensitive work fields clear on terminal state; departed-owner work invokes inactivation and no link/notification; reply never controls correctness. | `appview/internal/instagram/dispatcher_test.go`, `appview/internal/instagram/processor_test.go` |
| IT-005 | FR-008, FR-009, FR-030, NFR-005, RULE-002, RULE-003 | AC-014, AC-015, AC-032, AC-048 | Confirm links and resolve concurrent uniqueness/current-membership outcomes. | Pending candidates for same/different DID, IGSID, username; concurrent transactions; Alice/Bob routes; token refresh/new device for same DID; member removal. | Confirm/retry/race under same DID with different valid sessions, wrong DID, and departed DID. | One active ownership result; same-DID replay is idempotent across sessions/devices; wrong owner is resource not-found; departed owner is `profile_not_found`; conflict/audit rows are generic and existing ownership never transfers. | `appview/internal/api/instagram_confirmation_test.go`, `appview/internal/instagram/link_store_test.go` |
| IT-006 | FR-010, FR-011, FR-028, FR-030, RULE-003, RULE-004, RULE-010 | AC-009, AC-031–AC-033, AC-048 | Read/update/revoke/reactivate/refresh a verified mapping and invalidate dependents. | Every exact mapping state, pending/followed operations, actorful events/deliveries, conflict rows, membership depart/rejoin, fake changed username. | Toggle discovery, revoke, refresh username/collision, depart/rejoin/reactivate, and inspect owner/cross-owner status. | Wire states are exact; unwritten dependents invalidate/cancel; departure becomes `membershipInactive` without purge; rejoin never silently enables discovery; explicit reactivation is required; generic conflict only; successful follows/history and old-username non-transfer remain. | `appview/internal/api/instagram_account_test.go`, `appview/internal/instagram/link_lifecycle_test.go` |
| IT-007 | FR-001, FR-012, FR-018, FR-030, NFR-004, RULE-006, RULE-007 | AC-001, AC-016–AC-019, AC-026–AC-028, AC-040, AC-048 | Enforce the strict verified-only, following-only additive import create/list/read/reactivation/delete lifecycle and ownership contract. | Authenticated routes; Meta disabled/up; zero through 10,001 valid/invalid username entries; retention/direction/follower/raw/archive-like unknown fields; Alice/Bob verified/unverified/departed/rejoined sessions; long-lived and two-support imports. | POST/list/page/GET; reject retention/directional/follower payloads and unverified owners; PATCH only reactivation; DELETE owned/foreign/absent/purged IDs; remove one supporting import; simulate Meta outage. | Immutable snapshots and exact limits hold; create returns only import plus `followingCount` and never `initialSuggestionCount`; storage has no retention, expiry, direction, or follower-count columns; every handle remains until delete/revocation; inactive imports reactivate explicitly; ownership/membership/outage/support behavior remains unchanged. | `appview/internal/api/instagram_imports_test.go`, `appview/internal/db/instagram_migration_test.go` |
| IT-008 | FR-015, FR-016, FR-030, RULE-004–RULE-006 | AC-020–AC-023, AC-048 | Apply one eligibility policy while keeping automatic-follow operations private. | Full UT-006 policy matrix, safety-source failure, internal operation rows, registered route inventory, old suggestion route requests. | Create imports; match/persist operations; fail/restore safety sources; inspect all public responses; request list/accept/dismiss suggestion paths. | Only currently eligible exact following evidence creates internal operations; unavailable safety state fails closed; no operation/match reason is serialized; all old suggestion routes are unregistered/not found; no pagination/review surface remains. | `appview/internal/instagram/matcher_test.go`, `appview/internal/routes/instagram_routes_test.go` |
| IT-009 | FR-015, FR-017, FR-026, FR-030, NFR-005, NFR-008, RULE-008 | AC-024, AC-025, AC-029, AC-042, AC-048 | Process an automatic-follow operation through last-moment policy validation, deterministic PDS write, and idempotent notification completion. | Pending operation; fake PDS/follow service; stable operation key/rkey; delayed firehose; PDS failures; crash after PDS success; concurrent workers; policy changed immediately before put; target followed externally; Alice/Bob/departed owners. | Claim/retry/race, block/mute/hide/depart/follow target just before the external call, and restart after each durable boundary. | Final ineligible states write/notify nothing; eligible retries use one deterministic `putRecord`; crash recovery reconciles one follow and one actorful notification; externally pre-followed target becomes `alreadyFollowing` without notification; provider failures retry safely; owner/membership/account fences hold. | `appview/internal/instagram/automatic_follow_worker_test.go`, `appview/internal/followwrite/service_test.go` |
| IT-010 | FR-018, FR-028, FR-030, RULE-009, RULE-010 | AC-026–AC-028, AC-031, AC-044, AC-048 | Enforce verification-lifetime import/operation storage, additive support, export, reversible membership inactivation, and terminal/scoped purge. | Long-lived matched/unmatched following rows; multi-supported imports; verification revocation; departed/rejoined/terminal owner; successful-follow and historical-notification sentinels; other private record retention boundaries. | Advance time, delete one import, revoke, export/purge, depart/rejoin/reactivate each import, terminal-delete identity, and scoped-delete mapping/import. | Import handles do not age out; other support preserves unwritten work; revocation deletes imports and the private operation/suppression ledger; departure pauses; terminal/scoped purge is narrow; successful follows and historical notifications remain. | `appview/internal/instagram/retention_test.go`, `appview/internal/instagram/account_data_test.go`, `appview/internal/instagram/account_store_test.go` |
| IT-011 | FR-017, FR-019, FR-020, FR-028, FR-030, NFR-005, RULE-006, RULE-008, RULE-009 | AC-029–AC-031, AC-037, AC-048 | Produce initial and future automatic follows and actorful notifications transactionally. | Retained accounts-followed handles; initial import plus every future trigger; duplicate/concurrent runs; several eligible targets; revocation, policy loss, departure before/after PDS success. | Trigger import, verification confirmation/enable/reactivation, validated username change, explicit membership reactivation, and safety restoration; process operations; inspect PDS, feed/newness, and outbox. | Triggers dedupe one operation per importer/target; each successful deterministic PDS follow creates one separate actorful notification and at most one identity-free push; no digest/count/group/coalescing exists; pre-write invalidation cancels work; post-success lifecycle changes preserve follow/history. | `appview/internal/instagram/future_match_test.go`, `appview/internal/notifications/instagram_match_test.go` |
| IT-012 | FR-020–FR-022, FR-030 | AC-034–AC-037, AC-048 | Extend notification schema/API/newness/preferences/push with exact actorful type-discriminated constraints. | Existing social type fixtures; actorful `instagramMatch` fixture without `kind`, destination, system, or source record; malformed combinations; unknown type; fixed-scope preference; block/mute/unavailable actor; identity canaries. | Migrate; insert invalid/valid variants; activate/read/page/mark seen/patch/push each type; change relationship and actor visibility. | DB/API require target actor and forbid URI/CID/rkey/system for `instagramMatch`; API omits `kind`/destination; fixed scope rejects mutation; push contains only category/notification/account binding; row hydration reflects current follow state and existing actor-visibility rules; existing social types remain unchanged. | `appview/internal/api/instagram_notification_test.go`, `appview/internal/push/instagram_payload_test.go` |
| IT-013 | FR-001, NFR-002, NFR-004, NFR-006 | AC-001, AC-040, AC-046 | Wire fail-closed configuration, shared limiter, dependencies, route policies, worker lifecycle, and readiness before exposing routes. | Dev/prod complete/empty/partial configs; trusted/untrusted proxy setup; Postgres limiter; fake clients/workers; single/multi-replica flags; server shutdown. | Load config/dependencies before mux; start routes/workers; inspect readiness; disable/re-enable Meta dependency; shutdown with jobs leased. | Disabled mode creates no external client and gates only Meta-dependent work; complete mode wires the narrow client/dispatcher/shared limiter; policy inventory is exact; partial/unsafe scaling fails before route readiness; shutdown is bounded. | `appview/internal/app/instagram_wiring_test.go`, `appview/internal/routes/policy_test.go`, `appview/cmd/appview/instagram_lifecycle_test.go` |
| IT-014 | FR-013, FR-016, FR-025, RULE-007 | AC-016–AC-018, AC-023, AC-026, AC-039, AC-040 | Make the Flutter API/repository consume the retained wire fixtures while removing suggestion methods. | `http_mock_adapter`; shared golden success/error/state fixtures; parser outputs; import response without `initialSuggestionCount`; unknown additive fields; controlled raw-export canary. | Exercise every verification/account/settings/import method and all public states/errors; compile/inspect the API/repository surface for old suggestion methods/models. | Retained methods, IDs, success codes, camelCase unions, null/omitted fields, cursors, and errors match §12.1; fixed-account Dio is used; no suggestion method/model remains; no raw canary crosses the request boundary; additive response fields and safe unknown enums remain tolerated. | `app/test/instagram_migration/data/instagram_migration_api_client_test.dart`, `app/test/instagram_migration/data/instagram_wire_contract_test.dart` |
| IT-015 | FR-024–FR-026, NFR-008 | AC-003, AC-005, AC-009, AC-014, AC-023–AC-026, AC-035, AC-037, AC-042 | Coordinate account-scoped verification/import and notification-follow state. | A/B fixed repositories; controllable poll/mutation/follow completers; expiry timer; success/error/conflict/unavailable states. | Start under A, switch to B, release stale completions; retry, import, and toggle follow from a notification. | State transitions are correct, timers stop, no stale UI/mutation/navigation crosses account, and actions remain idempotent/retryable; no suggestion review state is loaded. | `app/test/instagram_migration/providers/instagram_migration_provider_test.dart`, `app/test/notifications/providers/notifications_provider_test.dart` |
| IT-016 | FR-016, FR-023–FR-026, NFR-007 | AC-003, AC-005, AC-009, AC-014, AC-018, AC-023, AC-026, AC-034, AC-038, AC-048, AC-050 | Render and use the typed Instagram page in every state. | Provider fakes for disabled/empty/loading/error/attempt/candidate/verified-account/active and membership-inactive imports/conflict; semantics tester. | Enter from Settings; copy/open/cancel/confirm/toggle/revoke; verify imports are hidden without verification; import manual/JSON/ZIP entries; open linked Notification Settings text; delete/reactivate imports; inspect every selector/default/position/copy state. | User-facing copy uses verified/verification; active summary is `Verified as @…`; enabled discovery uses semantic success/moss green; Instagram export is the default selector option; import copy discloses current/future automatic public follows; People You May Know is absent; revoke appears after imports at page bottom with confirmation; no preview/retention/review controls remain. | `app/test/instagram_migration/instagram_migration_page_test.dart`, `app/test/settings/settings_page_test.dart`, `app/test/router/instagram_route_test.dart` |
| IT-017 | FR-020–FR-023, FR-026, NFR-007, NFR-008 | AC-034–AC-038, AC-042 | Decode/render/configure/open actorful automatic-follow notifications under the exact account. | Known social/match/unknown feed rows; target profile and viewer relationship variants; identity-free provider payloads; A active/B retained. | Render feed/settings; patch push; toggle Follow/Following; open row and foreground/background push while switching accounts. | Match row identifies the followed target, uses localized automatic-follow copy, opens target profile, and reuses the ordinary follow control; scope is hidden; push patch works; push opens B's Notifications feed after account activation; stale work cannot affect A/B; existing categories remain unchanged. | `app/test/notifications/instagram_match_notification_test.dart`, `app/test/notifications/notification_open_flow_test.dart` |
| IT-018 | FR-029 | AC-032, AC-045 | Exercise bounded operator conflict/job/revoke/retention/resolve commands and exact conflict states. | Isolated DB; `open`, `resolvedKeepExisting`, `resolvedRevokeExisting`, and `expired` conflicts; jobs; terminal private records; controlled synthetic identity/secret canaries. | Run each supported CLI command with missing/wrong/correct opaque IDs, 500/501 row batches, each transition, and repeat. | Explicit opaque arguments, maximum 500-row pagination, bounded redacted output, valid audited transitions only, idempotency, evidence anonymization on expiry, no import-expiry command, and no silent transfer. | `appview/cmd/cli/instagram_test.go` |
| IT-019 | NFR-009 | AC-047 | Preserve canceled/499 observability across Instagram handlers and polling. | Handler/store/worker operations returning `context.Canceled`; real middleware/observer; genuine 5xx control. | Cancel requests and compare captured status/log/Sentry decisions. | Cancellation is classified 499/canceled and skipped by Sentry; genuine 5xx remains captured. | `appview/internal/api/instagram_observability_test.go`, `appview/internal/observability/error_classifier_test.go` |
| IT-020 | BR-001, FR-003, FR-015, FR-017, FR-018, FR-025, FR-028, FR-030, NFR-004 | AC-048 | Prove one shared current-member guard across every authenticated route and worker transition. | Valid unexpired Alice session/device ID whose DID is absent from `craftsky_profiles`; one instance of every owned resource; queued verification/match/automatic-follow/notification work; control current member. | Enumerate every authenticated Instagram route and run every worker transition for departed Alice, then reinsert her profile and retry before and after explicit mapping/per-import reactivation. | Every route is `404 profile_not_found`; no verified mapping/automatic-follow notification/PDS write is created; owner state is inactivated/paused and dependent unwritten work cancelled; rejoin alone restores nothing; mapping and each paused import require explicit reactivation; the current-member control still works. | `appview/internal/api/instagram_membership_test.go`, `appview/internal/instagram/membership_transition_test.go` |
| IT-021 | FR-002–FR-018, FR-020–FR-022, NFR-004 | AC-003–AC-005, AC-009, AC-014–AC-017, AC-023–AC-027, AC-032, AC-034–AC-037, AC-040 | Lock the retained routes and actorful notification wire contract with shared synthetic golden JSON. | One fixture for each retained §12.1 request/success/error, every public enum, social and actorful `instagramMatch` type, optional/omitted field, privacy-preserving DELETE, identical replay, and negative fixtures for removed suggestion/count/system/destination fields. | Serialize AppView responses and decode/encode the same corpus in Flutter; compare POST verification/import 201, reads/PATCH 200, DELETE 204, callback 200/403/429 plus Retry-After, documented errors, and removed route behavior. | Go/Dart agree on camelCase, enums, safe unknown client behavior, opaque IDs/cursors, errors, DELETE no-ops, webhook retry contract, actorful match shape, and absence of `kind`, destination, `initialSuggestionCount`, public suggestion payloads, and internal/private fields. | `appview/internal/api/instagram_wire_contract_test.go`, `app/test/instagram_migration/data/instagram_wire_contract_test.dart`, `docs/changes/2026-07-11-instagram-dm-verification/fixtures/instagram_wire/` |
| IT-022 | FR-003, FR-024, NFR-008 | AC-042, AC-049 | Resume one current verification attempt across Flutter page disposal without weakening account boundaries. | Alice/Bob current-attempt fakes; matching/missing/mismatched/expired secure snapshots; pending-DM, processing, and pending-confirmation server states; controllable late reads. | Create, dispose, reopen, switch accounts, expire, cancel, confirm, and simulate server supersession. | AppView current state is authoritative; matching local display data is restored only for its DID and verification ID; polling/confirmation resumes; missing display data is not reconstructed; terminal/mismatch/session-invalidated snapshots clear narrowly; no reopen creates a new attempt. | `appview/internal/instagram/verification_store_test.go`, `appview/internal/api/instagram_verifications_test.go`, `app/test/instagram_migration/data/instagram_verification_storage_test.dart`, `app/test/instagram_migration/providers/instagram_migration_provider_test.dart` |
| IT-023 | BR-002, FR-013, FR-025, FR-031, NFR-008, NFR-010, RULE-007, RULE-011 | AC-016–AC-019, AC-042, AC-050–AC-054 | Run native export selection through isolate parsing and the existing repository request boundary. | Widget/provider harness with verified Alice; picker returning JSON/ZIP paths or cancellation; real synthetic temporary ZIP; controllable isolate/parser completion; Bob account switch; recording repository and messenger. | Select each container, cancel, dispose, switch A-to-B before completion, trigger every safe parse error, and inspect visible copy/request/diagnostics. | Instagram export is default; JSON/ZIP submit only `instagramJson` plus normalized usernames; UI recommends accounts-followed export, explains all-information locality and automatic public follows; parsing stays asynchronous; cancellation is silent; errors are localized; late Alice result never uploads under Bob; raw path/bytes/URL/unrelated canaries are absent outside parser input. | `app/test/instagram_migration/instagram_import_privacy_test.dart`, `app/test/instagram_migration/instagram_migration_page_test.dart` |
| IT-024 | BR-003, FR-017, FR-032, NFR-005 | AC-055 | Select and rotate the exact owner's OAuth session during background automatic-follow writes. | Alice has old/new valid sessions, revoked/expired/inactive sessions, and a provider-invalid newest session; Bob has a newer valid session; one pending Alice operation; fake PDS records selected session IDs. | Claim the operation with each session matrix, simulate provider invalidation, retry, remove all Alice sessions, and race two workers. | Only Alice's most recently active usable session is selected with deterministic tie-break; Bob's session is never observed; provider invalidation revokes only the exact Alice session before the next Alice session is tried; no usable Alice session yields retry/no PDS call; concurrency still produces one deterministic write. | `appview/internal/instagram/automatic_follow_worker_test.go`, `appview/internal/auth/background_session_selector_test.go` |
| IT-025 | BR-004, FR-017, FR-019, RULE-012 | AC-025, AC-056 | Preserve manual unfollow as suppression for the current verification lifetime. | Alice automatic-followed Bob; indexed follow then manually deleted; same and different supporting imports; every reconciliation trigger; current and new verification generations. | Run duplicate import, username/discovery/membership/safety restoration and reconciliation before/after manual unfollow; revoke; verify/import again. | The followed/alreadyFollowing terminal ledger prevents a new operation or PDS write for the old generation; import support changes do not erase suppression; revocation deletes the private ledger; a new verification plus fresh import may create one new operation. | `appview/internal/instagram/reconciliation_test.go`, `appview/internal/instagram/automatic_follow_worker_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing `/v1/*` auth, device ID, body limits, route policy inventory, camelCase, and error envelopes remain consistent. | NFR-004 | AC-004, AC-005, AC-040 | Extend route-policy/contract tests and rerun representative auth/error middleware suites. |
| REG-002 | Existing social notification categories retain actor/source semantics, preference defaults, feed hydration, newness, and push behavior. | FR-020, FR-021 | AC-034–AC-037 | Run all AppView notification/index/push tests; `instagramMatch` is the only actorful type allowed to omit a social URI/CID/rkey, and it cannot weaken any existing type constraint. |
| REG-003 | Existing Flutter social and unknown notification rows/settings/open destinations remain functional and forward compatible. | FR-020–FR-023, FR-026 | AC-034–AC-038 | Run notification model/page/settings/destination/navigation/open suites after adding actorful automatic-follow copy/profile navigation and the existing follow control. |
| REG-004 | Ordinary profile follow/unfollow and deterministic follow service behavior remain idempotent; Instagram lifecycle never deletes a successful follow. | FR-017, RULE-008, RULE-010 | AC-024, AC-025, AC-031, AC-044 | Run existing follow handler/store/provider suites plus background follow-service tests, crash recovery, already-following no-op, and successful-follow lifecycle sentinels. |
| REG-005 | Current CraftSky membership and moderation/visibility boundaries remain authoritative even while an old session remains valid. | FR-015, FR-030 | AC-020, AC-021, AC-048 | Run profile/search/moderation tests; assert Instagram uses the shared current-member guard and one named eligibility policy, returns `profile_not_found` after departure, and fails closed when required block/mute state is unavailable. |
| REG-006 | No PDS token, Meta secret, private import/link value, or new lexicon is introduced into Flutter/PDS records. | BR-002, NFR-003, NFR-006 | AC-039, AC-040 | Run controlled-canary scans, inspect dependency/API boundaries, and assert `lexicon/` is unchanged; do not introduce real or user-derived fixtures. |
| REG-007 | Existing observability redaction, bounded attributes, and Sentry privacy remain intact. | NFR-003 | AC-039 | Run AppView/Flutter observability and secret-scan suites with Instagram sentinels. |
| REG-008 | Client cancellation remains 499/no-Sentry while real server/provider failures retain existing error capture. | NFR-009 | AC-047 | Run shared error-classifier, route metrics/logging, and cancellation regressions. |
| REG-009 | Multi-account auth, OAuth-session ownership, notification binding, and account-scoped providers remain isolated. | FR-032, NFR-008 | AC-042, AC-055 | Run account-switch routing, exact-DID background-session selection, narrow session invalidation, fixed-account Dio, and notification-open suites with Alice/Bob sessions. |
| REG-010 | Database migration up/down and clean bootstrap still work from an empty schema without turning membership departure into broad private-data deletion. | FR-009, FR-020, FR-028, FR-030 | AC-015, AC-028, AC-034, AC-044, AC-048 | Run migration/full-schema setup, actorful match constraints, operation idempotency, membership removal/rejoin, scoped purge, and terminal identity purge after the migration change. |
| REG-011 | AppView and Flutter cannot silently diverge on Instagram states, fields, removed suggestion routes, notification variants, success codes, or errors. | FR-002–FR-025, NFR-004 | AC-003–AC-005, AC-009, AC-014–AC-017, AC-023–AC-027, AC-032, AC-034–AC-040 | Run both consumers against TD-011 and fail on an unreviewed golden-fixture delta, including any reappearance of `kind`, destination, `system`, `initialSuggestionCount`, or suggestion models/routes. |
| REG-012 | Resumable verification storage cannot outlive or cross its owning account boundary. | FR-024, NFR-008 | AC-042, AC-049 | Run account-switch/session-invalidation tests with two independent snapshots and controlled late completions; verify only the invalidated account is cleared. |
| REG-013 | Direct JSON import and the AppView `instagramJson` wire/storage contract remain unchanged when ZIP is added. | BR-002, FR-012–FR-014, RULE-007, RULE-011 | AC-016–AC-019, AC-050, AC-054 | Run existing parser, request-model, shared-wire, API, repository, database, and UI tests; assert no `instagramZip`, raw path/archive field, schema migration, or source-specific server branch appears. |
| REG-014 | A manually unfollowed automatic match stays unfollowed, and no hidden suggestion review surface survives the migration. | FR-016, RULE-012 | AC-023, AC-056 | Run reconciliation repeatedly after manual unfollow and scan route/model/provider/widget registries for removed suggestion APIs and People You May Know; only revocation plus fresh verification/import may authorize a new operation. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Stable CraftSky actors | Alice/Bob/Carol current-member DIDs; Alice with valid session after profile removal and after rejoin; hidden/taken-down DID; several same-DID sessions plus cross-DID sessions; active-follow, manual-unfollow, block-both-directions, and importer-mute variants. | AT-001–AT-009, IT-001–IT-025 |
| TD-002 | Challenge/state fixtures | Deterministic secure random stream; canonical 30-symbol alphabet and 13-symbol `CSKY-XXXX-XXXX-XXXX-X` token; keyed digest secret; fake clock around ten-minute boundaries; every exact attempt/mapping/import/automatic-follow/conflict state and allowed/forbidden transition. | AT-002, UT-001, UT-002, UT-020, IT-001, IT-002, IT-004–IT-006, IT-021 |
| TD-003 | Meta callback fixtures | Wholly synthetic exact raw valid/mutated JSON bytes/signatures; one, 100, and 101 incoming events with `mid`, sender/recipient IDs, and text; echo/self/deleted/non-text/unsupported/wrong-account/unknown variants; expected minimal hashed work rows. | UT-003, UT-004, IT-003, IT-004, MAN-001 |
| TD-004 | Meta profile/reply fixtures | IGSID 100/200; valid/changed/missing/invalid usernames; 2xx/4xx/429/5xx/timeout bodies with secrets removed; messaging-window boundaries. | UT-006, UT-007, IT-004–IT-006 |
| TD-005 | Import/parser fixtures | Manual lines and wholly synthetic versioned accounts-followed JSON with legacy direct values and observed exact `_u/<username>` URL plus agreeing-title shapes; follower and unrelated media/message/profile canaries; mismatch/noncanonical URL/changed/malformed/20 MiB/Unicode/duplicate/follower-only variants. Approved real exports remain only in the separate manual lane. | AT-003, AT-009, UT-005, UT-009, UT-010, UT-017, IT-007, IT-014, IT-023, MAN-002, MAN-005 |
| TD-006 | Eligibility/mapping matrix | Every exact verified-mapping state; discovery/verification/conflict/current-username facts; same IGSID/username; old/new usernames; importer/target membership; self/follow; hide/takedown; block either direction; importer mute; safety-source outage. | AT-004–AT-006, AT-008, UT-006, IT-005, IT-006, IT-008–IT-012, IT-020, IT-025 |
| TD-007 | Import/lifecycle matrix | Two additive accounts-followed imports that jointly support one operation; verified/unverified owners; long elapsed time; per-import deletion; verification revocation; membership-inactivation/reactivation; followed/alreadyFollowing suppression; terminal-purge variants. | AT-005, AT-008, UT-020, IT-001, IT-007, IT-010, IT-011, IT-020, IT-025 |
| TD-008 | Notification matrix | Existing social categories; exact actorful `instagramMatch` payload without `kind`, destination, system, or source fields; target viewer Follow/Following states; unknown/malformed types; several distinct successful operations; pending/retry/leased/sent deliveries; A/B account bindings; block/mute/unavailable actor. | AT-004, AT-005, AT-007, UT-012–UT-014, IT-001, IT-009, IT-011, IT-012, IT-017, IT-021 |
| TD-009 | Controlled privacy canaries | Unique wholly synthetic challenge/digest, username, IGSID, handle list, webhook message/body, Meta token/app secret/verify token/signature, raw export filename/content, and upstream response; separate explicitly approved redacted fixture lane; no real/user-derived values. | UT-004, UT-010, UT-014, UT-015, IT-003, IT-007, IT-014, IT-018, REG-006, REG-007 |
| TD-010 | Concurrency/work traces | Barriers/completers for duplicate webhook, four workers plus fifth claimant, concurrent confirmation and automatic-follow claims, PDS success/failure/firehose delay/crash, safety change at final revalidation, session invalidation, membership removal, manual unfollow, account switch, and late response. | UT-002, UT-011, UT-019, UT-020, IT-003–IT-005, IT-009, IT-011, IT-015, IT-017, IT-020, IT-024, IT-025 |
| TD-011 | Shared wire golden corpus | Synthetic JSON for every retained §12.1 request/success/error; all attempt/mapping/import/conflict enums; type-discriminated social and actorful match notifications with no `kind`/destination; negative old suggestion/system/count fixtures; optional/omitted fields; owned/foreign/absent/purged DELETE 204; callback 200/403/429 and Retry-After; repeated idempotent results. | IT-002, IT-005–IT-008, IT-012, IT-014, IT-021 |
| TD-012 | Limit boundaries | Exact §12.4 maxima and values one below/at/one above: 256 KiB/100 events, rate buckets and response policy, one MiB/10,000 imports, 20 MiB standalone/ZIP-target JSON, 100,000 ZIP entries, 64 MiB ZIP central directory, 20/50 pagination, 5-second/64 KiB Meta calls, webhook concurrency four, automatic-follow batch 20/100, 60-second leases/five attempts/backoff, 15-minute webhook age, and 500-row operator batch. | UT-004, UT-007–UT-009, UT-016, UT-018, IT-003, IT-004, IT-007, IT-011, IT-018, IT-023, IT-024 |
| TD-013 | Synthetic ZIP corpus | Temporary archives generated during tests with one canonical following JSON; stored/deflated content; unrelated media/message/follower canaries; missing/duplicate/lookalike/encrypted/unsupported/corrupt targets; ZIP64 and central-directory/target-size boundaries. No user-derived archive, filename, username, or payload is committed. | AT-009, UT-018, IT-023, REG-013 |
| TD-014 | Background OAuth session matrix | Alice sessions covering valid/revoked/expired/inactive/provider-invalid states, distinct/equal last-active timestamps and stable IDs; a newer Bob session; fake PDS client recording session and deterministic rkey. | UT-019, IT-009, IT-024, REG-009 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-001, FR-004–FR-007, FR-027, NFR-002, NFR-006 | AC-006–AC-013, AC-040, AC-041, AC-043, AC-046 | Meta capability and production-configuration spike. | Create/configure the owned professional account and Meta Business app; provision secrets; subscribe HTTPS webhook; send from an unrelated personal account; inspect signed payload/IGSID; fetch username; send allowed reply; validate Standard/Advanced access, Live mode, token renewal, privacy/deletion/review and deployment/shared-limit requirements. | Real redacted fixtures match or update adapters; personal sender verifies; profile/reply work; integration remains disabled until every checklist item passes. |
| MAN-002 | FR-013, FR-014 | AC-016–AC-019, AC-051 | Current standalone Instagram JSON compatibility. | Obtain consented redacted current accounts-followed JSON exports; import them on each supported mobile platform without network inspection shortcuts. | Parser recognizes supported direct-value or exact URL/title following shapes locally, sends only normalized usernames, ignores follower data, and provides clear local guidance for unsupported variants. |
| MAN-003 | FR-016, FR-023–FR-026, NFR-007 | AC-003, AC-009, AC-014, AC-018, AC-023–AC-026, AC-034, AC-038 | Final mobile responsive, accessibility, clipboard, file-picker, external-link, and automatic-follow disclosure behavior. | On supported phones/tablets, use screen reader and large text; inspect long imports, errors, disabled integration, candidate confirmation, green discovery switch, picker cancel, DM link, default export selector, automatic-follow copy, absent People You May Know, bottom revoke action/dialog, and conflict states. | Focus/semantics/copy make verification, public automatic-follow consent, retention, and revocation unambiguous; user-facing copy never says linked/link; no raw data or server error is exposed; native interactions work safely. |
| MAN-004 | FR-020–FR-023, FR-026 | AC-035–AC-037, AC-042 | Physical-device actorful notification and identity-free push lifecycle for an inactive account. | Retain two accounts; produce separate successful automatic follows; disable/enable push; deliver foreground/background/terminated pushes; inspect provider payload; open push and row; toggle Follow/Following; revoke or unfollow after success. | Push names nobody and contains no target identity; correct account activates before Notifications feed; row identifies the target, opens the profile, and toggles current relationship; history remains after revoke/unfollow; invalid pre-success work does not deliver. |
| MAN-005 | FR-013, FR-025, FR-031, NFR-010 | AC-050–AC-053 | Real Instagram ZIP compatibility and mobile responsiveness. | Without copying it into the repository, select the approved 2026-07-21 all-information ZIP on iOS and Android; repeat with an export containing larger media; cancel once; background/foreground once; monitor UI responsiveness and peak memory; inspect the mocked/dev request only after local parsing. | The original ZIP imports its 88 agreeing following records, unrelated data stays local/unextracted, UI remains responsive, memory does not scale with total archive payload, cancel/background behavior is safe, and the request is the existing `instagramJson` normalized-entry shape. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Real Meta payload, profile lookup, reply, access level, token lifecycle, and messaging window cannot be proven hermetically. | BR-001, FR-004–FR-007, FR-027, NFR-006 | No Meta app or owned professional account is configured. | Keep adapters synthetic-fixture-driven and integration disabled; MAN-001 is a hard production gate and may contribute redacted schema observations from which non-user synthetic fixtures are created. |
| GAP-002 | Instagram export JSON/ZIP shape has no stable public schema and only one approved current archive has been inspected. | FR-013, FR-014, FR-031 | The sample validates one 2026-07-21 structure but cannot prove every locale/account/export variation. | Keep manual text fallback; encode only its structural observations in wholly synthetic fixtures; run MAN-002/MAN-005 with additional consented exports; never commit a user-derived archive. |
| GAP-003 | Physical push and OS process lifecycle cannot be fully reproduced in unit/widget tests. | FR-020–FR-023, FR-026 | Firebase/APNs and terminated launches depend on provider/OS state. | Keep actorful-row and identity-free provider simulations automated and require MAN-004 before release. |
| GAP-004 | Platform secure storage/network proxy, clipboard/file-picker path lifetime, isolate scheduling, and physical peak-memory guarantees are outside hermetic Dart/Go tests. | FR-031, NFR-003, NFR-007, NFR-010 | Mocks and temporary-file tests validate app behavior, not every iOS/Android document-provider or OS-memory condition. | Run MAN-003/MAN-005, platform configuration review, memory profiling, and privacy proxy inspection before release. |
| GAP-005 | Repository-wide member data-export/account-deletion routes do not yet exist. | FR-028 | This slice can test scoped export/purge, reversible membership inactivation, and terminal identity purge, but cannot wire a nonexistent general endpoint/UI. | Treat `IT-010`/`IT-011` as the reusable contract and require composition when the general lifecycle feature lands. |
| GAP-006 | The Postgres-backed shared limiter has not yet been exercised in the eventual live multi-replica deployment. | NFR-002 | Correct code and compose tests cannot prove the final edge/proxy/replica topology. | Keep unsafe multi-replica configuration fail-closed and validate the trusted-proxy plus shared-bucket behavior in MAN-001 before enablement. |
| GAP-007 | Full dispute evidence policy remains manual and exceptional. | FR-029 | Email/support evidence cannot be truthfully automated. | Automated tests prove no transfer and explicit audited commands; human evidence review remains operator procedure. |

## 10. Out Of Scope

- Scraping, Instagram follower/following API reads, OAuth-only verification, ManyChat, export-possession proof, or server-side archive parsing.
- Flutter web ZIP parsing, whole-archive in-memory decode, and extraction of
  unrelated ZIP entries.
- Collection or persistence of accounts that follow the importing member or follower-derived matches.
- PDS storage of private Instagram data or a lexicon change.
- Marketing/future-match Instagram DMs.
- A repository-wide member data-export/account-deletion feature; only scoped composable primitives are tested here.
- Automatic conflict adjudication or silent identity transfer.
- Member-facing Instagram suggestion list, accept, dismiss, pagination, or
  People You May Know behavior; regression tests assert these stay removed.
- Automatic re-follow from the same verification lifetime after a manual
  unfollow.
- Production enablement before every manual release gate is complete.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-11-instagram-dm-verification/`
- Risk level: **High**; this revised test design requires explicit user
  approval, followed by document review, before coding-plan or implementation
  work.
- Recommended first failing test for this revision: `IT-024` in
  `appview/internal/instagram/automatic_follow_worker_test.go`, proving exact
  owner OAuth-session selection before any background PDS write. The next
  failing test is `UT-020`/`IT-025` for durable manual-unfollow suppression.
- Suggested test order for implementation:
  1. `UT-019`, `IT-024`: exact owner OAuth-session selection, narrow
     invalidation, no cross-DID fallback, and retryable absence.
  2. `UT-020`, `IT-025`: automatic-follow operation transitions and
     manual-unfollow suppression before reconciliation changes.
  3. `IT-001`: migrate operation/notification constraints to the actorful
     contract while preserving clean bootstrap/up/down behavior.
  4. `UT-006`, `IT-008`: reuse final eligibility policy, make match/operation
     state private, and remove suggestion routes.
  5. `IT-009`: automatic worker claim, deterministic PDS write, crash recovery,
     already-following no-op, and exactly-one notification completion.
  6. `IT-010`, `IT-011`: import support/revocation lifecycle plus initial/future
     automatic-follow triggers.
  7. `UT-013`, `UT-014`, `IT-012`: fixed preference, actorful feed constraints,
     identity-free push, and current relationship hydration.
  8. `IT-021`: update the shared Go/Flutter corpus to remove suggestion/count/
     system/destination payloads and add actorful `instagramMatch`.
  9. `UT-011`, `UT-012`, `IT-014`, `IT-015`, `IT-017`: remove Flutter
     suggestion state, render actorful rows, and fence follow/profile navigation
     to the notification owner account.
  10. `IT-016`, `IT-023`: verified terminology, green discovery, default
      Instagram export, informed automatic-follow copy, absent People You May
      Know, and bottom revocation.
  11. `REG-001`–`REG-014`: broad API, notification, auth, privacy, cancellation,
      parser, lifecycle, and no-reappearance regressions.
- Commands discovered:
  - From `appview/`: focused `go test ./internal/instagram ./internal/integrations ./internal/api ./internal/routes ./internal/notifications ./internal/push ./internal/app ./internal/auth ./internal/followwrite`
  - From repository root: `just test`
  - From repository root: `just fmt`
  - From `app/`: `dart run build_runner build --delete-conflicting-outputs`
  - From `app/`: `flutter gen-l10n`
  - From `app/`: `flutter test test/instagram_migration test/notifications test/router test/settings`
  - From `app/`: `flutter analyze`
  - From `app/`: `flutter test`
- Blocking gaps: explicit approval of this revised acceptance-test
  specification and the required document-review stage precede implementation.
  `GAP-001`–`GAP-004` and `GAP-006`
  block relevant production enablement/release confidence; `GAP-005` blocks
  only future repository-wide lifecycle composition. `MAN-005` remains the
  physical-device release check for ZIP memory/path behavior.

## 12. AppView Audit Correction Tests

Status: Approved test-design amendment on 2026-08-14.

AT-004, AT-005, and AT-007 and the automatic-write expectations in IT-008,
IT-009, IT-011–IT-017, IT-020, IT-021, IT-023–IT-025, REG-004, REG-008,
REG-010, REG-012, and REG-014 are historical. They remain evidence for the
pre-audit implementation but are not green expectations. The tests below trace
to BR-005, FR-033–FR-037, NFR-011, RULE-013, and AC-057–AC-063.

### AT-010: Matching Creates A Private Suggestion Without A Public Effect

Requirement IDs: BR-005, FR-033, FR-035, AC-057, AC-062

```gherkin
Given Alice has an active verified Instagram import
And Bob is an exact eligible current CraftSky member
When initial or future reconciliation observes the match repeatedly
Then exactly one caller-private suggestion exists for Alice and Bob
And the suggestion records both current owner generations
And no OAuth session is selected
And no PDS follow writer, notification creator, or push sender is called
And Bob cannot discover that the suggestion or import evidence exists
```

### AT-011: Explicit Acceptance Is The Only Instagram Follow Boundary

Requirement IDs: BR-005, FR-034, NFR-011, AC-058–AC-060

```gherkin
Given Alice owns a pending private suggestion for Bob
And both participants remain current at the recorded generations
When Alice explicitly chooses Follow
Then AppView revalidates ownership, both generations, and the full safety policy
And the ordinary owner-effect/session coordinator requests one deterministic follow
And replaying the same acceptance returns the same followed or already-following result
And no instagramMatch automatic-follow notification is created
```

### AT-012: Lifecycle Transitions Retire Unwritten Suggestions

Requirement IDs: FR-033, FR-035, NFR-011, RULE-013, AC-061, AC-062

```gherkin
Given matching has created a private suggestion
When either participant departs or becomes terminal before acceptance crosses the final effect boundary
Then the suggestion becomes invalidated and no PDS call begins
And no background component can later select a session or write the follow
And account or terminal cleanup never deletes an app.bsky.graph.follow record
```

### Unit And Contract Cases

| ID | Requirement IDs | Test |
|---|---|---|
| UT-021 | FR-033, RULE-013 | Suggestion transition table rejects stale participant generations and makes terminal invalidation monotonic. |
| UT-022 | FR-034, NFR-011 | Accept/dismiss ownership and idempotency table covers pending, followed, already-following, dismissed, invalidated, stale, and foreign IDs. |
| UT-023 | FR-035, AC-062 | Dependency capability test proves matcher/reconciliation interfaces contain no session selector, PDS factory, follow writer, or public-write method. |

### Integration And Client Cases

| ID | Requirement IDs | Test |
|---|---|---|
| IT-026 | FR-033, FR-035, AC-057, AC-062 | Real-store duplicate/reordered matching creates one private suggestion while recording zero external-effect calls. |
| IT-027 | FR-034, NFR-011, AC-058–AC-061 | Current-member accept uses the owner-effect boundary exactly once; wrong-owner, stale-generation, departure, and terminal barriers make no new call. |
| IT-028 | FR-037, AC-060, AC-062 | Schema/routes/preferences/push no longer expose or create `instagramMatch`; ordinary relationship behavior remains green. |
| IT-029 | FR-036, AC-063 | Flutter lists private suggestions, explains explicit Follow, and fences Follow/Dismiss/navigation/late results to the captured account. |
| REG-015 | FR-035, RULE-013 | Search and dependency-wiring regression finds no automatic-follow worker startup, background session selector, follow writer, `PutRecord`, or cleanup deletion of `app.bsky.graph.follow`. |

### Required Failure-First Order

1. UT-023 and REG-015 demonstrate that the current worker still has the
   forbidden capability.
2. IT-026 demonstrates that matching currently progresses toward a public
   operation rather than stopping at a private suggestion.
3. IT-027 establishes the explicit lifecycle-fenced acceptance boundary.
4. IT-028 removes the now-invalid automatic-follow notification contract.
5. IT-029 restores the private review surface and fixed-account Flutter flow.
6. Re-run the unaffected verification, privacy, ZIP, Meta-adapter, membership,
   import-retention, and account-isolation tests from Sections 1–11.

## 13. Instagram Match Notification Restoration Tests

Status: Approved test-design amendment on 2026-08-21.

### AT-013: A New Private Match Notifies Without Following

Requirement IDs: FR-038–FR-040, NFR-012, AC-064–AC-067

```gherkin
Given Alice imports an Instagram following list containing Bob
And Bob is an exact eligible current CraftSky member
When matching creates Alice's pending private suggestion for Bob
Then the same transaction creates one actor-backed instagramMatch notification for Alice
And no OAuth session is selected and no PDS follow is written
And replaying matching creates no additional notification or delivery
And Alice can open Bob's profile and explicitly choose Follow from the notification row
```

| ID | Requirement IDs | Test |
|---|---|---|
| IT-030 | FR-038, NFR-012, AC-064, AC-065 | Real PostgreSQL matching creates one suggestion plus one source-less actor notification atomically and remains idempotent. |
| UT-024 | FR-039, AC-066 | Category, fixed-scope preference, and identity-free push payload contracts include `instagramMatch`. |
| IT-031 | FR-040, AC-067 | Flutter decodes the actor-backed row, renders non-automatic copy and Follow/Following, and opens the matched profile under the captured account. |

Required order: IT-030, UT-024, IT-031, then the affected Instagram,
notification, push, API, app-wiring, migration, and Flutter notification suites.
