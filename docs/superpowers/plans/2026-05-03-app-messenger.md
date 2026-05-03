# App Messenger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace direct `ScaffoldMessenger` access with a small swappable `AppMessenger` interface (info/warning/error + optional action), surfaced through a `BuildContext` extension and a `MessengerScope` `InheritedWidget`. Info auto-dismisses; warning/error stay until the user dismisses them. Migrate the four existing direct consumers.

**Architecture:** A stateless `AppMessenger` interface lives in `app/lib/shared/messaging/`. The default `ScaffoldMessengerImpl` holds a `GlobalKey<ScaffoldMessengerState>` attached to every `MaterialApp.scaffoldMessengerKey` in [app.dart](app/lib/app.dart) and routes calls through that key. Consumers reach the messenger via `context.showInfo/showWarning/showError(...)`, which resolves `MessengerScope.of(context)`. No state-management dependency for this feature; `MessengerScope` is the same `InheritedWidget` shape Flutter uses for `Theme`, `MediaQuery`, and `ScaffoldMessenger` itself. Tests override by wrapping in a different scope.

**Tech Stack:** Flutter (Material), `dart_mappable` ^4.6 for the `MessageAction` value object, `flutter_test` for tests. No new dependencies.

**Reference:** [docs/superpowers/specs/2026-05-03-app-messenger-design.md](docs/superpowers/specs/2026-05-03-app-messenger-design.md)

**All paths in this plan are relative to the repo root.** All `flutter`/`dart` commands run from `app/`. Where this plan says "run `flutter test ...`" you may use the equivalent `mcp__dart__run_tests` MCP call instead.

---

## Task 1: Add the `MessageAction` value object

**Files:**
- Create: `app/lib/shared/messaging/message_action.dart`
- Create (generated): `app/lib/shared/messaging/message_action.mapper.dart`

- [ ] **Step 1: Write `MessageAction`**

Create `app/lib/shared/messaging/message_action.dart`:

```dart
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'message_action.mapper.dart';

/// An optional action button shown alongside a message dispatched through
/// [AppMessenger]. `dismissOnTap` controls whether tapping the button
/// dismisses the message — defaults to `true`, matching Material's
/// `SnackBarAction`. Set it to `false` for actions whose effect should
/// leave the message in place (e.g. a "Retry" that triggers an async
/// operation and lets the same message reflect the next outcome).
@MappableClass()
class MessageAction with MessageActionMappable {
  const MessageAction({
    required this.label,
    required this.onPressed,
    this.dismissOnTap = true,
  });

  final String label;
  final VoidCallback onPressed;
  final bool dismissOnTap;
}
```

- [ ] **Step 2: Run codegen**

From `app/`:

```bash
dart run build_runner build --delete-conflicting-outputs
```

Expected: `Succeeded after Xs with Y outputs` (one of which is `message_action.mapper.dart`).

- [ ] **Step 3: Verify it analyses cleanly**

From `app/`:

```bash
flutter analyze lib/shared/messaging/
```

Expected: `No issues found!`

- [ ] **Step 4: Commit**

```bash
git add app/lib/shared/messaging/message_action.dart app/lib/shared/messaging/message_action.mapper.dart
git commit -m "feat(app): add MessageAction value object"
```

---

## Task 2: Define the `AppMessenger` interface

**Files:**
- Create: `app/lib/shared/messaging/app_messenger.dart`

- [ ] **Step 1: Write the interface**

Create `app/lib/shared/messaging/app_messenger.dart`:

```dart
import 'package:craftsky_app/shared/messaging/message_action.dart';

/// Dispatches semantic messages (snackbars in the default impl) to the user.
///
/// Three severities, three lifetime policies:
/// - [info] auto-dismisses after a short timeout.
/// - [warning] stays until the user dismisses it.
/// - [error] stays until the user dismisses it.
///
/// Each method accepts an optional [MessageAction] (e.g. "Retry"). Callers
/// reach an `AppMessenger` through `MessengerScope.of(context)`, or via the
/// `BuildContext` extension `context.showInfo(...)` etc.
abstract interface class AppMessenger {
  /// Shows a transient informational message. Auto-dismisses after ~4s.
  void info(String message, {MessageAction? action});

  /// Shows a sticky warning. Stays until the user dismisses it (close icon,
  /// swipe, action tap, or a subsequent message).
  void warning(String message, {MessageAction? action});

  /// Shows a sticky error. Same dismissal model as [warning].
  void error(String message, {MessageAction? action});

  /// Dismisses the currently visible message, if any. No-op otherwise.
  void dismiss();
}
```

- [ ] **Step 2: Verify it analyses cleanly**

From `app/`:

```bash
flutter analyze lib/shared/messaging/app_messenger.dart
```

Expected: `No issues found!`

- [ ] **Step 3: Commit**

```bash
git add app/lib/shared/messaging/app_messenger.dart
git commit -m "feat(app): define AppMessenger interface"
```

---

## Task 3: Add the `MessengerScope` `InheritedWidget`

**Files:**
- Create: `app/lib/shared/messaging/messenger_scope.dart`
- Create: `app/test/shared/messaging/messenger_scope_test.dart`

- [ ] **Step 1: Write the failing test**

Create `app/test/shared/messaging/messenger_scope_test.dart`:

