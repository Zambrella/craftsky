import 'dart:typed_data';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Centered avatar editor for the profile-edit page.
class EditProfileBannerAvatar extends StatelessWidget {
  const EditProfileBannerAvatar({
    required this.profile,
    this.avatarPreviewBytes,
    this.onPickAvatar,
    this.avatarUploading = false,
    this.avatarError = false,
    super.key,
  });

  final Profile profile;
  final Uint8List? avatarPreviewBytes;
  final VoidCallback? onPickAvatar;
  final bool avatarUploading;
  final bool avatarError;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;

    return Padding(
      padding: EdgeInsets.symmetric(vertical: spacing.sp4),
      child: Center(
        child: Stack(
          alignment: Alignment.bottomRight,
          children: [
            _EditableAvatar(
              seed: profile.displayName ?? profile.handle,
              avatarUrl: profile.avatar,
              customisation: profile.customisation,
              previewBytes: avatarPreviewBytes,
              isUploading: avatarUploading,
              hasError: avatarError,
            ),
            IconButton.filledTonal(
              onPressed: onPickAvatar,
              tooltip: l10n.editProfileChangeAvatar,
              icon: const Icon(CraftskyIconsBold.camera, size: 18),
            ),
          ],
        ),
      ),
    );
  }
}

class _EditableAvatar extends StatelessWidget {
  const _EditableAvatar({
    required this.seed,
    required this.avatarUrl,
    required this.customisation,
    required this.previewBytes,
    required this.isUploading,
    required this.hasError,
  });

  final String seed;
  final String? avatarUrl;
  final ProfileCustomisation customisation;
  final Uint8List? previewBytes;
  final bool isUploading;
  final bool hasError;

  @override
  Widget build(BuildContext context) {
    final avatar = ProfileAvatar(
      seed: seed,
      avatarUrl: avatarUrl,
      imageProvider: previewBytes == null ? null : MemoryImage(previewBytes!),
      size: ProfileAvatarSize.large,
      showShadow: false,
      customisation: customisation,
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
              : const Icon(CraftskyIcons.error, color: Colors.white),
        ),
      ],
    );
  }
}
