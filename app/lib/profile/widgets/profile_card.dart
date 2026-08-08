import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/profile/widgets/profile_bio.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/profile/widgets/profile_framed_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/profile/widgets/profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_sliver_app_bar.dart';
import 'package:craftsky_app/profile/widgets/profile_stats.dart';
import 'package:craftsky_app/profile/widgets/profile_tab_bar.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Compact profile summary shown from identity taps throughout the app.
///
/// The card owns only presentation and can interpolate responsively into the
/// full-screen profile geometry. Loading, navigation, and mutations stay with
/// its route presentation.
class ProfileCard extends StatelessWidget {
  const ProfileCard({
    required this.profile,
    required this.isOwnProfile,
    required this.onClose,
    required this.onVisitProfile,
    required this.onPrimaryAction,
    this.primaryColor,
    this.backgroundIllustration,
    this.avatarFrame,
    this.isPrimaryActionBusy = false,
    this.expansionProgress = 0,
    this.transitionProgress = 0,
    this.compactHeight,
    this.surfaceMeasurementKey,
    super.key,
  });

  final Profile profile;
  final bool isOwnProfile;
  final VoidCallback onClose;
  final VoidCallback onVisitProfile;
  final VoidCallback onPrimaryAction;
  final Color? primaryColor;
  final ProfileBackgroundIllustration? backgroundIllustration;
  final ProfileAvatarFrame? avatarFrame;
  final bool isPrimaryActionBusy;
  final double expansionProgress;
  final double transitionProgress;
  final double? compactHeight;
  final GlobalKey? surfaceMeasurementKey;

  @override
  Widget build(BuildContext context) {
    return ProfileCustomisationTheme(
      primaryColor: primaryColor,
      child: _ProfileCardSurface(
        profile: profile,
        backgroundIllustration: backgroundIllustration,
        avatarFrame: avatarFrame,
        isOwnProfile: isOwnProfile,
        isPrimaryActionBusy: isPrimaryActionBusy,
        expansionProgress: expansionProgress,
        transitionProgress: transitionProgress,
        compactHeight: compactHeight,
        surfaceMeasurementKey: surfaceMeasurementKey,
        onClose: onClose,
        onVisitProfile: onVisitProfile,
        onPrimaryAction: onPrimaryAction,
      ),
    );
  }
}

class _ProfileCardSurface extends StatelessWidget {
  const _ProfileCardSurface({
    required this.profile,
    required this.backgroundIllustration,
    required this.avatarFrame,
    required this.isOwnProfile,
    required this.isPrimaryActionBusy,
    required this.expansionProgress,
    required this.transitionProgress,
    required this.compactHeight,
    required this.surfaceMeasurementKey,
    required this.onClose,
    required this.onVisitProfile,
    required this.onPrimaryAction,
  });

  final Profile profile;
  final ProfileBackgroundIllustration? backgroundIllustration;
  final ProfileAvatarFrame? avatarFrame;
  final bool isOwnProfile;
  final bool isPrimaryActionBusy;
  final double expansionProgress;
  final double transitionProgress;
  final double? compactHeight;
  final GlobalKey? surfaceMeasurementKey;
  final VoidCallback onClose;
  final VoidCallback onVisitProfile;
  final VoidCallback onPrimaryAction;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final shadow = shadows.paper2.first;
    final progress = expansionProgress.clamp(0.0, 1.0);
    final timeline = transitionProgress.clamp(0.0, 1.0);
    final compactOpacity =
        1 -
        const Interval(
          0.15,
          0.75,
          curve: Curves.easeInOut,
        ).transform(timeline);
    final expandedOpacity = const Interval(
      0.25,
      0.85,
      curve: Curves.easeInOut,
    ).transform(timeline);
    final topPadding = MediaQuery.paddingOf(context).top;

