# Flutter Navigation Scaffolding Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scaffold the full screen inventory and navigation shape for the Craftsky Flutter app — every route from the spec stubbed out, shell shape locked in, redirect logic driven by stubbed auth/onboarding providers.

**Architecture:** `go_router` with typed routes + `StatefulShellRoute` (4 branches: Feed, Search, Notifications, Profile). Auth/onboarding screens outside the shell. Settings and foreign user profile pushed on the root navigator. Saved page lives inside the Profile branch. `AppShell` provides a dual `NavigationBar`/`NavigationRail` layout keyed off `FormFactorWidget`.

**Tech Stack:** Flutter, Dart 3.11+, `go_router: ^17.0.0` + `go_router_builder: ^4.1.3`, `flutter_riverpod: ^3.0.3` + `riverpod_annotation: ^4.0.0` (code generation), `flutter_test`.

**Spec:** [docs/superpowers/specs/2026-04-19-flutter-navigation-scaffolding-design.md](../specs/2026-04-19-flutter-navigation-scaffolding-design.md)

**Working directory:** `app/` (the Flutter client). All paths below are relative to repo root.

---

## Context for the implementer

**You are expected to follow these rules:**

- [.claude/rules/flutter.md](../../../.claude/rules/flutter.md) — widget architecture (one class per widget, no `_build*` helpers), theming, immutable data, modern Dart syntax. Prefer **Dart MCP tools** (`mcp__dart__*`) over raw shell invocations for analyze/format/test/pub/launch operations.
- [.claude/rules/riverpod.md](../../../.claude/rules/riverpod.md) — all providers use `@riverpod` codegen. `ref.watch` in build methods, `ref.read` in callbacks. Use switch pattern matching on `AsyncValue`, not `.when()`. Use `FutureOr<T>` for idle providers.

**Existing state to be aware of:**

- [app/lib/router/router.dart](../../../app/lib/router/router.dart) already exists with a minimal router containing a single `HomeRoute` → `HomePage`. You will replace this router.
- [app/lib/router/home_page.dart](../../../app/lib/router/home_page.dart) is a **design playground page** (not a real home) with typography, button, chip, card, and swatch samples. The playground is valuable — preserve it by moving it to `app/lib/design_playground/pages/design_playground_page.dart` and exposing a dev-only link to it from the Feed page (or Profile page dev section). Do **not** delete the playground content.
- [app/lib/router/error_screen.dart](../../../app/lib/router/error_screen.dart) already exists and uses `AppLocalizations`. Keep it but update its "Go home" target from `HomeRoute` to `FeedRoute`.
- [app/lib/router/widgets/](../../../app/lib/router/widgets) — the directory exists but is empty. Leave it alone.
- [app/lib/l10n/app_en.arb](../../../app/lib/l10n/app_en.arb) contains keys for `appTitle`, `homeSubtitle`, `homeVersionLabel`, `initializationFailedTitle`, `retryButton`, `routingErrorTitle`, `goHomeButton`. Per the spec, new page stub titles use hardcoded English strings with `// TODO: l10n` markers. The existing `homeSubtitle`/`homeVersionLabel` keys stay in place because the design playground still uses them.
- [app/test/app_test.dart](../../../app/test/app_test.dart) and [app/test/widget_test.dart](../../../app/test/widget_test.dart) currently assert `find.byType(HomePage)`. These tests will need updating to match the new post-boot landing (the welcome page, because auth stub is `false`).

**Dart MCP tooling:**

- `mcp__dart__analyze_files` with the `app/` root path for static analysis.
- `mcp__dart__run_tests` for running Flutter tests.
- `mcp__dart__pub` for pub commands (e.g. `{"command": "run", "arguments": ["build_runner", "build", "--delete-conflicting-outputs"]}`).
- `mcp__dart__dart_format` for formatting.

If the Dart MCP is not connected, fall back to: `flutter analyze app/`, `flutter test app/`, `cd app && dart run build_runner build --delete-conflicting-outputs`, `dart format app/lib app/test`.

**Package name:** `craftsky_app` (from [app/pubspec.yaml](../../../app/pubspec.yaml)). All imports use `package:craftsky_app/...`.

---

## File Structure

Files to be created (paths relative to repo root):

```
app/lib/
  router/
    route_locations.dart              # NEW — const route path strings
    router.dart                       # REPLACE — full scaffolded router
    router.g.dart                     # regenerated
    app_shell.dart                    # NEW — shell widget
    error_screen.dart                 # KEEP, update go-home target to FeedRoute
    home_page.dart                    # DELETE (moved to design_playground)
  design_playground/
    pages/
      design_playground_page.dart     # NEW — former HomePage contents
  auth/
    pages/
      welcome_page.dart               # NEW
      sign_in_page.dart               # NEW
    providers/
      auth_status_provider.dart       # NEW — stubbed
      auth_status_provider.g.dart     # generated
  onboarding/
    pages/
      onboarding_page.dart            # NEW
    providers/
      onboarding_status_provider.dart # NEW — stubbed
      onboarding_status_provider.g.dart # generated
  feed/
    pages/
      feed_page.dart                  # NEW
  search/
    pages/
      search_page.dart                # NEW
  notifications/
    pages/
      notifications_page.dart         # NEW
  profile/
    pages/
      profile_page.dart               # NEW (own profile, shell branch)
      user_profile_page.dart          # NEW (/profile/:handle, root nav)
      saved_page.dart                 # NEW (shell branch child)
  settings/
    pages/
      settings_page.dart              # NEW

app/test/
  auth/
    auth_status_provider_test.dart    # NEW
    welcome_page_test.dart            # NEW
    sign_in_page_test.dart            # NEW
  onboarding/
    onboarding_status_provider_test.dart # NEW
    onboarding_page_test.dart         # NEW
  feed/
    feed_page_test.dart               # NEW
  search/
    search_page_test.dart             # NEW
  notifications/
    notifications_page_test.dart      # NEW
  profile/
    profile_page_test.dart            # NEW
    user_profile_page_test.dart       # NEW
    saved_page_test.dart              # NEW
  settings/
    settings_page_test.dart           # NEW
  router/
    router_redirect_test.dart         # NEW
  app_test.dart                       # MODIFY — HomePage → WelcomePage
  widget_test.dart                    # MODIFY — HomePage → WelcomePage
```

