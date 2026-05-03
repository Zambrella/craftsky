# CraftskyDialog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a paper-cutout-styled `CraftskyDialog` widget plus three helper functions (neutral confirm, destructive confirm, alert) that replace ad hoc `AlertDialog` usage across the Flutter app, with async-aware confirm support that shows a loading spinner on the primary button while a `Future<void> Function()` runs and rethrows on error so the dialog stays open.

**Architecture:** A single file at `app/lib/theme/craftsky_dialog.dart` exports the `CraftskyDialog` `StatelessWidget` (pure visual primitive: `paper3` surface, 1.5px ink border, `r3` corners, hard-offset `dropLg` shadow drawn via stacked layers like `ChunkyButton`), a `CraftskyDialogAction` value class consumed by the helpers, and three top-level helper functions that wrap a private `_AsyncConfirmDialogHost` `StatefulWidget` to manage the loading state. The helpers use `showGeneralDialog<bool>` so the entrance animation can pick up `DurationTheme.modal` + `easePop` from the theme. Pop suppression during async work is enforced with `PopScope(canPop: !_isConfirming)`, which blocks both system-back and barrier-tap dismissal.

**Tech Stack:** Flutter (Material), Riverpod 3.x already wired into the app, theme extensions already defined in `app/lib/theme/theme_extensions.dart` (`SpacingTheme`, `RadiusTheme`, `BrandShadowTheme`, `BrandSwatchTheme`, `DurationTheme`), `ChunkyButton` from `app/lib/theme/chunky_button.dart` for the primary action, l10n via `AppLocalizations` generated from `app/lib/l10n/app_en.arb`.

---

## File Structure

| File | Status | Responsibility |
|---|---|---|
| `app/lib/theme/craftsky_dialog.dart` | Create | `CraftskyDialog` widget, `CraftskyDialogAction` value class, three helpers, private `_AsyncConfirmDialogHost` |
| `app/test/theme/craftsky_dialog_test.dart` | Create | Widget tests for the widget directly + each helper |
| `app/lib/l10n/app_en.arb` | Modify | Add `dialogConfirmDefault`, `dialogCancelDefault`, `dialogOkDefault` |
| `app/lib/l10n/generated/app_localizations.dart` | Regenerate | `flutter gen-l10n` output |
| `app/lib/profile/pages/edit_profile_dialog.dart` | Modify | Replace the vanilla `AlertDialog` at lines 248-269 with `showCraftskyConfirmDialog` |
| `app/lib/design_playground/pages/design_playground_page.dart` | Modify | Add `DialogsSample` widget + insert it into the `ListView` |

---

## How to run tests

All commands run from the `app/` directory:

- Single test file: `flutter test test/theme/craftsky_dialog_test.dart`
- All tests: `flutter test`
- Static analysis: `dart analyze`
- Format: `dart format lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart`

After modifying `app_en.arb`, regenerate: `flutter gen-l10n` (also auto-runs on the next build).

---

### Task 1: Add l10n default-label keys

**Files:**
- Modify: `app/lib/l10n/app_en.arb`
- Regenerate: `app/lib/l10n/generated/app_localizations.dart` (and `app_localizations_en.dart`)

- [ ] **Step 1: Add three keys to the ARB file**

Insert these entries near the top of `app/lib/l10n/app_en.arb`, right after the `homeVersionLabel` block (so they sit with other generic UI strings rather than buried among feature-specific entries):

```json
  "dialogConfirmDefault": "Confirm",
  "@dialogConfirmDefault": { "description": "Default label for the primary action button on a CraftskyDialog confirm helper when the caller does not provide one." },

  "dialogCancelDefault": "Cancel",
  "@dialogCancelDefault": { "description": "Default label for the secondary action button on a CraftskyDialog confirm helper when the caller does not provide one." },

  "dialogOkDefault": "OK",
  "@dialogOkDefault": { "description": "Default label for the dismiss button on a CraftskyDialog alert helper when the caller does not provide one." },
```

- [ ] **Step 2: Regenerate l10n**

Run from `app/`:

```bash
flutter gen-l10n
```

Expected: no output on success; the generated files under `lib/l10n/generated/` now expose `dialogConfirmDefault`, `dialogCancelDefault`, `dialogOkDefault` getters on `AppLocalizations`.