```dart
import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class _RecordingMessenger implements AppMessenger {
  AppMessenger? lastResolved;
  String? lastMethod;
  String? lastMessage;

  @override
  void info(String message, {MessageAction? action}) {
    lastMethod = 'info';
    lastMessage = message;
  }

  @override
  void warning(String message, {MessageAction? action}) {
    lastMethod = 'warning';
    lastMessage = message;
  }

  @override
  void error(String message, {MessageAction? action}) {
    lastMethod = 'error';
    lastMessage = message;
  }

  @override
  void dismiss() {
    lastMethod = 'dismiss';
  }
}

void main() {
  testWidgets('MessengerScope.of returns the messenger from the nearest scope', (
    tester,
  ) async {
    final messenger = _RecordingMessenger();
    AppMessenger? resolved;

    await tester.pumpWidget(
      MessengerScope(
        messenger: messenger,
        child: Builder(
          builder: (context) {
            resolved = MessengerScope.of(context);
            return const SizedBox();
          },
        ),
      ),
    );

    expect(resolved, same(messenger));
  });

  testWidgets('MessengerScope.of asserts when no scope is present', (
    tester,
  ) async {
    await tester.pumpWidget(
      Builder(
        builder: (context) {
          expect(() => MessengerScope.of(context), throwsAssertionError);
          return const SizedBox();
        },
      ),
    );
  });

  testWidgets('updateShouldNotify is true when the messenger reference changes', (
    tester,
  ) async {
    final messengerA = _RecordingMessenger();
    final messengerB = _RecordingMessenger();
    var rebuildCount = 0;

    Widget build(AppMessenger m) => MessengerScope(
      messenger: m,
      child: Builder(
        builder: (context) {
          MessengerScope.of(context);
          rebuildCount++;
          return const SizedBox();
        },
      ),
    );

    await tester.pumpWidget(build(messengerA));
    expect(rebuildCount, 1);

    await tester.pumpWidget(build(messengerB));
    expect(rebuildCount, 2);
  });
}
```

- [ ] **Step 2: Run the test to verify it fails**

From `app/`:

```bash
flutter test test/shared/messaging/messenger_scope_test.dart
```

Expected: failure — `messenger_scope.dart` does not exist yet, so the import fails to resolve.

- [ ] **Step 3: Write the implementation**

Create `app/lib/shared/messaging/messenger_scope.dart`:

```dart
import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:flutter/widgets.dart';

/// Provides an [AppMessenger] to the widget subtree. Mirrors how Flutter's
/// own `Theme`, `MediaQuery`, and `ScaffoldMessenger` are provided.
///
/// Tests override the messenger by wrapping the widget under test in a
/// `MessengerScope` whose [messenger] is a recording fake.
class MessengerScope extends InheritedWidget {
  const MessengerScope({
    required this.messenger,
    required super.child,
    super.key,
  });

  final AppMessenger messenger;

  static AppMessenger of(BuildContext context) {
    final scope = context
        .dependOnInheritedWidgetOfExactType<MessengerScope>();
    assert(
      scope != null,
      'MessengerScope.of() called with no MessengerScope ancestor.',
    );
    return scope!.messenger;
  }

  @override
  bool updateShouldNotify(MessengerScope old) => messenger != old.messenger;
}
```

- [ ] **Step 4: Run the test to verify it passes**

From `app/`:

```bash
flutter test test/shared/messaging/messenger_scope_test.dart
```

Expected: all three tests pass.

- [ ] **Step 5: Commit**

```bash
git add app/lib/shared/messaging/messenger_scope.dart app/test/shared/messaging/messenger_scope_test.dart
git commit -m "feat(app): add MessengerScope InheritedWidget"
```

---

## Task 4: Add the `messengerDismiss` localised string

**Files:**
- Modify: `app/lib/l10n/app_en.arb`

- [ ] **Step 1: Add the string**

