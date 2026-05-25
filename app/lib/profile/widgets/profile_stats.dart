import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Following / followers / projects count row. Counts are nullable so
/// callers can pass through whatever subset the AppView has surfaced;
/// missing values render as `—`. Each cell is tappable so the page can
/// later wire navigation to follower lists / project lists without
/// changing this widget's API.
class ProfileStats extends StatelessWidget {
  const ProfileStats({
    this.followingCount,
    this.followerCount,
    this.projectCount,
    this.onFollowingTap,
    this.onFollowersTap,
    this.onProjectsTap,
    super.key,
  });

  final int? followingCount;
  final int? followerCount;
  final int? projectCount;

  final VoidCallback? onFollowingTap;
  final VoidCallback? onFollowersTap;
  final VoidCallback? onProjectsTap;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return Row(
      children: [
        _ProfileStat(
          count: followingCount,
          label: l10n.profileStatsFollowing,
          onTap: onFollowingTap,
        ),
        SizedBox(width: spacing.sp4),
        _ProfileStat(
          count: followerCount,
          label: l10n.profileStatsFollowers,
          onTap: onFollowersTap,
        ),
        SizedBox(width: spacing.sp4),
        _ProfileStat(
          count: projectCount,
          label: l10n.profileStatsProjects,
          onTap: onProjectsTap,
        ),
      ],
    );
  }
}

class _ProfileStat extends StatelessWidget {
  const _ProfileStat({required this.count, required this.label, this.onTap});

  final int? count;
  final String label;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(spacing.sp1),
      child: Padding(
        padding: EdgeInsets.symmetric(vertical: spacing.sp1),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.baseline,
          textBaseline: TextBaseline.alphabetic,
          children: [
            Text(
              count == null ? '—' : _formatCount(count!),
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w800,
                color: theme.colorScheme.onSurface,
              ),
            ),
            SizedBox(width: spacing.sp1),
            Text(
              label,
              // `onSurfaceVariant` carries the brand's ink2 (secondary
              // text) per the ColorScheme override in app_theme.dart.
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _formatCount(int count) {
    if (count < 1000) return '$count';
    if (count < 10000) {
      final thousands = count / 1000;
      return '${thousands.toStringAsFixed(1)}k';
    }
    return '${(count / 1000).round()}k';
  }
}
