# CraftskyDialog — Branded Confirm/Alert Modal

## Summary

A reusable branded dialog component that applies the CraftSky paper-cutout aesthetic — chunky `r3` (14px) corners, 1.5px ink border, hard-offset `dropLg` shadow — to confirmation and alert flows across the Flutter app. Ships as a public widget plus three helper functions covering neutral confirm, destructive confirm, and alert. The primary action wires through to `ChunkyButton` and supports an async `onConfirm` callback so callers can run a Riverpod mutation while the button shows a loading spinner; if the callback throws, the dialog stays open and the error rethrows so existing `ref.listen` snackbar handling still works.

## Why now

The only dialog in the codebase today is the discard-confirm in [`edit_profile_dialog.dart:253`](../../app/lib/profile/pages/edit_profile_dialog.dart) — a vanilla Material `AlertDialog` that visually clashes with the paper-cutout direction (rounded but not chunky, soft Material shadow, no ink border). As more flows land (delete-post, block-user, sign-out, leave-thread), the project will accrete more bare `AlertDialog`s unless a branded primitive exists. Building the primitive now and migrating the one existing call site is cheap; doing it after five more call sites land is not.

The `ChunkyButton` work already proved out the "stack-of-rectangles with hard-offset shadow" approach for stateful interactive elements. The dialog reuses that pattern at a coarser scale.

## Non-goals (v1)

- **Iconography.** No leading icon/illustration slot. Body text carries the meaning. Easy to add later via a constructor parameter without breaking callers.
- **Top-right close ("X") affordance.** The cancel action button is the dismiss; barrier-tap also works for non-async non-destructive flows.
- **Custom barrier color.** Use Material's default scrim. Revisit if it reads poorly against paper.
- **Custom transition curve.** Use `showDialog`'s default fade+scale, but override the duration to `DurationTheme.modal` (320ms) and the curve to `easePop` so the entrance has the springy paper feel.
- **Three-or-more action layouts.** Helpers cover the 1- and 2-button cases. The underlying widget accepts an arbitrary `List<Widget>` for actions, so custom layouts remain possible without API expansion.
- **Cross-platform adaptive variants.** Single Material-based implementation. The chunky aesthetic doesn't want a Cupertino-style action sheet on iOS.

## Architecture

### File layout

```
app/lib/theme/
  craftsky_dialog.dart       # Public widget, action config, helpers, internal stateful confirm host
app/test/theme/
  craftsky_dialog_test.dart  # Widget tests for all three helpers + async behavior
app/lib/design_playground/pages/
  design_playground_page.dart  # New DialogsSample widget added
app/lib/profile/pages/
  edit_profile_dialog.dart   # Migrate existing AlertDialog at line 253 to showCraftskyConfirmDialog
app/lib/l10n/
  app_en.arb                 # New default labels: dialogConfirmDefault, dialogCancelDefault, dialogOkDefault
```

### Components

#### `CraftskyDialog` (public `StatelessWidget`)

The styled primitive. Used directly by advanced callers; otherwise reached via the helpers.

```dart
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
}
```

Visual contract:

| Aspect | Value | Source |
|---|---|---|
| Surface | `BrandSwatchTheme.paper3` | Theme extension |
| Border | 1.5px solid `colorScheme.onSurface` | `BorderSide` |
| Corner radius | `RadiusTheme.r3` (14px) | Theme extension |
| Shadow | `BrandShadowTheme.dropLg` (10/10 ink offset) | Theme extension |
| Inner padding | `SpacingTheme.sp5` (24px) all sides | Theme extension |
| Title→body gap | `SpacingTheme.sp4` (16px) | Theme extension |
| Body→actions gap | `SpacingTheme.sp5` (24px) | Theme extension |
| Title style | `textTheme.titleLarge` | Theme |
| Max width | 360 logical px | Hardcoded constant |
| Outer inset | `EdgeInsets.symmetric(horizontal: 24, vertical: 24)` | Hardcoded — reserves space for the 10px shadow |

The shadow uses the same `Stack` technique as `ChunkyButton`: an `onSurface`-coloured `RoundedRectangleBorder`-shaped layer translated by `Offset(10, 10)` behind a `paper3`-coloured layer at rest. `BoxDecoration.boxShadow` blurs by default, and even `blurRadius: 0` with a `BoxShadow` works but doesn't compose well with the border — the explicit two-rectangle approach matches the rest of the design system and keeps everything pixel-aligned.

