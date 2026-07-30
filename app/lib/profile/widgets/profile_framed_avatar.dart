import 'dart:math' as math;

import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:flutter/material.dart';

/// Large profile avatar with a configurable rim and an optional decorative
/// frame from the curated profile customisation set.
class ProfileFramedAvatar extends StatelessWidget {
  const ProfileFramedAvatar({
    required this.seed,
    this.avatarUrl,
    this.frame,
    this.rimColor,
    this.frameKey = const Key('profile-avatar-frame'),
    super.key,
  });

  final String seed;
  final String? avatarUrl;
  final ProfileAvatarFrame? frame;
  final Color? rimColor;
  final Key frameKey;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return SizedBox.square(
      dimension: 124,
      child: Stack(
        alignment: Alignment.center,
        children: [
          DecoratedBox(
            decoration: BoxDecoration(
              color: rimColor ?? colors.surface,
              shape: BoxShape.circle,
            ),
            child: const SizedBox.square(dimension: 112),
          ),
          ProfileAvatar(
            seed: seed,
            avatarUrl: avatarUrl,
            size: ProfileAvatarSize.large,
            showShadow: false,
          ),
          if (frame != null)
            CustomPaint(
              key: frameKey,
              size: const Size.square(124),
              painter: _ProfileAvatarFramePainter(
                frame: frame!,
                color: colors.primary,
              ),
            ),
        ],
      ),
    );
  }
}

class _ProfileAvatarFramePainter extends CustomPainter {
  const _ProfileAvatarFramePainter({
    required this.frame,
    required this.color,
  });

  final ProfileAvatarFrame frame;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final center = size.center(Offset.zero);
    final radius = size.shortestSide / 2 - 7;
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 4
      ..strokeCap = StrokeCap.round;
    switch (frame) {
      case ProfileAvatarFrame.stitched:
        for (var angle = 0.0; angle < math.pi * 2; angle += 0.20) {
          canvas.drawArc(
            Rect.fromCircle(center: center, radius: radius),
            angle,
            0.10,
            false,
            paint,
          );
        }
      case ProfileAvatarFrame.scalloped:
        paint.style = PaintingStyle.fill;
        for (var i = 0; i < 18; i++) {
          final angle = i * math.pi * 2 / 18;
          canvas.drawCircle(
            center + Offset(math.cos(angle), math.sin(angle)) * (radius + 1),
            7,
            paint,
          );
        }
      case ProfileAvatarFrame.braidedYarn:
        for (var i = 0; i < 16; i++) {
          final start = i * math.pi * 2 / 16;
          canvas.drawArc(
            Rect.fromCircle(center: center, radius: radius - 2),
            start,
            0.26,
            false,
            paint..strokeWidth = i.isEven ? 5 : 2.5,
          );
        }
    }
  }

  @override
  bool shouldRepaint(_ProfileAvatarFramePainter oldDelegate) {
    return oldDelegate.frame != frame || oldDelegate.color != color;
  }
}