- [ ] **Step 3: Verify the getters exist**

Run from `app/`:

```bash
grep -E 'dialogConfirmDefault|dialogCancelDefault|dialogOkDefault' lib/l10n/generated/app_localizations.dart
```

Expected: at least three matching lines (the abstract getter declarations); more if you also see the `_en` subclass overrides.

- [ ] **Step 4: Commit**

```bash
git add app/lib/l10n/app_en.arb app/lib/l10n/generated/
git commit -m "feat(app): add default labels for CraftskyDialog helpers"
```

---

### Task 2: Create the `CraftskyDialog` widget (visual primitive only)

This task builds the styled widget shell with no helper API and no async behavior — just title, body, actions in the chunky theme. Tests pump the widget directly inside a themed `MaterialApp` and assert layout.

**Files:**
- Create: `app/lib/theme/craftsky_dialog.dart`
- Create: `app/test/theme/craftsky_dialog_test.dart`

- [ ] **Step 1: Write the failing test**

Create `app/test/theme/craftsky_dialog_test.dart`:

```dart
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Widget pumpHarness(Widget child) {
    return MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(body: Center(child: child)),
    );
  }

  group('CraftskyDialog', () {
    testWidgets('renders title, body, and actions', (tester) async {
      await tester.pumpWidget(
        pumpHarness(
          const CraftskyDialog(
            title: 'A title',
            body: Text('A body'),
            actions: [
              Text('Action one'),
              Text('Action two'),
            ],
          ),
        ),
      );

      expect(find.text('A title'), findsOneWidget);
      expect(find.text('A body'), findsOneWidget);
      expect(find.text('Action one'), findsOneWidget);
      expect(find.text('Action two'), findsOneWidget);
    });
  });
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: compilation failure — `craftsky_dialog.dart` does not exist yet.

- [ ] **Step 3: Implement the widget**

Create `app/lib/theme/craftsky_dialog.dart`:

```dart
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// A branded confirm/alert dialog. Paper-cutout aesthetic: thick ink border,
/// chunky `r3` corners, hard-offset drop shadow drawn via stacked layers (the
/// same approach used by `ChunkyButton`).
///
/// Most callers should reach for [showCraftskyConfirmDialog],
/// [showCraftskyDestructiveConfirmDialog], or [showCraftskyAlertDialog]
/// rather than constructing this widget directly.
class CraftskyDialog extends StatelessWidget {
  const CraftskyDialog({
    required this.title,
    required this.body,
    required this.actions,
    super.key,
  });

  final String title;
  final Widget body;
  final List<Widget> actions;

  /// Maximum width on wide screens. Below this, the dialog tracks the
  /// available width minus [_horizontalInset] on each side.
  static const double _maxWidth = 360;

  /// Horizontal inset reserved on small screens so the 10px shadow never
  /// touches the edge.
  static const double _horizontalInset = 24;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;

