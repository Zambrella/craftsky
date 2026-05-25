import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Circular paper-cutout avatar with the chunky 1.5px ink border and
/// hard-offset drop shadow that are signature to the design system.
/// Reused on profile headers, post cards, comments, search results —
/// anywhere we render someone's face.
///
/// Falls back to a butter-coloured initial when [avatarUrl] is null,
/// while the image is loading, and on cache error. The initial is taken
/// from [seed] (handle or display name); avatars never show a generic
/// person glyph because the brand voice prefers a warm, hand-cut
/// character.
class ProfileAvatar extends ConsumerWidget {
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
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final dimension = size.dimension;
    final borderWidth = size.borderWidth;
    final fallbackBackground = _fallbackBackgroundFor(seed, swatches);

    final fallback = _AvatarInitialFallback(
      seed: seed,
      dimension: dimension,
      backgroundColor: fallbackBackground,
      foregroundColor: theme.colorScheme.onSurface,
    );

    return Container(
      width: dimension,
      height: dimension,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: fallbackBackground,
        border: Border.all(
          color: theme.colorScheme.onSurface,
          width: borderWidth,
        ),
        boxShadow: size.shadowsFrom(shadows),
      ),
      child: ClipOval(
        child: avatarUrl == null
            ? fallback
            : CachedNetworkImage(
                imageUrl: avatarUrl!,
                cacheManager: ref.watch(profileImageCacheManagerProvider),
                fit: BoxFit.cover,
                width: dimension,
                height: dimension,
                placeholder: (_, _) => fallback,
                errorWidget: (_, _, _) => fallback,
                // Quick cross-fade instead of the 500ms default —
                // CachedNetworkImage always remounts in the placeholder
                // state (even on a disk-cache hit), so a slow fade makes
                // every revisit feel laggy.
                fadeInDuration: const Duration(milliseconds: 150),
                fadeOutDuration: const Duration(milliseconds: 150),
              ),
      ),
    );
  }
}

Color _fallbackBackgroundFor(String seed, BrandSwatchTheme swatches) {
  final trimmed = seed.trim();
  final initial = trimmed.isEmpty
      ? null
      : trimmed.characters.first.toUpperCase();
  final codeUnit = initial?.codeUnitAt(0) ?? 0;
  final colors = [
    swatches.butter,
    swatches.clay,
    swatches.moss,
    swatches.sky,
    swatches.lilac,
    swatches.paper2,
  ];
  return colors[codeUnit % colors.length];
}

class _AvatarInitialFallback extends StatelessWidget {
  const _AvatarInitialFallback({
    required this.seed,
    required this.dimension,
    required this.backgroundColor,
    required this.foregroundColor,
  });

  final String seed;
  final double dimension;
  final Color backgroundColor;
  final Color foregroundColor;

  @override
  Widget build(BuildContext context) {
    final initial = seed.isEmpty ? '?' : seed.characters.first.toUpperCase();
    return Container(
      color: backgroundColor,
      alignment: Alignment.center,
      child: Text(
        initial,
        style: Theme.of(context).textTheme.displaySmall?.copyWith(
          fontSize: dimension * 0.5,
          color: foregroundColor,
        ),
      ),
    );
  }
}

/// Avatar size variants. Dimensions and border weights are tuned per
/// surface so the chunky border reads at every scale.
enum ProfileAvatarSize {
  small(dimension: 36, borderWidth: 2),
  medium(dimension: 48, borderWidth: 2),
  large(dimension: 96, borderWidth: 2);

  const ProfileAvatarSize({required this.dimension, required this.borderWidth});

  final double dimension;
  final double borderWidth;

  /// Hard-offset drop shadow scaled to the avatar's surface. Small avatars
  /// render flat so dense surfaces like post cards stay visually clean.
  List<BoxShadow> shadowsFrom(BrandShadowTheme shadows) {
    return switch (this) {
      ProfileAvatarSize.small => const [],
      ProfileAvatarSize.medium => shadows.dropSm,
      ProfileAvatarSize.large => shadows.dropSm,
    };
  }
}