Insert into `app/lib/l10n/app_en.arb` (alphabetical placement isn't enforced — add it next to the other shared strings near the top of the file, e.g. after `retryButton`):

```json
  "messengerDismiss": "Dismiss",
  "@messengerDismiss": { "description": "Semantics label and tooltip on the close icon shown on sticky warning/error messages dispatched via AppMessenger." },
```

(Don't forget the trailing comma on the previous entry if you're inserting in the middle.)

- [ ] **Step 2: Regenerate the AppLocalizations bindings**

The project uses Flutter's gen-l10n with `generate: true` in `pubspec.yaml`. Regenerate:

```bash
flutter pub get
```

(`flutter pub get` triggers the gen-l10n pipeline because of the `flutter: generate: true` flag.)

Expected: `app/lib/l10n/generated/app_localizations.dart` now contains a `messengerDismiss` getter on `AppLocalizations`.

- [ ] **Step 3: Verify it analyses cleanly**

From `app/`:

```bash
flutter analyze lib/l10n/
```

Expected: `No issues found!`

- [ ] **Step 4: Commit**

```bash
git add app/lib/l10n/app_en.arb app/lib/l10n/generated/app_localizations.dart app/lib/l10n/generated/app_localizations_en.dart
git commit -m "feat(app): add messengerDismiss l10n string"
```

(If your codegen produces additional generated locale files, include them. `git status` will tell you what changed.)

---

## Task 5: Add the `CraftskySnackBarContent` widget

**Files:**
- Create: `app/lib/shared/messaging/widgets/craftsky_snack_bar.dart`
- Create: `app/test/shared/messaging/widgets/craftsky_snack_bar_test.dart`

- [ ] **Step 1: Write the failing tests**

Create `app/test/shared/messaging/widgets/craftsky_snack_bar_test.dart`:

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Widget _wrap(Widget child) => MaterialApp(
  theme: AppTheme.lightThemeData,
  localizationsDelegates: AppLocalizations.localizationsDelegates,
  supportedLocales: AppLocalizations.supportedLocales,
  home: Scaffold(body: child),
);

void main() {
  group('CraftskySnackBarContent', () {
    testWidgets('info renders info_outline icon and no close icon', (
      tester,
    ) async {
      await tester.pumpWidget(
        _wrap(
          const CraftskySnackBarContent(
            severity: MessageSeverity.info,
            message: 'Saved',
          ),
        ),
      );

      expect(find.byIcon(Icons.info_outline), findsOneWidget);
      expect(find.byIcon(Icons.close), findsNothing);
      expect(find.text('Saved'), findsOneWidget);
    });

    testWidgets('warning renders warning_amber_rounded and a close icon', (
      tester,
    ) async {
      var dismissed = false;
      await tester.pumpWidget(
        _wrap(
          CraftskySnackBarContent(
            severity: MessageSeverity.warning,
            message: 'Hold up',
            onDismiss: () => dismissed = true,
          ),
        ),
      );

      expect(find.byIcon(Icons.warning_amber_rounded), findsOneWidget);
      expect(find.byIcon(Icons.close), findsOneWidget);

      await tester.tap(find.byIcon(Icons.close));
      expect(dismissed, isTrue);
    });

    testWidgets('error renders error_outline and a close icon', (tester) async {
      await tester.pumpWidget(
        _wrap(
          CraftskySnackBarContent(
            severity: MessageSeverity.error,
            message: 'Boom',
            onDismiss: () {},
          ),
        ),
      );

      expect(find.byIcon(Icons.error_outline), findsOneWidget);
      expect(find.byIcon(Icons.close), findsOneWidget);
    });

    testWidgets('renders an action button when MessageAction is supplied', (
      tester,
    ) async {
      var actionTaps = 0;
      final action = MessageAction(
        label: 'Retry',
        onPressed: () => actionTaps++,
      );

      await tester.pumpWidget(
        _wrap(
          CraftskySnackBarContent(
            severity: MessageSeverity.error,
            message: 'Boom',
            action: action,
            onDismiss: () {},
          ),
        ),
      );

      expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);

      await tester.tap(find.widgetWithText(TextButton, 'Retry'));
      expect(actionTaps, 1);
    });

    testWidgets(
      'info with no action and no onDismiss renders neither button',
      (tester) async {
        await tester.pumpWidget(
          _wrap(
            const CraftskySnackBarContent(
              severity: MessageSeverity.info,
              message: 'Saved',
            ),
          ),
        );

        expect(find.byType(TextButton), findsNothing);
        expect(find.byIcon(Icons.close), findsNothing);
      },
    );
  });
}
```

- [ ] **Step 2: Run tests to verify they fail**

From `app/`:

```bash
flutter test test/shared/messaging/widgets/craftsky_snack_bar_test.dart
```

Expected: failure — the widget doesn't exist yet.

- [ ] **Step 3: Write the implementation**

Create `app/lib/shared/messaging/widgets/craftsky_snack_bar.dart`:

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Severity levels surfaced by [AppMessenger]. Drives the leading icon and
/// (in the impl) the snackbar's lifetime.
enum MessageSeverity { info, warning, error }

/// The visual payload of every message dispatched through `AppMessenger`.
/// Owns the row layout `[icon · text · action? · close?]`.
///
/// `onDismiss` controls whether a trailing close icon is rendered:
/// `null` → no close icon (info messages don't need one because they
/// auto-dismiss); non-null → close icon visible, and tapping it invokes
/// `onDismiss` (which the impl wires to `messengerState.hideCurrentSnackBar()`).
class CraftskySnackBarContent extends StatelessWidget {
  const CraftskySnackBarContent({
    required this.severity,
    required this.message,
    this.action,
    this.onDismiss,
    super.key,
  });

  final MessageSeverity severity;
  final String message;
  final MessageAction? action;
  final VoidCallback? onDismiss;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final semantic = theme.extension<SemanticColorsTheme>()!;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(_iconFor(severity), size: 20, color: _colorFor(severity, semantic)),
        SizedBox(width: spacing.sp3),
        Expanded(
          child: Text(message, style: theme.textTheme.bodyMedium),
        ),
        if (action != null) ...[
          SizedBox(width: spacing.sp2),
          _MessageActionButton(action: action!),
        ],
        if (onDismiss != null) ...[
          SizedBox(width: spacing.sp2),
          IconButton(
            icon: const Icon(Icons.close, size: 20),
            tooltip: l10n.messengerDismiss,
            onPressed: onDismiss,
          ),
        ],
      ],
    );
  }

  static IconData _iconFor(MessageSeverity s) => switch (s) {
    MessageSeverity.info => Icons.info_outline,
    MessageSeverity.warning => Icons.warning_amber_rounded,
    MessageSeverity.error => Icons.error_outline,
  };

  static Color _colorFor(MessageSeverity s, SemanticColorsTheme c) =>
      switch (s) {
        MessageSeverity.info => c.info,
        MessageSeverity.warning => c.warning,
        MessageSeverity.error => c.error,
      };
}

class _MessageActionButton extends StatelessWidget {
  const _MessageActionButton({required this.action});

  final MessageAction action;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return TextButton(
      onPressed: action.onPressed,
      child: Text(action.label, style: theme.textTheme.labelLarge),
    );
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

From `app/`:

```bash
flutter test test/shared/messaging/widgets/craftsky_snack_bar_test.dart
```

Expected: all five tests pass.

- [ ] **Step 5: Commit**

```bash
git add app/lib/shared/messaging/widgets/craftsky_snack_bar.dart app/test/shared/messaging/widgets/craftsky_snack_bar_test.dart
git commit -m "feat(app): add CraftskySnackBarContent layout widget"
```

---

## Task 6: Add `ScaffoldMessengerImpl` and `defaultAppMessenger`

**Files:**
- Create: `app/lib/shared/messaging/scaffold_messenger_impl.dart`
- Create: `app/test/shared/messaging/scaffold_messenger_impl_test.dart`

- [ ] **Step 1: Write the failing tests**

Create `app/test/shared/messaging/scaffold_messenger_impl_test.dart`:

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

Future<({GlobalKey<ScaffoldMessengerState> key, ScaffoldMessengerImpl impl})>
_pumpHarness(WidgetTester tester) async {
  final key = GlobalKey<ScaffoldMessengerState>();
  final impl = ScaffoldMessengerImpl(key);

  await tester.pumpWidget(
    MaterialApp(
      scaffoldMessengerKey: key,
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: const Scaffold(body: SizedBox()),
    ),
  );

  return (key: key, impl: impl);
}

void main() {
  group('ScaffoldMessengerImpl', () {
    testWidgets('info shows a SnackBar with a 4-second duration', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      h.impl.info('Hello');
      await tester.pump(); // schedule the snackbar

      final snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
      expect(snackBar.duration, const Duration(seconds: 4));
      expect(find.text('Hello'), findsOneWidget);
      expect(find.byIcon(Icons.info_outline), findsOneWidget);
      expect(find.byIcon(Icons.close), findsNothing);
    });

    testWidgets('warning shows a SnackBar with sticky duration + close icon', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      h.impl.warning('Watch out');
      await tester.pump();

      final snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
      expect(snackBar.duration, const Duration(days: 365));
      expect(find.byIcon(Icons.warning_amber_rounded), findsOneWidget);
      expect(find.byIcon(Icons.close), findsOneWidget);
    });

    testWidgets('error shows a SnackBar with sticky duration + close icon', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      h.impl.error('Boom');
      await tester.pump();

      final snackBar = tester.widget<SnackBar>(find.byType(SnackBar));
      expect(snackBar.duration, const Duration(days: 365));
      expect(find.byIcon(Icons.error_outline), findsOneWidget);
      expect(find.byIcon(Icons.close), findsOneWidget);
    });

    testWidgets('a second call replaces the first (always-replace policy)', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);

      h.impl.info('First');
      await tester.pump();
      expect(find.text('First'), findsOneWidget);

      h.impl.error('Second');
      await tester.pump();
      // The first should be gone; the second is now showing.
      expect(find.text('First'), findsNothing);
      expect(find.text('Second'), findsOneWidget);
      // Exactly one SnackBar is on screen.
      expect(find.byType(SnackBar), findsOneWidget);
    });

    testWidgets('action onPressed runs and dismisses by default', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      var taps = 0;
      h.impl.error(
        'Boom',
        action: MessageAction(label: 'Retry', onPressed: () => taps++),
      );
      await tester.pump();

      expect(find.widgetWithText(TextButton, 'Retry'), findsOneWidget);

      await tester.tap(find.widgetWithText(TextButton, 'Retry'));
      // Drive the dismiss animation through.
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(taps, 1);
      expect(find.byType(SnackBar), findsNothing);
    });

    testWidgets(
      'action with dismissOnTap: false leaves the snackbar in place',
      (tester) async {
        final h = await _pumpHarness(tester);
        h.impl.error(
          'Boom',
          action: MessageAction(
            label: 'Retry',
            onPressed: () {},
            dismissOnTap: false,
          ),
        );
        await tester.pump();

        await tester.tap(find.widgetWithText(TextButton, 'Retry'));
        await tester.pump();

        // The SnackBar should still be visible.
        expect(find.byType(SnackBar), findsOneWidget);
        expect(find.text('Boom'), findsOneWidget);
      },
    );

    testWidgets('tapping the close icon dismisses the message', (
      tester,
    ) async {
      final h = await _pumpHarness(tester);
      h.impl.error('Boom');
      await tester.pump();

      await tester.tap(find.byIcon(Icons.close));
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(find.byType(SnackBar), findsNothing);
    });

    testWidgets('dismiss() hides the current message', (tester) async {
      final h = await _pumpHarness(tester);
      h.impl.error('Boom');
      await tester.pump();
      expect(find.byType(SnackBar), findsOneWidget);

      h.impl.dismiss();
      await tester.pump();
      await tester.pump(const Duration(seconds: 1));

      expect(find.byType(SnackBar), findsNothing);
    });
  });
}
```