---

## Chunk 1: Stubbed providers

Start with auth and onboarding providers since the router depends on them.

### Task 1: `authStatusProvider` stub

**Files:**
- Create: `app/lib/auth/providers/auth_status_provider.dart`
- Create: `app/lib/auth/providers/auth_status_provider.g.dart` (generated)
- Test: `app/test/auth/auth_status_provider_test.dart`

- [ ] **Step 1.1: Write the failing test**

Create `app/test/auth/auth_status_provider_test.dart`:

```dart
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('authStatusProvider', () {
    test('defaults to false', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      expect(container.read(authStatusProvider), isFalse);
    });

    test('signIn flips state to true', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(authStatusProvider.notifier).signIn();

      expect(container.read(authStatusProvider), isTrue);
    });

    test('signOut flips state back to false', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(authStatusProvider.notifier).signIn();

      container.read(authStatusProvider.notifier).signOut();

      expect(container.read(authStatusProvider), isFalse);
    });
  });
}
```

- [ ] **Step 1.2: Run the test and confirm it fails**

Preferred: `mcp__dart__run_tests` on `app/test/auth/auth_status_provider_test.dart`.
Fallback: `cd app && flutter test test/auth/auth_status_provider_test.dart`.
Expected: compile error — `auth_status_provider.dart` does not exist.

- [ ] **Step 1.3: Implement the provider**

Create `app/lib/auth/providers/auth_status_provider.dart`:

```dart
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'auth_status_provider.g.dart';

/// Stubbed auth status. Real implementation will be backed by the app-view
/// session token once atproto OAuth is wired up.
///
/// Exposes explicit `signIn` / `signOut` methods rather than a generic
/// setter so call sites read intent-fully (`signIn()` vs `setState(true)`).
@riverpod
class AuthStatus extends _$AuthStatus {
  @override
  bool build() => false;

  void signIn() => state = true;
  void signOut() => state = false;
}
```

- [ ] **Step 1.4: Run codegen**

Preferred: `mcp__dart__pub` with `{"command": "run", "arguments": ["build_runner", "build", "--delete-conflicting-outputs"], "roots": ["app"]}`.
Fallback: `cd app && dart run build_runner build --delete-conflicting-outputs`.
Expected: `auth_status_provider.g.dart` is generated, no errors.

- [ ] **Step 1.5: Run the test and confirm it passes**

Preferred: `mcp__dart__run_tests` on `app/test/auth/auth_status_provider_test.dart`.
Expected: all 3 tests pass.

- [ ] **Step 1.6: Commit**

```bash
git add app/lib/auth/providers/ app/test/auth/auth_status_provider_test.dart
git commit -m "feat(app): add stubbed authStatusProvider"
```

---

### Task 2: `onboardingStatusProvider` stub

**Files:**
- Create: `app/lib/onboarding/providers/onboarding_status_provider.dart`
- Create: `app/lib/onboarding/providers/onboarding_status_provider.g.dart` (generated)
- Test: `app/test/onboarding/onboarding_status_provider_test.dart`

- [ ] **Step 2.1: Write the failing test**

Create `app/test/onboarding/onboarding_status_provider_test.dart`:

```dart
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('onboardingStatusProvider', () {
    test('defaults to false', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      expect(container.read(onboardingStatusProvider), isFalse);
    });

    test('finish flips state to true', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      container.read(onboardingStatusProvider.notifier).finish();

      expect(container.read(onboardingStatusProvider), isTrue);
    });
  });
}
```

- [ ] **Step 2.2: Run the test and confirm it fails**

Expected: compile error — file does not exist.

- [ ] **Step 2.3: Implement the provider**

Create `app/lib/onboarding/providers/onboarding_status_provider.dart`:

```dart
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'onboarding_status_provider.g.dart';

/// Stubbed onboarding completion status. Real implementation will be backed
/// by the user's profile record once onboarding actually persists data.
@riverpod
class OnboardingStatus extends _$OnboardingStatus {
  @override
  bool build() => false;

  void finish() => state = true;
}
```

- [ ] **Step 2.4: Run codegen**

Same command as Step 1.4.
Expected: `onboarding_status_provider.g.dart` is generated.

- [ ] **Step 2.5: Run the test and confirm it passes**

Expected: both tests pass.

- [ ] **Step 2.6: Commit**

```bash
git add app/lib/onboarding/providers/ app/test/onboarding/onboarding_status_provider_test.dart
git commit -m "feat(app): add stubbed onboardingStatusProvider"
```

---

## Chunk 2: Page stubs

All page stubs follow the same shape and have similar tests. Group them so commits stay small but focused.

### Task 3: Route locations constants

**Files:**
- Create: `app/lib/router/route_locations.dart`

- [ ] **Step 3.1: Create the file**

```dart
/// Canonical route path strings. Both the router definitions and the redirect
/// logic reference these so the two can't drift.
class RouteLocations {
  RouteLocations._();

  static const welcome = '/welcome';
  static const signIn = '/sign-in';
  static const onboarding = '/onboarding';
  static const feed = '/feed';
  // Alias: the post-auth home landing. Keep as a const reference to `feed`
  // so renaming the branch in one place updates both usages.
  static const home = feed;
  static const search = '/search';
  static const notifications = '/notifications';
  static const profile = '/profile';
  static const savedChild = 'saved';
  static const settings = '/settings';
}
```

- [ ] **Step 3.2: Commit**

```bash
git add app/lib/router/route_locations.dart
git commit -m "feat(app): add RouteLocations constants"
```

---

