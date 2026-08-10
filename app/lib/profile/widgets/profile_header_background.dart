import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:flutter/material.dart';

const profileBackgroundAssets = <String, String>{
  'bayerdark': 'assets/profile_backgrounds/bayerdark.png',
  'cubedark': 'assets/profile_backgrounds/cubedark.png',
  'dotcrossdark': 'assets/profile_backgrounds/dotcrossdark.png',
  'scallopdark': 'assets/profile_backgrounds/scallopdark.png',
  'skewdark': 'assets/profile_backgrounds/skewdark.png',
  'x2': 'assets/profile_backgrounds/x2.png',
};

/// A bounded local texture layer for compact and full profile headers.
class ProfileHeaderBackground extends StatelessWidget {
  const ProfileHeaderBackground({
    this.customisation = ProfileCustomisation.defaults,
    this.backgroundKey = const Key('profile-header-background'),
    this.textureKey = const Key('profile-header-background-texture'),
    super.key,
  });

  final ProfileCustomisation customisation;
  final Key backgroundKey;
  final Key textureKey;

  @override
  Widget build(BuildContext context) {
    final bundle =
        profileColourBundles[customisation.colour] ??
        profileColourBundles[ProfileCustomisation.defaults.colour]!;
    final asset = profileBackgroundAssets[customisation.background];
    return ClipRect(
      child: ColoredBox(
        key: backgroundKey,
        color: profileColour(bundle.base),
        child: asset == null
            ? null
            : Image.asset(
                asset,
                key: textureKey,
                fit: BoxFit.none,
                repeat: ImageRepeat.repeat,
                color: profileColour(
                  bundle.textureTint,
                ).withValues(alpha: bundle.textureOpacity),
                colorBlendMode: BlendMode.srcIn,
                filterQuality: FilterQuality.none,
              ),
      ),
    );
  }
}
