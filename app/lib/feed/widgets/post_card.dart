import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

const _postCardMenuWidth = 48.0;

/// Card-shaped post row used by the feed and the profile Posts tab.
class PostCard extends StatelessWidget {
  const PostCard({required this.post, super.key, this.onDelete});

  final Post post;
  final VoidCallback? onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final displayName = post.author.displayName ?? post.author.handle;
    final bodyIndent = ProfileAvatarSize.small.dimension + spacing.sp3;

    return CraftskyCard(
      margin: EdgeInsets.fromLTRB(
        spacing.sp4,
        spacing.sp3,
        spacing.sp4,
        spacing.sp2,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: EdgeInsets.fromLTRB(
              spacing.sp3,
              spacing.sp3,
              spacing.sp3,
              0,
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    ProfileAvatar(
                      seed: displayName,
                      size: ProfileAvatarSize.small,
                    ),
                    SizedBox(width: spacing.sp3),
                    Expanded(
                      child: _PostCardHeader(
                        displayName: displayName,
                        handle: post.author.handle,
                      ),
                    ),
                    SizedBox(
                      width: _postCardMenuWidth,
                      child: Center(
                        child: _PostCardTime(postedAt: post.createdAt),
                      ),
                    ),
                  ],
                ),
                SizedBox(height: spacing.sp3),
                Padding(
                  padding: EdgeInsets.only(left: bodyIndent),
                  child: Text(
                    post.text,
                    style: theme.textTheme.bodyLarge,
                  ),
                ),
                SizedBox(height: spacing.sp3),
                const CraftskyDivider(),
                Row(
                  children: [
                    SizedBox(width: bodyIndent),
                    const _PostCardAction(icon: Icons.chat_bubble_outline),
                    SizedBox(width: spacing.sp4),
                    const _PostCardAction(icon: Icons.favorite_border),
                    SizedBox(width: spacing.sp4),
                    const _PostCardAction(icon: Icons.repeat),
                    const Spacer(),
                    if (onDelete != null) _PostCardMenu(onDelete: onDelete!),
                  ],
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
  });

  final String displayName;
  final String handle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          displayName,
          style: theme.textTheme.titleSmall,
          overflow: TextOverflow.ellipsis,
        ),
        Text(
          '@$handle',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.outline,
          ),
          overflow: TextOverflow.ellipsis,
        ),
      ],
    );
  }
}

class _PostCardTime extends StatelessWidget {
  const _PostCardTime({required this.postedAt});

  final DateTime postedAt;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Text(
      _relativeTime(postedAt),
      style: theme.textTheme.bodySmall?.copyWith(
        color: theme.colorScheme.outline,
      ),
      textAlign: TextAlign.end,
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

class _PostCardMenu extends StatelessWidget {
  const _PostCardMenu({required this.onDelete});

  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: _postCardMenuWidth,
      child: PopupMenuButton<_PostCardMenuAction>(
        icon: const Icon(Icons.more_horiz, size: 22),
        tooltip: AppLocalizations.of(context).postDeleteAction,
        padding: EdgeInsets.zero,
        onSelected: (action) {
          switch (action) {
            case _PostCardMenuAction.delete:
              onDelete();
          }
        },
        itemBuilder: (context) => [
          PopupMenuItem(
            value: _PostCardMenuAction.delete,
            child: Text(AppLocalizations.of(context).postDeleteAction),
          ),
        ],
      ),
    );
  }
}

class _PostCardAction extends StatelessWidget {
  const _PostCardAction({required this.icon});

  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Icon(
      icon,
      size: 22,
      color: theme.colorScheme.onSurface,
    );
  }
}

enum _PostCardMenuAction { delete }
