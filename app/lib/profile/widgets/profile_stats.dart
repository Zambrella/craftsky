import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

String formatJoinedAge(DateTime createdAt, {DateTime? now}) {
  final elapsed = (now ?? DateTime.now()).difference(createdAt);
  final days = elapsed.inDays;
  if (days <= 0) return 'today';
  if (days < 31) return '${days}d';
  if (days < 365) return '${days ~/ 30}mo';
  return '${days ~/ 365}y';
}

/// Profile summary row. Popularity metrics (followers/following) intentionally
/// do not render here; those counts remain available in the API but are shown
/// only one tap deeper from Settings.
class ProfileStats extends StatelessWidget {
  const ProfileStats({required this.profile, super.key});

  final Profile profile;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radius = theme.extension<RadiusTheme>()!;
    final stats = <_ProfileStatData>[
      if (profile.isCraftskyProfile && profile.createdAt != null)
        _ProfileStatData(
          value: formatJoinedAge(profile.createdAt!),
          label: 'here',
        ),
      if (profile.postsLast7Days != null)
        _ProfileStatData(
          value: '${_formatCount(profile.postsLast7Days!)} posts',
          label: '7 days',
        ),
      if (profile.projectCount != null)
        _ProfileStatData(
          value: _formatCount(profile.projectCount!),
          label: 'projects',
        ),
    ];
    if (stats.isEmpty) {
      return const SizedBox.shrink();
    }
    return DecoratedBox(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(color: theme.colorScheme.onSurface, width: 1.5),
        borderRadius: BorderRadius.circular(radius.r3),
      ),
      child: Padding(
        padding: EdgeInsets.symmetric(vertical: spacing.sp2),
        child: IntrinsicHeight(
          child: Row(
            children: [
              for (var i = 0; i < stats.length; i++) ...[
                Expanded(child: _ProfileStat(stat: stats[i])),
                if (i < stats.length - 1)
                  VerticalDivider(
                    width: 1,
                    thickness: 1,
                    color: theme.colorScheme.outlineVariant,
                  ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _ProfileStatData {
  const _ProfileStatData({required this.value, required this.label});

  final String value;
  final String label;
}

class _ProfileStat extends StatelessWidget {
  const _ProfileStat({required this.stat});

  final _ProfileStatData stat;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsets.symmetric(horizontal: spacing.sp2),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          FittedBox(
            fit: BoxFit.scaleDown,
            child: Text(
              stat.value,
              maxLines: 1,
              style: theme.textTheme.titleMedium?.copyWith(
                color: theme.colorScheme.onSurface,
                fontWeight: FontWeight.w800,
              ),
            ),
          ),
          SizedBox(height: spacing.sp1),
          Text(
            stat.label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              letterSpacing: 0.8,
            ),
          ),
        ],
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
