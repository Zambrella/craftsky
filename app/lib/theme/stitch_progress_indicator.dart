import 'dart:async';
import 'dart:math' as math;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';

/// Test-only view onto the private `_StitchProgressIndicatorState`. Lets
/// widget tests read animation progress without exposing it on the public
/// widget API.
@visibleForTesting
abstract interface class StitchProgressIndicatorStateForTesting {
  double get rotationTurns;
}

/// A Craftsky-branded indeterminate progress indicator.
///
/// Renders a rotating dashed "running-stitch" ring in the theme's primary
/// colour (cobalt by default). Drop-in replacement for
/// [CircularProgressIndicator] across the app.
///
/// The [value] parameter is reserved for a future determinate variant — it is
/// accepted but currently has no visual effect. See
/// `docs/superpowers/specs/2026-05-03-stitch-progress-indicator-design.md`.
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
  /// from in-button (~18 px) to full-screen sizes.
  final double? strokeWidth;

  /// Stroke colour. Defaults to `Theme.of(context).colorScheme.primary`.
  final Color? color;

  /// Reserved for the future determinate variant. Plumbed through but not
  /// yet rendered.
  final double? value;

  @override
  State<StitchProgressIndicator> createState() =>
      _StitchProgressIndicatorState();
}

class _StitchProgressIndicatorState extends State<StitchProgressIndicator>
    with SingleTickerProviderStateMixin
    implements StitchProgressIndicatorStateForTesting {
  static const _rotationDuration = Duration(milliseconds: 1400);

  late final AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: _rotationDuration,
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _syncAnimationToReduceMotion();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  double get rotationTurns => _controller.value;

  void _syncAnimationToReduceMotion() {
    final disableAnimations = MediaQuery.disableAnimationsOf(context);
    if (disableAnimations) {
      if (_controller.isAnimating) {
        _controller
          ..stop()
          ..value = 0;
      }
    } else if (!_controller.isAnimating) {
      unawaited(_controller.repeat());
    }
  }

  @override
  Widget build(BuildContext context) {
    final color = widget.color ?? Theme.of(context).colorScheme.primary;
    final strokeWidth =
        widget.strokeWidth ?? (widget.size / 12).clamp(1.4, 6.0);
    final dashCount = _computeDashCount(widget.size);

    return Semantics(
      label: AppLocalizations.of(context).loading,
      container: true,
      child: SizedBox.square(
        dimension: widget.size,
        child: AnimatedBuilder(
          animation: _controller,
          builder: (context, _) => CustomPaint(
            painter: _StitchPainter(
              color: color,
              strokeWidth: strokeWidth,
              dashCount: dashCount,
              rotationTurns: _controller.value,
              value: widget.value,
            ),
          ),
        ),
      ),
    );
  }

  /// Stitch density stays roughly constant across sizes by targeting ~14
  /// stitches at the default 36 px size, then scaling with circumference.
  static int _computeDashCount(double size) {
    const referenceSize = 36.0;
    const referenceDashCount = 14;
    final scaled = (size / referenceSize) * referenceDashCount;
    return scaled.round().clamp(6, 32);
  }
}

class _StitchPainter extends CustomPainter {
  _StitchPainter({
    required this.color,
    required this.strokeWidth,
    required this.dashCount,
    required this.rotationTurns,
    required this.value,
  });

  final Color color;
  final double strokeWidth;
  final int dashCount;
  final double rotationTurns;
  // Plumbed for the future determinate variant; not yet rendered.
  final double? value;

  @override
  void paint(Canvas canvas, Size size) {
    final center = size.center(Offset.zero);
    final radius = (math.min(size.width, size.height) - strokeWidth) / 2;

    canvas
      ..save()
      ..translate(center.dx, center.dy)
      ..rotate(rotationTurns * 2 * math.pi);

    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.butt
      ..strokeWidth = strokeWidth;

    // Total angle = 2π. Half the circumference is dashes, half is gaps
    // (1:1 dash:gap ratio).
    const fullAngle = 2 * math.pi;
    final segmentAngle = fullAngle / dashCount;
    final dashAngle = segmentAngle / 2;
    final rect = Rect.fromCircle(center: Offset.zero, radius: radius);

    for (var i = 0; i < dashCount; i++) {
      final start = i * segmentAngle;
      canvas.drawArc(rect, start, dashAngle, false, paint);
    }

    canvas.restore();
  }

  @override
  bool shouldRepaint(covariant _StitchPainter old) {
    return rotationTurns != old.rotationTurns ||
        color != old.color ||
        strokeWidth != old.strokeWidth ||
        dashCount != old.dashCount ||
        value != old.value;
  }
}
