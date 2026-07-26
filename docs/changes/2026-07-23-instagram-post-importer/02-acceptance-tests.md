# Acceptance Test Specification: Instagram Historical Post Importer

## 1. Test Strategy

This is a high-risk, cross-stack import feature spanning an untrusted local
archive, browser OAuth, direct PDS writes, a public Lexicon change, AppView
indexing and activity policy, and Flutter presentation. The test design keeps
the privacy boundary observable and moves vertically through these layers:

- TypeScript unit tests cover archive path/schema adapters, deterministic
  normalization and TIDs, text repair/truncation/facets, safety limits, review
  state, safe diagnostics, OAuth authority, publication state, and progress
  retention.
- Worker integration tests exercise Blob-backed ZIP/ZIP64 inspection,
  target-only extraction, cancellation, hostile metadata, privacy canaries,
  incremental image sanitization, and one-at-a-time media processing.
- Browser component and Playwright acceptance tests exercise the local-first
  flow, virtualized review, mocked OAuth/PDS publication, resume, retry,
  cancellation, rollback, origin modes, accessibility, and network boundaries.
- Go and real-Postgres integration tests cover the additive Lexicon contract,
  migration, provenance indexing, response shaping, historical profile
  pagination, search, original-backfill timeline exclusion, notification
  suppression, later engagement, and ordinary-post regressions.
- Flutter model/widget tests cover optional provenance decoding and the subtle
  localized label on every shared post-card surface.
- Manual release checks are limited to real third-party OAuth/PDS behavior,
  current consented exports, deployed Cloudflare headers, large-device/browser
  behavior, and end-to-end firehose convergence that cannot be made
  deterministic in repository tests.

Tests use wholly synthetic fixtures. The consented local export is structural
compatibility evidence only and is never copied into the repository, snapshots,
logs, or diagnostics. Real-Postgres tests require `TEST_DATABASE_URL`; a
skipped database test is not passing evidence.

Risk remains **High**. The product owner approved the requirements on
2026-07-23. Live OAuth/PDS, additional current export shapes, and deployed
browser validation remain explicit release gates rather than implementation
blockers.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-011 | AT-001, AT-005, IT-007, IT-008, IT-010 | Acceptance / Integration | Yes |
| BR-002 | AC-002, AC-017 | AT-002, UT-012, IT-002, IT-003, IT-017 | Acceptance / Unit / Integration | Yes |
| BR-003 | AC-012, AC-013 | AT-008, IT-011, IT-012, IT-013, REG-003, REG-004 | Acceptance / Integration / Regression | Yes |
| BR-004 | AC-006, AC-014, AC-015 | AT-004, AT-006, AT-007, IT-005, IT-018 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-020 | AT-001, IT-008, IT-016, REG-008 | Acceptance / Integration / Regression | Yes |
| FR-002 | AC-003 | AT-003, UT-013, IT-006, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-003, AC-018 | AT-003, AT-010, UT-013, IT-006, IT-016 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-004 | AT-003, IT-006, IT-008 | Acceptance / Integration | Yes |
| FR-005 | AC-002, AC-005, AC-016 | AT-001, AT-002, UT-001, IT-002, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-005, AC-016 | AT-001, UT-018, IT-002, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-005, AC-019 | AT-002, UT-002, UT-003, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-005, AC-019 | AT-002, UT-003, UT-008, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-002, AC-017 | AT-002, UT-012, IT-003, IT-017 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-006 | AT-004, UT-016, UT-019, UT-020, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-007 | AT-004, UT-016, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-007 | AT-004, UT-003, UT-016, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-008 | AT-004, UT-005 | Acceptance / Unit | Yes |
| FR-014 | AC-008 | AT-004, UT-005 | Acceptance / Unit | Yes |
| FR-015 | AC-009, AC-013 | AT-004, UT-006, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-010, AC-016 | AT-005, UT-001, UT-007, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-002, AC-010 | AT-005, UT-019, IT-004, IT-007, IT-017 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-011, AC-019 | AT-005, UT-009, IT-001, IT-007, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-011, AC-017 | AT-005, UT-009, IT-001, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-020 | AC-011, AC-012 | AT-008, IT-009, IT-010, IT-011, IT-014 | Acceptance / Integration | Yes |
| FR-021 | AC-012 | AT-008, IT-011, IT-012, REG-003 | Acceptance / Integration / Regression | Yes |
| FR-022 | AC-013 | AT-008, IT-013, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-023 | AC-011 | AT-009, UT-015, IT-010, IT-015, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| FR-024 | AC-014, AC-019 | AT-006, UT-008, UT-009, IT-007, IT-018 | Acceptance / Unit / Integration | Yes |
| FR-025 | AC-014, AC-017 | AT-006, UT-010, UT-017, IT-005, IT-018 | Acceptance / Unit / Integration | Yes |
| FR-026 | AC-014 | AT-006, UT-010, IT-005, IT-007, IT-018 | Acceptance / Unit / Integration | Yes |
| FR-027 | AC-015 | AT-007, UT-010, IT-007, IT-018 | Acceptance / Unit / Integration | Yes |
| FR-028 | AC-014, AC-015 | AT-006, AT-007, UT-011, IT-005, IT-007, IT-018 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-002, AC-016, AC-017 | AT-002, IT-002, IT-003, IT-017 | Acceptance / Integration | Yes |
| NFR-002 | AC-016, AC-019 | AT-002, UT-001, IT-002, IT-004 | Acceptance / Unit / Integration | Yes |
| NFR-003 | AC-016 | AT-002, UT-018, IT-002, IT-004 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-018, AC-020 | AT-010, UT-013, IT-016, MAN-001, MAN-005 | Acceptance / Unit / Integration / Manual | Partial |
| NFR-005 | AC-017 | AT-002, UT-012, IT-003, IT-005, IT-017 | Acceptance / Unit / Integration | Yes |
| NFR-006 | AC-019 | UT-009, IT-001, REG-001 | Unit / Integration / Regression | Yes |
| NFR-007 | AC-020 | AT-010, MAN-001, MAN-002, MAN-003 | Acceptance / Manual | Partial |
| NFR-008 | AC-012, AC-013, AC-021 | IT-010, IT-011, IT-012, IT-013, IT-014, REG-001, REG-002, REG-003, REG-004, REG-005, REG-006, REG-007 | Integration / Regression | Yes |
| RULE-001 | AC-004 | AT-003, IT-006 | Acceptance / Integration | Yes |
| RULE-002 | AC-011 | AT-005, AT-009, UT-009, UT-015 | Acceptance / Unit | Yes |
| RULE-003 | AC-001, AC-003 | AT-001, AT-003, IT-008 | Acceptance / Integration | Yes |
| RULE-004 | AC-012, AC-013 | AT-008, IT-011, IT-012, IT-013, IT-014 | Acceptance / Integration | Yes |
| RULE-005 | AC-007, AC-010 | AT-004, AT-005, UT-007, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-006 | AC-015 | AT-007, UT-010, IT-007, IT-018 | Acceptance / Unit / Integration | Yes |
| RULE-007 | AC-014 | AT-006, UT-008, IT-007, IT-018 | Acceptance / Unit / Integration | Yes |
| RULE-008 | AC-002, AC-017 | AT-002, IT-003, IT-017 | Acceptance / Integration | Yes |
| RULE-009 | AC-022 | AT-002, UT-004, IT-002 | Acceptance / Unit / Integration | Yes |
| RULE-010 | AC-010, AC-023 | AT-004, AT-005, UT-007, UT-019, IT-004 | Acceptance / Unit / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Review Locally Before Authorizing Publication

