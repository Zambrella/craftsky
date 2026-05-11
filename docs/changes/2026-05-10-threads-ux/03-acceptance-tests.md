# Acceptance Test Specification: Threads UX

## 1. Test Strategy
Use Flutter widget tests as the primary acceptance-test vehicle because this change is scoped to thread and composer UI behavior. Extend the existing `PostThreadPage` and `PostComposerSheet` test suites with fake repository data to verify initial anchoring, ancestor/reply ordering, selected-post no-op navigation, preserved navigation for other posts, compact reply preview, three-line truncation, and unchanged reply refs. Add a small number of regression checks to protect existing thread behaviors and confirm the feature remains Flutter-only. Manual review is recommended only for visual jank/theme fit that is difficult to assert reliably in widget tests.

## 2. Requirement Coverage Matrix
| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002 | AT-001, AT-002, UT-001 | Acceptance / Unit | Yes |
| BR-002 | AC-002, AC-003 | AT-002, AT-003, UT-001 | Acceptance / Unit | Yes |
| BR-003 | AC-004 | AT-004, UT-002 | Acceptance / Unit | Yes |
| BR-004 | AC-006, AC-007 | AT-006, AT-007, UT-003 | Acceptance / Unit | Yes |
| FR-001 | AC-001 | AT-001, UT-001 | Acceptance / Unit | Yes |
| FR-002 | AC-002 | AT-002, UT-001 | Acceptance / Unit | Yes |
| FR-003 | AC-003 | AT-003, UT-001 | Acceptance / Unit | Yes |
| FR-004 | AC-001, AC-002 | AT-001, AT-002, REG-001 | Acceptance / Regression | Yes |
| FR-005 | AC-004 | AT-004, UT-002 | Acceptance / Unit | Yes |
| FR-006 | AC-005 | AT-005, REG-002 | Acceptance / Regression | Yes |
| FR-007 | AC-006 | AT-006, UT-003 | Acceptance / Unit | Yes |
| FR-008 | AC-006, AC-007 | AT-006, AT-007, UT-003 | Acceptance / Unit | Yes |
| FR-009 | AC-008 | AT-008, REG-003 | Acceptance / Regression | Yes |
| NFR-001 | AC-009 | MAN-001, REG-004 | Manual / Regression | Partial |
| NFR-002 | AC-010 | MAN-002 | Manual | No |
| RULE-001 | AC-004, AC-005 | AT-004, AT-005, UT-002 | Acceptance / Unit | Yes |
| RULE-002 | AC-011 | REG-005 | Regression | Yes |

## 3. Acceptance Scenarios
### AT-001: Top-level thread opens with selected post anchored and replies below
Requirement IDs: BR-001, FR-001, FR-004
Acceptance Criteria: AC-001
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Thread opening position
  Scenario: Open a top-level post with replies
    Given the repository returns a top-level selected post with direct replies
    When the user opens the selected post's thread route
    Then the selected post is visible with its top just below the app bar
    And direct replies are rendered below the selected post
```

### AT-002: Reply thread opens with selected reply anchored and replies below
Requirement IDs: BR-001, BR-002, FR-002, FR-004
Acceptance Criteria: AC-002
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Thread opening position
  Scenario: Open a reply post with ancestors and replies
    Given the repository returns ancestors, a selected reply, and child replies
    When the user opens the selected reply's thread route
    Then the selected reply is visible with its top just below the app bar
    And child replies are rendered below the selected reply
```

### AT-003: Reply ancestors remain above selected reply in scrollback
Requirement IDs: BR-002, FR-003
Acceptance Criteria: AC-003
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Reply context
  Scenario: Scroll upward from selected reply to parent context
    Given the selected reply has a root ancestor and an immediate parent ancestor
    And the selected reply is initially anchored below the app bar
    When the user scrolls upward in the thread
    Then the root/top-most parent appears above the immediate parent
    And the immediate parent appears above the selected reply
```

### AT-004: Selected post does not navigate to itself
Requirement IDs: BR-003, FR-005, RULE-001
Acceptance Criteria: AC-004
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Thread navigation safety
  Scenario: Activate the selected post card
    Given the selected post thread is open
    When the user taps the selected post card
    Then the app does not push another route for the same post
    And the repository is not asked to load the same selected post again because of that tap
```

