import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_banner.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Read-only banner + avatar header for the profile-edit page. Mirrors
/// the live profile's hero geometry (banner with the avatar overlapping
/// its bottom edge) so the user has a "this is what your profile looks
/// like" reference while editing the rest of the form.
///
/// Photo uploads aren't wired in v1 — the AppView's `PUT /v1/profiles/me`
/// rejects banner/avatar fields. The hero is rendered behind a soft
/// overlay with a "Photo uploads coming soon" caption so the user
/// doesn't tap and wonder why nothing happened.
class EditProfileBannerAvatar extends StatelessWidget {
  const EditProfileBannerAvatar({
    required this.profile,
    required this.bannerColor,
    super.key,
  });

  final Profile profile;
  final Color bannerColor;

  /// Diameter of the avatar — matches `ProfileAvatarSize.large`. Hard-
  /// coded here because we need the value for the overlap math (negative
  /// bottom margin on the avatar) and the size enum doesn't expose a
  /// `byName` lookup.
  static const double _avatarDimension = 96;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);

    return Padding(
      // Reserve space below the stack equal to half the avatar so the
      // overflow doesn't collide with the form fields below.
      padding: const EdgeInsets.only(bottom: _avatarDimension / 2),
      child: Stack(
        clipBehavior: Clip.none,
        alignment: Alignment.bottomLeft,
        children: [
          ProfileBanner(color: bannerColor, bannerUrl: profile.banner),
          Positioned.fill(
            child: ColoredBox(
              // Soft scrim so the caption reads against any banner —
              // alphaBlend rather than `Opacity` per the brand rules
              // (no opacity tricks for muted colours).
              color: Color.alphaBlend(
                theme.colorScheme.onSurface.withValues(alpha: 0.32),
                Colors.transparent,
              ),
              child: Center(
                child: Padding(
                  padding: EdgeInsets.symmetric(horizontal: spacing.sp4),
                  child: Text(
                    l10n.editProfilePhotosComingSoon,
                    textAlign: TextAlign.center,
                    style: theme.textTheme.labelLarge?.copyWith(
                      color: theme.colorScheme.surface,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ),
              ),
            ),
          ),
          // Avatar overlapping the bottom edge of the banner. The
          // negative bottom positions half of the avatar below the
          // banner — same shape as the live profile hero.
          Positioned(
            left: spacing.sp4,
            bottom: -_avatarDimension / 2,
            child: ProfileAvatar(
              seed: profile.displayName ?? profile.handle,
              avatarUrl: profile.avatar,
              size: ProfileAvatarSize.large,
            ),
          ),
        ],
      ),
    );
  }
}
