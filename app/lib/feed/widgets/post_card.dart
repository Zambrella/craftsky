import 'package:craftsky_app/feed/models/placeholder_post.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';

/// Card-shaped post row used by the feed and the profile Posts tab.
/// Currently bound to [PlaceholderPost]; will swap to the real post
/// model once feed lexicon wiring lands. The reactions row is read-only
/// for now.
class PostCard extends StatelessWidget {
  const PostCard({required this.post, super.key});

  final PlaceholderPost post;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final onSurface = theme.colorScheme.onSurface;

    return Container(
      decoration: const BoxDecoration(
        border: Border(
          bottom: BorderSide(color: BrandColors.borderHair),
        ),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ProfileAvatar(
            seed: post.authorDisplayName,
            size: ProfileAvatarSize.small,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _PostCardHeader(
                  displayName: post.authorDisplayName,
                  handle: post.authorHandle,
                  postedAt: post.postedAt,
                ),
                const SizedBox(height: 4),
                Text('@${post.authorHandle}', style: theme.textTheme.bodySmall),
                const SizedBox(height: 8),
                Text(post.body, style: theme.textTheme.bodyMedium),
                const SizedBox(height: 10),
                _PostCardReactions(
                  replyCount: post.replyCount,
                  repostCount: post.repostCount,
                  likeCount: post.likeCount,
                  iconColor: onSurface,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _PostCardHeader extends StatelessWidget {
  const _PostCardHeader({
    required this.displayName,
    required this.handle,
    required this.postedAt,
  });

  final String displayName;
  final String handle;
  final DateTime postedAt;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.baseline,
      textBaseline: TextBaseline.alphabetic,
      children: [
        Flexible(
          child: Text(
            displayName,
            style: theme.textTheme.titleSmall,
            overflow: TextOverflow.ellipsis,
          ),
        ),
        const SizedBox(width: 8),
        Text(
          '· ${_relativeTime(postedAt)}',
          style: theme.textTheme.bodySmall?.copyWith(color: BrandColors.ink3),
        ),
      ],
    );
  }

  String _relativeTime(DateTime when) {
    final delta = DateTime.now().difference(when);
    if (delta.inMinutes < 1) return 'now';
    if (delta.inMinutes < 60) return '${delta.inMinutes}m';
    if (delta.inHours < 24) return '${delta.inHours}h';
    return '${delta.inDays}d';
  }
}

class _PostCardReactions extends StatelessWidget {
  const _PostCardReactions({
    required this.replyCount,
    required this.repostCount,
    required this.likeCount,
    required this.iconColor,
  });

  final int replyCount;
  final int repostCount;
  final int likeCount;
  final Color iconColor;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        _Reaction(icon: Icons.mode_comment_outlined, count: replyCount),
        const SizedBox(width: 20),
        _Reaction(icon: Icons.repeat, count: repostCount),
        const SizedBox(width: 20),
        _Reaction(
          icon: Icons.favorite,
          count: likeCount,
          tint: BrandColors.red,
        ),
        const Spacer(),
        Icon(Icons.bookmark_border, size: 18, color: iconColor),
      ],
    );
  }
}

class _Reaction extends StatelessWidget {
  const _Reaction({required this.icon, required this.count, this.tint});

  final IconData icon;
  final int count;
  final Color? tint;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = tint ?? BrandColors.ink2;
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 18, color: color),
        const SizedBox(width: 6),
        Text(
          '$count',
          style: theme.textTheme.bodySmall?.copyWith(color: BrandColors.ink2),
        ),
      ],
    );
  }
}
