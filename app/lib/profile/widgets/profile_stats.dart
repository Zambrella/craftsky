import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:timeago/timeago.dart' as timeago;

String formatJoinedAge(DateTime createdAt, {DateTime? now}) {
  return 'Joined ${timeago.format(createdAt, clock: now)}';
}

/// Profile summary row. Popularity metrics (followers/following) intentionally
/// do not render here; those counts remain available in the API but are shown
/// only one tap deeper from Settings.
class ProfileStats extends StatelessWidget {
  const ProfileStats({required this.profile, super.key});

  final Profile profile;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final stats = <String>[
      if (profile.isCraftskyProfile && profile.createdAt != null)
        formatJoinedAge(profile.createdAt!),
      if (profile.postsLast7Days != null)
        '${_formatCount(profile.postsLast7Days!)} posts in the last 7 days',
      if (profile.postCount != null)
        '${_formatCount(profile.postCount!)} posts',
      if (profile.projectCount != null)
        '${_formatCount(profile.projectCount!)} projects',
    ];
    if (stats.isEmpty) {
      return const SizedBox.shrink();
    }
    return Wrap(
      spacing: spacing.sp4,
      runSpacing: spacing.sp1,
      children: [for (final stat in stats) _ProfileStat(label: stat)],
    );
  }
}

class _ProfileStat extends StatelessWidget {
  const _ProfileStat({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsets.symmetric(vertical: spacing.sp1),
      child: Text(
        label,
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onSurfaceVariant,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

String _formatCount(int count) {
  if (count < 1000) return '$count';
  if (count < 10000) {
    final thousands = count / 1000;
    return '${thousands.toStringAsFixed(1)}k';
  }
  return '${(count / 1000).round()}k';
}