### Task 4: Simple page stubs (Feed, Search, Notifications, Saved, UserProfile, Settings — the ones without interactive stubs)

Each of these is a `ConsumerWidget` with `Scaffold(appBar: AppBar(title: Text(<name>)), body: Center(child: Text(<name>)))`. For brevity this task bundles all six page files + their tests into one sequence.

**Files:**
- Create: `app/lib/feed/pages/feed_page.dart`
- Create: `app/lib/search/pages/search_page.dart`
- Create: `app/lib/notifications/pages/notifications_page.dart`
- Create: `app/lib/profile/pages/saved_page.dart`
- Create: `app/lib/profile/pages/user_profile_page.dart`
- Test: `app/test/feed/feed_page_test.dart`
- Test: `app/test/search/search_page_test.dart`
- Test: `app/test/notifications/notifications_page_test.dart`
- Test: `app/test/profile/saved_page_test.dart`
- Test: `app/test/profile/user_profile_page_test.dart`

- [ ] **Step 4.1: Write the failing tests**

Use this template for each page — replace `FeedPage` and `'Feed'` per file:

```dart
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('FeedPage renders its title', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: FeedPage()),
      ),
    );
    expect(find.text('Feed'), findsWidgets);
  });
}
```

For `user_profile_page_test.dart` the page takes a `handle` parameter — instantiate with `UserProfilePage(handle: 'alice.bsky.social')` and assert `find.textContaining('alice')`.

- [ ] **Step 4.2: Run all six tests and confirm they fail**

Preferred: `mcp__dart__run_tests` on `app/test/`.
Expected: compile errors for each — the page files don't exist yet.

- [ ] **Step 4.3: Implement the page stubs**

Create `app/lib/feed/pages/feed_page.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class FeedPage extends ConsumerWidget {
  const FeedPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n — page titles will move to AppLocalizations when real UI lands.
    return Scaffold(
      appBar: AppBar(title: const Text('Feed')),
      body: const Center(child: Text('Feed')),
    );
  }
}
```

Create identical stubs for `SearchPage` ('Search'), `NotificationsPage` ('Notifications'), `SavedPage` ('Saved').

Create `app/lib/profile/pages/user_profile_page.dart`:

```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class UserProfilePage extends ConsumerWidget {
  const UserProfilePage({required this.handle, super.key});

  final String handle;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n
    return Scaffold(
      appBar: AppBar(title: Text('@$handle')),
      body: Center(child: Text('Profile for @$handle')),
    );
  }
}
```

- [ ] **Step 4.4: Run the tests and confirm they pass**

Expected: all five tests pass.

- [ ] **Step 4.5: Commit**

```bash
git add app/lib/feed app/lib/search app/lib/notifications app/lib/profile/pages/saved_page.dart app/lib/profile/pages/user_profile_page.dart app/test/feed app/test/search app/test/notifications app/test/profile/saved_page_test.dart app/test/profile/user_profile_page_test.dart
git commit -m "feat(app): add feed, search, notifications, saved, user profile page stubs"
```

---

### Task 5: `SettingsPage` stub (with dev sign-out button)

**Files:**
- Create: `app/lib/settings/pages/settings_page.dart`
- Test: `app/test/settings/settings_page_test.dart`

- [ ] **Step 5.1: Write the failing test**

```dart
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('SettingsPage renders title and sign-out button', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: SettingsPage()),
      ),
    );
    expect(find.text('Settings'), findsWidgets);
    expect(find.widgetWithText(OutlinedButton, 'Sign out (dev)'), findsOneWidget);
  });

  testWidgets('tapping sign-out button flips authStatusProvider to false', (tester) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    container.read(authStatusProvider.notifier).signIn();

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: SettingsPage()),
      ),
    );

    await tester.tap(find.widgetWithText(OutlinedButton, 'Sign out (dev)'));
    await tester.pump();

    expect(container.read(authStatusProvider), isFalse);
  });
}
```

- [ ] **Step 5.2: Run the tests and confirm they fail**

Expected: compile error — file does not exist.

- [ ] **Step 5.3: Implement `SettingsPage`**

Create `app/lib/settings/pages/settings_page.dart`:

```dart
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SettingsPage extends ConsumerWidget {
  const SettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: const Center(child: SettingsPageBody()),
    );
  }
}

class SettingsPageBody extends ConsumerWidget {
  const SettingsPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Settings'),
        const SizedBox(height: 24),
        OutlinedButton(
          onPressed: () => ref.read(authStatusProvider.notifier).signOut(),
          child: const Text('Sign out (dev)'),
        ),
      ],
    );
  }
}
```

- [ ] **Step 5.4: Run the tests and confirm they pass**

Expected: both tests pass.

- [ ] **Step 5.5: Commit**

```bash
git add app/lib/settings app/test/settings
git commit -m "feat(app): add SettingsPage stub with dev sign-out"
```

---

### Task 6: `WelcomePage` stub (with dev auth toggle)

**Files:**
- Create: `app/lib/auth/pages/welcome_page.dart`
- Test: `app/test/auth/welcome_page_test.dart`

- [ ] **Step 6.1: Write the failing test**

```dart
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('WelcomePage renders title and dev auth toggle', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: WelcomePage()),
      ),
    );
    expect(find.text('Welcome'), findsWidgets);
    expect(find.widgetWithText(OutlinedButton, 'Dev: toggle auth'), findsOneWidget);
  });

  testWidgets('tapping the dev toggle flips authStatusProvider to true', (tester) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: WelcomePage()),
      ),
    );

    await tester.tap(find.widgetWithText(OutlinedButton, 'Dev: toggle auth'));
    await tester.pump();

    expect(container.read(authStatusProvider), isTrue);
  });
}
```

- [ ] **Step 6.2: Run the test and confirm it fails**

- [ ] **Step 6.3: Implement `WelcomePage`**

Create `app/lib/auth/pages/welcome_page.dart`:

```dart
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class WelcomePage extends ConsumerWidget {
  const WelcomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Welcome')),
      body: const Center(child: WelcomePageBody()),
    );
  }
}

class WelcomePageBody extends ConsumerWidget {
  const WelcomePageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Welcome'),
        const SizedBox(height: 24),
        ElevatedButton(
          onPressed: () => context.go(RouteLocations.signIn),
          child: const Text('Sign in'),
        ),
        const SizedBox(height: 8),
        TextButton(
          onPressed: () => context.go(RouteLocations.signIn),
          child: const Text('Create account on a PDS'),
        ),
        const SizedBox(height: 32),
        OutlinedButton(
          onPressed: () => ref.read(authStatusProvider.notifier).signIn(),
          child: const Text('Dev: toggle auth'),
        ),
      ],
    );
  }
}
```

Note: the widget uses `context.go(...)` which requires a `GoRouter` ancestor. The **unit test** above wraps `WelcomePage` in a plain `MaterialApp` so the `context.go` calls are never exercised (only the button presence/tap for the dev toggle is tested). Route-level navigation is tested separately in the router redirect test.

- [ ] **Step 6.4: Run the test and confirm it passes**

- [ ] **Step 6.5: Commit**

```bash
git add app/lib/auth/pages/welcome_page.dart app/test/auth/welcome_page_test.dart
git commit -m "feat(app): add WelcomePage stub with dev auth toggle"
```

---

### Task 7: `SignInPage` stub

**Files:**
- Create: `app/lib/auth/pages/sign_in_page.dart`
- Test: `app/test/auth/sign_in_page_test.dart`

- [ ] **Step 7.1: Write the failing test**

```dart
import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('SignInPage renders a handle field and Continue button', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: SignInPage()),
      ),
    );
    expect(find.byType(TextField), findsOneWidget);
    expect(find.widgetWithText(ElevatedButton, 'Continue'), findsOneWidget);
  });

  testWidgets('Continue flips authStatusProvider to true', (tester) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: SignInPage()),
      ),
    );

    await tester.tap(find.widgetWithText(ElevatedButton, 'Continue'));
    await tester.pump();

    expect(container.read(authStatusProvider), isTrue);
  });
}
```

- [ ] **Step 7.2: Run the test and confirm it fails**

- [ ] **Step 7.3: Implement `SignInPage`**

```dart
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SignInPage extends ConsumerWidget {
  const SignInPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Sign in')),
      body: const Padding(
        padding: EdgeInsets.all(24),
        child: SignInPageBody(),
      ),
    );
  }
}

class SignInPageBody extends ConsumerWidget {
  const SignInPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const TextField(
          decoration: InputDecoration(
            labelText: 'Handle',
            hintText: 'alice.bsky.social',
          ),
        ),
        const SizedBox(height: 24),
        ElevatedButton(
          onPressed: () => ref.read(authStatusProvider.notifier).signIn(),
          child: const Text('Continue'),
        ),
      ],
    );
  }
}
```

- [ ] **Step 7.4: Run the test and confirm it passes**

- [ ] **Step 7.5: Commit**

```bash
git add app/lib/auth/pages/sign_in_page.dart app/test/auth/sign_in_page_test.dart
git commit -m "feat(app): add SignInPage stub"
```

---

### Task 8: `OnboardingPage` stub

**Files:**
- Create: `app/lib/onboarding/pages/onboarding_page.dart`
- Test: `app/test/onboarding/onboarding_page_test.dart`

- [ ] **Step 8.1: Write the failing test**

```dart
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('OnboardingPage renders Finish button', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: OnboardingPage()),
      ),
    );
    expect(find.widgetWithText(ElevatedButton, 'Finish'), findsOneWidget);
  });

  testWidgets('Finish flips onboardingStatusProvider to true', (tester) async {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: const MaterialApp(home: OnboardingPage()),
      ),
    );

    await tester.tap(find.widgetWithText(ElevatedButton, 'Finish'));
    await tester.pump();

    expect(container.read(onboardingStatusProvider), isTrue);
  });
}
```

- [ ] **Step 8.2: Run the test and confirm it fails**

- [ ] **Step 8.3: Implement `OnboardingPage`**

```dart
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class OnboardingPage extends ConsumerWidget {
  const OnboardingPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Onboarding')),
      body: const Center(child: OnboardingPageBody()),
    );
  }
}

class OnboardingPageBody extends ConsumerWidget {
  const OnboardingPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Onboarding'),
        const SizedBox(height: 24),
        ElevatedButton(
          onPressed: () => ref.read(onboardingStatusProvider.notifier).finish(),
          child: const Text('Finish'),
        ),
      ],
    );
  }
}
```

- [ ] **Step 8.4: Run the test and confirm it passes**

- [ ] **Step 8.5: Commit**

```bash
git add app/lib/onboarding/pages/onboarding_page.dart app/test/onboarding/onboarding_page_test.dart
git commit -m "feat(app): add OnboardingPage stub"
```

---

### Task 9: `ProfilePage` stub (with nav buttons to Settings, Saved, UserProfile)

**Files:**
- Create: `app/lib/profile/pages/profile_page.dart`
- Test: `app/test/profile/profile_page_test.dart`

- [ ] **Step 9.1: Write the failing test**

Because `ProfilePage` uses `context.go(...)`, the test stays narrow — only assert button presence.

```dart
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('ProfilePage renders nav buttons', (tester) async {
    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(home: ProfilePage()),
      ),
    );
    expect(find.widgetWithText(OutlinedButton, 'Settings'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Saved'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Open a user profile'), findsOneWidget);
  });
}
```

- [ ] **Step 9.2: Run the test and confirm it fails**

- [ ] **Step 9.3: Implement `ProfilePage`**