- [ ] **Step 2: Run tests to verify they fail**

From `app/`:

```bash
flutter test test/shared/messaging/scaffold_messenger_impl_test.dart
```

Expected: failure — `scaffold_messenger_impl.dart` does not exist yet.

- [ ] **Step 3: Write the implementation**

Create `app/lib/shared/messaging/scaffold_messenger_impl.dart`:

```dart
import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/widgets/craftsky_snack_bar.dart';
import 'package:flutter/material.dart';

/// The root ScaffoldMessenger key. Wired into every `MaterialApp` in
/// `app/lib/app.dart` so messages always go through the same messenger
/// regardless of which subtree the call site lives in.
final GlobalKey<ScaffoldMessengerState> appScaffoldMessengerKey =
    GlobalKey<ScaffoldMessengerState>();

/// The default production [AppMessenger]. Constructed once and reused —
/// the [GlobalKey] is the only piece of mutable state and Flutter owns it.
/// Not `const` because `GlobalKey()` is not a const expression.
final AppMessenger defaultAppMessenger =
    ScaffoldMessengerImpl(appScaffoldMessengerKey);

/// `AppMessenger` backed by Flutter's `ScaffoldMessenger`. Routes every
/// message through the global [appScaffoldMessengerKey] (never via
/// `ScaffoldMessenger.of(context)`), enforces the always-replace policy
/// (each call clears any current message before showing the new one), and
/// supplies the lifetime semantics declared on [AppMessenger].
class ScaffoldMessengerImpl implements AppMessenger {
  ScaffoldMessengerImpl(this._key);

  final GlobalKey<ScaffoldMessengerState> _key;

  static const Duration _infoDuration = Duration(seconds: 4);
  static const Duration _stickyDuration = Duration(days: 365);

  @override
  void info(String message, {MessageAction? action}) {
    _show(MessageSeverity.info, message, action, _infoDuration, sticky: false);
  }

  @override
  void warning(String message, {MessageAction? action}) {
    _show(
      MessageSeverity.warning,
      message,
      action,
      _stickyDuration,
      sticky: true,
    );
  }

  @override
  void error(String message, {MessageAction? action}) {
    _show(
      MessageSeverity.error,
      message,
      action,
      _stickyDuration,
      sticky: true,
    );
  }

  @override
  void dismiss() {
    _key.currentState?.hideCurrentSnackBar();
  }

  void _show(
    MessageSeverity severity,
    String message,
    MessageAction? action,
    Duration duration, {
    required bool sticky,
  }) {
    final state = _key.currentState;
    if (state == null) return;

    state.clearSnackBars();

    final wrappedAction = action == null
        ? null
        : MessageAction(
            label: action.label,
            dismissOnTap: action.dismissOnTap,
            onPressed: () {
              if (action.dismissOnTap) {
                state.hideCurrentSnackBar();
              }
              action.onPressed();
            },
          );

    state.showSnackBar(
      SnackBar(
        behavior: SnackBarBehavior.floating,
        dismissDirection: DismissDirection.horizontal,
        duration: duration,
        content: CraftskySnackBarContent(
          severity: severity,
          message: message,
          action: wrappedAction,
          onDismiss: sticky ? () => state.hideCurrentSnackBar() : null,
        ),
      ),
    );
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

From `app/`:

```bash
flutter test test/shared/messaging/scaffold_messenger_impl_test.dart
```

Expected: all eight tests pass.

- [ ] **Step 5: Commit**

```bash
git add app/lib/shared/messaging/scaffold_messenger_impl.dart app/test/shared/messaging/scaffold_messenger_impl_test.dart
git commit -m "feat(app): add ScaffoldMessenger-backed AppMessenger impl"
```

---

## Task 7: Add the `BuildContext` extension

**Files:**
- Create: `app/lib/shared/messaging/context_messenger_extension.dart`
- Create: `app/test/shared/messaging/context_messenger_extension_test.dart`

- [ ] **Step 1: Write the failing test**

Create `app/test/shared/messaging/context_messenger_extension_test.dart`:

```dart
import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class _RecordingMessenger implements AppMessenger {
  final calls = <(String severity, String message, MessageAction? action)>[];

  @override
  void info(String m, {MessageAction? action}) =>
      calls.add(('info', m, action));
  @override
  void warning(String m, {MessageAction? action}) =>
      calls.add(('warning', m, action));
  @override
  void error(String m, {MessageAction? action}) =>
      calls.add(('error', m, action));
  @override
  void dismiss() => calls.add(('dismiss', '', null));
}