### AT-005: Other thread content still navigates to different posts
Requirement IDs: FR-006, RULE-001
Acceptance Criteria: AC-005
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Thread navigation safety
  Scenario: Activate non-selected thread content
    Given the thread contains an ancestor, a reply, and a continuation control for posts other than the selected post
    When the user taps each navigable item
    Then the app navigates to that item's post thread route
    And the selected-post no-op rule does not disable those items
```

### AT-006: Reply composer shows compact target preview above input
Requirement IDs: BR-004, FR-007, FR-008
Acceptance Criteria: AC-006
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_sheet_test.dart`

```gherkin
Feature: Reply composer context
  Scenario: Open composer in reply mode
    Given the user is replying to a post with author context and text
    When the reply composer opens
    Then a compact non-actionable preview is displayed above the text input
    And the preview includes the reply target author context
    And the preview includes the reply target text
```

### AT-007: Reply preview truncates long target text after three lines
Requirement IDs: BR-004, FR-008
Acceptance Criteria: AC-007
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_sheet_test.dart`

```gherkin
Feature: Reply composer context
  Scenario: Reply target has long text
    Given the user is replying to a post whose text is longer than three preview lines
    When the reply composer opens
    Then the compact preview constrains target text to at most three lines
    And the preview does not push the text input out of the usable composer area
```

### AT-008: Reply submission preserves existing root and parent refs
Requirement IDs: FR-009
Acceptance Criteria: AC-008
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_composer_sheet_test.dart`

```gherkin
Feature: Reply creation
  Scenario: Submit a reply after viewing target preview
    Given the reply composer is opened for a reply target with existing root and parent refs
    When the user enters reply text and submits
    Then the create action receives the trimmed text
    And the reply root ref matches the target's thread root
    And the reply parent ref matches the target post
```

### AT-009: Empty reply thread keeps selected post anchored
Requirement IDs: FR-001, FR-002, FR-004
Acceptance Criteria: AC-001, AC-002
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Thread empty state
  Scenario: Selected post has no replies
    Given the repository returns a selected post with no replies
    When the thread opens
    Then the selected post is anchored just below the app bar
    And the empty replies state appears below the selected post