```dart
import 'package:craftsky_app/router/route_locations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

class ProfilePage extends ConsumerWidget {
  const ProfilePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // TODO: l10n
    return Scaffold(
      appBar: AppBar(title: const Text('Profile')),
      body: const Center(child: ProfilePageBody()),
    );
  }
}

class ProfilePageBody extends StatelessWidget {
  const ProfilePageBody({super.key});

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Text('Profile'),
        const SizedBox(height: 24),
        OutlinedButton(
          onPressed: () => context.go(
            '${RouteLocations.profile}/${RouteLocations.savedChild}',
          ),
          child: const Text('Saved'),
        ),
        const SizedBox(height: 8),
        OutlinedButton(
          onPressed: () => context.go(RouteLocations.settings),
          child: const Text('Settings'),
        ),
        const SizedBox(height: 8),
        OutlinedButton(
          onPressed: () =>
              context.go('${RouteLocations.profile}/alice.bsky.social'),
          child: const Text('Open a user profile'),
        ),
      ],
    );
  }
}
```

- [ ] **Step 9.4: Run the test and confirm it passes**

- [ ] **Step 9.5: Commit**

```bash
git add app/lib/profile/pages/profile_page.dart app/test/profile/profile_page_test.dart
git commit -m "feat(app): add ProfilePage stub with nav to settings, saved, user profile"
```

---

### Task 10: Relocate design playground

**Files:**
- Move: `app/lib/router/home_page.dart` → `app/lib/design_playground/pages/design_playground_page.dart`
- Rename inside moved file: `HomePage` → `DesignPlaygroundPage`

- [ ] **Step 10.1: Move and rename the file**

```bash
mkdir -p app/lib/design_playground/pages
git mv app/lib/router/home_page.dart app/lib/design_playground/pages/design_playground_page.dart
```

