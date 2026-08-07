# Requirements: Responsive App Navigation Drawer And Extended Rail

## 1. Initial Request

Add an app drawer on smaller screens alongside the existing bottom navigation bar. The drawer must be available from every top-level page by a leading hamburger action and a leading-edge swipe. It should present the five existing primary destinations plus Saved posts, Scheduled posts, Drafts, and Settings, followed by Terms, Privacy, and a Feedback action near the bottom.

Extend the large-screen navigation rail with the same destinations and footer actions. Use the attached Bluesky drawer screenshot as general layout inspiration, but do not add a prominent profile summary or profile header at the top.

Saved posts, Scheduled posts, and Drafts must also stop being Settings features: remove them from the Settings page and move their typed routes out from beneath `SettingsRoute` in the route hierarchy.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/router/app_shell.dart` owns the responsive shell. It currently shows a five-item `NavigationBar` for mobile/tablet and a five-item `NavigationRail` for laptop/desktop.
  - `app/lib/theme/form_factor.dart` classifies widths up to and including 900 logical pixels as small and widths above 900 logical pixels as large.
  - `app/lib/router/router.dart` defines the five stateful shell branches. `ScheduledPostsRoute`, `DraftsRoute`, and `SavedPostsRoute` are currently children of `SettingsRoute`, although their pages are lifted onto the root navigator.
  - `app/lib/router/route_locations.dart` centralizes route locations. The three personal-content routes currently resolve beneath `/profile/settings`.
  - `app/lib/feed/pages/feed_page.dart`, `app/lib/projects/pages/projects_page.dart`, `app/lib/search/pages/search_page.dart`, and `app/lib/notifications/pages/notifications_page.dart` each own a nested `Scaffold` and `SliverAppBar`.
  - `app/lib/profile/pages/profile_page.dart` and `app/lib/profile/widgets/profile_sliver_app_bar.dart` own the top-level Profile scaffold and custom collapsing `SliverAppBar`.
  - `app/lib/settings/pages/settings_page.dart` currently exposes Saved posts, Scheduled posts, and Drafts as Settings rows.
  - `app/lib/shared/link/external_link.dart` provides the existing external-link launch behavior, and `url_launcher` is already a runtime dependency.
  - `app/test/router/app_shell_account_switcher_test.dart` and `app/test/notifications/app_shell_notification_badge_test.dart` cover responsive shell behavior, profile/account switching, and notification badges.
- Existing patterns:
  - Primary destinations are ordered Feed, Projects, Search, Notifications, and Profile.
  - Selecting the current primary destination returns that branch to its initial location; selecting another branch preserves the stateful shell's branch state.
  - Notifications expose a capped visible badge and an accessible full-count label in both compact and large navigation.
  - The Profile destination uses the active account avatar and supports the existing account-switcher interaction.
  - Settings and its current child management pages are lifted onto the root navigator so they cover the responsive shell navigation and retain nested route back behavior.
- Current behavior:
  - Small screens show only the five-item bottom navigation bar and have no app drawer.
  - Large screens show only the five-item navigation rail.
  - Saved posts, Scheduled posts, and Drafts appear as rows on the Settings page and exist beneath Settings in the typed route hierarchy.
  - Top-level app bars are owned by branch pages rather than the outer shell.
- Constraints discovered:
  - Adding a `drawer` only to the outer shell `Scaffold` is sufficient for Flutter's leading-edge gesture, scrim, and dismissal behavior, but it will not automatically inject hamburger icons into app bars owned by nested branch scaffolds. Each top-level app-bar presentation needs a shared shell-drawer opener.
  - The custom Profile header needs the same top-level drawer affordance without introducing a profile summary above the menu.
  - The personal-content destinations must reuse their existing page implementations while moving their typed routes out of `SettingsRoute`; they must not create duplicate pages or a parallel navigation stack.
  - The current public legal destinations are `https://craftsky.social/terms` and `https://craftsky.social/privacy`.
  - The repository does not currently define an end-user Feedback destination or feedback flow, and the user has confirmed the first implementation should leave the button intentionally inert.
- Test/build commands discovered:
  - `just app-test [path-or-args]`
  - `just app-analyze`
  - `dart format` from `app/` for changed Dart sources during implementation
  - `dart run build_runner build --delete-conflicting-outputs` from `app/` only if generated routes or providers change
  - `git diff --check`

## 3. Clarifying Questions And Decisions