Future<void> _pumpUnderScope(
  WidgetTester tester,
  AppMessenger messenger,
  void Function(BuildContext context) onContext,
) async {
  await tester.pumpWidget(
    MessengerScope(
      messenger: messenger,
      child: MaterialApp(
        home: Builder(
          builder: (context) {
            onContext(context);
            return const SizedBox();
          },
        ),
      ),
    ),
  );
}

void main() {
  group('AppMessengerX', () {
    testWidgets('showInfo routes to messenger.info', (tester) async {
      final messenger = _RecordingMessenger();
      await _pumpUnderScope(
        tester,
        messenger,
        (c) => c.showInfo('Saved'),
      );
      expect(messenger.calls, [('info', 'Saved', null)]);
    });

    testWidgets('showWarning routes to messenger.warning', (tester) async {
      final messenger = _RecordingMessenger();
      await _pumpUnderScope(
        tester,
        messenger,
        (c) => c.showWarning('Heads up'),
      );
      expect(messenger.calls, [('warning', 'Heads up', null)]);
    });

    testWidgets('showError routes to messenger.error and forwards action', (
      tester,
    ) async {
      final messenger = _RecordingMessenger();
      final action = MessageAction(label: 'Retry', onPressed: () {});
      await _pumpUnderScope(
        tester,
        messenger,
        (c) => c.showError('Boom', action: action),
      );
      expect(messenger.calls.length, 1);
      expect(messenger.calls.first.$1, 'error');
      expect(messenger.calls.first.$2, 'Boom');
      expect(messenger.calls.first.$3, same(action));
    });

    testWidgets('dismissMessage routes to messenger.dismiss', (tester) async {
      final messenger = _RecordingMessenger();
      await _pumpUnderScope(
        tester,
        messenger,
        (c) => c.dismissMessage(),
      );
      expect(messenger.calls, [('dismiss', '', null)]);
    });
  });
}
```

- [ ] **Step 2: Run the test to verify it fails**

From `app/`:

```bash
flutter test test/shared/messaging/context_messenger_extension_test.dart
```

Expected: failure — `context_messenger_extension.dart` does not exist yet.

- [ ] **Step 3: Write the implementation**

Create `app/lib/shared/messaging/context_messenger_extension.dart`:

```dart
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/widgets.dart';

/// Terse wrappers over `MessengerScope.of(context)`. The extension is the
/// preferred call surface for widgets — `context.showInfo('Saved')` is
/// shorter than the equivalent `MessengerScope.of(context).info('Saved')`
/// and reads naturally next to `Theme.of(context)`.
extension AppMessengerX on BuildContext {
  void showInfo(String message, {MessageAction? action}) =>
      MessengerScope.of(this).info(message, action: action);

  void showWarning(String message, {MessageAction? action}) =>
      MessengerScope.of(this).warning(message, action: action);

  void showError(String message, {MessageAction? action}) =>
      MessengerScope.of(this).error(message, action: action);

