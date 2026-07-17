import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/moderation_warning_banner.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/projects/widgets/project_card.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

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
    this.onQuotedPostTap,
    this.onQuotedAuthorTap,
    this.onReposterTap,
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
    this.repostReason,
  });

  final Post post;
  final VoidCallback? onReply;
  final VoidCallback? onTap;
  final VoidCallback? onLike;
  final VoidCallback? onRepost;
  final VoidCallback? onQuote;
  final VoidCallback? onQuotedPostTap;
  final VoidCallback? onQuotedAuthorTap;
  final VoidCallback? onReposterTap;
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
  final RepostReason? repostReason;

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
    final quotedPost = post.quoteView?.post;
    final quotedPostParts = quotedPost == null
        ? null
        : parseCraftskyPostUri(quotedPost.uri);
    final openQuotedPost =
        onQuotedPostTap ??
        (quotedPostParts == null
            ? null
            : () => PostThreadRoute(
                did: quotedPostParts.did,
                rkey: quotedPostParts.rkey,
              ).push<void>(context));
    final openQuotedAuthor =
        onQuotedAuthorTap ??
        (quotedPost == null
            ? null
            : () => UserProfileRoute(
                handle: quotedPost.author.handle.toString(),
              ).push<void>(context));
    final openReposter =
        onReposterTap ??
        (repostReason == null
            ? null
            : () => UserProfileRoute(
                handle: repostReason!.by.handle.toString(),
              ).push<void>(context));

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
                    if (repostReason case final reason?) ...[
                      _RepostAttribution(
                        reason: reason,
                        onTap: openReposter,
                      ),
                      SizedBox(height: spacing.sp2),
                    ],
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
                            child: RelativeTimeText(
                              timestamp: post.createdAt,
                              textAlign: TextAlign.end,
                            ),
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
                      _QuotePreviewCard(
                        quoteView: quoteView,
                        onPostTap: openQuotedPost,
                        onAuthorTap: openQuotedAuthor,
                      ),
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

class _RepostAttribution extends StatelessWidget {
  const _RepostAttribution({required this.reason, required this.onTap});

  final RepostReason reason;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final displayName = reason.by.displayName ?? reason.by.handle;
    return Row(
      children: [
        Icon(Icons.repeat, size: 16, color: theme.colorScheme.outline),
        const SizedBox(width: 8),
        Expanded(
          child: _PostCardAuthorTapTarget(
            onTap: onTap,
            child: Text(
              l10n.postRepostedBy(displayName),
              style: theme.textTheme.labelMedium?.copyWith(
                color: theme.colorScheme.outline,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ),
      ],
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

  final VoidCallback? onTap;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      enabled: onTap != null,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: child,
      ),
    );
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
  const _QuotePreviewCard({
    required this.quoteView,
    required this.onPostTap,
    required this.onAuthorTap,
  });

  final QuoteView quoteView;
  final VoidCallback? onPostTap;
  final VoidCallback? onAuthorTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final l10n = AppLocalizations.of(context);
    final post = quoteView.post;

    return Material(
      color: swatches.paper2,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.r2),
        side: BorderSide(color: swatches.borderHair),
      ),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: quoteView.state == 'visible' && post != null ? onPostTap : null,
        child: Padding(
          padding: EdgeInsets.all(spacing.sp3),
          child: switch ((quoteView.state, post)) {
            ('visible', final quoted?) => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _QuotePreviewAuthor(
                  author: quoted.author,
                  onTap: onAuthorTap,
                ),
                SizedBox(height: spacing.sp2),
                if (quoted.images?.firstOrNull case final image?) ...[
                  _QuotePreviewImage(image: image),
                  SizedBox(height: spacing.sp2),
                ],
                if (quoted.project?.common.title?.trim() case final title?
                    when title.isNotEmpty) ...[
                  Text(title, style: theme.textTheme.titleMedium),
                  SizedBox(height: spacing.sp2),
                ],
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
      ),
    );
  }
}

class _QuotePreviewImage extends ConsumerWidget {
  const _QuotePreviewImage({required this.image});

  final PostImage image;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final radii = Theme.of(context).extension<RadiusTheme>()!;
    final imageUrl = image.thumb ?? image.fullsize;

    return ClipRRect(
      key: const Key('quote-preview-image'),
      borderRadius: BorderRadius.circular(radii.r1),
      child: SizedBox(
        width: double.infinity,
        height: 160,
        child: Semantics(
          label: image.alt,
          image: true,
          child: imageUrl == null
              ? const DecoratedBox(
                  decoration: BoxDecoration(color: Color(0xFFEAEAEA)),
                )
              : CachedNetworkImage(
                  imageUrl: imageUrl,
                  cacheManager: ref.watch(feedImageCacheManagerProvider),
                  fit: BoxFit.cover,
                ),
        ),
      ),
    );
  }
}

class _QuotePreviewAuthor extends StatelessWidget {
  const _QuotePreviewAuthor({required this.author, required this.onTap});

  final PostAuthor author;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final displayName = author.displayName;
    final avatarSeed = displayName ?? author.handle;
    return _PostCardAuthorTapTarget(
      onTap: onTap,
      child: Row(
        children: [
          ProfileAvatar(
            seed: avatarSeed,
            avatarUrl: author.avatar,
            size: ProfileAvatarSize.small,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
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
            ),
          ),
        ],
      ),
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
    final countLabel = _compactCountLabel(context, count);

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
    final countLabel = _compactCountLabel(context, count);
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

String? _compactCountLabel(BuildContext context, int count) {
  if (count == 0) return null;
  final locale = Localizations.localeOf(context).toLanguageTag();
  final formatter = NumberFormat.compact(locale: locale)..significantDigits = 2;
  return formatter.format(count).toLowerCase();
}
