import 'dart:typed_data';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_banner.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Editable banner + avatar header for the profile-edit page. Mirrors
/// the live profile's hero geometry (banner with the avatar overlapping
/// its bottom edge) while keeping the overlapping controls inside this
/// widget's hit-test bounds.
class EditProfileBannerAvatar extends StatelessWidget {
  const EditProfileBannerAvatar({
    required this.profile,
    required this.bannerColor,
    this.avatarPreviewBytes,
    this.bannerPreviewBytes,
    this.onPickAvatar,
    this.onPickBanner,
    this.avatarUploading = false,
    this.bannerUploading = false,
    this.avatarError = false,
    this.bannerError = false,
    super.key,
  });

  final Profile profile;
  final Color bannerColor;
  final Uint8List? avatarPreviewBytes;
  final Uint8List? bannerPreviewBytes;
  final VoidCallback? onPickAvatar;
  final VoidCallback? onPickBanner;
  final bool avatarUploading;
  final bool bannerUploading;
  final bool avatarError;
  final bool bannerError;

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

    return SizedBox(
      height: ProfileBanner.defaultHeight + _avatarDimension / 2,
      child: Stack(
        clipBehavior: Clip.none,
        alignment: Alignment.bottomLeft,
        children: [
          Positioned(
            top: 0,
            left: 0,
            right: 0,
            child: _EditableBanner(
              color: bannerColor,
              bannerUrl: profile.banner,
              previewBytes: bannerPreviewBytes,
              onPressed: onPickBanner,
              isUploading: bannerUploading,
              hasError: bannerError,
              label: l10n.editProfileChangeCover,
            ),
          ),
          Positioned(
            right: spacing.sp3,
            top: ProfileBanner.defaultHeight - spacing.sp3 - 40,
            child: FilledButton.tonalIcon(
              onPressed: onPickBanner,
              icon: const Icon(Icons.image_outlined),
              label: Text(l10n.editProfileChangeCover),
            ),
          ),
          // Avatar overlaps the banner visually, but stays inside this
          // SizedBox so its edit button can receive taps.
          Positioned(
            left: spacing.sp4,
            top: ProfileBanner.defaultHeight - _avatarDimension / 2,
            child: Stack(
              alignment: Alignment.bottomRight,
              children: [
                _EditableAvatar(
                  seed: profile.displayName ?? profile.handle,
                  avatarUrl: profile.avatar,
                  previewBytes: avatarPreviewBytes,
                  isUploading: avatarUploading,
                  hasError: avatarError,
                ),
                IconButton.filledTonal(
                  onPressed: onPickAvatar,
                  tooltip: l10n.editProfileChangeAvatar,
                  icon: const Icon(Icons.photo_camera_outlined, size: 18),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _EditableBanner extends StatelessWidget {
  const _EditableBanner({
    required this.color,
    required this.bannerUrl,
    required this.previewBytes,
    required this.onPressed,
    required this.isUploading,
    required this.hasError,
    required this.label,
  });

  final Color color;
  final String? bannerUrl;
  final Uint8List? previewBytes;
  final VoidCallback? onPressed;
  final bool isUploading;
  final bool hasError;
  final String label;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onPressed,
      child: Stack(
        children: [
          if (previewBytes == null)
            ProfileBanner(color: color, bannerUrl: bannerUrl)
          else
            Image.memory(
              previewBytes!,
              height: ProfileBanner.defaultHeight,
              width: double.infinity,
              fit: BoxFit.cover,
            ),
          if (isUploading || hasError)
            Positioned.fill(
              child: ColoredBox(
                color: Colors.black.withValues(alpha: 0.36),
                child: Center(
                  child: isUploading
                      ? const CircularProgressIndicator()
                      : Text(
                          label,
                          style: Theme.of(
                            context,
                          ).textTheme.labelLarge?.copyWith(color: Colors.white),
                        ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _EditableAvatar extends StatelessWidget {
  const _EditableAvatar({
    required this.seed,
    required this.avatarUrl,
    required this.previewBytes,
    required this.isUploading,
    required this.hasError,
  });

  final String seed;
  final String? avatarUrl;
  final Uint8List? previewBytes;
  final bool isUploading;
  final bool hasError;

  @override
  Widget build(BuildContext context) {
    final avatar = previewBytes == null
        ? ProfileAvatar(
            seed: seed,
            avatarUrl: avatarUrl,
            size: ProfileAvatarSize.large,
          )
        : ClipOval(
            child: Image.memory(
              previewBytes!,
              width: 96,
              height: 96,
              fit: BoxFit.cover,
            ),
          );
    if (!isUploading && !hasError) return avatar;
    return Stack(
      alignment: Alignment.center,
      children: [
        avatar,
        Container(
          width: 96,
          height: 96,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: Colors.black.withValues(alpha: 0.36),
          ),
          child: isUploading
              ? const Padding(
                  padding: EdgeInsets.all(28),
                  child: CircularProgressIndicator(strokeWidth: 3),
                )
              : const Icon(Icons.error_outline, color: Colors.white),
        ),
      ],
    );
  }
}