Requirement IDs: BR-001, FR-001, FR-005, FR-006, RULE-003
Acceptance Criteria: AC-001, AC-005
Priority: Must
Level: Acceptance
Automation Target: `instagram-importer/e2e/local-review.spec.ts`

```gherkin
Feature: Local-first historical import
  Scenario: Parse a supported export before OAuth
    Given the standalone importer is opened without an OAuth session
    And the member acknowledges that selected records and images will be public
    When the member selects a synthetic supported Instagram ZIP
    Then archive processing occurs in the importer worker
    And a review manifest is shown without any OAuth or PDS network request
    And the member is told that the raw archive remains on this device
    When the member proceeds from review
    Then and only then does OAuth authorization begin
```

### AT-002: Ignore Unrelated And Hostile Export Content

Requirement IDs: BR-002, FR-005–FR-009, NFR-001–NFR-003, NFR-005, RULE-008, RULE-009
Acceptance Criteria: AC-002, AC-005, AC-016, AC-017, AC-022
Priority: Must
Level: Acceptance
Automation Target: `instagram-importer/e2e/archive-privacy.spec.ts`

```gherkin
Feature: Private bounded archive processing
  Scenario: Inspect a full-information export with hostile and unrelated entries
    Given a synthetic ZIP64 export contains supported posts plus message,
      relationship, profile, advertising, unrelated-media, path-conflict,
      malformed, duplicate, timestamp-boundary, and privacy-canary entries
    When the worker builds its manifest
    Then it reads only the bounded supported post paths and selected media
    And equivalent posts merge while material conflicts become skipped items
    And invalid timestamps are skipped without replacement
    And no raw canary, archive path, caption, handle, fingerprint, or media byte
      reaches storage, diagnostics, URLs, snapshots, or the network
    And archive-wide failures use only safe error codes and Posts-only guidance
```

### AT-003: Authorize The Exact Account And Authority

Requirement IDs: FR-002–FR-004, NFR-004, RULE-001, RULE-003
Acceptance Criteria: AC-003, AC-004, AC-018
Priority: Must
Level: Acceptance
Automation Target: `instagram-importer/e2e/oauth-preflight.spec.ts`

```gherkin
Feature: Narrow browser OAuth and membership preflight
  Scenario Outline: Publication authorization completes or fails closed
    Given the member has reviewed a local manifest
    When they continue with a <destination>
    Then no password or app password field is shown
    And the requested authority is limited to base identity, CraftSky post
      create/delete, and supported image-blob upload
    And no wildcard, update, account, identity-management, or unrelated
      permission is requested
    And the importer reads social.craftsky.actor.profile/self from the
      authenticated PDS
    And <outcome>

    Examples:
      | destination | outcome |
      | compatible member PDS | the DID and exact post/image counts are shown for final confirmation |
      | PDS without profile/self | publication remains blocked with existing-member guidance |
      | PDS without granular authority | publication remains blocked without a broader fallback |
      | denied or cancelled authorization | local review remains available and no write occurs |
```

