# Project craft schema acceptance test specification

## 1. Test strategy

Use the [reviewed requirements](01-requirements.md) as the product contract. Phase 1 covers the existing crafts and must pass before the six Phase 2 primary crafts are available. Phase 2 adds their authoring, profile, onboarding and discovery surfaces and reruns the Phase 1 coverage.

Use Flutter widget tests and fixture-backed Svelte Playwright tests for visible workflows. Use Dart and TypeScript unit tests for draft, form and patch behaviour. AppView command and projection integration tests use the existing fake PDS boundary and real PostgreSQL where persistence, filters or visibility matter. Scheduled publication tests use the current payload and worker. Browser and widget fixtures verify client behaviour, while AppView tests verify handling of the corresponding requests. Use manual rollout checks for actual older builds and deployment order.

Exercise every new craft's detail and discovery contract. Use representative size lists, grouped selections and material associations across the publication, draft, schedule, edit and clear paths. Reuse the same expected catalog outcomes across client tests and keep search assertions on AppView results.

## 2. Requirement coverage matrix

| Requirement | Acceptance criteria | Test IDs | Levels | Automation |
|---|---|---|---|---|
| BR-001 | AC-004, AC-006, AC-007, AC-008 | AT-003, AT-004, AT-005, AT-006 | Acceptance | Automated |
| BR-002 | AC-002, AC-003 | AT-002, UT-001, REG-001 | Acceptance, unit, regression | Automated |
| FR-001 | AC-001 | AT-001, UT-002 | Acceptance, unit | Automated |
| FR-002 | AC-002 | AT-002, UT-005, IT-001 | Acceptance, unit, integration | Automated |
| FR-003 | AC-003, AC-010 | AT-008, UT-001, REG-001 | Acceptance, unit, regression | Automated |
| FR-004 | AC-004, AC-005 | AT-003, UT-003, IT-003 | Acceptance, unit, integration | Automated |
| FR-005 | AC-006 | AT-004, UT-003, IT-003, REG-002 | Acceptance, unit, integration, regression | Automated |
| FR-006 | AC-007, AC-011 | AT-005, IT-001, IT-003, IT-006 | Acceptance, integration | Automated |
| FR-007 | AC-008 | AT-006, IT-005, REG-002 | Acceptance, integration, regression | Automated |
| FR-008 | AC-009 | IT-005, REG-003 | Integration, regression | Automated |
| FR-009 | AC-002, AC-010 | AT-002, AT-008, IT-001, IT-003, IT-004 | Acceptance, integration | Automated |
| FR-010 | AC-015 | IT-002, MAN-001 | Integration, rollout | Automated and manual |
| FR-011 | AC-016 | AT-007, UT-004, IT-007 | Acceptance, unit, integration | Automated |
| NFR-001 | AC-012 | UT-002, IT-005 | Unit, integration | Automated |
| NFR-002 | AC-010, AC-013 | AT-008, UT-005, IT-001, IT-004, REG-002, MAN-002 | Acceptance, unit, integration, regression, rollout | Automated and manual |
| NFR-003 | AC-017 | REG-004 | Regression | Automated |
| RULE-001 | AC-011 | IT-006 | Integration | Automated |
| RULE-002 | AC-001, AC-003, AC-006, AC-013 | AT-001, AT-004, UT-001, UT-005, IT-003, REG-001, REG-002 | Acceptance, unit, integration, regression | Automated |
| RULE-003 | AC-014 | IT-003, IT-005 | Integration | Automated |
| RULE-004 | AC-018 | AT-003, UT-002, REG-002 | Acceptance, unit, regression | Automated |

## 3. Acceptance scenarios

All scenarios are Must. Use the named existing test suites as extension points. AppView assertions about actual public writes belong to §5.

### AT-001 — Design suggestions follow the primary craft

**Requirements** FR-001, RULE-002 · **Criteria** AC-001 · **Phase** 1

**Automation** Extend `app/test/projects/widgets/project_composer_metadata_test.dart`, `app/test/projects/pages/projects_page_test.dart`, and web's `authenticated-project-detail-parity.spec.ts` and `authenticated-filter-detail-parity.spec.ts`.

