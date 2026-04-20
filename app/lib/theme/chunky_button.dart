import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// A pill-shaped button with a hard-offset shadow that lifts on hover and
/// presses down onto its shadow on tap. The "press" signature move of the
/// CraftSky paper-cutout direction.
///
/// Extends [ButtonStyleButton] so it inherits focus, keyboard activation,
/// semantics, ink/splash handling, and [ButtonStyle] theming. Colour, border,
/// and shadow defaults come from `Theme.of(context)` and [BrandShadowTheme] —
/// callers can still override everything via `style:`.
class ChunkyButton extends ButtonStyleButton {
  const ChunkyButton({
    required super.onPressed,
    required Widget super.child,
    super.key,
    super.onLongPress,
    super.onHover,
    super.onFocusChange,
    super.style,
    super.focusNode,
    super.autofocus = false,
    super.clipBehavior = Clip.none,
    super.statesController,
    this.backgroundColor,
    this.foregroundColor,
  });

  /// Surface color at rest. Defaults to `colorScheme.primary` (cobalt). Hover
  /// and press states darken this via alpha-blend.
  final Color? backgroundColor;

  /// Label/icon color. Defaults to `colorScheme.onPrimary`.
  final Color? foregroundColor;

  /// Extra lift applied on hover (negative Y = up).
  static const Offset _hoverLift = Offset(-1, -1);

  /// Width of the foreground border. Matches the design-system default rule.
  static const double _borderWidth = 1.5;

  @override
  ButtonStyle defaultStyleOf(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final text = theme.textTheme;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final durations = theme.extension<DurationTheme>()!;

    final surface = backgroundColor ?? colors.primary;
    final onSurface = foregroundColor ?? colors.onPrimary;

    // Offset of the drop shadow at rest. The button travels this far when
    // pressed so it meets the shadow.
    final restShadow = shadows.dropSm.first;
    final restShadowOffset = restShadow.offset;
    final restShadowColor = restShadow.color;

    return ButtonStyle(
      textStyle: WidgetStatePropertyAll(
        (text.labelLarge ?? const TextStyle()).copyWith(
          fontWeight: FontWeight.w800,
          letterSpacing: 0.2,
        ),
      ),
      backgroundColor: const WidgetStatePropertyAll(Colors.transparent),
      foregroundColor: WidgetStatePropertyAll(onSurface),
      overlayColor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.pressed)) {
          return onSurface.withValues(alpha: 0.12);
        }
        if (states.contains(WidgetState.hovered)) {
          return onSurface.withValues(alpha: 0.08);
        }
        if (states.contains(WidgetState.focused)) {
          return onSurface.withValues(alpha: 0.10);
        }
        return null;
      }),

      // We draw our own shadow, so kill Material's elevation entirely.
      elevation: const WidgetStatePropertyAll(0),
      shadowColor: const WidgetStatePropertyAll(Colors.transparent),
      surfaceTintColor: const WidgetStatePropertyAll(Colors.transparent),

      shape: const WidgetStatePropertyAll(StadiumBorder()),

      padding: const WidgetStatePropertyAll(
        EdgeInsets.symmetric(horizontal: 28, vertical: 14),
      ),
      minimumSize: const WidgetStatePropertyAll(Size(64, 44)),
      maximumSize: const WidgetStatePropertyAll(Size.infinite),
      side: const WidgetStatePropertyAll(BorderSide.none),
      iconColor: WidgetStatePropertyAll(onSurface),
      iconSize: const WidgetStatePropertyAll(18),
      mouseCursor: WidgetStateProperty.resolveWith((states) {
        if (states.contains(WidgetState.disabled)) {
          return SystemMouseCursors.basic;
        }
        return SystemMouseCursors.click;
      }),
      visualDensity: theme.visualDensity,
      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
      animationDuration: durations.medium,
      enableFeedback: true,
      alignment: Alignment.center,
      splashFactory: InkRipple.splashFactory,

      // Custom background paints the hard-offset shadow AND the coloured
      // surface, then positions the foreground child on top. We paint the
      // surface ourselves (rather than letting Material do it) because the
      // surface needs to translate together with the foreground on press and
      // hover, while the shadow stays put.
      backgroundBuilder: (context, states, child) {
        final hovered = states.contains(WidgetState.hovered);
        final disabled = states.contains(WidgetState.disabled);

        final shadowColor = disabled
            ? colors.onSurface.withValues(alpha: 0.20)
            : restShadowColor;

        return _ChunkyBackground(
          shadowOffset: restShadowOffset,
          shadowColor: shadowColor,
          restSurfaceColor: disabled
              ? colors.onSurface.withValues(alpha: 0.12)
              : surface,
          borderColor: disabled
              ? colors.onSurface.withValues(alpha: 0.38)
              : colors.onSurface,
          borderWidth: _borderWidth,
          hovered: hovered,
          disabled: disabled,
          hoverLift: _hoverLift,
          pressOffset: restShadowOffset,
          pressDuration: durations.fast,
          releaseDuration: durations.medium,
          child: child!,
        );
      },
    );
  }

  @override
  ButtonStyle? themeStyleOf(BuildContext context) {
    // No dedicated theme extension yet — consumers style via `style:` or by
    // wrapping in a Theme with a ButtonStyle override.
    return null;
  }
}

class _ChunkyBackground extends StatefulWidget {
  const _ChunkyBackground({
    required this.shadowOffset,
    required this.shadowColor,
    required this.restSurfaceColor,
    required this.borderColor,
    required this.borderWidth,
    required this.hovered,
    required this.disabled,
    required this.hoverLift,
    required this.pressOffset,
    required this.pressDuration,
    required this.releaseDuration,
    required this.child,
  });

