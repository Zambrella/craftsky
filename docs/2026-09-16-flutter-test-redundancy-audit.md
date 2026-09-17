# Flutter test redundancy audit

- Date: 16 September 2026
- Repository snapshot: `8b53c004114e2094fa795962b35a70a2f284d54d`
- Scope: all `*_test.dart` files under `app/test/`, the production code needed to establish test ownership, and the Flutter test entry points in `app/pubspec.yaml` and `justfile`
- Audit type: static test-ownership and redundancy review; the full suite was not executed as part of this documentation-only audit
- Implementation status: all recommended phases implemented and verified on 16 September 2026

## Executive summary

The Flutter suite has broad behavioral coverage, but its maintenance cost is higher than the confidence it currently buys. The audit found four recurring causes:

1. Test infrastructure is copied into feature files instead of being composed from shared helpers.
2. Generic provider behaviors such as pagination and in-flight submission are retested feature by feature.
3. Large parent-widget suites reassert behavior already owned by focused child-widget suites.
4. A small number of tests verify constants, generated plumbing, or synthetic safe values without exercising a meaningful production boundary.

The suite contains 490 test files and at least 2,219 direct `test(...)` or `testWidgets(...)` declarations. It also contains 1,465 `pumpAndSettle()` calls across 122 files. The scale itself is not a problem, but the following duplication signals are material:

| Signal | Count |
|---|---:|
| Test files | 490 |
| Files constructing `ProviderScope` | 115 |
| Files constructing `MaterialApp` | 157 |
| Files referencing `SessionRegistryStorage` | 62 |
| Files using `pumpAndSettle()` | 122 |
| `pumpAndSettle()` calls | 1,465 |
| Flutter device/process integration tests | 0 |

A conservative first pass should remove or consolidate approximately 2,500 to 4,000 lines and 10 to 25 standalone tests without reducing meaningful behavioral coverage. The larger benefit will come from preventing each new feature from adding another local app harness, storage fake, post fixture, and copy of the pagination contract.

## Implementation outcome

All five improvement phases were completed in one repository pass. The resulting suite keeps the deliberate boundary, accessibility, responsive-layout, account-isolation, and design-contract repetition identified below.

| Measure | Before | After | Change |
|---|---:|---:|---:|
| Unit/widget test files | 490 | 483 | -7 net |
| Unit/widget test lines | 105,842 | 103,899 | -1,943 net |
| Direct `test(...)`/`testWidgets(...)` declarations | 2,219 | 2,176 | -43 |
| Files constructing `ProviderScope` | 115 | 112 | -3 |
| Files constructing `MaterialApp` | 157 | 149 | -8 |
| `pumpAndSettle()` calls | 1,465 | 1,389 | -76 |
| Flutter device/process journeys | 0 | 3 | +3 |

The net file count includes new contract and support tests. Eight standalone redundant test files and the two duplicate search-provider suites were deleted; reusable ownership contracts and one centralized mapper-bootstrap suite replaced them.

Implemented changes:

- Added composable test support for widget shells, Riverpod containers, language overrides, session storage, post fixtures, deterministic pumping, source scans, pagination, and async submission.
- Removed the duplicate app smoke test, pure forwarding repository suites, generated-value tests, duplicate mapper-decoding tests, and the isolated localization test.
- Replaced synthetic privacy assertions with production-path or structural boundary coverage.
- Consolidated search, pagination, report submission, and profile-tab infinite-scroll contracts while preserving feature-specific cases.
- Reduced `PostCard` and thread-page tests to parent composition ownership and moved child behavior to focused suites.
- Replaced 76 causally clear broad settles with exact pumps, direct awaits, or bounded condition/finder waits. Remaining settles cover finite transitions where complete animation settlement is intentional.
- Added recursive architecture scanning with consistent generated-file exclusions.
- Added three device-level journeys using the production app, router, providers, repositories, API clients, socket transport, and mobile plugin lifecycle against deterministic in-memory state and a loopback AppView.
- Added separate `just app-test-integration` and `just app-test-integration-ci` commands so device tests do not slow the normal widget-test path.

Verification completed after the combined refactor:

- `flutter test`: 2,358 tests passed.
- Dart/Flutter static analysis: no diagnostics.
- Dart formatting check: 504 files checked, no changes required.
- Device integration suite on iOS Simulator: all three journeys passed.
- `git diff --check`: passed.

