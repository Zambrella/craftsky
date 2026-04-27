import 'dart:ui' as ui;

import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_banner.dart';
import 'package:craftsky_app/profile/widgets/profile_banner_chip.dart';
import 'package:craftsky_app/profile/widgets/profile_identity.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/material.dart';

/// Collapsing profile header. Owns the banner, avatar, action row,
/// banner chip, and the large display-name + `@handle` identity block
/// — all of them live inside the [SliverAppBar]'s flexibleSpace so
/// they fade out together as the bar collapses, leaving the back
/// button + a faded-in compact title.
///
/// Putting the avatar here (rather than overlapping from the next
/// sliver) is what lets it overhang the banner cleanly: the SliverAppBar
/// reserves an extra `avatarOverhang` of paper-coloured space below the
/// banner, and the avatar straddles the boundary at left:16. When the
/// bar collapses, that paper strip, the avatar, the action row, and
/// the identity block collapse with it.
class ProfileSliverAppBar extends StatelessWidget {
  const ProfileSliverAppBar({
    required this.handle,
    required this.bannerColor,
    required this.actions,
    this.displayName,
    this.avatarUrl,
    this.bannerUrl,
    this.bannerChipLabel,
    super.key,
  });

  final String handle;
  final Color bannerColor;
  final ProfileActionSet actions;
  final String? displayName;
  final String? avatarUrl;
  final String? bannerUrl;
  final String? bannerChipLabel;

  static const double bannerHeight = 200;

  /// How far the avatar pokes up into the banner from the boundary.
  /// Tuned so the bottom of the avatar's circle lands flush with the
  /// bottom of the 44px action buttons inside the paper strip — the
  /// avatar's lower edge and the button row read as one horizontal
  /// line. Math: avatar bottom = `(96 - overlap)` below the banner;
  /// button bottom = `(stripHeight + 44) / 2` below the banner; they
  /// align when `overlap = 96 - (stripHeight + 44) / 2`.
  static const double avatarBannerOverlap = 42;

  /// Height of the paper-coloured strip below the banner inside the
  /// SliverAppBar. Hosts the avatar's bottom portion and the action
  /// row, with a small gap between the row and the banner edge.
  static const double paperStripHeight = 64;

  /// Vertical space reserved below the avatar/action strip for the
  /// large display-name + `@handle` identity block.
  static const double identityHeight = 84;

  static const double expandedHeight =
      bannerHeight + paperStripHeight + identityHeight;

  @override
  Widget build(BuildContext context) {
    return SliverAppBar(
      pinned: true,
      expandedHeight: expandedHeight,
      // Paper, not banner colour, so the collapsed strip reads as
      // continuous chrome with the rest of the page once the banner
      // has faded out. `shape` is intentionally not overridden so the
      // global AppBarTheme bottom rule (1.5px ink) applies and the
      // bar separates from the meta content underneath.
      backgroundColor: BrandColors.paper,
      foregroundColor: BrandColors.ink,
      surfaceTintColor: Colors.transparent,
      flexibleSpace: _ProfileFlexibleSpace(
        bannerColor: bannerColor,
        bannerUrl: bannerUrl,
        bannerChipLabel: bannerChipLabel,
        handle: handle,
        displayName: displayName,
        avatarUrl: avatarUrl,
        actions: actions,
      ),
    );
  }
}

class _ProfileFlexibleSpace extends StatelessWidget {
  const _ProfileFlexibleSpace({
    required this.bannerColor,
    required this.bannerUrl,
    required this.bannerChipLabel,
    required this.handle,
    required this.displayName,
    required this.avatarUrl,
    required this.actions,
  });

  final Color bannerColor;
  final String? bannerUrl;
  final String? bannerChipLabel;
  final String handle;
  final String? displayName;
  final String? avatarUrl;
  final ProfileActionSet actions;

  /// Reserved horizontal space for the avatar plus a 12px gap, so the
  /// action row sits cleanly to its right. Kept as a literal because
  /// enum-field access isn't const-evaluable; equals
  /// `ProfileAvatarSize.large.dimension + 12`.
  static const double _avatarLaneWidth = 96 + 12;