    return LayoutBuilder(
      builder: (context, constraints) {
        final outerPadding = ui.lerpDouble(spacing.sp4, 0, progress)!;
        final expandedWidth = constraints.maxWidth.isFinite
            ? constraints.maxWidth
            : 420.0;
        final maxWidth = ui.lerpDouble(420, expandedWidth, progress)!;
        final availableHeight = constraints.maxHeight.isFinite
            ? math
                  .max(
                    0,
                    constraints.maxHeight - outerPadding * 2,
                  )
                  .toDouble()
            : double.infinity;
        final minimumHeight = switch ((compactHeight, availableHeight)) {
          (final start?, final end) when end.isFinite => ui.lerpDouble(
            start,
            end,
            progress,
          )!,
          _ => 0.0,
        };
        final radius = BorderRadius.circular(
          ui.lerpDouble(radii.r4, 0, progress)!,
        );
        final headerHeight = ui.lerpDouble(
          160,
          topPadding + ProfileSliverAppBar.backgroundHeight,
          progress,
        )!;
        final avatarTop = ui.lerpDouble(
          98,
          topPadding + ProfileSliverAppBar.avatarTop,
          progress,
        )!;
        final bodyTopPadding = ui.lerpDouble(
          62,
          ProfileSliverAppBar.identityTop -
              ProfileSliverAppBar.backgroundHeight,
          progress,
        )!;
        final minimumIdentityHeight = ui.lerpDouble(
          0,
          ProfileSliverAppBar.minimumIdentityHeight,
          progress,
        )!;
        final identityToCrafts = ui.lerpDouble(
          spacing.sp3,
          spacing.sp2,
          progress,
        )!;
        final statsToActions = ui.lerpDouble(
          spacing.sp5,
          spacing.sp4,
          progress,
        )!;
        final actionsToExpandedContent = ui.lerpDouble(
          spacing.sp5,
          spacing.sp3,
          progress,
        )!;
        final horizontalPadding = ui.lerpDouble(
          spacing.sp5,
          spacing.sp4,
          progress,
        )!;
        final closeTop = ui.lerpDouble(
          spacing.sp3,
          topPadding + 4,
          progress,
        )!;
        return Center(
          child: Padding(
            padding: EdgeInsets.all(outerPadding),
            child: SizedBox(
              key: surfaceMeasurementKey,
              child: ConstrainedBox(
                key: const Key('profile-card-transition-surface'),
                constraints: BoxConstraints(
                  minHeight: math.min(minimumHeight, availableHeight),
                  maxWidth: maxWidth,
                  maxHeight: availableHeight,
                ),
                child: Stack(
                  clipBehavior: Clip.none,
                  children: [
                    Positioned.fill(
                      child: Transform.translate(
                        offset: shadow.offset,
                        child: DecoratedBox(
                          decoration: BoxDecoration(
                            color: shadow.color.withValues(
                              alpha: shadow.color.a * (1 - progress),
                            ),
                            borderRadius: radius,
                          ),
                        ),
                      ),
                    ),
                    SizedBox(
                      width: double.infinity,
                      height: compactHeight == null
                          ? null
                          : math.min(minimumHeight, availableHeight),
                      child: Material(
                        color: Color.lerp(
                          swatches.paper3,
                          theme.scaffoldBackgroundColor,
                          progress,
                        ),
                        clipBehavior: Clip.antiAlias,
                        shape: RoundedRectangleBorder(
                          borderRadius: radius,
                          side: BorderSide(
                            color: colors.onSurface.withValues(
                              alpha: 1 - progress,
                            ),
                            width: 1.5,
                          ),
                        ),
                        child: SingleChildScrollView(
                          child: Stack(
                            alignment: Alignment.topCenter,
                            children: [
                              Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  SizedBox(
                                    height: headerHeight,
                                    width: double.infinity,
                                    child: ProfileHeaderBackground(
                                      illustration: backgroundIllustration,
                                      backgroundKey: const Key(
                                        'profile-card-header',
                                      ),
                                      illustrationKey: const Key(
                                        'profile-card-background-illustration',
                                      ),
                                    ),
                                  ),
                                  Padding(
                                    padding: EdgeInsets.fromLTRB(
                                      horizontalPadding,
                                      bodyTopPadding,
                                      horizontalPadding,
                                      actionsToExpandedContent,
                                    ),
                                    child: Column(
                                      mainAxisSize: MainAxisSize.min,
                                      children: [
                                        ConstrainedBox(
                                          constraints: BoxConstraints(
                                            minHeight: minimumIdentityHeight,
                                          ),
                                          child: Center(
                                            child: ProfileIdentity(
                                              handle: profile.handle.toString(),
                                              displayName: profile.displayName,
                                              centered: true,
                                            ),
                                          ),
                                        ),
                                        if (profile.crafts.isNotEmpty) ...[
                                          SizedBox(
                                            height: identityToCrafts,
                                          ),
                                          ProfileCraftChips(
                                            crafts: profile.crafts,
                                            alignment: WrapAlignment.center,
                                          ),
                                        ],
                                        SizedBox(height: spacing.sp4),
                                        ProfileStats(profile: profile),
                                        SizedBox(height: statsToActions),
                                        ConstrainedBox(
                                          key: const Key(
                                            'profile-card-action-section',
                                          ),
                                          constraints: const BoxConstraints(
                                            maxWidth:
                                                profileActionSectionMaxWidth,
                                          ),
                                          child: SizedBox(
                                            width: double.infinity,
                                            child: _ProfileCardActions(
                                              isOwnProfile: isOwnProfile,
                                              isFollowing:
                                                  profile.viewerIsFollowing,
                                              isBusy: isPrimaryActionBusy,
                                              compactOpacity: compactOpacity,
                                              expandedOpacity: expandedOpacity,
                                              onVisitProfile: onVisitProfile,
                                              onPrimaryAction: onPrimaryAction,
                                            ),
                                          ),
                                        ),
                                      ],
                                    ),
                                  ),
                                  if (timeline > 0)
                                    Opacity(
                                      key: const Key(
                                        'profile-card-expanded-only',
                                      ),
                                      opacity: expandedOpacity,
                                      child: Column(
                                        crossAxisAlignment:
                                            CrossAxisAlignment.stretch,
                                        children: [
                                          if (profile.description?.isNotEmpty ??
                                              false)
                                            Padding(
                                              padding: EdgeInsets.fromLTRB(
                                                horizontalPadding,
                                                0,
                                                horizontalPadding,
                                                spacing.sp4,
                                              ),
                                              child: ProfileBio(
                                                description:
                                                    profile.description,
                                              ),
                                            ),
                                          DefaultTabController(
                                            length: ProfileTab.values.length,
                                            child: const SizedBox(
                                              height:
                                                  ProfileTabBarDelegate.height,
                                              child: ProfileTabBar(),
                                            ),
                                          ),
                                        ],
                                      ),
                                    ),
                                ],
                              ),
                              Positioned(
                                top: avatarTop,
                                child: ProfileFramedAvatar(
                                  seed:
                                      profile.displayName ??
                                      profile.handle.toString(),
                                  avatarUrl: profile.avatar,
                                  frame: avatarFrame,
                                  rimColor: Color.lerp(
                                    colors.surface,
                                    theme.scaffoldBackgroundColor,
                                    progress,
                                  ),
                                  frameKey: const Key(
                                    'profile-card-avatar-frame',
                                  ),
                                ),
                              ),
                              Positioned(
                                top: closeTop,
                                right: spacing.sp3,
                                child: Opacity(
                                  key: const Key(
                                    'profile-card-compact-close',
                                  ),
                                  opacity: compactOpacity,
                                  child: Material(
                                    color: Color.lerp(
                                      swatches.paper3,
                                      theme.scaffoldBackgroundColor,
                                      progress,
                                    ),
                                    shape: const CircleBorder(),
                                    child: IconButton(
                                      key: const Key('profile-card-close'),
                                      tooltip: MaterialLocalizations.of(
                                        context,
                                      ).closeButtonTooltip,
                                      onPressed: onClose,
                                      icon: const Icon(Icons.close),
                                    ),
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

class _ProfileCardActions extends StatelessWidget {
  const _ProfileCardActions({
    required this.isOwnProfile,
    required this.isFollowing,
    required this.isBusy,
    required this.compactOpacity,
    required this.expandedOpacity,
    required this.onVisitProfile,
    required this.onPrimaryAction,
  });

  final bool isOwnProfile;
  final bool isFollowing;
  final bool isBusy;
  final double compactOpacity;
  final double expandedOpacity;
  final VoidCallback onVisitProfile;
  final VoidCallback onPrimaryAction;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>()!;
    final primaryLabel = isFollowing
        ? l10n.profileFollowingAction
        : l10n.profileFollowAction;
    if (isOwnProfile) {
      return Stack(
        alignment: Alignment.centerRight,
        children: [
          Opacity(
            key: const Key('profile-card-compact-visit'),
            opacity: compactOpacity,
            child: SizedBox(
              width: double.infinity,
              child: OutlinedButton(
                onPressed: onVisitProfile,
                style: OutlinedButton.styleFrom(
                  foregroundColor: colors.primary,
                  side: BorderSide(color: colors.primary, width: 1.5),
                  minimumSize: const Size(0, 48),
                ),
                child: Text(l10n.profileVisitAction),
              ),
            ),
          ),
          if (expandedOpacity > 0)
            Opacity(
              opacity: expandedOpacity,
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  ChunkyButton(
                    onPressed: () {},
                    backgroundColor: swatches.paper3,
                    foregroundColor: colors.onSurface,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(Icons.edit_outlined),
                        SizedBox(width: spacing.sp2),
                        Text(l10n.profileEditAction),
                      ],
                    ),
                  ),
                  SizedBox(width: spacing.sp3),
                  ChunkyIconButton(
                    onPressed: () {},
                    icon: Icons.settings_outlined,
                    tooltip: l10n.profileSettingsAction,
                  ),
                ],
              ),
            ),
        ],
      );
    }

    return LayoutBuilder(
      builder: (context, constraints) {
        final compactButtonWidth = math
            .max(
              0,
              (constraints.maxWidth - spacing.sp3) / 2,
            )
            .toDouble();
        final compactVisitWidth = compactButtonWidth * compactOpacity;
        return Row(
          children: [
            ClipRect(
              child: SizedBox(
                width: compactVisitWidth,
                height: 48,
                child: OverflowBox(
                  alignment: Alignment.centerLeft,
                  minWidth: compactButtonWidth,
                  maxWidth: compactButtonWidth,
                  minHeight: 48,
                  maxHeight: 48,
                  child: Opacity(
                    key: const Key('profile-card-compact-visit'),
                    opacity: compactOpacity,
                    child: SizedBox(
                      width: compactButtonWidth,
                      child: OutlinedButton(
                        onPressed: onVisitProfile,
                        style: OutlinedButton.styleFrom(
                          foregroundColor: colors.primary,
                          side: BorderSide(
                            color: colors.primary,
                            width: 1.5,
                          ),
                          minimumSize: const Size(0, 48),
                        ),
                        child: Text(l10n.profileVisitAction),
                      ),
                    ),
                  ),
                ),
              ),
            ),
            SizedBox(width: spacing.sp3 * compactOpacity),
            Expanded(
              child: ChunkyButton(
                onPressed: isBusy ? null : onPrimaryAction,
                backgroundColor: isFollowing ? swatches.paper3 : null,
                foregroundColor: isFollowing ? colors.onSurface : null,
                child: Text(primaryLabel),
              ),
            ),
            SizedBox(width: spacing.sp3 * expandedOpacity),
            if (expandedOpacity > 0)
              ClipRect(
                child: Align(
                  alignment: Alignment.centerLeft,
                  widthFactor: expandedOpacity,
                  child: Opacity(
                    opacity: expandedOpacity,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        ChunkyIconButton(
                          onPressed: () {},
                          icon: Icons.volume_off_outlined,
                          tooltip: l10n.profileMuteAction,
                        ),
                        SizedBox(width: spacing.sp3),
                        CraftskyContextMenuButton(
                          tooltip: l10n.profileMoreActions,
                          groups: _expandedMenuGroups(l10n),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
          ],
        );
      },
    );
  }

  List<CraftskyContextMenuGroup> _expandedMenuGroups(
    AppLocalizations l10n,
  ) {
    return [
      CraftskyContextMenuGroup(
        items: [
          CraftskyContextMenuItem(
            text: l10n.profileShareAction,
            icon: Icons.ios_share_outlined,
            onPressed: () {},
          ),
        ],
      ),
      CraftskyContextMenuGroup(
        items: [
          CraftskyContextMenuItem(
            text: l10n.profileBlockAction,
            icon: Icons.block_outlined,
            onPressed: () {},
            style: CraftskyContextMenuItemStyle.destructive,
            semanticHint: l10n.destructiveActionHint,
          ),
          CraftskyContextMenuItem(
            text: l10n.profileReportAction,
            icon: Icons.flag_outlined,
            onPressed: () {},
          ),
        ],
      ),
    ];
  }
}