```

## 4. Unit Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | BR-001, BR-002, FR-001, FR-002, FR-003 | AC-001, AC-002, AC-003 | Verify thread render order and initial anchor targeting are derived from selected post identity, not from first list item. | `PostThread` with `[root, parent]`, selected `target`, and replies. | Ancestors are before selected post in scrollback; selected post is the initial anchor target; replies remain below. | `app/test/feed/pages/post_thread_page_test.dart` |
| UT-002 | BR-003, FR-005, FR-006, RULE-001 | AC-004, AC-005 | Verify post-card navigation callback is disabled only for the selected post and remains present for non-selected posts. | Selected post, ancestor post, direct reply post. | Selected post has no navigation action; ancestor/reply have route-push actions. | `app/test/feed/pages/post_thread_page_test.dart` |
| UT-003 | BR-004, FR-007, FR-008 | AC-006, AC-007 | Verify compact reply preview content and text constraint. | Reply target with display name/handle and long multiline text. | Preview shows author context, text widget is constrained to max three lines with overflow behavior. | `app/test/feed/widgets/post_composer_sheet_test.dart` |
| UT-004 | FR-009 | AC-008 | Verify reply ref calculation remains unchanged after adding preview UI. | Top-level target and nested reply target. | Top-level target uses itself as root/parent; nested target keeps existing root and uses target as parent. | `app/test/feed/widgets/post_composer_sheet_test.dart` |

## 5. Integration Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | RULE-002 | AC-011 | Confirm no API/AppView integration path is required for the UX change. | Existing `PostApiClient.getThread` and `PostThread` mapper tests remain unchanged. | Run existing app model/API tests. | Thread data shape remains compatible; no new backend field is needed. | Existing `app/test/feed/data/post_api_client_test.dart`, `app/test/feed/models/post_thread_test.dart` |

## 6. Regression Tests
| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Thread direct replies render under the selected post and nested grandchildren are not expanded as direct replies. | FR-004 | Keep/extend existing `shows focused thread content` coverage in `post_thread_page_test.dart`. |
| REG-002 | Continuation controls navigate to the continued reply thread. | FR-006 | Keep existing continuation navigation test and add selected-post no-op beside it. |
| REG-003 | Reply composer submits trimmed text and correct reply refs. | FR-009 | Extend existing `reply mode shows reply copy and forwards reply refs` test to also assert preview presence. |
| REG-004 | Narrow-width/text-scale thread layout does not overflow. | NFR-001 | Keep existing narrow-width text-scale test and include anchored selected-post expectations. |
| REG-005 | Feature remains Flutter-only with no AppView/API/lexicon changes. | RULE-002 | During review, changed files should be limited to app UI/tests and docs unless explicitly justified. |
| REG-006 | Small screens use sticky reply prompt and large screens use inline reply prompt. | FR-007, FR-009 | Existing prompt form-factor tests should continue passing; preview appears regardless of entry point. |

## 7. Test Data
| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Top-level selected post with replies. | `PostThread(post: rootTarget, ancestors: [], replies: [replyA, replyB])`, where `rootTarget.text = 'target post'`. | AT-001, AT-009, UT-001 |
| TD-002 | Reply selected post with ancestors. | `PostThread(ancestors: [root, parent], post: targetReply, replies: [childReply])`; ancestor text values should be unique and ordered. | AT-002, AT-003, UT-001 |
| TD-003 | Navigation identity data. | Selected post and another post with similar visible text but distinct `uri`, `author.did`, and `rkey`. | AT-004, AT-005, UT-002 |
| TD-004 | Compact reply preview target. | `Post` with display name, handle, and short target text. | AT-006, UT-003 |
| TD-005 | Long reply preview target. | `Post` with text long enough to exceed three rendered lines at phone width. | AT-007, UT-003 |
| TD-006 | Nested reply target refs. | Reply target with existing `reply.root` and `reply.parent`; expected submitted parent is target `uri/cid`. | AT-008, UT-004, REG-003 |
| TD-007 | Empty reply thread. | `PostThread(post: target, replies: const [])`. | AT-009 |

## 8. Manual Checks
| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-001 | Visual scroll-jank check. | Run the app, open a top-level thread and a reply thread on small and large form factors. | Content appears with selected post anchored; no obvious jump from ancestor-first layout. |
| MAN-002 | NFR-002 | Visual theme fit for compact preview. | Open reply composer from a thread and compare preview spacing/type/color with surrounding composer UI. | Preview looks native to Craftsky and does not crowd the input. |

## 9. Test Gaps And Risks
| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Pixel-perfect definition of "just below the app bar" may vary by Scaffold safe area and test environment. | FR-001, FR-002, NFR-001 | Widget tests can assert relative position/tolerance, but device chrome differs in real app usage. | Use tolerant widget assertions plus MAN-001. |
| GAP-002 | Theme consistency is partly subjective. | NFR-002 | Automated tests can find preview content/constraints, but visual polish requires review. | Use MAN-002 and Plannotator/folder review before implementation handoff. |

## 10. Out Of Scope
- AppView handler/store tests for thread response shape, because requirements explicitly avoid API/AppView changes.
- PDS write tests and lexicon validation tests, because reply record semantics must remain unchanged.
- Full end-to-end mobile automation on physical devices; widget tests plus manual visual checks are sufficient for this medium-risk UI change.
- New pagination tests for replies or ancestors; pagination is a non-goal.

## 11. Handoff To Document Review
- Requirements file: `02-requirements.md`
- Test specification: `03-acceptance-tests.md`
- Next review artifact: `04-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-10-threads-ux/`
- Recommended first failing test for implementation: `AT-002` in `app/test/feed/pages/post_thread_page_test.dart` — reply thread opens with selected reply anchored below the app bar while ancestors remain above in scrollback.
- Suggested test order for implementation:
  1. `AT-002` / `UT-001`: selected reply initial anchor with ancestors above and replies below.
  2. `AT-001` and `AT-009`: top-level and empty thread anchoring.
  3. `AT-004` / `UT-002`: selected post no-op navigation.
  4. `AT-005`: navigation for ancestors/replies/continuations still works.
  5. `AT-006` / `UT-003`: compact reply preview above composer input.
  6. `AT-007`: three-line truncation for long target text.
  7. `AT-008` / `UT-004` / `REG-003`: reply refs remain unchanged.
  8. `REG-004`, `MAN-001`, `MAN-002`: form-factor/text-scale and visual review.
- Commands discovered:
  - Focused thread page tests: `cd app && flutter test test/feed/pages/post_thread_page_test.dart`
  - Focused composer tests: `cd app && flutter test test/feed/widgets/post_composer_sheet_test.dart`
  - Broader Flutter feed tests: `cd app && flutter test test/feed`
  - Existing Go/AppView suite, if needed for regression confidence: `just test` after dev Postgres is running.
- Blocking gaps: None.
