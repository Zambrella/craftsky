import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:flutter/material.dart';

/// Solid themed profile header with an optional illustration from the curated
/// profile customisation set.
class ProfileHeaderBackground extends StatelessWidget {
  const ProfileHeaderBackground({
    this.illustration,
    this.backgroundKey = const Key('profile-header-background'),
    this.illustrationKey = const Key(
      'profile-header-background-illustration',
    ),
    super.key,
  });

  final ProfileBackgroundIllustration? illustration;
  final Key backgroundKey;
  final Key illustrationKey;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return ColoredBox(
      key: backgroundKey,
      color: colors.primary,
      child: illustration == null
          ? null
          : CustomPaint(
              key: illustrationKey,
              painter: _ProfileHeaderIllustrationPainter(
                illustration: illustration!,
                color: colors.onPrimary,
              ),
            ),
    );
  }
}

class _ProfileHeaderIllustrationPainter extends CustomPainter {
  const _ProfileHeaderIllustrationPainter({
    required this.illustration,
    required this.color,
  });

  final ProfileBackgroundIllustration illustration;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color.withValues(alpha: 0.22)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2
      ..strokeCap = StrokeCap.round;
    switch (illustration) {
      case ProfileBackgroundIllustration.botanical:
        _paintBotanical(canvas, size, paint);
      case ProfileBackgroundIllustration.yarn:
        _paintYarn(canvas, size, paint);
      case ProfileBackgroundIllustration.patchwork:
        _paintPatchwork(canvas, size, paint);
    }
  }

  void _paintBotanical(Canvas canvas, Size size, Paint paint) {
    final stem = Path()
      ..moveTo(size.width * 0.58, size.height)
      ..cubicTo(
        size.width * 0.62,
        size.height * 0.68,
        size.width * 0.76,
        size.height * 0.42,
        size.width * 0.72,
        0,
      );
    canvas.drawPath(stem, paint);
    for (final leaf in const [
      (0.62, 0.72, -0.16),
      (0.67, 0.54, 0.14),
      (0.70, 0.34, -0.14),
      (0.72, 0.16, 0.12),
    ]) {
      final center = Offset(size.width * leaf.$1, size.height * leaf.$2);
      canvas
        ..save()
        ..translate(center.dx, center.dy)
        ..rotate(leaf.$3)
        ..drawOval(
          Rect.fromCenter(center: Offset.zero, width: 44, height: 20),
          paint,
        )
        ..restore();
    }
  }

  void _paintYarn(Canvas canvas, Size size, Paint paint) {
    final center = Offset(size.width * 0.76, size.height * 0.46);
    canvas
      ..drawCircle(center, 46, paint)
      ..drawOval(
        Rect.fromCenter(center: center, width: 88, height: 28),
        paint,
      )
      ..drawOval(
        Rect.fromCenter(center: center, width: 34, height: 88),
        paint,
      );
    final strand = Path()
      ..moveTo(center.dx - 18, center.dy + 42)
      ..cubicTo(
        center.dx - 80,
        size.height * 0.86,
        size.width * 0.48,
        size.height * 0.68,
        size.width * 0.36,
        size.height,
      );
    canvas.drawPath(strand, paint);
  }

  void _paintPatchwork(Canvas canvas, Size size, Paint paint) {
    const tile = 52.0;
    for (var row = 0; row < 4; row++) {
      for (var column = 0; column < 4; column++) {
        final left = size.width - (4 - column) * tile + (row.isOdd ? 18 : 0);
        final top = row * 42.0 - 18;
        final path = Path()
          ..moveTo(left + tile / 2, top)
          ..lineTo(left + tile, top + tile / 2)
          ..lineTo(left + tile / 2, top + tile)
          ..lineTo(left, top + tile / 2)
          ..close();
        canvas.drawPath(path, paint);
      }
    }
  }

  @override
  bool shouldRepaint(_ProfileHeaderIllustrationPainter oldDelegate) {
    return oldDelegate.illustration != illustration ||
        oldDelegate.color != color;
  }
}