  @override
  Widget build(BuildContext context) {
    final settings = context
        .dependOnInheritedWidgetOfExactType<FlexibleSpaceBarSettings>();
    final topPadding = MediaQuery.paddingOf(context).top;
    final maxExtent = settings?.maxExtent ?? ProfileSliverAppBar.expandedHeight;
    final minExtent = settings?.minExtent ?? (kToolbarHeight + topPadding);
    final currentExtent = settings?.currentExtent ?? maxExtent;

    final range = (maxExtent - minExtent).abs();
    final collapsed = range == 0
        ? 0.0
        : ((maxExtent - currentExtent) / range).clamp(0.0, 1.0);

    final bannerVisualBottom = topPadding + ProfileSliverAppBar.bannerHeight;
    final seed = (displayName?.isNotEmpty ?? false) ? displayName! : handle;

    return Stack(
      fit: StackFit.expand,
      children: [
        // Paper strip below the banner. Painted as a full-bleed layer
        // so anything that scrolls into the strip during collapse meets
        // a paper background, not banner colour.
        const ColoredBox(color: BrandColors.paper),
        // Banner — extends behind status bar so the colour reaches the
        // very top of the screen. Blurs and fades on collapse so the
        // bar's resting state is clean paper, not a stale strip of
        // banner. `TileMode.clamp` keeps the blur from sampling
        // outside the banner's bounds, which would otherwise pull the
        // paper background in around the edges.
        Positioned(
          top: 0,
          left: 0,
          right: 0,
          height: bannerVisualBottom,
          child: Opacity(
            opacity: 1 - collapsed,
            // The blur kernel extends outside the banner's box, which
            // visually grows the image as collapse progresses and lets
            // it overlap the action row below. ClipRect bounds the
            // filtered output back to the banner area.
            child: ClipRect(
              child: ImageFiltered(
                imageFilter: ui.ImageFilter.blur(
                  sigmaX: collapsed * 12,
                  sigmaY: collapsed * 12,
                  tileMode: ui.TileMode.clamp,
                ),
                child: ProfileBanner(
                  color: bannerColor,
                  bannerUrl: bannerUrl,
                  height: bannerVisualBottom,
                ),
              ),
            ),
          ),
        ),
        if (bannerChipLabel != null)
          Positioned(
            top: topPadding + 8,
            right: 16,
            child: Opacity(
              opacity: 1 - collapsed,
              child: ProfileBannerChip(label: bannerChipLabel!),
            ),
          ),
        // Avatar — straddles banner / paper strip at left:16. Most of
        // it sits in the paper strip; only `avatarBannerOverlap`
        // pokes up into the banner.
        Positioned(
          left: 16,
          top: bannerVisualBottom - ProfileSliverAppBar.avatarBannerOverlap,
          child: Opacity(
            opacity: 1 - collapsed,
            child: ProfileAvatar(
              seed: seed,
              avatarUrl: avatarUrl,
              size: ProfileAvatarSize.large,
            ),
          ),
        ),
        // Action row — fills the paper strip to the right of the
        // avatar. Right-aligned via [Align] so a content-sized group
        // (e.g. self profile's Edit + cog) sits flush with the right
        // edge while a stretching group (visitor's Follow + share)
        // still spans the full lane.
        Positioned(
          left: 16 + _avatarLaneWidth,
          right: 16,
          top: bannerVisualBottom,
          height: ProfileSliverAppBar.paperStripHeight,
          child: IgnorePointer(
            ignoring: collapsed > 0.5,
            child: Opacity(
              opacity: 1 - collapsed,
              child: Align(
                alignment: Alignment.centerRight,
                child: ProfileActions(actions: actions),
              ),
            ),
          ),
        ),
        // Display-name + `@handle` identity block. Sits below the
        // avatar/action strip and fades with the bar so the collapsed
        // toolbar carries only the compact title variant.
        Positioned(
          left: 16,
          right: 16,
          top: bannerVisualBottom + ProfileSliverAppBar.paperStripHeight,
          height: ProfileSliverAppBar.identityHeight,
          child: Opacity(
            opacity: 1 - collapsed,
            child: Align(
              alignment: Alignment.centerLeft,
              child: ProfileIdentity(
                handle: handle,
                displayName: displayName,
              ),
            ),
          ),
        ),
        // Collapsed-state title. Fades in over the toolbar strip as the
        // bar reaches its minimum extent.
        Positioned(
          left: 56,
          right: 56,
          top: topPadding,
          height: kToolbarHeight,
          child: Opacity(
            opacity: collapsed,
            child: _CollapsedTitle(handle: handle, displayName: displayName),
          ),
        ),
        // Collapsed-state trailing action (settings cog for self,
        // share for visitor). Fades in alongside the compact title
        // and is ignored for hit-testing while the bar is expanded so
        // taps land on the full Edit/Follow row instead.
        Positioned(
          right: 4,
          top: topPadding,
          height: kToolbarHeight,
          width: 48,
          child: IgnorePointer(
            ignoring: collapsed < 0.5,
            child: Opacity(
              opacity: collapsed,
              child: _CollapsedTrailingAction(actions: actions),
            ),
          ),
        ),
      ],
    );
  }
}

/// Single icon button that surfaces the most useful action for the
/// collapsed bar — settings on a self profile, share on a visitor
/// profile. Pulls the callback out of the [ProfileActionSet] so the
/// page only has to wire one action set, not two.
class _CollapsedTrailingAction extends StatelessWidget {
  const _CollapsedTrailingAction({required this.actions});

  final ProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    return switch (actions) {
      SelfProfileActionSet(:final onSettings) => IconButton(
        tooltip: 'Settings',
        icon: const Icon(Icons.settings_outlined),
        onPressed: onSettings,
      ),
      VisitorProfileActionSet(:final onShare) => IconButton(
        tooltip: 'Share',
        icon: const Icon(Icons.ios_share_outlined),
        onPressed: onShare,
      ),
    };
  }
}

class _CollapsedTitle extends StatelessWidget {
  const _CollapsedTitle({required this.handle, required this.displayName});

  final String handle;
  final String? displayName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final name = (displayName?.isNotEmpty ?? false) ? displayName! : '@$handle';
    final showSubtitle = displayName?.isNotEmpty ?? false;

    return Align(
      alignment: Alignment.centerLeft,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(
            name,
            style: theme.textTheme.titleMedium,
            overflow: TextOverflow.ellipsis,
          ),
          if (showSubtitle)
            Text(
              '@$handle',
              style: theme.textTheme.bodySmall?.copyWith(
                color: BrandColors.ink2,
              ),
              overflow: TextOverflow.ellipsis,
            ),
        ],
      ),
    );
  }
}