Then edit the moved file:
- Rename class `HomePage` → `DesignPlaygroundPage`.
- Keep `HomeHeader`, `PlaygroundSection`, and all `*Sample` classes as-is (they're internal).

- [ ] **Step 10.2: Temporarily tolerate the broken imports**

After this move, `app/lib/router/router.dart`, `app/lib/router/error_screen.dart`, `app/test/widget_test.dart`, and `app/test/app_test.dart` will not compile. They'll all be updated in later tasks. Don't run analyze/tests yet.

- [ ] **Step 10.3: Commit**

```bash
git add app/lib/design_playground app/lib/router/home_page.dart
git commit -m "refactor(app): move HomePage to design_playground as DesignPlaygroundPage

Continues to expose the paper-cutout component playground, just not at
the root route. Router will be reworked to mount the new feature-page
tree in a later commit."
```

---

## Chunk 3: Router and shell

### Task 11: Replace `router.dart` with the scaffolded router

**Files:**
- Modify: `app/lib/router/router.dart` (full replacement)
- Create: `app/lib/router/router.g.dart` (regenerated)
- Modify: `app/lib/router/error_screen.dart` (go-home target change)

- [ ] **Step 11.1: Replace `router.dart`**

```dart
import 'package:craftsky_app/auth/pages/sign_in_page.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/notifications/pages/notifications_page.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/pages/saved_page.dart';
import 'package:craftsky_app/profile/pages/user_profile_page.dart';
import 'package:craftsky_app/router/app_shell.dart';
import 'package:craftsky_app/router/error_screen.dart';
import 'package:craftsky_app/router/route_locations.dart';
import 'package:craftsky_app/search/pages/search_page.dart';
import 'package:craftsky_app/settings/pages/settings_page.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'router.g.dart';

/// Singleton navigator keys. Globals that are recreated on hot reload cause
/// go_router to crash; holding them behind a class means hot reload keeps
/// the same instances.
class _NavigatorKeys {
  _NavigatorKeys._();

  static GlobalKey<NavigatorState>? _rootKey;
  static GlobalKey<NavigatorState>? _feedKey;
  static GlobalKey<NavigatorState>? _searchKey;
  static GlobalKey<NavigatorState>? _notificationsKey;
  static GlobalKey<NavigatorState>? _profileKey;

  static GlobalKey<NavigatorState> get rootNavigatorKey =>
      _rootKey ??= GlobalKey<NavigatorState>(debugLabel: 'rootNavigator');
  static GlobalKey<NavigatorState> get feedNavigatorKey =>
      _feedKey ??= GlobalKey<NavigatorState>(debugLabel: 'feedNavigator');
  static GlobalKey<NavigatorState> get searchNavigatorKey =>
      _searchKey ??= GlobalKey<NavigatorState>(debugLabel: 'searchNavigator');
  static GlobalKey<NavigatorState> get notificationsNavigatorKey =>
      _notificationsKey ??=
          GlobalKey<NavigatorState>(debugLabel: 'notificationsNavigator');
  static GlobalKey<NavigatorState> get profileNavigatorKey =>
      _profileKey ??=
          GlobalKey<NavigatorState>(debugLabel: 'profileNavigator');
}

@riverpod
GoRouter goRouter(Ref ref) {
  return GoRouter(
    initialLocation: RouteLocations.welcome,
    navigatorKey: _NavigatorKeys.rootNavigatorKey,
    debugLogDiagnostics: true,
    redirect: (context, state) {
      final isSignedIn = ref.read(authStatusProvider);
      final isOnboarded = ref.read(onboardingStatusProvider);

      const unauthenticatedRoutes = [
        RouteLocations.welcome,
        RouteLocations.signIn,
      ];
      const onboardingRoute = RouteLocations.onboarding;

      final loc = state.matchedLocation;

      if (!isSignedIn && !unauthenticatedRoutes.contains(loc)) {
        return RouteLocations.welcome;
      }
      if (isSignedIn && !isOnboarded && loc != onboardingRoute) {
        return onboardingRoute;
      }
      if (isSignedIn && isOnboarded &&
          (unauthenticatedRoutes.contains(loc) || loc == onboardingRoute)) {
        return RouteLocations.home;
      }
      return null;
    },
    routes: $appRoutes,
    errorBuilder: (context, state) =>
        ErrorScreen(error: state.error ?? 'Unknown routing error'),
  );
}

// --- Shell route -----------------------------------------------------------

@TypedStatefulShellRoute<AppShellRoute>(
  branches: [
    TypedStatefulShellBranch<FeedBranch>(
      routes: [
        TypedGoRoute<FeedRoute>(path: RouteLocations.feed, name: 'feed'),
      ],
    ),
    TypedStatefulShellBranch<SearchBranch>(
      routes: [
        TypedGoRoute<SearchRoute>(path: RouteLocations.search, name: 'search'),
      ],
    ),
    TypedStatefulShellBranch<NotificationsBranch>(
      routes: [
        TypedGoRoute<NotificationsRoute>(
          path: RouteLocations.notifications,
          name: 'notifications',
        ),
      ],
    ),
    TypedStatefulShellBranch<ProfileBranch>(
      routes: [
        TypedGoRoute<ProfileRoute>(
          path: RouteLocations.profile,
          name: 'profile',
          routes: [
            TypedGoRoute<SavedRoute>(
              path: RouteLocations.savedChild,
              name: 'saved',
            ),
          ],
        ),
      ],
    ),
  ],
)
class AppShellRoute extends StatefulShellRouteData {
  const AppShellRoute();

  @override
  Widget builder(
    BuildContext context,
    GoRouterState state,
    StatefulNavigationShell navigationShell,
  ) {
    return AppShell(navigationShell: navigationShell);
  }
}

class FeedBranch extends StatefulShellBranchData {
  const FeedBranch();
  static final GlobalKey<NavigatorState> $navigatorKey =
      _NavigatorKeys.feedNavigatorKey;
}

class SearchBranch extends StatefulShellBranchData {
  const SearchBranch();
  static final GlobalKey<NavigatorState> $navigatorKey =
      _NavigatorKeys.searchNavigatorKey;
}

class NotificationsBranch extends StatefulShellBranchData {
  const NotificationsBranch();
  static final GlobalKey<NavigatorState> $navigatorKey =
      _NavigatorKeys.notificationsNavigatorKey;
}

class ProfileBranch extends StatefulShellBranchData {
  const ProfileBranch();
  static final GlobalKey<NavigatorState> $navigatorKey =
      _NavigatorKeys.profileNavigatorKey;
}

class FeedRoute extends GoRouteData with $FeedRoute {
  const FeedRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) => const FeedPage();
}

class SearchRoute extends GoRouteData with $SearchRoute {
  const SearchRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const SearchPage();
}

class NotificationsRoute extends GoRouteData with $NotificationsRoute {
  const NotificationsRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const NotificationsPage();
}

class ProfileRoute extends GoRouteData with $ProfileRoute {
  const ProfileRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const ProfilePage();
}

class SavedRoute extends GoRouteData with $SavedRoute {
  const SavedRoute();
  @override
  Widget build(BuildContext context, GoRouterState state) => const SavedPage();
}

// --- Root-navigator routes (push over the shell) ---------------------------

@TypedGoRoute<WelcomeRoute>(path: RouteLocations.welcome, name: 'welcome')
class WelcomeRoute extends GoRouteData with $WelcomeRoute {
  const WelcomeRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const WelcomePage();
}

@TypedGoRoute<SignInRoute>(path: RouteLocations.signIn, name: 'sign-in')
class SignInRoute extends GoRouteData with $SignInRoute {
  const SignInRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) => const SignInPage();
}

@TypedGoRoute<OnboardingRoute>(
  path: RouteLocations.onboarding,
  name: 'onboarding',
)
class OnboardingRoute extends GoRouteData with $OnboardingRoute {
  const OnboardingRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const OnboardingPage();
}

@TypedGoRoute<SettingsRoute>(path: RouteLocations.settings, name: 'settings')
class SettingsRoute extends GoRouteData with $SettingsRoute {
  const SettingsRoute();

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      const SettingsPage();
}

@TypedGoRoute<UserProfileRoute>(
  path: '${RouteLocations.profile}/:handle',
  name: 'user-profile',
)
class UserProfileRoute extends GoRouteData with $UserProfileRoute {
  const UserProfileRoute({required this.handle});

  static final GlobalKey<NavigatorState> $parentNavigatorKey =
      _NavigatorKeys.rootNavigatorKey;

  final String handle;

  @override
  Widget build(BuildContext context, GoRouterState state) =>
      UserProfilePage(handle: handle);
}

extension GoRouterExtension on GoRouter {
  /// Pops any existing stack and replaces the current location.
  void clearStackAndNavigate(String location) {
    while (canPop()) {
      pop();
    }
    pushReplacement(location);
  }
}
```

Note: `/profile/:handle` and `/profile` share a prefix — go_router resolves `/profile` exactly to `ProfileRoute` (shell) and `/profile/<anything-else>` to `UserProfileRoute` (root navigator). `/profile/saved` matches the `SavedRoute` child of `ProfileRoute` because child routes are attempted before sibling parameterized routes. If this ordering breaks at runtime (verify in Step 11.3), the fix is to move `UserProfileRoute` under a different prefix (e.g. `/user/:handle`) — note it as an issue and pause.

- [ ] **Step 11.2: Update `error_screen.dart`**

Change `const HomeRoute().go(context)` → `const FeedRoute().go(context)`. Add `import 'package:craftsky_app/router/router.dart';` if not already present.

- [ ] **Step 11.3: Run codegen**

Preferred: `mcp__dart__pub` with `{"command": "run", "arguments": ["build_runner", "build", "--delete-conflicting-outputs"], "roots": ["app"]}`.
Expected: `router.g.dart` regenerates with symbols for all the new routes, no errors.

- [ ] **Step 11.4: Run analyze**

Preferred: `mcp__dart__analyze_files` on `app/`.
Expected: the codebase compiles. `AppShell` import is unresolved — that's fixed in Task 12. Do not commit yet.

---

### Task 12: Implement `AppShell`

**Files:**
- Create: `app/lib/router/app_shell.dart`

- [ ] **Step 12.1: Implement the shell**

```dart
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

/// Paired icon + label spec for a shell branch destination.
class _DestinationSpec {
  const _DestinationSpec({
    required this.icon,
    required this.selectedIcon,
    required this.label,
  });

  final IconData icon;
  final IconData selectedIcon;
  final String label;
}

const _destinations = <_DestinationSpec>[
  _DestinationSpec(
    icon: Icons.home_outlined,
    selectedIcon: Icons.home,
    label: 'Feed',
  ),
  _DestinationSpec(
    icon: Icons.search_outlined,
    selectedIcon: Icons.search,
    label: 'Search',
  ),
  _DestinationSpec(
    icon: Icons.notifications_outlined,
    selectedIcon: Icons.notifications,
    label: 'Notifications',
  ),
  _DestinationSpec(
    icon: Icons.person_outline,
    selectedIcon: Icons.person,
    label: 'Profile',
  ),
];

class AppShell extends ConsumerWidget {
  const AppShell({required this.navigationShell, super.key});

  final StatefulNavigationShell navigationShell;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final formFactor = FormFactorWidget.of(context);

    if (formFactor.isLarge) {
      return Scaffold(
        body: Row(
          children: [
            _ShellNavigationRail(
              selectedIndex: navigationShell.currentIndex,
              onDestinationSelected: (i) => _goBranch(i),
            ),
            const VerticalDivider(thickness: 1, width: 1),
            Expanded(child: navigationShell),
          ],
        ),
      );
    }

    return Scaffold(
      body: navigationShell,
      bottomNavigationBar: _ShellNavigationBar(
        selectedIndex: navigationShell.currentIndex,
        onDestinationSelected: (i) => _goBranch(i),
      ),
    );
  }

  void _goBranch(int index) {
    navigationShell.goBranch(
      index,
      initialLocation: index == navigationShell.currentIndex,
    );
  }
}

class _ShellNavigationBar extends StatelessWidget {
  const _ShellNavigationBar({
    required this.selectedIndex,
    required this.onDestinationSelected,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;

  @override
  Widget build(BuildContext context) {
    return NavigationBar(
      selectedIndex: selectedIndex,
      onDestinationSelected: onDestinationSelected,
      destinations: [
        for (final d in _destinations)
          NavigationDestination(
            icon: Icon(d.icon),
            selectedIcon: Icon(d.selectedIcon),
            label: d.label,
          ),
      ],
    );
  }
}

class _ShellNavigationRail extends StatelessWidget {
  const _ShellNavigationRail({
    required this.selectedIndex,
    required this.onDestinationSelected,
  });

  final int selectedIndex;
  final ValueChanged<int> onDestinationSelected;

  @override
  Widget build(BuildContext context) {
    return NavigationRail(
      selectedIndex: selectedIndex,
      onDestinationSelected: onDestinationSelected,
      labelType: NavigationRailLabelType.all,
      destinations: [
        for (final d in _destinations)
          NavigationRailDestination(
            icon: Icon(d.icon),
            selectedIcon: Icon(d.selectedIcon),
            label: Text(d.label),
          ),
      ],
    );
  }
}
```

- [ ] **Step 12.2: Run analyze**

Preferred: `mcp__dart__analyze_files` on `app/`.
Expected: no errors, no warnings.

- [ ] **Step 12.3: Format**

Preferred: `mcp__dart__dart_format` on `app/lib` and `app/test`.
Fallback: `dart format app/lib app/test`.

- [ ] **Step 12.4: Commit router + shell together**

```bash
git add app/lib/router app/test
git commit -m "feat(app): scaffold full navigation router and AppShell

Replaces single-route router with StatefulShellRoute (feed, search,
notifications, profile) plus root-navigator routes for welcome,
sign-in, onboarding, settings, and user-profile. Redirects driven by
stubbed auth and onboarding providers. AppShell provides dual
NavigationBar/NavigationRail layout keyed off FormFactorWidget."
```

---

## Chunk 4: Update existing tests & router redirect test

### Task 13: Update `app_test.dart` and `widget_test.dart`

**Files:**
- Modify: `app/test/app_test.dart`
- Modify: `app/test/widget_test.dart`

Current assertions reference `HomePage` which no longer exists in `router/`. Post-boot, the app lands on `WelcomePage` because `authStatusProvider` stub is `false`.

- [ ] **Step 13.1: Modify `widget_test.dart`**

Replace `HomePage` assertions with `WelcomePage`. Remove `'Scaffold ready'` and `'v1.0.0'` assertions — those came from the design-playground `HomePage` and are no longer on the post-boot landing.

```dart
import 'package:craftsky_app/app.dart';
import 'package:craftsky_app/app_dependencies.dart';
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:pub_semver/pub_semver.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  late SharedPreferences prefs;

  setUp(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    SharedPreferences.setMockInitialValues({});
    prefs = await SharedPreferences.getInstance();
  });

  AppDependencies stubDeps() => AppDependencies(
    packageInfo: PackageInfo(
      appName: 'craftsky_app',
      packageName: 'social.craftsky.app',
      version: '1.0.0',
      buildNumber: '1',
    ),
    deviceInfo: CraftskyDeviceInfo(
      platform: 'Test',
      deviceId: 'test',
      model: 'test',
      brand: 'test',
      osVersion: '0',
    ),
    sharedPreferences: prefs,
    appVersion: Version.parse('1.0.0'),
  );

  testWidgets('App boots unauthenticated and lands on WelcomePage',
      (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appDependenciesProvider.overrideWith((ref) async => stubDeps()),
        ],
        child: const App(),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.byType(WelcomePage), findsOneWidget);
  });
}
```

- [ ] **Step 13.2: Modify `app_test.dart`**

Replace the two `HomePage` references (imports + `find.byType`) with `WelcomePage`. Everything else in that file (loading, error, retry, log-once) stays the same — the retry test asserts recovery to the post-boot landing, which is now `WelcomePage`.

Change these two spots:

```dart
// top imports
import 'package:craftsky_app/auth/pages/welcome_page.dart';
// (remove) import 'package:craftsky_app/router/home_page.dart';
```

```dart
// inside 'retry invalidates' test
expect(find.byType(WelcomePage), findsOneWidget);
// and
expect(find.byType(HomePage), findsNothing);  // remove this line entirely
```

```dart
// inside 'loading state' test
expect(find.byType(HomePage), findsNothing);
// → change to:
expect(find.byType(WelcomePage), findsNothing);
```

- [ ] **Step 13.3: Run both tests**

Preferred: `mcp__dart__run_tests` on `app/test/app_test.dart` and `app/test/widget_test.dart`.
Expected: all pass.

- [ ] **Step 13.4: Commit**

```bash
git add app/test/app_test.dart app/test/widget_test.dart
git commit -m "test(app): update app tests for new welcome landing"
```

---

### Task 14: Router redirect test

**Files:**
- Create: `app/test/router/router_redirect_test.dart`

- [ ] **Step 14.1: Write the redirect test**

```dart
import 'package:craftsky_app/auth/pages/welcome_page.dart';
import 'package:craftsky_app/auth/providers/auth_status_provider.dart';
import 'package:craftsky_app/feed/pages/feed_page.dart';
import 'package:craftsky_app/onboarding/pages/onboarding_page.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/form_factor.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

Future<void> _pumpApp(WidgetTester tester, ProviderContainer container) async {
  final router = container.read(goRouterProvider);
  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        routerConfig: router,
        builder: (context, child) =>
            FormFactorWidget(child: child ?? const SizedBox.shrink()),
      ),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  group('router redirect', () {
    testWidgets('unauthenticated user lands on WelcomePage', (tester) async {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      await _pumpApp(tester, container);

      expect(find.byType(WelcomePage), findsOneWidget);
    });

    testWidgets('authed but not onboarded → OnboardingPage', (tester) async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(authStatusProvider.notifier).signIn();

      await _pumpApp(tester, container);

      expect(find.byType(OnboardingPage), findsOneWidget);
    });

    testWidgets('authed and onboarded → FeedPage', (tester) async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      container.read(authStatusProvider.notifier).signIn();
      container.read(onboardingStatusProvider.notifier).finish();

      await _pumpApp(tester, container);

      expect(find.byType(FeedPage), findsOneWidget);
    });
  });
}
```

Note on the fourth redirect rule (authed+onboarded on `/welcome` → `/feed`): go_router evaluates the redirect on the initial location, so with `initialLocation: '/welcome'` and both stubs `true`, the first redirect fires and the app lands on `/feed`. The third test above covers this.

- [ ] **Step 14.2: Run the test and confirm it passes**

Preferred: `mcp__dart__run_tests` on `app/test/router/router_redirect_test.dart`.
Expected: all 3 tests pass.

- [ ] **Step 14.3: Commit**

```bash
git add app/test/router/router_redirect_test.dart
git commit -m "test(app): add router redirect test"
```

---

## Chunk 5: Verification

### Task 15: Full verification pass

- [ ] **Step 15.1: Run analyze**

Preferred: `mcp__dart__analyze_files` on `app/`.
Fallback: `cd app && flutter analyze`.
Expected: no errors, no warnings. @superpowers:verification-before-completion — if anything is surfaced, fix it before proceeding.

- [ ] **Step 15.2: Run all tests**

Preferred: `mcp__dart__run_tests` on `app/test/`.
Fallback: `cd app && flutter test`.
Expected: every test passes.

- [ ] **Step 15.3: Run format**

Preferred: `mcp__dart__dart_format` on `app/lib` and `app/test`.
Fallback: `dart format app/lib app/test`.
Expected: no files changed (if files do change, stage and amend the most recent commit, or open a formatting commit).

- [ ] **Step 15.4: Launch the app manually and walk the flow**

Use `mcp__dart__launch_app` (or `mcp__flutter-skill__launch_app`) — iOS simulator or macOS is fine.

Walk the success criteria from the spec:

1. App opens at `/welcome`.
2. Tap "Dev: toggle auth" → redirect kicks in → lands on `/onboarding`.
3. Tap "Finish" → redirect kicks in → lands on `/feed`.
4. Tap each tab (Feed, Search, Notifications, Profile) → correct page shows, bottom nav stays visible.
5. On Profile, tap "Saved" → `/profile/saved` pushes, bottom nav stays visible.
6. On Profile, tap "Settings" → `/settings` pushes, bottom nav hidden.
7. On Profile, tap "Open a user profile" → `/profile/alice.bsky.social` pushes, bottom nav hidden.
8. On Settings, tap "Sign out (dev)" → the redirect **does not fire automatically** because `goRouterProvider` only `ref.read`s the auth status (by design — no reactive redirect). Pop back to Profile manually, then trigger any navigation (e.g. tap the Feed tab) and observe the redirect kicks in sending you to `/welcome`. If this manual trigger isn't acceptable UX later, a follow-up can add a `ref.listen(authStatusProvider)` inside the router or a shell-level listener — out of scope for this scaffold.
9. Resize the window (macOS / web) past the `FormFactor.laptop` breakpoint → layout switches from bottom nav to left-rail.

If any step fails, note the specific failing step and fix before proceeding.

- [ ] **Step 15.5: Commit anything outstanding**

If format or manual-walk revealed fixes, commit them with a focused message. Otherwise, skip.

---

## Done criteria

1. `flutter analyze app/` produces zero errors and zero warnings.
2. `flutter test app/` passes every test.
3. The manual walkthrough in Step 15.4 completes end-to-end without issues.
4. Every stub file lives in its feature folder per the File Organization section of the spec.
5. The design playground is reachable at `app/lib/design_playground/pages/design_playground_page.dart` (kept, not deleted).