  final Offset shadowOffset;
  final Color shadowColor;

  /// Surface color at rest. Hover/press states darken this via alpha-blend.
  final Color restSurfaceColor;

  final Color borderColor;
  final double borderWidth;

  final bool hovered;
  final bool disabled;

  final Offset hoverLift;

  /// Translation applied when fully pressed — the button meets its shadow.
  /// Any tap must animate to this value before returning to rest, even if
  /// the tap releases before the forward animation would naturally complete.
  final Offset pressOffset;

  final Duration pressDuration;
  final Duration releaseDuration;

  final Widget child;

  @override
  State<_ChunkyBackground> createState() => _ChunkyBackgroundState();
}

class _ChunkyBackgroundState extends State<_ChunkyBackground>
    with SingleTickerProviderStateMixin {
  // Drives translation and surface-darken together: 0 = rest, 1 = pressed.
  late final AnimationController _press;

  // Tracks whether the pointer is currently down. Set on pointer-down (from
  // the Listener, which fires immediately — unlike WidgetState.pressed which
  // only flips after the gesture recognizer resolves tap-vs-scroll).
  bool _pointerDown = false;

  @override
  void initState() {
    super.initState();
    _press = AnimationController(
      vsync: this,
      duration: widget.pressDuration,
      reverseDuration: widget.releaseDuration,
    )..addStatusListener(_onStatus);
  }

  @override
  void didUpdateWidget(covariant _ChunkyBackground oldWidget) {
    super.didUpdateWidget(oldWidget);
    _press
      ..duration = widget.pressDuration
      ..reverseDuration = widget.releaseDuration;
  }

  void _onStatus(AnimationStatus status) {
    // If the pointer was lifted while we were still animating forward,
    // start reversing as soon as the forward completes — this guarantees
    // the full press-down frame is seen even for very brief taps.
    if (status == AnimationStatus.completed && !_pointerDown) {
      _press.reverse();
    }
  }

  void _handlePointerDown(PointerDownEvent _) {
    if (widget.disabled) return;
    _pointerDown = true;
    _press.forward();
  }

  void _handlePointerUpOrCancel() {
    if (!_pointerDown) return;
    _pointerDown = false;
    if (_press.status == AnimationStatus.completed) {
      _press.reverse();
    }
    // Otherwise wait for _onStatus to trigger the reverse on completion.
  }

  @override
  void dispose() {
    _press.dispose();
    super.dispose();
  }

  Offset _translationFor(double t) {
    // Hover baseline — lift slightly when hovered and not pressing down.
    if (t == 0 && widget.hovered && !widget.disabled) {
      return widget.hoverLift;
    }
    // Lerp from hover-lift (or zero) → full press offset along press progress.
    final base = widget.hovered ? widget.hoverLift : Offset.zero;
    return Offset.lerp(base, widget.pressOffset, t)!;
  }

  Color _surfaceColorFor(double t) {
    if (widget.disabled) return widget.restSurfaceColor;
    // Hover darkens by 8%; press adds up to another 10% on top so a full
    // press is 18% darker than rest — matches the original _darken scale.
    final hoverBlend = widget.hovered ? 0.08 : 0.0;
    final pressBlend = hoverBlend + (0.18 - hoverBlend) * t;
    if (pressBlend == 0) return widget.restSurfaceColor;
    return Color.alphaBlend(
      Colors.black.withValues(alpha: pressBlend),
      widget.restSurfaceColor,
    );
  }

  @override
  Widget build(BuildContext context) {
    // Three stacked layers:
    //   1. Shadow — stadium, behind, offset down, never translated.
    //   2. Surface — stadium, coloured, translated together with layer 3.
    //   3. Foreground child — the label content from ButtonStyleButton.
    //
    // The Stack's intrinsic size comes from the unpositioned foreground
    // (layer 3). The shadow (layer 1) and surface (layer 2) both use
    // Positioned.fill so they stretch to match the Stack's final size —
    // critical when the button is placed in a CrossAxisAlignment.stretch
    // Column or similar full-width slot.
    //
    // A Listener wraps the whole stack to capture pointer-down immediately
    // (before the button's gesture recognizer resolves tap-vs-scroll), so
    // very brief taps still trigger the press animation.
    return Listener(
      onPointerDown: _handlePointerDown,
      onPointerUp: (_) => _handlePointerUpOrCancel(),
      onPointerCancel: (_) => _handlePointerUpOrCancel(),
      child: Stack(
        alignment: Alignment.center,
        clipBehavior: Clip.none,
        children: [
          Positioned.fill(
            child: Transform.translate(
              offset: widget.shadowOffset,
              child: DecoratedBox(
                decoration: ShapeDecoration(
                  color: widget.shadowColor,
                  shape: const StadiumBorder(),
                ),
              ),
            ),
          ),
          Positioned.fill(
            child: AnimatedBuilder(
              animation: _press,
              builder: (context, _) {
                final t = Curves.easeOut.transform(_press.value);
                return Transform.translate(
                  offset: _translationFor(t),
                  child: DecoratedBox(
                    decoration: ShapeDecoration(
                      color: _surfaceColorFor(t),
                      shape: StadiumBorder(
                        side: BorderSide(
                          color: widget.borderColor,
                          width: widget.borderWidth,
                        ),
                      ),
                    ),
                  ),
                );
              },
            ),
          ),
          AnimatedBuilder(
            animation: _press,
            builder: (context, innerChild) {
              final t = Curves.easeOut.transform(_press.value);
              return Transform.translate(
                offset: _translationFor(t),
                child: innerChild,
              );
            },
            child: widget.child,
          ),
        ],
      ),
    );
  }
}