### AT-004: Review Transformations At Scale

Requirement IDs: BR-004, FR-010–FR-015, RULE-005, RULE-010
Acceptance Criteria: AC-006–AC-009, AC-023
Priority: Must
Level: Acceptance
Automation Target: `instagram-importer/e2e/review.spec.tsx`

```gherkin
Feature: Bulk review
  Scenario: Review thousands of importable and skipped posts
    Given a manifest contains ordinary, nine-image, mixed-media, video-only,
      overlong, mojibake, unsupported-format, and conflicting posts
    When review opens
    Then every importable post, including warned posts, is selected by default
    And the virtualized list remains filterable and keyboard usable
    And aggregate selected, image, transformed, warning, and skipped counts are exact
    And every row shows its original date, final media selection, and warnings
    And its caption and image sections start collapsed and expand independently
    And the image heading states the number of images
    And the expanded caption editor grows to show its full content
    And expanding the image section progressively loads every selected
      thumbnail through the one-at-a-time worker queue
    And every thumbnail is at least 132 CSS pixels square
    When the member selects a thumbnail
    Then it opens in an accessible full-screen lightbox
    And the lightbox closes by its close button, Escape, or backdrop
    When the member deselects one post, deselects one image, and uses bulk
      deselect and select-all controls
    Then the per-post state and every aggregate count recompute exactly
    And the first four supported images remain in source order
    And videos and unsupported formats are never selected for upload
    And captions are repaired only when reversible, truncated at 2,000
      graphemes, and editable
    And repaired captions do not appear in warning badges, warning counts, or
      the warnings filter
    And final hashtags and URLs receive UTF-8-correct facets
    And Instagram handles remain plain text
```

### AT-005: Publish Exactly The Confirmed Shape

Requirement IDs: BR-001, FR-016–FR-019, RULE-002, RULE-005, RULE-010
Acceptance Criteria: AC-001, AC-010, AC-011, AC-019, AC-023
Priority: Must
Level: Acceptance
Automation Target: `instagram-importer/e2e/publication.spec.ts`

```gherkin
Feature: Direct PDS publication
  Scenario: Sanitize and publish selected posts
    Given final account-and-count confirmation is accepted
    When the importer processes one selected post
    Then it lazily decodes and re-encodes no more than one supported image at a time
    And it uploads only the final confirmed sanitized JPEG, PNG, or WebP images
    And each image carries empty alt text
    And it creates one create-only social.craftsky.feed.post record with the
      original createdAt, deterministic tid, final facets, and minimal
      self-asserted Instagram provenance
    And the record contains no Instagram identity, source identifier, filename,
      archive metadata, fingerprint, or external URL
    When any finally confirmed image fails at runtime
    Then that post fails and no record with a silently reduced shape is created
```

### AT-006: Resume Without Overwriting

Requirement IDs: BR-004, FR-024–FR-026, FR-028, RULE-007
Acceptance Criteria: AC-014
Priority: Must
Level: Acceptance
Automation Target: `instagram-importer/e2e/resume.spec.ts`

```gherkin
Feature: Idempotent resumable import
  Scenario: Resume interrupted and overlapping imports
    Given an import has successful, failed, and remaining deterministic items
    When the page reloads, the user signs out, and the same archive is reselected
    Then content-free progress remains and is bound to the same DID and manifest
    And successful records are never recreated, updated, or claimed again
    And failed and remaining records can continue
    And an independently deleted deterministic record can be recreated
    And an existing ordinary, conflicting, or edited record is skipped
    When a later export overlaps the earlier import
    Then it is a separate session and claims rollback ownership only for newly
      created records
```

### AT-007: Pause, Cancel, Retry, Clear, And Roll Back Safely

Requirement IDs: BR-004, FR-027, FR-028, RULE-006
Acceptance Criteria: AC-014, AC-015
Priority: Must
Level: Acceptance
Automation Target: `instagram-importer/e2e/recovery.spec.ts`

```gherkin
Feature: Import recovery controls
  Scenario: Control an incomplete import
    Given a partial import contains successes, transient failures, and pending work
    When the member pauses or cancels
    Then no new post starts and successful records remain
    When the member retries failures
    Then bounded backoff is applied and successful posts are not repeated
    When the member chooses rollback and confirms it
    Then only records with a durably stored create CID owned by that session
      are deleted using that CID as a precondition
    And already-absent, edited, replaced, CID-different recreated,
      pre-existing, ownership-ambiguous, and failed deletions are reported per record
    And the confirmation discloses that a byte-identical record recreated at
      the same rkey cannot be distinguished by its content CID
    And blobs and unrelated records are never targeted
    When the member clears local history
    Then the loss of resume and bulk-rollback capability is explained first
```

### AT-008: Index Historical Posts Without Backfill Activity

Requirement IDs: BR-003, FR-020–FR-022, NFR-008, RULE-004
Acceptance Criteria: AC-012, AC-013, AC-021
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/index/craftsky_post_import_test.go`,
`appview/internal/api/imported_post_store_test.go`

```gherkin
Feature: Imported-post AppView policy
  Scenario: Index an imported post and later engagement
    Given a member publishes imported and ordinary control posts
    When AppView indexes the original records
    Then the imported post appears on profile pages by original createdAt
    And it appears in applicable keyword and hashtag search
    And it never appears as an original authored home-timeline item
    And its mention facets, reply field, and quote embed activate no notification
    And the ordinary controls behave unchanged
    When another member later likes, reposts, quotes, or replies to the import
    Then the later activity follows ordinary timeline and notification rules