    final shadowOffset = shadows.dropLg.first.offset;
    final shadowColor = shadows.dropLg.first.color;
    final radius = BorderRadius.circular(radii.r3);

    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(
          horizontal: _horizontalInset,
          vertical: _horizontalInset,
        ),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: _maxWidth),
          child: Stack(
            children: [
              Positioned.fill(
                child: Transform.translate(
                  offset: shadowOffset,
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      color: shadowColor,
                      borderRadius: radius,
                    ),
                  ),
                ),
              ),
              Material(
                color: Colors.transparent,
                child: Container(
                  decoration: BoxDecoration(
                    color: swatches.paper3,
                    borderRadius: radius,
                    border: Border.all(color: colors.onSurface, width: 1.5),
                  ),
                  padding: EdgeInsets.all(spacing.sp5),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Text(title, style: theme.textTheme.titleLarge),
                      SizedBox(height: spacing.sp4),
                      DefaultTextStyle.merge(
                        style: theme.textTheme.bodyMedium,
                        child: body,
                      ),
                      SizedBox(height: spacing.sp5),
                      Wrap(
                        alignment: WrapAlignment.end,
                        spacing: spacing.sp2,
                        runSpacing: spacing.sp2,
                        children: actions,
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: PASS.

- [ ] **Step 5: Format and analyze**

```bash
cd app && dart format lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart && dart analyze lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart
```

Expected: no issues.

- [ ] **Step 6: Commit**

```bash
git add app/lib/theme/craftsky_dialog.dart app/test/theme/craftsky_dialog_test.dart
git commit -m "feat(app): add CraftskyDialog visual primitive"
```

---

### Task 3: Add `CraftskyDialogAction` value class + neutral confirm helper (no async yet)

**Files:**
- Modify: `app/lib/theme/craftsky_dialog.dart`
- Modify: `app/test/theme/craftsky_dialog_test.dart`

- [ ] **Step 1: Write the failing tests**

Append to `app/test/theme/craftsky_dialog_test.dart` inside `void main()` after the existing `group`:

```dart
  group('showCraftskyConfirmDialog', () {
    testWidgets('returns true when confirm tapped', (tester) async {
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyConfirmDialog(
                    context,
                    title: 'Discard?',
                    message: 'Your changes will be lost.',
                    confirmLabel: 'Discard',
                    cancelLabel: 'Keep editing',
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      expect(find.text('Discard?'), findsOneWidget);
      expect(find.text('Your changes will be lost.'), findsOneWidget);
      expect(find.text('Discard'), findsOneWidget);
      expect(find.text('Keep editing'), findsOneWidget);

      await tester.tap(find.text('Discard'));
      await tester.pumpAndSettle();

      expect(await resultFuture, isTrue);
      expect(find.byType(CraftskyDialog), findsNothing);
    });

    testWidgets('returns false when cancel tapped', (tester) async {
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyConfirmDialog(
                    context,
                    title: 'Discard?',
                    message: 'Your changes will be lost.',
                    confirmLabel: 'Discard',
                    cancelLabel: 'Keep editing',
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Keep editing'));
      await tester.pumpAndSettle();

      expect(await resultFuture, isFalse);
    });

    testWidgets('returns false when barrier tapped', (tester) async {
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyConfirmDialog(
                    context,
                    title: 'Discard?',
                    message: 'Your changes will be lost.',
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      // Tap on the modal barrier (anywhere outside the dialog box).
      await tester.tapAt(const Offset(10, 10));
      await tester.pumpAndSettle();

      expect(await resultFuture, isFalse);
    });

    testWidgets('falls back to localized labels when none given',
        (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () => showCraftskyConfirmDialog(
                  context,
                  title: 'T',
                  message: 'M',
                ),
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      expect(find.text('Confirm'), findsOneWidget);
      expect(find.text('Cancel'), findsOneWidget);
    });
  });
```

Add this import at the top of the test file:

```dart
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: compilation error — `showCraftskyConfirmDialog` not defined.

- [ ] **Step 3: Implement the value class and helper**

Append to `app/lib/theme/craftsky_dialog.dart`:

```dart
/// Configuration for a single button on a [CraftskyDialog]. Used by the
/// `show…Dialog` helpers; not consumed by [CraftskyDialog] itself.
class CraftskyDialogAction {
  const CraftskyDialogAction({
    required this.label,
    this.onPressed,
    this.isPrimary = false,
    this.isDestructive = false,
  });

  /// Visible button label.
  final String label;

  /// Tap handler. May be sync or async. `null` disables the button.
  final FutureOr<void> Function()? onPressed;

  /// Renders as a filled [ChunkyButton] when true; otherwise a [TextButton].
  final bool isPrimary;

  /// When [isPrimary] is also true, swaps the surface to [BrandColors.red]
  /// for delete/sign-out style actions.
  final bool isDestructive;
}

/// Shows a neutral two-button confirm dialog. Resolves to `true` if the user
/// taps the confirm action, `false` if they cancel, dismiss the barrier, or
/// hit the system back button.
///
/// If [onConfirm] is provided, the primary button shows a loading spinner
/// while the future completes; on success, the dialog pops with `true`. If
/// [onConfirm] throws, the dialog stays open, both buttons re-enable, and
/// the error rethrows so the caller's existing error-handling path runs.
Future<bool> showCraftskyConfirmDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? confirmLabel,
  String? cancelLabel,
  Future<void> Function()? onConfirm,
}) {
  return _showConfirmDialog(
    context,
    title: title,
    message: message,
    confirmLabel: confirmLabel,
    cancelLabel: cancelLabel,
    onConfirm: onConfirm,
    isDestructive: false,
  );
}

Future<bool> _showConfirmDialog(
  BuildContext context, {
  required String title,
  required String message,
  required String? confirmLabel,
  required String? cancelLabel,
  required Future<void> Function()? onConfirm,
  required bool isDestructive,
}) async {
  final l10n = AppLocalizations.of(context);
  final durations = Theme.of(context).extension<DurationTheme>()!;
  final result = await showGeneralDialog<bool>(
    context: context,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: Colors.black54,
    transitionDuration: durations.modal,
    pageBuilder: (_, _, _) => _AsyncConfirmDialogHost(
      title: title,
      message: message,
      confirmLabel: confirmLabel ?? l10n.dialogConfirmDefault,
      cancelLabel: cancelLabel ?? l10n.dialogCancelDefault,
      onConfirm: onConfirm,
      isDestructive: isDestructive,
    ),
    transitionBuilder: (_, animation, _, child) {
      final curved = CurvedAnimation(parent: animation, curve: durations.easePop);
      return FadeTransition(
        opacity: curved,
        child: ScaleTransition(
          scale: Tween<double>(begin: 0.96, end: 1.0).animate(curved),
          child: child,
        ),
      );
    },
  );
  return result ?? false;
}
```

Add these imports at the top of `app/lib/theme/craftsky_dialog.dart` (replacing the existing import block):

```dart
import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
```

Now add the private host widget at the bottom of the file. This task implements the **non-async** path; Task 4 will extend it with async behavior.

```dart
class _AsyncConfirmDialogHost extends StatefulWidget {
  const _AsyncConfirmDialogHost({
    required this.title,
    required this.message,
    required this.confirmLabel,
    required this.cancelLabel,
    required this.onConfirm,
    required this.isDestructive,
  });

  final String title;
  final String message;
  final String confirmLabel;
  final String cancelLabel;
  final Future<void> Function()? onConfirm;
  final bool isDestructive;

  @override
  State<_AsyncConfirmDialogHost> createState() =>
      _AsyncConfirmDialogHostState();
}

class _AsyncConfirmDialogHostState extends State<_AsyncConfirmDialogHost> {
  bool _isConfirming = false;

  Future<void> _handleConfirm() async {
    if (widget.onConfirm == null) {
      Navigator.of(context).pop(true);
      return;
    }
    // Async path is implemented in Task 4.
    Navigator.of(context).pop(true);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primaryBackground = widget.isDestructive
        ? BrandColors.red
        : theme.colorScheme.primary;

    return PopScope(
      canPop: !_isConfirming,
      child: CraftskyDialog(
        title: widget.title,
        body: Text(widget.message),
        actions: [
          TextButton(
            onPressed: _isConfirming
                ? null
                : () => Navigator.of(context).pop(false),
            child: Text(widget.cancelLabel),
          ),
          ChunkyButton(
            backgroundColor: primaryBackground,
            onPressed: _isConfirming ? null : _handleConfirm,
            child: Text(widget.confirmLabel),
          ),
        ],
      ),
    );
  }
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: PASS for all four `showCraftskyConfirmDialog` tests plus the existing `CraftskyDialog` test.

- [ ] **Step 5: Format and analyze**

```bash
cd app && dart format lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart && dart analyze lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart
```

Expected: no issues.

- [ ] **Step 6: Commit**

```bash
git add app/lib/theme/craftsky_dialog.dart app/test/theme/craftsky_dialog_test.dart
git commit -m "feat(app): add showCraftskyConfirmDialog neutral helper"
```

---

### Task 4: Add async-confirm support (loading spinner + error rethrow)

**Files:**
- Modify: `app/lib/theme/craftsky_dialog.dart` — flesh out `_handleConfirm` and the loading state
- Modify: `app/test/theme/craftsky_dialog_test.dart` — add three async behavior tests

- [ ] **Step 1: Write the failing tests**

Append to `app/test/theme/craftsky_dialog_test.dart` inside `void main()`:

```dart
  group('showCraftskyConfirmDialog (async)', () {
    testWidgets('shows spinner during onConfirm and pops with true on success',
        (tester) async {
      final completer = Completer<void>();
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyConfirmDialog(
                    context,
                    title: 'T',
                    message: 'M',
                    confirmLabel: 'Yes',
                    cancelLabel: 'No',
                    onConfirm: () => completer.future,
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Yes'));
      await tester.pump(); // start spinner
      await tester.pump(const Duration(milliseconds: 50));

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      // Confirm button label is hidden while spinner is shown.
      expect(find.text('Yes'), findsNothing);

      // Cancel must be disabled.
      final cancel = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'No'),
      );
      expect(cancel.onPressed, isNull);

      completer.complete();
      await tester.pumpAndSettle();

      expect(await resultFuture, isTrue);
      expect(find.byType(CraftskyDialog), findsNothing);
    });

    testWidgets('keeps dialog open and rethrows when onConfirm throws',
        (tester) async {
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyConfirmDialog(
                    context,
                    title: 'T',
                    message: 'M',
                    confirmLabel: 'Yes',
                    cancelLabel: 'No',
                    onConfirm: () async {
                      await Future<void>.delayed(
                        const Duration(milliseconds: 10),
                      );
                      throw StateError('nope');
                    },
                  );
                  // Suppress unhandled-exception in the test zone.
                  resultFuture.catchError((_) => false);
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Yes'));
      await tester.pump(const Duration(milliseconds: 50));
      await tester.pumpAndSettle();

      // Dialog still mounted after the throw.
      expect(find.byType(CraftskyDialog), findsOneWidget);
      // Spinner is gone, label is back.
      expect(find.byType(CircularProgressIndicator), findsNothing);
      expect(find.text('Yes'), findsOneWidget);

      // Cancel is re-enabled.
      final cancel = tester.widget<TextButton>(
        find.widgetWithText(TextButton, 'No'),
      );
      expect(cancel.onPressed, isNotNull);

      // The error propagated through the returned future.
      expect(resultFuture, throwsStateError);
    });

    testWidgets('barrier tap during async in flight is suppressed',
        (tester) async {
      final completer = Completer<void>();

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  showCraftskyConfirmDialog(
                    context,
                    title: 'T',
                    message: 'M',
                    confirmLabel: 'Yes',
                    cancelLabel: 'No',
                    onConfirm: () => completer.future,
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Yes'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));

      // While async is in flight, barrier-tap must not dismiss.
      await tester.tapAt(const Offset(10, 10));
      await tester.pump(const Duration(milliseconds: 50));
      expect(find.byType(CraftskyDialog), findsOneWidget);

      completer.complete();
      await tester.pumpAndSettle();
    });
  });
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: the new tests fail (the host's `_handleConfirm` always pops with `true`, never sets `_isConfirming`).

- [ ] **Step 3: Replace `_handleConfirm` and the primary button child**

Edit `app/lib/theme/craftsky_dialog.dart`. Replace the entire `_handleConfirm` method with:

```dart
  Future<void> _handleConfirm() async {
    final onConfirm = widget.onConfirm;
    if (onConfirm == null) {
      Navigator.of(context).pop(true);
      return;
    }

    setState(() => _isConfirming = true);
    Object? caughtError;
    StackTrace? caughtStack;
    try {
      await onConfirm();
    } catch (e, st) {
      caughtError = e;
      caughtStack = st;
    }

    if (!mounted) return;

    if (caughtError == null) {
      Navigator.of(context).pop(true);
      return;
    }

    setState(() => _isConfirming = false);
    Error.throwWithStackTrace(caughtError, caughtStack!);
  }
```

Replace the `ChunkyButton(...)` line in the `actions` list with a stateful child that swaps in a spinner while confirming. Update the `actions` list inside `build` to:

```dart
        actions: [
          TextButton(
            onPressed: _isConfirming
                ? null
                : () => Navigator.of(context).pop(false),
            child: Text(widget.cancelLabel),
          ),
          ChunkyButton(
            backgroundColor: primaryBackground,
            onPressed: _isConfirming ? null : _handleConfirm,
            child: _isConfirming
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      valueColor:
                          AlwaysStoppedAnimation<Color>(Colors.white),
                    ),
                  )
                : Text(widget.confirmLabel),
          ),
        ],
