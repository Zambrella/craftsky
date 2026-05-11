# Business Requirements: Threads UX

## 1. Summary
Improve the Flutter app's post thread experience so a selected post opens in a stable, readable position, preserves parent context for replies, avoids recursive self-navigation, and gives users clear context when composing a reply.

## 2. Problem / Opportunity
The current thread screen renders ancestors before the selected post in a plain list, so opening a reply can place the selected post below the initial viewport. The selected post is also tappable, which can push the same thread route repeatedly. Reply composition correctly submits reply references but does not show the post being replied to above the input, making it easier for users to lose context while writing.

## 3. Goals
- G-001: Make the selected post the visual anchor when a thread route opens.
- G-002: Preserve conversation context by keeping parent posts available above reply targets and replies below the selected post.
- G-003: Prevent recursive self-navigation from the selected post.
- G-004: Show replied-to context in the composer without distracting from text entry.

## 4. Non-Goals
- NG-001: Do not change the AppView thread API or `PostThread` response shape.
- NG-002: Do not change atproto lexicons, PDS write semantics, or reply record references.
- NG-003: Do not introduce new reply pagination, ancestor pagination, or continuation loading behavior.
- NG-004: Do not redesign `PostCard` globally or change feed/profile card behavior outside the thread and composer flows.

## 5. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| Reader | A signed-in user viewing posts and threads. | Open a post and immediately understand which post is selected, with usable conversation context. |
| Replier | A signed-in user composing a reply. | See the post being replied to while writing and submit the reply to the correct parent/root. |
| Test designer | Agent or engineer writing acceptance tests. | Stable behavior and requirement IDs that can be translated into widget and regression tests. |

## 6. Current Behavior
- `PostThreadPage` renders `thread.ancestors` before `thread.post`, followed by reply controls and `thread.replies`.
- A reply target can initially appear below the top of the viewport because its ancestors occupy the beginning of the list.
- `_ThreadPostCard` wires `PostCard.onTap` to push `PostThreadRoute` for all rendered posts, including the selected anchor post.
- Replies and continuation controls are already rendered below the selected post.
- `PostComposerSheet` accepts a `replyTarget` and creates correct reply refs, but only displays the reply title/hint and text input.

## 7. Desired Behavior
- Opening any post thread places the selected post's top just below the app bar once the thread content is loaded and laid out.
- If the selected post is top-level, its direct replies appear below it.
- If the selected post is a reply, its ancestors remain in the same scrollable content above it, ordered from root/top-most parent to immediate parent, and can be reached by scrolling upward.
- The selected post itself is not actionable as a navigation target and cannot push the same post thread route again.
- Other posts and continuation controls continue to navigate when they point to a different post.
- When composing a reply, a compact non-actionable preview of the reply target appears above the text input, showing author context and up to three lines of the target post text.