### Q1: Does the existing bottom navigation remain on smaller screens?

Answer: Yes, inferred directly from the request to add a drawer to the current small-screen navigation rather than replace the bottom bar.

Decision / implication: Mobile and tablet layouts retain the five core destinations in the bottom bar. The drawer adds a second route-access surface and exposes the secondary, legal, and feedback destinations.

### Q2: Does the large-screen navigation rail receive the new destinations too?

Answer: Yes. The request states that the navigation rail will contain the core five routes plus the additional routes and footer links.

Decision / implication: Drawer and rail are responsive presentations of one shared application-menu definition. The compact bottom bar remains intentionally limited to the core five destinations.

### Q3: Which pages count as top-level pages for drawer access?

Answer: The five existing stateful shell branch roots: Feed, Projects, Search, Notifications, and the signed-in member's Profile.

Decision / implication: Each branch root shows a hamburger action on small screens and accepts the leading-edge drawer gesture. Pushed detail, editor, management, settings, and modal pages retain their normal Back/Close behavior and do not replace it with a hamburger action as part of this change.

### Q4: How closely should the Bluesky reference be followed?

Answer: It is general layout inspiration only, and the user explicitly does not want a prominent profile section at the top.

Decision / implication: Use a straightforward CraftSky menu with icon-and-label destinations, grouping, separators, legal links, and a bottom Feedback action. Do not add a large avatar, display name, handle, follower counts, share action, or other profile-summary block.

### Q5: Which existing labels and routes should the additional destinations use?

Answer: Reuse the current product labels and typed route destinations.

Decision / implication: Use `Saved posts`, the existing localized `Scheduled posts` title for the requested “Schedule posts” entry, `Drafts`, and `Settings`. Reuse `SavedPostsRoute`, `ScheduledPostsRoute`, `DraftsRoute`, and `SettingsRoute` rather than adding routes.

### Q6: Where should Terms and Privacy open?

Answer: Use the established public CraftSky destinations already present in repository configuration.

Decision / implication: Terms opens `https://craftsky.social/terms` and Privacy opens `https://craftsky.social/privacy` through the platform's external browser behavior.

### Q7: What should Feedback open?

Answer: The user confirmed that Feedback can go nowhere for now.

Decision / implication: Render the Feedback button in the requested footer position, but make activation an intentional no-op: it shall not navigate, launch an external destination, submit data, or show failure feedback. Do not silently substitute GitHub Discussions, email, an app-store review, or a new in-app form.

### Q8: Where do Saved posts, Scheduled posts, and Drafts belong in the route hierarchy and Settings UI?

Answer: The user confirmed that none of the three should exist beneath Settings in either the route hierarchy or the Settings page.

Decision / implication: Move `SavedPostsRoute`, `ScheduledPostsRoute`, and `DraftsRoute` from children of `SettingsRoute` to sibling children of `ProfileRoute`, while retaining their root-navigator presentation. Their paths become `/profile/saved`, `/profile/scheduled`, and `/profile/drafts`. `SavedPostFolderRoute` remains a child of `SavedPostsRoute`, becoming `/profile/saved/folder`. Remove the three corresponding rows from `SettingsPage`. Because the app has no production users, do not add legacy `/profile/settings/...` redirects or aliases.

## 4. Candidate Approaches

### Option A: One shell-owned application menu with responsive drawer and rail presentations

Summary: Define one shared set of primary, secondary, legal, and feedback destinations in the app shell. Render it as an edge drawer on small form factors and as an extended/custom rail on large form factors. Give each top-level page app bar access to the shell drawer through a shared opener/controller.

Pros:
- Keeps labels, icons, ordering, badges, selected state, and route behavior consistent across form factors.
- Preserves the stateful shell and existing bottom navigation behavior.
- Centralizes the drawer gesture and lifecycle in the shell while accommodating nested app bars.
- Makes future navigation additions less likely to drift between drawer and rail.

Cons:
- Requires a small shared contract between the outer shell and all five top-level app-bar presentations.
- The current `NavigationRail` may need a composed or custom layout to support secondary groups and a footer cleanly.

Risks:
- Incorrect scaffold context could make the hamburger fail on one or more branch pages.
- Care is needed to retain Profile header layout, account switching, notification badges, and branch-state semantics.

### Option B: Add an independent drawer and menu definition to each top-level page

Summary: Give every branch page its own `Scaffold.drawer`, hamburger behavior, and destination list while changing the large rail separately.

