import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:flutter/material.dart';

/// Navigation/account-switcher adapter around the shared 36 px profile avatar.
class AccountAvatar extends StatelessWidget {
  const AccountAvatar({
    required this.avatarUrl,
    this.seed = '',
    this.customisation = ProfileCustomisation.defaults,
    this.selected = false,
    super.key,
  });

  final String? avatarUrl;
  final String seed;
  final ProfileCustomisation customisation;
  final bool selected;

  @override
  Widget build(BuildContext context) => Semantics(
    image: true,
    selected: selected,
    child: ProfileAvatar(
      seed: seed,
      avatarUrl: avatarUrl,
      size: ProfileAvatarSize.small,
      showShadow: false,
      customisation: customisation,
    ),
  );
}