## 8. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | The app shall make the selected post immediately identifiable when a thread route opens. | Users should not have to hunt for the post they clicked. | Initial prompt; Discovery Q1 | AC-001, AC-002 |
| BR-002 | Business | Must | The app shall preserve thread context around reply targets without making parent context the initial visual anchor. | Replies need parent context, but the selected post must remain the focus. | Initial prompt; Discovery Q1 | AC-002, AC-003 |
| BR-003 | Business | Must | The app shall prevent infinite or recursive navigation to the currently selected post. | Repeated self-routes create confusing navigation stacks and no useful state change. | Initial prompt; Discovery findings | AC-004 |
| BR-004 | Business | Must | The app shall show the reply target while a user is composing a reply. | Users need confidence they are replying to the intended post. | Initial prompt; Discovery Q2 | AC-006, AC-007 |
| FR-001 | Functional | Must | When a top-level post thread opens, the selected post shall be positioned with its top just below the app bar after content loads. | Satisfies the requested thread-opening behavior for root posts. | Initial prompt; Discovery recommendation | AC-001 |
| FR-002 | Functional | Must | When a reply thread opens, the selected reply shall be positioned with its top just below the app bar after content loads. | Confirmed that reply targets should also open selected-at-top. | Discovery Q1 | AC-002 |
| FR-003 | Functional | Must | For a selected reply, the thread shall keep available ancestors above the selected post in root-to-immediate-parent order within the same scrollable thread content. | Preserves parent context without changing the selected post's initial anchor. | Initial prompt; Discovery Q1 | AC-003 |
| FR-004 | Functional | Must | The thread shall render replies below the selected post for both top-level selected posts and selected replies. | Maintains conversation continuation below the focused post. | Initial prompt; Discovery findings | AC-001, AC-002 |
| FR-005 | Functional | Must | The selected post shall not navigate when tapped or otherwise activated as a post card. | Prevents recursive routes for the selected post. | Initial prompt; Discovery findings | AC-004 |
| FR-006 | Functional | Must | Ancestor posts, reply posts, and continuation controls shall retain navigation when their destination is a different post. | Avoids breaking normal thread exploration while disabling only self-navigation. | Discovery scope boundaries | AC-005 |
| FR-007 | Functional | Must | In reply mode, the composer shall display a compact, non-actionable preview of the reply target above the text input. | Provides reply context without exposing full post actions. | Discovery Q2 | AC-006 |
| FR-008 | Functional | Must | The compact reply preview shall include the reply target author's display context and up to three lines of the target post text. | Makes the preview informative and bounded. | User answer on preview truncation | AC-006, AC-007 |
| FR-009 | Functional | Must | Reply submission shall continue to use the selected reply target's existing root and parent references. | The UX change must not alter reply semantics. | Discovery findings; Non-goal NG-002 | AC-008 |
| NFR-001 | Non-functional | Should | Thread scroll anchoring should avoid visible layout jank after async data load across small and large form factors. | Scroll positioning is user-visible and device-size sensitive. | Discovery risks | AC-009 |
| NFR-002 | Non-functional | Should | The compact reply preview should use existing app theme spacing, typography, and color conventions. | Keeps the change visually consistent without a broader redesign. | Discovery open question | AC-010 |
| RULE-001 | Business rule | Must | The currently selected thread post is the only post card in the thread whose post-card navigation must be disabled solely because it is selected. | Prevents over-disabling navigation for other thread content. | Discovery scope boundaries | AC-004, AC-005 |
| RULE-002 | Business rule | Must | No AppView, API, lexicon, persistence, or PDS write changes shall be required for this feature. | The current response already supplies needed data; scope is Flutter UX. | Discovery Q3; Non-goals | AC-011 |

## 9. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-004 | Given a top-level post with replies, when the user opens its thread, then the selected post's top is positioned just below the app bar and its replies are rendered below it. |
| AC-002 | BR-001, BR-002, FR-002, FR-004 | Given a reply post with ancestors and replies, when the user opens its thread, then the selected reply's top is positioned just below the app bar and its replies are rendered below it. |
| AC-003 | BR-002, FR-003 | Given a selected reply with indexed ancestors, when the thread is open, then scrolling upward reveals ancestors above the selected post ordered from root/top-most parent to immediate parent. |
| AC-004 | BR-003, FR-005, RULE-001 | Given the thread is open, when the user taps or activates the selected post card, then the app does not push another route for the same selected post. |
| AC-005 | FR-006, RULE-001 | Given the thread contains an ancestor, reply, or continuation control for a different post, when the user taps that item, then the app navigates to that post's thread route. |
| AC-006 | BR-004, FR-007, FR-008 | Given the user opens the composer to reply to a post, when the composer appears, then a compact non-actionable preview is displayed above the text input with author context and target post text. |
| AC-007 | BR-004, FR-008 | Given the reply target text is longer than three preview lines, when the composer appears, then the preview truncates the target text after three lines rather than expanding unbounded. |
| AC-008 | FR-009 | Given the user submits a reply from the composer, when the create action is invoked, then the reply root and parent references match the reply target semantics used before this UX change. |
| AC-009 | NFR-001 | Given the thread content loads asynchronously on small or large form factors, when the loaded content is first displayed, then the selected post is anchored without an obvious jump from an ancestor-first position. |
| AC-010 | NFR-002 | Given the compact preview is visible, when compared with surrounding composer UI, then it uses existing app theme tokens/patterns rather than bespoke styling inconsistent with the app. |
| AC-011 | RULE-002 | Given this feature is implemented, when reviewing changed files, then no AppView API contract, lexicon schema, persistence migration, or PDS write behavior change is required. |

