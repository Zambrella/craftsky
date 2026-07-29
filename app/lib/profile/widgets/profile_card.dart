import 'dart:math' as math;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_stats.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

enum ProfileCardIllustration { botanical, yarn, patchwork }

enum ProfileAvatarFrame { stitched, scalloped, braidedYarn }

/// Compact profile summary shown from identity taps throughout the app.
///
/// The card owns only presentation. Profile loading, navigation, mutations,
/// and future persistence of customisation choices stay with its modal host.
class ProfileCard extends StatelessWidget {
  const ProfileCard({
    required this.profile,
    required this.isOwnProfile,
    required this.onClose,
    required this.onVisitProfile,
    required this.onPrimaryAction,
    this.primaryColor,
    this.backgroundIllustration,
    this.avatarFrame,
    this.isPrimaryActionBusy = false,
    super.key,
  });

  final Profile profile;
  final bool isOwnProfile;
  final VoidCallback onClose;
  final VoidCallback onVisitProfile;
  final VoidCallback onPrimaryAction;
  final Color? primaryColor;
  final ProfileCardIllustration? backgroundIllustration;
  final ProfileAvatarFrame? avatarFrame;
  final bool isPrimaryActionBusy;

  @override
  Widget build(BuildContext context) {
    final parentTheme = Theme.of(context);
    final primary = primaryColor ?? parentTheme.colorScheme.primary;
    final generatedScheme = ColorScheme.fromSeed(
      seedColor: primary,
      brightness: parentTheme.brightness,
    );
    final cardTheme = parentTheme.copyWith(
      colorScheme: parentTheme.colorScheme.copyWith(
        primary: primary,
        onPrimary: generatedScheme.onPrimary,
        primaryContainer: generatedScheme.primaryContainer,
        onPrimaryContainer: generatedScheme.onPrimaryContainer,
      ),
    );

    return Theme(
      data: cardTheme,
      child: _ProfileCardSurface(
        profile: profile,
        backgroundIllustration: backgroundIllustration,
        avatarFrame: avatarFrame,
        isOwnProfile: isOwnProfile,
        isPrimaryActionBusy: isPrimaryActionBusy,
        onClose: onClose,
        onVisitProfile: onVisitProfile,
        onPrimaryAction: onPrimaryAction,
      ),
    );
  }
}

class _ProfileCardSurface extends StatelessWidget {
  const _ProfileCardSurface({
    required this.profile,
    required this.backgroundIllustration,
    required this.avatarFrame,
    required this.isOwnProfile,
    required this.isPrimaryActionBusy,
    required this.onClose,
    required this.onVisitProfile,
    required this.onPrimaryAction,
  });

  final Profile profile;
  final ProfileCardIllustration? backgroundIllustration;
  final ProfileAvatarFrame? avatarFrame;
  final bool isOwnProfile;
  final bool isPrimaryActionBusy;
  final VoidCallback onClose;
  final VoidCallback onVisitProfile;
  final VoidCallback onPrimaryAction;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final radius = BorderRadius.circular(radii.r4);
    final shadow = shadows.paper2.first;