Pros:
- Flutter automatically supplies the hamburger icon when each page-local app bar and drawer share a scaffold.
- Each page can be implemented in isolation.

Cons:
- Duplicates menu definitions, routing, state, semantics, and footer behavior across five pages plus the rail.
- Makes label, icon, badge, account-avatar, and future destination drift likely.
- Couples product-wide navigation to feature pages.

Risks:
- Inconsistent behavior and accessibility between top-level pages.
- Higher regression and maintenance cost.

## 5. Recommended Direction

Recommended approach: Option A, one shell-owned application menu with responsive drawer and rail presentations.

Why: Navigation is already centralized in `AppShell`, and the five branch pages are only responsible for their content and app-bar presentation. A shared destination model preserves the existing stateful branch behavior and current notification/account affordances while solving the nested-scaffold hamburger constraint through one explicit shell integration. Moving the three personal-content routes beside Settings under Profile also makes the route hierarchy match their new first-class menu status without duplicating page logic.

## 6. Problem / Opportunity

Small-screen members can reach only the five core sections directly and must travel through Profile and Settings to reach common personal-content pages. Large-screen members have the same limited primary navigation. Adding a consistent application menu makes saved, scheduled, and draft content easier to find, gives every top-level page an obvious entry point to broader navigation, and provides persistent access to legal information and feedback without overloading the bottom bar.

## 7. Goals

- G-001: Make the full application menu available from every small-screen top-level page by button and swipe.
- G-002: Present a consistent destination set and ordering in the small-screen drawer and large-screen rail.
- G-003: Give members direct access to Saved posts, Scheduled posts, Drafts, Settings, Terms, Privacy, and Feedback.
- G-004: Preserve the existing five-item compact bottom navigation, stateful branch behavior, notification badge, and account-switcher affordances.
- G-005: Keep the menu visually simple and aligned with CraftSky rather than copying the reference app's prominent profile treatment.
- G-006: Remove Saved posts, Scheduled posts, and Drafts from Settings ownership in both the visible Settings UI and typed route hierarchy.

## 8. Non-Goals

- NG-001: Replacing or expanding the compact five-item bottom navigation bar.
- NG-002: Adding new Saved posts, Scheduled posts, Drafts, or Settings page implementations, APIs, persistence, permissions, or business behavior; relocating existing routes is in scope.
- NG-003: Adding a profile-summary header, follower/following counts, profile-sharing action, or account details at the top of the drawer or rail.
- NG-004: Redesigning top-level page content, unrelated Settings rows, unrelated detail-page back navigation, or account switching.
- NG-005: Creating legal content inside the Flutter app or changing the established Terms and Privacy pages.
- NG-006: Designing or connecting a feedback collection destination in this change.
- NG-007: Adding Help or other reference-screenshot destinations that were not requested.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in member on a small screen | Uses the mobile or tablet shell with bottom navigation. | Open the application menu from any top-level page and reach both core and secondary destinations. |
| Signed-in member on a large screen | Uses the laptop or desktop shell with a persistent rail. | See the expanded navigation and footer actions without opening a drawer. |
| Keyboard, switch-control, or screen-reader user | Navigates through semantics and focus rather than only touch gestures. | Discover, operate, understand, and dismiss every menu action predictably. |

## 10. Current Behavior

Mobile and tablet widths show a five-icon bottom `NavigationBar`; laptop and desktop widths show a labeled five-destination `NavigationRail`. Neither layout provides a broader application menu. The five branch roots each own their own scaffold/app-bar implementation, including a custom collapsing Profile header. Saved posts, Scheduled posts, and Drafts are nested beneath Settings, appear as Settings rows, and open on the root navigator.

## 11. Desired Behavior

At mobile and tablet widths, the shell provides a start-edge drawer. Feed, Projects, Search, Notifications, and Profile each display a leading hamburger action at their top-level root, and the drawer can also be opened with the platform-standard start-edge swipe. The five-item bottom bar remains visible when the drawer is closed.

The drawer presents the five core destinations first, then Saved posts, Scheduled posts, Drafts, and Settings. The three personal-content pages are siblings of Settings beneath Profile in the typed route hierarchy and no longer appear inside the Settings page. Terms and Privacy appear as lower-priority links, and Feedback is anchored in the footer area. The menu contains no prominent profile summary. Choosing an item closes the drawer and performs the corresponding navigation exactly once.