```gherkin
Scenario: Keep a saved tag when the primary craft changes
  Given a Sewing draft with the shared tags Plaid and Floral selected
  When the author changes its primary craft to Crochet
  Then Plaid remains selected and removable and is absent from new suggestions
  And Floral remains selected with its existing stored value

Scenario Outline: Offer craft-specific tag choices in the editor and filter
  Given the <craft> craft is selected
  When the author opens Design and a reader opens the craft's Design filter
  Then both surfaces offer <tag>
  Examples:
    | craft      | tag                  |
    | Quilting   | Book/Film Characters |
    | Embroidery | Quotes               |
```

### AT-002 — Select and publish multiple tool sizes

**Requirements** BR-002, FR-002, FR-009 · **Criteria** AC-002 · **Phase** 1

**Automation** Extend `app/test/projects/widgets/project_composer_details_test.dart`, `project_composer_submit_test.dart` and `project_card_test.dart` in the same directory, and web's `authenticated-project-detail-parity.spec.ts`.

```gherkin
Scenario Outline: Publish multiple sizes from a searchable picker
  Given an author is composing a <craft> project
  When they search for and select two distinct sizes, then publish
  Then the request carries both canonical strings in <field>
  And the published project details and reopened editor show both sizes
  Examples:
    | craft    | field         |
    | Knitting | needleSizesMm |
    | Crochet  | hookSizesMm   |
```

### AT-003 — Techniques remain independent of item types and design tags

**Requirements** BR-001, FR-004, RULE-004 · **Criteria** AC-004, AC-005, AC-018 · **Phase** 1

**Automation** Extend Flutter's `project_composer_details_test.dart` and `project_card_test.dart`, and web's `authenticated-project-detail-parity.spec.ts`.

```gherkin
Scenario: Edit the subtypes of one selected knitting technique
  Given a Knitting project with Cables and Colourwork selected
  When the author selects Intarsia and Stranded under Colourwork and reopens the project
  Then both subtypes appear under Colourwork and Cables remains selected
  When they remove Intarsia, then clear Stranded, then remove Colourwork
  Then the successive edits retain Stranded, then Colourwork alone, then Cables alone

Scenario: Describe a crochet blanket and find a technique by its alias
  Given a Crochet project classified as a blanket
  When the author adds Granny square and searches the Techniques picker for Afghan crochet
  Then the item classification stays Blanket and Tunisian crochet is offered
  And the picker adds catalog choices and offers no generic Other option
  And Amigurumi is absent from the Techniques picker

Scenario Outline: Keep existing Crochet item subtypes selectable and readable
  Given a Crochet project with the applicable parent item type selected
  When the author selects <subtype> as the item subtype and publishes
  Then the published details and reopened editor show <subtype> as the item subtype
  Examples:
    | subtype       |
    | Granny square |
    | Amigurumi     |

Scenario: Describe construction and appearance in Sewing
  When the author selects French seams, Invisible zip insertion and Print matching across the technique groups
  And they select Plaid in Design and publish
  Then the project shows the three techniques and Plaid as a separate design tag
```

### AT-004 — Embroidery styles stay with their own method

**Requirements** BR-001, FR-005, RULE-002 · **Criteria** AC-006 · **Phase** 1

**Automation** Extend Flutter's `project_composer_details_test.dart` and `project_card_test.dart`, and web's `authenticated-project-detail-parity.spec.ts`.

```gherkin
Scenario: Save and revise a project made by hand and machine
  Given an Embroidery project with Hand and Machine selected
  When the author adds Sashiko under Hand and 3D puff under Machine and reopens the project
  Then each style appears under its method
  And Single-needle and Multi-needle are absent from Machine Styles
  When the author removes Hand and saves
  Then Machine and 3D puff remain and Sashiko is removed with Hand
```

### AT-005 — The six new crafts can publish with and without details

**Requirements** BR-001, FR-006 · **Criteria** AC-007 · **Phase** 2

**Automation** Extend Flutter's `project_composer_details_test.dart`, `project_composer_submit_test.dart` and `project_card_test.dart`, and web's `authenticated-project-detail-parity.spec.ts`. Check the field inventory in both editors using §11 of the requirements. Use TD-006 for representative detail round trips.