## 10. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Selected post has no replies. | Selected post still anchors below the app bar and the existing empty-replies state appears below it. | FR-001, FR-002, FR-004 |
| EC-002 | Selected reply has missing or unavailable ancestors in the response. | Render the available ancestors above the selected post; do not block opening the selected post. | FR-002, FR-003 |
| EC-003 | Selected reply has many or tall ancestors. | Initial position still anchors the selected reply below the app bar; ancestors remain reachable by upward scrolling. | FR-002, FR-003, NFR-001 |
| EC-004 | Selected post and another rendered post have similar text/author labels. | Self-navigation disabling is based on the selected post identity, not visible text alone. | FR-005, FR-006, RULE-001 |
| EC-005 | Reply target text is empty or unusually short. | Preview still shows author context and any available text without reserving unnecessary extra space. | FR-007, FR-008 |
| EC-006 | Reply target text is very long or multiline. | Preview shows no more than three lines of target text and keeps the input accessible. | FR-008 |
| EC-007 | User opens composer from sticky prompt on small screens or inline prompt on large screens. | Both entry points show the compact reply-target preview and preserve reply refs. | FR-007, FR-009 |

## 11. Data / Persistence Impact
- New fields: None.
- Changed fields: None.
- Migration required: No.
- Backwards compatibility: Existing `PostThread` API/model and reply creation semantics remain unchanged.

## 12. UI / API / CLI Impact
- UI: Updates are required in the Flutter thread screen and reply composer UI.
- API: No API endpoint or response contract changes expected.
- CLI: None.
- Background jobs: None.

## 13. Security / Privacy / Permissions
- Authentication: No change.
- Authorization: No change.
- Sensitive data: No new sensitive data is introduced; preview displays already visible post content passed as the reply target.
- Abuse cases: None newly identified for this UX-only change.

## 14. Observability
- Events: None required.
- Logs: None required.
- Metrics: None required.
- Alerts: None required.

## 15. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Initial scroll anchoring may be flaky across async load, device size, or text scaling. | Selected post may appear off-position or visibly jump. | Cover with widget tests for top-level and reply targets; verify small and large form factors. |
| RISK-002 | Disabling selected-post navigation could accidentally disable navigation for other posts. | Thread exploration may regress. | Acceptance tests must cover selected post no-op and other post navigation. |
| RISK-003 | Reply preview could make the composer feel crowded. | Users may have less visible writing space. | Use compact preview, cap text at three lines, and preserve focus on the text input. |

## 16. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The existing `PostThread` response supplies enough ancestor/reply data for the desired UX. | API/AppView requirements would need to be reopened. |
| ASM-002 | "Just below the app bar" means the selected post begins at the top of the scrollable body area, accounting for normal scaffold/app-bar layout and safe areas. | Acceptance tests may need more precise pixel tolerances. |
| ASM-003 | A three-line target text cap is sufficient composer context for long reply targets. | Preview requirements may need design/product revision. |

## 17. Review Status
Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer: Unassigned
Date: 2026-05-10
Notes: Discovery review was opened and the user chose to move on. Requirements review is recommended before test design because scroll anchoring and navigation behavior are user-visible and form-factor sensitive.

## 18. Handoff To Test Design
- Requirements file: `02-requirements.md`
- Must-cover requirement IDs: BR-001, BR-002, BR-003, BR-004, FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-009, RULE-001, RULE-002
- Suggested test levels:
  - Flutter widget tests for `PostThreadPage` initial anchoring, ancestor/reply ordering, selected-post no-op, and other-post navigation.
  - Flutter widget tests for `PostComposerSheet` compact preview, three-line truncation, and preserved reply refs.
  - Regression checks confirming no AppView/API/lexicon changes are needed.
- Blocking open questions: None.
