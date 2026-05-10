import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

const _postCardMenuWidth = 48.0;
const _postCardActionIconSize = 22.0;

/// Card-shaped post row used by the feed and the profile Posts tab.
class PostCard extends StatelessWidget {
  const PostCard({
    required this.post,
    super.key,
    this.onReply,
    this.onTap,
    this.onLike,
    this.onRepost,
    this.onDelete,
    this.replyTooltip,
  });

  final Post post;
  final VoidCallback? onReply;
  final VoidCallback? onTap;
  final VoidCallback? onLike;
  final VoidCallback? onRepost;
  final VoidCallback? onDelete;
  final String? replyTooltip;

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
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(24),
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
                    child: Text(post.text, style: theme.textTheme.bodyLarge),
                  ),
                  SizedBox(height: spacing.sp3),
                  const CraftskyDivider(),
                  Row(
                    children: [
                      SizedBox(width: bodyIndent),
                      Expanded(
                        child: Row(
                          children: [
                            Expanded(
                              child: _PostCardAction(
                                icon: Icons.chat_bubble_outline,
                                count: post.replyCount,
                                selectedColor: BrandColors.sky,
                                tooltip: replyTooltip ?? 'Reply',
                                onPressed: onReply,
                              ),
                            ),
                            Expanded(
                              child: _PostCardAction(
                                icon: post.viewerHasLiked
                                    ? Icons.favorite
                                    : Icons.favorite_border,
                                count: post.likeCount,
                                isSelected: post.viewerHasLiked,
                                selectedColor: BrandColors.red,
                                tooltip: post.viewerHasLiked
                                    ? 'Unlike'
                                    : 'Like',
                                onPressed: onLike,
                              ),
                            ),
                            Expanded(
                              child: _PostCardAction(
                                icon: Icons.repeat,
                                count: post.repostCount,
                                isSelected: post.viewerHasReposted,
                                selectedColor: BrandColors.moss,
                                tooltip: post.viewerHasReposted
                                    ? 'Unrepost'
                                    : 'Repost',
                                onPressed: onRepost,
                              ),
                            ),
                          ],
                        ),
                      ),
                      if (onDelete != null) _PostCardMenu(onDelete: onDelete!),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _PostCardHeader extends StatelessWidget {
  const _PostCardHeader({required this.displayName, required this.handle});

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
    final l10n = AppLocalizations.of(context);
    return SizedBox(
      width: _postCardMenuWidth,
      child: CraftskyContextMenuButton(
        tooltip: l10n.postDeleteAction,
        groups: [
          CraftskyContextMenuGroup(
            items: [
              CraftskyContextMenuItem(
                text: l10n.postDeleteAction,
                icon: Icons.delete_outline,
                onPressed: onDelete,
                style: CraftskyContextMenuItemStyle.destructive,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _PostCardAction extends StatelessWidget {
  const _PostCardAction({
    required this.icon,
    required this.count,
    required this.selectedColor,
    required this.tooltip,
    this.isSelected = false,
    this.onPressed,
  });

  final IconData icon;
  final int count;
  final Color selectedColor;
  final String tooltip;
  final bool isSelected;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final color = isSelected ? selectedColor : BrandColors.ink2;
    final countLabel = _compactCountLabel(count);
    return Semantics(
      label: tooltip,
      button: true,
      enabled: onPressed != null,
      onTap: onPressed,
      child: ExcludeSemantics(
        child: Tooltip(
          message: tooltip,
          excludeFromSemantics: true,
          child: TextButton(
            onPressed: onPressed,
            style: TextButton.styleFrom(
              foregroundColor: color,
              disabledForegroundColor: color,
              minimumSize: Size(spacing.sp7, spacing.sp7),
              padding: EdgeInsets.symmetric(horizontal: spacing.sp1),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              alignment: Alignment.centerLeft,
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(icon, color: color, size: _postCardActionIconSize),
                if (countLabel != null) ...[
                  SizedBox(width: spacing.sp1),
                  Text(
                    countLabel,
                    style: theme.textTheme.labelLarge?.copyWith(color: color),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  String? _compactCountLabel(int count) {
    if (count == 0) return null;
    if (count.abs() < 1000) return '$count';

    final absoluteCount = count.abs();
    final sign = count.isNegative ? '-' : '';
    if (absoluteCount < 999950) {
      return '$sign${_trimCompactNumber(absoluteCount / 1000)}k';
    }
    if (absoluteCount < 999950000) {
      return '$sign${_trimCompactNumber(absoluteCount / 1000000)}m';
    }
    return '$sign${_trimCompactNumber(absoluteCount / 1000000000)}b';
  }

  String _trimCompactNumber(double value) {
    final digits = value < 10 ? 1 : 0;
    return value.toStringAsFixed(digits).replaceFirst(RegExp(r'\.0$'), '');
  }
}