At laptop and desktop widths, the shell presents the same core, secondary, legal, and feedback destinations in an extended rail/menu layout, with footer actions visually separated from route destinations. No drawer or compact bottom bar is shown. Both layouts retain active-route feedback, notification newness, the Profile avatar/account affordance, responsive behavior, localization, and accessibility.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Signed-in members shall be able to open the full application menu from every top-level page on a small screen. | Makes broader navigation consistently discoverable instead of dependent on the current branch. | Prompt; Codebase | AC-001, AC-010 |
| BR-002 | Business | Must | Signed-in members shall have direct application-menu access to Saved posts, Scheduled posts, Drafts, and Settings. | Removes unnecessary traversal through Profile and Settings for common personal-content tasks. | Prompt; Codebase | AC-004, AC-005 |
| BR-003 | Business | Must | Signed-in members shall have persistent application-menu access to Terms, Privacy, and Feedback. | Keeps legal information and the product feedback channel easy to find. | Prompt | AC-007, AC-008 |
| BR-004 | Business | Must | Saved posts, Scheduled posts, and Drafts shall be first-class personal-content destinations rather than Settings features. | Their placement in the new application menu should also be reflected in information architecture and Settings content. | User follow-up | AC-018, AC-019 |
| FR-001 | Functional | Must | At small form factors, the shell shall expose a start-edge drawer that can be opened both by a leading hamburger action and the platform-standard start-edge swipe from each of the five top-level branch roots. | Provides both visible and gestural access and matches expected Flutter navigation behavior. | Prompt; Codebase | AC-001, AC-011 |
| FR-002 | Functional | Must | At large form factors, the shell shall show the extended navigation rail/menu and shall not show the compact drawer or bottom navigation bar. | Preserves the established responsive split while extending large-screen navigation. | Prompt; Codebase | AC-002 |
| FR-003 | Functional | Must | Drawer and rail shall present the core destinations in the existing order: Feed, Projects, Search, Notifications, Profile. | Preserves current information architecture and muscle memory. | Codebase | AC-003 |
| FR-004 | Functional | Must | Drawer and rail shall present secondary destinations after the core group in this order: Saved posts, Scheduled posts, Drafts, Settings. | Provides the requested additional routes with consistent, predictable grouping. | Prompt; Q5 | AC-004 |
| FR-005 | Functional | Must | Core destinations shall retain existing stateful-shell behavior. Secondary destinations shall reuse the existing page implementations and root-navigator presentation, using their updated typed routes. On a small screen, selecting any drawer item shall first close the drawer and then perform at most one navigation action. | Prevents duplicate pages, stale overlays, and regression in branch or Back behavior while allowing the required hierarchy change. | Codebase; Recommended direction; User follow-up | AC-005, AC-013, AC-018 |
| FR-006 | Functional | Must | Terms shall open `https://craftsky.social/terms` and Privacy shall open `https://craftsky.social/privacy` using the existing safe external-link behavior. | Reuses established public legal destinations and platform browser handling. | Codebase; Q6 | AC-007 |
| FR-007 | Functional | Must | The shared menu shall provide a visually distinct Feedback button in its footer area. In this release, activating it shall intentionally perform no navigation, external launch, data submission, or user-visible side effect. | Establishes the requested placement without inventing an unapproved feedback channel. | Prompt; User follow-up; Q7 | AC-008 |
| FR-008 | Functional | Must | Navigation presentations shall preserve the current notification badge/count semantics and the Profile destination's active-account avatar and account-switcher affordance wherever those destinations currently expose them. | Avoids regressing existing account and notification behavior while centralizing navigation. | Codebase | AC-006, AC-017 |
| FR-009 | Functional | Must | The drawer and extended rail shall use icon-and-label rows, clear grouping, separators where helpful, and footer actions without a prominent profile-summary block. | Follows the requested reference direction while honoring the explicit profile constraint. | Prompt; Screenshot; Q4 | AC-009 |
| FR-010 | Functional | Must | The hamburger action shall appear only at the five branch roots on small screens. Pushed detail, editor, management, Settings, and modal pages shall retain their existing Back or Close affordance. | Prevents the drawer action from obscuring navigation hierarchy or replacing expected Back behavior. | Prompt; Codebase; Q3 | AC-010 |
| FR-011 | Functional | Must | The small-screen drawer shall support standard dismissal through destination selection, scrim tap, start-edge reversal/back gesture where supplied by the platform, and system Back or Escape where applicable; dismissing without selection shall not navigate. | Meets platform expectations and prevents accidental route changes. | Prompt; Flutter pattern | AC-011 |
| FR-012 | Functional | Must | Changing window size or device orientation across the existing form-factor boundary shall switch cleanly between compact drawer/bottom-bar and extended-rail presentations without duplicate navigation, lost current branch, stale scrim, or an orphaned open drawer. | The application already supports responsive rebuilds and must remain stable across them. | Codebase | AC-012 |
| FR-013 | Functional | Must | `SavedPostsRoute`, `ScheduledPostsRoute`, and `DraftsRoute` shall be sibling children of `ProfileRoute`, not children of `SettingsRoute`; `SavedPostFolderRoute` shall remain a child of `SavedPostsRoute`. Normal route-hierarchy Back behavior from each relocated top-level page shall return to Profile, never Settings. | Makes route ownership and parent navigation match the user's requested information architecture while preserving personal-content grouping under Profile. | User follow-up; Codebase | AC-018 |
| FR-014 | Functional | Must | `SettingsPage` shall not display rows or other navigation actions for Saved posts, Scheduled posts, or Drafts; all unrelated existing Settings actions shall remain available and keep their behavior. | Removes the three destinations from Settings without causing unrelated Settings regressions. | User follow-up; Codebase | AC-019 |
| NFR-001 | Non-functional | Must | Every hamburger, destination, legal link, and Feedback action shall have an accurate accessible name, selected/expanded state where applicable, sufficient touch target, logical reading/focus order, and keyboard activation and dismissal behavior on supported platforms. | The expanded navigation must be usable without relying on icons, color, or swipe gestures alone. | Accessibility baseline; Codebase | AC-014 |
| NFR-002 | Non-functional | Must | All new user-visible labels, tooltips, semantic descriptions, and launch-failure messages shall be localized through the existing app localization system. | Keeps the navigation consistent with the localized application. | Codebase | AC-015 |
| NFR-003 | Non-functional | Must | The drawer and rail shall respect safe areas, text scaling, RTL/start-edge direction, and short or narrow viewports without clipping, overflow, or making footer actions unreachable. | Navigation is app-critical and must remain usable across supported layouts and accessibility settings. | Prompt; Responsive UI baseline | AC-009, AC-016 |
| NFR-004 | Non-functional | Must | Automated verification shall cover both form-factor families, all five branch roots, every destination action, dismissal, active state, route reuse, badges/account affordances, resize behavior, and accessibility semantics, with targeted manual device checks for native edge gestures and screen readers. | The change crosses the shell, multiple app bars, routing, and platform gestures. | Codebase; Risk assessment | AC-017 |
| RULE-001 | Business rule | Must | The existing form-factor policy remains authoritative: mobile and tablet (width at or below 900 logical pixels) use the compact drawer plus bottom bar; laptop and desktop (width above 900 logical pixels) use the extended rail. | Avoids introducing a competing breakpoint or changing established responsive behavior. | Codebase | AC-002, AC-012, AC-013 |
| RULE-002 | Business rule | Must | “Top-level page” for this change means only the root of Feed, Projects, Search, Notifications, or the signed-in member's Profile branch. | Makes drawer coverage and Back behavior testable. | Codebase; Q3 | AC-001, AC-010 |
| RULE-003 | Business rule | Must | The compact bottom navigation remains limited to the five existing core destinations and is not expanded with secondary, legal, or feedback actions. | Nine or more bottom-bar destinations would be crowded and were not requested. | Prompt; Q1 | AC-013 |
| RULE-004 | Business rule | Must | The application menu shall not include a large profile identity/header section or unrequested Help action. | Preserves the user's explicit design boundary and avoids copying unrelated screenshot features. | Prompt; Screenshot | AC-009 |
| RULE-005 | Business rule | Must | The relocated routes shall use `/profile/saved`, `/profile/scheduled`, and `/profile/drafts`; the saved-folder descendant shall use `/profile/saved/folder`; the old `/profile/settings/saved`, `/profile/settings/scheduled`, `/profile/settings/drafts`, and saved-folder descendant shall not remain as routes, aliases, or redirects. | Establishes a clean pre-launch route hierarchy without preserving obsolete Settings ownership. | User follow-up; Project constraint | AC-018 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, RULE-002 | Given the app is at a mobile or tablet width and the member is at the root of Feed, Projects, Search, Notifications, or Profile, when they tap the leading hamburger action or perform the supported start-edge swipe, then one drawer opens from the logical start edge over the current page. |
| AC-002 | FR-002, RULE-001 | Given the app width is above 900 logical pixels, when any top-level branch root renders, then the extended rail is visible and neither a drawer hamburger nor compact bottom bar is shown. |
| AC-003 | FR-003 | Given either responsive application menu is visible, then Feed, Projects, Search, Notifications, and Profile appear once each, in that order, with an icon and accessible label. |
| AC-004 | BR-002, FR-004 | Given either responsive application menu is visible, then Saved posts, Scheduled posts, Drafts, and Settings appear once each after the core group, in that order, using the current localized product labels. |
| AC-005 | BR-002, FR-005 | Given the member selects a core destination, then the existing stateful-shell branch behavior runs exactly once; and given they select Saved posts, Scheduled posts, Drafts, or Settings, then the corresponding typed route opens exactly once with its full-screen and Back behavior and no duplicate page is introduced. On a small screen, the drawer is no longer visible after the selection. |
| AC-006 | FR-008 | Given notification newness or active-account identity is available, when drawer, bottom bar, or rail renders the relevant destinations, then the Notifications destination retains its capped visible badge and full accessible count, and the existing Profile avatar/account-switcher behaviors remain available without showing another account's identity. |
| AC-007 | BR-003, FR-006 | Given the member selects Terms or Privacy, when external launching succeeds, then exactly the corresponding established CraftSky URL opens outside the app; if launching fails, the app remains usable and presents safe localized feedback. |
| AC-008 | BR-003, FR-007 | Given the application menu is visible, then a clearly labeled Feedback button is reachable in the footer area; when selected, the current app route and menu state remain unchanged and no navigation, external launch, data submission, or message occurs. |
| AC-009 | FR-009, NFR-003, RULE-004 | Given the drawer or extended rail renders, then it uses CraftSky styling, logical grouping, and a separated footer; it does not display a large avatar/profile summary, display name, handle, follower/following counts, share action, Help action, or any other unrequested reference-screenshot item. |
| AC-010 | BR-001, FR-010, RULE-002 | Given the member is on a pushed detail, management, Settings, editor, or modal page, when its app bar renders, then its existing Back or Close affordance remains authoritative and this change does not replace it with a hamburger; returning to a branch root restores small-screen drawer access. |
| AC-011 | FR-001, FR-011 | Given the small-screen drawer is open, when the member taps the scrim, uses system Back or Escape, reverses the gesture where supported, or selects a destination, then the drawer dismisses appropriately; dismissal without a destination performs no route change, and rapid repeated open/select input does not cause duplicate navigation. |
| AC-012 | FR-012, RULE-001 | Given the member changes orientation or resizes across 900 logical pixels while on any core branch, then the UI changes to the correct navigation presentation, keeps the current branch and its state, and leaves no stale drawer, scrim, or duplicate navigation surface. |
| AC-013 | FR-005, RULE-001, RULE-003 | Given the app is at a mobile or tablet width and the drawer is closed, then the existing bottom bar still contains only Feed, Projects, Search, Notifications, and Profile and retains its current selection/reselection behavior. |
| AC-014 | NFR-001 | Given a supported screen reader, keyboard, switch-control, or large-touch-target audit, when the member traverses the menu, then the hamburger exposes button and expanded/collapsed meaning, every item announces its localized name and selected state where applicable, focus order follows visual order, all actions can be activated without a swipe, focus remains contained while the modal drawer is open, and dismissal returns focus sensibly. |
| AC-015 | NFR-002 | Given any supported app locale, when the new navigation UI or a launch failure renders, then no new user-facing English string is hard-coded outside the localization system and labels remain understandable within available space. |
| AC-016 | NFR-003 | Given narrow supported widths, short landscape height, safe-area insets, RTL direction, or increased system text scaling, when the drawer or rail renders, then content opens from the logical start edge, does not overflow or clip, and every destination plus Terms, Privacy, and Feedback remains reachable by scrolling where needed. |
| AC-017 | NFR-004, FR-008 | Given feature verification runs, then automated tests cover compact and large shell selection, all five top-level hamburger integrations, drawer open/dismiss/select behavior, all core and secondary route actions, exact legal URLs, the configured Feedback action, badge/account regressions, resize across the breakpoint, semantics, and overflow; targeted manual checks cover native edge swipes and physical VoiceOver/TalkBack behavior. |
| AC-018 | BR-004, FR-005, FR-013, RULE-005 | Given the generated route hierarchy is inspected or any personal-content route is opened, then Saved posts resolves at `/profile/saved`, Scheduled posts at `/profile/scheduled`, Drafts at `/profile/drafts`, and a saved folder at `/profile/saved/folder`; all three top-level personal-content routes are sibling children of `ProfileRoute`, the folder route is a child of `SavedPostsRoute`, none is a child of `SettingsRoute`, normal hierarchy Back from the three relocated top-level pages returns to Profile rather than Settings, and none of the former `/profile/settings/...` locations resolves through a route, alias, or redirect. |
| AC-019 | BR-004, FR-014 | Given Settings renders after this change, then it contains no Saved posts, Scheduled posts, or Drafts row or action, while every unrelated existing Settings item remains present and navigates as before. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | The current top-level destination is selected again from the drawer or rail. | Reuse the existing branch reselection behavior, return that branch to its initial location as currently defined, and perform no duplicate navigation. | FR-005 |
| EC-002 | The member taps a drawer item repeatedly while it is closing. | At most one destination action runs and no duplicate page is pushed. | FR-005, FR-011 |
| EC-003 | The drawer is open when the viewport crosses to a large form factor. | Remove the drawer and scrim cleanly, retain the current branch, and show one extended rail. | FR-012, RULE-001 |
| EC-004 | A top-level page has a custom or collapsing app bar, including Profile. | The hamburger remains visible, operable, and correctly aligned at the branch root without damaging collapse behavior or adding a profile-summary menu header. | FR-001, FR-009, FR-010 |
| EC-005 | A secondary route covers the shell on the root navigator. | Preserve the route's current full-screen Back behavior; returning to the shell restores the appropriate responsive menu. | FR-005, FR-010 |
| EC-006 | Terms or Privacy cannot be opened. | Stay on the current app route, dismiss the drawer if a selection was made, and show safe localized failure feedback without exposing internal exception details. | FR-006, NFR-002 |
| EC-007 | Menu content cannot fit vertically at large text scale or in landscape. | Keep all actions reachable through a bounded scrolling region while respecting footer grouping and safe areas. | NFR-001, NFR-003 |
| EC-008 | The interface uses RTL directionality. | Drawer, hamburger, gestures, separators, icons, labels, and focus order use the logical start edge and directional layout. | FR-001, NFR-003 |
| EC-009 | Notification count or active profile identity changes while the menu is visible. | Update only the active account's badge/avatar presentation without closing the menu or showing stale identity. | FR-008 |
| EC-010 | A caller or stale deep link uses a former `/profile/settings/...` personal-content path. | Treat it as an unknown route under the existing router error behavior; do not silently redirect it to the relocated page. | FR-013, RULE-005 |