```gherkin
Scenario Outline: Publish shared fields and then save specialist details
  Given the author selects <craft> as the primary craft
  When they publish without specialist details
  Then the project is published and displays its shared fields
  When they add the <sample> details using the existing editor and save
  Then the editor reopens with those values and the project displays them
  Examples:
    | craft        | sample                                       |
    | Spinning     | wheel-spun two-ply yarn and yield            |
    | Dyeing       | spun yarn or unspun fleece and dye method    |
    | Punch Needle | rug, foundation fabric and loop height      |
    | Macramé      | two cords with different diameters and knots |
    | Weaving      | distinct warp and weft and sett with unit    |
    | Felting      | Needle method, sculpture and two gauges      |
```

For Macramé and Weaving, assert each diameter remains with its cord and warp and weft remain distinguishable after reopening. For Dyeing and Punch Needle, verify equivalent fibre or yarn information can be entered once in shared Materials and is visible in the project.

### AT-006 — Discover Machine Knitting and Needle Felting within their crafts

**Requirements** BR-001, FR-007 · **Criteria** AC-008 · **Phases** 1 for Machine, 2 for Needle

**Automation** Extend Flutter's `project_composer_details_test.dart`, `project_card_test.dart` and `projects_page_test.dart`, and web's `authenticated-project-detail-parity.spec.ts` and `authenticated-filter-detail-parity.spec.ts`.

```gherkin
Scenario Outline: Save and find a cardigan with machine knitting details
  Given the author is editing a Knitting project with the Cardigan item subtype
  When the author checks Machine and sets Hand to <handSelection> in Knitting methods
  Then the editor exposes the machine fields
  When the author enters 4.5 mm machine needle spacing and a measured stitch gauge of 24 stitches and 32 rows per 10 cm, saves and reopens the project
  Then the saved methods list contains exactly <methods>
  And the editor and published details retain those selections, both distinct gauges and the Cardigan subtype
  And the project appears when a reader filters Knitting by Machine
  Examples:
    | handSelection | methods          |
    | unchecked     | Machine          |
    | checked       | Hand and Machine |

Scenario: Clear the last selected Knitting method
  Given a Knitting cardigan with Hand selected
  When the author unchecks Hand, saves and reopens the project
  Then neither method is selected and the cardigan's method is unspecified

Scenario: Find a needle-felted sculpture through the Felting filter
  Given a Felting sculpture made with the Needle method
  When the reader filters Felting by Needle
  Then the result shows Felting as its craft and retains its method and sculptural form
```

### AT-007 — New craft choices reach members across the existing surfaces

**Requirements** FR-011 · **Criteria** AC-016 · **Phase** 2

**Automation** Extend `app/test/onboarding/onboarding_crafts_step_test.dart`, `app/test/profile/widgets/edit_profile_crafts_picker_test.dart`, Flutter's existing Home and search coverage, and web's `authenticated-input-parity.spec.ts`, `authenticated-navigation-parity.spec.ts` and `authenticated-search.spec.ts`.

```gherkin
Scenario: Add, save and use a new craft interest
  Given a new member is on the craft step and an existing member can edit interests
  When they choose a new primary craft and save their choices
  Then reopening preferences shows the choice and Home prioritises it
  When the existing member removes that interest and saves
  Then the updated interests persist and that craft feed remains reachable

Scenario: Reach all craft feeds regardless of interests
  Given a viewer with a subset of saved interests and fixtures with tags for every craft
  When the viewer opens Home and grouped hashtag discovery
  Then every one of the eleven primary crafts is reachable and has its eligible hashtag group
  And Machine Knitting and Needle Felting are reached through their parent crafts
```

Check the full six-craft inventory at each relevant surface. A representative choice is sufficient for the interaction sequence when catalog coverage verifies the rest.

### AT-008 — Revised project values survive draft and current schedule flows

**Requirements** FR-003, FR-009, NFR-002 · **Criteria** AC-010 · **Phases** 1 and 2