    return LayoutBuilder(
      builder: (context, constraints) {
        final availableHeight = constraints.maxHeight.isFinite
            ? math
                  .max(
                    0,
                    constraints.maxHeight - spacing.sp4 * 2,
                  )
                  .toDouble()
            : double.infinity;
        return Center(
          child: Padding(
            padding: EdgeInsets.all(spacing.sp4),
            child: ConstrainedBox(
              constraints: BoxConstraints(
                maxWidth: 420,
                maxHeight: availableHeight,
              ),
              child: Stack(
                clipBehavior: Clip.none,
                children: [
                  Positioned.fill(
                    child: Transform.translate(
                      offset: shadow.offset,
                      child: DecoratedBox(
                        decoration: BoxDecoration(
                          color: shadow.color,
                          borderRadius: radius,
                        ),
                      ),
                    ),
                  ),
                  Material(
                    color: swatches.paper3,
                    clipBehavior: Clip.antiAlias,
                    shape: RoundedRectangleBorder(
                      borderRadius: radius,
                      side: BorderSide(color: colors.onSurface, width: 1.5),
                    ),
                    child: SingleChildScrollView(
                      child: Stack(
                        alignment: Alignment.topCenter,
                        children: [
                          Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              SizedBox(
                                height: 160,
                                width: double.infinity,
                                child: ColoredBox(
                                  key: const Key('profile-card-header'),
                                  color: colors.primary,
                                  child: backgroundIllustration == null
                                      ? null
                                      : CustomPaint(
                                          key: const Key(
                                            'profile-card-background-'
                                            'illustration',
                                          ),
                                          painter:
                                              _ProfileCardIllustrationPainter(
                                                illustration:
                                                    backgroundIllustration!,
                                                color: colors.onPrimary,
                                              ),
                                        ),
                                ),
                              ),
                              Padding(
                                padding: EdgeInsets.fromLTRB(
                                  spacing.sp5,
                                  62,
                                  spacing.sp5,
                                  spacing.sp5,
                                ),
                                child: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    _ProfileCardIdentity(profile: profile),
                                    if (profile.crafts.isNotEmpty) ...[
                                      SizedBox(height: spacing.sp3),
                                      _ProfileCardCrafts(
                                        crafts: profile.crafts,
                                      ),
                                    ],
                                    SizedBox(height: spacing.sp4),
                                    _ProfileCardStats(profile: profile),
                                    SizedBox(height: spacing.sp5),
                                    _ProfileCardActions(
                                      isOwnProfile: isOwnProfile,
                                      isFollowing: profile.viewerIsFollowing,
                                      isBusy: isPrimaryActionBusy,
                                      onVisitProfile: onVisitProfile,
                                      onPrimaryAction: onPrimaryAction,
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                          Positioned(
                            top: 98,
                            child: _ProfileCardAvatar(
                              profile: profile,
                              frame: avatarFrame,
                            ),
                          ),
                          Positioned(
                            top: spacing.sp3,
                            right: spacing.sp3,
                            child: Material(
                              color: swatches.paper3,
                              shape: const CircleBorder(),
                              child: IconButton(
                                key: const Key('profile-card-close'),
                                tooltip: MaterialLocalizations.of(
                                  context,
                                ).closeButtonTooltip,
                                onPressed: onClose,
                                icon: const Icon(Icons.close),
                              ),
                            ),
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
      },
    );
  }
}

class _ProfileCardAvatar extends StatelessWidget {
  const _ProfileCardAvatar({required this.profile, required this.frame});

  final Profile profile;
  final ProfileAvatarFrame? frame;

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
              color: colors.surface,
              shape: BoxShape.circle,
            ),
            child: const SizedBox.square(dimension: 112),
          ),
          ProfileAvatar(
            seed: profile.displayName ?? profile.handle.toString(),
            avatarUrl: profile.avatar,
            size: ProfileAvatarSize.large,
            showShadow: false,
          ),
          if (frame != null)
            CustomPaint(
              key: const Key('profile-card-avatar-frame'),
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

class _ProfileCardIdentity extends StatelessWidget {
  const _ProfileCardIdentity({required this.profile});

  final Profile profile;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final name = switch (profile.displayName) {
      final displayName? when displayName.trim().isNotEmpty => displayName,
      _ => '@${profile.handle}',
    };
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(
          name,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          textAlign: TextAlign.center,
          style: theme.textTheme.headlineMedium,
        ),
        if (profile.displayName?.trim().isNotEmpty ?? false)
          Text(
            '@${profile.handle}',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
      ],
    );
  }
}

class _ProfileCardCrafts extends StatelessWidget {
  const _ProfileCardCrafts({required this.crafts});

  final List<String> crafts;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    return Wrap(
      alignment: WrapAlignment.center,
      spacing: spacing.sp2,
      runSpacing: spacing.sp2,
      children: [
        for (final craft in crafts)
          Container(
            padding: EdgeInsets.symmetric(
              horizontal: spacing.sp3,
              vertical: 6,
            ),
            decoration: BoxDecoration(
              color: colors.primaryContainer,
              borderRadius: BorderRadius.circular(radii.rPill),
            ),
            child: Text(
              _sentenceCase(craft),
              style: theme.textTheme.labelMedium?.copyWith(
                color: colors.onPrimaryContainer,
              ),
            ),
          ),
      ],
    );
  }
}

class _ProfileCardStats extends StatelessWidget {
  const _ProfileCardStats({required this.profile});

  final Profile profile;

  @override
  Widget build(BuildContext context) {
    final stats = <({IconData icon, String value, String label})>[
      if (profile.isCraftskyProfile && profile.createdAt != null)
        (
          icon: Icons.calendar_today_outlined,
          value: formatJoinedAge(profile.createdAt!),
          label: 'here',
        ),
      if (profile.postsLast7Days != null)
        (
          icon: Icons.edit_outlined,
          value: '${_compactCount(profile.postsLast7Days!)} posts',
          label: '7 days',
        ),
      if (profile.projectCount != null)
        (
          icon: Icons.inventory_2_outlined,
          value: _compactCount(profile.projectCount!),
          label: 'projects',
        ),
    ];
    if (stats.isEmpty) return const SizedBox.shrink();

    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    return DecoratedBox(
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border.all(color: colors.outlineVariant),
        borderRadius: BorderRadius.circular(radii.r3),
      ),
      child: Padding(
        padding: EdgeInsets.symmetric(vertical: spacing.sp3),
        child: IntrinsicHeight(
          child: Row(
            children: [
              for (var index = 0; index < stats.length; index++) ...[
                Expanded(
                  child: _ProfileCardStat(data: stats[index]),
                ),
                if (index < stats.length - 1)
                  VerticalDivider(
                    width: 1,
                    thickness: 1,
                    color: colors.outlineVariant,
                  ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _ProfileCardStat extends StatelessWidget {
  const _ProfileCardStat({required this.data});

  final ({IconData icon, String value, String label}) data;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: spacing.sp1),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(data.icon, color: theme.colorScheme.primary, size: 20),
          SizedBox(height: spacing.sp1),
          FittedBox(
            fit: BoxFit.scaleDown,
            child: Text(
              data.value,
              maxLines: 1,
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w800,
              ),
            ),
          ),
          Text(
            data.label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

class _ProfileCardActions extends StatelessWidget {
  const _ProfileCardActions({
    required this.isOwnProfile,
    required this.isFollowing,
    required this.isBusy,
    required this.onVisitProfile,
    required this.onPrimaryAction,
  });

  final bool isOwnProfile;
  final bool isFollowing;
  final bool isBusy;
  final VoidCallback onVisitProfile;
  final VoidCallback onPrimaryAction;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>()!;
    final isQuietUnfollow = !isOwnProfile && isFollowing;
    final primaryLabel = isOwnProfile
        ? l10n.profileEditAction
        : isFollowing
        ? l10n.profileFollowingAction
        : l10n.profileFollowAction;
    return Row(
      children: [
        Expanded(
          child: OutlinedButton(
            onPressed: onVisitProfile,
            style: OutlinedButton.styleFrom(
              foregroundColor: colors.primary,
              side: BorderSide(color: colors.primary, width: 1.5),
              minimumSize: const Size(0, 48),
            ),
            child: Text(l10n.profileVisitAction),
          ),
        ),
        SizedBox(width: spacing.sp3),
        Expanded(
          child: ChunkyButton(
            onPressed: isBusy ? null : onPrimaryAction,
            backgroundColor: isQuietUnfollow ? swatches.paper3 : null,
            foregroundColor: isQuietUnfollow ? colors.onSurface : null,
            child: Text(primaryLabel),
          ),
        ),
      ],
    );
  }
}

String _sentenceCase(String value) {
  final trimmed = value.trim();
  if (trimmed.isEmpty) return trimmed;
  return '${trimmed[0].toUpperCase()}${trimmed.substring(1).toLowerCase()}';
}

String _compactCount(int count) {
  if (count < 1000) return '$count';
  if (count < 10000) return '${(count / 1000).toStringAsFixed(1)}k';
  return '${(count / 1000).round()}k';
}

class _ProfileCardIllustrationPainter extends CustomPainter {
  const _ProfileCardIllustrationPainter({
    required this.illustration,
    required this.color,
  });

  final ProfileCardIllustration illustration;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color.withValues(alpha: 0.22)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2
      ..strokeCap = StrokeCap.round;
    switch (illustration) {
      case ProfileCardIllustration.botanical:
        _paintBotanical(canvas, size, paint);
      case ProfileCardIllustration.yarn:
        _paintYarn(canvas, size, paint);
      case ProfileCardIllustration.patchwork:
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
  bool shouldRepaint(_ProfileCardIllustrationPainter oldDelegate) {
    return oldDelegate.illustration != illustration ||
        oldDelegate.color != color;
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
