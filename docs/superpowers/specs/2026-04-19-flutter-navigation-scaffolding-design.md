# Flutter Navigation Scaffolding — Design

**Date:** 2026-04-19
**Status:** Approved for implementation planning
**Scope:** `app/` (Flutter client)

## Context

The Flutter app currently has no router. The existing `lib/` tree contains `app.dart`, `app_dependencies.dart`, `bootstrap.dart`, `main.dart`, `l10n/`, `shared/widgets/`, and `theme/`. `MaterialApp.router` in [app/lib/app.dart](app/lib/app.dart) already expects a `goRouterProvider`, but the provider does not exist yet.

This design scaffolds the full screen inventory and navigation shape for the app **without any business logic or content** — every page is a stub `ConsumerWidget` returning a `Scaffold` with an `AppBar` and placeholder body. The goal is to lock in the route tree and shell shape so feature work can proceed in parallel.

The Squiddies app's [router](../../../squiddies_flutter/lib/router) (referenced in-session, not part of this repo) is the structural template: `go_router` with typed routes + `StatefulShellRoute` + dual bottom-nav/rail layout.

## Non-Goals

- No real authentication. `authStatusProvider` and `onboardingStatusProvider` are hardcoded stubs.
- No badge counts on tabs.
- No post-creation affordance (FAB, center-tab composer).
- No post-detail route.
- No notification listeners, deep-link handling, or lifecycle observers in the shell.
- No wiring of `atproto.dart` OAuth — the sign-in page is a visual stub.

These are all intentionally deferred. Each will be added as its own feature slice.

## Route Tree

```
/                           → redirect logic (no page)
/welcome                    → WelcomePage          (root nav, unauth)
/sign-in                    → SignInPage           (root nav, unauth)
/onboarding                 → OnboardingPage       (root nav, auth required, pre-onboarding)
/settings                   → SettingsPage         (root nav, pushed over shell)
/profile/:handle            → UserProfilePage      (root nav, pushed over shell)

StatefulShellRoute (AppShell) — shell-scoped routes:
  Branch 0: /feed          → FeedPage
  Branch 1: /search        → SearchPage
  Branch 2: /notifications → NotificationsPage
  Branch 3: /profile       → ProfilePage           (own profile, branch root)
                /profile/saved → SavedPage         (child of profile branch)
```

**Shell vs root-navigator placement:**