## 15. Data / Persistence Impact

- New fields: None identified.
- Changed fields: None identified.
- Migration required: No.
- Backwards compatibility: No persisted or wire-format compatibility impact. The three former Settings-nested route locations are intentionally removed without redirects because the app has no production users.

## 16. UI / API / CLI Impact

- UI:
  - Add a shell-owned start-edge drawer for mobile and tablet form factors.
  - Add a hamburger opener to all five branch-root app-bar presentations on small screens.
  - Extend the large-screen rail/menu with secondary destinations and footer actions.
  - Preserve the five-item small-screen bottom bar.
  - Remove Saved posts, Scheduled posts, and Drafts from the Settings page.
  - Add localized navigation, legal, Feedback, semantics, and safe legal-link launch-failure copy.
- Router:
  - Move Saved posts, Scheduled posts, and Drafts from children of `SettingsRoute` to sibling children of `ProfileRoute`.
  - Keep saved folders nested beneath Saved posts.
  - Regenerate typed route output and update all affected callers/tests to the new locations without legacy aliases.
- API: None.
- CLI: None.
- Background jobs: None.

## 17. Security / Privacy / Permissions

- Authentication: The menu remains inside the signed-in shell and introduces no authentication change.
- Authorization: Existing destinations retain their current authorization and account ownership behavior.
- Sensitive data: The menu must not add a profile-summary block or expose additional account data. Existing avatar/account behavior remains active-account scoped.
- External destinations: Terms and Privacy must use explicit approved HTTPS destinations and the existing safe launcher; Feedback is intentionally inert in this release, and arbitrary or user-controlled URLs are out of scope.
- Abuse cases: None identified beyond preventing duplicate activation and avoiding raw launch exceptions in user-facing feedback.

