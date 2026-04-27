import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Circular paper-cutout avatar with the chunky 1.5px ink border and
/// hard-offset drop shadow that are signature to the design system.
/// Reused on profile headers, post cards, comments, search results —
/// anywhere we render someone's face.
///
/// Falls back to a butter-coloured initial when [avatarUrl] is null. The
/// initial is taken from [seed] (handle or display name); avatars never
/// show a generic person glyph because the brand voice prefers a warm,
/// hand-cut character.
class ProfileAvatar extends StatelessWidget {
  const ProfileAvatar({
    required this.seed,
    this.avatarUrl,
    this.size = ProfileAvatarSize.medium,
    super.key,
  });

  final String seed;
  final String? avatarUrl;
  final ProfileAvatarSize size;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final dimension = size.dimension;
    final borderWidth = size.borderWidth;
    final initial = seed.isEmpty ? '?' : seed.characters.first.toUpperCase();
    final shadows = theme.extension<BrandShadowTheme>()!;

    return Container(
      width: dimension,
      height: dimension,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: BrandColors.butter,
        border: Border.all(
          color: theme.colorScheme.onSurface,
          width: borderWidth,
        ),
        boxShadow: size.shadowsFrom(shadows),
        image: avatarUrl == null
            ? null
            : DecorationImage(
                image: NetworkImage(avatarUrl!),
                fit: BoxFit.cover,
              ),
      ),
      alignment: Alignment.center,
      child: avatarUrl != null
          ? null
          : Text(
              initial,
              style: theme.textTheme.displaySmall?.copyWith(
                fontSize: dimension * 0.5,
                color: BrandColors.ink,
              ),
            ),
    );
  }
}

/// Avatar size variants. Dimensions and border weights are tuned per
/// surface so the chunky border reads at every scale.
enum ProfileAvatarSize {
  small(dimension: 36, borderWidth: 1),
  medium(dimension: 48, borderWidth: 1.5),
  large(dimension: 96, borderWidth: 2)
  ;

  const ProfileAvatarSize({required this.dimension, required this.borderWidth});

  final double dimension;
  final double borderWidth;

  /// Hard-offset drop shadow scaled to the avatar's surface. All sizes
  /// currently use `dropSm` (3,3) so the avatar's shadow tail matches
  /// the chunky buttons it sits next to in the profile header — kept
  /// as a switch so we can dial up to `drop` (6,6) for hero variants
  /// later without touching call sites.
  List<BoxShadow> shadowsFrom(BrandShadowTheme shadows) {
    return switch (this) {
      ProfileAvatarSize.small => shadows.dropSm,
      ProfileAvatarSize.medium => shadows.dropSm,
      ProfileAvatarSize.large => shadows.dropSm,
    };
  }
}