**Automation** Extend Flutter's `app/test/projects/composer/project_draft_snapshot_adapter_test.dart`, its project composer widget suites and `app/test/scheduled_posts/scheduled_composer_submission_test.dart`. Extend web's `authenticated-project-detail-parity.spec.ts` for drafts and `authenticated-app.spec.ts` for schedules. Exercise worker publication in IT-004.

```gherkin
Scenario: Reopen a local draft saved with a scalar size
  Given a local draft containing only a valid legacy needle or hook size
  When the author opens it in the updated editor
  Then the size appears as one selection and other draft values are retained

Scenario: Save, reopen and publish revised metadata
  Given a Knitting project with a needle size list and Colourwork subtypes
  When the author saves and reopens a local draft or a new schedule in the current flow
  Then the editor shows the saved choices and their relationships
  And publication retains those choices on the resulting project
```

Use one Phase 2 material-association payload as a second shape across this suite and IT-004. Verify URI preservation and conflict handling through IT-003, and publication retry through IT-004.

## 4. Unit test cases

| ID | Requirements | Criteria | Inputs and expected result | Automation target |
|---|---|---|---|---|
| UT-001 | BR-002, FR-003, RULE-002 | AC-003 | For each tool, test absent fields, a valid scalar including one outside the catalog, a list plus conflicting scalar, and an empty list plus scalar. Resolve respectively to no size, a singleton, the given list, and no size. Explicit clear stays clear after a round trip. | Flutter `project_composer_hydrator_test.dart` and `project_draft_snapshot_adapter_test.dart`, web `src/lib/features/composer/model.test.ts` |
| UT-002 | FR-001, NFR-001, RULE-004 | AC-001, AC-012, AC-018 | Check the design suggestion sets in requirements §11 and representative techniques, aliases and groupings in equivalent Flutter and web contexts. Check the canonical stored IDs and the new-craft design fallback. Saved valid unfamiliar values can be shown and removed. New values come from the catalog. | Flutter `app/test/projects/options/project_option_catalogs_test.dart`, web `composer/model.test.ts` and `social/filter-model.test.ts`, AppView `internal/projectcatalog/catalog_test.go` for catalog facts |
| UT-003 | FR-004, FR-005 | AC-004, AC-006 | Serialize and hydrate Cables plus Colourwork with two subtypes and both Embroidery method groups. Removing one child or parent leaves unrelated groups and their children intact. Verify explicit clear versus omitted change when constructing an edit. | Flutter `project_composer_payload_test.dart` and `app/test/feed/composer/post_edit_changes_test.dart`, web `composer/model.test.ts` |
| UT-004 | FR-011 | AC-016 | Assert all six new primary crafts use their approved labels and profile identifiers in onboarding, preferences and discovery inputs. Existing preferences still order Home choices first. Machine and Needle belong to the parent crafts. | Flutter `onboarding_profile_payload_test.dart`, `preferred_craft_links_test.dart` and `blank_search_provider_test.dart`, web existing social catalog and navigation test coverage |
| UT-005 | FR-002, NFR-002, RULE-002 | AC-002, AC-013 | Validate representative legacy, canonical and unknown-but-valid records plus malformed shapes and values outside the approved size and field limits. Keep existing valid bounded size strings as list entries. Verify new object shapes and optional fields survive generated JSON and CBOR round trips. | AppView `internal/lexicon/schema/catalog_test.go`, `internal/lexicon/craftsky/open_union_test.go`, `internal/api/post_request_test.go`, and `just lexgen-check` |

## 5. Integration test cases

