import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
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
    final craft = profileCraftBackgrounds[customisation.background];
    final tint = profileColour(
      bundle.textureTint,
    ).withValues(alpha: bundle.textureOpacity);
    return ClipRect(
      child: ColoredBox(
        key: backgroundKey,
        color: profileColour(bundle.base),
        child: switch ((asset, craft)) {
          (final asset?, _) => Image.asset(
            asset,
            key: textureKey,
            fit: BoxFit.none,
            repeat: ImageRepeat.repeat,
            color: tint,
            colorBlendMode: BlendMode.srcIn,
            filterQuality: FilterQuality.none,
          ),
          (_, final craft?) => _CraftIconTile(
            key: textureKey,
            craft: craft,
            color: tint,
          ),
          _ => null,
        },
      ),
    );
  }
}

class _CraftIconTile extends StatelessWidget {
  const _CraftIconTile({
    required this.craft,
    required this.color,
    super.key,
  });

  static const _tileSize = 40.0;
  static const _iconSize = 20.0;

  final String craft;
  final Color color;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final columns = (constraints.maxWidth / _tileSize).ceil() + 1;
      final rows = (constraints.maxHeight / _tileSize).ceil() + 1;
      return Stack(
        children: [
          for (var row = 0; row < rows; row++)
            for (var column = 0; column < columns; column++)
              Positioned(
                left:
                    column * _tileSize +
                    (row.isOdd ? _tileSize / 2 : 0) -
                    _iconSize / 2,
                top: row * _tileSize - _iconSize / 2,
                child: CraftIcon(
                  craft: craft,
                  color: color,
                ),
              ),
        ],
      );
    },
  );
}