Actions row is a `Wrap(alignment: WrapAlignment.end, spacing: sp.sp2, runSpacing: sp.sp2)` so a long-label confirm wraps below cancel on narrow screens (e.g. iPhone SE in landscape with a system text-scale of 1.3).

#### `CraftskyDialogAction` (value class)

Configuration consumed by the helpers; not used directly by the widget.

```dart
class CraftskyDialogAction {
  const CraftskyDialogAction({
    required this.label,
    this.onPressed,
    this.isPrimary = false,
    this.isDestructive = false,
  });

  final String label;
  final FutureOr<void> Function()? onPressed; // null disables the button
  final bool isPrimary;
  final bool isDestructive;
}
```

The helpers translate each `CraftskyDialogAction` into a widget:
- **Primary, non-destructive** → `ChunkyButton` with default `colorScheme.primary` surface.
- **Primary, destructive** → `ChunkyButton` with `backgroundColor: BrandColors.red` (the brand "danger" surface, also the existing `colorScheme.secondary`/`colorScheme.error` value).
- **Non-primary** → `TextButton` with `textTheme.labelLarge`, `onSurface` foreground. Cancel always falls into this bucket.

#### Helper functions

```dart
Future<bool> showCraftskyConfirmDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? confirmLabel,
  String? cancelLabel,
  Future<void> Function()? onConfirm,
});

Future<bool> showCraftskyDestructiveConfirmDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? confirmLabel,
  String? cancelLabel,
  Future<void> Function()? onConfirm,
});

Future<void> showCraftskyAlertDialog(
  BuildContext context, {
  required String title,
  required String message,
  String? dismissLabel,
});
```

Behavior:

- Confirm helpers return `Future<bool>`. `true` = confirm pressed and `onConfirm` (if any) completed. `false` = cancelled, dismissed by barrier tap, or system back-press. Implementation: the helper awaits `showGeneralDialog<bool>(...)` which returns `Future<bool?>`, and finishes with `return result ?? false`. Cancel button calls `Navigator.pop(context, false)`; barrier tap and system back pop without a value, which the `?? false` collapses to `false`. The user never sees `null`.
- The destructive variant is identical to the neutral variant except for the primary button's red surface and the default confirm label (locale-overridable but distinct).
- Alert helper returns `Future<void>` and shows a single `ChunkyButton` with the `dismissLabel`, defaulting to "OK" via `dialogOkDefault`. Tapping the button or the barrier dismisses it.
- All three call `showGeneralDialog`, not `showDialog`, so the transition duration can be set to `DurationTheme.modal` and the curve to `easePop`. Transition is opacity 0→1 plus scale 0.96→1.0.

### Async confirm behavior

The helpers wrap their content in a private `_AsyncConfirmDialogHost` `StatefulWidget` that owns `_isConfirming: bool`.

Flow when the user taps confirm with `onConfirm` set:

1. `setState(() => _isConfirming = true)`.
2. The primary `ChunkyButton`'s child swaps from `Text(label)` to `SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, valueColor: AlwaysStoppedAnimation(colorScheme.onPrimary)))`. The button's `onPressed` becomes `null` (disabled).
3. The cancel button's `onPressed` becomes `null` (disabled).
4. The dialog's `barrierDismissible` becomes `false` and a `PopScope(canPop: false)` blocks system back.
5. `await onConfirm()`.
6. **On success:** `if (mounted) Navigator.of(context).pop(true)`. The helper's `Future<bool>` completes with `true`.
7. **On throw:** `if (mounted) setState(() => _isConfirming = false)`. The dialog stays open and re-enables both buttons. The error is rethrown after the `setState` so the calling code's `try`/`catch` or `ref.listen` for the underlying mutation provider runs unchanged. (The helper does not swallow.)

For the non-async case (`onConfirm == null`), confirm immediately pops with `true` and never enters the loading state. There is one shared host widget for both cases so the implementation doesn't fork.

### Localization

Three new keys in [`app_en.arb`](../../app/lib/l10n/app_en.arb), used as defaults when a helper caller passes `null`:

| Key | Value |
|---|---|
| `dialogConfirmDefault` | `Confirm` |
| `dialogCancelDefault` | `Cancel` |
| `dialogOkDefault` | `OK` |

Helpers call `AppLocalizations.of(context)` internally to resolve defaults. Callers should pass explicit labels for any flow with non-trivial semantics (e.g. "Discard", "Delete", "Sign out") — the defaults exist purely as a fallback so a quick prototype call site doesn't compile-fail.

### Migration

Replace the `AlertDialog` block in [`edit_profile_dialog.dart:248-269`](../../app/lib/profile/pages/edit_profile_dialog.dart) with:

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

The function's `Future<bool>` return type stays the same; the body simplifies because the helper guarantees a non-null result. Existing call sites of `_confirmDiscard` need no change.

### Design playground

Add `DialogsSample` to [`design_playground_page.dart`](../../app/lib/design_playground/pages/design_playground_page.dart) and wire it into the `ListView` between Cards and Swatches. It contains four `ChunkyButton`s:

1. **"Show neutral confirm"** → `showCraftskyConfirmDialog` with placeholder strings.
2. **"Show destructive confirm"** → `showCraftskyDestructiveConfirmDialog`.
3. **"Show alert"** → `showCraftskyAlertDialog`.
4. **"Show async confirm"** → `showCraftskyConfirmDialog` with an `onConfirm` that `await Future.delayed(const Duration(milliseconds: 1500))` then 50% of the time throws a `StateError`, so the playground demonstrates both success-and-pop and error-keeps-open paths.

Each button captures the result and pushes a `SnackBar` showing what came back, mirroring the other playground samples' "see what happens" style.

## Tests

[`app/test/theme/craftsky_dialog_test.dart`](../../app/test/theme/craftsky_dialog_test.dart):

1. **Neutral confirm renders title, message, confirm + cancel buttons.** Pump a `MaterialApp` with `AppTheme.lightThemeData`, call `showCraftskyConfirmDialog`, assert `find.text` for all four strings, tap cancel, assert future resolves to `false`. Repeat with confirm tap, assert `true`.
2. **Destructive confirm primary button uses brand red.** Pump, find the `ChunkyButton`, walk to its `_ChunkyBackground` child, assert `restSurfaceColor == BrandColors.red`. (Or assert via the `backgroundColor` constructor argument by inspecting the `ChunkyButton` widget directly; whichever survives the chunky-button internals.)
3. **Alert returns void and shows only one button.** Pump, assert `find.byType(ChunkyButton)` has exactly one match, tap it, assert the future completes.
4. **Async onConfirm: success path.** Pass `onConfirm: () async { await Future.delayed(Duration(milliseconds: 50)); }`. Tap confirm, pump 25ms, assert spinner is visible and cancel is disabled, pump 50ms more, assert dialog is gone and future resolves to `true`.
5. **Async onConfirm: throw path.** Pass `onConfirm: () async { throw StateError('nope'); }`. Tap confirm, pump until settled. Assert dialog is still mounted (`find.byType(CraftskyDialog)` matches), assert spinner is gone, both buttons are re-enabled, and the error propagated to the test's zone-level handler (use `runZonedGuarded` or the standard `expect`/throw-matching pattern).
6. **Barrier tap during async in flight is a no-op.** Tap on the `ModalBarrier` (or simulate with `tester.tapAt(Offset.zero)`) while `_isConfirming == true`, pump, assert dialog is still open.

Tests run on the host via `flutter test` (not the appview Postgres path).

## Open questions

None. All API and visual decisions resolved during the brainstorm.

## Future work

- **Icon slot.** A leading `Icon` or asset glyph at the top of the dialog. Add as an optional `leading: Widget?` parameter on `CraftskyDialog` and `leadingIcon: IconData?` on the helpers. Non-breaking.
- **Custom barrier color.** If the default Material scrim ever reads poorly against the paper background, switch to `BrandColors.ink.withValues(alpha: 0.5)` and revisit.
- **Bottom-sheet variant.** A sibling primitive `CraftskyBottomSheet` for the same chunky aesthetic on full-width mobile prompts. Same shadow/border/radius decisions; different layout.
- **Three+ action layout.** A vertical stack on narrow screens. Add when the first call site needs it.
