import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/feed/models/timeline_page.dart';
import 'package:craftsky_app/feed/widgets/post_image_carousel.dart';
import 'package:craftsky_app/feed/widgets/post_image_gallery.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/moderation/widgets/moderation_warning_banner.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/projects/widgets/project_card.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/saved_posts/widgets/saved_post_bookmark_button.dart';
import 'package:craftsky_app/shared/rich_text/widgets/faceted_text.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
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
class PostCard extends ConsumerWidget {
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
    this.onRevealQuotedPost,
    this.onRevealPost,
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
    this.hideWhenAuthorProtected = false,
  });

  final Post post;
  final VoidCallback? onReply;
  final VoidCallback? onTap;
  final VoidCallback? onLike;
  final VoidCallback? onRepost;
  final VoidCallback? onQuote;
  final VoidCallback? onQuotedPostTap;
  final VoidCallback? onQuotedAuthorTap;
  final VoidCallback? onRevealQuotedPost;
  final VoidCallback? onRevealPost;
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
  final bool hideWhenAuthorProtected;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (post.isProtected) {
      return _ProtectedPostCard(post: post, onReveal: onRevealPost);
    }
    final auth = ref.watch(authSessionProvider).value;
    final isViewerOwned = auth is SignedIn && auth.did == post.author.did;
    final account = auth is SignedIn ? AccountKey(auth.did.toString()) : null;
    final relationshipProvider = account == null || isViewerOwned
        ? null
        : profileRelationshipProvider(account, post.author.did.toString());
    final relationship = relationshipProvider == null
        ? null
        : ref.watch(relationshipProvider);
    final authorViewerRelationship = post.author.hasViewerState
        ? ProfileRelationship.fromProfileFlags(
            muted: post.author.muted ?? false,
            blocking: post.author.blocking ?? false,
            blockedBy: post.author.blockedBy ?? false,
          )
        : const ProfileRelationship(initialized: true);
    if (relationshipProvider != null && !(relationship?.initialized ?? false)) {
      unawaited(
        Future<void>.microtask(
          () => ref
              .read(relationshipProvider.notifier)
              .seed(authorViewerRelationship),
        ),
      );
    }
    final effectiveRelationship = relationship?.initialized ?? false
        ? relationship
        : post.author.hasViewerState
        ? authorViewerRelationship
        : null;
    final reposter = repostReason?.by;
    final isReposterViewerOwned =
        auth is SignedIn && reposter != null && auth.did == reposter.did;
    final reposterRelationshipProvider =
        reposter == null || account == null || isReposterViewerOwned
        ? null
        : reposter.did == post.author.did
        ? relationshipProvider
        : profileRelationshipProvider(account, reposter.did.toString());
    final reposterRelationship = reposterRelationshipProvider == null
        ? null
        : reposterRelationshipProvider == relationshipProvider
        ? relationship
        : ref.watch(reposterRelationshipProvider);
    final reposterViewerRelationship = reposter?.hasViewerState ?? false
        ? ProfileRelationship.fromProfileFlags(
            muted: reposter?.muted ?? false,
            blocking: reposter?.blocking ?? false,
            blockedBy: reposter?.blockedBy ?? false,
          )
        : const ProfileRelationship(initialized: true);
    if (reposterRelationshipProvider != null &&
        reposterRelationshipProvider != relationshipProvider &&
        !(reposterRelationship?.initialized ?? false)) {
      unawaited(
        Future<void>.microtask(
          () => ref
              .read(reposterRelationshipProvider.notifier)
              .seed(reposterViewerRelationship),
        ),
      );
    }
    final effectiveReposterRelationship =
        reposterRelationship?.initialized ?? false
        ? reposterRelationship
        : reposter?.hasViewerState ?? false
        ? reposterViewerRelationship
        : null;
    if (hideWhenAuthorProtected &&
        ((effectiveRelationship?.muted ?? false) ||
            (effectiveRelationship?.hasBlock ?? false) ||
            (effectiveReposterRelationship?.muted ?? false) ||
            (effectiveReposterRelationship?.hasBlock ?? false))) {
      return const SizedBox.shrink();
    }
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
                    if (effectiveRelationship?.muted ?? false) ...[
                      Semantics(
                        liveRegion: true,
                        child: Text(
                          l10n.profileMuteAnnotation,
                          style: theme.textTheme.labelLarge?.copyWith(
                            color: colors.onSurfaceVariant,
                          ),
                        ),
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
                        onReveal: onRevealQuotedPost,
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
                        if (account != null)
                          SavedPostBookmarkButton(
                            account: account,
                            post: post,
                          ),
                        _PostCardMenu(
                          onDelete: onDelete,
                          onReport: onReport,
                          tooltip: deleteTooltip,
                          label: deleteLabel,
                          reportLabel: reportLabel,
                          isMuted: effectiveRelationship?.muted ?? false,
                          isBlocking: effectiveRelationship?.blocking ?? false,
                          isRelationshipBusy:
                              effectiveRelationship?.pendingAction != null,
                          onMuteToggle: relationshipProvider == null
                              ? null
                              : () => unawaited(
                                  _mutateAuthorRelationship(
                                    context,
                                    ref,
                                    effectiveRelationship?.muted ?? false
                                        ? ProfileRelationshipAction.unmute
                                        : ProfileRelationshipAction.mute,
                                  ),
                                ),
                          onBlockToggle: relationshipProvider == null
                              ? null
                              : () => unawaited(
                                  _confirmBlockAuthor(
                                    context,
                                    ref,
                                    effectiveRelationship?.blocking ?? false,
                                  ),
                                ),
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

  Future<void> _confirmBlockAuthor(
    BuildContext context,
    WidgetRef ref,
    bool isBlocking,
  ) async {
    final l10n = AppLocalizations.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(
          isBlocking
              ? l10n.profileUnblockConfirmTitle
              : l10n.profileBlockConfirmTitle,
        ),
        content: Text(
          isBlocking
              ? l10n.profileUnblockConfirmBody
              : l10n.profileBlockConfirmBody,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.actionCancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(
              isBlocking ? l10n.profileUnblockAction : l10n.profileBlockAction,
            ),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return;
    await _mutateAuthorRelationship(
      context,
      ref,
      isBlocking
          ? ProfileRelationshipAction.unblock
          : ProfileRelationshipAction.block,
    );
  }

  Future<void> _mutateAuthorRelationship(
    BuildContext context,
    WidgetRef ref,
    ProfileRelationshipAction action,
  ) async {
    final auth = ref.read(authSessionProvider).value;
    if (auth is! SignedIn) return;
    final provider = profileRelationshipProvider(
      AccountKey(auth.did.toString()),
      post.author.did.toString(),
    );
    await ref.read(provider.notifier).mutate(action);
    if (!context.mounted) return;
    final failed = ref.read(provider).lastError != null;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          failed
              ? AppLocalizations.of(context).profileRelationshipError
              : switch (action) {
                  ProfileRelationshipAction.mute => AppLocalizations.of(
                    context,
                  ).profileMuteSuccess,
                  ProfileRelationshipAction.unmute => AppLocalizations.of(
                    context,
                  ).profileUnmuteSuccess,
                  ProfileRelationshipAction.block => AppLocalizations.of(
                    context,
                  ).profileBlockSuccess,
                  ProfileRelationshipAction.unblock => AppLocalizations.of(
                    context,
                  ).profileUnblockSuccess,
                },
        ),
      ),
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
    required this.isMuted,
    required this.isBlocking,
    required this.isRelationshipBusy,
    required this.onMuteToggle,
    required this.onBlockToggle,
    this.tooltip,
    this.label,
    this.reportLabel,
  });

  final VoidCallback? onDelete;
  final VoidCallback? onReport;
  final String? tooltip;
  final String? label;
  final String? reportLabel;
  final bool isMuted;
  final bool isBlocking;
  final bool isRelationshipBusy;
  final VoidCallback? onMuteToggle;
  final VoidCallback? onBlockToggle;

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
              if (onMuteToggle != null)
                CraftskyContextMenuItem(
                  text: isMuted
                      ? l10n.profileUnmuteAction
                      : l10n.profileMuteAction,
                  icon: Icons.volume_off_outlined,
                  onPressed: isRelationshipBusy ? null : onMuteToggle,
                ),
              if (onBlockToggle != null)
                CraftskyContextMenuItem(
                  text: isBlocking
                      ? l10n.profileUnblockAction
                      : l10n.profileBlockAction,
                  icon: Icons.block_outlined,
                  onPressed: isRelationshipBusy ? null : onBlockToggle,
                  style: CraftskyContextMenuItemStyle.destructive,
                  semanticHint: isBlocking ? null : l10n.destructiveActionHint,
                ),
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
    required this.onReveal,
  });

  final QuoteView quoteView;
  final VoidCallback? onPostTap;
  final VoidCallback? onAuthorTap;
  final VoidCallback? onReveal;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;

    return Material(
      color: swatches.paper2,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radii.r2),
        side: BorderSide(color: swatches.borderHair),
      ),
      clipBehavior: Clip.antiAlias,
      child: PostSummary(
        data: PostSummaryData.fromQuoteView(quoteView),
        onTap: onPostTap,
        onAuthorTap: onAuthorTap,
        onReveal: onReveal,
      ),
    );
  }
}

class _ProtectedPostCard extends StatelessWidget {
  const _ProtectedPostCard({required this.post, required this.onReveal});

  final Post post;
  final VoidCallback? onReveal;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final muted = post.availability == 'muted';
    return Semantics(
      label: muted
          ? l10n.postMutedPlaceholder
          : l10n.postUnavailablePlaceholder,
      child: CraftskyCard(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                muted
                    ? l10n.postMutedPlaceholder
                    : l10n.postUnavailablePlaceholder,
              ),
              if (muted &&
                  post.relationship?.revealable == true &&
                  onReveal != null)
                TextButton(
                  onPressed: onReveal,
                  child: Text(l10n.postRevealAction),
                ),
            ],
          ),
        ),
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