| ID | Requirements | Criteria | Setup, action and expected result | Automation target |
|---|---|---|---|---|
| IT-001 | FR-002, FR-006, FR-009, NFR-002 | AC-002, AC-007, AC-010, AC-013 | Create projects through AppView using canonical size lists and representative Phase 2 detail variants, including a new craft without details. Capture the fake PDS write and read the indexed project. Arrays persist as bounded metric strings under the plural field, specialist values and material associations are intact, and records without optional details remain readable. | `appview/internal/api/post_create_commands_test.go` and `internal/index/craftsky_post_test.go` |
| IT-002 | FR-010 | AC-015 | Cover both tool fields with legacy Flutter and web request shapes. Create with a scalar and assert the public record carries a singleton list. Fetch both the public project response and the private owner edit seed at `editable.project`. Each exposes that singleton as the legacy scalar, while both reads leave the authoritative record intact. Generate or replay legacy change and explicit clear patches from the returned editor seed and observed version. A replacement writes only the canonical list and clearing leaves no effective size. On a multi-size record, a legacy unrelated edit preserves the list, while explicit legacy scalar replacement and clear attempts use the existing save-error flow and leave the PDS record intact. See REG-001 for accepted updated-client changes to the same shape. | `appview/internal/api/post_create_commands_test.go`, `post_edit_commands_test.go` and `appview/internal/index/post_edit_http_integration_test.go` |
| IT-003 | FR-004, FR-005, FR-006, FR-009, RULE-002, RULE-003 | AC-004, AC-006, AC-007, AC-010, AC-013, AC-014 | Use the published owner edit flow with Phase 1 grouped Knitting and Embroidery selections and a Phase 2 Macramé variant. Remove Intarsia from Colourwork while retaining Stranded and Cables, and remove Hand and its styles while retaining Machine, its 3D puff style and an unknown field on that untouched group. In Phase 2, add Macramé details to a published shared-only project and update one cord's diameter while the other cord's details stay paired. Assert the selected changes in the actual AppView PDS write and indexed project read. Successful edits retain the URI and untouched extensions. Stale and unauthorized writes leave the existing record intact. An unpublished project detail value remains absent from public reads and search before publication. | `appview/internal/index/post_edit_http_integration_test.go`, `appview/internal/api/post_edit_commands_test.go` and `appview/internal/api/scheduled_privacy_test.go` |
| IT-004 | FR-009, NFR-002 | AC-010 | Create and update a new Knitting schedule carrying needle sizes from TD-002 and Colourwork subtypes from TD-004, plus a separate TD-006 material-association example. Reopen the schedules before running the worker. Verify the PDS post and read projection preserve entered values, and an injected retry via the existing publication harness results in one post. | `appview/internal/api/scheduled_post_request_test.go`, `scheduled_post_http_test.go`, `internal/scheduledposts/worker_acceptance_test.go` and `publication_test.go` |
| IT-005 | FR-007, FR-008, NFR-001, RULE-003 | AC-008, AC-009, AC-012, AC-014 | Index visible and hidden controls for scalar and plural sizes, independent design and technique uses, grouped styles, and craft methods. Search by sizes, a technique, a subtype or style, a method, a reviewed alias and a recognized parent. Verify matching visible results, shared Floral across crafts, and no result for a marker present only in an uncontracted extension. The Knitting Machine filter includes Machine-only and Hand-and-Machine projects and excludes Hand-only and unspecified projects. Filter Felting by Needle. Verify visibility in both keyword and filter paths. | `appview/internal/api/search_fields_test.go`, `search_store_test.go` and `search_improvements_privacy_test.go` |
| IT-006 | FR-006, RULE-001 | AC-011 | Publish a Spinning skein with Spinning details and a Knitting sock project with Spinning as an additional craft. Only the skein qualifies for the Spinning feed. The socks keep Knitting details and still participate in Spinning keyword and hashtag discovery. Submit a Knitting-primary, Spinning-additional command with the skein's recognized Spinning detail variant. AppView rejects the mismatch before any PDS write. Use the approved craft list to verify each Phase 2 ID can also be selected as an additional craft. | `appview/internal/api/search_fields_test.go`, `search_store_test.go`, `post_request_test.go` and `post_create_commands_test.go` |
| IT-007 | FR-011 | AC-016 | Submit and reload representative new interests through onboarding and profile editing. Assert saved identifiers and preference ordering. Verify grouped hashtag discovery and primary craft feed queries recognize each of the six IDs, including for a viewer who follows a different craft. | `appview/internal/pdscommands/onboarding_profile_integration_test.go`, `internal/api/profile_identity_crafts_test.go`, `search_fields_test.go` |

## 6. Regression tests

