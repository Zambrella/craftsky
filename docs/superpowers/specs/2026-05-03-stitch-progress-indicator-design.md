# StitchProgressIndicator — Design

A custom Craftsky-branded replacement for `CircularProgressIndicator`, used everywhere the app shows an indeterminate loading state.

## Motivation

The Flutter app uses Material's `CircularProgressIndicator` in eight places — full-screen loaders during app boot, profile loading, dialogs, and small in-button spinners. None of these carry Craftsky's visual identity. The app already has a custom button (`ChunkyButton`), brand text fields, and a paper/cobalt design system; the spinner is the loudest unbranded chrome left.

A "stitch ring" — a dashed circle that reads as a running-stitch line — fits the textile-craft theme without becoming a literal yarn-ball illustration that degrades at small sizes.

## Visual Direction

A simple rotating stitch ring:

- Cobalt stroke (`colorScheme.primary` = `BrandColors.cobalt` `#1535D6`).
- Dashed stroke (`stroke-dasharray`-equivalent) — reads as evenly-spaced stitches around the ring.
- Rotates at a steady 1.4-second linear cycle. No easing, no pulsing, no accent color, no counter-rotation. The brief was "interesting but simple"; we explored chevrons, comet trails, self-drawing rings, dual rings, and a needle-around-hoop motion. Each added personality but also added busyness; the plain rotating dashed ring won on legibility at every size.

The full exploration (with live SVG mockups) lives in `.superpowers/brainstorm/67691-1777820452/content/` for posterity.

## Public API

Lives at `app/lib/theme/stitch_progress_indicator.dart`, alongside `chunky_button.dart`. It is a brand widget — same shelf as the other custom Material replacements.

```dart
class StitchProgressIndicator extends StatefulWidget {
  const StitchProgressIndicator({
    super.key,
    this.size = 36,
    this.strokeWidth,
    this.color,
    this.value,
  });

  /// Diameter in logical pixels. Defaults to 36 (matches Material's
  /// `CircularProgressIndicator` footprint).
  final double size;

  /// Stroke width in logical pixels. When `null`, derived as
  /// `(size / 12).clamp(1.4, 6.0)` so the ring stays visually balanced
  /// from in-button (~18 px) to full-screen (~96 px) sizes.
  final double? strokeWidth;

  /// Stroke color. Defaults to `Theme.of(context).colorScheme.primary`.
  final Color? color;

  /// Reserved for the future determinate variant. When non-null, the painter
  /// will draw `floor(value * dashCount)` stitches starting at 12 o'clock and
  /// stop rotating. Today this parameter is plumbed through but not yet
  /// rendered — passing it has no visible effect. See "Determinate readiness."
  final double? value;
}
```

The two in-button sites pass `size: 18` to replace `CircularProgressIndicator(strokeWidth: 2)`; everything else uses the default.

## Implementation

A `StatefulWidget` with a single `AnimationController` (period 1.4 s, `repeat()`) feeding a `CustomPainter`.

### Painter

`_StitchPainter` draws a dashed circle. Inputs:

- `radius`, `strokeWidth`, `color` — geometry.
- `dashCount` — number of stitches around the ring. Auto-derived in the widget so stitch density stays constant across sizes: target ~14 dashes at the default 36 px size, scaling roughly with `2π · r / desiredDashLen`. A small spinner therefore gets fewer but proportionate dashes, a large one does not become visually crowded.
- `rotationTurns` — value in `[0, 1)` from the controller; painter applies `canvas.rotate(2π · rotationTurns)` around the ring center before drawing.
- `value` — nullable, passed through but not used in the rendered output today (see "Determinate readiness").

Drawing approach: compute the per-dash arc angle, then for each dash call `canvas.drawArc` with a stroked `Paint` (`StrokeCap.butt`, no shader). The dash/gap ratio is fixed at roughly `1:1` so the stitched feel reads even at small sizes.

`shouldRepaint` returns true on `rotationTurns` change; everything else uses cheap field-equality comparisons.

### Animation

```dart
late final AnimationController _controller = AnimationController(
  vsync: this,
  duration: const Duration(milliseconds: 1400),
)..repeat();
```

The widget rebuilds the painter via `AnimatedBuilder(animation: _controller, ...)`. The 1400 ms period is a const in the file; it is not promoted to `DurationTheme` because no other widget consumes it (YAGNI).