## 18. Observability

- Events: No new analytics event is required by this change.
- Logs: External launch failures may use existing safe error reporting, without sensitive account data or full exception leakage in user-visible messages.
- Metrics: None identified.
- Alerts: None identified.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | The outer shell drawer is not discoverable from nested branch app bars. | One or more top-level pages lacks a working hamburger despite swipe support. | Define one explicit shell drawer-opener contract and test every branch root, including the custom Profile header. |
| RISK-002 | Drawer and rail destination definitions drift. | Labels, order, icons, badges, or route behavior differ by form factor. | Use one shared destination model and shared action mapping for both presentations. |
| RISK-003 | Extending the rail causes vertical overflow on short screens or with large text. | Destinations or footer actions become unreachable. | Use bounded scrolling and responsive grouping; test short landscape and increased text scale. |
| RISK-004 | Root-navigator secondary routes conflict with shell selection or Back behavior. | Duplicate screens, lost branch state, or confusing selected state. | Reuse existing typed routes and navigation semantics; cover route entry and return paths in integration tests. |
| RISK-005 | Hamburger insertion disrupts page-specific app-bar actions or the custom Profile header. | Existing filters, notification settings, collapse layout, or Back affordances regress. | Keep page actions unchanged, reserve the leading slot only at branch roots, and add page-specific regression tests. |
| RISK-006 | The intentionally inert Feedback button is accidentally wired to an assumed destination. | Members are sent to an unapproved or nonfunctional channel. | Keep its action as an explicit no-op and verify that activation causes no navigation, launch, submission, or message. |
| RISK-007 | Account avatar or notification badge state becomes shared incorrectly during centralization. | Navigation shows stale or cross-account identity/newness. | Preserve current active-account-scoped providers and existing regression tests in every applicable presentation. |
| RISK-008 | Relocating personal-content routes leaves generated paths, callers, deep-link tests, or Settings tests partially on the former hierarchy. | Navigation may fail, old paths may remain reachable, or removed Settings rows may reappear. | Change the typed route tree and location constants together, regenerate once, search for former paths and Settings labels, and add positive new-path plus negative old-path coverage. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | “Smaller screens” and “larger screens” use the existing `FormFactor.isSmall`/`isLarge` split at 900 logical pixels. | A new breakpoint and additional responsive test matrix would be required. |
| ASM-002 | The existing bottom navigation remains alongside the new small-screen drawer. | If the drawer replaces it, compact navigation behavior and visual scope would change materially. |
| ASM-003 | The requested “Schedule posts” entry should use the existing page label `Scheduled posts`. | Visible copy and localization keys would need adjustment if imperative wording is intended. |
| ASM-004 | Terms and Privacy should open externally at the established extensionless URLs used by OAuth metadata. | In-app web views or different canonical destinations would change interaction and tests. |
| ASM-005 | The reference screenshot informs grouping and hierarchy only; CraftSky's current theme, spacing, icons, and account-switcher model remain authoritative. | A closer Bluesky visual reproduction would require separate design decisions and likely broader UI work. |

