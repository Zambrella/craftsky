import 'package:craftsky_app/theme/brand_colors.dart';
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
    return Row(
      children: [
        _ProfileStat(
          count: followingCount,
          label: 'following',
          onTap: onFollowingTap,
        ),
        const SizedBox(width: 20),
        _ProfileStat(
          count: followerCount,
          label: 'followers',
          onTap: onFollowersTap,
        ),
        const SizedBox(width: 20),
        _ProfileStat(
          count: projectCount,
          label: 'projects',
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
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(4),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.baseline,
          textBaseline: TextBaseline.alphabetic,
          children: [
            Text(
              count == null ? '—' : _formatCount(count!),
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.w800,
                color: BrandColors.ink,
              ),
            ),
            const SizedBox(width: 4),
            Text(
              label,
              style: theme.textTheme.bodySmall?.copyWith(
                color: BrandColors.ink2,
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