```

### AT-009: Show Durable Subtle Provenance

Requirement IDs: FR-023, RULE-002
Acceptance Criteria: AC-011
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/index/craftsky_post_import_test.go`,
`appview/internal/api/post_response_test.go`,
`app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Imported post presentation
  Scenario: Render imported and ordinary posts
    Given AppView returns an Instagram-imported post and an ordinary post
    When shared CraftSky post cards render them on profile, search, direct,
      and quoted-post preview views
    Then the imported post shows a subtle localized "Imported from Instagram" label
    And the ordinary post shows no import label
    And the label does not claim account verification or unchanged content
    When a CraftSky-aware replacement retains the provenance field
    Then its provenance and label remain
    When a third-party replacement omits that optional field
    Then AppView and Flutter no longer classify it as imported
```

### AT-010: Isolate Production OAuth And Deployment

Requirement IDs: FR-003, NFR-004, NFR-007
Acceptance Criteria: AC-018, AC-020
Priority: Must
Level: Acceptance
Automation Target: `instagram-importer/e2e/origin-policy.spec.ts`,
`instagram-importer/src/config/runtime.test.ts`

```gherkin
Feature: Origin-isolated static deployment
  Scenario Outline: Apply the correct runtime policy
    Given the importer is built for <mode>
    When runtime configuration and rendered security metadata are inspected
    Then <capability>
    And scripts and assets are self-hosted
    And no analytics, advertising, session replay, archive backend, or
      undisclosed network destination is present
    And handle entry discloses resolution through https://bsky.social

    Examples:
      | mode | capability |
      | localhost | real OAuth may use localhost-specific client metadata |
      | production import.craftsky.social | real OAuth uses production metadata and restrictive headers |
      | stable staging | real OAuth requires distinct staging metadata |
      | ephemeral preview | only synthetic fixtures and mocked PDS behavior are available |
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-005, FR-016, NFR-002 | AC-016, AC-019 | Validate the named/versioned safety envelope one below, at, and above every numeric boundary, including no overall/cumulative-size limit and one media job. | Boundary table for all configured limits | At/below accepted; above archive limits stop; above media limits omit; config stays typed/versioned | `instagram-importer/src/config/safety.test.ts` |
| UT-002 | FR-007 | AC-005, AC-019 | Normalize only exact supported paths under zero or one wrapper and reject traversal, absolute, duplicate, case-conflicting, confusable, arbitrary nested, or ambiguous targets. | Synthetic entry-name table covering flat Posts, six-digit date-partitioned Posts, flat Other, and rejected unrelated/nested paths | Deterministic accepted candidates or safe local error | `instagram-importer/src/archive/paths.test.ts` |
| UT-003 | FR-007, FR-008, FR-012 | AC-005, AC-007, AC-019 | Validate supported Instagram JSON adapters, media types and observed directories, equivalent merges, caption/timestamp source precedence, and material conflicts. | Versioned synthetic JSON variants, including dual legacy/detailed representations with an empty root caption, misleading metadata timestamps, and current date-partitioned/Other media paths | One normalized post per equivalent source using the supported media path/caption/timestamp; one skipped ambiguity per genuine conflict | `instagram-importer/src/archive/adapters.test.ts`, `instagram-importer/src/archive/inspectArchive.test.ts` |
| UT-004 | FR-018, FR-024, RULE-009 | AC-011, AC-022 | Enforce inclusive timestamp window under a fixed clock without substituting now. | Before/at/after launch and now+24h; malformed/missing | Only inclusive valid values survive | `instagram-importer/src/domain/timestamps.test.ts` |
| UT-005 | FR-013, FR-014 | AC-008 | Count graphemes, truncate exactly, repair only reversible UTF-8/Latin-1 mojibake, and preserve ambiguous text. | Unicode boundary and mojibake vectors | Deterministic visible final text and warnings | `instagram-importer/src/text/normalizeCaption.test.ts` |
| UT-006 | FR-015 | AC-009 | Generate hashtag/link facets using UTF-8 byte offsets and leave Instagram handles plain. | Unicode, URL, hashtag, and @handle vectors | Valid byte ranges; no mention facets | `instagram-importer/src/text/facets.test.ts` |
| UT-007 | FR-016, RULE-005, RULE-010 | AC-010, AC-016, AC-023 | Classify image signatures/MIME, dimensions, pixels, ratio, animation, format, and output constraints. | JPEG/PNG/WebP and spoofed/corrupt/HEIC/AVIF/GIF vectors | Supported candidates or explicit omission reason; never convert unsupported formats | `instagram-importer/src/media/policy.test.ts` |
| UT-008 | FR-008, FR-024, RULE-007 | AC-014, AC-019 | Derive stable distinct valid TIDs and detect canonical identity/rkey collisions. | Golden normalized-post vectors including same timestamps | Stable keys across order/reload/version; collisions remain skipped | `instagram-importer/src/domain/recordKey.test.ts` |
| UT-009 | FR-018, FR-019, FR-024, NFR-006, RULE-002 | AC-011, AC-019 | Build and validate the create-only post wire record and minimal provenance. | Valid/invalid records, unknown bounded source, and generated contract | Instagram has only the source token; an unknown bounded source remains Lexicon-valid but unclassified; invalid shape/key fails | `instagram-importer/src/pds/postRecord.test.ts` |
| UT-010 | FR-025, FR-026, FR-027, RULE-006 | AC-014, AC-015 | Reduce progress across DID/fingerprint binding, owned create/CID, existing-unowned, collision, ambiguous create result, failure, deletion, retry, newer session, cancel, CID-conflict rollback, sign-out, and clear. | State-transition table | Only durably CID-owned creates become rollback targets | `instagram-importer/src/progress/state.test.ts` |
| UT-011 | FR-028 | AC-014, AC-015 | Apply bounded transient retry/backoff and classify auth/rate/permanent failures. | Fake clock and safe PDS error codes | Bounded schedule, pause when required, no successful-item restart | `instagram-importer/src/pds/retry.test.ts` |
| UT-012 | FR-009, FR-025, NFR-005 | AC-002, AC-017 | Redact content and identifiers from safe errors, importer persistence DTOs, logs, URLs, and stringification while keeping OAuth-library state encapsulated. | Sentinel canaries in every sensitive field and OAuth adapter boundary | Only bounded safe codes/counts remain outside the OAuth library | `instagram-importer/src/privacy/safeDiagnostics.test.ts` |
| UT-013 | FR-002, FR-003, NFR-004 | AC-003, AC-018 | Build origin-specific OAuth client metadata/scopes, canonicalize `localhost` to the equivalent loopback IP before state creation, and fail closed on previews/incompatible grants. | Runtime-mode, loopback-URL, and permission matrices | Exact least authority; RFC 8252-compatible local callback; real clients only on allowed origins | `instagram-importer/src/config/runtime.test.ts`, `instagram-importer/src/auth/oauthConfig.test.ts` |
| UT-014 | FR-020–FR-022 | AC-012, AC-013 | Classify indexed post provenance and original-backfill activity without affecting later interaction types. | Imported/ordinary post records and activity kinds | Only original imported authored activity is suppressed | `appview/internal/index/craftsky_post_import_test.go` |
| UT-015 | FR-023 | AC-011 | Decode optional provenance for full posts and quote previews and select localized label copy. | Missing/Instagram/unknown source DTOs | Instagram label only for supported import source in both response shapes | `app/test/feed/models/post_test.dart`, `app/test/l10n/imported_post_l10n_test.dart` |
| UT-016 | FR-010–FR-012 | AC-006, AC-007 | Derive default selection, filters, counts, and warnings for large manifests. | Mixed manifest transitions | Exact stable counts and all eligible selected by default | `instagram-importer/src/review/reviewState.test.ts` |
| UT-017 | FR-025 | AC-014, AC-017 | Derive versioned manifest fingerprints without persisting content. | Equivalent and changed synthetic manifests | Equivalent input matches; material change differs; persistence excludes source values | `instagram-importer/src/progress/fingerprint.test.ts` |
| UT-018 | FR-006, NFR-003 | AC-005, AC-016 | Validate typed worker request/result/cancel protocol and stale-operation fencing. | Operation IDs and cancel races | Stale/cancelled results cannot mutate active review | `instagram-importer/src/worker/protocol.test.ts` |
| UT-019 | FR-010, FR-017, RULE-010 | AC-006, AC-010, AC-023 | Decide review eligibility after media omissions and require explicit text-only confirmation. | Mixed valid/missing/unsupported media and captions | Remaining images stay selected; text-only confirmation; empty skip; no alt UI | `instagram-importer/src/review/mediaSelection.test.ts` |
| UT-020 | FR-010 | AC-006 | Load expanded selected thumbnails, release them when collapsed or deselected, and open/close the selected image lightbox. | Selected/deselected multi-image posts and mocked sanitized blobs | No eager load while collapsed; expanded thumbnails appear; abort/revoke on close; accessible modal opens and closes | `instagram-importer/src/app/components/ReviewList.test.tsx` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-018, FR-019, NFR-006 | AC-011, AC-019 | Cross-language Lexicon and ADR contract. | Load post JSON schema and generated Go/web shapes | Validate Instagram, unknown bounded, and invalid provenance and run generation check | Optional minimal Instagram and unknown source validate; only Instagram is recognized; invalid shapes/drift fail; ADR exists | `appview/internal/lexicon/craftsky/feedpost_import_test.go`, `instagram-importer/src/generated/contract.test.ts` |
| IT-002 | FR-005, FR-006, FR-007, FR-008, NFR-001, NFR-002, NFR-003, RULE-009 | AC-005, AC-016, AC-019, AC-022 | Validate EOCD/ZIP64 bounds before zip.js allocation, then stream discovery and metadata extraction. | Synthetic ordinary/ZIP64/sparse/wrapped/encrypted/malformed/limit archives | Parse/cancel each in a real browser worker | Deterministic manifest or safe local failure without full-file/directory over-allocation | `instagram-importer/e2e/archive-worker.spec.ts` |
| IT-003 | BR-002, FR-009, NFR-001, NFR-005, RULE-008 | AC-002, AC-017 | Enforce canary minimization across a real browser worker boundary and app storage. | Full-information synthetic archive with unique canaries | Parse, review, fail, stringify, and inspect IndexedDB | Only approved normalized fields cross boundary; canaries absent | `instagram-importer/e2e/archive-privacy.spec.ts` |
| IT-004 | FR-016, FR-017, NFR-002, NFR-003, RULE-005, RULE-010 | AC-010, AC-016, AC-023 | Header-inspect, preflight, decode, sanitize, resize/re-encode, transfer, and release media incrementally in a browser worker. | Synthetic metadata-bearing, animated, decode-bomb-header, and boundary images | Process selected entries with worker instrumentation | Limits reject before decode; supported safe blobs only; EXIF absent; one active decode; transferred buffers/revoked URLs; bounded cancel | `instagram-importer/e2e/media-worker.spec.ts` |
| IT-005 | FR-025, FR-026, FR-028 | AC-014, AC-017 | Persist content-free import history, returned owned-create CIDs, isolate OAuth-library state, and clear each through its owner. | Fake importer and OAuth-library IndexedDB stores plus session transitions | Save/reload/sign-out/reselect/clear | Resume/ownership metadata survives; importer storage contains only approved fields; CIDs never enter diagnostics; OAuth state is removed on sign-out | `instagram-importer/src/progress/repository.integration.test.ts` |
| IT-006 | FR-002–FR-004, RULE-001 | AC-003, AC-004 | Browser OAuth adapter and profile self-record preflight. | Fake standards-compatible OAuth/PDS agents | Authorize, restore, deny, return incompatible grant, read profile/self | Correct DID/session binding; fail closed before writes | `instagram-importer/src/auth/authService.integration.test.ts` |
| IT-007 | FR-017, FR-018, FR-024, FR-026, FR-027, FR-028, RULE-006, RULE-007 | AC-001, AC-010, AC-014, AC-015 | PDS publisher create/delete/retry and ownership semantics. | Fake XRPC agent with blob/create/get/delete/rate-limit, lost-create-result, replacement, and CID-different delete/recreate scripts | Publish, resume, collide, retry, and CID-guarded rollback | Upload-before-create, create-only, ambiguous existing-unowned, `swapRecord` deletes only current-CID matches, content-CID incarnation limitation disclosed, safe per-item results | `instagram-importer/src/pds/publisher.integration.test.ts`, `instagram-importer/src/app/App.test.tsx` |
| IT-008 | FR-001, FR-002, FR-004, FR-010, RULE-003 | AC-001, AC-003, AC-004, AC-006 | Complete mocked browser flow. | Playwright, synthetic ZIP, mocked OAuth/PDS | Acknowledge, parse, review/edit, authorize, preflight, confirm, import | Accessible complete flow and exact created records/counts | `instagram-importer/e2e/import-flow.spec.ts` |
| IT-009 | FR-020, NFR-008 | AC-011, AC-012, AC-021 | Reversible post-provenance/profile-order migration. | Isolated real Postgres at migration 000024 | Up/down/up 000025 and inspect schema/indexes | Existing rows default ordinary; new fields/indexes reversible | `appview/internal/db/instagram_import_migration_test.go` |
| IT-010 | FR-018, FR-019, FR-020, FR-023, NFR-006, NFR-008 | AC-011, AC-019, AC-021 | Index and return provenance through canonical and quoted-post responses. | Member and imported/ordinary Tap records plus retaining/omitting replacement records in real Postgres | Create/update/delete and hydrate all post-shaped responses | Provenance follows the authoritative record, survives a retaining replacement, clears after an omitting replacement, raw record is preserved, and ordinary behavior is unchanged | `appview/internal/index/craftsky_post_import_test.go`, `appview/internal/api/post_response_test.go` |
| IT-011 | FR-020, FR-021, NFR-008, RULE-004 | AC-012, AC-021 | Profile chronology and pagination. | Imported/ordinary posts with crossed created/indexed times | Page author posts through ties and cursors | Imported profile order follows original createdAt; ordinary behavior and no gaps/dupes | `appview/internal/api/imported_post_store_test.go` |
| IT-012 | FR-021, NFR-008, RULE-004 | AC-012, AC-021 | Home-timeline original exclusion and later share inclusion. | Author/follower/import/ordinary/repost/quote rows | Page timelines | Original imports absent; later repost/quote and all ordinary controls present normally | `appview/internal/api/timeline_store_test.go` |
| IT-013 | FR-015, FR-022, NFR-008, RULE-004 | AC-009, AC-013, AC-021 | Suppress original import notification activation only. | Imported record with valid mention facets, reply field, quote embed, and later interactions | Index source and later events | No source activation/materialized mention notification; later/ordinary events normal | `appview/internal/index/notification_post_test.go`, `notification_ingestion_test.go` |
| IT-014 | FR-020, NFR-008, RULE-004 | AC-012, AC-021 | Search imported text and hashtags with original chronology. | Search fixtures with imported and ordinary posts | Query keyword/tag and paginate | Both eligible; ranking/date behavior matches contract; ordinary results unchanged | `appview/internal/api/search_store_test.go`, `search_ranking_test.go` |
| IT-015 | FR-023 | AC-011 | Flutter optional provenance contract and shared card label. | Imported/ordinary full-post and quote-preview JSON fixtures plus localized widget harness | Decode and render profile/search/direct/quote-preview contexts | Imported label visible/accessibly named in both shapes; ordinary label absent | `app/test/feed/models/post_test.dart`, `app/test/feed/widgets/post_card_test.dart` |
| IT-016 | FR-001, FR-003, NFR-004 | AC-018, AC-020 | Static build, metadata, dependency, CSP, and preview policy. | Build each supported runtime mode | Inspect emitted assets/headers/metadata and execute preview | Self-hosted restricted output; preview cannot real-authorize/write | `instagram-importer/src/config/build.integration.test.ts`, `instagram-importer/e2e/origin-policy.spec.ts` |
| IT-017 | BR-002, FR-009, FR-017, NFR-001, NFR-005 | AC-002, AC-017 | Browser network/storage privacy capture. | Playwright canary archive, service-worker/network observer, IndexedDB inspector | Run parse through rollback with injected failures | Only OAuth/PDS/resolver requests and confirmed public bytes; no content diagnostics | `instagram-importer/e2e/privacy-boundary.spec.ts` |
| IT-018 | FR-024–FR-028, RULE-006, RULE-007 | AC-014, AC-015 | Browser reload, reselect, retry, later-session, clear, and rollback flow. | Persistent Playwright context and scripted fake PDS | Interrupt at every durable boundary | Exact resume/ownership semantics and scoped delete set | `instagram-importer/e2e/recovery.spec.ts` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing post records validate, generate, index, and serialize without provenance. | NFR-006, NFR-008 | AC-011, AC-019, AC-021 | Existing Lexicon/generated/index/response suites remain green with optional provenance absent. |
| REG-002 | Ordinary profile posts retain current ordering and opaque pagination. | NFR-008 | AC-012, AC-021 | Existing ordinary-only profile pagination vectors return identical order/cursors. |
| REG-003 | Ordinary authored posts and normal reposts/quotes appear in home timelines. | BR-003, FR-021, NFR-008 | AC-012, AC-021 | Existing timeline suite plus ordinary controls remains green. |
| REG-004 | Ordinary post/reply/quote/mention and later interaction notifications activate normally. | BR-003, FR-022, NFR-008 | AC-013, AC-021 | Existing notification lifecycle/ingestion suites remain green. |
| REG-005 | Ordinary keyword/hashtag search ranking and pagination are unchanged. | FR-020, NFR-008 | AC-012, AC-021 | Existing search store/ranking/recent suites remain green. |
| REG-006 | Post deletion, moderation, relationship filtering, saved posts, and interaction counts are unchanged. | NFR-008 | AC-021 | Existing index/API lifecycle and moderation suites remain green. |
| REG-007 | Ordinary Flutter post cards and quote previews show no extra label and retain layout/actions. | FR-023, NFR-008 | AC-011, AC-021 | Existing `post_card_test.dart` cases remain green with provenance absent. |
| REG-008 | Flutter contains no archive parser/import route and the existing static marketing site remains independent. | FR-001 | AC-001, AC-020 | Source/dependency scan and existing Flutter/web tests prove the importer stays in its own package. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Minimal supported export | Synthetic wrapped/unwrapped `posts_1.json` plus tiny JPEG | AT-001, IT-002, IT-008 |
| TD-002 | Full-export privacy canaries | Synthetic post, message, relationship, profile, ad, unrelated media and unique sentinel values | AT-002, IT-003, IT-017 |
| TD-003 | Path/ZIP attacks | Traversal, absolute, duplicate, case conflict, confusable, encrypted, malformed, unsupported compression, ZIP64 | UT-002, IT-002 |
| TD-004 | Duplicate adapters | Equivalent and materially conflicting supported JSON representations, including current media-caption/media-timestamp precedence and observed date-partitioned/Other media paths | UT-003, IT-002 |
| TD-005 | Text vectors | 2,000/2,001 graphemes, emoji clusters, reversible mojibake, ambiguous text, URLs/tags/@handles | UT-005, UT-006, AT-004 |
| TD-006 | Timestamp/rkey goldens | Launch/now boundaries, malformed, same-timestamp ordering, canonical identities and collisions | UT-004, UT-008 |
| TD-007 | Media policy corpus | Synthetic JPEG/PNG/WebP, metadata, spoofed MIME, corrupt, missing, dimensions/pixels/ratio boundaries, HEIC/AVIF/GIF/animated | UT-007, IT-004 |
| TD-008 | Large review manifest | 25,000 synthetic mixed-state posts with no real content | AT-004, UT-016 |
| TD-009 | OAuth/PDS scripts | Compatible/denied/cancelled/incompatible grants, missing profile, rate limit, expiry, collision, absent delete | AT-003, IT-006, IT-007 |
| TD-010 | Progress transition journal | Synthetic DIDs/fingerprints/URIs/CIDs and owned/unowned/conflict states represented only inside isolated test code and never snapshots | UT-010, IT-005, IT-018 |
| TD-011 | Lexicon records | Minimal valid ordinary/import records and invalid provenance/source-detail records | UT-009, IT-001, IT-010 |
| TD-012 | AppView chronology | Imported/ordinary posts with crossed created/indexed times and tie values | IT-011, IT-012, IT-014 |
| TD-013 | Notification graph | Imported original with facets plus later like/repost/quote/reply and ordinary controls | IT-013 |
| TD-014 | Runtime modes | Localhost, canonical production, stable staging, ephemeral preview configs | UT-013, IT-016, AT-010 |
| TD-015 | Flutter fixtures | Optional Instagram provenance, absent provenance, and unknown future source | UT-015, IT-015, REG-007 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | FR-002, FR-003, FR-004, NFR-004, NFR-007 | AC-003, AC-004, AC-018, AC-020 | Real browser OAuth and scoped PDS operations | Use localhost, production, and optional stable staging against compatible test accounts/PDSs | Exact authority is granted, profile self-record is read, create/delete works, incompatible PDS fails before writes |
| MAN-002 | FR-005, FR-006, FR-007, FR-008, NFR-007 | AC-005, AC-020 | Current consented export/browser compatibility | Observe additional current exports structurally, recreate shapes synthetically, and run Chrome/Edge/Firefox/Safari | Supported shapes import or report a safe compatibility error; no raw fixture is committed |
| MAN-003 | FR-005, FR-006, FR-028, NFR-001, NFR-002, NFR-003, NFR-007 | AC-005, AC-014, AC-016, AC-020 | Large ZIP64, suspension, memory, and rate limiting | Use generated sparse/large archives and test browser suspend/reload/rate limiting | UI stays responsive, resume is accurate, and resource use stays bounded |
| MAN-004 | FR-020, FR-021, FR-022, FR-023, NFR-008 | AC-011, AC-012, AC-013, AC-021 | Live firehose/AppView convergence | Import synthetic records to a test PDS and inspect profile/search/timeline/notifications/Flutter | Imported originals are profile/search-only with label; later engagement behaves normally |
| MAN-005 | NFR-004 | AC-018 | Deployed Cloudflare isolation | Inspect production/staging headers, CSP, network requests, assets, metadata, and Pages rollback | Dedicated origin/project is restrictive, self-hosted, analytics-free, and recoverable |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Additional Instagram export shapes | FR-007, FR-008, NFR-007 | Instagram's format is undocumented and one consented sample cannot establish all variants. | Observe additional consented exports, retain structures only, and add synthetic adapters/fixtures before release. |
| GAP-002 | Live OAuth scope interoperability | FR-002, FR-003, NFR-004, NFR-007 | Repository mocks cannot prove every PDS implements current granular permissions identically. | Run MAN-001 against production-target PDS implementations before enabling live production OAuth. |
| GAP-003 | Browser/PDS/firehose end-to-end convergence | FR-020–FR-023, NFR-007 | Relay timing and deployed-origin behavior are external and non-deterministic. | Complete MAN-004 on a dedicated test account and record release evidence. |
| GAP-004 | Cloudflare Pages project configuration | NFR-004 | Repository files can specify headers/builds but cannot prove dashboard isolation or rollback. | Complete MAN-005 during deployment setup. |

## 10. Out Of Scope

- Instagram follower/following, stories, messages, comments, likes, profile,
  contacts, searches, and advertising-data import tests.
- Video/audio conversion, still generation, HEIC/AVIF/GIF conversion, alt-text
  generation/editing, Instagram ownership verification, AT-handle mapping, and
  project-post inference.
- Importer backend, archive upload/proxy, server-side job, Flutter archive
  parsing, `app.bsky.feed.post` publication, onboarding, or new AppView importer
  endpoints.
- Guaranteeing undocumented future Instagram export compatibility.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- Recommended first failing test for implementation: `IT-001`, asserting the
  optional minimal Instagram provenance on the generated post contract after
  ADR creation and before the Lexicon/codegen edit.
- Suggested implementation order:
  1. `IT-001`, `UT-009`, `IT-009`, `UT-014`, `IT-010`, `IT-011`,
     `IT-012`, `IT-013`, `IT-014`, `UT-015`, `IT-015`, `REG-001`,
     `REG-002`, `REG-003`, `REG-004`, `REG-005`, `REG-006`, `REG-007`.
  2. Importer scaffold/config: `UT-001`, `UT-013`, `IT-016`, `REG-008`.
  3. Pure local domain: `UT-002`, `UT-003`, `UT-004`, `UT-005`,
     `UT-006`, `UT-007`, `UT-008`, `UT-016`, `UT-017`, `UT-018`,
     `UT-019`.
  4. Worker boundaries: `IT-002`, `IT-003`, `IT-004`.
  5. Progress/auth/publication adapters: `UT-010`, `UT-011`, `UT-012`,
     `IT-005`, `IT-006`, `IT-007`.
  6. Browser flows: `AT-001`, `AT-002`, `AT-003`, `AT-004`, `AT-005`,
     `AT-006`, `AT-007`, `AT-008`, `AT-009`, `AT-010`, `IT-008`,
     `IT-017`, `IT-018`.
- Commands discovered:
  - `just lexgen`
  - `just lexgen-check`
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...`
  - `just app-test`
  - Importer `test`, `typecheck`, `lint`, `build`, and `test:e2e` scripts will
    be owned by its new package.
- Blocking gaps: None for implementation. `GAP-001` through `GAP-004` remain
  release gates.