The integration suite intentionally does not automate initial OAuth ownership, APNs/FCM provider delivery, OS media permissions, or writes against shared live AppView/PDS infrastructure. Those journeys require disposable identities, signed platform services, or mutable credentials; replacing them with deeper mocks would recreate widget tests rather than add device-boundary confidence. The exact boundary and future requirements are documented in [app/integration_test/README.md](../app/integration_test/README.md).

## Overall assessment

| Area | Assessment | Direction |
|---|---|---|
| Unit and provider tests | Strong coverage, repeated contracts and setup | Consolidate common contracts and fixtures |
| Widget tests | Strong component coverage, excessive parent/child overlap | Assign one owner per behavior |
| API tests | Good transport coverage, some repository layers repeat it | Keep transport assertions at the HTTP boundary |
| Privacy and architecture tests | Important boundaries, but a few assertions are tautological | Preserve boundary scans; replace synthetic sink tests |
| Generated/value-object tests | Mixed: some protect application behavior, some test generator output | Classify individually |
| Integration tests | No Flutter `integration_test` layer | Add a small set of critical journeys |
| Test reliability | Heavy use of global settling | Prefer deterministic pumping and state-based waits |

## Best-practice baseline

This audit applies the following principles.

### Match the test level to the risk

Flutter's official testing guidance recommends many unit and widget tests, plus enough integration tests to cover important use cases. It explicitly describes the tradeoff: unit tests are cheap but provide lower confidence, widget tests provide higher confidence at higher maintenance cost, and integration tests provide the highest confidence at the highest cost.

CraftSky has the first two layers in depth but no Flutter device/process integration layer. More mocked widget coverage is therefore not always the best response to a confidence gap.

