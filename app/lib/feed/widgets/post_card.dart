import 'dart:async';

import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/moderation_warning_banner.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/projects/widgets/project_card.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
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
    this.onQuote,
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
    this.projectVariant = ProjectCardVariant.summary,
  });

  final Post post;
  final VoidCallback? onReply;
  final VoidCallback? onTap;
  final VoidCallback? onLike;
  final VoidCallback? onRepost;
  final VoidCallback? onQuote;
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
  final ProjectCardVariant projectVariant;

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
    final canShowShareAction = showRepostAction && post.reply == null;
    final borderRadius = isFlat
        ? BorderRadius.zero
        : BorderRadius.circular(radii.r3);
    void openAuthorProfile() => UserProfileRoute(
      handle: post.author.handle.toString(),
    ).push<void>(context);

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
                        _PostCardAuthorTapTarget(
                          onTap: openAuthorProfile,
                          child: ProfileAvatar(
                            seed: displayName,
                            avatarUrl: post.author.avatar,
                            size: ProfileAvatarSize.small,
                          ),
                        ),
                        SizedBox(width: spacing.sp3),
                        Expanded(
                          child: _PostCardHeader(
                            displayName: displayName,
                            handle: post.author.handle,
                            onTap: openAuthorProfile,
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
                    if (post.project case final project?) ...[
                      ProjectCard(project: project, variant: projectVariant),
                      SizedBox(height: spacing.sp3),
                    ],
                    FacetedText(
                      text: post.text,
                      facets: post.facets,
                      style: theme.textTheme.bodyLarge,
                    ),
                    if (post.quoteView case final quoteView?) ...[
                      SizedBox(height: spacing.sp3),
                      _QuotePreviewCard(quoteView: quoteView),
                    ],
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
                            if (canShowShareAction)
                              _PostCardShareAction(
                                count: post.repostCount + post.quoteCount,
                                isSelected: post.viewerHasReposted,
                                selectedColor: semanticColors.success,
                                tooltip: l10n.postShareAction,
                                repostLabel: post.viewerHasReposted
                                    ? l10n.postUnrepostAction
                                    : l10n.postRepostAction,
                                quoteLabel: l10n.postQuoteAction,
                                onRepost: onRepost,
                                onQuote: onQuote,
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
  const _PostCardHeader({
    required this.displayName,
    required this.handle,
    required this.onTap,
  });

  final String displayName;
  final String handle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return _PostCardAuthorTapTarget(
      onTap: onTap,
      child: Column(
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
      ),
    );
  }
}

class _PostCardAuthorTapTarget extends StatelessWidget {
  const _PostCardAuthorTapTarget({required this.onTap, required this.child});

  final VoidCallback onTap;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: child,
      ),
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

class _QuotePreviewCard extends StatelessWidget {
  const _QuotePreviewCard({required this.quoteView});

  final QuoteView quoteView;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final l10n = AppLocalizations.of(context);
    final post = quoteView.post;

    return DecoratedBox(
      decoration: BoxDecoration(
        color: swatches.paper2,
        borderRadius: BorderRadius.circular(radii.r2),
        border: Border.all(color: swatches.borderHair),
      ),
      child: Padding(
        padding: EdgeInsets.all(spacing.sp3),
        child: switch ((quoteView.state, post)) {
          ('visible', final quoted?) => Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _QuotePreviewAuthor(author: quoted.author),
              SizedBox(height: spacing.sp2),
              Text(
                quoted.text,
                maxLines: 4,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodyMedium,
              ),
            ],
          ),
          ('hidden', _) => Text(
            l10n.postQuoteHidden,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
          _ => Text(
            l10n.postQuoteUnavailable,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
        },
      ),
    );
  }
}

class _QuotePreviewAuthor extends StatelessWidget {
  const _QuotePreviewAuthor({required this.author});

  final PostAuthor author;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final displayName = author.displayName;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (displayName != null && displayName.trim().isNotEmpty)
          Text(
            displayName,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.titleSmall,
          ),
        Text(
          '@${author.handle}',
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.outline,
          ),
        ),
      ],
    );
  }
}

class _PostCardShareAction extends StatelessWidget {
  const _PostCardShareAction({
    required this.count,
    required this.selectedColor,
    required this.tooltip,
    required this.repostLabel,
    required this.quoteLabel,
    this.isSelected = false,
    this.onRepost,
    this.onQuote,
  });

  final int count;
  final Color selectedColor;
  final String tooltip;
  final String repostLabel;
  final String quoteLabel;
  final bool isSelected;
  final VoidCallback? onRepost;
  final VoidCallback? onQuote;

  bool get _isEnabled => onRepost != null || onQuote != null;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final color = isSelected
        ? selectedColor
        : theme.colorScheme.onSurfaceVariant;
    final countLabel = _compactCountLabel(count);

    return Builder(
      builder: (buttonContext) {
        void openMenu() {
          if (!_isEnabled) return;
          unawaited(_showMenu(buttonContext));
        }

        return Semantics(
          label: tooltip,
          button: true,
          enabled: _isEnabled,
          onTap: _isEnabled ? openMenu : null,
          child: ExcludeSemantics(
            child: Tooltip(
              message: tooltip,
              excludeFromSemantics: true,
              child: TextButton.icon(
                icon: Icon(
                  Icons.repeat,
                  color: color,
                  size: _postCardActionIconSize,
                ),
                onPressed: _isEnabled ? openMenu : null,
                style: TextButton.styleFrom(
                  foregroundColor: color,
                  disabledForegroundColor: color,
                  padding: EdgeInsets.symmetric(horizontal: spacing.sp1),
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  alignment: Alignment.centerLeft,
                ),
                label: countLabel != null
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
      },
    );
  }

  Future<void> _showMenu(BuildContext context) {
    return showCraftskyContextMenu(
      context,
      position: craftskyContextMenuAnchorPosition(context),
      groups: [
        CraftskyContextMenuGroup(
          items: [
            if (onRepost != null)
              CraftskyContextMenuItem(
                text: repostLabel,
                icon: Icons.repeat,
                onPressed: onRepost,
                isSelected: isSelected,
              ),
            if (onQuote != null)
              CraftskyContextMenuItem(
                text: quoteLabel,
                icon: Icons.format_quote,
                onPressed: onQuote,
              ),
          ],
        ),
      ],
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