  void dismissMessage() => MessengerScope.of(this).dismiss();
}
```

- [ ] **Step 4: Run the test to verify it passes**

From `app/`:

```bash
flutter test test/shared/messaging/context_messenger_extension_test.dart
```

Expected: all four tests pass.

- [ ] **Step 5: Commit**

```bash
git add app/lib/shared/messaging/context_messenger_extension.dart app/test/shared/messaging/context_messenger_extension_test.dart
git commit -m "feat(app): add BuildContext.showInfo/showWarning/showError extensions"
```

---

## Task 8: Add the `RecordingMessenger` test fake

**Files:**
- Create: `app/test/fakes/recording_messenger.dart`

- [ ] **Step 1: Write the fake**

Create `app/test/fakes/recording_messenger.dart`:

```dart
import 'package:craftsky_app/shared/messaging/app_messenger.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';

/// Test double for [AppMessenger] that records every call. Wrap the widget
/// under test in `MessengerScope(messenger: RecordingMessenger(), ...)`
/// and assert against [calls].
///
/// Each entry is `(severity, message, action)`. For `dismiss()` the
/// severity is `'dismiss'` and `message` is the empty string.
class RecordingMessenger implements AppMessenger {
  final List<(String severity, String message, MessageAction? action)> calls =
      [];

  @override
  void info(String message, {MessageAction? action}) =>
      calls.add(('info', message, action));

  @override
  void warning(String message, {MessageAction? action}) =>
      calls.add(('warning', message, action));

  @override
  void error(String message, {MessageAction? action}) =>
      calls.add(('error', message, action));

  @override
  void dismiss() => calls.add(('dismiss', '', null));
}
```

- [ ] **Step 2: Verify it analyses cleanly**

From `app/`:

```bash
flutter analyze test/fakes/recording_messenger.dart
```

Expected: `No issues found!`

- [ ] **Step 3: Commit**

```bash
git add app/test/fakes/recording_messenger.dart
git commit -m "test(app): add RecordingMessenger test double"
```

---

## Task 9: Wire `MessengerScope` and `scaffoldMessengerKey` into `app.dart`

**Files:**
- Modify: `app/lib/app.dart`
- Modify: `app/test/app_test.dart` (add a test confirming the scope is reachable from the home subtree)

- [ ] **Step 1: Write the failing test**

The existing `app/test/app_test.dart` heavily stubs `appDependenciesProvider` because the real one calls `PackageInfo.fromPlatform()` / `DeviceInfoPlugin` / `SharedPreferences.getInstance()` (none of which work in a bare test environment). Add a new test inside the existing `group('App initialisation', ...)` block that overrides the provider to a never-completing future so the `_LoadingApp` branch renders cleanly, then asserts `MessengerScope.of(context)` resolves to `defaultAppMessenger`:

```dart
testWidgets(
  'App wires MessengerScope and scaffoldMessengerKey on every MaterialApp',
  (tester) async {
    // Keep appDependenciesProvider in flight forever so we render the
    // _LoadingApp branch, which is the cheapest of the three branches to
    // pump (no router, no theme dependencies that need the full deps).
    final neverComplete = Completer<AppDependencies>();
    addTearDown(neverComplete.complete); // tidy on tear-down

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appDependenciesProvider.overrideWith((ref) => neverComplete.future),
        ],
        child: const App(),
      ),
    );

    final BuildContext context = tester.element(find.byType(MaterialApp));
    expect(MessengerScope.of(context), same(defaultAppMessenger));

    final materialApp = tester.widget<MaterialApp>(find.byType(MaterialApp));
    expect(materialApp.scaffoldMessengerKey, same(appScaffoldMessengerKey));
  },
);
```

Add (or dedupe with) these imports at the top of `app/test/app_test.dart`:

```dart
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
```

(`App`, `appDependenciesProvider`, `ProviderScope`, `Completer`, and `MaterialApp` are already imported by the existing file.)

- [ ] **Step 2: Run the test to verify it fails**

From `app/`:

```bash
flutter test test/app_test.dart --plain-name "App wires MessengerScope"
```

Expected: failure — the imports resolve, but no `MessengerScope` ancestor exists in the current `App` tree, so `MessengerScope.of(context)` hits its assertion.

- [ ] **Step 3: Edit `app/lib/app.dart`**

Add the import near the top:

```dart
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:craftsky_app/shared/messaging/scaffold_messenger_impl.dart';
```

Wrap **each** of the three `MaterialApp` / `MaterialApp.router` constructions in a `MessengerScope` and pass `scaffoldMessengerKey: appScaffoldMessengerKey`. Concretely:

In `_ReadyApp.build` (around line 47), change:

```dart
return MaterialApp.router(
  onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
  // ...
);
```

to:

```dart
return MessengerScope(
  messenger: defaultAppMessenger,
  child: MaterialApp.router(
    scaffoldMessengerKey: appScaffoldMessengerKey,
    onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
    // ... rest unchanged
  ),
);
```

In `_LoadingApp.build` (around line 72), change:

```dart
return const MaterialApp(
  debugShowCheckedModeBanner: false,
  // ...
);
```

The `const` has to come off because `MessengerScope(messenger: defaultAppMessenger, ...)` references a non-const top-level `final`. Update to:

```dart
return MessengerScope(
  messenger: defaultAppMessenger,
  child: MaterialApp(
    scaffoldMessengerKey: appScaffoldMessengerKey,
    debugShowCheckedModeBanner: false,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: const InitializationLoadingScreen(),
  ),
);
```

In `_ErrorApp.build` (around line 88), change:

```dart
return MaterialApp(
  debugShowCheckedModeBanner: false,
  // ...
);
```

to:

```dart
return MessengerScope(
  messenger: defaultAppMessenger,
  child: MaterialApp(
    scaffoldMessengerKey: appScaffoldMessengerKey,
    debugShowCheckedModeBanner: false,
    localizationsDelegates: AppLocalizations.localizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: InitializationErrorScreen(
      error: error,
      onRetry: () => ref.invalidate(appDependenciesProvider),
    ),
  ),
);
```

- [ ] **Step 4: Run the test to verify it passes**

From `app/`:

```bash
flutter test test/app_test.dart
```

Expected: all tests in `app_test.dart` pass, including the new one.

- [ ] **Step 5: Run the full app test suite to make sure nothing else broke**

From `app/`:

```bash
flutter test
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add app/lib/app.dart app/test/app_test.dart
git commit -m "feat(app): wire MessengerScope and scaffoldMessengerKey in App"
```

---

## Task 10: Migrate `clear_image_cache_tile.dart` and update its test

**Files:**
- Modify: `app/lib/settings/widgets/clear_image_cache_tile.dart`
- Modify: `app/test/settings/clear_image_cache_tile_test.dart`

- [ ] **Step 1: Update the test to use `RecordingMessenger`**

Replace the contents of `app/test/settings/clear_image_cache_tile_test.dart` with:

```dart
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/image_cache_fakes.dart';
import '../fakes/recording_messenger.dart';

