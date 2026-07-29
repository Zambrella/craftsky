import 'dart:math' as math;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/profile/widgets/profile_framed_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/profile/widgets/profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_stats.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Compact profile summary shown from identity taps throughout the app.
///
/// The card owns only presentation. Profile loading, navigation, mutations,
/// and future persistence of customisation choices stay with its modal host.
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
    required this.onClose,
    required this.onVisitProfile,
    required this.onPrimaryAction,
  });

  final Profile profile;
  final ProfileBackgroundIllustration? backgroundIllustration;
  final ProfileAvatarFrame? avatarFrame;
  final bool isOwnProfile;
  final bool isPrimaryActionBusy;
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
    final radius = BorderRadius.circular(radii.r4);
    final shadow = shadows.paper2.first;

    return LayoutBuilder(
      builder: (context, constraints) {
        final availableHeight = constraints.maxHeight.isFinite
            ? math
                  .max(
                    0,
                    constraints.maxHeight - spacing.sp4 * 2,
                  )
                  .toDouble()
            : double.infinity;
        return Center(
          child: Padding(
            padding: EdgeInsets.all(spacing.sp4),
            child: ConstrainedBox(
              constraints: BoxConstraints(
                maxWidth: 420,
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
                          color: shadow.color,
                          borderRadius: radius,
                        ),
                      ),
                    ),
                  ),
                  Material(
                    color: swatches.paper3,
                    clipBehavior: Clip.antiAlias,
                    shape: RoundedRectangleBorder(
                      borderRadius: radius,
                      side: BorderSide(color: colors.onSurface, width: 1.5),
                    ),
                    child: SingleChildScrollView(
                      child: Stack(
                        alignment: Alignment.topCenter,
                        children: [
                          Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              SizedBox(
                                height: 160,
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
                                  spacing.sp5,
                                  62,
                                  spacing.sp5,
                                  spacing.sp5,
                                ),
                                child: Column(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    ProfileIdentity(
                                      handle: profile.handle.toString(),
                                      displayName: profile.displayName,
                                      centered: true,
                                    ),
                                    if (profile.crafts.isNotEmpty) ...[
                                      SizedBox(height: spacing.sp3),
                                      ProfileCraftChips(
                                        crafts: profile.crafts,
                                        alignment: WrapAlignment.center,
                                      ),
                                    ],
                                    SizedBox(height: spacing.sp4),
                                    ProfileStats(profile: profile),
                                    SizedBox(height: spacing.sp5),
                                    _ProfileCardActions(
                                      isOwnProfile: isOwnProfile,
                                      isFollowing: profile.viewerIsFollowing,
                                      isBusy: isPrimaryActionBusy,
                                      onVisitProfile: onVisitProfile,
                                      onPrimaryAction: onPrimaryAction,
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                          Positioned(
                            top: 98,
                            child: ProfileFramedAvatar(
                              seed:
                                  profile.displayName ??
                                  profile.handle.toString(),
                              avatarUrl: profile.avatar,
                              frame: avatarFrame,
                              frameKey: const Key(
                                'profile-card-avatar-frame',
                              ),
                            ),
                          ),
                          Positioned(
                            top: spacing.sp3,
                            right: spacing.sp3,
                            child: Material(
                              color: swatches.paper3,
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
                        ],
                      ),
                    ),
                  ),
                ],
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
    required this.onVisitProfile,
    required this.onPrimaryAction,
  });

  final bool isOwnProfile;
  final bool isFollowing;
  final bool isBusy;
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
    return Row(
      children: [
        Expanded(
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
        if (!isOwnProfile) ...[
          SizedBox(width: spacing.sp3),
          Expanded(
            child: ChunkyButton(
              onPressed: isBusy ? null : onPrimaryAction,
              backgroundColor: isFollowing ? swatches.paper3 : null,
              foregroundColor: isFollowing ? colors.onSurface : null,
              child: Text(primaryLabel),
            ),
          ),
        ],
      ],
    );
  }
}