Source: [Flutter testing overview](https://docs.flutter.dev/testing/overview)

### Test observable contracts, not incidental implementation

Widget tests should find and interact with UI through stable, user-visible contracts such as text, semantics, and deliberate keys. Exact offsets, concrete border objects, internal child counts, and raw colors belong only where geometry or visual tokens are the behavior under test.

Source: [Flutter widget-test finders](https://docs.flutter.dev/cookbook/testing/widget/finders)

### Isolate Riverpod state per test

Riverpod recommends `ProviderContainer.test()` for provider tests and a fresh `ProviderScope` for widget tests. Overrides are the intended seam for dependencies. Auto-disposed providers should be kept alive with a subscription while the test observes them.

The suite generally follows this guidance. The opportunity is to package repeated overrides and subscriptions into small composable helpers, not to introduce one universal test application.

Source: [Riverpod testing providers](https://riverpod.dev/docs/how_to/testing)

### Wait for the event being tested

Use `pump()` for synchronous rebuilds, `pump(duration)` for known timers or animation intervals, and a bounded state/finder wait for asynchronous work. Reserve `pumpAndSettle()` for cases that genuinely require all scheduled frames to settle. Broad settling hides which event a test depends on, adds runtime, and can hang when an animation intentionally repeats.

The suite already demonstrates the preferred approach in [app_test.dart](../app/test/app_test.dart#L94-L117), where a single pump is used because the loading indicator repeats forever.

### Share stable data, keep scenario behavior local

Canonical model builders, wire payloads, language overrides, and storage implementations should be shared. Sequence fakes and recording fakes that encode one scenario's behavior should usually remain next to that scenario. Over-centralizing behavior-rich fakes makes tests harder to read and couples unrelated features.

## Findings

### FT-001 - Repeated app and dependency harnesses

**Priority:** High

The suite repeatedly reconstructs the same dependency shell:

- 115 test files construct a `ProviderScope`.
- 157 test files construct a `MaterialApp`.
- 62 test files reference `SessionRegistryStorage`, commonly through another local in-memory implementation.
- English `LanguagePreferences`, post objects, `Scaffold`, and messenger setup are repeated throughout feature suites.

A shared harness already exists at [app/test/test_support/app_harness.dart](../app/test/test_support/app_harness.dart), but it only supplies the provider retry policy and error reporter, and only its own test currently imports it. It is too narrow to reduce the common setup.

Six theme suites also declare near-identical local `_Harness` widgets:

- `app/test/theme/craftsky_text_inputs_test.dart`
- `app/test/theme/craftsky_form_builder_text_field_test.dart`
- `app/test/theme/craftsky_form_builder_dropdown_test.dart`
- `app/test/theme/craftsky_form_builder_radio_test.dart`
- `app/test/theme/craftsky_form_builder_multi_select_test.dart`
- `app/test/theme/craftsky_field_scaffold_test.dart`

**Recommendation:** create a small composable test kit rather than a universal harness:

- `InMemorySessionRegistryStorage`
- `englishLanguagePreferencesOverride`
- `postFixture(...)` and shared wire-payload fixtures
- `createTestContainer(...)`
- `pumpCraftskyWidget(...)` with optional overrides, locale, theme, surface size, and messenger

Helpers should have explicit parameters and minimal defaults. Tests for responsive layout, RTL, dark mode, and unusual dependency states must remain able to build their own shell.

### FT-002 - Duplicate app boot smoke coverage

**Priority:** High

[widget_test.dart](../app/test/widget_test.dart#L52-L71) only proves that a signed-out app reaches `WelcomePage`. [app_test.dart](../app/test/app_test.dart#L94-L177) already covers loading, initialization, signed-out routing, stable loading presentation, retry behavior, app wiring, and recovery states.

**Recommendation:** delete `app/test/widget_test.dart`. Keep `app_test.dart` as the owner of application bootstrap behavior.

### FT-003 - Pure forwarding repositories are over-tested

**Priority:** High

[saved_post_repository_test.dart](../app/test/saved_posts/data/saved_post_repository_test.dart#L13-L101) uses approximately half of its 200 lines to prove that a typed forwarding adapter passes each argument to `SavedPostApi`. The real transport contract is already covered in `saved_post_api_client_test.dart`.

The same issue is stronger in search: [search_repository_test.dart](../app/test/search/data/search_repository_test.dart#L35-L102) constructs a real `SearchApiClient` and mocked Dio transport, repeating endpoint, query, and decoding coverage already owned by `search_api_client_test.dart`.

`post_repository_test.dart` contains similar overlap around its real client-backed cases.

**Recommendation:**

- Delete repository tests when the implementation is a pure one-line delegate with no mapping, policy, caching, or error behavior.
- If a forwarding seam is considered valuable, retain one concise smoke test using a reusable fake rather than one assertion per argument for every method.
- Keep repository-named tests that are the only transport contract. For example, `api_onboarding_repository_test.dart` must remain unless transport coverage is first moved to a dedicated client suite.

### FT-004 - `PostCard` retests child-widget behavior

**Priority:** High

`post_card_test.dart` is approximately 3,082 lines with 88 tests. It repeatedly verifies details already owned by focused child suites:

| Behavior repeated through `PostCard` | Focused owner |
|---|---|
| External-card content and presentation | `external_card_test.dart` |
| Quote summary content, avatar details, image behavior, and taps | `post_summary_test.dart` |
| Carousel image behavior and indicators | `post_image_carousel_test.dart` |
| Interaction labels and zero-state details | `post_interaction_summary_test.dart` |

Representative overlap appears in [post_card_test.dart](../app/test/feed/widgets/post_card_test.dart#L178-L203), [post_card_test.dart](../app/test/feed/widgets/post_card_test.dart#L1124-L1244), and [post_card_test.dart](../app/test/feed/widgets/post_card_test.dart#L2543-L2649).

**Recommendation:** keep `PostCard` tests for composition seams only:

- The correct child is selected from the post model.
- Embed precedence is correct.
- Parent callbacks and routing are wired to the child.
- Parent-specific style and accessibility behavior is correct.
- One representative interaction reaches each important seam.

Labels, child styling, image cache behavior, carousel mechanics, and child semantics should be asserted only by the child suite.

### FT-005 - Search provider suites are parameter variations of one contract

**Priority:** Medium

[post_search_provider_test.dart](../app/test/search/providers/post_search_provider_test.dart) and [project_search_provider_test.dart](../app/test/search/providers/project_search_provider_test.dart) duplicate the post builder, English-language override, container setup, initial-load behavior, cursor forwarding, de-duplication, append behavior, and end-of-list no-op.

**Recommendation:** extract a parameterized search-provider contract that accepts the provider factory, fake callback, item identity accessor, query, and expected cursor. Keep separate files only for behavior that differs by search type.

### FT-006 - Pagination contracts repeat across providers and profile tabs

**Priority:** Medium

Timeline, user-post, search, and project providers repeatedly assert:

- Initial page loading
- Cursor forwarding
- Append and de-duplication
- End-of-list no-op
- Visible-state preservation after failure
- Retry with the same cursor

Examples include `timeline_provider_test.dart`, `user_posts_provider_test.dart`, and the two search-provider suites. Profile posts, comments, and projects tabs separately repeat the same “scroll near the end and append the next page” widget scenario and nearly identical `ProviderScope -> MessengerScope -> MaterialApp -> Scaffold -> CustomScrollView` shells.

**Recommendation:** introduce two narrow contracts:

- A provider-level pagination contract for generic state transitions.
- A profile-tab widget harness with one reusable infinite-scroll interaction.

Retain feature-specific cases such as timeline repost identity, invalid-cursor restart, pin metadata, rendering differences, navigation, and action menus.

### FT-007 - Report providers duplicate the same in-flight state machine

**Priority:** Medium

`report_post_provider_test.dart` and `report_profile_provider_test.dart` both verify that a second submission is ignored while the first is in flight. Failure and retry are covered only for post reporting.

**Recommendation:** define one async-submit contract and run it against both notifiers. The contract should cover initial state, in-flight duplicate suppression, success, failure, and retry. Keep payload-shape assertions in feature-specific tests.

### FT-008 - Tautological privacy assertions provide false confidence

**Priority:** High

[video_secret_scan_test.dart](../app/test/observability/video_secret_scan_test.dart#L5-L27) constructs a `VideoDiagnosticEvent` with safe fields and then asserts that unrelated secret literals are absent. Those literals never enter the object under test. The constructor-shape assertion at lines 30-42 is useful because it constrains the data the API can accept.

[business_privacy_architecture_test.dart](../app/test/business/business_privacy_architecture_test.dart#L77-L101) compares prohibited values against `_recordBoundedSink`, a local helper that returns only the three safe strings supplied by the test. It never calls a real logger, reporter, trace, metric, or route diagnostic.

By contrast, `instagram_import_privacy_test.dart` is meaningful because canaries enter a real archive and parsing/serialization path before the output is inspected.

**Recommendation:**

- Keep compile-time/API-shape constraints that prevent secret-bearing fields.
- Keep source-boundary scans that ban observability dependencies from sensitive feature code.
- Replace negative checks against values that were never supplied with call-path tests that insert canaries into actual inputs and inspect actual diagnostic output.
- Delete a negative privacy assertion when no production path can receive the canary and no enforceable boundary is being tested.

### FT-009 - `pumpAndSettle()` is the default wait mechanism

**Priority:** Medium

The suite contains 1,465 `pumpAndSettle()` calls across 122 files. Some are appropriate, but this volume indicates that tests often wait for the entire scheduler rather than the specific state transition under test.

**Recommendation:** replace calls incrementally when touching a suite:

- Use `pump()` after synchronous taps and provider overrides.
- Use `pump(duration)` for a known debounce or animation interval.
- Await notifier/provider futures directly in provider tests.
- Add a bounded `pumpUntilFound` or `pumpUntilState` helper for asynchronous UI, with a useful timeout failure.
- Keep `pumpAndSettle()` for finite multi-frame transitions where complete settlement is the contract.

Do not perform a blind global replacement. Each conversion must identify the event that drives the next assertion.

### FT-010 - Generated and trivial value-object behavior is tested inconsistently

**Priority:** Low

[user_posts_state_test.dart](../app/test/feed/models/user_posts_state_test.dart) tests a trivial `cursor != null` getter, generated `copyWith`, and exact `toString()` output. Exact generator output provides little application confidence and increases churn.

`search_post_page_test.dart` and `top_hashtags_test.dart` repeat generated mapper decoding already exercised through the real search API client.

`search_mapper_registration_test.dart` does protect mapper bootstrap, but a feature-specific empty-object test is a weak way to own that requirement.

**Recommendation:**

- Remove exact generated `toString`, equality, and ordinary `copyWith` tests.
- Test custom getters only when they encode non-trivial policy or a documented regression.
- Keep application-specific fallback, compatibility, and canonicalization behavior.
- Replace per-model mapper-registration smoke tests with one centralized bootstrap test that enumerates the mappers the application requires.
- Retain nullable-clear regression tests only when the generator's null-sentinel semantics have caused or could plausibly cause a real defect; document the regression in the test name or comment.

### FT-011 - Exact constants are mirrored instead of tested through consumers

**Priority:** Low

[media_config_test.dart](../app/test/feed/media/media_config_test.dart) mirrors every configuration value. It fails on any deliberate policy adjustment but does not prove that validators, upload preparation, or composer behavior use those limits.

Exact raw colors and geometry also appear in router and consumer-widget tests even where focused theme/model suites already own the design token.

**Recommendation:**

- Keep exact values that are external protocol or product-policy requirements.
- Test boundaries through consumers: just below, at, and above each limit.
- Keep exact palette and component-theme tests in the theme owner.
- In consumers, compare against the semantic token or resolved customization bundle rather than repeating raw hex values.
- Router tests should prefer destinations, selected state, accessibility, and responsive mode over offsets, pixel widths, icon counts, and border internals.

### FT-012 - Isolated localization assertions have little ownership

**Priority:** Low

`imported_post_l10n_test.dart` asserts one translated string that is also rendered and asserted in widget tests.

**Recommendation:** fold isolated string checks into a parameterized localization completeness or accessibility contract. Keep widget assertions where the copy itself is a user-visible requirement, but avoid asserting the same literal at both model/localization and widget levels without a distinct reason.

### FT-013 - Architecture scans should be shared, not removed

**Priority:** Medium

Source scans such as `router_usage_test.dart`, `observability/import_boundary_test.dart`, and `notification_architecture_test.dart` enforce negative cross-file constraints that runtime tests cannot prove. Their apparent implementation coupling is intentional.

The duplication lies in directory traversal, generated-file exclusion, and forbidden-pattern reporting. In addition, [link_preview_network_boundary_test.dart](../app/test/architecture/link_preview_network_boundary_test.dart#L7-L26) scans a fixed list of files, so a new preview source file could bypass the boundary.

**Recommendation:**

- Keep each architectural invariant as a separately named test.
- Extract shared recursive source-scan utilities with consistent generated-file exclusions and diagnostics.
- Prefer scanning an owned directory or dependency graph over a fixed file list.
- Replace scans with analyzer rules only when the rule provides equal or better enforcement and failure messages.

### FT-014 - Test labels overstate integration coverage

**Priority:** High

There is no `app/integration_test/` directory and no `integration_test` dev dependency. The app test recipe at [justfile](../justfile#L319) runs only `flutter test`. Tests named `IT-*` are generally widget tests, provider tests, or mocked-Dio tests; the identifier does not make them device/process integration tests.

This is not redundant coverage by itself, but it changes the value calculation: thousands of mocked assertions do not cover application/plugin lifecycle, real navigation across a running app, platform channels, or client/server integration.

**Recommendation:** add a deliberately small integration layer for critical journeys:

- Authentication and account switching
- One authenticated read flow
- One AppView-mediated PDS write flow
- Notification/deep-link routing
- One media/plugin lifecycle flow if it can run reliably in CI

Do not duplicate every widget test in integration tests. Cover journeys whose confidence specifically depends on a running application or real boundary.

## Intentional repetition to retain

The following similar-looking tests protect distinct risks and should not be consolidated away:

- Account/session isolation and account-switch behavior
- Signed-out redirects for each externally reachable deep link
- Privacy canaries that traverse real parsing, serialization, reporting, or networking paths
- Architecture scans enforcing different dependency or ownership rules
- Compact, wide, RTL, dark-mode, large-text, and semantics variants
- Timeline repost identity semantics
- User-post invalid-cursor recovery and pin metadata
- Theme-owner tests that lock exact approved design tokens
- One parent/child integration assertion at each important composition seam
- Local sequence or recording fakes whose behavior is specific to one scenario

## Improvement outline

### Phase 1 - Remove clear duplication

1. Delete `app/test/widget_test.dart` after running `app_test.dart`.
2. Remove trivial generated `toString`, ordinary `copyWith`, and duplicated mapper-decoding tests.
3. Remove or reduce pure forwarding repository suites, preserving the only owner of each HTTP contract.
4. Replace the two tautological privacy checks with real call-path coverage or delete the ineffective assertions.
5. Fold isolated localization-string tests into an owned contract.

Expected result: fewer standalone files and immediate maintenance savings with no behavioral loss.

### Phase 2 - Establish shared test primitives

1. Add canonical builders for post models and common wire payloads.
2. Add shared English-language and common provider overrides.
3. Add one in-memory session-registry storage implementation.
4. Expand the existing app harness into small composable widget/container helpers.
5. Add recursive source-scan utilities.

Expected result: feature tests describe their scenario rather than bootstrapping infrastructure.

### Phase 3 - Consolidate behavior contracts

1. Extract search-provider and generic pagination contracts.
2. Extract the async-submit contract for report providers.
3. Add a profile-tab infinite-scroll harness.
4. Reduce `PostCard` and other parent suites to composition responsibilities.
5. Centralize mapper-bootstrap coverage.

Expected result: generic behavior is tested once per implementation contract, while feature-specific differences remain explicit.

### Phase 4 - Improve determinism

1. Inventory `pumpAndSettle()` by suite, starting with files that use it ten or more times.
2. Replace broad settling with direct future awaits, exact pumps, or bounded state/finder waits.
3. Record full-suite duration and flaky retries before and after each batch.
4. Avoid adding sleeps or unbounded polling helpers.

Expected result: faster failures, clearer causal tests, and fewer animation-related timeouts.

### Phase 5 - Add high-value integration coverage

1. Add `integration_test` and a dedicated test entry point.
2. Implement only the critical journeys listed in FT-014.
3. Define local and CI commands separately from the fast `flutter test` suite.
4. Keep integration fixtures disposable and independent of production credentials.

Expected result: higher confidence at real boundaries without recreating the unit/widget suite at a slower level.

## Suggested ownership rules

Apply these rules during the cleanup and in future reviews:

| Behavior | Primary owner |
|---|---|
| JSON path, query, status, and decoding | API client test |
| Mapping, caching, policy, or error translation | Repository test |
| Generic state transition | Provider/notifier test |
| Child content, semantics, and callbacks | Child widget test |
| Child selection and wiring | Parent widget test |
| Navigation destination and redirect | Router test |
| Exact design token | Theme/model owner test |
| Responsive use of a token | Consumer widget test |
| Cross-file forbidden dependency | Architecture test or analyzer rule |
| Running-app/plugin/client-server journey | Integration test |

If a second test asserts the same behavior, its name should state the distinct boundary or regression it protects. Otherwise, remove it or move the assertion to the primary owner.

## Completion criteria

The audit recommendations are complete when:

- Every removed test has a named surviving owner for meaningful behavior.
- Shared helpers reduce setup without hiding scenario-specific dependencies.
- Provider contracts are parameterized only where implementations truly share semantics.
- Parent widget suites no longer retest child internals.
- Privacy tests pass canaries through real production paths or enforce a structural boundary.
- Architecture scans discover new files automatically.
- `pumpAndSettle()` is no longer the default wait strategy.
- A small Flutter integration suite covers critical running-app journeys.
- `flutter test`, static analysis, and the new integration command pass in CI.
- Suite size, runtime, and flaky retry rate are recorded before and after the cleanup.

## Risks

- Over-abstraction can make tests harder to read than local duplication. Extract only stable setup and genuinely shared contracts.
- A large mechanical cleanup can accidentally erase regression intent. Work in small feature-scoped batches and require a surviving-owner note in each change.
- Parameterized contracts can obscure feature-specific differences. Keep exception cases in named feature tests.
- Replacing `pumpAndSettle()` without identifying the awaited event can introduce race conditions. Convert intentionally, not mechanically.
- Integration tests can become slow and flaky if they depend on shared external state. Keep the initial set small, isolated, and deterministic.

## Recommended first pull requests

1. Test support primitives and migration of two representative suites.
2. Clear deletions: duplicate app smoke, trivial generated tests, and redundant repository cases.
3. Search and pagination contract consolidation.
4. `PostCard` ownership reduction.
5. Deterministic pumping in the highest-use suites.
6. Flutter integration-test scaffold with one critical journey.

Each pull request should report tests removed, lines removed, full-suite runtime, and the surviving owner for every deleted behavior.