```

The spinner uses `Colors.white` deliberately: the destructive button surface is `BrandColors.red` and the neutral button surface is `colorScheme.primary` (cobalt) — both have white as their `onPrimary`. Hardcoding white avoids a context-lookup dependency inside the spinner builder.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: all tests PASS, including the three async cases.

- [ ] **Step 5: Format and analyze**

```bash
cd app && dart format lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart && dart analyze lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart
```

Expected: no issues.

- [ ] **Step 6: Commit**

```bash
git add app/lib/theme/craftsky_dialog.dart app/test/theme/craftsky_dialog_test.dart
git commit -m "feat(app): wire async onConfirm with spinner and error rethrow"
```

---

### Task 5: Add the destructive confirm helper

**Files:**
- Modify: `app/lib/theme/craftsky_dialog.dart`
- Modify: `app/test/theme/craftsky_dialog_test.dart`

- [ ] **Step 1: Write the failing test**

Append to `app/test/theme/craftsky_dialog_test.dart` inside `void main()`:

```dart
  group('showCraftskyDestructiveConfirmDialog', () {
    testWidgets('primary button uses BrandColors.red surface', (tester) async {
      late Future<bool> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyDestructiveConfirmDialog(
                    context,
                    title: 'Delete?',
                    message: 'This cannot be undone.',
                    confirmLabel: 'Delete',
                    cancelLabel: 'Cancel',
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      final primary = tester.widget<ChunkyButton>(find.byType(ChunkyButton));
      expect(primary.backgroundColor, BrandColors.red);

      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(await resultFuture, isFalse);
    });
  });
```

Add these imports at the top of the test file:

```dart
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: compilation error — `showCraftskyDestructiveConfirmDialog` not defined.

- [ ] **Step 3: Add the helper**

Append to `app/lib/theme/craftsky_dialog.dart` (after `showCraftskyConfirmDialog`):

```dart
/// Shows a destructive two-button confirm dialog. Identical to
/// [showCraftskyConfirmDialog] except the primary action surface is
/// [BrandColors.red] for delete-style flows.
Future<bool> showCraftskyDestructiveConfirmDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? confirmLabel,
  String? cancelLabel,
  Future<void> Function()? onConfirm,
}) {
  return _showConfirmDialog(
    context,
    title: title,
    message: message,
    confirmLabel: confirmLabel,
    cancelLabel: cancelLabel,
    onConfirm: onConfirm,
    isDestructive: true,
  );
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: PASS.

- [ ] **Step 5: Format, analyze, commit**

```bash
cd app && dart format lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart && dart analyze lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart
git add app/lib/theme/craftsky_dialog.dart app/test/theme/craftsky_dialog_test.dart
git commit -m "feat(app): add showCraftskyDestructiveConfirmDialog helper"
```

---

### Task 6: Add the alert helper

**Files:**
- Modify: `app/lib/theme/craftsky_dialog.dart`
- Modify: `app/test/theme/craftsky_dialog_test.dart`

- [ ] **Step 1: Write the failing tests**

Append to `app/test/theme/craftsky_dialog_test.dart` inside `void main()`:

```dart
  group('showCraftskyAlertDialog', () {
    testWidgets('renders title, message, single dismiss button', (tester) async {
      late Future<void> resultFuture;

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () {
                  resultFuture = showCraftskyAlertDialog(
                    context,
                    title: 'Saved',
                    message: 'Your changes are live.',
                    dismissLabel: 'Got it',
                  );
                },
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      expect(find.text('Saved'), findsOneWidget);
      expect(find.text('Your changes are live.'), findsOneWidget);
      expect(find.byType(ChunkyButton), findsOneWidget);
      expect(
        find.descendant(
          of: find.byType(CraftskyDialog),
          matching: find.byType(TextButton),
        ),
        findsNothing,
      );

      await tester.tap(find.text('Got it'));
      await tester.pumpAndSettle();

      // Returned future completes.
      await resultFuture;
      expect(find.byType(CraftskyDialog), findsNothing);
    });

    testWidgets('falls back to localized dismiss label', (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () => showCraftskyAlertDialog(
                  context,
                  title: 'T',
                  message: 'M',
                ),
                child: const Text('open'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      expect(find.text('OK'), findsOneWidget);
    });
  });
```

Note: the "no cancel button" assertion is scoped to descendants of `CraftskyDialog` because the Scaffold's "open" button is also a `TextButton` and would otherwise match.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: compilation error — `showCraftskyAlertDialog` not defined.

- [ ] **Step 3: Add the helper and a tiny private host**

Append to `app/lib/theme/craftsky_dialog.dart`:

```dart
/// Shows a single-button informational dialog. Resolves when the user taps
/// the dismiss button or the modal barrier.
Future<void> showCraftskyAlertDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? dismissLabel,
}) async {
  final l10n = AppLocalizations.of(context);
  final durations = Theme.of(context).extension<DurationTheme>()!;
  final theme = Theme.of(context);
  await showGeneralDialog<void>(
    context: context,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: Colors.black54,
    transitionDuration: durations.modal,
    pageBuilder: (dialogContext, _, _) => CraftskyDialog(
      title: title,
      body: Text(message),
      actions: [
        ChunkyButton(
          backgroundColor: theme.colorScheme.primary,
          onPressed: () => Navigator.of(dialogContext).pop(),
          child: Text(dismissLabel ?? l10n.dialogOkDefault),
        ),
      ],
    ),
    transitionBuilder: (_, animation, _, child) {
      final curved = CurvedAnimation(parent: animation, curve: durations.easePop);
      return FadeTransition(
        opacity: curved,
        child: ScaleTransition(
          scale: Tween<double>(begin: 0.96, end: 1.0).animate(curved),
          child: child,
        ),
      );
    },
  );
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd app && flutter test test/theme/craftsky_dialog_test.dart
```

Expected: PASS.

- [ ] **Step 5: Format, analyze, commit**

```bash
cd app && dart format lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart && dart analyze lib/theme/craftsky_dialog.dart test/theme/craftsky_dialog_test.dart
git add app/lib/theme/craftsky_dialog.dart app/test/theme/craftsky_dialog_test.dart
git commit -m "feat(app): add showCraftskyAlertDialog single-button helper"
```

---

### Task 7: Migrate `edit_profile_dialog.dart` to the new helper

**Files:**
- Modify: `app/lib/profile/pages/edit_profile_dialog.dart` (lines 248-270)

- [ ] **Step 1: Read the current implementation to confirm line numbers**

```bash
sed -n '248,270p' app/lib/profile/pages/edit_profile_dialog.dart
```

Expected: shows the `_confirmDiscard` method with the inline `AlertDialog`.

- [ ] **Step 2: Replace the body of `_confirmDiscard`**

Replace the entire `_confirmDiscard` method with:

```dart
  Future<bool> _confirmDiscard() async {
    final l10n = AppLocalizations.of(context);
    return showCraftskyConfirmDialog(
      context,
      title: l10n.editProfileDiscardTitle,
      message: l10n.editProfileDiscardMessage,
      confirmLabel: l10n.editProfileDiscardConfirm,
      cancelLabel: l10n.editProfileDiscardCancel,
    );
  }
```

- [ ] **Step 3: Add the import**

At the top of `app/lib/profile/pages/edit_profile_dialog.dart`, add:

```dart
import 'package:craftsky_app/theme/craftsky_dialog.dart';
```

(Sorted alphabetically with the existing `craftsky_app` imports.)

- [ ] **Step 4: Run analyze + the existing edit-profile tests**

```bash
cd app && dart analyze lib/profile/pages/edit_profile_dialog.dart
cd app && flutter test test/profile/
```

Expected: analyze clean. Existing edit-profile tests still pass (the discard dialog is internal — no test asserts on its widget tree).

- [ ] **Step 5: Run the full test suite to make sure nothing else broke**

```bash
cd app && flutter test
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add app/lib/profile/pages/edit_profile_dialog.dart
git commit -m "refactor(app): use showCraftskyConfirmDialog for discard confirm"
```

---

### Task 8: Add `DialogsSample` to the design playground

**Files:**
- Modify: `app/lib/design_playground/pages/design_playground_page.dart`

- [ ] **Step 1: Add the import**

At the top of `app/lib/design_playground/pages/design_playground_page.dart`, add:

```dart
import 'package:craftsky_app/theme/craftsky_dialog.dart';
```

- [ ] **Step 2: Insert the new `PlaygroundSection` into the `ListView`**

Find the section labelled `'eyebrow: 'Cards'` in the `build` method's `ListView` children. Immediately after the `SizedBox(height: sp.sp7)` that follows the Cards section, insert:

```dart
          const PlaygroundSection(
            eyebrow: 'Dialogs',
            child: DialogsSample(),
          ),
          SizedBox(height: sp.sp7),
```

- [ ] **Step 3: Add the `DialogsSample` widget at the bottom of the file**

Append to `app/lib/design_playground/pages/design_playground_page.dart`:

```dart
class DialogsSample extends StatelessWidget {
  const DialogsSample({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;

    return Wrap(
      spacing: spacing.sp3,
      runSpacing: spacing.sp3,
      children: [
        ChunkyButton(
          onPressed: () async {
            final result = await showCraftskyConfirmDialog(
              context,
              title: 'Discard draft?',
              message: 'Your changes will be lost.',
              confirmLabel: 'Discard',
              cancelLabel: 'Keep editing',
            );
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('Confirm result: $result')),
            );
          },
          child: const Text('Show neutral confirm'),
        ),
        ChunkyButton(
          backgroundColor: theme.colorScheme.error,
          onPressed: () async {
            final result = await showCraftskyDestructiveConfirmDialog(
              context,
              title: 'Delete this post?',
              message: 'This cannot be undone.',
              confirmLabel: 'Delete',
              cancelLabel: 'Cancel',
            );
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(content: Text('Destructive result: $result')),
            );
          },
          child: const Text('Show destructive confirm'),
        ),
        ChunkyButton(
          onPressed: () async {
            await showCraftskyAlertDialog(
              context,
              title: 'Saved',
              message: 'Your profile is live.',
              dismissLabel: 'Got it',
            );
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Alert dismissed')),
            );
          },
          child: const Text('Show alert'),
        ),
        ChunkyButton(
          onPressed: () async {
            try {
              final result = await showCraftskyConfirmDialog(
                context,
                title: 'Sync draft?',
                message: 'Pretends to do work for 1.5s, throws ~50% of the time.',
                confirmLabel: 'Sync',
                cancelLabel: 'Cancel',
                onConfirm: () async {
                  await Future<void>.delayed(const Duration(milliseconds: 1500));
                  if (DateTime.now().millisecondsSinceEpoch.isEven) {
                    throw StateError('Pretend network error');
                  }
                },
              );
              if (!context.mounted) return;
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('Async result: $result')),
              );
            } catch (e) {
              if (!context.mounted) return;
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('Async threw: $e')),
              );
            }
          },
          child: const Text('Show async confirm'),
        ),
      ],
    );
  }
}
```

- [ ] **Step 4: Format and analyze**

```bash
cd app && dart format lib/design_playground/pages/design_playground_page.dart && dart analyze lib/design_playground/pages/design_playground_page.dart
```

Expected: no issues.

- [ ] **Step 5: Smoke-test in a running app**

```bash
cd app && flutter run -d <device>
```

In the app, navigate to the design playground and tap each of the four buttons. Verify:
- Neutral and destructive confirms render with the chunky shadow + ink border, primary button uses cobalt or red as appropriate.
- Alert renders with a single button.
- Async confirm shows the spinner during the 1.5s wait, pops on success, stays open and surfaces an error snackbar on throw, blocks barrier-tap during the wait.

(If you can't run the app, do a deeper widget test of `DialogsSample` — but this step is intentionally a manual visual check, since the playground exists for that purpose.)

- [ ] **Step 6: Commit**

```bash
git add app/lib/design_playground/pages/design_playground_page.dart
git commit -m "feat(app): add Dialogs sample to design playground"
```

---

## Self-Review Notes

- **Spec coverage** — Each section maps to a task: visual primitive (T2), value class + neutral helper (T3), async behavior (T4), destructive (T5), alert (T6), migration (T7), playground (T8), l10n (T1). Tests are interleaved per TDD. All six test cases from the spec land in the test file by the end of T6.
- **Type consistency** — `CraftskyDialog`, `CraftskyDialogAction`, `_AsyncConfirmDialogHost`, `showCraftskyConfirmDialog`, `showCraftskyDestructiveConfirmDialog`, `showCraftskyAlertDialog`, `_showConfirmDialog`, `dialogConfirmDefault`, `dialogCancelDefault`, `dialogOkDefault` — all referenced names match across tasks.
- **No placeholders** — Every code step shows the literal code; every command shows the literal invocation.
- **Async-throw test note** — Task 4's "rethrow" test attaches `resultFuture.catchError((_) => false)` immediately after the call so the unhandled-exception zone handler doesn't fail the test, and then asserts `expect(resultFuture, throwsStateError)` for the actual rethrow assertion.