Future<({FakeBaseCacheManager profile, FakeBaseCacheManager feed, RecordingMessenger messenger})>
_pump(WidgetTester tester, {Object? throwOnEmptyCache}) async {
  final profileFake = FakeBaseCacheManager();
  final feedFake = FakeBaseCacheManager();
  if (throwOnEmptyCache != null) {
    profileFake.throwOnEmptyCache = throwOnEmptyCache;
  }
  final messenger = RecordingMessenger();

  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
        feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
      ],
      child: MessengerScope(
        messenger: messenger,
        child: const MaterialApp(
          home: Scaffold(body: ClearImageCacheTile()),
        ),
      ),
    ),
  );

  return (profile: profileFake, feed: feedFake, messenger: messenger);
}

void main() {
  group('ClearImageCacheTile', () {
    testWidgets('tap calls emptyCache on both managers', (tester) async {
      final h = await _pump(tester);

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(h.profile.emptyCacheCalls, 1);
      expect(h.feed.emptyCacheCalls, 1);
    });

    testWidgets('shows info message when both caches clear', (tester) async {
      final h = await _pump(tester);

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(h.messenger.calls.length, 1);
      expect(h.messenger.calls.first.$1, 'info');
      expect(h.messenger.calls.first.$2, 'Image cache cleared');
    });

    testWidgets('shows error message when a cache fails to clear', (
      tester,
    ) async {
      final h = await _pump(tester, throwOnEmptyCache: StateError('disk full'));

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(h.messenger.calls.length, 1);
      expect(h.messenger.calls.first.$1, 'error');
      expect(h.messenger.calls.first.$2, contains('Could not clear cache'));
    });
  });
}
```

- [ ] **Step 2: Run the test to verify it fails**

From `app/`:

```bash
flutter test test/settings/clear_image_cache_tile_test.dart
```

Expected: failure — the widget still calls `ScaffoldMessenger.of(context)` directly.

- [ ] **Step 3: Migrate the widget**

Replace the contents of `app/lib/settings/widgets/clear_image_cache_tile.dart` with:

```dart
import 'package:craftsky_app/shared/image/clear_image_cache_provider.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Settings tile that empties both image caches. The action is reversible
/// (images re-download on next view) so there is no confirmation dialog.
class ClearImageCacheTile extends ConsumerWidget {
  const ClearImageCacheTile({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(clearImageCacheProvider);

    ref.listen(clearImageCacheProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncData()):
          context.showInfo('Image cache cleared');
        case (AsyncLoading(), AsyncError(:final error)):
          context.showError('Could not clear cache: $error');
        case _:
          break;
      }
    });

    return ListTile(
      leading: const Icon(Icons.cleaning_services_outlined),
      title: const Text('Clear image cache'),
      enabled: state is! AsyncLoading,
      onTap: () => ref.read(clearImageCacheProvider.notifier).clear(),
    );
  }
}
```

- [ ] **Step 4: Run the test to verify it passes**

From `app/`:

```bash
flutter test test/settings/clear_image_cache_tile_test.dart
```

Expected: all three tests pass.

- [ ] **Step 5: Commit**

```bash
git add app/lib/settings/widgets/clear_image_cache_tile.dart app/test/settings/clear_image_cache_tile_test.dart
git commit -m "refactor(app): migrate ClearImageCacheTile to AppMessenger"
```

---

## Task 11: Migrate `sign_in_page.dart`

**Files:**
- Modify: `app/lib/auth/pages/sign_in_page.dart`

- [ ] **Step 1: Edit the page**

Open `app/lib/auth/pages/sign_in_page.dart`. Add the import:

```dart
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
```

Replace the `ref.listen` callback body (around lines 26-36):

```dart
ref.listen(authControllerProvider, (prev, next) {
  switch ((prev, next)) {
    case (AsyncLoading(), AsyncError(:final error)):
      final message = _messageFor(error);
      ScaffoldMessenger.of(context)
        ..clearSnackBars()
        ..showSnackBar(SnackBar(content: Text(message)));
    case _:
      break;
  }
});
```

with:

```dart
ref.listen(authControllerProvider, (prev, next) {
  switch ((prev, next)) {
    case (AsyncLoading(), AsyncError(:final error)):
      context.showError(_messageFor(error));
    case _:
      break;
  }
});
```

(The manual `clearSnackBars()` is no longer needed — the always-replace policy is built into the messenger.)

- [ ] **Step 2: Run the existing sign-in tests to confirm nothing broke**

From `app/`:

```bash
flutter test test/auth/sign_in_page_test.dart
```

Expected: all tests pass. The existing tests assert against `find.text(...)` for the error message; that still works because the message text is rendered inside `CraftskySnackBarContent`.

- [ ] **Step 3: Verify analysis is clean**

From `app/`:

```bash
flutter analyze lib/auth/pages/sign_in_page.dart
```

Expected: `No issues found!`

- [ ] **Step 4: Commit**

```bash
git add app/lib/auth/pages/sign_in_page.dart
git commit -m "refactor(app): migrate SignInPage to AppMessenger"
```

---

## Task 12: Migrate `profile_page.dart`

**Files:**
- Modify: `app/lib/profile/pages/profile_page.dart`

- [ ] **Step 1: Edit the page**

Open `app/lib/profile/pages/profile_page.dart`. Add the import:

```dart
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
```

Replace the two `ScaffoldMessenger.of(context).showSnackBar(...)` calls (around lines 126 and 131) inside the `VisitorProfileActionSet` callbacks:

```dart
return VisitorProfileActionSet(
  isFollowing: false,
  onFollowToggle: () {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(l10n.profileFollowComingSoon)),
    );
  },
  onShare: () {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(l10n.profileShareComingSoon)),
    );
  },
);
```

with:

```dart
return VisitorProfileActionSet(
  isFollowing: false,
  onFollowToggle: () => context.showInfo(l10n.profileFollowComingSoon),
  onShare: () => context.showInfo(l10n.profileShareComingSoon),
);
```

- [ ] **Step 2: Run the existing profile tests**

From `app/`:

```bash
flutter test test/profile/profile_page_test.dart
```

Expected: all tests pass.

- [ ] **Step 3: Verify analysis is clean**

From `app/`:

```bash
flutter analyze lib/profile/pages/profile_page.dart
```

Expected: `No issues found!`

- [ ] **Step 4: Commit**

```bash
git add app/lib/profile/pages/profile_page.dart
git commit -m "refactor(app): migrate ProfilePage to AppMessenger"
```

---

## Task 13: Migrate `edit_profile_dialog.dart`

**Files:**
- Modify: `app/lib/profile/pages/edit_profile_dialog.dart`

- [ ] **Step 1: Edit the dialog**

Open `app/lib/profile/pages/edit_profile_dialog.dart`. Add the import:

```dart
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
```

Replace the snackbar call inside the `ref.listen` callback (around line 292):

```dart
case (AsyncLoading(), AsyncError()):
  ScaffoldMessenger.of(context).showSnackBar(
    SnackBar(content: Text(l10n.editProfileSaveError)),
  );