### Reduce-motion support

When `MediaQuery.of(context).disableAnimations` is `true`, the controller is created but never started, and the painter renders a static dashed ring at `rotationTurns = 0`. This matches Flutter's standard reduce-motion contract.

### Accessibility

The widget wraps the painter in `Semantics(label: l10n.loading, container: true)` so screen readers announce it the same way Material's spinner does. The `loading` string is added to the existing `l10n` ARB bundle.

## Determinate readiness (deferred)

Per the brainstorm decision, the painter is parameterised for a future determinate mode without shipping it now:

- `dashCount` is already a discrete integer — naturally suited to "fill `n` of `N` stitches."
- `value` is plumbed all the way through `StitchProgressIndicator → AnimatedBuilder → _StitchPainter`.
- The painter's draw loop iterates `dashCount` times; switching it to "draw the first `floor(value * dashCount)` dashes only" is a one-line conditional.
- Adding `_controller.stop()` when `value != null` is the only widget-side change.

When this lands, no API break and no painter rewrite — only a `value`-aware branch in `paint()` and the controller-stop call.

## Tests

All under `app/test/theme/stitch_progress_indicator_test.dart`:

- **Widget test (animation runs).** Pump the indicator, advance the clock by 700 ms (half cycle), confirm the painter's `rotationTurns` has changed. Implementation: expose `rotationTurns` via a `@visibleForTesting` getter on the State and assert it has advanced from 0.
- **Golden tests.** Default size (36 px) and small size (18 px) at a fixed `rotationTurns` value (set the controller to `0.25` and pump). Two goldens total.
- **Reduce-motion test.** Wrap in a `MediaQuery` with `disableAnimations: true`; pump for one second; assert the controller never advanced past zero. The static ring still renders (golden assertion optional).
- **Disposal test.** Push the indicator into the tree, then pump a different widget; assert the controller is disposed (no leaked tickers — `tester.pump()` would otherwise flag the ticker).

Existing widget tests that match `find.byType(CircularProgressIndicator)` need updates wherever they apply to one of the eight replaced sites; see "Replacement plan" below.

## Design playground

Add a section to `app/lib/design_playground/pages/design_playground_page.dart`:

```dart
const PlaygroundSection(
  eyebrow: 'Progress',
  child: ProgressSample(),
),
```

The new `ProgressSample` widget shows the indicator at three sizes — 18, 36, 64 — laid out horizontally on the paper-grain background, with size labels underneath. This mirrors how `ButtonsSample` and `ChipsSample` already showcase their components.

## Replacement plan

The eight `CircularProgressIndicator` call sites get replaced with `StitchProgressIndicator`:

| Site | New call |
|------|----------|
| `app/lib/app.dart:106` | `const StitchProgressIndicator()` |
| `app/lib/auth/pages/sign_in_page.dart:61` | `const StitchProgressIndicator(size: 18)` |
| `app/lib/auth/pages/auth_complete_page.dart:53` | `const StitchProgressIndicator()` |
| `app/lib/profile/pages/edit_profile_dialog.dart:85` | `const StitchProgressIndicator()` |
| `app/lib/profile/pages/edit_profile_dialog.dart:101` | `const StitchProgressIndicator()` |
| `app/lib/profile/pages/edit_profile_dialog.dart:495` | `const StitchProgressIndicator(size: 18)` (avatar-upload overlay) |
| `app/lib/profile/pages/profile_page.dart:44` | `const StitchProgressIndicator()` |
| `app/lib/profile/pages/profile_page.dart:81` | `const StitchProgressIndicator()` |

The `.claude/rules/riverpod.md` doc contains `CircularProgressIndicator` inside example switch snippets. Those are illustrative — left alone. Flutter team docs and rules referencing the Material widget aren't binding on Craftsky's own widget choice.

## Out of scope

- **Determinate rendering and its tests.** Parameter is plumbed; UI/logic deferred until a real consumer needs it.
- **`RefreshIndicator` replacement.** Different widget, different interaction model — would be its own spec.
- **Promoting the 1.4 s rotation period to `DurationTheme`.** No second consumer; no need.
- **Updating illustrative `CircularProgressIndicator` snippets in `.claude/rules/riverpod.md`.** Snippets, not load-bearing.

## Open questions

None. Visual direction is locked, scope is locked (all eight sites), determinate-readiness is locked (plumbed but not rendered).
