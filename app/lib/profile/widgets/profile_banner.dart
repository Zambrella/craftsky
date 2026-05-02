import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Flat coloured banner that sits behind the profile header. Renders a
/// user-supplied banner image on top of the colour swatch when one is
/// set; otherwise the swatch is the banner. The swatch shows through
/// during image load and on cache error.
///
/// Height is fixed so the avatar overlap math in `ProfileHeaderHero`
/// stays predictable.
class ProfileBanner extends ConsumerWidget {
  const ProfileBanner({
    required this.color,
    this.bannerUrl,
    this.height = ProfileBanner.defaultHeight,
    super.key,
  });

  static const double defaultHeight = 160;

  final Color color;
  final String? bannerUrl;
  final double height;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Container(
      height: height,
      width: double.infinity,
      color: color,
      child: bannerUrl == null
          ? null
          : CachedNetworkImage(
              imageUrl: bannerUrl!,
              cacheManager: ref.watch(profileImageCacheManagerProvider),
              fit: BoxFit.cover,
              placeholder: (_, _) => const SizedBox.shrink(),
              errorWidget: (_, _, _) => const SizedBox.shrink(),
            ),
    );
  }
}