```

with:

```dart
case (AsyncLoading(), AsyncError()):
  context.showError(l10n.editProfileSaveError);
```

(Note: the error becomes sticky, per the spec — that's intentional. A failed profile save should not silently scroll out of view.)

- [ ] **Step 2: Run the existing edit-profile tests**

From `app/`:

```bash
flutter test test/profile/edit_profile_dialog_test.dart
```

Expected: all tests pass.

- [ ] **Step 3: Verify analysis is clean**

From `app/`:

```bash
flutter analyze lib/profile/pages/edit_profile_dialog.dart
```

Expected: `No issues found!`

- [ ] **Step 4: Commit**

```bash
git add app/lib/profile/pages/edit_profile_dialog.dart
git commit -m "refactor(app): migrate EditProfileDialog to AppMessenger"
```

---

## Task 14: Final verification

**Files:** none (verification + cleanup pass)

- [ ] **Step 1: Confirm there are no remaining direct ScaffoldMessenger consumers**

From the repo root:

```bash
grep -rn "ScaffoldMessenger\.of\|showSnackBar" app/lib
```

Expected: no matches (the only mentions of `ScaffoldMessenger` should be in `app/lib/shared/messaging/scaffold_messenger_impl.dart`'s comments and in `app/lib/app.dart`'s `scaffoldMessengerKey:` argument). If any production code still calls `ScaffoldMessenger.of(...).showSnackBar(...)` or `clearSnackBars()`, migrate it before continuing.

- [ ] **Step 2: Run the full app test suite**

From `app/`:

```bash
flutter test
```

Expected: every test passes.

- [ ] **Step 3: Run static analysis on the whole app package**

From `app/`:

```bash
flutter analyze
```

Expected: `No issues found!`

- [ ] **Step 4: Smoke-test by running the app**

If a simulator/device is available, launch the app (`flutter run` from `app/`, or via the `mcp__dart__launch_app` tool) and exercise:
- The sign-in error path (enter an empty handle and press Continue → sticky error with close icon).
- The Settings → Clear image cache tile (info message that auto-dismisses after ~4s).
- A visitor profile's Follow / Share buttons (info messages).

This step is not gated by a test command — it's a manual confirmation that the visual treatment looks right end-to-end.

- [ ] **Step 5: Final commit if anything was tweaked, then PR**

If steps 1–4 surfaced anything (e.g. an analyse warning, a small style fix in `CraftskySnackBarContent`), commit it as a follow-up. Otherwise no further commit is needed.

The branch is ready for review. Use the `superpowers:finishing-a-development-branch` skill (or follow the project's normal PR workflow) to open the PR.

---

## Implementation notes

- **Codegen**: Only `MessageAction` requires `dart_mappable` codegen. After Task 1 you should not need to run `build_runner` again unless you edit a `@MappableClass`.
- **l10n codegen**: Triggered automatically by `flutter pub get` because of `flutter: generate: true` in `pubspec.yaml`. Do not edit `app/lib/l10n/generated/` by hand.
- **No new dependencies** are added in this plan — `dart_mappable`, `dart_mappable_builder`, `flutter`, `flutter_test`, and `flutter_localizations` are already in `pubspec.yaml`.
- **`MessageAction` equality** falls back to reference equality on the `onPressed` closure (closures don't structurally compare). The messenger never deduplicates actions, so this is fine.
- **Why `dismiss()` is in the interface but barely used**: it gives consumers a programmatic close path (e.g. an `AsyncLoading → AsyncData` transition that wants to clear a previous sticky error). No call site uses it in this plan, but exposing it now keeps the interface complete and avoids a follow-up change.
