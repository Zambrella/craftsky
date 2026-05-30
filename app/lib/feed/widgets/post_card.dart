import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/moderation_warning_banner.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

const _postCardMenuWidth = 48.0;
const _postCardActionIconSize = 22.0;

enum PostCardStyle { card, flat }

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
    this.onReport,
    this.deleteTooltip,
    this.deleteLabel,
    this.reportLabel,
    this.replyTooltip,
    this.showRepostAction = true,
    this.showReplyCount = true,
    this.showReplyLabel = false,
    this.isHighlighted = false,
    this.style = PostCardStyle.card,
  });

  final Post post;
  final VoidCallback? onReply;
  final VoidCallback? onTap;
  final VoidCallback? onLike;
  final VoidCallback? onRepost;
  final VoidCallback? onDelete;
  final VoidCallback? onReport;
  final String? deleteTooltip;
  final String? deleteLabel;
  final String? reportLabel;
  final String? replyTooltip;
  final bool showRepostAction;
  final bool showReplyCount;
  final bool showReplyLabel;
  final bool isHighlighted;
  final PostCardStyle style;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final semanticColors = theme.extension<SemanticColorsTheme>()!;
    final colors = theme.colorScheme;
    final l10n = AppLocalizations.of(context);
    final displayName = post.author.displayName ?? post.author.handle;
    final isFlat = style == PostCardStyle.flat;
    final borderRadius = isFlat
        ? BorderRadius.zero
        : BorderRadius.circular(radii.r3);

    final content = AnimatedContainer(
      duration: const Duration(milliseconds: 350),
      curve: Curves.easeOutCubic,
      decoration: BoxDecoration(
        color: isHighlighted ? swatches.sky.withValues(alpha: 0.32) : null,
        border: isHighlighted
            ? Border(
                left: BorderSide(color: colors.primary, width: 6),
              )
            : null,
      ),
      child: Material(
        type: MaterialType.transparency,
        borderRadius: borderRadius,
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          borderRadius: borderRadius,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Padding(
                padding: EdgeInsets.fromLTRB(
                  spacing.sp3,
                  isFlat ? spacing.sp2 : spacing.sp3,
                  spacing.sp3,
                  isFlat ? spacing.sp2 : 0,
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
                    if (post.images case final images?
                        when images.isNotEmpty) ...[
                      PostImageCarousel(
                        images: images,
                        onImageTap: (index) {
                          unawaited(
                            showPostImageGallery(
                              context,
                              images: images,
                              initialIndex: index,
                            ),
                          );
                        },
                      ),
                      SizedBox(height: spacing.sp3),
                    ],
                    if (post.moderation?.warningKind case final kind?) ...[
                      ModerationWarningBanner(warningKind: kind),
                      SizedBox(height: spacing.sp3),
                    ],
                    Text(post.text, style: theme.textTheme.bodyLarge),
                    SizedBox(height: spacing.sp2),
                    if (!isFlat) const CraftskyDivider(),
                    Row(
                      children: [
                        Row(
                          children: [
                            _PostCardAction(
                              icon: post.viewerHasLiked
                                  ? Icons.favorite
                                  : Icons.favorite_border,
                              count: post.likeCount,
                              isSelected: post.viewerHasLiked,
                              selectedColor: semanticColors.error,
                              tooltip: post.viewerHasLiked
                                  ? l10n.postUnlikeAction
                                  : l10n.postLikeAction,
                              onPressed: onLike,
                            ),
                            _PostCardAction(
                              icon: Icons.chat_bubble_outline,
                              count: showReplyCount ? post.replyCount : 0,
                              isSelected: post.viewerHasReplied,
                              selectedColor: swatches.clay,
                              tooltip: replyTooltip ?? l10n.postReplyAction,
                              label: showReplyLabel
                                  ? l10n.postReplyAction
                                  : null,
                              onPressed: onReply,
                            ),
                            if (showRepostAction)
                              _PostCardAction(
                                icon: Icons.repeat,
                                count: post.repostCount,
                                isSelected: post.viewerHasReposted,
                                selectedColor: semanticColors.success,
                                tooltip: post.viewerHasReposted
                                    ? l10n.postUnrepostAction
                                    : l10n.postRepostAction,
                                onPressed: onRepost,
                              ),
                          ],
                        ),
                        const Spacer(),
                        _PostCardMenu(
                          onDelete: onDelete,
                          onReport: onReport,
                          tooltip: deleteTooltip,
                          label: deleteLabel,
                          reportLabel: reportLabel,
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );

    if (isFlat) return content;

    return CraftskyCard(
      margin: EdgeInsets.fromLTRB(
        spacing.sp4,
        spacing.sp3,
        spacing.sp4,
        spacing.sp2,
      ),
      child: content,
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
    return Tooltip(
      message: _fullTimestamp(context, postedAt),
      child: Text(
        _relativeTime(postedAt),
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
        ),
        textAlign: TextAlign.end,
      ),
    );
  }

  String _fullTimestamp(BuildContext context, DateTime when) {
    final local = when.toLocal();
    final material = MaterialLocalizations.of(context);
    final date = material.formatFullDate(local);
    final time = material.formatTimeOfDay(
      TimeOfDay.fromDateTime(local),
      alwaysUse24HourFormat: MediaQuery.alwaysUse24HourFormatOf(context),
    );
    return '$date at $time ${local.timeZoneName}';
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
  const _PostCardMenu({
    required this.onDelete,
    required this.onReport,
    this.tooltip,
    this.label,
    this.reportLabel,
  });

  final VoidCallback? onDelete;
  final VoidCallback? onReport;
  final String? tooltip;
  final String? label;
  final String? reportLabel;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return SizedBox(
      width: _postCardMenuWidth,
      child: CraftskyContextMenuButton(
        tooltip: tooltip ?? l10n.postMoreActions,
        groups: [
          CraftskyContextMenuGroup(
            items: [
              if (onReport != null)
                CraftskyContextMenuItem(
                  text: reportLabel ?? l10n.postReportAction,
                  icon: Icons.flag_outlined,
                  onPressed: onReport,
                ),
              if (onDelete != null)
                CraftskyContextMenuItem(
                  text: label ?? l10n.postDeleteAction,
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
    this.label,
    this.isSelected = false,
    this.onPressed,
  });

  final IconData icon;
  final int count;
  final Color selectedColor;
  final String tooltip;
  final String? label;
  final bool isSelected;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final color = isSelected
        ? selectedColor
        : theme.colorScheme.onSurfaceVariant;
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
          child: TextButton.icon(
            icon: Icon(
              icon,
              color: color,
              size: _postCardActionIconSize,
            ),
            onPressed: onPressed,
            style: TextButton.styleFrom(
              foregroundColor: color,
              disabledForegroundColor: color,
              padding: EdgeInsets.symmetric(horizontal: spacing.sp1),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              alignment: Alignment.centerLeft,
            ),
            label: label != null
                ? Text(
                    label!,
                    style: theme.textTheme.labelLarge?.copyWith(color: color),
                  )
                : countLabel != null
                ? Text(
                    countLabel,
                    style: theme.textTheme.labelLarge?.copyWith(
                      color: color,
                      fontFeatures: const [FontFeature.tabularFigures()],
                    ),
                  )
                : const SizedBox.shrink(),
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