| ID | Requirements | Criteria | Existing behaviour and verification | Automation target |
|---|---|---|---|---|
| REG-001 | BR-002, FR-003, RULE-002 | AC-003 | Reopen scalar-only and mixed-representation records using the updated editor. An unrelated edit preserves the source values. Explicit size replacement or clear removes the stored scalar and a subsequent read shows no resurrected value. From the same multi-size input rejected for legacy scalar edits in IT-002, verify the updated client can explicitly replace the list and clear it. Read back the accepted replacement and the empty result from the stored public record. | Flutter project hydration and published edit suites, web `composer/model.test.ts`, AppView `post_edit_commands_test.go` |
| REG-002 | FR-005, FR-007, NFR-002, RULE-002, RULE-004 | AC-006, AC-008, AC-013, AC-018 | Old Embroidery without details and Knitting without a method open and edit without invented values. A bounded unfamiliar value remains visible and removable. Valid unknown fields and selections survive an unrelated save, and a public record retaining only old fields needs no rewrite. | Flutter and web project editor parity suites, AppView `post_edit_commands_test.go` and `internal/lexicon/craftsky/open_union_test.go` |
| REG-003 | FR-008 | AC-009 | An existing Granny square subtype still matches the subtype filter. A blanket with the same technique matches keyword search and retains its separate subtype. Keep established design and item type matches as controls. | AppView `search_fields_test.go` and `search_store_test.go` |
| REG-004 | NFR-003 | AC-017 | Exercise the Phase 1 scenarios against the Phase 1 candidate without depending on new craft editors or choices. On the Phase 2 candidate, rerun the Phase 1 suite and check all required Phase 2 entry points. Use the planned release candidates or equivalent checked-in configurations. | Existing AppView, Flutter and Svelte gates for each candidate |

## 7. Test data

| ID | Purpose | Fixture | Used by |
|---|---|---|---|
| TD-001 | Scoped suggestions and saved unknowns | Sewing Plaid and Floral, the workbook sets in requirements §11, Quilting Book/Film Characters, Embroidery Quotes, and one valid saved tag outside the catalog | AT-001, UT-002, REG-002, IT-005 |
| TD-002 | Size lists | Knitting `needleSizesMm` with `4.0mm` and `4.5mm`. Crochet `hookSizesMm` with `4.0mm` and `5.0mm`. Use equivalent option aliases already in each catalog. | AT-002, AT-008, UT-005, IT-001, IT-002, IT-004, IT-005 |
| TD-003 | Existing and conflicting size representations | Scalar-only value for each tool, an unfamiliar bounded scalar, list plus different scalar, empty list plus scalar and complete absence. Include an existing scalar local draft. | AT-008, UT-001, UT-005, IT-002, IT-005, REG-001 |
| TD-004 | Nested choices and distinct craft concepts | Knitting with Cables and Colourwork using Intarsia and Stranded, plus a Fair Isle search fixture. Crochet blanket with Granny square technique, standalone Granny square and Amigurumi subtype records, and Tunisian crochet with Afghan crochet alias. Sewing Plaid with French seams, Invisible zip insertion and Print matching. Embroidery Hand with Sashiko and Machine with 3D puff. | AT-003, AT-004, AT-008, UT-002, UT-003, IT-003, IT-004, IT-005, REG-003 |
| TD-005 | Methods and visibility | Cardigans with Machine only, Hand and Machine, Hand only, and unspecified methods. Include a Needle Felting sculpture in Phase 2 and a hidden matching project for each affected visibility path. Store machine spacing separately from measured stitch gauge. | AT-006, IT-003, IT-005, REG-002 |
| TD-006 | Phase 2 craft detail inventory | Fixtures for all six primary crafts with and without optional details. Include yarn and unspun Dyeing examples, 3 mm and 4 mm cords of different materials, warp and weft with distinct materials and a sett unit, and a Felting sample with multiple needle gauges. Use requirements §11 as the full inventory. | AT-005, AT-008, IT-001, IT-003, IT-004 |
| TD-007 | Craft discovery and preference | Spinning skein, Knitting socks with Spinning additional, and a mismatched Knitting command that copies the skein's Spinning details. Include each of the eleven primary crafts with a distinct eligible hashtag, a viewer with a subset of interests, and a new member with incomplete onboarding. | AT-007, UT-004, IT-006, IT-007 |
| TD-008 | Data integrity and bounds | Owner and unrelated viewer, stale edit version, a valid unknown field on the retained Machine group and a unique keyword marker in an unknown extension, malformed list member and size/array values at approved bounds and just over. | UT-005, IT-003, IT-005, REG-002 |

