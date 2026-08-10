import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:flutter/material.dart';

/// Compatibility wrapper that centres the standard large profile avatar in
/// the existing 124 px profile-header positioning box.
class ProfileFramedAvatar extends StatelessWidget {
  const ProfileFramedAvatar({
    required this.seed,
    this.avatarUrl,
    this.customisation = ProfileCustomisation.defaults,
    super.key,
  });

  final String seed;
  final String? avatarUrl;
  final ProfileCustomisation customisation;

  @override
  Widget build(BuildContext context) => SizedBox.square(
    dimension: 124,
    child: Center(
      child: ProfileAvatar(
        seed: seed,
        avatarUrl: avatarUrl,
        size: ProfileAvatarSize.large,
        showShadow: false,
        customisation: customisation,
      ),
    ),
  );
}