- `SettingsPage` is pushed on the **root navigator** from the profile page. It covers the whole screen and hides the bottom nav / rail — signals "you've left the main app."
- `UserProfilePage` (`/profile/:handle` — viewing someone else's profile) is pushed on the **root navigator** over the shell, matching Instagram's pattern.
- `SavedPage` is a **child of the profile branch** so the bottom nav stays visible — it's content browsing, not a modal flow.

**Route-class convention (go_router typed routes):** root-navigator routes set `static final GlobalKey<NavigatorState> $parentNavigatorKey = _NavigatorKeys.rootNavigatorKey;` so they push over the shell.

## Redirect Logic

```
- Not signed in + accessing protected route           → /welcome
- Signed in + not onboarded + not on /onboarding      → /onboarding
- Signed in + onboarded + on /welcome or /sign-in     → /feed
- Signed in + onboarded + on /onboarding              → /feed
```

Unauthenticated routes: `/welcome`, `/sign-in`.
Onboarding route (auth required, bypasses onboarding check): `/onboarding`.

Two stubbed providers drive this:

- `authStatusProvider` — returns `false` initially. A dev-only toggle on the welcome page can flip it to `true` during development so we can navigate through the flow.
- `onboardingStatusProvider` — returns `false` initially. The onboarding page has a "Finish" button that flips it to `true`.

Both live under their respective feature folders (`auth/providers/`, `onboarding/providers/`) and use the `@riverpod` annotation per [.claude/rules/riverpod.md](.claude/rules/riverpod.md).

## Auth Flow Shape

Given the atproto architecture ([AGENTS.md](AGENTS.md) rule 2: Flutter app never holds PDS tokens; OAuth tokens live in the App View), there is no traditional register flow. Account creation happens on the user's chosen PDS, not in-app.

- `/welcome` — landing screen with branding and "Sign in" / "Create account on a PDS" CTAs. Also hosts the dev-only auth toggle during development.
- `/sign-in` — handle entry field + "Continue" button. Stub for now — real implementation will kick off the atproto OAuth flow against the user's PDS.

No separate `/register` route.

## AppShell

Single `AppShell` widget, dual-path layout keyed off the existing [FormFactorWidget](app/lib/theme/form_factor.dart):

- **Small screens:** `NavigationBar` with 4 destinations. No "More" overflow — all 4 branches fit comfortably.
- **Large screens:** `NavigationRail` with the same 4 destinations, `labelType: NavigationRailLabelType.all`.

Both paths share a single list of destination specs (icon, selected icon, label) to avoid duplication.

**Icons (Material):**

| Tab | Unselected | Selected |
|---|---|---|
| Feed | `home_outlined` | `home` |
| Search | `search_outlined` | `search` |
| Notifications | `notifications_outlined` | `notifications` |
| Profile | `person_outline` | `person` |

**Tap-same-tab behavior:** `_onDestinationSelected` calls `navigationShell.goBranch(index, initialLocation: index == navigationShell.currentIndex)` — tapping the active tab pops the branch stack to its root.

**Explicitly out of scope for the shell:**

- No `WidgetsBindingObserver` / lifecycle hooks.
- No `ref.listen` notification / deep-link handlers.
- No badge wiring.
- No app-launch provider invocation.

These were all present in the Squiddies shell for feature reasons — they get added back per-feature, not up front.

## Navigator Keys & Branch Classes

Follow Squiddies' singleton pattern to avoid hot-reload issues — globals re-created on reload cause go_router to crash. One `_NavigatorKeys` class in `router.dart`:

```dart
class _NavigatorKeys {
  static GlobalKey<NavigatorState>? _rootKey;
  static GlobalKey<NavigatorState>? _feedKey;
  static GlobalKey<NavigatorState>? _searchKey;
  static GlobalKey<NavigatorState>? _notificationsKey;
  static GlobalKey<NavigatorState>? _profileKey;

  static GlobalKey<NavigatorState> get rootNavigatorKey =>
    _rootKey ??= GlobalKey<NavigatorState>(debugLabel: 'rootNavigator');
  // ... one getter per branch
}
```

Four `StatefulShellBranchData` subclasses — `FeedBranch`, `SearchBranch`, `NotificationsBranch`, `ProfileBranch` — each exposing `$navigatorKey` from the singleton.

## File Organization

```
app/lib/
  router/
    router.dart                  # GoRouter provider, typed routes, branch classes
    router.g.dart                # generated by go_router_builder + riverpod_generator
    app_shell.dart               # AppShell widget
    error_screen.dart            # fallback errorBuilder target
  auth/
    pages/
      welcome_page.dart
      sign_in_page.dart
    providers/
      auth_status_provider.dart
      auth_status_provider.g.dart
  onboarding/
    pages/
      onboarding_page.dart
    providers/
      onboarding_status_provider.dart
      onboarding_status_provider.g.dart
  feed/
    pages/
      feed_page.dart
  search/
    pages/
      search_page.dart
  notifications/
    pages/
      notifications_page.dart
  profile/
    pages/
      profile_page.dart          # own profile (shell branch root)
      user_profile_page.dart     # /profile/:handle (root nav)
      saved_page.dart            # /profile/saved (shell branch child)
  settings/
    pages/
      settings_page.dart
```

Matches the feature-folder layout used by Squiddies and the Riverpod file-organization convention in [.claude/rules/riverpod.md](.claude/rules/riverpod.md).

## Page Stubs

Every page is a `ConsumerWidget` returning:

```dart
Scaffold(
  appBar: AppBar(title: const Text('<Page Name>')),
  body: const Center(child: Text('<Page Name>')),
)
```

Exceptions:

- `WelcomePage` — also exposes a dev-only auth toggle button that flips `authStatusProvider`, plus "Sign in" / "Create account" buttons that navigate to `/sign-in`.
- `SignInPage` — stub handle-entry field + "Continue" button. Continue flips `authStatusProvider` to `true` (dev stub — real OAuth comes later).
- `OnboardingPage` — "Finish" button that flips `onboardingStatusProvider` to `true`.
- `ProfilePage` — contains navigation buttons to `/settings` and `/profile/saved` so the routes are reachable.

Per [.claude/rules/flutter.md](.claude/rules/flutter.md), each stub is a separate widget class — no `_build*` helpers.

## Error Handling

`GoRouter.errorBuilder` renders `ErrorScreen(error: state.error!)` — a minimal `Scaffold` with the error message and a "Go home" button that navigates to `/feed` (if authed) or `/welcome` (if not).

## Localization

Page titles go through `AppLocalizations` ([app/lib/l10n](app/lib/l10n)). Stubs can use hardcoded English strings initially with `// TODO: l10n` markers — adding l10n keys for stub pages would churn when real UI lands.

## Testing

Minimal — we're testing wiring, not UI.

- **One widget test per page:** verifies the page renders without throwing. Shared helper so this is mostly a table of route classes.
- **One router test:** verifies redirect rules.
  - unauth → `/welcome`
  - auth + not onboarded → `/onboarding`
  - auth + onboarded → `/feed`
  - auth + onboarded + navigating to `/welcome` → `/feed`

No deep-linking tests, no shell-state-preservation tests yet — those belong with the features that depend on them.

## Dependencies

No new `pubspec.yaml` entries. All required packages are already present:

- `go_router: ^17.0.0`
- `go_router_builder: ^4.1.3` (dev)
- `flutter_riverpod: ^3.0.3`
- `riverpod_annotation: ^4.0.0`

## Success Criteria

1. `flutter run` starts at `/welcome` with auth stub `false`.
2. Flipping the dev auth toggle navigates to `/onboarding`.
3. Tapping "Finish" on onboarding navigates to `/feed`.
4. All 4 tabs switch branches; active-tab tap pops to branch root.
5. `/profile/saved` is reachable from the profile page with the bottom nav still visible.
6. `/settings` is reachable from the profile page and covers the full screen (no bottom nav).
7. `/profile/:handle` is reachable (dev-only nav button acceptable) and covers the full screen.
8. Router and all pages pass `dart run build_runner build` with no errors.
9. `flutter analyze` passes with no warnings.
10. Widget and router tests pass.

## Follow-Ups (Out of Scope, Noted)

Tracked here so they aren't forgotten but explicitly not part of this change:

- Real atproto OAuth wiring on `/sign-in`.
- Real `authStatusProvider` / `onboardingStatusProvider` backed by app-view session + profile data.
- Post-detail route (`/post/:uri` — shape pending atproto post model).
- Post-composer affordance + route.
- Notification badge on the Notifications tab.
- Deep-link handling for profile handles and post URIs.
- Responsive navigation rail polish (extended mode at XL breakpoints, etc).