## 8. Manual checks

| ID | Requirements | Criteria | Check and expected result |
|---|---|---|---|
| MAN-001 | FR-010 | AC-015 | On the legacy mobile and web builds selected for rollout, publish a singleton and check it in both the public details and reopened owner editor. Change it, reopen, and clear it from the editor. Confirm an unrelated edit leaves a multi-size record intact. Use the compatible AppView in an authorized test environment and record build identities alongside IT-002 evidence. |
| MAN-002 | NFR-002 | AC-013 | Confirm the AppView release candidate accepts and reads legacy and new records in the rollout environment before enabling canonical writes in updated clients. Record the AppView version and the client versions or release artifacts used for the check. |

## 9. Test gaps and risks

| ID | Requirements | Criteria | Gap and follow-up |
|---|---|---|---|
| GAP-001 | FR-002, RULE-002 | AC-002, AC-013 | New collection limits beyond the existing 20-byte and 20-grapheme size-string limits are schema-design decisions. Use the final approved limits for the at-boundary and over-boundary cases in UT-005. |
| GAP-002 | FR-010, NFR-002 | AC-013, AC-015 | Automated fixtures cover known legacy payload shapes. Installed beta versions and deployment sequence need MAN-001 and MAN-002 at rollout. Their availability does not block document review. |
| GAP-003 | FR-009, RULE-003 | AC-010, AC-014 | Flutter and Svelte UI fixtures do not prove server publication or isolation. Pair those results with IT-003 and IT-004. A controlled rollout check remains distinct from local integration evidence. |

## 10. Out of scope

- Legacy schedule payloads and older scheduling clients. Scheduling tests use the current request and worker flow.
- Bulk record rewrites, new search surfaces, free-text picker creation, new Quilting fields, and the other product exclusions in requirements §8.
- Running implementation, production writes or publishing the rollout during test design.

## 11. Handoff to document review

- Review [01-requirements.md](01-requirements.md) alongside this specification. The revisions respond to the [document review](03-document-review.md). DR-001 maps to IT-002, REG-001 and MAN-001. DR-002 maps to IT-003. DR-003 maps to AT-006. DR-004 maps to AT-003. DR-005 maps to IT-006. The change remains high risk because it spans public record writes, legacy editing and multiple client releases. Explicit approval is required before implementation.
- Start with **IT-002**. Fetch public details and the owner edit seed for a legacy singleton. Submit a representative legacy change from that seed and exercise a destructive scalar edit against a multi-size project. Assert the PDS candidate and persisted size state. Use the existing AppView command and HTTP integration suites and pair with REG-001 for accepted updated-client changes.
- Continue with UT-001, UT-005, IT-001 and REG-001 for size reads, schema and publication. Add Phase 1 catalog, editor, search and current scheduling scenarios. Verify REG-004 for the Phase 1 candidate. Then cover the Phase 2 craft variants, editors and profile and discovery surfaces, and rerun Phase 1 coverage.
- Existing command targets from [justfile](../../../justfile) and [web-client/package.json](../../../web-client/package.json) follow. Run the focused AppView command with the local PostgreSQL and MinIO services. The full AppView gate uses an isolated stack. Web Playwright uses fixture-backed AppView responses.

```sh
just lexgen-check
just appview-test-unit
./scripts/appview-test focused ./internal/api -run 'TestEditPost|TestProjectSearch|TestHashtagAllTimeCountsAndAdditionalCraftDiscovery'
just appview-check
just app-test test/projects test/onboarding test/profile/widgets/edit_profile_crafts_picker_test.dart test/scheduled_posts/scheduled_composer_submission_test.dart
```

From `web-client/` run `npm run check`, `npm run test` or `npm run test:release` for the full web gate. An individual authenticated Playwright run can use `npm run build` followed by `npm run test:e2e:app -- authenticated-project-detail-parity.spec.ts`.

No gap blocks document review. GAP-001 is resolved during schema design, and GAP-002 and GAP-003 are verified at implementation and rollout.