## 21. Open Questions

None identified.

## 22. Review Status

Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date: 2026-08-06
Notes: Review is recommended because the change crosses the responsive shell, five independently owned top-level app bars, stateful routing, a typed route hierarchy relocation, root-navigator routes, Settings content, account/notification navigation state, native drawer gestures, and accessibility. The user explicitly approved direct TDD implementation without the intermediate workflow stages.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001–BR-004, FR-001–FR-014, NFR-001–NFR-004, RULE-001–RULE-005
- Suggested test levels:
  - Flutter widget tests for drawer/rail content, grouping, selection, footer positioning, overflow, RTL, text scaling, semantics, and each page-specific hamburger.
  - Flutter router/integration tests for core branch selection/reselection, relocated secondary typed routes, positive new paths, negative former paths, drawer close-before-navigation, root-navigator Back behavior, resize across the breakpoint, external legal links, and Feedback action/failure.
  - Regression tests for the compact bottom bar, Settings-row removal with unrelated Settings preservation, notification badge semantics, active-account Profile avatar/switcher behavior, top-level app-bar actions, and the collapsing Profile header.
  - Manual iOS and Android checks for leading-edge swipe, system Back, safe areas, orientation changes, VoiceOver/TalkBack, and representative physical screen sizes.
- Blocking open questions: None.
